---
phase: 07-backup-orchestration
plan: 01
subsystem: backup
tags: [go, postgresql, backup, orchestrator, csv-export, yaml-serialization, partitions]

# Dependency graph
requires:
  - phase: 06-dependency-resolution-manifest
    provides: resolve.BuildManifest, resolve.WriteManifest, resolve.SkipEntry for manifest assembly
  - phase: 05-serialization
    provides: per-kind Serializer implementations used for YAML output
  - phase: 04-ddl-generation
    provides: all ObjectKind types and core interfaces
  - phase: 02-schema-fetch-tables-sequences
    provides: fetch/table, fetch/sequence SchemaFetcher implementations
  - phase: 03-schema-fetch-views-functions-triggers-policies
    provides: remaining fetch package SchemaFetcher implementations
provides:
  - RunBackup() entry point orchestrating full backup of all user schemas
  - core.TableDataMeta struct for CSV export metadata (import-cycle-safe mirror of export.DataMeta)
  - TableDef.DataMeta field enabling def.yaml data: block emission
  - internal/backup/skip.go with skipReason() helper for bytea/xml/pg_lsn/txid_snapshot detection
  - Directory layout implementation: spec section 4 partitioned table routing
affects: [08-cli-wiring, 09-integration-tests, 10-snapshot-mode]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "kindToDir map drives spec section 4 directory layout for all object types"
    - "childOf map built from Partitioning.Children enables O(1) partition routing"
    - "TableDataMeta defined in core (not export) to avoid import cycles"
    - "serializerFor() switch function returns correct Serializer per kind"
    - "skipReason() uses strings.Contains to catch type array variants like bytea[]"

key-files:
  created:
    - internal/backup/skip.go
    - internal/backup/orchestrator.go
    - internal/backup/orchestrator_test.go
  modified:
    - internal/core/types.go
    - internal/serialize/table/serializer.go

key-decisions:
  - "TableDataMeta defined in core (not export) to break potential import cycle: export imports core, orchestrator imports both"
  - "First-pass table fetch + childOf map pattern avoids N passes over all objects to route partition children"
  - "skipReason() internal (not exported) since tests are in same package (package backup)"
  - "Pointer receiver (&fetchTable.SchemaFetcher{}) required — compile-time check var _ SchemaFetcher = (*SchemaFetcher)(nil) forces pointer receiver"
  - "ForeignKey objects tracked in allObjects for manifest but not written as standalone YAML files per spec"

patterns-established:
  - "backup package is same-package with skip.go: use package backup (not backup_test) for testing internal helpers"
  - "Integration test stubs use t.Skip('integration: requires live DB') for non-short-mode gating"

requirements-completed: [BKUP-01, BKUP-02]

# Metrics
duration: 15min
completed: 2026-03-11
---

# Phase 7 Plan 01: Backup Orchestration Core Summary

**RunBackup() orchestrator with spec section 4 directory layout, partition child routing, skip detection for 4 unsafe column types, and DataMeta injection enabling data: block in def.yaml**

## Performance

- **Duration:** 15 min
- **Started:** 2026-03-11T14:52:13Z
- **Completed:** 2026-03-11T15:07:00Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- skip.go with skipReason() detecting bytea, bytea[], xml, pg_lsn, txid_snapshot using strings.Contains
- core.TableDataMeta struct added to break import cycle; TableDef.DataMeta field enables serializer data: block
- Table serializer updated to emit data: block from DataMeta when non-nil (no change to existing tests)
- RunBackup() with schema discovery, per-kind fetching, partition routing, CSV export, manifest assembly

## Task Commits

Each task was committed atomically:

1. **Task 1: Test stubs (Wave 0) + skip detection helper** - `1209733` (test)
2. **Task 2: DataMeta injection + serializer update + RunBackup orchestrator** - `fe7f0f8` (feat)

## Files Created/Modified
- `internal/backup/skip.go` - skipReason() helper with 4 skip types and array variant detection
- `internal/backup/orchestrator.go` - RunBackup() entry point, discoverSchemas, buildChildOf, tableObjectDir, columnNames, serializerFor helpers
- `internal/backup/orchestrator_test.go` - TestSkipUnsupportedColumns (6 cases), TestSkipWarning, 3 integration stubs
- `internal/core/types.go` - Added TableDataMeta struct and TableDef.DataMeta field
- `internal/serialize/table/serializer.go` - toYAML() now emits data: block when DataMeta non-nil

## Decisions Made
- TableDataMeta defined in core (not export) to avoid import cycles: export already imports core, so orchestrator importing both would create a cycle if TableDataMeta lived in export
- Pointer receiver required for fetchTable.SchemaFetcher — compile-time interface check in fetch/table uses pointer receiver, so `(&fetchTable.SchemaFetcher{})` not `fetchTable.SchemaFetcher{}` for the first-pass call
- ForeignKey objects tracked for manifest but not written as YAML files — FK definition is embedded in table DDL and has no standalone spec section 5 file format

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed pointer receiver on fetchTable.SchemaFetcher first-pass call**
- **Found during:** Task 2 (orchestrator compilation)
- **Issue:** Plan pseudocode used `fetchTable.SchemaFetcher{}.Fetch(...)` (value receiver) but SchemaFetcher implements the interface via pointer receiver
- **Fix:** Changed to `(&fetchTable.SchemaFetcher{}).Fetch(...)`
- **Files modified:** internal/backup/orchestrator.go
- **Verification:** `go build ./internal/backup/...` exits 0
- **Committed in:** fe7f0f8 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - bug fix)
**Impact on plan:** Necessary correctness fix, no scope change.

## Issues Encountered
- None beyond the pointer receiver fix above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- RunBackup() compiles and unit tests pass; ready for Plan 02 (snapshot mode, RepeatableRead transaction wrapping)
- CLI wiring (Plan 08) can now call RunBackup(ctx, conn, outDir, snapshot)
- Integration tests (Plan 09) can exercise RunBackup against a live PostgreSQL instance

---
*Phase: 07-backup-orchestration*
*Completed: 2026-03-11*
