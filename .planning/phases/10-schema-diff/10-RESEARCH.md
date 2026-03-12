# Phase 10: Schema Diff - Research

**Researched:** 2026-03-12
**Domain:** Go schema comparison, backup directory traversal, structured text output
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DIFF-01 | バックアップ同士のスキーマ比較（データ差分は対象外） | Load both backup dirs via `resolve.ReadManifest` + per-kind serializers; data fields ignored by comparing only schema-bearing ObjectDef fields |
| DIFF-02 | オブジェクトの追加・削除検出 | Build two sets from RestoreOrder (or manifest Objects); set difference gives additions and deletions; object key = `schema.kind.name` |
| DIFF-03 | テーブル変更検出（カラム追加・削除・型変更・NULL制約・デフォルト値、インデックス、制約、トリガー、RLS） | Field-by-field comparison of `*core.TableDef`; columns compared by name with change subtypes; indexes/constraints/RLS by name set-difference |
| DIFF-04 | ビュー・マテビュー・関数・シーケンス・型・ドメイン・ENUMの変更検出（変更あり/なし） | Compare serialized YAML or individual fields; any field difference = "changed", otherwise "unchanged" |
| DIFF-05 | 差分結果の整形出力（仕様書9.3準拠） | `fmt.Fprintf` to stdout; sections: 追加オブジェクト, 削除オブジェクト, 変更オブジェクト; table changes list each sub-change |
</phase_requirements>

---

## Summary

Phase 10 implements `pgbackup diff <backup-a> <backup-b>`. It reads two backup directories from disk, deserializes all schema objects using the existing serializer infrastructure, compares them, and prints a structured human-readable report to stdout.

The phase has no new external dependencies. All required types (`core.*Def`), serializers (`serialize/*/serializer.go`), and file path helpers are already built in Phases 2–7. The diff engine is pure in-memory comparison with no database connection required. The `diff` subcommand in `cmd/pgbackup/cmd/diff.go` is already registered but has no implementation.

**Primary recommendation:** Create `internal/diff/` package with a `Run(backupA, backupB string, w io.Writer) error` function. Load all objects from each backup using the manifest's RestoreOrder to discover which YAML files exist, deserialize via existing serializers, then compare. The `diffCmd` in `cmd/` wires the two positional args to this function and writes to `os.Stdout`.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `go.yaml.in/yaml/v3` | v3.0.4 | Read def.yaml files | Already in go.mod; same YAML library used by all serializers |
| `github.com/pgbkrs/pgbackup/internal/core` | internal | All schema object types | Authoritative type definitions for this project |
| `github.com/pgbkrs/pgbackup/internal/resolve` | internal | ReadManifest, RestoreEntry | Already provides manifest loading |
| `github.com/pgbkrs/pgbackup/internal/serialize/*` | internal | Deserialize each kind | All 11 serializers already exist and are tested |
| stdlib `fmt`, `io`, `os`, `sort` | Go 1.24 | Output, sorting, file access | No new dependencies needed |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/spf13/cobra` | v1.10.2 | CLI flags (`--backup-a`, `--backup-b` or positional args) | Already used for all commands |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Reusing existing serializers | Re-reading YAML directly with a generic map | Reusing serializers gives typed structs for field-level comparison; generic map can't distinguish column type changes from other changes |
| Loading via manifest RestoreOrder | Walking filesystem directly | Manifest is the authoritative inventory; filesystem walk would need to reconstruct kind from path, reimplements manifest logic |

**Installation:** No new packages. All dependencies already in `go.mod`.

---

## Architecture Patterns

### Recommended Project Structure
```
internal/diff/
├── diff.go          # Run() entry point; loads A and B, calls compare, writes output
├── loader.go        # loadBackup(dir) → map[string]core.ObjectDef keyed by "schema.kind.name"
├── compare.go       # compareObjects() → DiffResult; per-kind comparison functions
├── report.go        # formatReport(DiffResult, w io.Writer); spec 9.3 output
└── diff_test.go     # unit tests for compare functions and report formatting

cmd/pgbackup/cmd/diff.go  # wire positional args → diff.Run()
```

### Pattern 1: Manifest-Driven Object Loading

**What:** Use the backup's `_manifest.yaml` `restore_order` to discover which objects exist in a backup directory, then deserialize each via the appropriate serializer. This exactly mirrors how `internal/restore/loader.go` works.

**When to use:** Loading all schema objects from a backup directory.

**Example:**
```go
// Source: internal/restore/loader.go (existing code)
// Reuse defYAMLPath and serializerFor patterns — do NOT duplicate them.
// Option A: expose loadObjectDef from restore package (export_test.go pattern)
// Option B: duplicate the small helpers in internal/diff/loader.go

func loadBackup(backupDir string) (map[string]core.ObjectDef, error) {
    manifest, err := resolve.ReadManifest(filepath.Join(backupDir, "_manifest.yaml"))
    if err != nil {
        return nil, err
    }
    objects := make(map[string]core.ObjectDef)
    for _, entry := range manifest.RestoreOrder {
        if entry.Kind == "fk" {
            // FKs have a YAML file in foreignkeys/; use serializeForeignkey.Serializer
            // Key: "schema.fk.name"
        }
        // Load def.yaml via serializer, key = entry.Schema + "." + entry.Kind + "." + entry.Name
    }
    return objects, nil
}
```

The key insight: `restore/loader.go`'s `defYAMLPath` and `serializerFor` functions already solve this. The diff loader is a parallel implementation or can call them if they are exported. Given the project convention of keeping packages internal-only and the existing `export_test.go` pattern, the cleanest approach is to duplicate the small helpers in `internal/diff/loader.go` (they are ~40 lines) rather than create a cross-package dependency from `diff` on `restore`.

### Pattern 2: Object Keying Strategy

**What:** Each object gets a unique key for set-difference comparisons.

**Key format for non-FK objects:** `schema.kind.name`
- Examples: `public.table.users`, `public.view.active_users`, `public.function.calc_total`

**Key format for FKs:** `schema.fk.constraint_name`
- FKs appear in RestoreOrder with `kind="fk"`; they have YAML in `<schema>/foreignkeys/<name>.yaml`

**Why not `schema.name`?** Two objects of different kinds can share a schema+name (e.g., a type and a function named `status`). The manifest's `Objects[].id` uses `schema.name` but RestoreOrder has `kind`; use RestoreOrder for loading.

### Pattern 3: Comparison Result Types

**What:** Structured result types separate the comparison logic from formatting.

```go
// Source: spec section 9.2 + 9.3

type DiffResult struct {
    Added    []ObjectHeader      // objects in B not in A
    Removed  []ObjectHeader      // objects in A not in B
    Changed  []ObjectChange      // objects in both with differences
}

type ObjectChange struct {
    Header  core.ObjectHeader
    Details []string            // human-readable change lines per spec 9.3
    // For non-table kinds: Details = ["本体変更あり"] or ["定義変更あり"]
    // For tables: Details = ["カラム追加  : phone_number varchar(20) nullable", ...]
}
```

### Pattern 4: Table Comparison (DIFF-03 is the complex case)

**What:** Tables require field-level diff across columns, indexes, constraints (PK, UK, check), triggers (by TriggerRef name), and RLS (enabled flag + policies).

**Sub-change categories (spec 9.3):**
- `カラム追加  : <name> <type> <nullable>`
- `カラム削除  : <name>`
- `カラム変更  : <name> type <old> → <new>` (and/or nullable, default)
- `インデックス追加: <name>` / `インデックス削除: <name>`
- `制約追加   : <name>` / `制約削除   : <name>`
- `FK追加     : <name> → <target>` / `FK削除     : <name>`
- `トリガー追加: <name>` / `トリガー削除: <name>`
- `RLS変更    : enabled <old> → <new>`

**Column comparison algorithm:**
```go
// Build map[colName]ColumnDef for each table
// 1. Columns in B not in A → added
// 2. Columns in A not in B → removed
// 3. Columns in both: compare Type, Nullable, Default → changed
```

**Note on TriggerRef:** `TableDef` in `core/types.go` does NOT have a `Triggers []TriggerRef` field — this is absent from the current struct. The spec section 5.2 shows `triggers: [{name: ..., ref: ...}]` but Phase 2 serializer's `yamlTableDef` does NOT include a triggers field. Check whether TriggerDef objects in the backup handle this separately via the triggers/ directory. The diff should detect trigger additions/deletions at the manifest level (by comparing which TriggerDef objects reference which tables) rather than through TableDef fields.

### Pattern 5: Non-Table Changed Detection (DIFF-04)

**What:** For views, materialized views, functions, sequences, types, domains, ENUMs — report only "changed/unchanged" (no field-level diff).

**Implementation:** Compare the serialized YAML bytes of the two objects. If identical bytes, unchanged. If different, changed. This avoids writing per-kind field comparison logic and is robust to new fields.

Alternative: compare key fields directly (e.g., `ViewDef.Definition` equality). Either approach works; byte comparison is simpler and future-proof.

### Pattern 6: CLI Integration

**What:** `diffCmd` needs two positional arguments (backup-a, backup-b) and no DB connection. The current `rootCmd.PersistentPreRunE` connects to the DB for ALL subcommands; `diff` must opt out.

**Critical issue:** The current `PersistentPreRunE` on `rootCmd` runs DB connection for every subcommand including `diff`. `diff` does not need a DB connection. This must be addressed.

**Solution patterns:**
1. Give `diffCmd` its own `PersistentPreRunE: cobra.NoopCommand` (or empty func returning nil) to override the parent — **but cobra does not allow child `PreRunE` to override parent `PersistentPreRunE`**. Both run.
2. Add a skip flag: check `cmd.Name() == "diff"` inside `rootCmd.PersistentPreRunE` and skip DB connection.
3. Move DB flags to `backupCmd` and `restoreCmd` directly (larger refactor, not needed).
4. **Recommended**: In `rootCmd.PersistentPreRunE`, check if the invoked command is `diff` (via `cmd.HasParent()` and name check) and skip connection. Or use a boolean flag `--no-db` (ugly). The simplest: check `cmd.Name() == "diff"` in `PersistentPreRunE` and return nil early.

**Example:**
```go
// In root.go PersistentPreRunE:
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
    if cmd.Name() == "diff" {
        return nil // diff does not need a DB connection
    }
    // ... existing DB connection logic
},
```

### Pattern 7: Output Format (DIFF-05, spec 9.3)

**What:** Print to stdout in the exact format from spec section 9.3.

```
比較: <backup-a-basename> → <backup-b-basename>

[追加オブジェクト]
  table    : public.new_feature
  function : public.new_func(integer)

[削除オブジェクト]
  index    : public.idx_orders_old
  table    : public.obsolete_log

[変更オブジェクト]
  table: public.users
    カラム追加  : phone_number varchar(20) nullable
    カラム変更  : status type varchar(20) → varchar(50)

  function: public.calc_total(integer)
    本体変更あり

  view: public.active_users
    定義変更あり
```

**Notes from spec 9.3:**
- Sections are printed only if non-empty
- Function names include arg types in parens for display: `calc_total(integer)` — this comes from `FunctionDef.ArgTypes`
- Object kind is left-padded/aligned in addition/deletion sections
- Table changes list one line per sub-change, indented with 4 spaces

### Anti-Patterns to Avoid
- **Connecting to the DB in `diff`:** The diff compares two on-disk backups; no live DB is needed. Do not require `--host`/`--user`/etc.
- **Walking the filesystem directly:** Use the manifest as inventory. Direct filesystem walk would need to re-derive kind from path segments and handle partition children specially.
- **Duplicating serializer YAML structs:** Use the existing `Deserialize()` methods. Do not re-implement YAML parsing.
- **Sorting output by manifest order vs. alphabetical:** The spec output example shows additions/deletions as a flat list; sort alphabetically by `schema.name` for deterministic output.
- **Treating partitioned table children as top-level:** Partition children appear in RestoreOrder with `from_table != ""`. The diff should treat parent+children as a unit or handle partition children as separate table entries (whichever produces a clean diff). Conservative approach: include partition children as their own table entries in the diff.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Manifest loading | Custom YAML parser | `resolve.ReadManifest` | Already implemented, tested, handles all edge cases |
| YAML deserialization | Custom parsing per kind | `serialize/*/Serializer.Deserialize()` | 11 serializers fully tested; handles all type assertions |
| File path resolution | Custom path builder | Pattern from `restore/loader.go`'s `defYAMLPath` | Handles tables, partition children, non-table paths |
| CLI arg parsing | Manual `os.Args` | `cobra` positional args with `Args: cobra.ExactArgs(2)` | Already the project's CLI library |

**Key insight:** This phase is primarily assembly of existing pieces. The serializers are the hard part and they are done. The diff logic itself is straightforward set/field comparison — the complexity is in wiring correctly, not in novel algorithms.

---

## Common Pitfalls

### Pitfall 1: DB Connection Required for `diff`
**What goes wrong:** The current `rootCmd.PersistentPreRunE` connects to PostgreSQL for every subcommand. If `diff` inherits this, it requires `--host`/`--user`/etc. even though it only reads files.
**Why it happens:** `PersistentPreRunE` runs for all child commands by design in cobra.
**How to avoid:** Add an early-return check in `PersistentPreRunE` for `cmd.Name() == "diff"`.
**Warning signs:** Running `pgbackup diff backup_a backup_b` with no DB flags returns a connection error.

### Pitfall 2: ForeignKey Objects in Diff
**What goes wrong:** FKs use `kind="fk"` in the manifest RestoreOrder (not `kind="foreign_key"`), and have YAML in `<schema>/foreignkeys/<name>.yaml`. The manifest objects list uses `kind="fk"`.
**Why it happens:** The manifest uses "fk" as a shorthand (established in Phase 6 decisions); the core type uses "foreign_key".
**How to avoid:** Use the same `kind == "fk"` special-case pattern as in `restore/loader.go`'s `serializerFor`. Key FK objects as `schema.fk.name` in the diff maps.

### Pitfall 3: Partition Children Double-Counting
**What goes wrong:** A partitioned table parent and its children all appear in RestoreOrder. Counting both parent and child as independent objects is correct (they are separate tables with separate def.yaml files), but displaying a partition child deletion without context may be confusing.
**Why it happens:** Partitioning is stored in `RestoreEntry.FromTable` when non-empty.
**How to avoid:** Include both parent and child entries in the diff. Partition children identified by `entry.FromTable != ""` can be displayed as `<schema>.tables.<parent>/partitions/<child>` if desired, or just as `<schema>.<child>`.

### Pitfall 4: Trigger Comparison via TableDef Fields
**What goes wrong:** `core.TableDef` does NOT have a `Triggers []TriggerRef` field in the current implementation (the spec section 5.2 YAML shows `triggers:` but the Go struct does not include it — the serializer's `yamlTableDef` has no triggers field). Looking at `table/serializer.go` confirms this.
**Why it happens:** Triggers are stored as independent `TriggerDef` objects in `<schema>/triggers/`. The table YAML does not carry trigger references.
**How to avoid:** Detect trigger additions/deletions by comparing TriggerDef objects (from the triggers/ directory) by `TableName` field. A trigger is "added to table X" if a TriggerDef with `TableName=X` exists in B but not A.

### Pitfall 5: RLS Policy Comparison
**What goes wrong:** `core.TableDef.RLS` only has `Enabled bool`. Policies are separate `PolicyDef` objects. The diff for "RLS changes" to a table should check both `RLSDef.Enabled` changes on the table and additions/deletions of `PolicyDef` objects for that table.
**Why it happens:** RLS policies are independent objects in `<schema>/policies/`.
**How to avoid:** Report `RLS有効化`/`RLS無効化` from `TableDef.RLS.Enabled` changes. Report policy additions/deletions separately in the `[追加オブジェクト]`/`[削除オブジェクト]` sections or as a table-level change line.

### Pitfall 6: Object Key Collisions
**What goes wrong:** Using `schema.name` as the map key (as manifest Objects.id does) can cause collisions between e.g. a `type` and a `function` with the same name.
**Why it happens:** The manifest `objects[].id` uses `schema.name` without kind, but RestoreOrder has `kind`.
**How to avoid:** Use `schema.kind.name` as the key in diff maps. For FK entries, `kind="fk"`.

---

## Code Examples

### Loading a Backup into a Typed Map

```go
// Source: patterns from internal/restore/loader.go and internal/resolve/manifest.go

func loadBackup(backupDir string) (map[string]core.ObjectDef, error) {
    manifestPath := filepath.Join(backupDir, "_manifest.yaml")
    m, err := resolve.ReadManifest(manifestPath)
    if err != nil {
        return nil, fmt.Errorf("read manifest %s: %w", backupDir, err)
    }

    result := make(map[string]core.ObjectDef, len(m.RestoreOrder))
    for _, entry := range m.RestoreOrder {
        yamlPath := defYAMLPath(backupDir, entry)
        if yamlPath == "" {
            continue // FK with no def.yaml path (should not happen for fk kind)
        }
        data, err := os.ReadFile(yamlPath)
        if err != nil {
            if os.IsNotExist(err) {
                continue // skip missing files gracefully
            }
            return nil, fmt.Errorf("read %s: %w", yamlPath, err)
        }
        s := serializerFor(entry.Kind)
        if s == nil {
            continue
        }
        def, err := s.Deserialize(data)
        if err != nil {
            return nil, fmt.Errorf("deserialize %s.%s: %w", entry.Schema, entry.Name, err)
        }
        key := entry.Schema + "." + entry.Kind + "." + entry.Name
        result[key] = def
    }
    return result, nil
}
```

### Table Column Diff

```go
// Compare columns between two TableDef instances
func diffColumns(a, b *core.TableDef) []string {
    var changes []string
    aByName := make(map[string]core.ColumnDef)
    for _, col := range a.Columns {
        aByName[col.Name] = col
    }
    bByName := make(map[string]core.ColumnDef)
    for _, col := range b.Columns {
        bByName[col.Name] = col
    }
    // Additions: in B not in A
    for _, col := range b.Columns {
        if _, ok := aByName[col.Name]; !ok {
            nullable := "not null"
            if col.Nullable { nullable = "nullable" }
            changes = append(changes, fmt.Sprintf("カラム追加  : %s %s %s", col.Name, col.Type, nullable))
        }
    }
    // Deletions: in A not in B
    for _, col := range a.Columns {
        if _, ok := bByName[col.Name]; !ok {
            changes = append(changes, fmt.Sprintf("カラム削除  : %s", col.Name))
        }
    }
    // Modifications: in both, fields differ
    for _, col := range b.Columns {
        aCol, ok := aByName[col.Name]
        if !ok { continue }
        if aCol.Type != col.Type {
            changes = append(changes, fmt.Sprintf("カラム変更  : %s type %s → %s", col.Name, aCol.Type, col.Type))
        }
        if aCol.Nullable != col.Nullable {
            changes = append(changes, fmt.Sprintf("カラム変更  : %s nullable %v → %v", col.Name, aCol.Nullable, col.Nullable))
        }
        if aCol.Default != col.Default {
            changes = append(changes, fmt.Sprintf("カラム変更  : %s default %q → %q", col.Name, aCol.Default, col.Default))
        }
    }
    return changes
}
```

### Non-Table Change Detection via Serialization Comparison

```go
// Source: project convention; no external library needed
func isChanged(a, b core.ObjectDef, s core.Serializer) (bool, error) {
    aBytes, err := s.Serialize(a)
    if err != nil { return false, err }
    bBytes, err := s.Serialize(b)
    if err != nil { return false, err }
    return !bytes.Equal(aBytes, bBytes), nil
}
```

### CLI Wiring

```go
// cmd/pgbackup/cmd/diff.go
var diffCmd = &cobra.Command{
    Use:   "diff <backup-a> <backup-b>",
    Short: "Compare two backup directories at the schema level",
    Args:  cobra.ExactArgs(2),
    RunE: func(cmd *cobra.Command, args []string) error {
        return diff.Run(args[0], args[1], os.Stdout)
    },
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| diffCmd stub (no-op) | Full implementation | Phase 10 | Enables schema comparison |
| No `internal/diff` package | New package `internal/diff` | Phase 10 | Clean separation of diff logic |

**Deprecated/outdated:**
- The `diff.go` stub that returns nil: replaced by real implementation in Phase 10.

---

## Open Questions

1. **Trigger comparison granularity**
   - What we know: `TableDef` has no `Triggers` field; triggers are independent `TriggerDef` objects with a `TableName` field.
   - What's unclear: Spec 9.3 shows "トリガー追加: audit_trigger" under a table's change block. Should trigger changes appear as table-level details OR in the top-level added/removed sections?
   - Recommendation: Report trigger additions/deletions as table-level changes when the trigger's `TableName` matches a changed/existing table. Otherwise include in top-level added/removed.

2. **RLS policy reporting**
   - What we know: `TableDef.RLS.Enabled` tracks enabled/disabled; `PolicyDef` is a separate object.
   - What's unclear: Should policy additions/deletions be shown as table-level details (matching spec 9.2 "RLSポリシー") or as top-level object additions?
   - Recommendation: Show `RLS有効化/無効化` as a table-level detail; show individual policy adds/removes as top-level additions/deletions. This matches the spec's separation of RLS setting from individual policies.

3. **Output when no differences exist**
   - What we know: Spec 9.3 shows only non-empty sections.
   - What's unclear: What to print when backups are identical?
   - Recommendation: Print the header line (`比較: A → B`) and one line: `(差分なし)` or similar. No spec requirement; this is Claude's discretion.

4. **Index comparison within tables**
   - What we know: `IndexDef` has `Name`, `Method`, `Definition` fields in `TableDef.Indexes`.
   - What's unclear: Should index changes show as "index changed" with old/new definition, or just added/removed?
   - Recommendation: Use name-based set difference (added/removed). If same name but different definition, report as removed+added. This matches the spec's output format showing `インデックス追加`/`インデックス削除` without "changed" variant.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package (go 1.24) |
| Config file | none — uses `go test` directly |
| Quick run command | `go test -short ./internal/diff/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DIFF-01 | loadBackup reads manifest + deserializes objects | unit | `go test -short ./internal/diff/... -run TestLoadBackup` | Wave 0 |
| DIFF-02 | Added objects: in B not in A; Removed: in A not in B | unit | `go test -short ./internal/diff/... -run TestDiffAddedRemoved` | Wave 0 |
| DIFF-03 | Table column add/remove/type-change/nullable/default; indexes; constraints; RLS | unit | `go test -short ./internal/diff/... -run TestDiffTable` | Wave 0 |
| DIFF-04 | Non-table changed/unchanged detection | unit | `go test -short ./internal/diff/... -run TestDiffNonTable` | Wave 0 |
| DIFF-05 | Output format matches spec 9.3 (sections, indentation, Japanese labels) | unit | `go test -short ./internal/diff/... -run TestFormatReport` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -short ./internal/diff/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/diff/diff_test.go` — covers DIFF-01 through DIFF-05
- [ ] `internal/diff/diff.go` — package stub (empty `Run` function returning nil)
- [ ] `internal/diff/loader.go` — stub `loadBackup` returning empty map
- [ ] `internal/diff/compare.go` — stub comparison functions
- [ ] `internal/diff/report.go` — stub formatter

---

## Sources

### Primary (HIGH confidence)
- `internal/core/types.go` — authoritative ObjectDef types; all fields directly inspectable
- `internal/restore/loader.go` — definitive file-loading pattern with `defYAMLPath` and `serializerFor`
- `internal/resolve/manifest.go` — `ReadManifest`, `RestoreEntry` struct
- `internal/serialize/table/serializer.go` — YAML field mapping for TableDef (confirms no Triggers field)
- `docs/pre/pgbackup_spec.md` sections 9.1–9.3 — canonical output format and detection scope

### Secondary (MEDIUM confidence)
- `cmd/pgbackup/cmd/root.go` — `PersistentPreRunE` DB connection pattern; confirmed diff needs skip
- `cmd/pgbackup/cmd/diff.go` — current stub; confirms location and cobra registration

### Tertiary (LOW confidence)
- None — all findings are directly verified from codebase.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all dependencies verified in go.mod and existing code
- Architecture: HIGH — all file formats and type structures directly inspected
- Pitfalls: HIGH — derived from actual code inspection (e.g., no Triggers in TableDef, "fk" vs "foreign_key" inconsistency, PersistentPreRunE DB issue)

**Research date:** 2026-03-12
**Valid until:** Stable — all dependencies are internal; no external library churn risk
