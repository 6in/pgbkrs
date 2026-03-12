---
phase: 10-schema-diff
plan: "01"
subsystem: database
tags: [schema-diff, tdd, go, testing]

# Dependency graph
requires:
  - phase: 09-restore-partial-safety
    provides: restore pipeline complete; all internal/core types fully defined
  - phase: 06-dependency-resolution-manifest
    provides: resolve.Manifest and ReadManifest used by loadBackup stub
provides:
  - internal/diff package skeleton with Run() entry point
  - loadBackup/compareObjects/formatReport stubs
  - DiffResult and ObjectChange types
  - Five RED test contracts covering DIFF-01 through DIFF-05
affects: [10-02-schema-diff-implementation]

# Tech tracking
tech-stack:
  added: []
  patterns: [Nyquist TDD Wave 0 — stubs written before implementation so tests define contract]

key-files:
  created:
    - internal/diff/diff.go
    - internal/diff/loader.go
    - internal/diff/compare.go
    - internal/diff/report.go
    - internal/diff/diff_test.go
  modified: []

key-decisions:
  - "White-box test package (package diff) for access to unexported functions loadBackup/compareObjects/formatReport"
  - "Test keys use schema.kind.name format consistent with internal/diff keying convention"
  - "TestLoadBackup uses t.TempDir() with minimal manifest.yaml for valid-dir branch"

patterns-established:
  - "Nyquist Wave 0: all five test functions written against stubs; stubs return zero values causing immediate RED"
  - "DiffResult.Changed carries []string Details for human-readable per-object change lines"

requirements-completed: [DIFF-01, DIFF-02, DIFF-03, DIFF-04, DIFF-05]

# Metrics
duration: 2min
completed: 2026-03-12
---

# Phase 10 Plan 01: Schema Diff Package Skeleton Summary

**internal/diff package scaffolded with five failing TDD contracts covering load, compare (added/removed/table/non-table), and format requirements**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-12T08:19:47Z
- **Completed:** 2026-03-12T08:21:40Z
- **Tasks:** 1
- **Files modified:** 5

## Accomplishments

- Created `internal/diff/` directory with four production stub files (`diff.go`, `loader.go`, `compare.go`, `report.go`)
- Defined `DiffResult` and `ObjectChange` types in `compare.go`
- Wrote `diff_test.go` with all five required test functions (TestLoadBackup, TestDiffAddedRemoved, TestDiffTable, TestDiffNonTable, TestFormatReport)
- Package compiles cleanly (`go build ./internal/diff/...` exits 0); `go test -short ./internal/diff/...` shows FAIL as required at Wave 0
- No regressions in other packages (`go build ./...` exits 0)

## Task Commits

Each task was committed atomically:

1. **Task 1: Write test stubs for all five DIFF requirements (Wave 0)** - `23de377` (test)

## Files Created/Modified

- `internal/diff/diff.go` - Run() entry point stub accepting (backupA, backupB string, w io.Writer) returning nil
- `internal/diff/loader.go` - loadBackup stub returning empty map[string]core.ObjectDef
- `internal/diff/compare.go` - DiffResult/ObjectChange type definitions + compareObjects stub returning DiffResult{}
- `internal/diff/report.go` - formatReport stub writing nothing
- `internal/diff/diff_test.go` - Five RED test functions covering all DIFF-01 through DIFF-05 requirements

## Decisions Made

- White-box test package (`package diff`) chosen to access unexported functions `loadBackup`, `compareObjects`, `formatReport` — consistent with other internal packages in this project
- Test keys use `schema.kind.name` format (e.g., `"public.table.users"`) matching the keying convention documented in the plan
- `TestLoadBackup` covers two branches: non-existent dir (expects error) and valid dir with minimal manifest.yaml (expects empty non-nil map)
- `TestDiffTable` checks for `"カラム追加"` in detail lines to pin the Japanese terminology required by spec 9.3

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/diff` package skeleton is ready for Plan 02 implementation
- All five test contracts are locked in RED state; Plan 02 turns them GREEN by implementing `loadBackup`, `compareObjects`, and `formatReport`
- No blockers

---
*Phase: 10-schema-diff*
*Completed: 2026-03-12*

## Self-Check: PASSED

- internal/diff/diff.go: FOUND
- internal/diff/loader.go: FOUND
- internal/diff/compare.go: FOUND
- internal/diff/report.go: FOUND
- internal/diff/diff_test.go: FOUND
- .planning/phases/10-schema-diff/10-01-SUMMARY.md: FOUND
- commit 23de377: FOUND
