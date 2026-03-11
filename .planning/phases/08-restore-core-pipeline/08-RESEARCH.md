# Phase 8: Restore - Core Pipeline - Research

**Researched:** 2026-03-12
**Domain:** Go restore orchestrator — manifest parsing, DDL execution, COPY FROM, SETVAL, pgx/v5
**Confidence:** HIGH

## Summary

Phase 8 builds the full restore pipeline for `pgbackup restore`. The codebase already has every building block: all DDL generators (`GenerateDDL` / `GenerateDrop`), all YAML deserializers, `COPY TO` (model for `COPY FROM`), the manifest structure (`resolve.Manifest` / `resolve.RestoreEntry`), and `RunBackup` (model for `RunRestore`). The restore orchestrator follows the same shape as `RunBackup` but in reverse/forward order and reads from disk rather than writing to it.

The critical design insight is that the restore pipeline is **multi-wave**: DROP (reverse manifest order) → CREATE schema objects (manifest order, tables first, no indexes/FKs yet) → COPY FROM data → SETVAL sequences → CREATE indexes → ADD FK constraints → CREATE views/functions/triggers/policies. The `restore_order` in `_manifest.yaml` already encodes this sequence; the orchestrator just needs to implement the filtering per wave.

The pre-restore backup (REST-01) requires calling `RunBackup` before any mutations, using the same `conn` and a caller-supplied output directory. Index DDL is not in the serialized YAML (only `name` and `method` are stored) but is stored verbatim in `IndexDef.Definition` — however, because the serializer drops `Definition`, the restore path cannot reconstruct index DDL from the YAML alone. The solution is to generate index DDL from the current database using `pg_get_indexdef` after tables are created, or to extend the YAML to carry `IndexDef.Definition`. This is the **single significant design gap** to resolve.

**Primary recommendation:** Implement `internal/restore/orchestrator.go` mirroring `internal/backup/orchestrator.go`. Use the manifest `restore_order` list as the authoritative driver; filter each wave by `Kind`. For indexes, read `IndexDef.Definition` from `def.yaml` — but first confirm whether the serializer round-trip preserves it or not (it currently does NOT preserve `Definition`). Either extend `yamlIndex` to carry `definition`, or re-fetch index DDL from `pg_get_indexdef` after table CREATE.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| REST-01 | Pre-restore automatic backup | Call `RunBackup` before mutations; same `conn`, configurable output dir |
| REST-02 | DROP in reverse restore_order + drop.log | Reverse `manifest.RestoreOrder`, call `GenerateDrop` per kind, exec via `conn.Exec` |
| REST-04 | CREATE in restore_order | Iterate `manifest.RestoreOrder`, call `GenerateDDL` per kind, exec via `conn.Exec` — tables only (no indexes, no FKs) in this pass |
| REST-05 | COPY FROM for tables + partition children | `conn.PgConn().CopyFrom` — mirrors `CopyTo` already used in `export/csv.go` |
| REST-06 | SETVAL for sequences | `SELECT setval('schema.name', last_value, is_called)` via `conn.Exec` |
| REST-07 | Index creation after data load | Execute `IndexDef.Definition` strings — requires `definition` field in YAML (gap: see below) |
| REST-08 | FK constraints as a single batch after all tables + data | All `kind=fk` entries from `RestoreOrder`; call `foreignkey.DDLGenerator.GenerateDDL` |
| REST-09 | Views, functions, triggers, policies in manifest order | Already tracked in `RestoreOrder`; just call appropriate `GenerateDDL` per kind |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/jackc/pgx/v5` | v5.8.0 | PostgreSQL driver — DDL exec, COPY FROM | Already in `go.mod`; used throughout |
| `go.yaml.in/yaml/v3` | v3.0.4 | Read `_manifest.yaml` + per-object `def.yaml` | Already in `go.mod`; used in serializers |
| `github.com/spf13/cobra` | v1.10.2 | CLI flags for `restore` subcommand | Already in `go.mod`; `restoreCmd` stub exists |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/pgbkrs/pgbackup/internal/backup` | (local) | `RunBackup` for pre-restore backup | REST-01 |
| `github.com/pgbkrs/pgbackup/internal/resolve` | (local) | `Manifest`, `RestoreEntry` types + `ReadManifest` | Reading `_manifest.yaml` |
| `github.com/pgbkrs/pgbackup/internal/ddl/*` | (local) | `GenerateDDL` / `GenerateDrop` for all object kinds | REST-02, REST-04, REST-07, REST-08, REST-09 |
| `github.com/pgbkrs/pgbackup/internal/serialize/*` | (local) | `Deserialize` for loading `def.yaml` | All phases that need object data |

**Installation:** No new dependencies required.

---

## Architecture Patterns

### Recommended Project Structure
```
internal/restore/
├── orchestrator.go        # RunRestore: main pipeline entry point
├── orchestrator_test.go   # Integration tests (TestRestoreIntegration)
├── loader.go              # loadManifest, loadObjectDef per kind
└── loader_test.go         # Unit tests for loader functions
```

The `cmd/pgbackup/cmd/restore.go` file needs flags wired and `restore.RunRestore(...)` called — mirroring `backup.go`.

### Pattern 1: Manifest-Driven Multi-Wave Pipeline

**What:** The manifest's `restore_order` list is the single authoritative sequence. Each wave filters by `Kind`:
- Wave 1 (DROP): All entries reversed — call `GenerateDrop`
- Wave 2 (CREATE): Tables, sequences, types, domains, enums, composite types (no indexes — see gap)
- Wave 3 (COPY FROM): Table entries in manifest order
- Wave 4 (SETVAL): Sequence entries
- Wave 5 (INDEXES): Table entries — generate `CREATE INDEX` from stored `definition` field
- Wave 6 (FK BATCH): All `kind=fk` entries — call `GenerateDDL` on `ForeignKeyDef`
- Wave 7 (DDL remainder): Views, materialized views, functions, triggers, policies

**When to use:** Always — the manifest already encodes dependencies.

**Example:**
```go
// Source: internal/backup/orchestrator.go (RunBackup pattern, mirrored for restore)
func RunRestore(ctx context.Context, conn *pgx.Conn, backupDir string, preBackupDir string) error {
    // Step 1: Pre-restore backup (REST-01)
    if err := backup.RunBackup(ctx, conn, preBackupDir, false); err != nil {
        return fmt.Errorf("pre-restore backup: %w", err)
    }

    // Step 2: Load manifest
    manifest, err := resolve.ReadManifest(filepath.Join(backupDir, "_manifest.yaml"))
    ...

    // Step 3: DROP in reverse order (REST-02)
    for i := len(manifest.RestoreOrder) - 1; i >= 0; i-- {
        entry := manifest.RestoreOrder[i]
        def, err := loadObjectDef(backupDir, entry)
        stmts, err := ddlGeneratorFor(entry.Kind).GenerateDrop(def)
        conn.Exec(ctx, stmt)
    }

    // Steps 4-9: waves filtered by kind...
}
```

### Pattern 2: COPY FROM via pgx PgConn

**What:** Mirror of `export/csv.go`'s `CopyTo` — use `conn.PgConn().CopyFrom(ctx, reader, sql)`.

**When to use:** REST-05 data loading.

**Example:**
```go
// Source: pgx/v5 docs — CopyFrom mirrors CopyTo
f, err := os.Open(csvPath)
defer f.Close()
sql := fmt.Sprintf(`COPY "%s"."%s" (%s) FROM STDIN WITH (FORMAT CSV)`, schema, table, colList)
_, err = conn.PgConn().CopyFrom(ctx, f, sql)
```

The column list must be quoted and comma-separated. Derive from the `def.yaml` `data.columns` field (already stored in `TableDataMeta`/`yamlData`).

### Pattern 3: SETVAL for Sequences

**What:** Call PostgreSQL's `setval()` function after sequences are created but before data is loaded (sequences may be used by serial/identity columns, though data load via COPY does not advance sequences automatically).

**When to use:** REST-06.

**Example:**
```go
// Source: PostgreSQL docs — setval(regclass, bigint, boolean)
sql := fmt.Sprintf(`SELECT setval('%s.%s', %d, %v)`, sd.Schema, sd.Name, sd.LastValue, sd.IsCalled)
conn.Exec(ctx, sql)
```

Note: If `IsCalled = false` (sequence never used), `setval(seq, last_value, false)` sets the next call to return `last_value`. This matches the `SequenceDef.IsCalled` field.

### Pattern 4: Loading Object Definitions from Disk

**What:** For each `RestoreEntry`, determine the file path from the directory layout (mirrors `kindToDir` in backup orchestrator), read the `def.yaml`, then call the kind-appropriate `Serializer.Deserialize`.

**When to use:** All waves except pre-restore backup.

```go
// Derive path — mirrors backup orchestrator's kindToDir map
func defYAMLPath(backupDir string, entry resolve.RestoreEntry) string {
    switch entry.Kind {
    case "table":
        if entry.FromTable != "" {
            // partition child
            return filepath.Join(backupDir, entry.Schema, "tables", entry.FromTable, "partitions", entry.Name, "def.yaml")
        }
        return filepath.Join(backupDir, entry.Schema, "tables", entry.Name, "def.yaml")
    case "view":
        return filepath.Join(backupDir, entry.Schema, "views", entry.Name+".yaml")
    ...
    }
}
```

### Anti-Patterns to Avoid

- **Executing DDL inside a transaction for the whole restore:** PostgreSQL DDL statements like `CREATE TABLE` and `DROP TABLE` are transactional per statement. Wrapping everything in one transaction is fine for atomicity but risky for long restores (lock contention). For Phase 8, execute statements individually without a wrapping transaction — match Phase 7's non-snapshot approach.
- **Using `kindPriority` for DROP order instead of reversing RestoreOrder:** DROP must be the exact reverse of the manifest's `restore_order`. Do not re-sort — just reverse the slice.
- **Applying indexes via `table.DDLGenerator.GenerateDDL`:** That method returns `CREATE TABLE` + `CREATE INDEX` together. For restore, table creation and index creation are separate waves. Do NOT call `GenerateDDL` for the index wave — execute `IndexDef.Definition` strings directly. This requires the YAML gap fix (see Open Questions).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| COPY FROM streaming | Custom CSV reader | `conn.PgConn().CopyFrom(ctx, reader, sql)` | pgx wraps PgWireProtocol correctly; handles escaping, backpressure |
| DDL generation | String templates per object | Existing `internal/ddl/*` generators | Already implemented and tested for all 11 object kinds |
| YAML deserialization | Custom YAML parser | Existing `internal/serialize/*` `Deserialize` methods | Round-trip already tested; handles type assertions correctly |
| Manifest parsing | Custom YAML parser | `go.yaml.in/yaml/v3` unmarshal into `resolve.Manifest` struct | Struct already defined with correct tags |
| Pre-restore backup | New backup code | `backup.RunBackup(ctx, conn, dir, false)` | Already end-to-end tested |

**Key insight:** The entire restore pipeline is a composition of already-built components. The main new code is the orchestration logic (`RunRestore`) and the `COPY FROM` data loader.

---

## Common Pitfalls

### Pitfall 1: Index DDL Not Round-Tripped Through YAML

**What goes wrong:** `internal/serialize/table/serializer.go` stores only `name` and `method` in `yamlIndex` — the `Definition` field (full `pg_get_indexdef` output) is deliberately dropped. When deserializing a `def.yaml` for restore, `IndexDef.Definition` is empty, so `table.DDLGenerator.GenerateDDL` produces a `CREATE TABLE` statement but no usable `CREATE INDEX` statements.

**Why it happens:** The YAML spec was designed for human readability; full index DDL is verbose. The backup orchestrator never needed to read it back.

**How to avoid:** Two options:
1. Extend `yamlIndex` to include `definition string` (and `fromYAML` / `toYAML`). This is a backward-compatible YAML addition.
2. Accept that indexes must be re-fetched from PostgreSQL after table creation (re-query `pg_get_indexdef`). This is more complex but doesn't change the YAML format.

**Recommendation:** Option 1 (extend YAML with `definition` field). The YAML format change is additive and backward-compatible. New backups include the field; old backups without it simply omit index recreation (acceptable for Phase 8).

**Warning signs:** `CREATE INDEX` statements are empty/missing during restore; tables created without their non-PK indexes.

### Pitfall 2: Partition Children Not Found via kindToDir

**What goes wrong:** In the backup, partition child tables are written to `<schema>/tables/<parent>/partitions/<child>/def.yaml`, not `<schema>/tables/<child>/def.yaml`. The `RestoreEntry.FromTable` field records the parent. If the restore loader ignores `FromTable`, it will fail to find child table YAML files.

**Why it happens:** `RestoreEntry` has a `from_table` field specifically for this case. It's easy to miss when implementing the path resolver.

**How to avoid:** Always check `entry.FromTable != ""` when computing the path for `kind=table` entries.

**Warning signs:** `open .../tables/partitioned_table_2024/def.yaml: no such file or directory`

### Pitfall 3: FK Drop Order With CASCADE

**What goes wrong:** FKs are listed at the END of `restore_order`. When reversing for DROP, FKs are dropped first (correct). But if you try to DROP a table before its FKs are dropped, `DROP TABLE IF EXISTS ... CASCADE` handles it. However, if using `DROP CONSTRAINT` separately, order matters.

**Why it happens:** The manifest appends FKs after all other objects. Reversed, FKs come first in the drop pass.

**How to avoid:** Since `table.DDLGenerator.GenerateDrop` uses `CASCADE`, table DROPs will cascade to FKs automatically. The `foreignkey.DDLGenerator.GenerateDrop` (ALTER TABLE ... DROP CONSTRAINT) can still be run first without harm — it uses `IF EXISTS`.

### Pitfall 4: SETVAL Timing — Before or After COPY FROM

**What goes wrong:** If sequences are owned by serial columns, loading data via COPY FROM does NOT advance the sequence. After restore, the next `INSERT` would generate a conflicting value.

**Why it happens:** `COPY FROM` bypasses the default expression entirely when explicit values are provided (which COPY TO always does).

**How to avoid:** Run SETVAL **after** COPY FROM, not before. This ensures that even if data was loaded with sequence values, SETVAL overrides to the backed-up last value.

**Warning signs:** Duplicate key errors on `INSERT` after restore (sequence still at 1 while data has IDs up to N).

### Pitfall 5: `conn.Exec` for Multi-Statement DDL

**What goes wrong:** Some `GenerateDDL` methods return multiple statements (e.g., `table.DDLGenerator` returns `CREATE TABLE` plus `CREATE INDEX` statements in one slice). If the caller joins them with `;` and passes as one `Exec` call, pgx may reject multi-statement SQL.

**Why it happens:** pgx's `Exec` uses the simple query protocol for multi-statement SQL but its behavior with complex DDL is unreliable. Previous code always executes statements from the slice individually.

**How to avoid:** Always iterate the `[]string` slice and call `conn.Exec` once per statement. This is already the pattern to follow from Phase 4 tests.

### Pitfall 6: Type Assertion Consistency (Value vs Pointer)

**What goes wrong:** DDL generators use inconsistent receiver types. Some take value assertions (`core.ViewDef`) and others take pointer assertions (`*core.TableDef`, `*core.ForeignKeyDef`). If the restore loader deserializes an object and passes it directly to `GenerateDDL`, the assertion will fail for the wrong receiver type.

**Why it happens:** Established in Phase 4 decisions. The pattern is: `TableDef` and `ForeignKeyDef` use pointer receivers; all others use value receivers.

**How to avoid:** After `Deserialize`, match the type assertion pattern:
- `table.Serializer.Deserialize` → returns `*core.TableDef` ✓
- `foreignkey.Serializer.Deserialize` → returns `*core.ForeignKeyDef` ✓
- All others → return value types ✓

No conversion needed if using the existing serializers.

---

## Code Examples

Verified patterns from existing codebase:

### Reading _manifest.yaml
```go
// Source: internal/resolve/manifest.go — Manifest struct already defined
// Need to add ReadManifest function (currently only WriteManifest exists)
func ReadManifest(path string) (*Manifest, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read manifest: %w", err)
    }
    var m Manifest
    if err := yaml.Unmarshal(data, &m); err != nil {
        return nil, fmt.Errorf("parse manifest: %w", err)
    }
    return &m, nil
}
```

### Executing DDL Statements
```go
// Source: pgx/v5 — conn.Exec for DDL
for _, stmt := range stmts {
    if _, err := conn.Exec(ctx, stmt); err != nil {
        return fmt.Errorf("exec DDL %q: %w", stmt[:min(50, len(stmt))], err)
    }
}
```

### COPY FROM
```go
// Source: pgx/v5 PgConn.CopyFrom — mirrors export/csv.go CopyTo
f, err := os.Open(csvPath)
if err != nil {
    return fmt.Errorf("open data file: %w", err)
}
defer f.Close()
colList := quotedCols(td.DataMeta.Columns) // `"col1", "col2", ...`
sql := fmt.Sprintf(`COPY "%s"."%s" (%s) FROM STDIN WITH (FORMAT CSV)`, schema, table, colList)
if _, err := conn.PgConn().CopyFrom(ctx, f, sql); err != nil {
    return fmt.Errorf("COPY FROM %s.%s: %w", schema, table, err)
}
```

### SETVAL
```go
// Source: PostgreSQL docs — setval(regclass, bigint, boolean)
// sd is core.SequenceDef deserialized from YAML
sql := fmt.Sprintf(`SELECT setval('%s.%s', %d, %v)`, sd.Schema, sd.Name, sd.LastValue, sd.IsCalled)
if _, err := conn.Exec(ctx, sql); err != nil {
    return fmt.Errorf("setval %s.%s: %w", sd.Schema, sd.Name, err)
}
```

### Deserializing an Object from def.yaml
```go
// Source: internal/serialize/* — Deserialize methods
data, err := os.ReadFile(defYAMLPath)
s := serializerFor(ObjectKind(entry.Kind)) // same switch as backup.serializerFor
def, err := s.Deserialize(data)
```

### Kind-to-DDLGenerator Switch
```go
// Source: pattern mirroring backup.serializerFor
func ddlGeneratorFor(kind string) core.DDLGenerator {
    switch core.ObjectKind(kind) {
    case core.KindTable:            return &ddlTable.DDLGenerator{}
    case core.KindView:             return &ddlView.DDLGenerator{}
    case core.KindMaterializedView: return &ddlMatview.DDLGenerator{}
    case core.KindFunction:         return &ddlFunction.DDLGenerator{}
    case core.KindSequence:         return &ddlSequence.DDLGenerator{}
    case core.KindTrigger:          return &ddlTrigger.DDLGenerator{}
    case core.KindType:             return &ddlCompositetype.DDLGenerator{}
    case core.KindDomain:           return &ddlDomain.DDLGenerator{}
    case core.KindEnum:             return &ddlEnum.DDLGenerator{}
    case core.KindPolicy:           return &ddlPolicy.DDLGenerator{}
    case core.KindForeignKey:       return &ddlForeignkey.DDLGenerator{}
    }
    return nil
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `conn.Exec` with full SQL string | `conn.PgConn().CopyFrom` for bulk data | pgx/v5 from day 1 | Streaming, no full-table buffering |
| Separate FK handling per table | FK as independent `ObjectDef` + manifest ordering | Phase 4 decision | Circular FK cycle broken by design |
| Inline index DDL in CREATE TABLE | Separate `IndexDef.Definition` strings returned by `GenerateDDL` | Phase 4 decision | Enables index creation as separate wave |

**Deprecated/outdated:**
- Nothing relevant — this is a greenfield phase.

---

## Open Questions

1. **Index DDL Round-Trip Gap**
   - What we know: `yamlIndex` only stores `name` and `method`. `IndexDef.Definition` (the full `pg_get_indexdef` string) is not serialized.
   - What's unclear: Whether to (a) extend `yamlIndex` with `definition` field, or (b) re-fetch index DDL from PostgreSQL using the table definition after creation.
   - Recommendation: Extend `yamlIndex` in Phase 8 Wave 0 (add `definition` field, update `toYAML`/`fromYAML`). This is a non-breaking change to the YAML format. Any existing backups without the field will simply not create non-PK indexes during restore (PKs are in the CREATE TABLE constraints, not indexes).

2. **`ReadManifest` Function Missing**
   - What we know: `resolve.WriteManifest` exists. No `ReadManifest` exists yet.
   - What's unclear: Nothing — it needs to be added.
   - Recommendation: Add `ReadManifest(path string) (*Manifest, error)` to `internal/resolve/manifest.go`.

3. **Pre-restore backup output directory**
   - What we know: REST-01 requires "automatically taken" pre-restore backup. The CLI for `restore` has no flags yet.
   - What's unclear: Where to write the pre-restore backup — same `--output` dir? A `--pre-backup-dir` flag? Auto-detect sibling of `backupDir`?
   - Recommendation: Auto-place pre-restore backup in the same parent directory as the backup being restored (e.g., if restoring from `/data/backup_20260312_120000`, write pre-restore to `/data/`). Add `--pre-backup-dir` flag defaulting to `"."` mirroring `backup --output`.

4. **Policy DROP order (table must exist for `DROP POLICY`)**
   - What we know: `DROP POLICY IF EXISTS name ON schema.table` requires the table to exist.
   - What's unclear: If we drop in reverse manifest order, policies appear before tables (since policies come after tables in forward order). If the table was already dropped (via CASCADE), the policy DROP returns "no such policy" but `IF EXISTS` handles it.
   - Recommendation: Use `IF EXISTS` consistently (already in all `GenerateDrop` implementations). Rely on CASCADE from table DROP.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + `go test` |
| Config file | none (no pytest.ini/jest.config; uses `go test ./...`) |
| Quick run command | `go test ./internal/restore/... -short` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REST-01 | Pre-restore backup runs before mutations | unit (mock or integration) | `go test ./internal/restore/... -run TestPreRestoreBackup` | Wave 0 |
| REST-02 | DROP in reverse restore_order | integration | `go test ./internal/restore/... -run TestRestoreDropOrder` | Wave 0 |
| REST-04 | CREATE in restore_order (tables/types first) | integration | `go test ./internal/restore/... -run TestRestoreCreateOrder` | Wave 0 |
| REST-05 | COPY FROM loads table + partition child data | integration | `go test ./internal/restore/... -run TestRestoreCopyFrom` | Wave 0 |
| REST-06 | SETVAL restores sequence to backed-up value | integration | `go test ./internal/restore/... -run TestRestoreSetval` | Wave 0 |
| REST-07 | Indexes created after data load | integration | `go test ./internal/restore/... -run TestRestoreIndexes` | Wave 0 |
| REST-08 | FK constraints applied as single batch | integration | `go test ./internal/restore/... -run TestRestoreFKBatch` | Wave 0 |
| REST-09 | Views/functions/triggers/policies created in order | integration | `go test ./internal/restore/... -run TestRestoreViewsFunctions` | Wave 0 |

All integration tests follow the established pattern: skip if `TEST_DATABASE_URL` not set, skip in `-short` mode.

### Sampling Rate
- **Per task commit:** `go test ./internal/restore/... -short`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/restore/orchestrator_test.go` — integration test stubs for all 8 requirements above
- [ ] `internal/restore/loader_test.go` — unit test stubs for manifest reading, defYAMLPath, deserialize round-trip
- [ ] `internal/resolve/manifest.go` needs `ReadManifest` function
- [ ] `internal/serialize/table/serializer.go` needs `yamlIndex.definition` field (index DDL gap fix)

---

## Sources

### Primary (HIGH confidence)
- `internal/backup/orchestrator.go` — architectural model for RunRestore
- `internal/resolve/manifest.go` — Manifest struct, RestoreEntry, WriteManifest
- `internal/resolve/kahn.go` — BuildRestoreOrder, kindPriority
- `internal/core/types.go` — all ObjectDef types and fields
- `internal/core/interfaces.go` — DDLGenerator, Serializer, SchemaFetcher interfaces
- `internal/ddl/*/generator.go` — all 11 DDL generators (CREATE + DROP)
- `internal/serialize/*/serializer.go` — all 11 serializers (Serialize + Deserialize)
- `internal/export/csv.go` — CopyTo pattern → model for CopyFrom
- `go.mod` — confirmed dependency versions

### Secondary (MEDIUM confidence)
- pgx/v5 `PgConn.CopyFrom` API — mirrors `CopyTo` which is already in use; HIGH confidence it exists in v5.8.0
- PostgreSQL `setval(regclass, bigint, boolean)` — standard PostgreSQL function, documented behavior

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in use with known versions
- Architecture: HIGH — RunBackup is direct model; manifest structure fully known
- Pitfalls: HIGH — index DDL gap confirmed by reading serializer source; type assertion patterns confirmed by reading DDL generators
- Open questions: MEDIUM — index gap solution is straightforward; ReadManifest is trivially addable

**Research date:** 2026-03-12
**Valid until:** Stable — no external dependencies changing; internal codebase is the primary source
