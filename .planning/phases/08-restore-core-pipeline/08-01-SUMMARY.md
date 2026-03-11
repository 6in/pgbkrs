---
phase: 08-restore-core-pipeline
plan: "01"
subsystem: database
tags: [go, yaml, restore, manifest, tdd, nyquist]

# Dependency graph
requires:
  - phase: 07-backup-orchestration
    provides: WriteManifest, Manifest struct, backup orchestrator reference for restore pattern
  - phase: 06-dependency-resolution-manifest
    provides: RestoreEntry, ObjectEntry, DAG resolution order
  - phase: 05-serialization
    provides: table.Serializer Serialize/Deserialize, IndexDef, TableDef core types
provides:
  - ReadManifest(path string) (*Manifest, error) in internal/resolve
  - yamlIndex.definition field enabling IndexDef.Definition round-trip through YAML
  - internal/restore package with loader stub and all 8+3 RED test stubs
affects: [08-restore-core-pipeline/08-02, 08-restore-core-pipeline/08-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Nyquist test-first scaffolding: RED stubs written before implementation, integration tests skip on absent TEST_DATABASE_URL"
    - "TDD RED/GREEN for ReadManifest and definition round-trip"
    - "package restore (internal) for unit stubs, package restore_test for integration stubs"

key-files:
  created:
    - internal/restore/loader.go
    - internal/restore/orchestrator_test.go
    - internal/restore/loader_test.go
  modified:
    - internal/resolve/manifest.go
    - internal/serialize/table/serializer.go
    - internal/resolve/manifest_test.go
    - internal/serialize/table/serializer_test.go

key-decisions:
  - "ReadManifest added alongside WriteManifest in manifest.go as symmetric read counterpart"
  - "yamlIndex.definition uses omitempty since empty Definition is valid for older YAML files without the field"
  - "loader_test.go uses package restore (not restore_test) to allow testing of unexported functions like defYAMLPath"
  - "orchestrator_test.go uses package restore_test (external) matching the integration test pattern from backup package"

patterns-established:
  - "Restore package test pattern: integration tests in restore_test package with requireDB() skip guard, unit tests in restore package for unexported functions"
  - "ReadManifest/WriteManifest are symmetric: same YAML library, same error wrapping pattern"

requirements-completed: [REST-01, REST-02, REST-04, REST-05, REST-06, REST-07, REST-08, REST-09]

# Metrics
duration: 3min
completed: 2026-03-12
---

# Phase 08 Plan 01: Restore Core Pipeline Wave 0 Summary

**ReadManifest function and yamlIndex definition field added, plus all 11 RED test stubs for the restore package (8 integration + 3 unit)**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-03-11T15:23:05Z
- **Completed:** 2026-03-11T15:26:00Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Added ReadManifest to internal/resolve/manifest.go enabling the restore orchestrator to load _manifest.yaml
- Extended yamlIndex with a definition field so IndexDef.Definition survives Serialize/Deserialize round-trips
- Created internal/restore package with loader stub and 11 RED test stubs covering all 8 REST requirements

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: Add failing tests for ReadManifest and index definition** - `ca660c4` (test)
2. **Task 1 GREEN: Add ReadManifest and yamlIndex definition field** - `7b8c129` (feat)
3. **Task 2: Create internal/restore package with RED test stubs** - `08ea3a5` (test)

## Files Created/Modified
- `internal/resolve/manifest.go` - Added ReadManifest function (symmetric counterpart to WriteManifest)
- `internal/serialize/table/serializer.go` - Added definition field to yamlIndex, updated toYAML/fromYAML loops
- `internal/resolve/manifest_test.go` - Added TestReadManifest_NonexistentPath and TestReadManifest_ValidFile
- `internal/serialize/table/serializer_test.go` - Added TestIndexDefinitionRoundTrip
- `internal/restore/loader.go` - Minimal package stub with defYAMLPath placeholder
- `internal/restore/orchestrator_test.go` - 8 integration test stubs with requireDB() skip guard
- `internal/restore/loader_test.go` - 3 unit test stubs for unexported functions

## Decisions Made
- `yamlIndex.definition` uses `omitempty` since Definition is empty string for indexes without DDL passthrough in older backup files
- Loader test uses `package restore` (internal) to enable testing of unexported `defYAMLPath` in future plans
- Orchestrator test uses `package restore_test` (external) matching the established pattern from `internal/backup`

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- internal/restore package scaffolded and ready for 08-02 loader implementation
- All RED tests in place; 08-02 will implement defYAMLPath, loadObjectDef, and make unit stubs GREEN
- ReadManifest available for orchestrator_test.go to use in integration tests
- IndexDef.Definition round-trip now works, unblocking REST-07 (index recreation from stored DDL)

---
*Phase: 08-restore-core-pipeline*
*Completed: 2026-03-12*
