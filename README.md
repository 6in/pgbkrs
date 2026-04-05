# pgbackup

PostgreSQL schema backup, restore, and diff tool. Exports schema definitions as YAML and data as CSV, enabling object-level restore and schema comparison between backups.

## Installation

```bash
git clone https://github.com/pgbkrs/pgbackup
cd pgbackup
go build -o pgbackup ./cmd/pgbackup
```

## Connection Flags

All subcommands (except `diff` and `data-diff`) require a database connection:

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `localhost` | PostgreSQL host |
| `--port` | `5432` | PostgreSQL port |
| `--user` | | PostgreSQL user |
| `--password` | | PostgreSQL password |
| `--dbname` | | Target database name |

## Commands

> Run `pgbackup <command> --help` to see all available flags for each subcommand.

### `backup`

Back up a PostgreSQL database to a timestamped directory of YAML and CSV files.

```bash
pgbackup backup --host localhost --dbname mydb --output /backups
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `.` | Directory where the timestamped backup folder will be created |
| `--snapshot` | false | Use `REPEATABLE READ` transaction for a consistent snapshot across all reads |

The `--snapshot` flag guarantees data consistency across all objects at the cost of holding a long-running transaction. Without it, objects are fetched individually (faster, but without strict consistency).

### `backup-ai`

Back up a PostgreSQL database in a format optimised for AI consumption.

```bash
pgbackup backup-ai --host localhost --dbname mydb --output /backups
```

Produces the same schema YAML files as `backup`, but with two key differences:

- **Sampled CSV data:** each table exports at most 10 rows, with a column header row. Useful for giving an AI a representative sample without flooding its context window.
- **README.md:** written at the backup root, summarising the database structure — table list, column counts, primary keys, foreign key relationships, and skipped tables — along with reading instructions for an AI assistant.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `.` | Directory where the timestamped backup folder will be created |
| `--snapshot` | false | Use `REPEATABLE READ` transaction for a consistent snapshot across all reads |

### `restore`

Restore a PostgreSQL database from a backup directory.

```bash
pgbackup restore --host localhost --dbname mydb --input /backups/backup_20240101_120000
```

Before restoring, a safety backup of the target database is automatically written to the directory specified by `--pre-backup-dir`.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--input` | **(required)** | Path to the backup directory to restore from |
| `--pre-backup-dir` | `.` | Directory where the pre-restore safety backup will be written |
| `--schema` | | Restore only objects in this schema (e.g. `myschema`) |
| `--object` | | Restore this object and its transitive dependencies (`schema.name` format, e.g. `public.mytable`) |
| `--log-dir` | `.` | Directory for restore log files |

**Partial restore:** use `--schema` to restore a single schema, or `--object` to restore one object and all objects it depends on. If a dependency is missing from the target database, restore stops with an error.

Restore logs are written to a `restore_YYYYMMDD_HHMMSS/` directory containing `drop.log`, `restore.log`, and `summary.log`.

### `restore-tui`

Interactively select schemas and tables to restore from a backup using a terminal UI.

```bash
pgbackup restore-tui --host localhost --dbname mydb --input /backups/backup_20240101_120000
```

The TUI displays all schemas and tables found in the backup as a checkbox list. Navigate and select what to restore, then press Enter to start.

**Keys:**

| Key | Action |
|-----|--------|
| `↑` / `↓` or `k` / `j` | Move cursor |
| `space` | Toggle selection (on a schema row: toggles all its tables) |
| `a` | Select all |
| `d` | Deselect all |
| `Enter` | Confirm and start restore |
| `q` / `Esc` | Cancel |

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--input` | **(required)** | Path to the backup directory to restore from |
| `--pre-backup-dir` | `.` | Directory where the pre-restore safety backup will be written |
| `--log-dir` | `.` | Directory for restore log files |

A pre-restore safety backup is taken before any changes are made, same as `restore`. When all tables in a schema are selected, the full schema filter is applied; when individual tables are selected, each is restored along with its transitive dependencies.

### `diff`

Compare two backup directories at the schema level (data differences are not included).

```bash
pgbackup diff /backups/backup_20240101_120000 /backups/backup_20240201_120000
```

Detects added/removed objects and field-level changes for tables (columns, indexes, constraints, triggers, RLS policies), views, materialized views, functions, sequences, types, domains, and enums.

## Backup Directory Structure

Each backup creates a timestamped directory:

```
backup_YYYYMMDD_HHMMSS/
├── _manifest.yaml               # Dependency graph, restore order, metadata
└── public/                      # One directory per schema
    ├── sequences/
    │   └── user_id_seq.yaml
    ├── tables/
    │   ├── users/
    │   │   ├── def.yaml         # DDL, constraints, indexes, grants
    │   │   └── data.csv         # COPY-format data
    │   └── orders/              # Partitioned table (parent)
    │       ├── def.yaml
    │       └── partitions/
    │           ├── orders_2024/
    │           │   ├── def.yaml
    │           │   └── data.csv
    │           └── orders_2025/
    │               ├── def.yaml
    │               └── data.csv
    ├── views/
    ├── materialized_views/
    ├── functions/
    ├── triggers/
    ├── policies/
    ├── types/
    ├── enums/
    └── domains/
```

## Limitations

Tables containing the following column types are skipped and recorded in `_manifest.yaml` under `skipped_tables`:

- `bytea`
- `xml`
- `pg_lsn`
- `txid_snapshot`

Extensions are not backed up.
