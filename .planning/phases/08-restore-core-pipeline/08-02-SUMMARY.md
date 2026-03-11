---
phase: 08-restore-core-pipeline
plan: "02"
subsystem: database
tags: [go, restore, pipeline, tdd, loader, orchestrator]

# Dependency graph
requires:
  - phase: 08-01
    provides: internal/restore package stubs, ReadManifest, IndexDef.Definition round-trip
  - phase: 07-backup-orchestration
    provides: RunBackup (called for pre-restore safety backup)
  - phase: 06-dependency-resolution-manifest
    provides: ReadManifest, RestoreEntry, Manifest struct
  - phase: 05-serialization
    provides: all Serializer implementations for 10 object kinds
  - phase: 04-ddl-generation
    provides: all DDLGenerator implementations for 11 object kinds
provides:
  - defYAMLPath(backupDir string, entry RestoreEntry) string
  - serializerFor(kind string) core.Serializer
  - ddlGeneratorFor(kind string) core.DDLGenerator
  - loadObjectDef(backupDir string, entry RestoreEntry) (core.ObjectDef, error)
  - RunRestore(ctx, conn, backupDir, preBackupDir string) error — full 7-wave restore pipeline
affects: [08-restore-core-pipeline/08-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "7-wave restore pipeline: pre-backup, DROP reverse, CREATE schema, COPY FROM, SETVAL, indexes, FK batch, remainder"
    - "FK drop uses minimal ForeignKeyDef (no def.yaml for FKs)"
    - "FK create re-fetches from live DB via fetchForeignkey.SchemaFetcher post-table-creation"
    - "Table DDL wave executes only stmts[0] (CREATE TABLE); indexes deferred to Wave 5"
    - "SETVAL runs after COPY FROM so sequence values override any serial column behavior"
    - "manifest 'fk' string handled explicitly in ddlGeneratorFor (differs from core.KindForeignKey)"

key-files:
  created:
    - internal/restore/orchestrator.go
  modified:
    - internal/restore/loader.go
    - internal/restore/loader_test.go
    - internal/restore/orchestrator_test.go

key-decisions:
  - "FK create wave re-fetches FKs from live DB rather than deserializing from disk — FKs have no def.yaml"
  - "manifest 'fk' string handled with explicit check before ObjectKind switch in ddlGeneratorFor"
  - "Table CREATE wave executes only first DDL statement to defer index creation to Wave 5"
  - "SETVAL executed after COPY FROM to ensure sequence values match backed-up state"

requirements-completed: [REST-01, REST-02, REST-04, REST-05, REST-06, REST-07, REST-08, REST-09]

# Metrics
duration: 8min
completed: 2026-03-12
---

# Phase 08 Plan 02: Restore Core Pipeline Summary

**Full 7-wave restore pipeline: loader helpers (defYAMLPath, serializerFor, ddlGeneratorFor, loadObjectDef) plus RunRestore orchestrator with pre-backup, DROP, CREATE, COPY FROM, SETVAL, indexes, FK batch, and remainder waves**

## Performance

- **Duration:** ~8 min
- **Completed:** 2026-03-12
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Implemented `internal/restore/loader.go` with all 4 helper functions covering all 11 object kinds
- Implemented `internal/restore/orchestrator.go` with `RunRestore` executing all 7 pipeline waves
- All unit tests GREEN: TestDefYAMLPath, TestSerializerFor, TestDDLGeneratorFor, TestDeserializeRoundTrip, TestReadManifestRoundTrip
- All integration tests skip without TEST_DATABASE_URL (not fail)
- Full project build clean: `go build ./...` passes

## Task Commits

1. **Task 1: loader.go implementation** - `53604cf` (feat)
2. **Task 2: orchestrator.go + RunRestore** - `2ab6ca5` (feat)

## Files Created/Modified

- `internal/restore/loader.go` — defYAMLPath, serializerFor, ddlGeneratorFor, loadObjectDef
- `internal/restore/orchestrator.go` — RunRestore with all 7 waves + execDrop, copyFromTable, execFKCreate helpers
- `internal/restore/loader_test.go` — Unit tests for all loader functions
- `internal/restore/orchestrator_test.go` — Integration test stubs with real test logic (skip without DB)

## Decisions Made

- FK create wave re-fetches FKs from live DB via `fetchForeignkey.SchemaFetcher` rather than from disk — FKs have no def.yaml in the backup directory
- The manifest uses `"fk"` as the Kind string for FK entries, but `core.KindForeignKey = "foreign_key"`. An explicit `kind == "fk"` check in `ddlGeneratorFor` handles this mapping correctly
- Table CREATE wave executes only `stmts[0]` (the CREATE TABLE statement); index DDL is handled separately in Wave 5 to ensure indexes are created after data is loaded
- SETVAL runs after COPY FROM (Wave 4 after Wave 3) so that sequence last_value matches the backed-up state rather than being reset by serial column triggers

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None — no external service configuration required for this plan.

## Self-Check

- [x] `internal/restore/loader.go` exists and implements defYAMLPath, serializerFor, ddlGeneratorFor, loadObjectDef
- [x] `internal/restore/orchestrator.go` exists and exports RunRestore
- [x] `go build ./...` passes
- [x] `go test ./internal/restore/... -short` passes (unit GREEN, integration SKIP)
- [x] Commit 53604cf exists (loader.go)
- [x] Commit 2ab6ca5 exists (orchestrator.go)

## Self-Check: PASSED

## Next Phase Readiness

- `RunRestore` is fully implemented and ready for CLI integration in 08-03
- All REST-01 through REST-09 requirements have implementation (integration tests will verify with live DB)
- The 7-wave pipeline handles all object kinds and edge cases (partitions, FKs, sequences)

---
*Phase: 08-restore-core-pipeline*
*Completed: 2026-03-12*
