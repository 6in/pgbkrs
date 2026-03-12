---
phase: 10-schema-diff
plan: "02"
subsystem: database
tags: [schema-diff, go, diff-engine, yaml, japanese-output]

# Dependency graph
requires:
  - phase: 10-01-schema-diff
    provides: internal/diff package skeleton with RED tests, DiffResult/ObjectChange types
  - phase: 08-restore-core-pipeline
    provides: resolve.ReadManifest and restore/loader.go serializer patterns
  - phase: 05-serialization
    provides: all serializer packages (serialize/view, serialize/table, etc.)
provides:
  - loadBackup reading _manifest.yaml via resolve.ReadManifest and deserializing all schema objects
  - compareObjects detecting Added/Removed/Changed across two object maps
  - diffTable with column/index/constraint/RLS change detection in Japanese
  - diffNonTable via YAML byte comparison with kind-appropriate Japanese labels
  - formatReport producing spec 9.3 output with three conditional sections
  - Run() wiring all three functions as diff engine entry point
affects: [10-03-schema-diff-cli]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Duplicate kindToDir/defYAMLPath/serializerFor from restore/loader.go — no import of restore package"
    - "YAML byte comparison via go.yaml.in/yaml/v3 Marshal for non-table diffNonTable"
    - "Post-processing pass moves TriggerDef/PolicyDef Added/Removed into owning table Changed.Details"

key-files:
  created: []
  modified:
    - internal/diff/loader.go
    - internal/diff/compare.go
    - internal/diff/report.go
    - internal/diff/diff.go
    - internal/diff/diff_test.go

key-decisions:
  - "Duplicate kindToDir/defYAMLPath/serializerFor verbatim from restore/loader.go — established pattern per RESEARCH.md"
  - "diffNonTable uses YAML marshal byte comparison over serializer dispatch — simpler, avoids needing ObjectDef kind for dispatch"
  - "Trigger/policy post-processing pass in compareObjects — moves attached objects into table's Changed.Details when table exists in both backups"
  - "Auto-fix: test file used manifest.yaml; corrected to _manifest.yaml per system-wide convention"

patterns-established:
  - "schema.kind.name key format in loadBackup for all entries (matches diff_test.go fixture keys)"
  - "Japanese change labels: カラム追加/削除/変更, インデックス追加/削除, UNIQUE追加/削除, CHECK追加/削除, RLS変更, 本体変更あり, 定義変更あり, 設定変更あり"
  - "spec 9.3 formatReport: 比較 header, blank line + [セクション] + 2-space indent for entries, 4-space indent for details"

requirements-completed: [DIFF-01, DIFF-02, DIFF-03, DIFF-04, DIFF-05]

# Metrics
duration: 2min
completed: 2026-03-12
---

# Phase 10 Plan 02: Schema Diff Engine Implementation Summary

**Full diff engine turning all five RED tests GREEN: loadBackup reading typed object maps, compareObjects detecting structural changes with Japanese labels, formatReport producing spec 9.3 output**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-12T08:25:09Z
- **Completed:** 2026-03-12T08:27:56Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Implemented `loadBackup` reading `_manifest.yaml` and deserializing all schema objects into `map[string]core.ObjectDef` keyed `schema.kind.name`
- Implemented `compareObjects` detecting additions, removals, and changes with full table-level structural diff (columns, indexes, constraints, RLS)
- Implemented `diffNonTable` via YAML byte comparison returning Japanese change labels per object kind
- Implemented `formatReport` writing spec 9.3 output with correct Japanese section headers and indentation
- Wired `Run()` calling all three functions in sequence
- All five tests in `diff_test.go` are GREEN with no regressions in the full suite

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement loader and compare engine (go GREEN on DIFF-01 through DIFF-04)** - `42a65a9` (feat)
2. **Task 2: Implement report formatter and wire Run() (go GREEN on DIFF-05)** - `a50d8c1` (feat)

## Files Created/Modified

- `internal/diff/loader.go` - loadBackup reading _manifest.yaml and deserializing via existing serializers; kindToDir/defYAMLPath/serializerFor duplicated from restore/loader.go
- `internal/diff/compare.go` - DiffResult/ObjectChange types, compareObjects, diffObject, diffTable, diffNonTable, diffColumns/Indexes/Constraints helpers
- `internal/diff/report.go` - formatReport writing spec 9.3 format with Japanese section headers and indented entries
- `internal/diff/diff.go` - Run() wiring loadBackup + compareObjects + formatReport
- `internal/diff/diff_test.go` - Auto-fix: corrected manifest filename from manifest.yaml to _manifest.yaml

## Decisions Made

- `diffNonTable` uses `yaml.Marshal` byte comparison rather than serializer dispatch — simpler and avoids needing to look up serializer by ObjectDef type at runtime; uses `aHeader.Kind` for selecting the Japanese label
- Trigger/policy post-processing pass in `compareObjects` moves attached objects into their owning table's `Changed.Details` when the table exists in both backups, matching spec intent that table-attached objects appear as table-level changes

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed manifest filename in test from manifest.yaml to _manifest.yaml**
- **Found during:** Task 1 (loader implementation)
- **Issue:** diff_test.go wrote `manifest.yaml` but the plan and system convention use `_manifest.yaml`; implementing loader with `_manifest.yaml` would have caused the "empty directory" test branch to fail
- **Fix:** Changed `filepath.Join(dir, "manifest.yaml")` to `filepath.Join(dir, "_manifest.yaml")` in diff_test.go
- **Files modified:** internal/diff/diff_test.go
- **Verification:** TestLoadBackup/empty_directory_returns_empty_map passes with _manifest.yaml loader
- **Committed in:** 42a65a9 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug in test file)
**Impact on plan:** Necessary for correctness — test file had wrong filename vs. system convention. No scope creep.

## Issues Encountered

None — all plan instructions were clear and executable.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/diff.Run()` is implemented and callable
- All DIFF-01 through DIFF-05 requirements are GREEN
- Plan 03 (CLI wiring) can now import and call `diff.Run()` to expose `pgbkrs diff` subcommand
- No blockers

---
*Phase: 10-schema-diff*
*Completed: 2026-03-12*
