---
phase: 04-ddl-generation
plan: 03
subsystem: database
tags: [ddl, views, materialized-views, functions, triggers, postgresql, code-generation]

requires:
  - phase: 04-ddl-generation/04-01
    provides: "Core types (ViewDef, MaterializedViewDef, FunctionDef, TriggerDef), DDL generator stubs"
provides:
  - "View DDL generator (CREATE VIEW AS, DROP VIEW)"
  - "Materialized view DDL generator (CREATE MATERIALIZED VIEW, WITH NO DATA, DROP)"
  - "Function DDL generator (passthrough pg_get_functiondef, DROP with arg signature)"
  - "Trigger DDL generator (assembled timing/events/table/function, DROP ON table)"
affects: [04-ddl-generation/04-04, 05-serialization]

tech-stack:
  added: []
  patterns: ["value-receiver type assertion for ObjectDef", "single-statement DDL return pattern"]

key-files:
  created: []
  modified:
    - internal/ddl/view/generator.go
    - internal/ddl/view/generator_test.go
    - internal/ddl/matview/generator.go
    - internal/ddl/matview/generator_test.go
    - internal/ddl/function/generator.go
    - internal/ddl/function/generator_test.go
    - internal/ddl/trigger/generator.go
    - internal/ddl/trigger/generator_test.go

key-decisions:
  - "Value receiver type assertions (core.ViewDef not *core.ViewDef) matching Header() receiver pattern"
  - "Function DDL is pure passthrough of pg_get_functiondef output -- no reconstruction"
  - "Trigger defaults to FOR EACH ROW (TriggerDef lacks row/statement granularity)"
  - "Trigger schema applied to both table and function references (same-schema assumption)"

patterns-established:
  - "DDL generator pattern: type-assert value receiver, return []string{stmt}, descriptive error on wrong type"
  - "MatView IsPopulated flag: append WITH NO DATA only when false"

requirements-completed: [DDLG-03, DDLG-04]

duration: 2min
completed: 2026-03-11
---

# Phase 4 Plan 3: View, MatView, Function, Trigger DDL Generators Summary

**Four DDL generators producing CREATE/DROP SQL for views, materialized views, functions (passthrough), and triggers (assembled from timing/events/table/function)**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-11T11:49:49Z
- **Completed:** 2026-03-11T11:51:54Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- View generator wraps SELECT body in CREATE VIEW schema.name AS
- MatView generator handles IsPopulated flag (WITH NO DATA when false)
- Function generator passes through pg_get_functiondef Definition unchanged
- Trigger generator assembles timing, events (joined with OR), table, and function into CREATE TRIGGER
- All four generators produce correct DROP DDL with schema qualification and CASCADE
- Function DROP includes argument signature for overload disambiguation
- Trigger DROP includes ON schema.tablename for table-scoped naming

## Task Commits

Each task was committed atomically:

1. **Task 1: View and MatView DDL generators**
   - `573e198` (test: failing tests for view and matview)
   - `f83ea63` (feat: implement view and matview generators)
2. **Task 2: Function and Trigger DDL generators**
   - `b85e29a` (test: failing tests for function and trigger)
   - `6f09d46` (feat: implement function and trigger generators)

_TDD tasks have RED (test) then GREEN (feat) commits._

## Files Created/Modified
- `internal/ddl/view/generator.go` - CREATE VIEW AS definition, DROP VIEW IF EXISTS CASCADE
- `internal/ddl/view/generator_test.go` - CREATE, DROP, wrong type tests
- `internal/ddl/matview/generator.go` - CREATE MATERIALIZED VIEW with IsPopulated, DROP
- `internal/ddl/matview/generator_test.go` - Populated, not populated, DROP, wrong type tests
- `internal/ddl/function/generator.go` - Passthrough Definition, DROP with arg signature
- `internal/ddl/function/generator_test.go` - Passthrough, DROP with/without args, wrong type tests
- `internal/ddl/trigger/generator.go` - Assembled CREATE TRIGGER, DROP ON table
- `internal/ddl/trigger/generator_test.go` - Multi-event, single event, DROP, wrong type tests

## Decisions Made
- Value receiver type assertions (core.ViewDef not *core.ViewDef) matching Header() receiver pattern
- Function DDL is pure passthrough -- pg_get_functiondef output is already complete
- Trigger defaults to FOR EACH ROW since TriggerDef lacks row/statement granularity
- Trigger schema applied to both table and function references (same-schema assumption for standard setups)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Four of the DDL generator types are now complete (view, matview, function, trigger)
- Remaining generators (table, sequence, type, domain, enum, policy, foreignkey, compositetype) addressed in other plans
- All generators follow consistent pattern: value-receiver assertion, single-statement return, descriptive errors

---
*Phase: 04-ddl-generation*
*Completed: 2026-03-11*
