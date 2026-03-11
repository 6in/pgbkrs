# Phase 7: Backup Orchestration - Research

**Researched:** 2026-03-11
**Domain:** Go backup orchestration — directory layout, skip detection, transaction isolation, CLI wiring
**Confidence:** HIGH

## Summary

Phase 7 is the integration layer that wires together all previously built components (Phases 1-6) into a functional `pgbackup backup` command. The command must enumerate schemas, fetch all object kinds, detect and skip unsupported column types, export data via COPY TO, serialize YAML definitions, resolve dependency order, and write the complete directory tree specified in spec section 4.

All the hard pieces are done: fetchers exist for every object kind, serializers produce spec-compliant YAML, `ExportTableData` does CSV export with checksum, `BuildManifest` generates `_manifest.yaml`, and `BuildRestoreOrder` determines restore order. Phase 7 stitches them together via a new `internal/backup` package and wires it into `cmd/pgbackup/cmd/backup.go`.

The two distinct operation modes (snapshot vs. non-snapshot) affect only how the `pgx.Conn` is used — snapshot mode opens a `REPEATABLE READ` transaction that is held open across all fetches and COPY operations, while non-snapshot mode calls each operation individually on the shared connection. The skip detection (BKUP-02) requires scanning `TableDef.Columns` for `bytea`, `xml`, `pg_lsn`, and `txid_snapshot` column types before the table reaches serialization or export.

**Primary recommendation:** Create `internal/backup/orchestrator.go` as the single orchestration entry point that the `backupCmd` calls. Keep all directory-creation and file-writing logic in the orchestrator, not in the command layer.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| BKUP-01 | スキーマ単位のディレクトリ構造生成（仕様書セクション4準拠） | Spec section 4 defines exact directory layout; `os.MkdirAll` creates nested paths; serializer output maps to per-kind subdirectories |
| BKUP-02 | スキップ対象テーブル検出（bytea, xml, pg_lsn, txid_snapshot列）と警告ログ | `TableDef.Columns[].Type` contains `format_type()` output; type matching must handle qualified and array variants; `log.Printf` or `fmt.Fprintf(os.Stderr)` for warning |
| BKUP-03 | --snapshotオプションによるトランザクション一貫性モード | pgx `conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})` opens snapshot-consistent transaction; all fetch and COPY operations run within it |
| BKUP-04 | スナップショットなしモード（デフォルト、各オブジェクト個別取得） | Default path: pass `*pgx.Conn` directly to all fetchers and `ExportTableData`; no transaction wrapping |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/jackc/pgx/v5` | v5.8.0 | DB connection, COPY TO, transactions | Already in go.mod; used in all phases |
| `go.yaml.in/yaml/v3` | v3.0.4 | YAML serialization | Project decision (Phase 1: over gopkg.in) |
| `os` (stdlib) | Go 1.24 | `MkdirAll`, `WriteFile`, `Create` | Directory + file creation |
| `time` (stdlib) | Go 1.24 | `time.Now().UTC().Format(time.RFC3339)` for `backup_at` | Timestamp generation |
| `strings` (stdlib) | Go 1.24 | Column type substring matching for skip detection | Already used throughout |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `fmt` (stdlib) | Go 1.24 | Warning message formatting | Logging skipped tables |
| `context` (stdlib) | Go 1.24 | `context.Background()` passed through all fetch/export calls | All async operations |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `os.MkdirAll` | `os.Mkdir` per level | MkdirAll is idempotent; handles nested paths in one call |
| `pgx.RepeatableRead` | `pgx.Serializable` | Repeatable Read is sufficient for consistent snapshot reads; Serializable adds unnecessary write-conflict overhead |

**Installation:**
No new dependencies needed. All required packages are in go.mod already.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── backup/
│   └── orchestrator.go     # RunBackup(ctx, conn, outDir, snapshot) error
│   └── orchestrator_test.go
cmd/pgbackup/cmd/
└── backup.go               # Wire --snapshot flag + call backup.RunBackup
```

### Pattern 1: Orchestrator Function
**What:** A single exported function `RunBackup` in `internal/backup` that takes the live `*pgx.Conn`, output directory path, and snapshot bool. It drives the full backup pipeline.
**When to use:** Always — keeps backup logic testable independently of the Cobra command layer.
**Example:**
```go
// internal/backup/orchestrator.go
func RunBackup(ctx context.Context, conn *pgx.Conn, outDir string, snapshot bool) error {
    // 1. Create timestamped backup root: outDir/backup_YYYYMMDD_HHMMSS/
    // 2. If snapshot: begin REPEATABLE READ tx
    // 3. For each schema (initially "public" or flag-provided list):
    //    - Fetch all object kinds via registry
    //    - Classify tables: skip vs. export
    //    - Write per-kind YAML files under schema/ subdirectory
    //    - Export table data to tables/<name>/data.csv
    // 4. Build manifest from all collected objects + skip list
    // 5. Write _manifest.yaml at backup root
    // 6. If snapshot: commit tx
}
```

### Pattern 2: Schema Discovery
**What:** Query `pg_namespace` to list non-system schemas, then iterate.
**When to use:** When backing up multiple schemas (default behavior should back up all user schemas).
**Example:**
```go
// Source: pg_catalog direct query
rows, err := conn.Query(ctx,
    `SELECT nspname FROM pg_namespace
     WHERE nspname NOT IN ('pg_catalog','information_schema')
       AND nspname NOT LIKE 'pg_%'
     ORDER BY nspname`)
```

### Pattern 3: Skip Detection
**What:** Before processing a `TableDef`, scan `Columns` for unsupported types.
**When to use:** Applied to every `TableDef` returned by the table fetcher.
**Example:**
```go
// skipTypes are the base type names triggering skip per spec section 3.3
var skipTypes = []string{"bytea", "xml", "pg_lsn", "txid_snapshot"}

func shouldSkip(td *core.TableDef) (bool, string) {
    for _, col := range td.Columns {
        for _, bad := range skipTypes {
            // format_type() may return "bytea[]", "_bytea", etc.
            if strings.Contains(col.Type, bad) {
                return true, fmt.Sprintf("contains %s column: %s", bad, col.Name)
            }
        }
    }
    return false, ""
}
```

### Pattern 4: Directory Layout per Spec Section 4
**What:** Create directories matching the spec layout.
**When to use:** Whenever writing any object file.
```
<backup_root>/
├── _manifest.yaml
└── <schema>/
    ├── sequences/
    │   └── <name>.yaml
    ├── tables/
    │   └── <name>/
    │       ├── def.yaml
    │       └── data.csv          (non-partitioned tables only)
    │       └── partitions/       (partitioned tables only)
    │           └── <child>/
    │               ├── def.yaml
    │               └── data.csv
    ├── views/
    │   └── <name>.yaml
    ├── materialized_views/
    │   └── <name>.yaml
    ├── functions/
    │   └── <name>.yaml
    ├── triggers/
    │   └── <name>.yaml
    ├── policies/
    │   └── <name>.yaml
    ├── types/
    │   └── <name>.yaml
    ├── enums/
    │   └── <name>.yaml
    └── domains/
        └── <name>.yaml
```

**Mapping ObjectKind to subdirectory:**
```go
var kindToDir = map[core.ObjectKind]string{
    core.KindTable:            "tables",
    core.KindView:             "views",
    core.KindMaterializedView: "materialized_views",
    core.KindFunction:         "functions",
    core.KindSequence:         "sequences",
    core.KindTrigger:          "triggers",
    core.KindType:             "types",
    core.KindDomain:           "domains",
    core.KindEnum:             "enums",
    core.KindPolicy:           "policies",
}
```

Note: Tables get a subdirectory (`tables/<name>/def.yaml`), all other kinds get a flat file (`<kind-dir>/<name>.yaml`). ForeignKeyDef is manifest-only — no YAML file written.

### Pattern 5: Snapshot Mode with pgx
**What:** Use `pgx.Conn.BeginTx` with `RepeatableRead` isolation to hold a consistent snapshot.
**When to use:** When `--snapshot` flag is set.
**Example:**
```go
// Source: pgx/v5 transaction API
tx, err := conn.BeginTx(ctx, pgx.TxOptions{
    IsoLevel: pgx.RepeatableRead,
})
if err != nil {
    return fmt.Errorf("begin snapshot tx: %w", err)
}
defer tx.Rollback(ctx) // safe no-op if committed

// All fetch/COPY calls use tx.Conn() or pass tx directly
// ...
if err := tx.Commit(ctx); err != nil {
    return fmt.Errorf("commit snapshot tx: %w", err)
}
```

Note: `pgx.Tx` embeds the same query interface as `*pgx.Conn`. However, `ExportTableData` takes a `*pgx.Conn`. In snapshot mode, use `tx.Conn()` to get the underlying connection — COPY operations issued on that connection participate in the transaction because pgx v5 uses a single underlying `PgConn`.

### Pattern 6: Wiring the --snapshot flag in backup.go
**What:** Add a `--snapshot` persistent flag to `backupCmd` and pass it to `RunBackup`.
**When to use:** Always — matches spec section 3.4.
**Example:**
```go
var snapshotFlag bool

var backupCmd = &cobra.Command{
    Use:   "backup",
    Short: "Back up PostgreSQL schema and data to YAML+CSV files",
    RunE: func(cmd *cobra.Command, args []string) error {
        return backup.RunBackup(cmd.Context(), conn, ".", snapshotFlag)
    },
}

func init() {
    backupCmd.Flags().BoolVar(&snapshotFlag, "snapshot", false,
        "Use REPEATABLE READ transaction for consistent snapshot")
}
```

### Pattern 7: Writing Table def.yaml with DataMeta
**What:** The table serializer's `yamlTableDef` has a `Data *yamlData` field. After CSV export, populate it before serializing.
**When to use:** For every non-skipped, non-partitioned table.

The existing `table.Serializer.Serialize(def)` method serializes a `*core.TableDef` but the `yamlTableDef.Data` field is populated from the export result. Because the serializer's `toYAML` function builds the struct, there is no current path to inject `DataMeta` into the YAML output from outside. **This is a key gap** — the orchestrator will need to either:
- (a) Call a new `SerializeWithData(def, meta)` method added to `table.Serializer`, or
- (b) Directly marshal the YAML struct in the orchestrator (bypassing the Serializer interface), or
- (c) Extend `core.TableDef` with an optional `Data` field and populate it before calling `Serialize`

Option (c) is most consistent with the existing pattern: add an exported `DataMeta` field to `core.TableDef` (or a wrapper struct) and have `table/serializer.go` emit it when non-nil. This avoids breaking the `Serializer` interface contract.

### Anti-Patterns to Avoid
- **Writing files from individual serializer packages:** Keep all file I/O in the orchestrator. Serializers return `[]byte`, orchestrator writes the file.
- **One fetcher call per schema discovery pass:** Fetch all kinds in a single schema pass so transaction state (if snapshot mode) is consistent.
- **Hardcoding schema list:** Discover user schemas from `pg_namespace` dynamically; allow override via CLI flag.
- **Skipping tables after COPY:** Detect skip-eligible tables before any I/O. Do not create the table directory and then delete it.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CSV streaming + checksum | Custom write loop | `export.ExportTableData` (Phase 5) | Already implements `io.MultiWriter(file, hash)` + COPY TO |
| Dependency ordering | Custom topological sort | `resolve.BuildRestoreOrder` (Phase 6) | Kahn's algorithm with FK separation already tested |
| Manifest YAML | Custom YAML builder | `resolve.BuildManifest` + `resolve.WriteManifest` (Phase 6) | Spec-compliant structure fully tested |
| YAML serialization per kind | Custom marshaling | Each `internal/serialize/<kind>/serializer.go` | All 10 object kinds have tested serializers |
| pg_catalog fetching | New SQL queries | Each `internal/fetch/<kind>/fetcher.go` | All 10 object kinds have tested fetchers |
| Transaction isolation | Manual `BEGIN` SQL | `pgx.Conn.BeginTx` with `pgx.TxOptions` | pgx handles savepoints, error propagation correctly |

**Key insight:** Phase 7 is a glue phase. The complexity budget is in orchestration logic (directory creation, skip detection, mode branching), not in any new algorithm or data-transformation component.

## Common Pitfalls

### Pitfall 1: Column Type Matching for Skip Detection
**What goes wrong:** `format_type()` in PostgreSQL returns decorated names: `bytea[]` for bytea arrays, `character varying(255)` not `varchar`. A simple `== "bytea"` check misses arrays and domains over bytea.
**Why it happens:** The fetcher stores the raw `format_type()` output without normalization.
**How to avoid:** Use `strings.Contains(col.Type, bad)` as shown in Pattern 3. This catches `bytea`, `bytea[]`, `_bytea` (internal array name).
**Warning signs:** Integration test passes with plain `bytea` column but fails when column is `bytea[]`.

### Pitfall 2: DataMeta not in def.yaml
**What goes wrong:** `table.Serializer.Serialize` currently ignores DataMeta — the `Data` field in `yamlTableDef` is always nil. The resulting `def.yaml` has no `data:` section even after CSV export.
**Why it happens:** The serializer was built in Phase 5 before CSV export results were available at serialize time. See the `toYAML` function — it never sets `yd.Data`.
**How to avoid:** The orchestrator must either (a) extend `core.TableDef` with a `DataMeta` field, (b) add a dedicated serialize method, or (c) build the final YAML struct directly. Choose (a) or a new `SerializeTableWithData(def, meta)` function — must be decided in planning.
**Warning signs:** `def.yaml` files missing the `data:` block after a full backup run.

### Pitfall 3: Snapshot Mode and ExportTableData Signature
**What goes wrong:** `export.ExportTableData` accepts `*pgx.Conn` (not `pgx.Tx`). In snapshot mode, the transaction is a `pgx.Tx`. Calling `ExportTableData` on a new `conn` breaks the snapshot.
**Why it happens:** COPY TO uses `conn.PgConn().CopyTo(...)` which bypasses the transaction object.
**How to avoid:** In snapshot mode, pass `tx.Conn()` to `ExportTableData`. In pgx v5, `tx.Conn()` returns the underlying `*pgx.Conn` that is currently in-transaction. COPY operations issued on it share the snapshot.
**Warning signs:** No error from COPY, but reads in snapshot mode are inconsistent with schema reads.

### Pitfall 4: Partition Child Tables Appearing Twice
**What goes wrong:** If the table fetcher returns both the parent (relkind='p') and all children (relkind='r', which are regular tables that happen to be partition children), the orchestrator writes separate directories for the children at `tables/<child>/` AND inside `tables/<parent>/partitions/<child>/`.
**Why it happens:** `pg_class` lists all relkind='r' and 'p' tables; partition children are relkind='r'.
**How to avoid:** Detect partition parents via `TableDef.Partitioning != nil`. Place child tables under `tables/<parent>/partitions/<child>/`. A child can be detected by querying `pg_inherits`, or by cross-referencing parent `Partitioning.Children` list against all `TableDef` names. The orchestrator should build a `childTables map[string]string` (child name → parent name) to route each child to the correct path.
**Warning signs:** `tables/orders_2024/def.yaml` exists at top-level AND inside `tables/orders/partitions/orders_2024/`.

### Pitfall 5: Backup Root Directory Naming
**What goes wrong:** Two backup runs in the same second produce identical `backup_YYYYMMDD_HHMMSS/` directories and one overwrites the other.
**Why it happens:** Second-precision timestamp collision during testing or rapid re-runs.
**How to avoid:** Use `time.Now().UTC().Format("backup_20060102_150405")` and check for existence; or accept overwrite as intended behavior (spec does not address this). The simplest safe approach: create the directory, fail if it already exists.
**Warning signs:** Silent data loss in unit tests running in parallel.

### Pitfall 6: Manifest BackupAt and PgVersion
**What goes wrong:** `ManifestParams.PgVersion` is left empty because there is no established pattern for fetching the PostgreSQL server version.
**Why it happens:** No existing code fetches `server_version`. The manifest struct was built to accept it as a string parameter.
**How to avoid:** Add a small helper function: `SELECT current_setting('server_version')` or `SHOW server_version` via a single `conn.QueryRow`. Fetch this at the start of `RunBackup`.
**Warning signs:** `_manifest.yaml` has `pg_version: ""` after backup.

## Code Examples

Verified patterns from official sources:

### pgx v5 Transaction with RepeatableRead
```go
// Source: pgx/v5 godoc — pgx.TxOptions
tx, err := conn.BeginTx(ctx, pgx.TxOptions{
    IsoLevel:   pgx.RepeatableRead,
    AccessMode: pgx.ReadOnly, // READ ONLY is safe for backups, prevents accidental writes
})
```

Note: `pgx.ReadOnly` in combination with `RepeatableRead` is valid for backup use. It does NOT block COPY TO (which is a read from the server's perspective). Confirm this is acceptable before setting ReadOnly — the spec does not require it but it is a safety net.

### Fetch Server Version
```go
// Source: PostgreSQL docs — server_version GUC
var pgVersion string
err := conn.QueryRow(ctx, "SELECT current_setting('server_version')").Scan(&pgVersion)
```

### Directory Creation (all parents)
```go
// Source: Go stdlib os.MkdirAll
path := filepath.Join(backupRoot, schema, "tables", tableName)
if err := os.MkdirAll(path, 0755); err != nil {
    return fmt.Errorf("create table dir %s: %w", path, err)
}
```

### Write YAML def file
```go
// Pattern used by WriteManifest in resolve/manifest.go
data, err := serializer.Serialize(def)
if err != nil {
    return fmt.Errorf("serialize %s: %w", def.Header().Name, err)
}
if err := os.WriteFile(filepath.Join(dir, "def.yaml"), data, 0644); err != nil {
    return fmt.Errorf("write def.yaml: %w", err)
}
```

### Detect Skippable Table
```go
var skipTypes = []string{"bytea", "xml", "pg_lsn", "txid_snapshot"}

func skipReason(td *core.TableDef) string {
    for _, col := range td.Columns {
        for _, bad := range skipTypes {
            if strings.Contains(col.Type, bad) {
                return fmt.Sprintf("contains %s column: %s", bad, col.Name)
            }
        }
    }
    return ""
}
```

### Partition Child Routing
```go
// Build child→parent map from all partition parents
childOf := make(map[string]string) // child name → parent name
for _, def := range tableDefs {
    td := def.(*core.TableDef)
    if td.Partitioning != nil {
        for _, child := range td.Partitioning.Children {
            childOf[child] = td.Name
        }
    }
}

// When writing a table:
func tableDir(backupRoot, schema string, td *core.TableDef, childOf map[string]string) string {
    if parent, ok := childOf[td.Name]; ok {
        return filepath.Join(backupRoot, schema, "tables", parent, "partitions", td.Name)
    }
    return filepath.Join(backupRoot, schema, "tables", td.Name)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `backup.go` stub with `return nil` | Full orchestrator in `internal/backup` | Phase 7 | `backupCmd.RunE` becomes a thin wrapper |
| DataMeta absent from def.yaml | DataMeta injected before serialization | Phase 7 | `def.yaml` gains the `data:` block |
| No schema list | Dynamic schema discovery from `pg_namespace` | Phase 7 | Backs up all user schemas automatically |

## Open Questions

1. **DataMeta injection into table serializer**
   - What we know: `table.Serializer.toYAML` never sets `yd.Data`; `yamlTableDef.Data` struct exists but is always nil.
   - What's unclear: Whether to extend `core.TableDef` with an optional `*export.DataMeta` field, or add a new function `SerializeTableWithData`.
   - Recommendation: Add `DataMeta *export.DataMeta` to `core.TableDef` (zero value nil = no data section). The table serializer checks it and emits the `data:` block. This is consistent with the existing pattern of embedding all metadata in `TableDef`.

2. **Schema scope: user-specified vs. all user schemas**
   - What we know: Current `backupCmd` has no `--schema` flag. Spec section 3 says "スキーマ単位フォルダ".
   - What's unclear: Should Phase 7 back up only `public` (simplest to test) or all user schemas?
   - Recommendation: Discover all user schemas from `pg_namespace` dynamically as the default. Accept the question as a planning decision.

3. **Output directory flag**
   - What we know: Spec names `backup_YYYYMMDD_HHMMSS/` but doesn't specify where it lives.
   - What's unclear: Should output directory be current working directory or a `--output` flag?
   - Recommendation: Default to CWD; add optional `--output` flag if the planner decides to include it. Phase 7 can default to `.` and leave the flag for Phase 9/v2.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none (go test flags) |
| Quick run command | `go test -short ./internal/backup/... ./cmd/pgbackup/...` |
| Full suite command | `TEST_DATABASE_URL=... go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BKUP-01 | `RunBackup` creates spec-compliant directory tree | integration | `go test ./internal/backup/... -run TestRunBackup` | Wave 0 |
| BKUP-01 | Non-table objects written as `<kind>/<name>.yaml` | unit | `go test ./internal/backup/... -run TestObjectFilePath -short` | Wave 0 |
| BKUP-01 | `_manifest.yaml` written at backup root | integration | `go test ./internal/backup/... -run TestManifestWritten` | Wave 0 |
| BKUP-02 | Tables with `bytea` column are skipped | unit | `go test ./internal/backup/... -run TestSkipDetection -short` | Wave 0 |
| BKUP-02 | Skipped tables appear in manifest `skipped_tables` | unit | `go test ./internal/backup/... -run TestSkippedInManifest -short` | Wave 0 |
| BKUP-02 | Warning logged for each skipped table | unit | `go test ./internal/backup/... -run TestSkipWarning -short` | Wave 0 |
| BKUP-03 | `--snapshot` flag starts REPEATABLE READ tx | integration | `go test ./internal/backup/... -run TestSnapshotMode` | Wave 0 |
| BKUP-04 | Default (no snapshot) runs without transaction | unit | `go test ./internal/backup/... -run TestNoSnapshotMode -short` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -short ./internal/backup/... ./cmd/pgbackup/...`
- **Per wave merge:** `TEST_DATABASE_URL=... go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/backup/orchestrator.go` — package stub so tests compile
- [ ] `internal/backup/orchestrator_test.go` — RED stubs for all req-mapped tests
- [ ] `internal/backup/skip.go` (optional) — `skipReason()` helper, unit-testable in isolation

## Sources

### Primary (HIGH confidence)
- Source code inspection: `internal/export/csv.go` — `ExportTableData` signature and behavior
- Source code inspection: `internal/resolve/manifest.go` — `BuildManifest` + `WriteManifest` API
- Source code inspection: `internal/resolve/kahn.go` — `BuildRestoreOrder` API
- Source code inspection: `internal/serialize/table/serializer.go` — confirms `Data` field exists but is never populated
- Source code inspection: `internal/fetch/*/fetcher.go` — all 10 object kinds confirmed implemented
- Source code inspection: `cmd/pgbackup/cmd/root.go` — `conn *pgx.Conn` package-level var pattern
- Spec: `docs/pre/pgbackup_spec.md` sections 3, 4, 5 — authoritative directory layout and skip rules

### Secondary (MEDIUM confidence)
- pgx/v5 `BeginTx` / `TxOptions` API — standard pgx transaction pattern, consistent with pgx v5 documentation conventions

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries in go.mod, all component APIs verified by source inspection
- Architecture: HIGH — spec section 4 defines directory layout exactly; all components have known interfaces
- Pitfalls: HIGH — DataMeta gap, partition routing, snapshot/conn issue all identified from direct source inspection
- Open questions: MEDIUM — DataMeta injection strategy requires a planning decision

**Research date:** 2026-03-11
**Valid until:** 2026-04-10 (stable internal codebase; only invalidated by Phase 5/6 changes)
