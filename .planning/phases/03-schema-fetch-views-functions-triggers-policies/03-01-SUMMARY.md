---
phase: 03-schema-fetch-views-functions-triggers-policies
plan: "01"
subsystem: database
tags: [postgres, views, matviews, functions, triggers, policies, types, testing]

# Dependency graph
requires:
  - phase: 02-schema-fetch-tables-sequences
    provides: "ObjectHeader, ObjectDef interface, testhelpers/setup.go, existing *Def stubs"
provides:
  - "Expanded ViewDef, MaterializedViewDef, FunctionDef, TriggerDef, PolicyDef structs"
  - "Phase 3 test fixtures in testhelpers/setup.go"
  - "Integration test stubs for all five Phase 3 fetcher packages"
affects: [03-02, 03-03, 03-04, 04-serialization]

# Tech tracking
tech-stack:
  added: []
  patterns: [Nyquist test-before-implementation, separate DDL blocks in setup.go, positional fmt.Sprintf for schema vars]

key-files:
  created:
    - internal/core/types_phase3_test.go
    - internal/fetch/view/fetcher_test.go
    - internal/fetch/matview/fetcher_test.go
    - internal/fetch/function/fetcher_test.go
    - internal/fetch/trigger/fetcher_test.go
    - internal/fetch/policy/fetcher_test.go
  modified:
    - internal/core/types.go
    - internal/fetch/testhelpers/setup.go

key-decisions:
  - "Phase 3 DDL added as separate fmt.Sprintf block (ddl2) to avoid positional argument numbering conflicts with existing Phase 2 DDL"
  - "TDD RED/GREEN for struct expansion: types_phase3_test.go verifies all new fields compile and are accessible"

patterns-established:
  - "Separate DDL blocks per phase in testhelpers/setup.go using positional fmt.Sprintf(%[1]s)"
  - "Nyquist: test stubs created before implementation packages exist (Wave 2 creates SchemaFetcher)"

requirements-completed: [FETCH-02, FETCH-03, FETCH-04, FETCH-05, FETCH-10]

# Metrics
duration: 3min
completed: 2026-03-11
---

# Phase 3 Plan 01: Types + Test Stubs Summary

**Expanded five *Def structs (View, MatView, Function, Trigger, Policy) with full fields and created 8 integration test stubs with Phase 3 fixtures**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-11T08:35:59Z
- **Completed:** 2026-03-11T08:38:56Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Expanded ViewDef (Owner), MaterializedViewDef (Owner, IsPopulated), FunctionDef (ArgTypes, ReturnType), TriggerDef (Timing, Events, TableName, FunctionName), PolicyDef (TableName, Command, Roles, Using, WithCheck)
- Extended testhelpers/setup.go with view, materialized view, function, audit_log table, trigger function, trigger, and RLS policy fixtures
- Created five test stub files with 8 test functions covering all Phase 3 fetcher contracts

## Task Commits

Each task was committed atomically:

1. **Task 1: Expand *Def structs in types.go** - `ebf3a80` (feat)
2. **Task 2: Extend testhelpers and create test stubs** - `b7dd6bf` (test)

## Files Created/Modified
- `internal/core/types.go` - Expanded five stub structs with full Phase 3 fields
- `internal/core/types_phase3_test.go` - TDD test verifying all new struct fields
- `internal/fetch/testhelpers/setup.go` - Phase 3 DDL fixtures (view, matview, function, trigger, policy)
- `internal/fetch/view/fetcher_test.go` - TestFetchView integration stub
- `internal/fetch/matview/fetcher_test.go` - TestFetchMatView + TestFetchMatViewUnpopulated stubs
- `internal/fetch/function/fetcher_test.go` - TestFetchFunction stub with trigger function check
- `internal/fetch/trigger/fetcher_test.go` - TestFetchTrigger + TestFetchTriggerExcludesInternal stubs
- `internal/fetch/policy/fetcher_test.go` - TestFetchPolicy + TestFetchPolicyPublicRoles stubs

## Decisions Made
- Phase 3 DDL added as separate `fmt.Sprintf` block (`ddl2`) with `%[1]s` positional args to avoid numbering conflicts with existing Phase 2 DDL
- TDD approach: created `types_phase3_test.go` that fails to compile before struct expansion (RED), then passes after (GREEN)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All five expanded *Def structs ready for Wave 2 fetcher implementations (Plans 02, 03, 04)
- Test stubs provide concrete test contracts for each SchemaFetcher
- testhelpers/setup.go creates all required Phase 3 fixtures

---
*Phase: 03-schema-fetch-views-functions-triggers-policies*
*Completed: 2026-03-11*
