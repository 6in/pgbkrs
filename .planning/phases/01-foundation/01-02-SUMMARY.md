---
phase: 01-foundation
plan: 02
subsystem: cli
tags: [go, pgx, cobra, cli, makefile, db-connection, tdd]

# Dependency graph
requires:
  - phase: 01-foundation/01-01
    provides: Go module github.com/pgbkrs/pgbackup with pgx/v5, cobra, go-isatty; internal/core types package
provides:
  - Runnable bin/pgbackup binary with cobra root command
  - Three subcommand stubs: backup, restore, diff
  - internal/db package with BuildConnString() and ConnectAndPing() helpers
  - PersistentPreRunE fail-fast DB connection on all subcommands
  - Connection flags (--host, --port, --user, --password, --dbname) on root
  - Makefile with build/test/lint/clean targets
  - Unit tests for FOUND-01 (root_test.go) and FOUND-02 (connect_test.go)
affects: [02-table-fetcher, 03-serializer, 04-ddl-generator, 05-cli-scaffold, all-phases-using-pgbackup-binary]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - cobra PersistentPreRunE for fail-fast DB connection before any subcommand runs
    - Package-level *pgx.Conn variable in cmd package shared by all subcommands
    - go-isatty TTY detection for colored vs plain stderr error output
    - keyword/value connstring format via BuildConnString for pgx compatibility

key-files:
  created:
    - cmd/pgbackup/main.go (entry point calling cmd.Execute())
    - cmd/pgbackup/cmd/root.go (rootCmd, PersistentFlags, PersistentPreRunE, printError)
    - cmd/pgbackup/cmd/backup.go (backupCmd cobra stub)
    - cmd/pgbackup/cmd/restore.go (restoreCmd cobra stub)
    - cmd/pgbackup/cmd/diff.go (diffCmd cobra stub)
    - internal/db/connect.go (BuildConnString, ConnectAndPing with 10s timeout)
    - internal/db/connect_test.go (TestBuildConnString, TestConnect integration)
    - cmd/pgbackup/cmd/root_test.go (TestRootNoArgs, TestSubcommandsRecognized)
    - Makefile (build/test/lint/clean targets)
  modified: []

key-decisions:
  - "PersistentPreRunE used on rootCmd so DB connection runs exactly once for all subcommands"
  - "Package-level *pgx.Conn var in cmd package chosen over context.Context storage for simplicity"
  - "10-second timeout on ConnectAndPing prevents hanging on unreachable hosts"
  - "SilenceUsage and SilenceErrors set on rootCmd to suppress cobra's usage dump on errors"
  - "bin/pgbackup committed to git as the build artifact for convenience"

patterns-established:
  - "Pattern 5: cobra PersistentPreRunE on root provides fail-fast DB connection inherited by all subcommands"
  - "Pattern 6: internal/db.BuildConnString + ConnectAndPing are the single connection entry points; cmd package uses them"
  - "Pattern 7: printError in cmd package handles TTY detection centrally for all error output"

requirements-completed: [FOUND-01, FOUND-02]

# Metrics
duration: 2min
completed: 2026-03-11
---

# Phase 1 Plan 2: CLI Scaffold and DB Connection Summary

**Cobra CLI binary pgbackup with three subcommand stubs (backup/restore/diff), fail-fast pgx DB connection via PersistentPreRunE, and internal/db helpers**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-03-11T07:02:00Z
- **Completed:** 2026-03-11T07:03:43Z
- **Tasks:** 2 (TDD: RED + GREEN commits for Task 2)
- **Files modified:** 9

## Accomplishments
- Created `internal/db` package with `BuildConnString` and `ConnectAndPing` (10s timeout) helpers
- Built runnable `bin/pgbackup` binary with cobra root command and three subcommand stubs
- Implemented fail-fast DB connection via `PersistentPreRunE` — all subcommands inherit it automatically
- Connection flags (`--host`, `--port`, `--user`, `--password`, `--dbname`) defined once on root as `PersistentFlags`
- All 4 unit tests passing: TestBuildConnString, TestRootNoArgs, TestSubcommandsRecognized, TestInterfaceCompliance
- Makefile with build/test/lint/clean targets at repo root

## Task Commits

Each task was committed atomically:

1. **Task 1: Create internal/db connection helper and Makefile** - `cededf7` (feat)
2. **Task 2 RED: Add failing tests for cobra CLI scaffold** - `89f69f8` (test)
3. **Task 2 GREEN: Implement cobra CLI scaffold** - `a66c397` (feat)

## Files Created/Modified
- `internal/db/connect.go` - BuildConnString() and ConnectAndPing() with 10-second timeout context
- `internal/db/connect_test.go` - TestBuildConnString (unit) and TestConnect (integration, skips without TEST_DATABASE_URL)
- `cmd/pgbackup/main.go` - Entry point; calls cmd.Execute()
- `cmd/pgbackup/cmd/root.go` - rootCmd with PersistentFlags, PersistentPreRunE, Execute(), printError()
- `cmd/pgbackup/cmd/backup.go` - backupCmd cobra.Command stub
- `cmd/pgbackup/cmd/restore.go` - restoreCmd cobra.Command stub
- `cmd/pgbackup/cmd/diff.go` - diffCmd cobra.Command stub
- `cmd/pgbackup/cmd/root_test.go` - TestRootNoArgs and TestSubcommandsRecognized
- `Makefile` - build/test/lint/clean targets for bin/pgbackup

## Decisions Made
- Used `PersistentPreRunE` on root (not child) commands — cobra propagates it to all three subcommands automatically; no duplication needed
- Package-level `conn *pgx.Conn` in `cmd` package preferred over `context.Context` storage — simpler and correct for sequential single-user CLI
- `SilenceUsage: true` and `SilenceErrors: true` on rootCmd — DB errors should not dump usage to screen; user can always run `--help`
- 10-second timeout in `ConnectAndPing` prevents infinite hang on unreachable hosts (Pitfall 3 from RESEARCH.md)
- `bin/pgbackup` committed to git for easy access — can be `.gitignore`d in a future cleanup pass

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None — all libraries were already in go.mod from Plan 01-01. Build and tests passed on first attempt.

## User Setup Required
None — no external service configuration required.

Integration test (`TestConnect`) requires `TEST_DATABASE_URL` env var to run against a live PostgreSQL instance. Without it, the test is automatically skipped (`t.Skip`).

## Next Phase Readiness
- `bin/pgbackup` binary is runnable and recognizes all three subcommands
- `internal/db` package is the DB connection entry point for all future phases
- `PersistentPreRunE` wiring is in place — Phase 2+ implementations in `backupCmd.RunE` will have `conn` available automatically
- No blockers — `go test ./...` is fully green

## Self-Check: PASSED

All created files verified present. All task commits verified in git log.
