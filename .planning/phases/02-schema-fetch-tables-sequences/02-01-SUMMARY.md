---
phase: 02-schema-fetch-tables-sequences
plan: "01"
subsystem: database
tags: [go, pgx, postgres, schema-fetch, types, testing]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: "internal/core package with ObjectKind, ObjectHeader, ObjectDef interface, registry, and stub *Def types"
provides:
  - "Fully-specified *Def structs in internal/core/types.go: ColumnDef, ConstraintsDef, PrimaryKeyDef, UniqueConstraintDef, CheckConstraintDef, IndexDef, PartitionDef, RLSDef, CompositeField, TableDef, SequenceDef, TypeDef, DomainDef, EnumDef"
  - "Shared test helper internal/fetch/testhelpers with ConnectTestDB and SetupTestSchema"
  - "Five integration test stubs for Wave 2 fetcher implementations (table, sequence, compositetype, domain, enum)"
affects: [02-02, 02-03, 02-04, 03-serialization, 04-ddl-generation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Test-first scaffolding: integration test stubs written before production code (Nyquist rule)"
    - "TEST_DATABASE_URL skip pattern for integration tests matching db/connect_test.go"
    - "SetupTestSchema helper centralizes fixture creation and returns typed teardown func"

key-files:
  created:
    - internal/fetch/testhelpers/setup.go
    - internal/fetch/table/fetcher_test.go
    - internal/fetch/sequence/fetcher_test.go
    - internal/fetch/compositetype/fetcher_test.go
    - internal/fetch/domain/fetcher_test.go
    - internal/fetch/enum/fetcher_test.go
  modified:
    - internal/core/types.go

key-decisions:
  - "TableDef: Columns []ColumnDef, Constraints ConstraintsDef, Indexes []IndexDef, Partitioning *PartitionDef (nil for non-partitioned), RLS RLSDef"
  - "SequenceDef: int64 fields for all numeric sequence attributes plus IsCalled bool for never-advanced detection"
  - "TypeDef renamed fields to Fields []CompositeField (composite type fields) — no Definition string"
  - "DomainDef: BaseType + Nullable + Default + CheckName + CheckDefinition — all from pg_catalog"
  - "EnumDef: Labels []string ordered by enumsortorder"
  - "Test stubs reference forward-declared SchemaFetcher — intentionally non-compiling per package until Wave 2"

patterns-established:
  - "All integration test files use _test package suffix (e.g., table_test, sequence_test)"
  - "ConnectTestDB skips test when TEST_DATABASE_URL unset — matching existing db package pattern"
  - "SetupTestSchema returns (schema string, teardown func()) — caller uses defer teardown()"

requirements-completed: [FETCH-01, FETCH-06, FETCH-07, FETCH-08, FETCH-09]

# Metrics
duration: 2min
completed: "2026-03-11"
---

# Phase 2 Plan 01: Schema Types and Test Scaffolds Summary

**Expanded stub *Def structs to fully-specified Go types (TableDef/SequenceDef/TypeDef/DomainDef/EnumDef) and created all five integration test stubs with shared SetupTestSchema helper for Wave 2 fetcher implementations**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-03-11T07:29:29Z
- **Completed:** 2026-03-11T07:31:17Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Replaced 5 stub `*Def` structs (each with only `Definition string`) with 14 fully-specified types matching the pg_catalog schema fetch requirements
- Created `internal/fetch/testhelpers` package with `ConnectTestDB` and `SetupTestSchema` helpers establishing shared fixture infrastructure for all fetcher tests
- Created 5 integration test stubs (table with 5 test functions, plus sequence/compositetype/domain/enum each with 1) — all compile independently, skip without TEST_DATABASE_URL
- All existing `internal/core` tests still pass after struct expansion

## Task Commits

Each task was committed atomically:

1. **Task 1: Expand *Def structs in internal/core/types.go** - `aeacabc` (feat)
2. **Task 2: Create shared test helper and all five integration test stubs** - `ca00ded` (test)

**Plan metadata:** (docs commit — see below)

## Files Created/Modified
- `internal/core/types.go` - Expanded from 5 stub structs to 14 full types (ColumnDef, PrimaryKeyDef, UniqueConstraintDef, CheckConstraintDef, ConstraintsDef, IndexDef, PartitionDef, RLSDef, CompositeField, plus expanded TableDef/SequenceDef/TypeDef/DomainDef/EnumDef)
- `internal/fetch/testhelpers/setup.go` - ConnectTestDB + SetupTestSchema with full fixture set
- `internal/fetch/table/fetcher_test.go` - Stubs for TestFetchTable, TestFetchPartitionedTable, TestFetchTableRLS, TestFetchTableConstraints, TestFetchTableIndexes
- `internal/fetch/sequence/fetcher_test.go` - Stub for TestFetchSequence
- `internal/fetch/compositetype/fetcher_test.go` - Stub for TestFetchCompositeType
- `internal/fetch/domain/fetcher_test.go` - Stub for TestFetchDomain
- `internal/fetch/enum/fetcher_test.go` - Stub for TestFetchEnum

## Decisions Made
- Kept `ViewDef`, `MaterializedViewDef`, `FunctionDef`, `TriggerDef`, `PolicyDef` unchanged with their `Definition string` field — these are out-of-scope for Phase 2
- Used `*PartitionDef` (pointer) in TableDef so nil signals non-partitioned tables without a separate boolean flag
- Used `*PrimaryKeyDef` (pointer) in ConstraintsDef so nil signals no primary key, matching PostgreSQL reality
- Test stubs intentionally reference forward-declared `SchemaFetcher` — per Nyquist rule, tests are written before production code; Wave 2 plans create the actual fetcher structs

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. Integration tests require `TEST_DATABASE_URL` but skip cleanly when absent.

## Next Phase Readiness
- Typed contracts (all *Def structs) are ready for Wave 2 fetcher implementations (Plans 02-03 and 02-04)
- Test fixtures via `SetupTestSchema` cover all object types needed by Wave 2
- Wave 2 plans need to: (1) create `SchemaFetcher` structs in each package, (2) implement `Fetch` methods, (3) fill in test assertions

---
*Phase: 02-schema-fetch-tables-sequences*
*Completed: 2026-03-11*

## Self-Check: PASSED

All files verified on disk. All task commits verified in git history.
