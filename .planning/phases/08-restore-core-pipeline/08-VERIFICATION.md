---
phase: 08-restore-core-pipeline
verified: 2026-03-12T00:00:00Z
status: human_needed
score: 10/10 must-haves verified
re_verification: false
human_verification:
  - test: "Run pgbackup backup then pgbackup restore against a real PostgreSQL database"
    expected: "Tables, data, sequences, views, FK constraints, and indexes all reconstructed correctly; pre-restore backup directory is populated"
    why_human: "Integration tests for REST-01 through REST-09 skip without TEST_DATABASE_URL; the 7-wave pipeline correctness can only be confirmed with a live database"
  - test: "Run pgbackup restore --help"
    expected: "Output shows --input (required) and --pre-backup-dir (optional, default '.') flags"
    why_human: "Cannot invoke compiled binary in this environment; CLI flag display requires running the built binary"
---

# Phase 08: Restore Core Pipeline Verification Report

**Phase Goal:** Running `pgbackup restore` against a backup directory fully reconstructs the database — dropping existing objects in reverse order, creating them in dependency order, loading data, restoring sequence values, creating indexes, and applying FK constraints as a final batch
**Verified:** 2026-03-12T00:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | ReadManifest exists in internal/resolve and correctly parses _manifest.yaml | VERIFIED | `internal/resolve/manifest.go` lines 199-210; TestReadManifest_NonexistentPath and TestReadManifest_ValidFile PASS |
| 2 | yamlIndex carries a definition field that round-trips IndexDef.Definition through YAML | VERIFIED | `internal/serialize/table/serializer.go` line 63: `Definition string \`yaml:"definition,omitempty"\`}`; toYAML/fromYAML both populate it; TestIndexDefinitionRoundTrip PASS |
| 3 | defYAMLPath returns correct paths for regular tables, partition children, non-table objects, and empty string for FK | VERIFIED | `internal/restore/loader.go` lines 51-66; TestDefYAMLPath (5 subtests) all PASS |
| 4 | loadObjectDef deserializes each of the 10 supported object kinds from a def.yaml file | VERIFIED | `internal/restore/loader.go` lines 133-153; serializerFor covers all 10 non-FK kinds; TestDeserializeRoundTrip PASS |
| 5 | RunRestore calls backup.RunBackup before any DROP statements (REST-01) | VERIFIED | `internal/restore/orchestrator.go` line 33: `backup.RunBackup(ctx, conn, preBackupDir, false)` is Wave 0, before the DROP loop at line 45 |
| 6 | Objects are dropped in exact reverse of manifest restore_order (REST-02) | VERIFIED | `internal/restore/orchestrator.go` lines 45-50: iterates `len(manifest.RestoreOrder)-1` down to 0 |
| 7 | Tables/sequences/types/domains/enums created in manifest order before data load; indexes deferred to Wave 5 (REST-04, REST-07) | VERIFIED | Wave 2 at lines 52-88 handles schema-only kinds; table wave executes only `stmts[0]` (CREATE TABLE); Wave 5 at lines 121-141 executes IndexDef.Definition strings |
| 8 | COPY FROM loads data for regular and partition child tables (REST-05) | VERIFIED | `copyFromTable` at lines 232-263 opens data.csv and calls `conn.PgConn().CopyFrom`; handles partition path via FromTable |
| 9 | SETVAL restores each sequence to backed-up last_value with is_called flag, after COPY FROM (REST-06) | VERIFIED | Wave 4 at lines 101-118: `SELECT setval('%s.%s', %d, %v)` using `sd.LastValue, sd.IsCalled`; runs after Wave 3 COPY FROM |
| 10 | FK constraints applied as single batch after all tables and data exist; views/functions/triggers/policies created after FK batch (REST-08, REST-09) | VERIFIED | Wave 6 at lines 143-151 iterates only "fk" entries via execFKCreate (re-fetches from live DB); Wave 7 at lines 153-178 handles remainder kinds |
| 11 | pgbackup restore CLI subcommand wired with --input (required) and --pre-backup-dir (optional) flags calling RunRestore | VERIFIED | `cmd/pgbackup/cmd/restore.go`: calls `restore.RunRestore(cmd.Context(), conn, restoreInputDir, restorePreBackupDir)`; `MarkFlagRequired("input")` at line 26; registered in root.go line 57 |

**Score:** 10/10 structural truths verified (automated); 2 behaviors require human verification with live database

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/resolve/manifest.go` | ReadManifest function | VERIFIED | Present at line 199; reads file, unmarshals YAML, returns *Manifest |
| `internal/serialize/table/serializer.go` | yamlIndex with definition field | VERIFIED | Line 63: `Definition string \`yaml:"definition,omitempty"\``; toYAML/fromYAML both propagate the field |
| `internal/restore/loader.go` | defYAMLPath, serializerFor, ddlGeneratorFor, loadObjectDef | VERIFIED | All 4 functions implemented; 154 lines, substantive implementation covering 11 object kinds |
| `internal/restore/orchestrator.go` | RunRestore — full 7-wave pipeline | VERIFIED | 303 lines; all 7 waves implemented with helpers execDrop, copyFromTable, execFKCreate |
| `internal/restore/orchestrator_test.go` | 8 integration test stubs for REST-01..09 | VERIFIED | 384 lines; 8 tests with real logic (not `t.Fatal("not implemented")`); requireDB() skip guard present |
| `internal/restore/loader_test.go` | Unit test stubs for loader functions | VERIFIED | 175 lines; 5 tests (TestDefYAMLPath, TestSerializerFor, TestDDLGeneratorFor, TestDeserializeRoundTrip, TestReadManifestRoundTrip) all PASS |
| `cmd/pgbackup/cmd/restore.go` | Wired restore subcommand with --input and --pre-backup-dir flags | VERIFIED | 27 lines; restoreCmd defined, flags wired, MarkFlagRequired("input"), registered via rootCmd.AddCommand |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/restore/orchestrator.go` | `internal/backup.RunBackup` | pre-restore backup call | WIRED | Line 33: `backup.RunBackup(ctx, conn, preBackupDir, false)` |
| `internal/restore/orchestrator.go` | `conn.PgConn().CopyFrom` | data load wave | WIRED | Line 260: `conn.PgConn().CopyFrom(ctx, f, sql)` |
| `internal/restore/orchestrator.go` | `internal/resolve.ReadManifest` | manifest loading | WIRED | Line 39: `resolve.ReadManifest(manifestPath)` |
| `cmd/pgbackup/cmd/restore.go` | `internal/restore.RunRestore` | RunE function body | WIRED | Line 17: `restore.RunRestore(cmd.Context(), conn, restoreInputDir, restorePreBackupDir)` |
| `internal/restore/orchestrator.go` | `fetchForeignkey.SchemaFetcher.Fetch` | FK re-fetch from live DB | WIRED | Line 280: `fetcher.Fetch(ctx, conn, entry.Schema)` inside execFKCreate |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| REST-01 | 08-01, 08-02, 08-03 | Pre-restore automatic backup | SATISFIED | Wave 0 in orchestrator.go line 33 calls backup.RunBackup before any mutation |
| REST-02 | 08-01, 08-02, 08-03 | DROP in reverse restore_order | SATISFIED | Wave 1 in orchestrator.go lines 45-50 iterates manifest.RestoreOrder in reverse |
| REST-04 | 08-01, 08-02, 08-03 | CREATE in restore_order | SATISFIED | Wave 2 in orchestrator.go lines 52-88 creates tables/sequences/types/domains/enums in manifest order |
| REST-05 | 08-01, 08-02, 08-03 | COPY FROM for regular and partition child tables | SATISFIED | Wave 3 + copyFromTable helper; handles both regular and partition (FromTable) paths |
| REST-06 | 08-01, 08-02, 08-03 | SETVAL sequence restoration | SATISFIED | Wave 4 in orchestrator.go lines 101-118; runs after Wave 3 COPY FROM |
| REST-07 | 08-01, 08-02, 08-03 | Index creation after data load | SATISFIED | Wave 5 in orchestrator.go lines 121-141; uses IndexDef.Definition strings stored via yamlIndex.definition field |
| REST-08 | 08-01, 08-02, 08-03 | FK constraints as single batch after all tables and data | SATISFIED | Wave 6 in orchestrator.go lines 143-151; all "fk" entries processed after Waves 1-5 |
| REST-09 | 08-01, 08-02, 08-03 | Views/functions/triggers/policies created last | SATISFIED | Wave 7 in orchestrator.go lines 153-178; remainder kinds processed after FK batch |

**Orphaned requirements check:** REST-03 (DROP leak detection) is in REQUIREMENTS.md as unchecked/pending for Phase 9 — correctly not claimed by Phase 8 plans. No orphaned requirements for Phase 8.

**Requirements outside Phase 8 scope (correctly excluded):** REST-03 (Phase 9), REST-10 through REST-13 (Phase 9).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/restore/orchestrator.go` | 113 | SETVAL SQL uses unquoted `'%s.%s'` format for sequence name | Info | Schema and sequence names with special characters or capitals would fail. For the current scope (standard identifiers from pg_catalog), this works. Not a blocker. |

No TODO/FIXME/PLACEHOLDER comments found. No empty implementations found. No stub patterns found in any of the three main files.

### Human Verification Required

#### 1. Full Restore Round-Trip

**Test:** Create a PostgreSQL test database with at least: one table with data (3+ rows), one sequence, one view, and one FK relationship. Run `pgbackup backup --output /tmp/test-backup`, then run `pgbackup restore --input /tmp/test-backup/backup_YYYYMMDD_HHMMSS --pre-backup-dir /tmp/pre-restore` against the same or a clean database.
**Expected:**
- Pre-restore backup created at `/tmp/pre-restore/backup_*/`
- All tables exist and contain the original row count
- Sequences have the correct `last_value` (verify with `SELECT last_value FROM schema.sequence_name`)
- Views are queryable
- FK constraints are present (verify with `\d tablename` or `SELECT * FROM pg_constraint WHERE contype='f'`)
- Indexes are present (verify with `\d tablename`)
**Why human:** Integration tests (TestPreRestoreBackup, TestRestoreCopyFrom, TestRestoreSetval, TestRestoreIndexes, TestRestoreFKBatch, TestRestoreViewsFunctions) skip without TEST_DATABASE_URL. The 7-wave pipeline can only be end-to-end verified with a live PostgreSQL instance.

#### 2. CLI Flag Verification

**Test:** Run `./bin/pgbackup restore --help` after building with `go build -o ./bin/pgbackup ./cmd/pgbackup`.
**Expected:** Help output shows `--input` (marked required) and `--pre-backup-dir` (with default ".") flags.
**Why human:** Cannot invoke the compiled binary in the static analysis environment.

### Gaps Summary

No structural gaps found. All 10 truths verified, all 7 required artifacts exist and are substantive, all 5 key links are wired. Requirements REST-01 through REST-09 are all satisfied by implementation evidence. The human verification items are confirmatory (testing behavior that cannot be verified without a live database), not gap closures.

---

_Verified: 2026-03-12T00:00:00Z_
_Verifier: Claude (gsd-verifier)_
