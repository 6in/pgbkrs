---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: executing
stopped_at: Completed 03-schema-fetch-views-functions-triggers-policies/03-01-PLAN.md
last_updated: "2026-03-11T08:39:34.426Z"
last_activity: 2026-03-11 — Roadmap created, 47 requirements mapped across 10 phases
progress:
  total_phases: 10
  completed_phases: 2
  total_plans: 10
  completed_plans: 7
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-11)

**Core value:** PostgreSQLのスキーマ構造とデータを人間が読めるYAML+CSV形式でオブジェクト単位にバックアップし、依存関係を自動解決して正しい順序でリストアできること
**Current focus:** Phase 3 - Schema Fetch (Views, Functions, Triggers, Policies)

## Current Position

Phase: 3 of 10 (Schema Fetch - Views/Functions/Triggers/Policies)
Plan: 1 of 4 in current phase
Status: Executing
Last activity: 2026-03-11 — Completed 03-01 types expansion + test stubs

Progress: [███████░░░] 70%

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
| Phase 02-schema-fetch-tables-sequences P01 | 2 | 2 tasks | 7 files |
| Phase 02-schema-fetch-tables-sequences P03 | 5 | 2 tasks | 4 files |
| Phase 02-schema-fetch-tables-sequences P04 | 2 | 2 tasks | 4 files |
| Phase 02-schema-fetch-tables-sequences P02 | 2 | 1 task | 2 files |
| Phase 03-schema-fetch-views-functions-triggers-policies P01 | 3 | 2 tasks | 8 files |

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
- [Phase 02-schema-fetch-tables-sequences]: TableDef fully-specified with ColumnDef/ConstraintsDef/IndexDef/PartitionDef/RLSDef; SequenceDef with int64 numeric fields + IsCalled bool; TypeDef with Fields []CompositeField; DomainDef with BaseType/Nullable/Default/CheckName/CheckDefinition; EnumDef with Labels []string
- [Phase 02-schema-fetch-tables-sequences]: Test-first scaffolding: integration test stubs written before production fetcher code (Nyquist); TestFetchXxx stubs reference forward-declared SchemaFetcher until Wave 2
- [Phase 02-schema-fetch-tables-sequences]: pg_sequences view over pg_sequence catalog: exposes last_value and is_called for never-advanced sequences via COALESCE
- [Phase 02-schema-fetch-tables-sequences]: pgx v5 scans PostgreSQL text[] from array_agg directly into []string — no pgtype.Array wrapper needed for enum labels
- [Phase 02-schema-fetch-tables-sequences]: External test packages (compositetype_test, domain_test) reference SchemaFetcher via package-qualified name requiring explicit import; composite type field grouping done in Go using ordered query output
- [Phase 02-schema-fetch-tables-sequences]: TableSchemaFetcher uses N+1 query pattern (columns/constraints/indexes/partitioning per table) — acceptable for CLI tool; PK columns parsed from pg_get_constraintdef output rather than conkey join
- [Phase 03-schema-fetch-views-functions-triggers-policies]: Phase 3 DDL added as separate fmt.Sprintf block (ddl2) with %[1]s positional args to avoid numbering conflicts with existing Phase 2 DDL
- [Phase 03-schema-fetch-views-functions-triggers-policies]: TDD RED/GREEN for struct expansion: types_phase3_test.go verifies all new fields compile and are accessible

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Session Continuity

Last session: 2026-03-11T08:39:00Z
Stopped at: Completed 03-schema-fetch-views-functions-triggers-policies/03-01-PLAN.md
Resume file: None
