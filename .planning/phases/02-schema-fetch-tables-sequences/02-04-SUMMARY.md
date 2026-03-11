---
phase: 02-schema-fetch-tables-sequences
plan: "04"
subsystem: database
tags: [go, pgx, postgres, pg_catalog, schema-fetch, composite-type, domain, tdd]

# Dependency graph
requires:
  - phase: 02-schema-fetch-tables-sequences
    plan: "01"
    provides: "Fully-specified TypeDef/DomainDef structs, test stubs with ConnectTestDB/SetupTestSchema helpers"
provides:
  - "CompositeTypeSchemaFetcher in internal/fetch/compositetype implementing core.SchemaFetcher (FETCH-07)"
  - "DomainSchemaFetcher in internal/fetch/domain implementing core.SchemaFetcher (FETCH-08)"
affects: [03-serialization, 04-ddl-generation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Composite type grouping: query returns one row per field, group by OID in Go using ORDER BY t.typname, a.attnum"
    - "relkind='c' filter on pg_class to exclude table-row composite types (RESEARCH.md Pitfall 1)"
    - "LEFT JOIN pg_constraint c ON c.contypid = t.oid for optional domain check constraints"
    - "COALESCE for nullable pg_catalog columns (conname, pg_get_constraintdef) to return empty string"

key-files:
  created:
    - internal/fetch/compositetype/fetcher.go
    - internal/fetch/domain/fetcher.go
  modified:
    - internal/fetch/compositetype/fetcher_test.go
    - internal/fetch/domain/fetcher_test.go

key-decisions:
  - "External test package (compositetype_test, domain_test) requires explicit package import in test files referencing SchemaFetcher"
  - "Composite type field grouping done in Go (not SQL) using ordered query output — simpler than SQL aggregation"
  - "Domain fetcher uses COALESCE on conname/pg_get_constraintdef so domains without check constraints return empty strings"
  - "CheckDefinition assertion uses strings.Contains (not equality) to be resilient to PostgreSQL parenthesization of CHECK clause"

patterns-established:
  - "Both fetchers follow identical SchemaFetcher interface pattern established by RESEARCH.md Architecture Pattern 1"
  - "Compile-time interface check: var _ core.SchemaFetcher = (*SchemaFetcher)(nil) at bottom of each fetcher file"

requirements-completed: [FETCH-07, FETCH-08]

# Metrics
duration: 2min
completed: "2026-03-11"
---

# Phase 2 Plan 04: Composite Type and Domain Fetchers Summary

**pg_catalog-based CompositeTypeSchemaFetcher (relkind='c' exclusion) and DomainSchemaFetcher (LEFT JOIN pg_constraint on contypid) implementing core.SchemaFetcher for FETCH-07 and FETCH-08**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-03-11T07:33:44Z
- **Completed:** 2026-03-11T07:35:45Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Implemented `CompositeTypeSchemaFetcher` with correct relkind='c' filter that excludes table-row composite types; groups rows by OID in Go to build TypeDef.Fields slice preserving attnum order
- Implemented `DomainSchemaFetcher` with LEFT JOIN pg_constraint on contypid, COALESCE on nullable columns, and correct typnotnull → Nullable inversion
- Updated both test stubs (Wave 1) with full assertions matching plan behavior spec; TDD flow confirmed RED → GREEN for each task

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement CompositeTypeSchemaFetcher (FETCH-07)** - `7d0fc87` (feat)
2. **Task 2: Implement DomainSchemaFetcher (FETCH-08)** - `06b3c40` (feat)

**Plan metadata:** (docs commit — see below)

_Note: TDD tasks had RED (test update) → GREEN (implementation) flow; committed together per task_

## Files Created/Modified
- `internal/fetch/compositetype/fetcher.go` - SchemaFetcher with pg_type+pg_class+pg_attribute query, row-by-OID grouping, relkind='c' filter
- `internal/fetch/compositetype/fetcher_test.go` - Full assertions: address_type 3-field check (name+type in order), simple_table exclusion check
- `internal/fetch/domain/fetcher.go` - SchemaFetcher with pg_type+pg_constraint LEFT JOIN, COALESCE for optional fields
- `internal/fetch/domain/fetcher_test.go` - Full assertions: BaseType, Nullable, Default, CheckName non-empty, CheckDefinition Contains "VALUE > 0"

## Decisions Made
- External test packages (`compositetype_test`, `domain_test`) reference `SchemaFetcher` via package-qualified name (e.g., `compositetype.SchemaFetcher{}`), requiring explicit import — this is standard Go external test package pattern
- Composite type row grouping done in Go rather than SQL aggregation (e.g., `array_agg`) — query result order (`ORDER BY t.typname, a.attnum`) guarantees rows are already grouped, Go grouping is straightforward
- Domain COALESCE approach: `pg_get_constraintdef(c.oid)` returns NULL when c.oid is NULL (LEFT JOIN miss), COALESCE maps this to empty string rather than a Go nil pointer

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. Integration tests require `TEST_DATABASE_URL` but skip cleanly when absent.

## Next Phase Readiness
- CompositeTypeSchemaFetcher and DomainSchemaFetcher are complete and compile-verified
- Both implement `core.SchemaFetcher` via compile-time check
- Integration tests will run fully against a live PostgreSQL when `TEST_DATABASE_URL` is set
- Wave 2 Plans 02-02 (tables), 02-03 (sequences), and 02-05 (enums if applicable) complete the full fetcher set for Phase 2

---
*Phase: 02-schema-fetch-tables-sequences*
*Completed: 2026-03-11*

## Self-Check: PASSED

All files verified on disk:
- FOUND: internal/fetch/compositetype/fetcher.go
- FOUND: internal/fetch/domain/fetcher.go
- FOUND: .planning/phases/02-schema-fetch-tables-sequences/02-04-SUMMARY.md

All task commits verified in git history:
- FOUND commit: 7d0fc87 (feat(02-04): implement CompositeTypeSchemaFetcher)
- FOUND commit: 06b3c40 (feat(02-04): implement DomainSchemaFetcher)
