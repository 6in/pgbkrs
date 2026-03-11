# Phase 9: Restore - Partial & Safety - Research

**Researched:** 2026-03-12
**Domain:** Go restore orchestrator — partial restore scoping, transitive dependency resolution, drop-leak detection, structured log files
**Confidence:** HIGH

## Summary

Phase 9 extends the fully working Phase 8 `RunRestore` pipeline with four new capabilities: (1) scope filtering (schema or single object), (2) transitive dependency resolution for partial scopes, (3) drop-leak detection after the DROP wave, and (4) structured log output to `drop.log`, `restore.log`, and `summary.log`. The orchestrator `RunRestore` already exists at `internal/restore/orchestrator.go`; all new code is additive to that function and supporting helpers.

The key architectural insight is that the manifest's `objects[].depends_on` field is already the complete serialized dependency graph — no live DB query is needed to compute transitive deps for a partial restore. The `DependsOn` field on `resolve.ObjectEntry` records `schema.name` IDs, enabling a BFS/DFS walk from any seed set to collect all required objects. Once the filtered object set is known, it is intersected with `RestoreOrder` to produce the scoped execution plan; the topological order is preserved automatically.

Drop-leak detection requires snapshotting all objects present in the DB before the DROP wave and comparing to what remains after it. The spec (section 8.3) defines the algorithm precisely: objects surviving the DROP phase that are NOT in the backup's restore_order are "leaks." The implementation queries `pg_class`, `pg_proc`, `pg_views`, etc. for a union of all backed-up kinds — the same set that `SchemaFetcher` interfaces already know how to enumerate.

Structured logging is a new concern in Phase 9. The spec defines three log files under a timestamped directory (`restore_YYYYMMDD_HHMMSS/`). No external logging library is needed — standard `fmt.Fprintf` to `os.File` is sufficient. The log directory should be written alongside the restore itself; the path can be derived from the backup input directory or a new `--log-dir` flag.

**Primary recommendation:** Augment `RunRestore` with an `Options` struct (filter scope + log dir), extract a `filteredRestoreOrder` helper using the manifest `objects` DAG for BFS, add a `snapshotLiveObjects` helper using pg_catalog queries, and add a `Logger` struct that writes the three log files. All changes stay inside `internal/restore/`.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| REST-03 | DROP-leak detection: compare object lists before/after DROP, warn on survivors | pg_catalog union query before DROP; same query after DROP; diff → warnings in drop.log |
| REST-10 | Restore granularity: DB-wide, schema-scoped (`--schema`), object-scoped (`--object`) | New `Options` struct passed to `RunRestore`; manifest `restore_order` filtered by scope |
| REST-11 | Partial restore auto-resolves transitive depends_on chain | BFS over `manifest.Objects[].DependsOn`; seed = requested object(s); collect closure |
| REST-12 | Missing dependency stops restore with clear error naming the missing object | After BFS, verify each dep ID appears in backup directory (or manifest object list); `fmt.Errorf("dependency %s not present in backup", id)` |
| REST-13 | drop.log, restore.log, summary.log written at end of every restore run | `Logger` struct writing to three `os.File` handles; flushed/closed via defer |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/jackc/pgx/v5` | v5.8.0 | PostgreSQL driver — pg_catalog queries for live-object snapshot | Already in `go.mod`; used throughout |
| `go.yaml.in/yaml/v3` | v3.0.4 | Manifest reading (already done in Phase 8) | Already in `go.mod` |
| `github.com/spf13/cobra` | v1.10.2 | New CLI flags (`--schema`, `--object`, `--log-dir`) | Already in `go.mod` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib `os`, `fmt`, `time`, `path/filepath` | stdlib | Log file creation, timestamping, path joining | REST-13 always |
| Go stdlib `sort` | stdlib | Deterministic log output | When emitting warning lists |

**Installation:** No new dependencies required.

---

## Architecture Patterns

### Recommended Project Structure

The Phase 8 layout is preserved. New files are additive:

```
internal/restore/
├── orchestrator.go       # RunRestore: extended with Options parameter
├── orchestrator_test.go  # Existing + new tests for REST-03, REST-10..13
├── loader.go             # Unchanged — loadObjectDef, defYAMLPath, serializerFor, ddlGeneratorFor
├── loader_test.go        # Unchanged
├── filter.go             # NEW: filteredRestoreOrder, buildTransitiveClosure, verifyDeps
├── filter_test.go        # NEW: unit tests for BFS closure, dep verification
├── leak.go               # NEW: snapshotLiveObjects, detectLeaks
├── leak_test.go          # NEW: unit tests for leak detection logic
└── logger.go             # NEW: Logger struct, drop.log / restore.log / summary.log writers
```

### Pattern 1: Options Struct for Scoped Restore

**What:** Pass a typed `Options` struct to `RunRestore` instead of positional booleans. The struct carries filter scope and log directory.

**When to use:** Always — replaces the current bare function signature `(ctx, conn, backupDir, preBackupDir string)`.

**Example:**
```go
// internal/restore/orchestrator.go
type Options struct {
    BackupDir    string
    PreBackupDir string
    Schema       string // REST-10: if non-empty, restrict to this schema
    Object       string // REST-10: if non-empty, "schema.name" format; implies REST-11 BFS
    LogDir       string // REST-13: directory for drop.log / restore.log / summary.log
}

func RunRestore(ctx context.Context, conn *pgx.Conn, opts Options) error { ... }
```

The CLI in `cmd/pgbackup/cmd/restore.go` constructs `Options` from cobra flags and passes it through.

### Pattern 2: BFS Transitive Closure from Manifest Objects

**What:** For object-scoped restores (`--object myschema.mytable`), walk `manifest.Objects[].DependsOn` in breadth-first order to collect all transitively required objects.

**When to use:** REST-11 — when `opts.Object` is non-empty.

**Example:**
```go
// internal/restore/filter.go

// buildTransitiveClosure returns the set of object IDs (schema.name) reachable
// from seeds by following DependsOn edges in the manifest.
func buildTransitiveClosure(manifest *resolve.Manifest, seeds []string) map[string]bool {
    // Build index: id -> ObjectEntry
    byID := make(map[string]resolve.ObjectEntry, len(manifest.Objects))
    for _, obj := range manifest.Objects {
        byID[obj.ID] = obj
    }

    visited := make(map[string]bool)
    queue := append([]string{}, seeds...)
    for len(queue) > 0 {
        id := queue[0]
        queue = queue[1:]
        if visited[id] {
            continue
        }
        visited[id] = true
        for _, dep := range byID[id].DependsOn {
            if !visited[dep] {
                queue = append(queue, dep)
            }
        }
    }
    return visited
}
```

### Pattern 3: Filtered Restore Order

**What:** Filter `manifest.RestoreOrder` to only the entries that belong to the requested scope. Preserve the topological order.

**When to use:** All partial restore paths (schema-scoped or object-scoped).

**Example:**
```go
// internal/restore/filter.go

// filteredRestoreOrder returns the subset of RestoreOrder entries matching the scope.
// For schema scope: keep entries where entry.Schema == schema.
// For object scope: keep entries where "schema.name" is in closure.
// For full restore: return manifest.RestoreOrder unchanged.
func filteredRestoreOrder(manifest *resolve.Manifest, opts Options) ([]resolve.RestoreEntry, error) {
    if opts.Object != "" {
        // REST-11: BFS closure
        closure := buildTransitiveClosure(manifest, []string{opts.Object})
        // REST-12: verify all deps exist in backup (manifest object list)
        if err := verifyDepsPresent(manifest, closure); err != nil {
            return nil, err
        }
        var result []resolve.RestoreEntry
        for _, e := range manifest.RestoreOrder {
            id := e.Schema + "." + e.Name
            if closure[id] {
                result = append(result, e)
            }
        }
        return result, nil
    }
    if opts.Schema != "" {
        // REST-10: schema filter
        var result []resolve.RestoreEntry
        for _, e := range manifest.RestoreOrder {
            if e.Schema == opts.Schema {
                result = append(result, e)
            }
        }
        return result, nil
    }
    // Full restore
    return manifest.RestoreOrder, nil
}
```

### Pattern 4: Dependency Verification (REST-12)

**What:** After computing the BFS closure, verify each object ID is present in the manifest's `Objects` list. An ID in `DependsOn` that has no corresponding `ObjectEntry` means the dependency was never backed up.

**When to use:** REST-12 — always called after BFS.

**Example:**
```go
// internal/restore/filter.go

func verifyDepsPresent(manifest *resolve.Manifest, closure map[string]bool) error {
    byID := make(map[string]bool, len(manifest.Objects))
    for _, obj := range manifest.Objects {
        byID[obj.ID] = true
    }
    for id := range closure {
        if !byID[id] {
            return fmt.Errorf("dependency %q is required but not present in backup", id)
        }
    }
    return nil
}
```

**Note:** The error message must clearly name the missing object — spec section 7.4 and REST-12 require this.

### Pattern 5: Drop-Leak Detection (REST-03)

**What:** Query the live database for all objects of the kinds tracked in the backup, before and after the DROP wave. Subtract the post-DROP set from the pre-DROP set; any object that survived but is NOT in the restore_order is a "leak."

**When to use:** REST-03 — always in `RunRestore`, immediately after the DROP wave.

**Algorithm:**
1. Before DROP wave: collect live objects for all schemas appearing in `manifest.RestoreOrder` → `preDrop`
2. Execute DROP wave (existing Wave 1 code)
3. After DROP wave: collect live objects again → `postDrop`
4. `leaked = preDrop - postDrop - {objects not in restore_order}` — but the simpler framing is: any object in `postDrop` that was also in the set that was supposed to be dropped is a leak.
5. Actually: `leaked = {objects in postDrop whose schema.name was in restore_order}` — these were supposed to be dropped but weren't.

**pg_catalog query for live object snapshot:**
```go
// internal/restore/leak.go
//
// liveObject is a simple kind+schema+name triple for comparison.
type liveObject struct {
    Schema string
    Kind   string
    Name   string
}

// snapshotLiveObjects queries pg_catalog for all objects of backed-up kinds
// across the schemas present in the manifest's restore_order.
func snapshotLiveObjects(ctx context.Context, conn *pgx.Conn, schemas []string) ([]liveObject, error) {
    // Build schema list for SQL IN clause
    // Query tables, views, matviews, functions, sequences, types, domains, enums, policies
    // using pg_class, pg_proc, pg_namespace union
}
```

The pg_catalog query pattern mirrors the existing `SchemaFetcher` implementations (Phase 2-3). For this purpose, a simpler enumeration query suffices — only schema + name + kind, no full definition needed.

**Reference queries (HIGH confidence — same pg_catalog tables used in Phase 2-3 fetchers):**

```sql
-- Tables
SELECT n.nspname AS schema, 'table' AS kind, c.relname AS name
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'r' AND n.nspname = ANY($1)

-- Views
SELECT n.nspname, 'view', c.relname
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'v' AND n.nspname = ANY($1)

-- Materialized views
SELECT n.nspname, 'materialized_view', c.relname
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'm' AND n.nspname = ANY($1)

-- Sequences
SELECT n.nspname, 'sequence', c.relname
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'S' AND n.nspname = ANY($1)

-- Functions (only non-aggregate, non-window, user-defined)
SELECT n.nspname, 'function', p.proname
FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE p.prokind = 'f' AND n.nspname = ANY($1)

-- Types (only composite types)
SELECT n.nspname, 'type', t.typname
FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE t.typtype = 'c' AND n.nspname = ANY($1)

-- Domains
SELECT n.nspname, 'domain', t.typname
FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE t.typtype = 'd' AND n.nspname = ANY($1)

-- Enums
SELECT n.nspname, 'enum', t.typname
FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE t.typtype = 'e' AND n.nspname = ANY($1)
```

These can be UNION-ed into a single query or executed separately; separate queries are simpler to read.

### Pattern 6: Structured Log Output (REST-13)

**What:** Write three log files to a timestamped directory. Entries are appended as the restore progresses; summary is written at the end.

**When to use:** Always in `RunRestore`.

**Log directory:** `{LogDir}/restore_YYYYMMDD_HHMMSS/` where `LogDir` defaults to the directory containing the backup being restored. If `opts.LogDir` is empty, use `filepath.Dir(opts.BackupDir)`.

**Example:**
```go
// internal/restore/logger.go

type Logger struct {
    dropLog    *os.File
    restoreLog *os.File
    summaryLog *os.File

    dropCount    int
    restoreCount int
    warnCount    int
    startTime    time.Time
}

func NewLogger(logDir string) (*Logger, error) {
    ts := time.Now().Format("20060102_150405")
    dir := filepath.Join(logDir, "restore_"+ts)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, fmt.Errorf("create log dir: %w", err)
    }
    // open drop.log, restore.log, summary.log
    ...
}

func (l *Logger) LogDrop(schema, kind, name string, err error) {
    status := "OK"
    if err != nil {
        status = fmt.Sprintf("ERROR: %v", err)
    }
    fmt.Fprintf(l.dropLog, "[%s] DROP  schema=%s kind=%-5s name=%-30s → %s\n",
        time.Now().Format("2006-01-02 15:04:05"), schema, kind, name, status)
    l.dropCount++
}

func (l *Logger) LogRestore(schema, kind, name string, err error) { ... }
func (l *Logger) LogLeakWarning(schema, kind, name string)        { ... }
func (l *Logger) WriteSummary(...)                                 { ... }
func (l *Logger) Close() error                                     { ... }
```

**Log entry format (from spec section 8.2):**
```
[2024-01-01 12:00:00] DROP START  schema=public kind=view  name=active_users → OK
[2024-01-01 12:00:01] DROP VERIFY 期待DROP数=5 実績DROP数=5 → 一致
[2024-01-01 12:00:03] DROP VERIFY 警告: public.extra_view (view) survived DROP
```

### Anti-Patterns to Avoid

- **Re-sorting the filtered RestoreOrder:** Do not re-sort after filtering. The manifest's `restore_order` is already topologically ordered by Phase 6 Kahn's sort. Filtering preserves that order; re-sorting would break dependency ordering.
- **Using `information_schema` for live object snapshot:** `information_schema.tables` does not include sequences, types, domains, enums, or functions. Use `pg_catalog` directly — the same source used by all fetchers in the project.
- **Building dep closure from live DB instead of manifest:** REST-11 says "depends_on再帰走査" — walk the manifest's `depends_on` field, not live DB FK relationships. The manifest already has the computed dependency graph.
- **Writing logs inside the backup directory:** Log files go in a separate `restore_YYYYMMDD_HHMMSS/` directory. Do not write them inside `backupDir` (that's the source, not a workspace).
- **Panicking on missing log directory:** If the log directory cannot be created (permissions), degrade gracefully — log to stderr and continue the restore. Failing a restore because logs can't be written would be wrong.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Transitive dep graph | Custom recursive SQL query | Walk `manifest.Objects[].DependsOn` (already computed by Phase 6 Kahn) | DAG already serialized; no live DB query needed |
| Live object enumeration | Full SchemaFetcher re-invocation | Direct `pg_catalog` queries (same tables, simpler columns) | Only need schema/name/kind, not full definitions |
| Log file rotation/formatting | `logrus`, `zap`, custom framework | Go stdlib `os.File` + `fmt.Fprintf` | Zero deps; spec format is simple fixed-width text |
| Scope filtering | New CLI subcommand | `--schema` / `--object` flags on existing `restore` command | Spec section 7.2 defines them as flags, not subcommands |
| Topological re-sort for partial scope | Re-run Kahn on subset | Filter `manifest.RestoreOrder` slice in place | Manifest order already correct; filtering preserves it |

**Key insight:** Phase 9 is a filtering and instrumentation layer on top of Phase 8's pipeline. Every building block already exists.

---

## Common Pitfalls

### Pitfall 1: BFS Closure Must Include FK Entries

**What goes wrong:** The manifest `objects` list includes FK entries with `kind=fk`. When doing object-scoped BFS from `myschema.mytable`, the FK objects for that table (`myschema.mytable_fkey`) are FK entries, not regular objects. If the BFS only walks non-FK entries, FK entries that depend on the requested table will be omitted from the restore.

**Why it happens:** FK entries have kind `fk` in the manifest objects list but `foreign_key` in `core.ObjectKind`. The `DependsOn` for an FK entry lists `schema.source_table` and `schema.target_table` as IDs. The BFS must include FK entries whose `DependsOn` overlaps the closure.

**How to avoid:** During BFS closure building, also scan FK entries in `manifest.Objects` for any `DependsOn` IDs that are already in the closure — add those FK entries to the closure too. This is a post-BFS pass, not a standard edge-follow.

**Warning signs:** Foreign key constraints missing after object-scoped restore.

### Pitfall 2: Schema-Scoped Restore Drops Cross-Schema Objects

**What goes wrong:** If `--schema myschema` is requested but `restore_order` contains objects from other schemas (e.g., `public.some_enum` that `myschema.mytable` depends on), those cross-schema objects will not be dropped or restored. If they don't exist in the target DB, the restore will fail.

**Why it happens:** Schema-scoped filter keeps only `entry.Schema == opts.Schema` entries. Cross-schema dependencies are silently absent.

**How to avoid:** For schema-scoped restores, assume cross-schema dependencies exist in the target DB and are not managed by this restore. This is the correct behavior (spec section 7.2 defines schema-scoped as "指定スキーマのオブジェクト全て" — all objects of the specified schema). Document this assumption in the log file. If a cross-schema dependency is missing, the CREATE DDL will fail with a PostgreSQL error — this is expected behavior, not a bug.

**Warning signs:** `ERROR: type "public.status_type" does not exist` during schema-scoped restore.

### Pitfall 3: Drop-Leak Includes Pre-Existing Non-Backup Objects

**What goes wrong:** The drop-leak detection compares pre-DROP and post-DROP object snapshots. Any object that existed before the backup that is NOT in `restore_order` will appear to survive the DROP phase — but it was never supposed to be dropped. This would produce false-positive leak warnings.

**Why it happens:** The spec (section 8.3) says to warn only about objects that "survived" the DROP phase where those objects WERE in the restore_order. The implementation must intersect post-DROP survivors with the `restore_order` set, not compare raw snapshots.

**How to avoid:** Build a `shouldBeDropped` set from `restore_order` before the DROP wave. After the DROP wave, re-query live objects. Warn only if `(post-DROP object) AND (schema.name is in shouldBeDropped)`. Pre-existing objects not in `restore_order` are expected to remain and should not be warned about.

**Warning signs:** Spurious leak warnings about user-created objects that were never in the backup.

### Pitfall 4: `--object` Flag Requires `schema.name` Format

**What goes wrong:** If the user passes `--object mytable` (without schema), the BFS seed `mytable` won't match any manifest object ID (which are always `schema.name` format).

**Why it happens:** All manifest object IDs use `schema.name` format per Phase 6 decisions.

**How to avoid:** Validate that `opts.Object` contains a `.` separator. If not, return early with `fmt.Errorf("--object must be in schema.name format, got %q", opts.Object)`.

**Warning signs:** Empty restore (no objects match) instead of a clear error.

### Pitfall 5: Logger Close Must be Deferred After Summary Write

**What goes wrong:** If `RunRestore` returns an error partway through, the log files may be left open without the summary. If `Close()` is deferred before `WriteSummary()` is called, the summary may not be written on error.

**Why it happens:** Defer ordering — deferred calls run in LIFO order.

**How to avoid:** Write the summary before closing. Use a single cleanup function:
```go
logger, err := NewLogger(logDir)
if err != nil { /* handle */ }
defer func() {
    logger.WriteSummary(...)
    logger.Close()
}()
```

**Warning signs:** `summary.log` absent or empty when restore fails mid-way.

### Pitfall 6: RunRestore Signature Change Breaks Existing Tests

**What goes wrong:** Phase 8 tests call `restore.RunRestore(ctx, conn, backupDir, preBackupDir)`. Adding an `Options` struct changes the signature.

**Why it happens:** Breaking change to the public function signature.

**How to avoid:** Keep the existing `RunRestore(ctx, conn, backupDir, preBackupDir string)` as a compatibility wrapper, or update all callers in the same plan. The Phase 8 test file is at `internal/restore/orchestrator_test.go` and must be updated. Update the CLI wiring in `cmd/pgbackup/cmd/restore.go` simultaneously.

**Warning signs:** Compile errors in `orchestrator_test.go` and `cmd/pgbackup/cmd/restore.go`.

---

## Code Examples

Verified patterns from existing codebase and spec:

### Options-Driven RunRestore Signature
```go
// internal/restore/orchestrator.go
// Source: Phase 8 RunRestore + new Options pattern
type Options struct {
    BackupDir    string
    PreBackupDir string
    Schema       string
    Object       string
    LogDir       string
}

func RunRestore(ctx context.Context, conn *pgx.Conn, opts Options) error {
    // Validate
    if opts.Object != "" && !strings.Contains(opts.Object, ".") {
        return fmt.Errorf("--object must be in schema.name format, got %q", opts.Object)
    }
    // ... rest of pipeline
}
```

### Manifest Object ID → BFS Walk
```go
// internal/restore/filter.go
// Source: resolve.Manifest struct (manifest.go) — Objects[].DependsOn is []string of "schema.name" IDs
func buildTransitiveClosure(objects []resolve.ObjectEntry, seeds []string) map[string]bool {
    byID := make(map[string]resolve.ObjectEntry, len(objects))
    for _, o := range objects {
        byID[o.ID] = o
    }
    visited := make(map[string]bool)
    queue := make([]string, len(seeds))
    copy(queue, seeds)
    for len(queue) > 0 {
        id := queue[0]
        queue = queue[1:]
        if visited[id] {
            continue
        }
        visited[id] = true
        for _, dep := range byID[id].DependsOn {
            if !visited[dep] {
                queue = append(queue, dep)
            }
        }
    }
    return visited
}
```

### Live Object Snapshot via pg_catalog
```go
// internal/restore/leak.go
// Source: same pg_catalog tables used in internal/fetch/* fetchers (HIGH confidence)
const liveObjectsSQL = `
    SELECT n.nspname, 'table',             c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname=ANY($1)
    UNION ALL
    SELECT n.nspname, 'view',              c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='v' AND n.nspname=ANY($1)
    UNION ALL
    SELECT n.nspname, 'materialized_view', c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='m' AND n.nspname=ANY($1)
    UNION ALL
    SELECT n.nspname, 'sequence',          c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='S' AND n.nspname=ANY($1)
    UNION ALL
    SELECT n.nspname, 'function',          p.proname FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE p.prokind='f' AND n.nspname=ANY($1)
    UNION ALL
    SELECT n.nspname, 'type',              t.typname FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE t.typtype='c' AND n.nspname=ANY($1)
    UNION ALL
    SELECT n.nspname, 'domain',            t.typname FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE t.typtype='d' AND n.nspname=ANY($1)
    UNION ALL
    SELECT n.nspname, 'enum',              t.typname FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE t.typtype='e' AND n.nspname=ANY($1)
`
```

### Logger Construction and Use
```go
// internal/restore/logger.go
// Source: spec section 8.1-8.3 — log format and file names

func NewLogger(logDir string) (*Logger, error) {
    ts := time.Now().Format("20060102_150405")
    dir := filepath.Join(logDir, "restore_"+ts)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, fmt.Errorf("create log dir %s: %w", dir, err)
    }
    openLog := func(name string) (*os.File, error) {
        return os.Create(filepath.Join(dir, name))
    }
    drop, err := openLog("drop.log");    if err != nil { return nil, err }
    restore, err := openLog("restore.log"); if err != nil { return nil, err }
    summary, err := openLog("summary.log"); if err != nil { return nil, err }
    return &Logger{dropLog: drop, restoreLog: restore, summaryLog: summary, startTime: time.Now()}, nil
}
```

### CLI Flag Wiring
```go
// cmd/pgbackup/cmd/restore.go
// Source: existing cobra pattern in this file
restoreCmd.Flags().StringVar(&restoreSchema, "schema", "",
    "Restore only objects in this schema (e.g. myschema)")
restoreCmd.Flags().StringVar(&restoreObject, "object", "",
    "Restore this object and its transitive dependencies (e.g. myschema.mytable)")
restoreCmd.Flags().StringVar(&restoreLogDir, "log-dir", "",
    "Directory for restore log files (default: parent of --input dir)")
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Full restore only | Granular restore (schema/object scope) | Phase 9 | Users can target specific schemas or objects |
| No DROP verification | Drop-leak detection pre/post DROP | Phase 9 | Warns about objects that survive the DROP unexpectedly |
| No log files | Three structured log files per run | Phase 9 | Auditable restore operations |
| Positional string args to RunRestore | `Options` struct | Phase 9 | Extensible for future flags without signature churn |

**Deprecated/outdated:**
- The bare `RunRestore(ctx, conn, backupDir, preBackupDir string)` signature will be replaced by `RunRestore(ctx, conn, Options)`. All callers must be updated in the same wave.

---

## Open Questions

1. **Schema-Scoped DROP Behavior**
   - What we know: `--schema myschema` should restore only that schema's objects. The spec says "指定スキーマのオブジェクト全て."
   - What's unclear: Should the DROP wave also be scoped to that schema? Or should it still drop all objects in `restore_order`?
   - Recommendation: Scope the DROP wave to match the restore scope. If only `myschema` is being restored, only drop `myschema` objects. This is the safest behavior — avoids dropping objects from other schemas that aren't being restored.

2. **Log Directory Default When `--log-dir` Not Provided**
   - What we know: Log files need to go somewhere. The spec shows a `restore_YYYYMMDD_HHMMSS/` directory without specifying its parent.
   - What's unclear: Use `filepath.Dir(opts.BackupDir)` (sibling of backup), or current working directory?
   - Recommendation: Default to the current working directory (`"."`), matching how `--pre-backup-dir` defaults to `"."` in Phase 8. This is the least surprising behavior.

3. **BFS Closure and Trigger/Policy Objects**
   - What we know: Triggers and policies have their parent table in `DependsOn`. If `myschema.mytable` is requested, `myschema.audit_trigger` (depends on `myschema.mytable`) will be included in the BFS if we walk reverse edges.
   - What's unclear: BFS as described walks forward (from requested object following its `depends_on` to find prerequisites). It does NOT automatically include objects that depend ON the requested object (like a trigger on that table). Should triggers/policies attached to a requested table be included?
   - Recommendation: For `--object`, include only the transitive prerequisites (BFS forward along `depends_on`). Triggers and policies are dependents, not dependencies. They will not be restored unless explicitly requested. Document this clearly.

4. **`--object` with Multiple Objects**
   - What we know: The flag is defined as a single `schema.name` string.
   - What's unclear: Should multiple `--object` flags be supported (cobra supports `StringArrayVar`)?
   - Recommendation: Support single object only in Phase 9 (simplest). Multiple objects can be added as v2 enhancement. Document in `--help`.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + `go test` |
| Config file | none — uses `go test ./...` |
| Quick run command | `go test ./internal/restore/... -short` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REST-03 | Drop-leak detected and warned after DROP wave | integration | `go test ./internal/restore/... -run TestDropLeakDetection` | Wave 0 |
| REST-10 | `--schema` flag restores only that schema's objects | integration | `go test ./internal/restore/... -run TestSchemaScoped` | Wave 0 |
| REST-11 | `--object` restores object + all transitive deps | integration | `go test ./internal/restore/... -run TestObjectScopedTransitive` | Wave 0 |
| REST-12 | Missing dep stops restore with named error | unit | `go test ./internal/restore/... -run TestMissingDependency` | Wave 0 |
| REST-13 | drop.log, restore.log, summary.log written on every run | integration | `go test ./internal/restore/... -run TestLogFiles` | Wave 0 |
| REST-03 (unit) | Leak detection logic (no DB) | unit | `go test ./internal/restore/... -run TestDetectLeaks -short` | Wave 0 |
| REST-11 (unit) | BFS closure correctness | unit | `go test ./internal/restore/... -run TestBuildTransitiveClosure -short` | Wave 0 |

Integration tests skip if `TEST_DATABASE_URL` not set. Unit tests for BFS closure, leak detection logic, and dep verification run without a DB (use in-memory `resolve.Manifest` structs).

### Sampling Rate
- **Per task commit:** `go test ./internal/restore/... -short`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/restore/filter.go` + `filter_test.go` — BFS closure, filteredRestoreOrder, verifyDepsPresent (unit tests, no DB needed)
- [ ] `internal/restore/leak.go` + `leak_test.go` — snapshotLiveObjects, detectLeaks
- [ ] `internal/restore/logger.go` — Logger struct, NewLogger, LogDrop, LogRestore, LogLeakWarning, WriteSummary, Close
- [ ] `internal/restore/orchestrator.go` updated to accept `Options` struct (breaking change — all callers updated in same wave)
- [ ] `internal/restore/orchestrator_test.go` updated for new signature + stub tests for REST-03, REST-10, REST-11, REST-12, REST-13
- [ ] `cmd/pgbackup/cmd/restore.go` updated with `--schema`, `--object`, `--log-dir` flags constructing `Options`

---

## Sources

### Primary (HIGH confidence)
- `internal/restore/orchestrator.go` — current RunRestore pipeline (Phase 8 baseline)
- `internal/restore/loader.go` — loadObjectDef, kindToDir, serializerFor, ddlGeneratorFor
- `internal/resolve/manifest.go` — Manifest, ObjectEntry (DependsOn field), RestoreEntry structs
- `internal/resolve/dag.go` — dependency graph structure; confirms DependsOn uses "schema.name" IDs
- `cmd/pgbackup/cmd/restore.go` — current CLI flag wiring; defines signature change target
- `docs/pre/pgbackup_spec.md` sections 7.2, 7.4, 8.1-8.3 — authoritative spec for granularity, dep resolution, log format
- `internal/fetch/table/fetcher.go` — pg_catalog query patterns for pg_class; confirms relkind codes
- `.planning/REQUIREMENTS.md` — REST-03, REST-10, REST-11, REST-12, REST-13 confirmed

### Secondary (MEDIUM confidence)
- Phase 8 RESEARCH.md — established patterns for orchestrator structure, test setup, pgx usage
- PostgreSQL documentation — `pg_class.relkind` values ('r'=table, 'v'=view, 'm'=matview, 'S'=sequence), `pg_proc.prokind` ('f'=function), `pg_type.typtype` ('c'=composite, 'd'=domain, 'e'=enum) — consistent with existing fetch code

### Tertiary (LOW confidence)
- None — all findings verified against project source or spec

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new deps; all libraries already in use
- Architecture (Options struct, BFS, filter): HIGH — derived directly from manifest struct and spec
- Drop-leak detection: HIGH — pg_catalog queries confirmed against existing fetcher source
- Log file format: HIGH — spec section 8.1-8.3 is authoritative
- Pitfalls: HIGH — all derived from reading actual Phase 8 code

**Research date:** 2026-03-12
**Valid until:** Stable — all sources are internal project files + spec; no external dependency drift
