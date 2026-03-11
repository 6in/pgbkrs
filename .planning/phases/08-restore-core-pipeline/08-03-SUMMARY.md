---
phase: 08-restore-core-pipeline
plan: "03"
subsystem: database
tags: [go, restore, cli, cobra, flags]

# Dependency graph
requires:
  - phase: 08-02
    provides: RunRestore(ctx, conn, backupDir, preBackupDir string) error — full 7-wave restore pipeline
  - phase: 07-backup-orchestration
    provides: RunBackup (used for pre-restore safety backup inside RunRestore)
provides:
  - pgbackup restore CLI subcommand with --input (required) and --pre-backup-dir (optional) flags
  - Full end-to-end restore pipeline reachable via CLI
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Mirror backup.go flag pattern: StringVar flags in init(), MarkFlagRequired for mandatory flags"
    - "Package-level conn *pgx.Conn from PersistentPreRunE inherited by all subcommands"

key-files:
  created: []
  modified:
    - cmd/pgbackup/cmd/restore.go

key-decisions:
  - "MarkFlagRequired('input') on restoreCmd makes cobra return an error if --input is absent when RunE is reached"
  - "--pre-backup-dir defaults to '.' matching backup's --output default convention"

requirements-completed: [REST-01, REST-02, REST-04, REST-05, REST-06, REST-07, REST-08, REST-09]

# Metrics
duration: 1min
completed: 2026-03-12
---

# Phase 08 Plan 03: Restore CLI Wiring Summary

**pgbackup restore subcommand wired with --input (required) and --pre-backup-dir (optional, default '.') flags calling restore.RunRestore for full 7-wave pipeline execution**

## Performance

- **Duration:** ~1 min
- **Started:** 2026-03-11T15:33:51Z
- **Completed:** 2026-03-11T15:34:37Z
- **Tasks:** 2 (1 auto + 1 checkpoint auto-approved)
- **Files modified:** 1

## Accomplishments

- Replaced stub restore.go with fully wired restore subcommand
- `--input` flag marked required; `--pre-backup-dir` defaults to `"."`
- `pgbackup restore --input <dir>` now executes the full 7-wave RunRestore pipeline
- `go build ./...` clean; `pgbackup restore --help` shows both flags correctly

## Task Commits

1. **Task 1: Wire restore CLI flags and call RunRestore** - `905eea8` (feat)
2. **Task 2: Human verify full restore round-trip** - auto-approved (checkpoint:human-verify, --auto mode)

## Files Created/Modified

- `cmd/pgbackup/cmd/restore.go` — Replaced stub with wired restoreCmd: --input (required), --pre-backup-dir (default "."), calls restore.RunRestore(cmd.Context(), conn, restoreInputDir, restorePreBackupDir)

## Decisions Made

- `MarkFlagRequired("input")` used to enforce required flag at cobra level — matches the task's done criteria for "required flag(s) 'input' not set" error
- Pre-backup-dir defaults to `"."` matching the `--output` default in backup.go for consistency

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None — no external service configuration required for this plan.

## Self-Check

- [x] `cmd/pgbackup/cmd/restore.go` exists with RunRestore call and flag wiring
- [x] `go build ./...` passes
- [x] `./bin/pgbackup restore --help` shows `--input` and `--pre-backup-dir` flags
- [x] Commit 905eea8 exists (restore.go wiring)

## Self-Check: PASSED

## Next Phase Readiness

- Full pgbackup CLI is complete: backup, restore, diff subcommands all wired
- Integration testing with a real database can be performed using `pgbackup backup` then `pgbackup restore`
- All REST-01 through REST-09 requirements implemented and CLI-accessible

---
*Phase: 08-restore-core-pipeline*
*Completed: 2026-03-12*
