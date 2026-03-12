# Roadmap: pgbackup

## Overview

Build a PostgreSQL backup/restore CLI tool in Go that exports schema definitions as YAML and table data as CSV, resolves object dependencies via DAG+Kahn topological sort, and restores in the correct order. The journey proceeds from project scaffold through schema fetching, DDL generation, serialization, dependency resolution, and backup orchestration, then into the restore pipeline and finally schema diff.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation** - Project scaffold, CLI subcommands, DB connection, command pattern interfaces (completed 2026-03-11)
- [ ] **Phase 2: Schema Fetch - Tables & Sequences** - pg_catalog extraction for tables, sequences, composite types, domains, and ENUMs
- [x] **Phase 3: Schema Fetch - Views, Functions, Triggers & Policies** - pg_catalog extraction for views, materialized views, functions, triggers, and RLS policies (completed 2026-03-11)
- [x] **Phase 4: DDL Generation** - CREATE and DROP DDL generation from fetched definitions for all object types (completed 2026-03-11)
- [ ] **Phase 5: Serialization** - YAML output for all object types, CSV data export, partition handling, checksums
- [x] **Phase 6: Dependency Resolution & Manifest** - In-memory DAG, Kahn topological sort, FK isolation, _manifest.yaml output (completed 2026-03-11)
- [x] **Phase 7: Backup Orchestration** - End-to-end backup command: directory structure, skip detection, snapshot mode (completed 2026-03-11)
- [x] **Phase 8: Restore - Core Pipeline** - Pre-backup, DROP cycle, CREATE cycle, data load, sequence restoration, indexes, FK application (completed 2026-03-11)
- [x] **Phase 9: Restore - Partial & Safety** - Granular restore (schema/object scope), dependency chain resolution, drop-leak detection, restore logs (completed 2026-03-11)
- [ ] **Phase 10: Schema Diff** - Compare two backups at schema level, detect additions/deletions/modifications, formatted output

## Phase Details

### Phase 1: Foundation
**Goal**: Developers can build and run the `pgbackup` binary with all three subcommands, connect to PostgreSQL, and the command pattern infrastructure is in place for registering new object handlers with a single line
**Depends on**: Nothing (first phase)
**Requirements**: FOUND-01, FOUND-02, FOUND-03, FOUND-04
**Success Criteria** (what must be TRUE):
  1. Running `pgbackup backup --host localhost --port 5432 --user u --dbname d` opens a PostgreSQL connection without error
  2. Running `pgbackup restore` and `pgbackup diff` are recognized as valid subcommands (no "unknown command" error)
  3. A new object type can be registered in CommandRegistry with one `Register()` call and is immediately resolvable
  4. `SchemaFetcher`, `Serializer`, and `DDLGenerator` interfaces exist and the build passes
**Plans**: 2 plans

Plans:
- [ ] 01-01-PLAN.md — Go module init + internal/core interfaces, types, registry, and unit tests
- [ ] 01-02-PLAN.md — internal/db connection helper, cobra CLI scaffold (backup/restore/diff), Makefile

### Phase 2: Schema Fetch - Tables & Sequences
**Goal**: The tool can query pg_catalog and return complete in-memory definitions for tables (including columns, constraints, indexes, RLS, partitioning), sequences (including current value), composite types, domains, and ENUMs
**Depends on**: Phase 1
**Requirements**: FETCH-01, FETCH-06, FETCH-07, FETCH-08, FETCH-09
**Success Criteria** (what must be TRUE):
  1. Fetching a table returns columns with types, nullability, defaults, primary key, unique constraints, check constraints, indexes, and RLS flag
  2. Fetching a partitioned table returns partition strategy and lists child tables
  3. Fetching a sequence returns its definition and current value (last_value)
  4. Fetching a composite type, domain, and ENUM each return their full pg_catalog-sourced definition
**Plans**: 4 plans

Plans:
- [ ] 02-01-PLAN.md — Expand *Def structs in internal/core/types.go + all integration test scaffolds (Wave 1)
- [ ] 02-02-PLAN.md — internal/fetch/table: TableSchemaFetcher with columns, constraints, indexes, partitioning, RLS (Wave 2)
- [ ] 02-03-PLAN.md — internal/fetch/sequence + internal/fetch/enum: SequenceSchemaFetcher and EnumSchemaFetcher (Wave 2, parallel)
- [ ] 02-04-PLAN.md — internal/fetch/compositetype + internal/fetch/domain: CompositeTypeSchemaFetcher and DomainSchemaFetcher (Wave 2, parallel)

### Phase 3: Schema Fetch - Views, Functions, Triggers & Policies
**Goal**: The tool can retrieve complete in-memory definitions for views, materialized views, functions (via pg_get_functiondef), triggers, and RLS policies from pg_catalog
**Depends on**: Phase 1
**Requirements**: FETCH-02, FETCH-03, FETCH-04, FETCH-05, FETCH-10
**Success Criteria** (what must be TRUE):
  1. Fetching a view returns its definition SQL and owner
  2. Fetching a materialized view returns its definition SQL and whether it is populated
  3. Fetching a function returns the body via `pg_get_functiondef` and its argument/return types
  4. Fetching a trigger returns the event, timing, function reference, and target table
  5. Fetching RLS policies for a table returns each policy's name, command, roles, and USING/WITH CHECK expressions
**Plans**: 4 plans

Plans:
- [ ] 03-01-PLAN.md — Expand *Def structs in types.go + extend testhelpers/setup.go + all 5 integration test stubs (Wave 1)
- [ ] 03-02-PLAN.md — internal/fetch/view + internal/fetch/matview: ViewSchemaFetcher and MatViewSchemaFetcher (Wave 2, parallel)
- [ ] 03-03-PLAN.md — internal/fetch/function + internal/fetch/trigger: FunctionSchemaFetcher and TriggerSchemaFetcher (Wave 2, parallel)
- [ ] 03-04-PLAN.md — internal/fetch/policy: PolicySchemaFetcher with polroles unnest and PUBLIC sentinel handling (Wave 2, parallel)

### Phase 4: DDL Generation
**Goal**: Given an in-memory object definition, the tool can produce correct CREATE DDL (excluding FK constraints for tables), separate FK ALTER TABLE DDL, and DROP DDL for every supported object type
**Depends on**: Phase 2, Phase 3
**Requirements**: DDLG-01, DDLG-02, DDLG-03, DDLG-04
**Success Criteria** (what must be TRUE):
  1. Table CREATE DDL is generated without FK constraints and can be executed against PostgreSQL to create the table
  2. FK ALTER TABLE DDL is generated separately and applies all foreign key constraints when executed after all tables exist
  3. DROP DDL is generated for every object type (table, view, materialized view, function, trigger, sequence, type, domain, ENUM, policy)
  4. CREATE DDL for views, materialized views, functions, and triggers is generated and syntactically valid
**Plans**: 4 plans

Plans:
- [ ] 04-01-PLAN.md — ForeignKeyDef type + PartitionDef.KeyExpression + FK SchemaFetcher + all 11 DDL generator test stubs (Wave 1)
- [ ] 04-02-PLAN.md — Table and FK DDL generators: CREATE TABLE (no FK), ALTER TABLE ADD/DROP CONSTRAINT (Wave 2)
- [ ] 04-03-PLAN.md — View, MatView, Function, Trigger DDL generators: CREATE + DROP (Wave 2, parallel)
- [ ] 04-04-PLAN.md — Sequence, CompositeType, Domain, Enum, Policy DDL generators: CREATE + DROP (Wave 2, parallel)

### Phase 5: Serialization
**Goal**: The tool can write each object's in-memory definition to a YAML file conforming to the spec, export table data as COPY TO CSV, handle partition parent/child split, and record checksum and row count in each data file's metadata
**Depends on**: Phase 2, Phase 3, Phase 4
**Requirements**: SRLZ-01, SRLZ-02, SRLZ-03, SRLZ-04, SRLZ-05
**Success Criteria** (what must be TRUE):
  1. A table def.yaml is written with all fields matching spec section 5.2 (columns, constraints, indexes, owner, RLS, partition info)
  2. View, materialized view, function, trigger, sequence, type, domain, ENUM, and policy each produce a correctly structured def.yaml
  3. A table's data.csv is written in COPY TO format and can be loaded back via COPY FROM without error
  4. For a partitioned table, the parent produces no data.csv and each child table produces its own data.csv
  5. Each data.csv is accompanied by a sha256 checksum and row count in the manifest metadata
**Plans**: 3 plans

Plans:
- [ ] 05-01-PLAN.md — Wave 0 test stubs for all serializers + table YAML serializer (spec 5.2)
- [ ] 05-02-PLAN.md — All 10 remaining object serializers (view, matview, function, trigger, sequence, type, domain, enum, policy, foreignkey)
- [ ] 05-03-PLAN.md — Data exporter: COPY TO CSV with streaming SHA256 checksum, partition detection

### Phase 6: Dependency Resolution & Manifest
**Goal**: Given a set of fetched object definitions, the tool builds a dependency DAG, resolves restore order via Kahn's algorithm, isolates FK constraints as independent objects to break circular dependencies, and writes a complete _manifest.yaml
**Depends on**: Phase 2, Phase 3
**Requirements**: DEPS-01, DEPS-02, DEPS-03, DEPS-04
**Success Criteria** (what must be TRUE):
  1. Objects with dependencies are ordered so that dependencies always appear before dependents in restore_order
  2. Circular FK dependencies do not cause an error — FK constraints are separated into independent objects ordered after all tables
  3. _manifest.yaml is written containing object list, restore_order, dependency graph, skip list, and backup metadata
  4. Kahn's algorithm detects and reports a true (non-FK) circular dependency as an error rather than silently producing a wrong order
**Plans**: 2 plans

Plans:
- [ ] 06-01-PLAN.md — DAG builder with dependency extraction + Kahn's topological sort with FK isolation and cycle detection
- [ ] 06-02-PLAN.md — Manifest struct, BuildManifest, WriteManifest, and end-to-end integration tests

### Phase 7: Backup Orchestration
**Goal**: Running `pgbackup backup` against a live database produces a complete, correctly structured backup directory that can be used for restore and diff operations
**Depends on**: Phase 5, Phase 6
**Requirements**: BKUP-01, BKUP-02, BKUP-03, BKUP-04
**Success Criteria** (what must be TRUE):
  1. The backup directory contains one subdirectory per schema, each with object subdirectories matching spec section 4
  2. Tables containing bytea, xml, pg_lsn, or txid_snapshot columns are skipped and appear in a warnings log
  3. Running with `--snapshot` wraps all fetching in a single transaction (REPEATABLE READ) and the manifest records snapshot mode
  4. Running without `--snapshot` (default) fetches each object independently and the manifest records non-snapshot mode
**Plans**: 2 plans

Plans:
- [ ] 07-01-PLAN.md — Test stubs + skip detection helper + core.TableDef DataMeta field + table serializer update + RunBackup orchestrator (non-snapshot path)
- [ ] 07-02-PLAN.md — Snapshot mode (RepeatableRead tx + tx.Conn()) + CLI flag wiring (--snapshot, --output) + integration tests

### Phase 8: Restore - Core Pipeline
**Goal**: Running `pgbackup restore` against a backup directory fully reconstructs the database — dropping existing objects in reverse order, creating them in dependency order, loading data, restoring sequence values, creating indexes, and applying FK constraints as a final batch
**Depends on**: Phase 7
**Requirements**: REST-01, REST-02, REST-04, REST-05, REST-06, REST-07, REST-08, REST-09
**Success Criteria** (what must be TRUE):
  1. Before restore begins, a pre-restore backup is automatically taken
  2. Objects are dropped in reverse restore_order and re-created in restore_order; a table restored before its dependency causes an error
  3. Table data is loaded via COPY FROM for regular tables and each partition child table
  4. Sequences are restored to their backed-up value using SETVAL
  5. Indexes are created after all tables and data are loaded
  6. FK constraints are applied in a single batch after all tables and data exist
  7. Views, functions, triggers, and policies are created in their manifest-specified order
**Plans**: 3 plans

Plans:
- [ ] 08-01-PLAN.md — Wave 0: ReadManifest + yamlIndex.definition gap fix + test stubs (Wave 1)
- [ ] 08-02-PLAN.md — restore/loader.go + restore/orchestrator.go: full 7-wave pipeline (Wave 2)
- [ ] 08-03-PLAN.md — CLI wiring (--input, --pre-backup-dir) + human verify checkpoint (Wave 3)

### Phase 9: Restore - Partial & Safety
**Goal**: Users can restore a subset of objects (by schema or by individual object name), the tool automatically pulls in transitive dependencies, stops on missing dependencies, detects objects that survived the DROP phase, and writes structured logs for every restore operation
**Depends on**: Phase 8
**Requirements**: REST-03, REST-10, REST-11, REST-12, REST-13
**Success Criteria** (what must be TRUE):
  1. Running `pgbackup restore --schema myschema` restores only objects in that schema
  2. Running `pgbackup restore --object myschema.mytable` restores that object plus all transitive dependencies
  3. If a dependency object is not present in the backup, restore stops with a clear error identifying the missing object
  4. After the DROP phase, any object that still exists in the database is reported as a drop-leak warning
  5. drop.log, restore.log, and summary.log are written at the end of every restore run
**Plans**: 3 plans

Plans:
- [ ] 09-01-PLAN.md — filter.go + leak.go + logger.go + Options struct refactor (Wave 1)
- [ ] 09-02-PLAN.md — Wire filter/leak/logger into RunRestore pipeline + integration tests (Wave 2)
- [ ] 09-03-PLAN.md — CLI flags --schema, --object, --log-dir + human verify checkpoint (Wave 3)

### Phase 10: Schema Diff
**Goal**: Running `pgbackup diff <backup-a> <backup-b>` compares the two backup directories at the schema level and prints a structured, human-readable diff showing added, removed, and modified objects
**Depends on**: Phase 7
**Requirements**: DIFF-01, DIFF-02, DIFF-03, DIFF-04, DIFF-05
**Success Criteria** (what must be TRUE):
  1. Objects present in backup-b but not backup-a are reported as additions; objects missing from backup-b are reported as deletions
  2. Table changes show column additions/deletions/type changes, NULL constraint changes, default value changes, index changes, constraint changes, trigger changes, and RLS changes
  3. Views, materialized views, functions, sequences, types, domains, and ENUMs each report whether they changed (changed/unchanged) without line-level diffs
  4. The diff output follows spec section 9.3 formatting and is printed to stdout
**Plans**: 3 plans

Plans:
- [ ] 10-01-PLAN.md — Wave 0 test stubs for all five DIFF requirements + internal/diff package skeletons
- [ ] 10-02-PLAN.md — Full diff engine: loader, compareObjects, diffTable, diffNonTable, formatReport
- [ ] 10-03-PLAN.md — CLI wiring (diffCmd positional args, PersistentPreRunE DB skip) + human verify checkpoint

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 2/2 | Complete   | 2026-03-11 |
| 2. Schema Fetch - Tables & Sequences | 4/4 | Complete   | 2026-03-11 |
| 3. Schema Fetch - Views, Functions, Triggers & Policies | 4/4 | Complete   | 2026-03-11 |
| 4. DDL Generation | 4/4 | Complete   | 2026-03-11 |
| 5. Serialization | 0/3 | Not started | - |
| 6. Dependency Resolution & Manifest | 2/2 | Complete   | 2026-03-11 |
| 7. Backup Orchestration | 2/2 | Complete   | 2026-03-11 |
| 8. Restore - Core Pipeline | 3/3 | Complete   | 2026-03-11 |
| 9. Restore - Partial & Safety | 3/3 | Complete   | 2026-03-11 |
| 10. Schema Diff | 1/3 | In Progress|  |
