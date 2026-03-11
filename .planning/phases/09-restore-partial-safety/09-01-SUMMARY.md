---
phase: 09-restore-partial-safety
plan: "01"
subsystem: database
tags: [go, restore, bfs, filter, leak-detection, logging, options-struct]

# Dependency graph
requires:
  - phase: 08-restore-core-pipeline
    provides: RunRestore pipeline, orchestrator_test.go integration tests, loader.go
provides:
  - "BFS transitive closure engine (buildTransitiveClosure) with FK post-pass"
  - "filteredRestoreOrder: schema/object/full scoped restore order filtering"
  - "verifyDepsPresent: closure dependency validation"
  - "liveObject type with snapshotLiveObjects and detectLeaks"
  - "Logger struct with drop.log/restore.log/summary.log writers"
  - "Options struct replacing positional RunRestore parameters"
  - "RunRestore(ctx, conn, Options) new signature"
affects:
  - 09-02
  - 09-03

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Options struct instead of positional string args for extensible function signatures"
    - "BFS with visited guard for cycle-safe transitive closure"
    - "Post-BFS FK pass: add FK entries only when all deps are in closure"
    - "Leak detection: compare post-drop snapshot against shouldBeDropped set"
    - "Logger deferred close with WriteSummary at RunRestore exit"

key-files:
  created:
    - internal/restore/filter.go
    - internal/restore/filter_test.go
    - internal/restore/leak.go
    - internal/restore/leak_test.go
    - internal/restore/logger.go
  modified:
    - internal/restore/orchestrator.go
    - internal/restore/orchestrator_test.go
    - cmd/pgbackup/cmd/restore.go

key-decisions:
  - "Options struct defined in filter.go (not orchestrator.go) as it owns the filter types; orchestrator imports it"
  - "snapshotLiveObjects failure is non-fatal: log to stderr and skip leak detection rather than abort restore"
  - "Logger creation failure is non-fatal: log to stderr and proceed without file logging"
  - "FK post-BFS pass uses len(obj.DependsOn) > 0 guard to exclude FK entries with no deps from empty closure"
  - "TestDetectLeaks_NoLeaks uses empty shouldBeDropped map (not identical preDrop/postDrop sets)"

patterns-established:
  - "Wave infrastructure pattern: stub integration tests with t.Skip in Wave 0, fill in Plan 02"
  - "filteredRestoreOrder as single dispatch point for all scope variants (full/schema/object)"

requirements-completed: [REST-03, REST-10, REST-11, REST-12, REST-13]

# Metrics
duration: 25min
completed: 2026-03-12
---

# Phase 9 Plan 01: Restore Partial Safety Wave 0 Infrastructure Summary

**BFS closure engine, leak detection, structured logger, and Options-based RunRestore signature enabling scoped partial restore for Phase 9**

## Performance

- **Duration:** 25 min
- **Started:** 2026-03-12T00:00:00Z
- **Completed:** 2026-03-12T00:25:00Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- filter.go: BFS transitive closure with cycle guard and FK post-pass; filteredRestoreOrder dispatches schema/object/full scoping; verifyDepsPresent validates closure completeness
- leak.go: liveObject type, snapshotLiveObjects pg_catalog UNION ALL query, detectLeaks comparing postDrop against shouldBeDropped set
- logger.go: Logger with three log files (drop.log, restore.log, summary.log), drop/restore/leak-warn methods, deferred WriteSummary
- orchestrator.go: RunRestore migrated to Options struct; integrates all new components as stubs; filteredRestoreOrder replaces hard-coded manifest.RestoreOrder
- orchestrator_test.go: all Phase 8 integration tests updated to Options{} syntax; REST-03/10/11/13 stub tests added with t.Skip
- cmd/restore.go: updated caller with placeholder Schema/Object/LogDir variables (flag wiring in Plan 03)
- 12 unit tests all pass; `go build ./...` and `go vet ./...` clean

## Task Commits

Each task was committed atomically:

1. **Task 1: filter.go + filter_test.go** - `0c0e857` (feat)
2. **Task 2: leak.go + logger.go + orchestrator refactor + caller update** - `e140da3` (feat)

## Files Created/Modified
- `/home/parallels/workspaces/pgbkrs/internal/restore/filter.go` - Options struct, buildTransitiveClosure, verifyDepsPresent, filteredRestoreOrder
- `/home/parallels/workspaces/pgbkrs/internal/restore/filter_test.go` - 9 unit tests for BFS closure and filtering
- `/home/parallels/workspaces/pgbkrs/internal/restore/leak.go` - liveObject type, snapshotLiveObjects, detectLeaks
- `/home/parallels/workspaces/pgbkrs/internal/restore/leak_test.go` - 3 unit tests for detectLeaks
- `/home/parallels/workspaces/pgbkrs/internal/restore/logger.go` - Logger struct with 3 log file writers
- `/home/parallels/workspaces/pgbkrs/internal/restore/orchestrator.go` - RunRestore(ctx, conn, Options); integrated filters/leak/logger
- `/home/parallels/workspaces/pgbkrs/internal/restore/orchestrator_test.go` - updated to Options{}; stub tests for Plan 02
- `/home/parallels/workspaces/pgbkrs/cmd/pgbackup/cmd/restore.go` - Options construction from flags with placeholders

## Decisions Made
- Options struct lives in filter.go (it's the filter type), not orchestrator.go as the plan suggested; orchestrator imports it naturally — cleaner separation
- snapshotLiveObjects failure is non-fatal: log stderr and skip leak detection rather than aborting restore (graceful degradation for environments where pg_catalog queries have restricted access)
- Logger creation failure is non-fatal: same graceful degradation approach
- FK post-BFS pass uses `len(obj.DependsOn) > 0` guard to prevent FK entries with no deps from being vacuously included in empty closures
- TestDetectLeaks_NoLeaks required test correction: "no leaks" means shouldBeDropped is empty (objects in postDrop that aren't targeted for drop are not leaks); original draft had both objects in shouldBeDropped which correctly yields 2 leaks

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Options struct placement moved to filter.go instead of orchestrator.go**
- **Found during:** Task 1 (filter.go creation)
- **Issue:** Plan spec placed Options struct in orchestrator.go but filter.go needs it as a parameter type; placing in orchestrator would create a circular dependency pattern since filter.go is also in package restore
- **Fix:** Defined Options in filter.go where it is first used; orchestrator.go uses it directly (same package)
- **Files modified:** internal/restore/filter.go
- **Verification:** go build ./... passes; both files in package restore have no import cycle
- **Committed in:** 0c0e857 (Task 1 commit)

**2. [Rule 1 - Bug] TestDetectLeaks_NoLeaks test correction**
- **Found during:** Task 2 (leak_test.go TDD RED→GREEN)
- **Issue:** Test initially had both objects in shouldBeDropped but expected 0 leaks — semantically wrong; objects present in postDrop AND in shouldBeDropped ARE leaks by definition
- **Fix:** Changed test to use empty shouldBeDropped map: no objects were targeted for drop, so nothing remaining is a leak
- **Files modified:** internal/restore/leak_test.go
- **Verification:** TestDetectLeaks_NoLeaks passes with empty shouldBeDropped; TestDetectLeaks_WithLeak/PreExisting verify positive cases
- **Committed in:** e140da3 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 - Bug)
**Impact on plan:** Both fixes corrected logical issues before they could propagate. No scope creep.

## Issues Encountered
None beyond the deviations documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Wave 0 infrastructure complete: filter.go, leak.go, logger.go all compilable and unit-tested
- RunRestore uses Options struct; all Phase 8 integration tests updated and compiling
- Plan 02 can implement full integration wiring for REST-03, REST-10, REST-11, REST-13
- Plan 03 can wire --schema, --object, --log-dir cobra flags into restore.go

---
*Phase: 09-restore-partial-safety*
*Completed: 2026-03-12*
