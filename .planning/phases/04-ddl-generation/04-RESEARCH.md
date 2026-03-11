# Phase 4: DDL Generation - Research

**Researched:** 2026-03-11
**Domain:** PostgreSQL DDL generation from in-memory Go struct definitions
**Confidence:** HIGH

## Summary

Phase 4 implements the `DDLGenerator` interface for all 10 object kinds already defined in `internal/core/types.go`. The interface is simple -- `GenerateDDL(ObjectDef) ([]string, error)` and `GenerateDrop(ObjectDef) ([]string, error)` -- and all input data structures are fully specified from Phases 2 and 3. This is a pure code-generation phase with no database queries; each generator takes an in-memory `*Def` struct and produces DDL strings via `fmt.Sprintf`.

The key architectural constraint is that table CREATE DDL must exclude FK constraints (DDLG-01), with FK ALTER TABLE DDL generated separately (DDLG-02). This is already baked into the design: `fetchConstraints` only fetches `contype IN ('p', 'u', 'c')` -- FK constraints (`contype='f'`) are deliberately excluded from `TableDef.Constraints`. FK constraints will need their own data structure and generator, producing `ALTER TABLE ... ADD CONSTRAINT ... FOREIGN KEY ...` statements.

The FunctionDef already contains the complete `CREATE OR REPLACE FUNCTION` statement from `pg_get_functiondef()`, so its generator is trivial (pass-through). Similarly, IndexDef stores the full `CREATE INDEX` statement from `pg_get_indexdef()`.

**Primary recommendation:** Create one generator per object kind in `internal/ddl/<kind>/generator.go`, each implementing `core.DDLGenerator`. Use `fmt.Sprintf` for DDL assembly. Add a new `ForeignKeyDef` struct to `core/types.go` and a corresponding FK fetcher query (contype='f') plus FK DDL generator. All generators are pure functions (no DB access), making them unit-testable without PostgreSQL.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DDLG-01 | Table CREATE DDL (FK constraint excluded) | TableDef struct already excludes FK. Generator assembles CREATE TABLE from Columns, Constraints (PK/Unique/Check only), Partitioning. Indexes generated as separate statements. |
| DDLG-02 | FK constraint ALTER TABLE DDL (separate phase) | Requires new ForeignKeyDef struct + SchemaFetcher for contype='f'. Generator produces ALTER TABLE ADD CONSTRAINT FOREIGN KEY. |
| DDLG-03 | View, MatView, Function, Trigger CREATE DDL | ViewDef/MatViewDef have Definition (SELECT body) -- wrap in CREATE VIEW/MATERIALIZED VIEW. FunctionDef.Definition is already complete DDL. TriggerDef fields map directly to CREATE TRIGGER syntax. |
| DDLG-04 | DROP DDL for all object types (10 kinds) | Each generator produces DROP IF EXISTS with CASCADE for its kind. Function DROP requires argument signature. Trigger DROP requires ON table_name. Policy DROP requires ON table_name. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| fmt (stdlib) | go1.24 | DDL string assembly via Sprintf | No external template engine needed for SQL generation |
| strings (stdlib) | go1.24 | String joining, quoting | Column lists, identifier handling |
| testing (stdlib) | go1.24 | Unit tests for generators | Pure function testing, no DB needed |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pgx/v5 | 5.8.0 | FK constraint fetching (new query) | Only for the new FK SchemaFetcher |
| testhelpers (internal) | - | Integration test DB setup | For round-trip tests that execute generated DDL |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| fmt.Sprintf | text/template | Overkill for SQL DDL; Sprintf is clearer for positional args |
| Manual quoting | pg_catalog quote_ident | Not needed -- all identifiers come from pg_catalog already properly formed |

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── core/
│   └── types.go           # Add ForeignKeyDef struct
├── ddl/
│   ├── table/
│   │   └── generator.go   # TableDDLGenerator (DDLG-01)
│   │   └── generator_test.go
│   ├── foreignkey/
│   │   └── generator.go   # FKDDLGenerator (DDLG-02)
│   │   └── generator_test.go
│   ├── view/
│   │   └── generator.go   # ViewDDLGenerator
│   │   └── generator_test.go
│   ├── matview/
│   │   └── generator.go   # MatViewDDLGenerator
│   │   └── generator_test.go
│   ├── function/
│   │   └── generator.go   # FunctionDDLGenerator
│   │   └── generator_test.go
│   ├── trigger/
│   │   └── generator.go   # TriggerDDLGenerator
│   │   └── generator_test.go
│   ├── sequence/
│   │   └── generator.go   # SequenceDDLGenerator
│   │   └── generator_test.go
│   ├── compositetype/
│   │   └── generator.go   # TypeDDLGenerator
│   │   └── generator_test.go
│   ├── domain/
│   │   └── generator.go   # DomainDDLGenerator
│   │   └── generator_test.go
│   ├── enum/
│   │   └── generator.go   # EnumDDLGenerator
│   │   └── generator_test.go
│   └── policy/
│       └── generator.go   # PolicyDDLGenerator
│       └── generator_test.go
├── fetch/
│   └── foreignkey/
│       └── fetcher.go     # FK SchemaFetcher (contype='f')
│       └── fetcher_test.go
```

### Pattern 1: DDLGenerator Implementation
**What:** Each generator is a struct implementing `core.DDLGenerator` with type assertion on the input `ObjectDef`.
**When to use:** Every object kind.
**Example:**
```go
package table

import (
    "fmt"
    "strings"
    "github.com/pgbkrs/pgbackup/internal/core"
)

var _ core.DDLGenerator = (*DDLGenerator)(nil)

type DDLGenerator struct{}

func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
    td, ok := def.(*core.TableDef)
    if !ok {
        return nil, fmt.Errorf("table DDLGenerator: expected *TableDef, got %T", def)
    }
    // Build CREATE TABLE statement
    var stmts []string
    stmts = append(stmts, buildCreateTable(td))
    // Indexes as separate statements
    for _, idx := range td.Indexes {
        stmts = append(stmts, idx.Definition)
    }
    return stmts, nil
}

func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
    td, ok := def.(*core.TableDef)
    if !ok {
        return nil, fmt.Errorf("table DDLGenerator: expected *TableDef, got %T", def)
    }
    return []string{fmt.Sprintf("DROP TABLE IF EXISTS %s.%s CASCADE", td.Schema, td.Name)}, nil
}
```

### Pattern 2: ForeignKeyDef as Independent Object
**What:** FK constraints stored as separate ObjectDef instances, not embedded in TableDef.
**When to use:** DDLG-02 requirement. This matches the project's architectural decision to isolate FK constraints.
**Example:**
```go
// In core/types.go
type ForeignKeyDef struct {
    ObjectHeader                    // Kind=KindForeignKey, Schema=source table schema, Name=constraint name
    SourceTable    string           // table that has the FK column(s)
    TargetSchema   string           // referenced table's schema
    TargetTable    string           // referenced table name
    Definition     string           // pg_get_constraintdef() output e.g. "FOREIGN KEY (col) REFERENCES other(id)"
}

// Generated DDL:
// ALTER TABLE schema.source_table ADD CONSTRAINT name FOREIGN KEY ...;
```

### Pattern 3: Compile-Time Interface Check
**What:** `var _ core.DDLGenerator = (*DDLGenerator)(nil)` at package level.
**When to use:** Every generator package. Matches established fetcher pattern.

### Anti-Patterns to Avoid
- **Embedding FK in table CREATE:** The entire project architecture separates FK constraints to break circular dependencies. Never include FK constraints in GenerateDDL for tables.
- **Quoting identifiers unnecessarily:** Identifiers from pg_catalog are already the canonical PostgreSQL form. Do not double-quote unless the name requires it (contains uppercase or special chars). Use schema-qualified `schema.name` format consistently.
- **Reconstructing function bodies:** FunctionDef.Definition already contains the complete DDL from pg_get_functiondef(). Do not parse or reconstruct it.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Function CREATE DDL | Reconstruct from ArgTypes/ReturnType/body | Pass through FunctionDef.Definition | pg_get_functiondef() output is already complete and handles all edge cases (SECURITY DEFINER, SET config, etc.) |
| Index CREATE DDL | Build from column names + method | Use IndexDef.Definition directly | pg_get_indexdef() output handles expression indexes, partial indexes, collations, etc. |
| Constraint definitions | Parse and reconstruct constraint text | Use pg_get_constraintdef() output in Definition fields | Already stores the canonical form |

**Key insight:** Phase 2/3 deliberately stored full DDL fragments from pg_get_*def() functions. Phase 4 generators should reuse these stored fragments rather than reconstructing DDL from decomposed fields.

## Common Pitfalls

### Pitfall 1: Forgetting Schema Qualification in DROP Statements
**What goes wrong:** `DROP TABLE mytable` drops from search_path, not the intended schema.
**Why it happens:** Name comes from ObjectHeader.Name which is unqualified.
**How to avoid:** Always use `schema.name` format: `DROP TABLE IF EXISTS %s.%s CASCADE`.
**Warning signs:** Tests pass because test schema is on search_path.

### Pitfall 2: Function DROP Requires Argument Signature
**What goes wrong:** `DROP FUNCTION schema.funcname` fails when overloaded functions exist.
**Why it happens:** PostgreSQL requires argument types to disambiguate overloaded functions.
**How to avoid:** Generate `DROP FUNCTION IF EXISTS schema.funcname(argtypes) CASCADE` using FunctionDef.ArgTypes.
**Warning signs:** Works in test but fails on production schemas with overloaded functions.

### Pitfall 3: Trigger DROP Requires Table Reference
**What goes wrong:** `DROP TRIGGER name` is invalid syntax.
**Why it happens:** PostgreSQL trigger names are scoped to their table.
**How to avoid:** `DROP TRIGGER IF EXISTS triggername ON schema.tablename CASCADE`.
**Warning signs:** Syntax error at or near "CASCADE".

### Pitfall 4: Policy DROP Requires Table Reference
**What goes wrong:** `DROP POLICY name` is invalid syntax.
**Why it happens:** Similar to triggers -- policies are scoped to a table.
**How to avoid:** `DROP POLICY IF EXISTS policyname ON schema.tablename`.
**Warning signs:** Syntax error in DROP statement.

### Pitfall 5: Sequence CREATE with Non-Default Values
**What goes wrong:** Omitting MINVALUE/MAXVALUE produces sequences with PostgreSQL defaults that may differ from the backed-up sequence.
**Why it happens:** PostgreSQL defaults vary by data type (bigint vs integer sequences).
**How to avoid:** Explicitly include all attributes: START, MINVALUE, MAXVALUE, INCREMENT, CACHE, CYCLE/NO CYCLE.
**Warning signs:** Restored sequence has different limits than original.

### Pitfall 6: Materialized View WITH NO DATA
**What goes wrong:** Creating a matview always populates it, changing semantics.
**Why it happens:** Default CREATE MATERIALIZED VIEW populates data.
**How to avoid:** When `IsPopulated=false`, append `WITH NO DATA` to the CREATE statement.
**Warning signs:** Matview that was empty now has data after restore.

### Pitfall 7: Domain CHECK Constraint Name Preservation
**What goes wrong:** Domain recreated without preserving original check constraint name.
**Why it happens:** Using inline CHECK without CONSTRAINT keyword.
**How to avoid:** Use `CONSTRAINT checkname CHECK (definition)` when CheckName is non-empty.
**Warning signs:** Diff shows constraint name change after backup/restore cycle.

### Pitfall 8: Partitioned Table CREATE Must Include PARTITION BY
**What goes wrong:** Partitioned table created as regular table.
**Why it happens:** Forgetting to check PartitionDef and append PARTITION BY clause.
**How to avoid:** When `td.Partitioning != nil`, append `PARTITION BY strategy (columns)`.
**Warning signs:** Table created but partitions cannot be attached.

## Code Examples

### Table CREATE DDL Assembly
```go
func buildCreateTable(td *core.TableDef) string {
    var b strings.Builder
    fmt.Fprintf(&b, "CREATE TABLE %s.%s (\n", td.Schema, td.Name)

    // Columns
    for i, col := range td.Columns {
        fmt.Fprintf(&b, "    %s %s", col.Name, col.Type)
        if !col.Nullable {
            b.WriteString(" NOT NULL")
        }
        if col.Default != "" {
            fmt.Fprintf(&b, " DEFAULT %s", col.Default)
        }
        if i < len(td.Columns)-1 || hasInlineConstraints(td) {
            b.WriteString(",")
        }
        b.WriteString("\n")
    }

    // PK constraint
    if td.Constraints.PrimaryKey != nil {
        pk := td.Constraints.PrimaryKey
        fmt.Fprintf(&b, "    CONSTRAINT %s PRIMARY KEY (%s)",
            pk.Name, strings.Join(pk.Columns, ", "))
        if len(td.Constraints.Unique) > 0 || len(td.Constraints.Check) > 0 {
            b.WriteString(",")
        }
        b.WriteString("\n")
    }

    // Unique constraints
    for i, uc := range td.Constraints.Unique {
        fmt.Fprintf(&b, "    CONSTRAINT %s %s", uc.Name, uc.Definition)
        if i < len(td.Constraints.Unique)-1 || len(td.Constraints.Check) > 0 {
            b.WriteString(",")
        }
        b.WriteString("\n")
    }

    // Check constraints
    for i, cc := range td.Constraints.Check {
        fmt.Fprintf(&b, "    CONSTRAINT %s %s", cc.Name, cc.Definition)
        if i < len(td.Constraints.Check)-1 {
            b.WriteString(",")
        }
        b.WriteString("\n")
    }

    b.WriteString(")")

    // Partitioning
    if td.Partitioning != nil {
        // Note: partition key columns need to be extracted or stored.
        // This is a known gap -- see Open Questions.
        fmt.Fprintf(&b, " PARTITION BY %s (...)", strings.ToUpper(td.Partitioning.Strategy))
    }

    return b.String()
}
```

### View CREATE DDL
```go
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
    vd, ok := def.(core.ViewDef)
    if !ok {
        return nil, fmt.Errorf("view DDLGenerator: expected ViewDef, got %T", def)
    }
    stmt := fmt.Sprintf("CREATE VIEW %s.%s AS\n%s", vd.Schema, vd.Name, vd.Definition)
    return []string{stmt}, nil
}
```

### Trigger CREATE DDL
```go
func buildCreateTrigger(td core.TriggerDef) string {
    events := strings.Join(td.Events, " OR ")
    return fmt.Sprintf(
        "CREATE TRIGGER %s\n    %s %s ON %s.%s\n    FOR EACH ROW EXECUTE FUNCTION %s.%s()",
        td.Name, td.Timing, events, td.Schema, td.TableName, td.Schema, td.FunctionName,
    )
}
```

### Sequence CREATE DDL
```go
func buildCreateSequence(sd core.SequenceDef) string {
    cycle := "NO CYCLE"
    if sd.Cycle {
        cycle = "CYCLE"
    }
    return fmt.Sprintf(
        "CREATE SEQUENCE %s.%s START %d MINVALUE %d MAXVALUE %d INCREMENT BY %d %s CACHE %d",
        sd.Schema, sd.Name, sd.StartValue, sd.MinValue, sd.MaxValue,
        sd.IncrementBy, cycle, sd.Cache,
    )
}
```

### DROP DDL Patterns (All Types)
```go
// Table:      DROP TABLE IF EXISTS schema.name CASCADE
// View:       DROP VIEW IF EXISTS schema.name CASCADE
// MatView:    DROP MATERIALIZED VIEW IF EXISTS schema.name CASCADE
// Function:   DROP FUNCTION IF EXISTS schema.name(argtypes) CASCADE
// Trigger:    DROP TRIGGER IF EXISTS name ON schema.tablename CASCADE
// Sequence:   DROP SEQUENCE IF EXISTS schema.name CASCADE
// Type:       DROP TYPE IF EXISTS schema.name CASCADE
// Domain:     DROP DOMAIN IF EXISTS schema.name CASCADE
// Enum:       DROP TYPE IF EXISTS schema.name CASCADE  (ENUMs are types in PG)
// Policy:     DROP POLICY IF EXISTS name ON schema.tablename
// ForeignKey: ALTER TABLE schema.sourcetable DROP CONSTRAINT IF EXISTS name CASCADE
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Reconstruct function DDL from parts | pg_get_functiondef() passthrough | Phase 3 decision | FunctionDDLGenerator is trivial |
| FK in table constraints | FK as independent objects | Project design decision | Separate ForeignKeyDef + generator |
| Complex template engines | fmt.Sprintf DDL assembly | Go standard practice | No dependencies needed |

## Open Questions

1. **Partition Key Columns**
   - What we know: `PartitionDef` stores `Strategy` and `Children`, but NOT the partition key columns.
   - What's unclear: `PARTITION BY RANGE (created_at)` needs the column name(s), which are not currently in PartitionDef.
   - Recommendation: Either (a) extend PartitionDef to include key column expression (fetched via `pg_get_partkeydef(oid)`), or (b) store the partition key expression as a string field. This requires a small addition to `fetchPartitioning()` in the table fetcher. This is a Phase 2 gap that must be addressed in Phase 4.

2. **ForeignKeyDef Does Not Exist Yet**
   - What we know: FK constraints are deliberately excluded from TableDef (contype='f' not fetched). The project decision is to treat them as independent objects.
   - What's unclear: There is no `KindForeignKey` constant, no `ForeignKeyDef` struct, and no FK SchemaFetcher.
   - Recommendation: Phase 4 must add these. The FK fetcher query is straightforward: `SELECT conname, conrelid table, confrelid reftable, pg_get_constraintdef(oid) FROM pg_constraint WHERE contype='f' AND schema=$1`.

3. **Pointer vs Value Receivers for ObjectDef**
   - What we know: TableDef uses pointer receiver (`*core.TableDef`) in fetcher results, while ViewDef, FunctionDef etc. use value receivers. Type assertion in generators must match.
   - What's unclear: Whether this inconsistency is intentional.
   - Recommendation: Generators should handle both via a type switch checking pointer first, then value. Or standardize on one. The test for `findTable` already casts to `*core.TableDef`.

4. **Trigger FOR EACH ROW vs FOR EACH STATEMENT**
   - What we know: TriggerDef does not store row-level vs statement-level granularity. The current test fixtures only use FOR EACH ROW.
   - What's unclear: Whether the fetcher captures this (tgtype bit 0x01 = FOR EACH ROW).
   - Recommendation: For Phase 4, default to FOR EACH ROW if not captured. This may need a fetcher enhancement but is not blocking for DDL generation if the test fixtures only use row-level triggers.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | go test (stdlib) |
| Config file | none (go test convention) |
| Quick run command | `go test ./internal/ddl/... -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DDLG-01 | Table CREATE DDL without FK | unit | `go test ./internal/ddl/table/ -run TestGenerateDDL -count=1` | Wave 0 |
| DDLG-01 | Table CREATE includes PK, unique, check, columns, partitioning | unit | `go test ./internal/ddl/table/ -run TestGenerateDDL -count=1` | Wave 0 |
| DDLG-01 | Table CREATE excludes FK constraints | unit | `go test ./internal/ddl/table/ -run TestGenerateDDL_NoFK -count=1` | Wave 0 |
| DDLG-02 | FK ALTER TABLE DDL generation | unit | `go test ./internal/ddl/foreignkey/ -run TestGenerateDDL -count=1` | Wave 0 |
| DDLG-03 | View CREATE DDL | unit | `go test ./internal/ddl/view/ -run TestGenerateDDL -count=1` | Wave 0 |
| DDLG-03 | MatView CREATE DDL (with/without NO DATA) | unit | `go test ./internal/ddl/matview/ -run TestGenerateDDL -count=1` | Wave 0 |
| DDLG-03 | Function CREATE DDL (passthrough) | unit | `go test ./internal/ddl/function/ -run TestGenerateDDL -count=1` | Wave 0 |
| DDLG-03 | Trigger CREATE DDL | unit | `go test ./internal/ddl/trigger/ -run TestGenerateDDL -count=1` | Wave 0 |
| DDLG-04 | DROP DDL for all 10 object types | unit | `go test ./internal/ddl/... -run TestGenerateDrop -count=1` | Wave 0 |
| DDLG-04 | Function DROP includes arg signature | unit | `go test ./internal/ddl/function/ -run TestGenerateDrop -count=1` | Wave 0 |
| DDLG-04 | Trigger DROP includes ON table | unit | `go test ./internal/ddl/trigger/ -run TestGenerateDrop -count=1` | Wave 0 |
| DDLG-04 | Policy DROP includes ON table | unit | `go test ./internal/ddl/policy/ -run TestGenerateDrop -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/ddl/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/ddl/table/generator_test.go` -- covers DDLG-01
- [ ] `internal/ddl/foreignkey/generator_test.go` -- covers DDLG-02
- [ ] `internal/ddl/view/generator_test.go` -- covers DDLG-03 (view)
- [ ] `internal/ddl/matview/generator_test.go` -- covers DDLG-03 (matview)
- [ ] `internal/ddl/function/generator_test.go` -- covers DDLG-03 (function)
- [ ] `internal/ddl/trigger/generator_test.go` -- covers DDLG-03 (trigger)
- [ ] `internal/ddl/sequence/generator_test.go` -- covers DDLG-04 (sequence)
- [ ] `internal/ddl/compositetype/generator_test.go` -- covers DDLG-04 (type)
- [ ] `internal/ddl/domain/generator_test.go` -- covers DDLG-04 (domain)
- [ ] `internal/ddl/enum/generator_test.go` -- covers DDLG-04 (enum)
- [ ] `internal/ddl/policy/generator_test.go` -- covers DDLG-04 (policy)
- [ ] `internal/core/types.go` -- add ForeignKeyDef, KindForeignKey
- [ ] `internal/fetch/foreignkey/fetcher.go` -- FK constraint fetcher (contype='f')
- [ ] `internal/fetch/foreignkey/fetcher_test.go` -- FK fetcher integration test

## Sources

### Primary (HIGH confidence)
- `internal/core/types.go` -- all *Def struct definitions reviewed
- `internal/core/interfaces.go` -- DDLGenerator interface specification
- `internal/core/registry.go` -- CommandRegistry pattern
- `internal/fetch/table/fetcher.go` -- FK exclusion confirmed (contype IN ('p', 'u', 'c'))
- `internal/fetch/*/fetcher.go` -- all 10 fetcher implementations reviewed
- `internal/fetch/testhelpers/setup.go` -- test fixture DDL patterns
- PostgreSQL documentation (training data) -- DDL syntax for all object types

### Secondary (MEDIUM confidence)
- Partition key extraction via `pg_get_partkeydef()` -- known PostgreSQL function but not yet used in codebase

### Tertiary (LOW confidence)
- Trigger FOR EACH ROW/STATEMENT granularity -- need to verify tgtype bitmask bit 0x01 handling

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - stdlib only, no new dependencies needed for generators
- Architecture: HIGH - mirrors established fetcher pattern, interface already defined
- Pitfalls: HIGH - PostgreSQL DDL syntax is well-documented and stable
- Open questions: MEDIUM - partition key gap and ForeignKeyDef are real gaps requiring resolution

**Research date:** 2026-03-11
**Valid until:** 2026-04-11 (stable domain, no breaking changes expected)
