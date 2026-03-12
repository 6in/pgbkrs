---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: executing
stopped_at: Completed 10-schema-diff/10-03-PLAN.md
last_updated: "2026-03-12T08:33:17.568Z"
last_activity: 2026-03-11 — Completed 06-02 Manifest Writer
progress:
  total_phases: 10
  completed_phases: 10
  total_plans: 30
  completed_plans: 30
  percent: 95
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-11)

**Core value:** PostgreSQLのスキーマ構造とデータを人間が読めるYAML+CSV形式でオブジェクト単位にバックアップし、依存関係を自動解決して正しい順序でリストアできること
**Current focus:** Phase 6 - Dependency Resolution & Manifest

## Current Position

Phase: 6 of 10 (Dependency Resolution & Manifest)
Plan: 2 of 2 in current phase
Status: Executing
Last activity: 2026-03-11 — Completed 06-02 Manifest Writer

Progress: [██████████] 95%

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
| Phase 03 P04 | 1 | 1 tasks | 1 files |
| Phase 03 P03 | 2 | 2 tasks | 2 files |
| Phase 03 P02 | 1 | 2 tasks | 2 files |
| Phase 04-ddl-generation P01 | 4 | 2 tasks | 26 files |
| Phase 04-ddl-generation P02 | 2 | 2 tasks | 4 files |
| Phase 04-ddl-generation P03 | 2 | 2 tasks | 8 files |
| Phase 04-ddl-generation P04 | 3 | 2 tasks | 10 files |
| Phase 05 P01 | 3 | 2 tasks | 25 files |
| Phase 05 P02 | 5 | 2 tasks | 20 files |
| Phase 05-serialization P03 | 2 | 2 tasks | 2 files |
| Phase 06-dependency-resolution-manifest P01 | 3 | 2 tasks | 4 files |
| Phase 06 P02 | 3 | 2 tasks | 3 files |
| Phase 07-backup-orchestration P01 | 3 | 2 tasks | 5 files |
| Phase 07-backup-orchestration P02 | 3 | 2 tasks | 3 files |
| Phase 08-restore-core-pipeline P01 | 3 | 2 tasks | 7 files |
| Phase 08-restore-core-pipeline P02 | 8 | 2 tasks | 4 files |
| Phase 08-restore-core-pipeline P03 | 1 | 2 tasks | 1 files |
| Phase 09 P01 | 25 | 2 tasks | 8 files |
| Phase 09-restore-partial-safety P02 | 20 | 2 tasks | 3 files |
| Phase 09-restore-partial-safety P03 | 5 | 1 tasks | 1 files |
| Phase 10-schema-diff P01 | 2 | 1 tasks | 5 files |
| Phase 10-schema-diff P02 | 2 | 2 tasks | 5 files |
| Phase 10-schema-diff P03 | 2 | 1 tasks | 2 files |

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
- [Phase 03]: Followed plan exactly: single SQL query with LEFT JOIN LATERAL unnest for role OID resolution
- [Phase 03]: pg_get_functiondef() for complete CREATE OR REPLACE FUNCTION DDL rather than reconstructing from pg_proc columns
- [Phase 03]: tgtype bitmask decoded in Go with deterministic event ordering: INSERT, UPDATE, DELETE, TRUNCATE
- [Phase 03]: Canonical fetcher pattern applied directly to pg_views and pg_matviews -- no query complexity needed for system view-based introspection
- [Phase 04-ddl-generation]: ForeignKeyDef as independent ObjectDef with embedded ObjectHeader, not nested in TableDef
- [Phase 04-ddl-generation]: PartitionDef.KeyExpression populated via separate pg_get_partkeydef query in fetchPartitioning
- [Phase 04-ddl-generation]: DDL generator stubs return fmt.Errorf not-implemented to make tests compilable but RED (Nyquist Wave 0)
- [Phase 04-ddl-generation]: Items slice for comma-separated SQL body elements avoids trailing comma bugs
- [Phase 04-ddl-generation]: PARTITION BY appended outside parentheses with uppercase strategy per PostgreSQL syntax
- [Phase 04-ddl-generation]: Value receiver type assertions (core.ViewDef not *core.ViewDef) matching Header() receiver pattern
- [Phase 04-ddl-generation]: Function DDL is pure passthrough of pg_get_functiondef output -- no reconstruction
- [Phase 04-ddl-generation]: Trigger defaults to FOR EACH ROW (TriggerDef lacks row/statement granularity)
- [Phase 04-ddl-generation]: Trigger schema applied to both table and function references (same-schema assumption)
- [Phase 04-ddl-generation]: Value receiver type assertions for all 5 remaining generators (SequenceDef, TypeDef, DomainDef, EnumDef, PolicyDef)
- [Phase 04-ddl-generation]: Policy DROP omits CASCADE per PostgreSQL semantics (policies do not cascade)
- [Phase 04-ddl-generation]: Domain CheckDefinition passed through verbatim from pg_get_constraintdef (already contains CHECK (...))
- [Phase 05-serialization]: Intermediate YAML structs in serializer.go decouple YAML format from core types (no yaml tags on core)
- [Phase 05-serialization]: nullable field always emitted (no omitempty) since false is semantically meaningful
- [Phase 05-serialization]: Pointer type assertion (*core.TableDef) for table serializer matching DDL generator convention
- [Phase 05-serialization]: Index YAML includes name and method only; Definition is for DDL passthrough, not YAML output
- [Phase 05]: Value type assertions for all serializers except ForeignKey (pointer) matching DDL generator convention
- [Phase 05]: No omitempty on boolean fields where false is meaningful (nullable, is_populated, cycle, is_called)
- [Phase 05-serialization]: Quoted schema/table identifiers in COPY command for mixed-case safety
- [Phase 05-serialization]: Single-pass streaming checksum via io.MultiWriter(file, hash) for data export
- [Phase 06-dependency-resolution-manifest]: Two-pass DAG: AddObject registers nodes/customTypes, Resolve builds edges -- avoids ordering issues
- [Phase 06-dependency-resolution-manifest]: Kind-priority tiebreaking in Kahn's sort instead of explicit view->table edges -- simpler and more robust
- [Phase 06-dependency-resolution-manifest]: inDegree tracks dependency count per node (zero = ready to restore first)
- [Phase 06-dependency-resolution-manifest]: FKs sorted alphabetically by schema.name when appended for deterministic output
- [Phase 06]: Object IDs use schema.name format in manifest (not schema.kind.name used internally by DAG)
- [Phase 06]: BuildManifest delegates to BuildRestoreOrder as single authoritative sort source, builds separate DAG only for edge reading
- [Phase 06]: FK ObjectEntry uses kind=fk with DependsOn listing both source and target tables
- [Phase 07-backup-orchestration]: TableDataMeta defined in core (not export) to avoid import cycles — orchestrator imports both core and export packages
- [Phase 07-backup-orchestration]: skipReason() uses strings.Contains to detect array variants (bytea[] matches bytea check); pointer receiver required for fetchTable.SchemaFetcher
- [Phase 07-backup-orchestration]: queryConn threads tx.Conn() or conn through all fetches/COPY TO — single change point for snapshot mode without altering function signatures
- [Phase 07-backup-orchestration]: Integration tests skip on absent TEST_DATABASE_URL rather than fail — supports CI without live DB requirement
- [Phase 08-restore-core-pipeline]: ReadManifest added alongside WriteManifest in manifest.go as symmetric read counterpart
- [Phase 08-restore-core-pipeline]: yamlIndex.definition uses omitempty; loader_test.go uses package restore for unexported function testing; orchestrator_test.go uses package restore_test matching backup integration pattern
- [Phase 08-restore-core-pipeline]: FK create wave re-fetches FKs from live DB — FKs have no def.yaml in backup directory
- [Phase 08-restore-core-pipeline]: manifest 'fk' string handled explicitly in ddlGeneratorFor; table CREATE wave executes only stmts[0]; SETVAL runs after COPY FROM
- [Phase 08-restore-core-pipeline]: MarkFlagRequired('input') on restoreCmd makes cobra return error if --input absent; --pre-backup-dir defaults to '.' matching backup convention
- [Phase 09]: Options struct defined in filter.go (owns filter types), orchestrator uses it in same package — avoids circular dep
- [Phase 09]: snapshotLiveObjects and Logger creation failures are non-fatal: log stderr and continue (graceful degradation)
- [Phase 09]: FK post-BFS pass uses len(DependsOn)>0 guard to exclude vacuous FK inclusions in empty closures
- [Phase 09-restore-partial-safety]: export_test.go pattern used to expose filteredRestoreOrder for black-box unit testing without exporting it publicly
- [Phase 09-restore-partial-safety]: TestMissingDependencyError constructs minimal resolve.Manifest directly for pure unit test of REST-12 without DB dependency
- [Phase 09-restore-partial-safety]: Flag registration placed in init() after existing flags; MarkFlagRequired kept at end; placeholder comment removed
- [Phase 10-schema-diff]: White-box test package (package diff) for access to unexported functions; test keys use schema.kind.name format; DiffResult.Changed carries []string Details for human-readable per-object change lines
- [Phase 10-schema-diff]: diffNonTable uses yaml.Marshal byte comparison over serializer dispatch — simpler, avoids runtime serializer lookup by ObjectDef type
- [Phase 10-schema-diff]: Trigger/policy post-processing pass in compareObjects moves attached objects into table Changed.Details when table exists in both backups
- [Phase 10-schema-diff]: Auto-fix: diff_test.go manifest filename corrected from manifest.yaml to _manifest.yaml per system-wide convention
- [Phase 10-schema-diff]: cmd.Name() == diff guard in PersistentPreRunE skips DB connection for disk-only diff subcommand

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Session Continuity

Last session: 2026-03-12T08:33:17.566Z
Stopped at: Completed 10-schema-diff/10-03-PLAN.md
Resume file: None
