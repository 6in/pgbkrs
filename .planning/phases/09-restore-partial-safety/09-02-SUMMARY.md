---
phase: 09-restore-partial-safety
plan: "02"
subsystem: database
tags: [go, restore, bfs, filter, leak-detection, logging, integration-tests, tdd]

# Dependency graph
requires:
  - phase: 09-restore-partial-safety
    plan: "01"
    provides: "filter.go, leak.go, logger.go, Options struct, RunRestore with full Phase 9 wiring"
provides:
  - "Full integration test implementations for REST-03, REST-10, REST-11, REST-12, REST-13"
  - "TestDropLeakDetection: verifies drop.log entries after restore run"
  - "TestSchemaScoped: verifies schema-scoped restore excludes other schemas"
  - "TestObjectScopedTransitive: verifies BFS transitive dep pull-in"
  - "TestMissingDependencyError: unit test verifying error message for missing dep"
  - "TestLogFiles: verifies all three log files created on every restore run"
  - "export_test.go: FilteredRestoreOrder exported alias for black-box testing"
affects:
  - 09-03

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "export_test.go pattern: unexported functions exposed via exported alias in package-internal test file"
    - "Integration test structure: requireDB(t) guard + t.Cleanup for schema teardown"
    - "TestMissingDependencyError as pure unit test within integration test file — no requireDB"

key-files:
  created:
    - internal/restore/export_test.go
  modified:
    - internal/restore/orchestrator_test.go

key-decisions:
  - "export_test.go used to expose filteredRestoreOrder for external test package (restore_test) instead of exporting it directly — maintains internal API surface while enabling black-box unit testing"
  - "TestDropLeakDetection verifies drop.log contains object entries rather than simulating actual leaks — DROP IF EXISTS CASCADE prevents realistic leak scenarios without mocking; log presence and content verified as proxy"
  - "TestMissingDependencyError constructs a minimal resolve.Manifest directly to avoid any DB dependency — fastest possible unit test for REST-12"

patterns-established:
  - "export_test.go: standard Go pattern for white-box testing of unexported functions from external test packages"

requirements-completed: [REST-03, REST-10, REST-11, REST-12, REST-13]

# Metrics
duration: 20min
completed: 2026-03-12
---

# Phase 9 Plan 02: Restore Partial Safety Integration Tests Summary

**Full integration and unit test implementations for REST-03/10/11/12/13 — schema-scoped restore, object-scoped transitive BFS, missing-dep error, and structured log file verification**

## Performance

- **Duration:** 20 min
- **Started:** 2026-03-12T00:30:00Z
- **Completed:** 2026-03-12T00:50:00Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Task 1: Confirmed orchestrator.go fully implements all Phase 9 wiring (filter + leak + logger integrated in all 7 pipeline waves) — delivered in Plan 01, verified here
- Task 2: Replaced all t.Skip stubs in orchestrator_test.go with full implementations for 5 Phase 9 tests; added export_test.go to expose filteredRestoreOrder
- TestMissingDependencyError passes as green unit test without any DB connection
- All 4 integration tests (TestDropLeakDetection, TestSchemaScoped, TestObjectScopedTransitive, TestLogFiles) compile and skip cleanly without TEST_DATABASE_URL

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire filter + leak + logger into RunRestore** - `e140da3` (feat, committed in Plan 01 — verified unchanged)
2. **Task 2: Integration tests for REST-03/10/11/12/13** - `904c65b` (test)

## Files Created/Modified
- `/home/parallels/workspaces/pgbkrs/internal/restore/orchestrator_test.go` - Full implementations for TestDropLeakDetection, TestSchemaScoped, TestObjectScopedTransitive, TestMissingDependencyError, TestLogFiles
- `/home/parallels/workspaces/pgbkrs/internal/restore/export_test.go` - FilteredRestoreOrder exported alias for black-box unit testing of filteredRestoreOrder

## Decisions Made
- export_test.go approach chosen over exporting filteredRestoreOrder directly — keeps the internal API unexported while enabling TestMissingDependencyError in the external test package
- TestDropLeakDetection verifies log content (drop.log has entries for backed-up objects) rather than forcing actual leaks — DROP IF EXISTS CASCADE always succeeds, making genuine leak simulation impossible without connection mocking
- TestSchemaScoped creates full s1/s2 schema setup to verify scoping excludes s2 after restore with Schema:"s1"

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added export_test.go for black-box unit testing**
- **Found during:** Task 2 (TestMissingDependencyError)
- **Issue:** TestMissingDependencyError is in package restore_test (external), but filteredRestoreOrder is unexported — direct call would fail to compile
- **Fix:** Created export_test.go in package restore with exported FilteredRestoreOrder alias; this is the standard Go white-box testing pattern
- **Files modified:** internal/restore/export_test.go (created)
- **Verification:** go build ./... passes; TestMissingDependencyError passes with restore.FilteredRestoreOrder call
- **Committed in:** 904c65b (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 2 - missing test infrastructure)
**Impact on plan:** Required for TestMissingDependencyError to compile in external test package. No scope creep.

## Issues Encountered
- Task 1 was already fully implemented in Plan 01 commit e140da3 — no changes to orchestrator.go were needed. The task was verified as complete and its commit hash documented.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All Phase 9 REST-03/10/11/12/13 requirements have tests that compile and skip cleanly
- Plan 03 can wire --schema, --object, --log-dir cobra flags into cmd/restore.go
- Integration tests will run and verify behavior when TEST_DATABASE_URL is provided

---
*Phase: 09-restore-partial-safety*
*Completed: 2026-03-12*
