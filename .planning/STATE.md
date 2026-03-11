---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: planning
stopped_at: Completed 01-foundation/01-02-PLAN.md
last_updated: "2026-03-11T07:07:29.703Z"
last_activity: 2026-03-11 — Roadmap created, 47 requirements mapped across 10 phases
progress:
  total_phases: 10
  completed_phases: 1
  total_plans: 2
  completed_plans: 2
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-11)

**Core value:** PostgreSQLのスキーマ構造とデータを人間が読めるYAML+CSV形式でオブジェクト単位にバックアップし、依存関係を自動解決して正しい順序でリストアできること
**Current focus:** Phase 1 - Foundation

## Current Position

Phase: 1 of 10 (Foundation)
Plan: 0 of TBD in current phase
Status: Ready to plan
Last activity: 2026-03-11 — Roadmap created, 47 requirements mapped across 10 phases

Progress: [░░░░░░░░░░] 0%

## Performance Metrics

**Velocity:**
- Total plans completed: 0
- Average duration: -
- Total execution time: 0 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| - | - | - | - |

**Recent Trend:**
- Last 5 plans: none yet
- Trend: -

*Updated after each plan completion*
| Phase 01-foundation P01 | 3 | 2 tasks | 7 files |
| Phase 01-foundation P02 | 2 | 2 tasks | 9 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Initial: サブコマンド方式（1バイナリ） — 配布・管理が容易
- Initial: FK制約を独立オブジェクトとして分離 — 循環FK問題を根本的に回避
- Initial: Kahn法トポロジカルソート — 安定した依存解決、循環検知が容易
- Initial: bytea等のカラムを含むテーブルをスキップ — CSVで安全に扱えない型を排除
- [Phase 01-foundation]: Go module path github.com/pgbkrs/pgbackup, go.yaml.in/yaml/v3 over archived gopkg.in, internal/core as shared types anchor with no internal imports
- [Phase 01-foundation]: PersistentPreRunE on rootCmd for fail-fast DB connection inherited by all subcommands
- [Phase 01-foundation]: Package-level *pgx.Conn in cmd package (not context.Context) for simplicity in sequential CLI
- [Phase 01-foundation]: SilenceUsage/SilenceErrors on rootCmd — DB errors should not dump usage to screen

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Session Continuity

Last session: 2026-03-11T07:04:47.186Z
Stopped at: Completed 01-foundation/01-02-PLAN.md
Resume file: None
