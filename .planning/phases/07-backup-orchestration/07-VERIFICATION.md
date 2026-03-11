---
phase: 07-backup-orchestration
verified: 2026-03-12T00:00:00Z
status: passed
score: 9/9 must-haves verified
re_verification: false
---

# Phase 7: Backup Orchestration Verification Report

**Phase Goal:** Running `pgbackup backup` against a live database produces a complete, correctly structured backup directory that can be used for restore and diff operations
**Verified:** 2026-03-12
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A table with a bytea, xml, pg_lsn, or txid_snapshot column is detected as skippable before any I/O occurs | VERIFIED | `skip.go` iterates columns with `strings.Contains`; `TestSkipUnsupportedColumns` covers all 4 types + bytea[] array variant; all 6 sub-tests PASS |
| 2 | A non-skipped regular table produces a directory at `<backup_root>/<schema>/tables/<name>/` with def.yaml and data.csv | VERIFIED | `tableObjectDir` returns `filepath.Join(backupRoot, schema, "tables", td.Name)` for non-partition tables; `os.MkdirAll` creates it; `os.WriteFile` writes `def.yaml`; `export.ExportTableData` writes `data.csv` when `ShouldExportData` is true |
| 3 | A non-table object (view, function, etc.) produces a YAML file at `<backup_root>/<schema>/<kind-dir>/<name>.yaml` | VERIFIED | `kindToDir` map covers all non-table kinds; orchestrator calls `serializerFor(kf.kind).Serialize(def)` and writes to `filepath.Join(dir, def.Header().Name+".yaml")` |
| 4 | A partitioned table's children are routed to `<backup_root>/<schema>/tables/<parent>/partitions/<child>/` not top-level `tables/` | VERIFIED | `buildChildOf` constructs child→parent map from `Partitioning.Children`; `tableObjectDir` returns `filepath.Join(backupRoot, schema, "tables", parent, "partitions", td.Name)` when parent found in map |
| 5 | def.yaml for non-partitioned tables includes a data: section with file, columns, row_count, and checksum | VERIFIED | `toYAML` in `serializer.go` emits `yd.Data = &yamlData{...}` when `td.DataMeta != nil`; orchestrator populates `td.DataMeta` from `export.ExportTableData` result; `TestSerialize` suite passes |
| 6 | Running `pgbackup backup --snapshot` wraps all fetches and COPY TO in a single REPEATABLE READ transaction | VERIFIED | `RunBackup` opens `conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})` when `snapshot=true`; `queryConn = tx.Conn()` threads connection through all `Fetch` and `ExportTableData` calls; `tx.Commit` called after all schemas processed |
| 7 | The `_manifest.yaml` records `snapshot: true` when `--snapshot` was used, `snapshot: false` otherwise | VERIFIED | `resolve.ManifestParams{Snapshot: snapshot}` passes the boolean to `resolve.BuildManifest`; integration tests `TestBackupSnapshotMode` / `TestBackupNonSnapshotMode` assert manifest field |
| 8 | `backupCmd.RunE` calls `backup.RunBackup` with the correct conn, outDir, and snapshot arguments | VERIFIED | `cmd/pgbackup/cmd/backup.go` line 17: `return backup.RunBackup(cmd.Context(), conn, outputDir, snapshotFlag)` — fully wired; `backupCmd` registered in `root.go` line 57 |
| 9 | Integration tests against live DB verify the full backup directory tree exists after RunBackup | VERIFIED | `TestBackupIntegration`, `TestBackupSnapshotMode`, `TestBackupNonSnapshotMode` all present with `connectTestDB` / `findBackupRoot` / `parseManifest` helpers; skip cleanly in `-short` mode; `go test ./internal/backup/... -short` exits 0 |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/backup/skip.go` | skipReason() helper for skip detection | VERIFIED | 27 lines; `skipTypes` var + `skipReason` func using `strings.Contains`; handles bytea[] array variants |
| `internal/backup/orchestrator.go` | RunBackup entry point and core directory layout logic | VERIFIED | 322 lines; `RunBackup`, `discoverSchemas`, `buildChildOf`, `tableObjectDir`, `columnNames`, `serializerFor` all implemented; snapshot branch and non-snapshot path both present |
| `internal/backup/orchestrator_test.go` | Unit tests + integration tests | VERIFIED | 299 lines; `TestSkipUnsupportedColumns` (6 sub-tests PASS), `TestSkipWarning` (PASS), 3 integration tests skip cleanly in `-short` mode with full implementations |
| `internal/core/types.go` | TableDef.DataMeta field + TableDataMeta struct | VERIFIED | `TableDataMeta` struct at line 92; `DataMeta *TableDataMeta` field in `TableDef` at line 109 |
| `internal/serialize/table/serializer.go` | DataMeta emission in def.yaml when non-nil | VERIFIED | `toYAML` emits `yd.Data = &yamlData{...}` at line 139–146 when `td.DataMeta != nil`; existing serializer tests pass |
| `cmd/pgbackup/cmd/backup.go` | --snapshot flag wired to RunBackup, --output flag | VERIFIED | 27 lines; both flags defined in `init()`; `RunE` calls `backup.RunBackup` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/backup/orchestrator.go` | `internal/export/csv.go` | `export.ExportTableData` returns `*export.DataMeta`, assigned to `TableDef.DataMeta` | VERIFIED | Line 155: `meta, err := export.ExportTableData(ctx, queryConn, ...)` then `td.DataMeta = &core.TableDataMeta{...meta fields...}` at line 159 |
| `internal/backup/orchestrator.go` | `internal/serialize/table/serializer.go` | `serializeTable.Serializer{}.Serialize(td)` emits data: block from td.DataMeta | VERIFIED | Line 168-169: `s := &serializeTable.Serializer{}` then `s.Serialize(td)` |
| `internal/backup/orchestrator.go` | `internal/fetch/table/fetcher.go` | `fetchTable.SchemaFetcher{}.Fetch(ctx, conn, schema)` | VERIFIED | Line 123: `(&fetchTable.SchemaFetcher{}).Fetch(ctx, queryConn, schema)` |
| `cmd/pgbackup/cmd/backup.go` | `internal/backup/orchestrator.go` | `backup.RunBackup(cmd.Context(), conn, outputDir, snapshotFlag)` | VERIFIED | Line 17: exact match; `backupCmd` added to `rootCmd` in `root.go:57` |
| `internal/backup/orchestrator.go` | `pgx.Tx` | `conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})` when snapshot=true | VERIFIED | Lines 84-88: `conn.BeginTx` with `RepeatableRead` + `ReadOnly`; `defer tx.Rollback(ctx)` + `tx.Commit(ctx)` |
| `orchestrator.go snapshot branch` | `export.ExportTableData` | `tx.Conn()` passed as `*pgx.Conn` so COPY participates in the transaction | VERIFIED | Line 92: `queryConn = tx.Conn()`; line 155: `export.ExportTableData(ctx, queryConn, ...)` uses this value |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| BKUP-01 | 07-01, 07-02 | スキーマ単位のディレクトリ構造生成（仕様書セクション4準拠） | SATISFIED | `kindToDir` map + `tableObjectDir` + `buildChildOf` implement full spec section 4 layout including partition routing |
| BKUP-02 | 07-01 | スキップ対象テーブル検出（bytea, xml, pg_lsn, txid_snapshot列）と警告ログ | SATISFIED | `skip.go` `skipReason()` + `fmt.Fprintf(os.Stderr, "WARNING: ...")` in orchestrator; 6 unit test cases all PASS |
| BKUP-03 | 07-02 | --snapshotオプションによるトランザクション一貫性モード | SATISFIED | Full REPEATABLE READ transaction branch in `RunBackup`; `--snapshot` flag wired in `backup.go`; integration test verifies manifest records `snapshot: true` |
| BKUP-04 | 07-02 | スナップショットなしモード（デフォルト、各オブジェクト個別取得） | SATISFIED | `queryConn = conn` (non-snapshot path); `--snapshot` defaults to `false`; integration test verifies manifest records `snapshot: false` |

No orphaned requirements: REQUIREMENTS.md traceability table maps BKUP-01, BKUP-02, BKUP-03, BKUP-04 to Phase 7; all four are claimed by plans 07-01 and 07-02.

### Anti-Patterns Found

None. Scanned `internal/backup/` and `cmd/pgbackup/cmd/backup.go` for TODO/FIXME/placeholder comments, empty implementations, and stub returns. Zero findings.

### Human Verification Required

#### 1. Integration test against live database

**Test:** Set `TEST_DATABASE_URL=postgres://...` pointing at a populated PostgreSQL database and run `go test ./internal/backup/... -count=1 -run TestBackupIntegration`
**Expected:** Test passes; a `backup_YYYYMMDD_HHMMSS/` directory is created in a temp dir; `_manifest.yaml` is present with non-empty `backup_at` and `pg_version` fields
**Why human:** Requires a live PostgreSQL instance; CI environment may not have one available

#### 2. Snapshot mode produces consistent read

**Test:** Run `pgbackup backup --snapshot --output /tmp/test` against a busy database with concurrent writes
**Expected:** All objects in the backup reflect a single consistent point in time (no half-committed transactions visible in data.csv)
**Why human:** Requires concurrent workload to exercise the REPEATABLE READ isolation guarantee; cannot be verified statically

#### 3. Directory layout for partitioned tables

**Test:** Run `pgbackup backup` against a database with a partitioned table (e.g. `orders` partitioned by `created_at` with children `orders_2024`, `orders_2025`)
**Expected:** `backup_root/public/tables/orders/def.yaml` exists (parent); `backup_root/public/tables/orders/partitions/orders_2024/def.yaml` and `data.csv` exist (children)
**Why human:** Requires a database with actual partitioned tables to exercise the `buildChildOf` routing live

### Gaps Summary

No gaps. All 9 observable truths are verified against the actual codebase. All 4 required artifacts exist, are substantive, and are wired. All 3 key links from Plan 07-01 and 3 from Plan 07-02 are present and connected. All requirement IDs (BKUP-01 through BKUP-04) are fully accounted for. The entire test suite passes in `-short` mode with zero failures.

---
_Verified: 2026-03-12_
_Verifier: Claude (gsd-verifier)_
