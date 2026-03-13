# pgbackup

PostgreSQL schema backup, restore, and diff tool. Exports schema definitions as YAML and data as CSV, enabling object-level restore and schema comparison between backups.

## Installation

```bash
git clone https://github.com/pgbkrs/pgbackup
cd pgbackup
go build -o pgbackup ./cmd/pgbackup
```

## Connection Flags

All subcommands (except `diff`) require a database connection:

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `localhost` | PostgreSQL host |
| `--port` | `5432` | PostgreSQL port |
| `--user` | | PostgreSQL user |
| `--password` | | PostgreSQL password |
| `--dbname` | | Target database name |

## Commands

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
