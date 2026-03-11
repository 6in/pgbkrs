---
phase: 02-schema-fetch-tables-sequences
plan: "02"
subsystem: database
tags: [go, pgx, postgres, schema-fetch, tables, pg_catalog, tdd]

# Dependency graph
requires:
  - phase: 02-schema-fetch-tables-sequences
    plan: "01"
    provides: "Fully-specified TableDef struct in internal/core/types.go, shared test helpers, integration test stubs"
provides:
  - "internal/fetch/table/fetcher.go: TableSchemaFetcher implementing core.SchemaFetcher via pg_catalog queries"
  - "Full integration test assertions in fetcher_test.go for columns, constraints, indexes, RLS, and partitioning"
affects: [03-serialization, 04-ddl-generation, 05-backup-restore]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "N+1 query pattern for table sub-objects (columns/constraints/indexes/partitioning): acceptable for CLI backup tool"
    - "pgx.CollectRows + pgx.RowToStructByName for all multi-row scans with db struct tags"
    - "pg_get_constraintdef output parsed in Go to extract PK column names (split on comma inside parens)"
    - "relkind='p' gate for partition queries avoids unnecessary pg_partitioned_table lookups on regular tables"

key-files:
  created:
    - internal/fetch/table/fetcher.go
  modified:
    - internal/fetch/table/fetcher_test.go

key-decisions:
  - "Use pointer receiver (*SchemaFetcher) for Fetch method to satisfy interface; compile-time check via var _ core.SchemaFetcher = (*SchemaFetcher)(nil)"
  - "PK column parsing from pg_get_constraintdef output (e.g. PRIMARY KEY (id)) rather than conkey join — avoids secondary pg_attribute query"
  - "Constraint-backing index exclusion via NOT EXISTS subquery on pg_constraint.conindid — matches RESEARCH.md recommendation"
  - "Partitioning queries only run for relkind='p' tables — no-op for regular tables (relkind='r')"

patterns-established:
  - "Table fetcher package: package table with external test package table_test"
  - "findTable helper in tests locates *core.TableDef by Name in returned []core.ObjectDef slice"
  - "All assertions use stdlib t.Fatal/t.Errorf — no third-party assertion library"

requirements-completed: [FETCH-01]

# Metrics
duration: 2min
completed: "2026-03-11"
---

# Phase 2 Plan 02: TableSchemaFetcher Implementation Summary

**pg_catalog-backed TableSchemaFetcher with five parameterized SQL queries returning fully-populated TableDef objects including columns, PK/unique/check constraints, non-constraint indexes, partitioning strategy+children, and RLS flag**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-03-11T07:33:34Z
- **Completed:** 2026-03-11T07:36:12Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments
- Implemented `internal/fetch/table/fetcher.go` with five pg_catalog queries (tables list, columns, constraints, indexes, partition info)
- Updated `fetcher_test.go` stub to full assertions: TestFetchTable, TestFetchTableConstraints, TestFetchTableIndexes, TestFetchTableRLS, TestFetchPartitionedTable
- Compile-time interface compliance confirmed: `var _ core.SchemaFetcher = (*SchemaFetcher)(nil)`
- All five test functions skip cleanly without TEST_DATABASE_URL; full assertions ready for integration testing with live PostgreSQL

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement TableSchemaFetcher** - `6aaf091` (feat)

**Plan metadata:** (docs commit — see below)

_Note: This was a TDD task — the RED stubs were written in Plan 01 (commit ca00ded). This commit brings the GREEN phase._

## Files Created/Modified
- `internal/fetch/table/fetcher.go` - SchemaFetcher struct with five pg_catalog query functions
- `internal/fetch/table/fetcher_test.go` - Updated from STUB log statements to real assertions for all five TestFetch* functions

## Decisions Made
- PK columns extracted by parsing `pg_get_constraintdef` output (splitting inside parentheses) rather than querying `pg_attribute` again — simpler, no join needed
- Constraint-backing index exclusion done at SQL level (NOT EXISTS subquery) rather than Go-level filtering
- Partition queries skip entirely for relkind='r' tables (guarded by `if t.Relkind == "p"`)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. Integration tests require `TEST_DATABASE_URL` but skip cleanly when absent.

## Next Phase Readiness
- `TableSchemaFetcher` is ready for wiring into the backup command registry
- Integration tests (TestFetchTable, TestFetchTableConstraints, etc.) are ready to run against a live PostgreSQL instance
- Wave 2 fetchers for sequence, enum, composite type, and domain are already complete (Plans 02-03 and 02-04)
- Phase 3 (serialization) can now consume `*core.TableDef` values from this fetcher

---
*Phase: 02-schema-fetch-tables-sequences*
*Completed: 2026-03-11*

## Self-Check: PASSED

All files verified on disk. Task commit 6aaf091 verified in git history.
