---
phase: 01-foundation
plan: 01
subsystem: core
tags: [go, pgx, cobra, interfaces, registry, command-pattern]

# Dependency graph
requires: []
provides:
  - Go module github.com/pgbkrs/pgbackup with pgx/v5, cobra, go.yaml.in/yaml/v3, go-isatty
  - SchemaFetcher, Serializer, DDLGenerator interfaces in internal/core
  - All 10 ObjectKind constants (table, view, materialized_view, function, sequence, trigger, type, domain, enum, policy)
  - ObjectHeader struct and ObjectDef interface
  - 10 concrete *Def structs (TableDef, ViewDef, MaterializedViewDef, FunctionDef, SequenceDef, TriggerDef, TypeDef, DomainDef, EnumDef, PolicyDef)
  - CommandRegistry with Register/Fetcher/Serializer/Generator API
  - Unit tests covering FOUND-03 and FOUND-04 requirements
affects: [02-table-fetcher, 03-serializer, 04-ddl-generator, 05-cli-scaffold, all-internal-packages]

# Tech tracking
tech-stack:
  added:
    - github.com/jackc/pgx/v5 v5.8.0 (PostgreSQL driver)
    - github.com/spf13/cobra v1.10.2 (CLI framework)
    - go.yaml.in/yaml/v3 v3.0.4 (YAML serialization)
    - github.com/mattn/go-isatty v0.0.20 (TTY detection)
  patterns:
    - Command pattern via CommandRegistry (ObjectKind -> SchemaFetcher/Serializer/DDLGenerator)
    - ObjectDef interface with embedded ObjectHeader for type-safe schema objects
    - internal/core as shared types package with no internal imports (prevents circular deps)

key-files:
  created:
    - go.mod (module declaration, go 1.24.0, all dependencies)
    - go.sum (dependency checksums)
    - internal/core/interfaces.go (SchemaFetcher, Serializer, DDLGenerator interfaces)
    - internal/core/types.go (ObjectKind constants, ObjectHeader, ObjectDef, all *Def structs)
    - internal/core/registry.go (CommandRegistry struct with Register/Fetcher/Serializer/Generator)
    - internal/core/registry_test.go (TestCommandRegistry, TestRegistrySingleLine)
    - internal/core/interfaces_test.go (TestInterfaceCompliance, TestObjectDefHeader)
  modified: []

key-decisions:
  - "Go module path github.com/pgbkrs/pgbackup matches workspace directory name"
  - "go.yaml.in/yaml/v3 used instead of archived gopkg.in/yaml.v3 (API-identical maintained fork)"
  - "Go 1.24.0 required by pgx v5.8.0 (toolchain auto-upgraded from 1.23.6)"
  - "internal/core has zero imports from other internal/ packages to prevent circular dependencies"
  - "All *Def structs implement ObjectDef via value receiver Header() ObjectHeader returning embedded ObjectHeader"

patterns-established:
  - "Pattern 1: CommandRegistry maps ObjectKind to interface triples; single Register() call makes all three resolvable"
  - "Pattern 2: ObjectDef interface satisfied by embedding ObjectHeader and implementing Header() ObjectHeader"
  - "Pattern 3: internal/core is the shared types anchor — all other internal packages import it, it imports nothing internal"
  - "Pattern 4: External test package (core_test) for registry tests; internal package (core) for compile-time interface assertions"

requirements-completed: [FOUND-03, FOUND-04]

# Metrics
duration: 3min
completed: 2026-03-11
---

# Phase 1 Plan 1: Foundation Core Interfaces Summary

**Go module with SchemaFetcher/Serializer/DDLGenerator interfaces, 10 ObjectKind constants, and CommandRegistry command pattern using pgx/v5 and cobra**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-03-11T06:57:09Z
- **Completed:** 2026-03-11T06:59:37Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Initialized Go module `github.com/pgbkrs/pgbackup` with go 1.24.0 and all four project dependencies
- Created `internal/core` package with SchemaFetcher, Serializer, DDLGenerator interfaces and 10 ObjectKind constants
- Implemented CommandRegistry with single-call Register() that makes Fetcher/Serializer/Generator all resolvable
- All 10 concrete *Def structs (TableDef through PolicyDef) satisfying ObjectDef interface
- 4 unit tests passing: TestCommandRegistry, TestRegistrySingleLine, TestInterfaceCompliance, TestObjectDefHeader

## Task Commits

Each task was committed atomically:

1. **Task 1: Initialize Go module and create internal/core package** - `31f4f60` (feat)
2. **Task 2: Write unit tests for internal/core (FOUND-03, FOUND-04)** - `b695ea5` (test)

## Files Created/Modified
- `go.mod` - Module declaration github.com/pgbkrs/pgbackup, go 1.24.0, all 4 direct deps
- `go.sum` - Dependency checksums
- `internal/core/interfaces.go` - SchemaFetcher, Serializer, DDLGenerator interfaces importing pgx/v5
- `internal/core/types.go` - ObjectKind string type, 10 constants, ObjectHeader, ObjectDef interface, 10 concrete *Def structs
- `internal/core/registry.go` - CommandRegistry struct, NewCommandRegistry, Register, Fetcher, Serializer, Generator methods
- `internal/core/registry_test.go` - External package tests: TestCommandRegistry, TestRegistrySingleLine with stub implementations
- `internal/core/interfaces_test.go` - Internal package tests: compile-time var _ assertions, TestInterfaceCompliance, TestObjectDefHeader

## Decisions Made
- Used `go.yaml.in/yaml/v3` (maintained fork) instead of archived `gopkg.in/yaml.v3` per RESEARCH.md
- Go version auto-upgraded to 1.24.0 by toolchain because pgx v5.8.0 requires it (go get resolved this automatically)
- Module path `github.com/pgbkrs/pgbackup` chosen to match workspace directory as recommended in RESEARCH.md
- Value receivers used for all *Def.Header() methods (consistent with spec, allows non-pointer ObjectDef in var _ assertions)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Go 1.23.6 was active via goenv but pgx v5.8.0 requires go >= 1.24.0. The `go get` command auto-resolved this by downloading go1.24.0 and updating go.mod to `go 1.24.0`. No manual intervention required.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `internal/core` package is the foundation all subsequent phases build on
- All three interfaces (SchemaFetcher, Serializer, DDLGenerator) are ready to be implemented by object-specific packages
- CommandRegistry is ready to receive Register() calls from concrete implementations
- No blockers — go test ./internal/core/... is green

## Self-Check: PASSED

All created files verified present. All task commits verified in git log.

- go.mod: FOUND
- internal/core/interfaces.go: FOUND
- internal/core/types.go: FOUND
- internal/core/registry.go: FOUND
- internal/core/registry_test.go: FOUND
- internal/core/interfaces_test.go: FOUND
- 01-01-SUMMARY.md: FOUND
- Commit 31f4f60 (feat: initialize Go module): FOUND
- Commit b695ea5 (test: add unit tests): FOUND

---
*Phase: 01-foundation*
*Completed: 2026-03-11*
