---
phase: 04-ddl-generation
plan: 01
subsystem: database
tags: [postgresql, ddl, foreign-key, partition, schema-fetcher, tdd]

# Dependency graph
requires:
  - phase: 02-schema-fetch-tables-sequences
    provides: "TableDef, PartitionDef, SequenceDef, TypeDef, DomainDef, EnumDef types"
  - phase: 03-schema-fetch-views-functions-triggers-policies
    provides: "ViewDef, MaterializedViewDef, FunctionDef, TriggerDef, PolicyDef types"
provides:
  - "ForeignKeyDef struct with SourceTable, TargetSchema, TargetTable, Definition fields"
  - "KindForeignKey ObjectKind constant"
  - "PartitionDef.KeyExpression field for pg_get_partkeydef output"
  - "FK SchemaFetcher querying contype='f'"
  - "11 DDL generator packages with stub generator.go and test files (Nyquist Wave 0)"
affects: [04-02, 04-03, 04-04, 04-05, 04-06, 04-07, 04-08, 04-09, 04-10, 04-11, 04-12]

# Tech tracking
tech-stack:
  added: []
  patterns: ["stub DDL generator with not-implemented returns for Wave 0 scaffolding"]

key-files:
  created:
    - internal/core/types_phase4_test.go
    - internal/fetch/foreignkey/fetcher.go
    - internal/fetch/foreignkey/fetcher_test.go
    - internal/ddl/table/generator.go
    - internal/ddl/table/generator_test.go
    - internal/ddl/foreignkey/generator.go
    - internal/ddl/foreignkey/generator_test.go
    - internal/ddl/view/generator.go
    - internal/ddl/view/generator_test.go
    - internal/ddl/matview/generator.go
    - internal/ddl/matview/generator_test.go
    - internal/ddl/function/generator.go
    - internal/ddl/function/generator_test.go
    - internal/ddl/trigger/generator.go
    - internal/ddl/trigger/generator_test.go
    - internal/ddl/sequence/generator.go
    - internal/ddl/sequence/generator_test.go
    - internal/ddl/compositetype/generator.go
    - internal/ddl/compositetype/generator_test.go
    - internal/ddl/domain/generator.go
    - internal/ddl/domain/generator_test.go
    - internal/ddl/enum/generator.go
    - internal/ddl/enum/generator_test.go
    - internal/ddl/policy/generator.go
    - internal/ddl/policy/generator_test.go
  modified:
    - internal/core/types.go
    - internal/fetch/table/fetcher.go

key-decisions:
  - "ForeignKeyDef as independent ObjectDef with embedded ObjectHeader, not nested in TableDef"
  - "PartitionDef.KeyExpression populated via separate pg_get_partkeydef query in fetchPartitioning"
  - "DDL generator stubs return fmt.Errorf not-implemented to make tests compilable but RED"

patterns-established:
  - "DDL generator package layout: internal/ddl/<kind>/generator.go + generator_test.go"
  - "Stub DDLGenerator struct with GenerateDDL and GenerateDrop returning not-implemented"

requirements-completed: [DDLG-01, DDLG-02, DDLG-03, DDLG-04]

# Metrics
duration: 4min
completed: 2026-03-11
---

# Phase 4 Plan 01: Types, FK Fetcher, and DDL Test Scaffolding Summary

**ForeignKeyDef type with KindForeignKey constant, FK SchemaFetcher querying pg_constraint, PartitionDef.KeyExpression via pg_get_partkeydef, and 11 DDL generator test stubs in RED state**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-11T11:43:39Z
- **Completed:** 2026-03-11T11:47:49Z
- **Tasks:** 2
- **Files modified:** 26

## Accomplishments
- Added ForeignKeyDef struct with SourceTable, TargetSchema, TargetTable, Definition fields and KindForeignKey constant
- Extended PartitionDef with KeyExpression field; fetchPartitioning now queries pg_get_partkeydef
- Created FK SchemaFetcher that queries contype='f' from pg_constraint with proper JOINs
- Established 11 DDL generator packages with compilable stub generators and test files (Nyquist Wave 0)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add ForeignKeyDef, KindForeignKey, and PartitionDef.KeyExpression** (TDD)
   - `10e04c9` (test: failing tests for new types)
   - `7f1b801` (feat: implement ForeignKeyDef, KindForeignKey, KeyExpression)
2. **Task 2: Create FK SchemaFetcher and 11 DDL generator test stubs** - `05ec0e1` (feat)

## Files Created/Modified
- `internal/core/types.go` - Added KindForeignKey, ForeignKeyDef struct, PartitionDef.KeyExpression
- `internal/core/types_phase4_test.go` - TDD tests for new types
- `internal/fetch/table/fetcher.go` - fetchPartitioning now queries pg_get_partkeydef
- `internal/fetch/foreignkey/fetcher.go` - FK SchemaFetcher querying contype='f'
- `internal/fetch/foreignkey/fetcher_test.go` - Integration test stub
- `internal/ddl/{table,foreignkey,view,matview,function,trigger,sequence,compositetype,domain,enum,policy}/generator.go` - Stub DDL generators
- `internal/ddl/{table,foreignkey,view,matview,function,trigger,sequence,compositetype,domain,enum,policy}/generator_test.go` - Test stubs with fixtures

## Decisions Made
- ForeignKeyDef as independent ObjectDef with embedded ObjectHeader (consistent with all other *Def types)
- PartitionDef.KeyExpression populated via separate pg_get_partkeydef($1) query rather than modifying existing strategy query
- DDL generator stubs use fmt.Errorf("not implemented") to make tests compilable but in RED state

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 11 DDL generator packages have test stubs defining the expected API contract
- Wave 2 plans can implement each generator by replacing "not implemented" with real logic
- ForeignKeyDef and FK SchemaFetcher ready for FK-specific DDL generation plan

## Self-Check: PASSED

All 25 created files verified. All 3 commits (10e04c9, 7f1b801, 05ec0e1) verified.

---
*Phase: 04-ddl-generation*
*Completed: 2026-03-11*
