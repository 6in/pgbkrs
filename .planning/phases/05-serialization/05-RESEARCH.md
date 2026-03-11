# Phase 5: Serialization - Research

**Researched:** 2026-03-11
**Domain:** Go YAML serialization, PostgreSQL COPY TO CSV, sha256 checksums
**Confidence:** HIGH

## Summary

Phase 5 implements the `Serializer` interface for all 11 object types (table, view, materialized view, function, trigger, sequence, composite type, domain, enum, policy, foreign key) plus a `DataExporter` for table data CSV output. The project already has `go.yaml.in/yaml/v3` (v3.0.4) and `pgx/v5` (v5.8.0) in go.mod, and the `Serializer` interface is defined in `internal/core/interfaces.go` with `Serialize(def ObjectDef) ([]byte, error)` and `Deserialize(data []byte) (ObjectDef, error)`.

The YAML output must conform to the spec in `docs/pre/pgbackup_spec.md` section 5.2 (table def.yaml format). Data export uses `conn.PgConn().CopyTo()` with `COPY table_name TO STDOUT WITH (FORMAT CSV)` to stream CSV data to an `io.Writer`. Checksums are computed using Go's standard `crypto/sha256` with an `io.TeeReader` or `io.MultiWriter` pattern to avoid double-pass over the data. Row counts can be extracted from the `CommandTag` returned by `CopyTo`.

**Primary recommendation:** Follow the established per-kind package pattern (`internal/serialize/<kind>/serializer.go`) mirroring `internal/ddl/<kind>/generator.go` and `internal/fetch/<kind>/fetcher.go`. Use yaml struct tags on intermediate YAML-specific structs (not on core types directly) to match the spec's YAML field names exactly.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SRLZ-01 | Table def.yaml output (spec 5.2) | YAML struct with yaml tags mapping TableDef fields to spec 5.2 schema; go.yaml.in/yaml/v3 Marshal |
| SRLZ-02 | View, matview, function, trigger, sequence, type, domain, enum, policy YAML output | Per-kind serializer following same pattern; each has a YAML-specific struct |
| SRLZ-03 | Table data COPY TO CSV output | `conn.PgConn().CopyTo(ctx, writer, "COPY schema.table TO STDOUT WITH (FORMAT CSV)")` |
| SRLZ-04 | Partition parent/child split (parent no data, children each get data.csv) | Check `TableDef.Partitioning != nil` to skip parent data; iterate `Partitioning.Children` for child exports |
| SRLZ-05 | sha256 checksum and row count in data file metadata | `io.MultiWriter(file, sha256Hash)` for streaming checksum; `CommandTag` for row count; embed in DataMeta struct |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| go.yaml.in/yaml/v3 | v3.0.4 | YAML marshal/unmarshal | Already in go.mod; successor to gopkg.in/yaml.v3 (archived) |
| github.com/jackc/pgx/v5 | v5.8.0 | PostgreSQL COPY TO via PgConn.CopyTo | Already in go.mod; native COPY protocol support |
| crypto/sha256 | stdlib | SHA256 checksum computation | Go standard library; no external dep needed |
| encoding/hex | stdlib | Hex encoding for checksum strings | Go standard library |
| io | stdlib | TeeReader/MultiWriter for streaming checksum | Go standard library |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| fmt | stdlib | Error formatting | Type assertion errors in serializers |
| os | stdlib | File creation for data.csv | DataExporter writes to file |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go.yaml.in/yaml/v3 | goccy/go-yaml | goccy is faster but not in go.mod; yaml.v3 already used |
| io.MultiWriter | Two-pass (write then hash) | MultiWriter is single-pass, more efficient for large tables |

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── serialize/              # NEW: all serializers
│   ├── table/
│   │   └── serializer.go
│   ├── view/
│   │   └── serializer.go
│   ├── matview/
│   │   └── serializer.go
│   ├── function/
│   │   └── serializer.go
│   ├── trigger/
│   │   └── serializer.go
│   ├── sequence/
│   │   └── serializer.go
│   ├── compositetype/
│   │   └── serializer.go
│   ├── domain/
│   │   └── serializer.go
│   ├── enum/
│   │   └── serializer.go
│   ├── policy/
│   │   └── serializer.go
│   └── foreignkey/
│       └── serializer.go
├── export/                 # NEW: data exporter
│   └── csv.go              # DataExporter for COPY TO CSV + checksum
```

### Pattern 1: YAML Serializer with Intermediate Structs

**What:** Each serializer converts a `core.*Def` to an intermediate struct with yaml tags, then marshals to YAML. This decouples the in-memory model from the on-disk format.

**When to use:** All object types.

**Example:**
```go
// internal/serialize/table/serializer.go
package table

import (
    "fmt"
    "go.yaml.in/yaml/v3"
    "github.com/pgbkrs/pgbackup/internal/core"
)

var _ core.Serializer = (*Serializer)(nil)

type Serializer struct{}

// yamlTableDef mirrors spec 5.2 structure
type yamlTableDef struct {
    Kind    string          `yaml:"kind"`
    Schema  string          `yaml:"schema"`
    Name    string          `yaml:"name"`
    Columns []yamlColumn    `yaml:"columns"`
    Constraints yamlConstraints `yaml:"constraints"`
    Indexes []yamlIndex     `yaml:"indexes"`
    Partitioning *yamlPartition `yaml:"partitioning"`
    RLS     yamlRLS         `yaml:"rls"`
    Data    *yamlData       `yaml:"data,omitempty"`
}

type yamlColumn struct {
    Name     string `yaml:"name"`
    Type     string `yaml:"type"`
    Nullable bool   `yaml:"nullable"`
    Default  string `yaml:"default,omitempty"`
}

type yamlData struct {
    File     string `yaml:"file"`
    Columns  []string `yaml:"columns"`
    RowCount int64  `yaml:"row_count"`
    Checksum string `yaml:"checksum"`
}

func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
    td, ok := def.(core.TableDef)
    if !ok {
        return nil, fmt.Errorf("table.Serializer.Serialize: expected core.TableDef, got %T", def)
    }
    yd := toYAML(td)
    return yaml.Marshal(yd)
}
```

### Pattern 2: Data Exporter with Streaming Checksum

**What:** COPY TO CSV data is streamed through an `io.MultiWriter` that writes to both a file and a `sha256.Hash` simultaneously. Row count comes from the `CommandTag`.

**When to use:** Table data export (SRLZ-03, SRLZ-05).

**Example:**
```go
// internal/export/csv.go
package export

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "os"

    "github.com/jackc/pgx/v5"
)

type DataMeta struct {
    File     string
    Columns  []string
    RowCount int64
    Checksum string // "sha256:<hex>"
}

func ExportTableData(ctx context.Context, conn *pgx.Conn, schema, table string, filePath string, columns []string) (*DataMeta, error) {
    f, err := os.Create(filePath)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    h := sha256.New()
    w := io.MultiWriter(f, h)

    sql := fmt.Sprintf("COPY %s.%s TO STDOUT WITH (FORMAT CSV)", schema, table)
    tag, err := conn.PgConn().CopyTo(ctx, w, sql)
    if err != nil {
        return nil, err
    }

    checksum := "sha256:" + hex.EncodeToString(h.Sum(nil))
    return &DataMeta{
        File:     "data.csv",
        Columns:  columns,
        RowCount: tag.RowsAffected(),
        Checksum: checksum,
    }, nil
}
```

### Pattern 3: Partition-Aware Data Export

**What:** If `TableDef.Partitioning != nil`, skip data export for the parent table. Export each child table's data separately.

**When to use:** SRLZ-04.

**Example:**
```go
func shouldExportData(td *core.TableDef) bool {
    return td.Partitioning == nil // only non-partitioned tables get data.csv
}
// For partitioned tables, caller iterates td.Partitioning.Children
// and exports each child separately
```

### Pattern 4: Type Assertion Convention

**What:** Table DDL generators use pointer receivers (`*core.TableDef`), but view/etc. generators use value receivers (`core.ViewDef`). This project follows the convention established in Phase 4.

**When to use:** All serializers must match the same receiver pattern as their DDL generator counterparts.

**Critical detail from Phase 4 decisions:**
- Table, Sequence, ForeignKey: pointer type assertion (`*core.TableDef`)
- View, MatView, Function, Trigger, Type, Domain, Enum, Policy: value type assertion (`core.ViewDef`)

### Anti-Patterns to Avoid
- **Adding yaml tags directly to core types:** Core types are shared across fetch/ddl/serialize. Adding yaml tags would couple all layers to the on-disk format. Use intermediate YAML-specific structs instead.
- **Double-pass for checksum:** Reading the file after writing to compute checksum wastes I/O. Use `io.MultiWriter` for single-pass.
- **Using encoding/csv for COPY TO output:** PostgreSQL COPY CSV has its own escaping rules that differ from RFC 4180. Let PostgreSQL handle the format via `COPY TO ... WITH (FORMAT CSV)`.
- **Exporting data for partition parent tables:** Partition parents contain no data in PostgreSQL. Only child tables have rows.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CSV formatting for PG data | Custom CSV writer | `COPY TO STDOUT WITH (FORMAT CSV)` | PG COPY CSV handles NULLs, escaping, multi-line text correctly |
| YAML serialization | Manual string building | go.yaml.in/yaml/v3 Marshal | Edge cases in YAML quoting, multiline strings, special chars |
| SHA256 computation | Manual byte-level hashing | crypto/sha256 + io.MultiWriter | Standard library, streaming, correct |
| Row counting | Scanning output and counting newlines | CommandTag.RowsAffected() | COPY protocol returns exact row count |

**Key insight:** PostgreSQL's COPY protocol handles all the tricky CSV edge cases (NULL representation, escaping of quotes and newlines within fields). Let the database do the formatting; the Go side just streams bytes to disk.

## Common Pitfalls

### Pitfall 1: Value vs Pointer Type Assertions
**What goes wrong:** `def.(core.TableDef)` fails when `def` is `*core.TableDef` (and vice versa)
**Why it happens:** Phase 4 established that TableDef uses pointer receivers but ViewDef uses value receivers. Inconsistency across kinds.
**How to avoid:** Check the DDL generator for each kind to see which convention it uses. Table/Sequence/ForeignKey use `*core.XxxDef`; all others use `core.XxxDef`.
**Warning signs:** Runtime panic or "expected X, got Y" error in type assertion.

### Pitfall 2: YAML Field Name Mismatch with Spec
**What goes wrong:** YAML output uses Go field names (e.g., `PrimaryKey`) instead of spec names (e.g., `primary_key`)
**Why it happens:** Forgot to add yaml struct tags on intermediate structs.
**How to avoid:** Define yaml tags on every field of the YAML-specific structs. Cross-reference with spec section 5.2.
**Warning signs:** YAML output doesn't match the example in the spec doc.

### Pitfall 3: Forgetting omitempty for Optional Fields
**What goes wrong:** YAML output contains `partitioning: null` or `default: ""` cluttering the output.
**Why it happens:** yaml.v3 marshals zero values by default.
**How to avoid:** Use `omitempty` tag on optional fields (default, partitioning, data). But be careful: `nullable: false` must NOT be omitted (it's semantically meaningful), so don't use omitempty on bool fields where false is meaningful.
**Warning signs:** Unnecessary null/empty fields in output YAML.

### Pitfall 4: COPY TO with Schema-Qualified Table Names
**What goes wrong:** `COPY public.users TO STDOUT` works but `COPY "Public".users TO STDOUT` needs quoting for non-lowercase schemas.
**Why it happens:** PostgreSQL identifier quoting rules.
**How to avoid:** Always quote identifiers in the COPY command: `COPY "schema"."table" TO STDOUT WITH (FORMAT CSV)`.
**Warning signs:** Errors with mixed-case schema/table names.

### Pitfall 5: Partition Child Tables Are Regular Tables
**What goes wrong:** Child partition tables are already fetched as regular tables by the table fetcher (relkind='r'). Exporting data for both parent and children would cause confusion.
**Why it happens:** PostgreSQL partition children show up as relkind='r' in pg_class.
**How to avoid:** The parent (relkind='p') has `Partitioning != nil` -- skip data export for it. Children are regular tables and get exported normally. The parent's def.yaml records the children list; each child is independently serialized with its own def.yaml and data.csv.
**Warning signs:** Parent table has empty data.csv or children's data appears in parent.

### Pitfall 6: Large Table Memory Usage
**What goes wrong:** Loading entire table data into memory before writing.
**Why it happens:** Using bytes.Buffer as the writer for CopyTo.
**How to avoid:** Stream directly to `io.MultiWriter(file, hash)`. CopyTo streams data; never buffer it in memory.
**Warning signs:** OOM on large tables.

## Code Examples

### Table def.yaml Serialization (Spec 5.2 Conformant)
```go
// Intermediate YAML struct matching spec section 5.2
type yamlTableDef struct {
    Kind         string              `yaml:"kind"`
    Schema       string              `yaml:"schema"`
    Name         string              `yaml:"name"`
    Columns      []yamlColumn        `yaml:"columns"`
    Constraints  yamlConstraints     `yaml:"constraints"`
    Indexes      []yamlIndex         `yaml:"indexes,omitempty"`
    Partitioning *yamlPartitioning   `yaml:"partitioning"`
    RLS          yamlRLS             `yaml:"rls"`
    Data         *yamlData           `yaml:"data,omitempty"`
}

type yamlConstraints struct {
    PrimaryKey *yamlPK              `yaml:"primary_key,omitempty"`
    Unique     []yamlUnique         `yaml:"unique,omitempty"`
    Check      []yamlCheck          `yaml:"check,omitempty"`
}

type yamlPK struct {
    Name    string   `yaml:"name"`
    Columns []string `yaml:"columns,flow"`
}
```

### Deserialize Pattern
```go
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
    var yd yamlTableDef
    if err := yaml.Unmarshal(data, &yd); err != nil {
        return nil, fmt.Errorf("table.Serializer.Deserialize: %w", err)
    }
    return fromYAML(yd), nil
}
```

### COPY TO CSV with Streaming Checksum
```go
h := sha256.New()
f, _ := os.Create(filePath)
defer f.Close()
w := io.MultiWriter(f, h)

sql := fmt.Sprintf(`COPY "%s"."%s" TO STDOUT WITH (FORMAT CSV)`, schema, table)
tag, err := conn.PgConn().CopyTo(ctx, w, sql)
// tag.RowsAffected() => row count
// hex.EncodeToString(h.Sum(nil)) => sha256 hex string
```

### Simple Object Serializer (View Example)
```go
type yamlViewDef struct {
    Kind       string `yaml:"kind"`
    Schema     string `yaml:"schema"`
    Name       string `yaml:"name"`
    Definition string `yaml:"definition"`
    Owner      string `yaml:"owner"`
}

func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
    vd, ok := def.(core.ViewDef)
    if !ok {
        return nil, fmt.Errorf("view.Serializer.Serialize: expected core.ViewDef, got %T", def)
    }
    yd := yamlViewDef{
        Kind:       string(vd.Kind),
        Schema:     vd.Schema,
        Name:       vd.Name,
        Definition: vd.Definition,
        Owner:      vd.Owner,
    }
    return yaml.Marshal(yd)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| gopkg.in/yaml.v3 | go.yaml.in/yaml/v3 | 2024 (gopkg.in archived) | Project already uses new import path |
| pgx v4 CopyTo | pgx v5 conn.PgConn().CopyTo() | pgx v5 (2022) | Method is on PgConn, not Conn directly |
| Two-pass hash (write then read) | io.MultiWriter streaming | N/A (always available) | Single pass, half the I/O |

**Deprecated/outdated:**
- gopkg.in/yaml.v3: Archived, use go.yaml.in/yaml/v3 instead (project already does)
- pgx v4 CopyTo API: v5 moved it to PgConn sub-object

## Open Questions

1. **Grants/Permissions in def.yaml**
   - What we know: Spec 5.2 includes a `grants` section in table def.yaml. Current `TableDef` struct does NOT have a Grants field.
   - What's unclear: Whether grants should be added in this phase or deferred.
   - Recommendation: Omit grants from YAML output for now (no GrantDef in core types). This can be added later without breaking the serializer. The spec mentions it but the Phase 5 requirements don't explicitly include grants.

2. **Triggers reference in table def.yaml**
   - What we know: Spec 5.2 shows `triggers:` section in table def.yaml with refs to trigger yaml files.
   - What's unclear: Whether table serializer should include trigger refs (this cross-references data not in TableDef).
   - Recommendation: Defer trigger refs in table def.yaml to Phase 7 (backup orchestration) where the full directory structure is known. Phase 5 serializers focus on individual object serialization.

3. **ForeignKey def.yaml format**
   - What we know: ForeignKeyDef is an independent ObjectDef (Phase 4 decision). It needs its own serializer.
   - What's unclear: Exact YAML format -- spec doesn't have a dedicated FK YAML example.
   - Recommendation: Serialize ForeignKeyDef with kind/schema/name/source_table/target_schema/target_table/definition fields.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None (Go convention) |
| Quick run command | `go test -short ./internal/serialize/... ./internal/export/...` |
| Full suite command | `go test ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SRLZ-01 | Table def.yaml matches spec 5.2 | unit | `go test ./internal/serialize/table/... -run TestSerialize -x` | Wave 0 |
| SRLZ-01 | Table Deserialize round-trip | unit | `go test ./internal/serialize/table/... -run TestDeserialize -x` | Wave 0 |
| SRLZ-02 | View YAML output correct | unit | `go test ./internal/serialize/view/... -run TestSerialize -x` | Wave 0 |
| SRLZ-02 | MatView YAML output correct | unit | `go test ./internal/serialize/matview/... -run TestSerialize -x` | Wave 0 |
| SRLZ-02 | Function YAML output correct | unit | `go test ./internal/serialize/function/... -run TestSerialize -x` | Wave 0 |
| SRLZ-02 | Trigger YAML output correct | unit | `go test ./internal/serialize/trigger/... -run TestSerialize -x` | Wave 0 |
| SRLZ-02 | Sequence YAML output correct | unit | `go test ./internal/serialize/sequence/... -run TestSerialize -x` | Wave 0 |
| SRLZ-02 | Type YAML output correct | unit | `go test ./internal/serialize/compositetype/... -run TestSerialize -x` | Wave 0 |
| SRLZ-02 | Domain YAML output correct | unit | `go test ./internal/serialize/domain/... -run TestSerialize -x` | Wave 0 |
| SRLZ-02 | Enum YAML output correct | unit | `go test ./internal/serialize/enum/... -run TestSerialize -x` | Wave 0 |
| SRLZ-02 | Policy YAML output correct | unit | `go test ./internal/serialize/policy/... -run TestSerialize -x` | Wave 0 |
| SRLZ-03 | CSV data export via COPY TO | integration | `go test ./internal/export/... -run TestExportTableData` | Wave 0 |
| SRLZ-04 | Partition parent skipped, children exported | unit+integration | `go test ./internal/export/... -run TestPartition` | Wave 0 |
| SRLZ-05 | SHA256 checksum and row count correct | unit | `go test ./internal/export/... -run TestChecksum` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -short ./internal/serialize/... ./internal/export/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/serialize/table/serializer_test.go` -- covers SRLZ-01
- [ ] `internal/serialize/view/serializer_test.go` -- covers SRLZ-02 (view)
- [ ] `internal/serialize/matview/serializer_test.go` -- covers SRLZ-02 (matview)
- [ ] `internal/serialize/function/serializer_test.go` -- covers SRLZ-02 (function)
- [ ] `internal/serialize/trigger/serializer_test.go` -- covers SRLZ-02 (trigger)
- [ ] `internal/serialize/sequence/serializer_test.go` -- covers SRLZ-02 (sequence)
- [ ] `internal/serialize/compositetype/serializer_test.go` -- covers SRLZ-02 (type)
- [ ] `internal/serialize/domain/serializer_test.go` -- covers SRLZ-02 (domain)
- [ ] `internal/serialize/enum/serializer_test.go` -- covers SRLZ-02 (enum)
- [ ] `internal/serialize/policy/serializer_test.go` -- covers SRLZ-02 (policy)
- [ ] `internal/serialize/foreignkey/serializer_test.go` -- covers SRLZ-02 (fk)
- [ ] `internal/export/csv_test.go` -- covers SRLZ-03, SRLZ-04, SRLZ-05

## Sources

### Primary (HIGH confidence)
- `docs/pre/pgbackup_spec.md` sections 5.1-5.3 -- YAML and CSV format specifications
- `internal/core/interfaces.go` -- Serializer interface definition
- `internal/core/types.go` -- All *Def struct definitions
- `internal/ddl/*/generator.go` -- Established pattern for per-kind packages
- `go.mod` -- Exact library versions (yaml v3.0.4, pgx v5.8.0)
- [pgconn.PgConn.CopyTo docs](https://pkg.go.dev/github.com/jackc/pgx/v5/pgconn#PgConn.CopyTo) -- CopyTo(ctx, w io.Writer, sql string) signature

### Secondary (MEDIUM confidence)
- [go.yaml.in/yaml/v3 package docs](https://pkg.go.dev/go.yaml.in/yaml/v3) -- yaml struct tags, omitempty, flow
- [PostgreSQL COPY documentation](https://www.postgresql.org/docs/current/sql-copy.html) -- COPY TO STDOUT WITH (FORMAT CSV)
- [crypto/sha256 package docs](https://pkg.go.dev/crypto/sha256) -- streaming hash computation

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries already in go.mod, APIs verified against official docs
- Architecture: HIGH - follows established per-kind package pattern from Phase 2-4
- Pitfalls: HIGH - based on actual code inspection of existing type assertion patterns and Phase 4 decisions
- COPY TO CSV: HIGH - pgx PgConn.CopyTo signature verified against official package docs

**Research date:** 2026-03-11
**Valid until:** 2026-04-11 (stable libraries, no fast-moving dependencies)
