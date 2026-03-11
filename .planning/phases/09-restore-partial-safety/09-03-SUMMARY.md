---
phase: 09-restore-partial-safety
plan: "03"
subsystem: database
tags: [go, restore, cobra, cli, flags, partial-restore]

# Dependency graph
requires:
  - phase: 09-restore-partial-safety
    plan: "01"
    provides: "filter.go, leak.go, logger.go, Options struct with Schema/Object/LogDir fields, RunRestore with full Phase 9 wiring"
  - phase: 09-restore-partial-safety
    plan: "02"
    provides: "Integration tests for REST-03/10/11/12/13 that exercise the new flags"
provides:
  - "cmd/pgbackup/cmd/restore.go: --schema, --object, --log-dir cobra flags registered and wired to Options"
  - "pgbackup restore --help shows all three new flags with descriptions"
  - "CLI surface for partial restore (REST-10, REST-13) is complete"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "cobra StringVar flag registration with descriptive help text matching Options field semantics"

key-files:
  created: []
  modified:
    - cmd/pgbackup/cmd/restore.go

key-decisions:
  - "Flag registration placed in init() after existing --input and --pre-backup-dir flags — consistent with existing ordering convention"
  - "MarkFlagRequired('input') kept at end of init() after all flag registrations"
  - "Placeholder comment from Plan 01 removed now that flags are live"

patterns-established:
  - "Cobra flag help text mirrors the format validation error message: 'schema.name format, e.g. public.mytable'"

requirements-completed: [REST-10, REST-13]

# Metrics
duration: 5min
completed: 2026-03-12
---

# Phase 9 Plan 03: Restore CLI Flag Wiring Summary

**Three cobra flags (--schema, --object, --log-dir) wired into pgbackup restore, completing the Phase 9 partial-restore CLI surface**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-12T00:50:00Z
- **Completed:** 2026-03-12T00:55:00Z
- **Tasks:** 1 auto + 1 checkpoint
- **Files modified:** 1

## Accomplishments
- Registered `--schema`, `--object`, `--log-dir` cobra flags in `restore.go` `init()` function
- Removed Plan 01 placeholder comment now that flags are live
- Verified `go build ./...` passes and `pgbackup restore --help` shows all three new flags
- `RunE` was already passing all five fields to `Options` from Plan 01 — no change needed

## Task Commits

Each task was committed atomically:

1. **Task 1: Register --schema, --object, --log-dir cobra flags** - `d27b8cf` (feat)

## Files Created/Modified
- `/home/parallels/workspaces/pgbkrs/cmd/pgbackup/cmd/restore.go` - Added three StringVar flag registrations; removed Plan 01 placeholder comment

## Decisions Made
- `RunE` was already fully wired from Plan 01 (all five Options fields). Only the `init()` registrations were missing. No structural changes needed.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `go build ./...` succeeded but did not overwrite `bin/pgbackup` (Go build without `-o` writes to `$GOPATH/bin` or no output for non-main packages). Used `go build -o bin/pgbackup ./cmd/pgbackup` for explicit binary update. This is expected Go build behavior, not a bug.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 9 is functionally complete: filter.go (BFS), leak.go (drop leak detection), logger.go (structured logs), full orchestrator wiring, and CLI flag surface
- Integration tests compile and skip cleanly without TEST_DATABASE_URL
- All REST-10, REST-13 requirements implemented and tested

---
*Phase: 09-restore-partial-safety*
*Completed: 2026-03-12*
