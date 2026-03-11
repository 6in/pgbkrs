---
phase: 06-dependency-resolution-manifest
plan: 01
subsystem: database
tags: [dag, topological-sort, kahn, dependency-resolution, postgresql]

requires:
  - phase: 01-foundation
    provides: core types package with ObjectDef interface and all *Def types
  - phase: 04-ddl-generation
    provides: ForeignKeyDef as independent ObjectDef with embedded ObjectHeader
provides:
  - DAG builder with dependency extraction for all 10 non-FK object kinds
  - Kahn's topological sort with kind-priority deterministic tiebreaking
  - BuildRestoreOrder function with FK isolation
  - Cycle detection with participant reporting
affects: [06-02, 07-backup-orchestration, 08-restore-orchestration]

tech-stack:
  added: []
  patterns: [two-pass-dag-resolution, kind-priority-tiebreaking, fk-isolation]

key-files:
  created:
    - internal/resolve/dag.go
    - internal/resolve/dag_test.go
    - internal/resolve/kahn.go
    - internal/resolve/kahn_test.go
  modified: []

key-decisions:
  - "Two-pass DAG: AddObject registers nodes/customTypes, Resolve builds edges -- avoids ordering issues"
  - "Kind-priority tiebreaking in Kahn's sort instead of explicit view->table edges -- simpler and more robust"
  - "inDegree tracks dependency count per node (how many things this node depends on), zero means ready to restore"
  - "FKs sorted alphabetically by schema.name when appended to ensure deterministic output"

patterns-established:
  - "Two-pass resolution: register all nodes first, then resolve edges against full node set"
  - "Kind priority ordering: seq(0) < enum(1) < type(2) < domain(3) < table(4) < view(5) < matview(6) < func(7) < trigger(8) < policy(9)"

requirements-completed: [DEPS-01, DEPS-02, DEPS-03]

duration: 3min
completed: 2026-03-11
---

# Phase 6 Plan 1: DAG Builder and Topological Sort Summary

**In-memory DAG with Kahn's topological sort for PostgreSQL object dependency resolution with FK isolation and kind-priority tiebreaking**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-11T14:21:13Z
- **Completed:** 2026-03-11T14:24:30Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- DAG builder correctly extracts dependencies for all 10 non-FK object kinds (table->seq via nextval, table->type via column types, trigger->table+function, policy->table, function->type, domain->type, type->type)
- Kahn's topological sort produces deterministic restore order with kind-priority tiebreaking
- Circular dependency detection returns error listing participating nodes
- ForeignKeyDef objects excluded from DAG and appended after all other objects
- 25 unit tests all passing including realistic integration scenario

## Task Commits

Each task was committed atomically:

1. **Task 1: DAG builder with dependency extraction** - `73eec64` (feat)
2. **Task 2: Kahn's topological sort with FK isolation and cycle detection** - `4eec888` (feat)

_Note: TDD tasks -- tests and implementation written together per plan specification_

## Files Created/Modified
- `internal/resolve/dag.go` - DAG type with AddNode, AddObject, Resolve, dependency extraction by ObjectKind
- `internal/resolve/dag_test.go` - 15 unit tests for DAG construction, dependency extraction, isolated nodes
- `internal/resolve/kahn.go` - TopologicalSort method, BuildRestoreOrder function, kindPriority map
- `internal/resolve/kahn_test.go` - 10 unit tests for topological sort, cycle detection, FK isolation, integration

## Decisions Made
- Two-pass DAG design: AddObject registers nodes and customTypes in first pass, Resolve builds edges against complete node set -- avoids dependency target ordering issues
- Kind-priority tiebreaking (seq=0 through policy=9) instead of explicit view->table edges -- eliminates fragile dependency guessing for views/matviews
- inDegree semantics: tracks how many dependencies a node has (not how many dependents), so zero-inDegree nodes are ready to restore first
- FK constraints sorted alphabetically by schema.name for deterministic append order

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- DAG builder and topological sort ready for use by Plan 06-02 (manifest generation)
- BuildRestoreOrder is the main entry point for backup/restore orchestration
- All exports: DAG, NewDAG, NodeID, AddNode, AddObject, TopologicalSort, BuildRestoreOrder

---
*Phase: 06-dependency-resolution-manifest*
*Completed: 2026-03-11*
