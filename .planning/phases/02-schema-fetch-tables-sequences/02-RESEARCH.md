# Phase 2: Schema Fetch - Tables & Sequences - Research

**Researched:** 2026-03-11
**Domain:** PostgreSQL pg_catalog introspection, pgx v5 query API, Go struct design for schema objects
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FETCH-01 | Table definition from pg_catalog (columns, constraints, indexes, grants, RLS, partition definition) | pg_class + pg_attribute + pg_constraint + pg_index + pg_attrdef + pg_partitioned_table queries documented below |
| FETCH-06 | Sequence definition and current value from pg_catalog | pg_sequence + `SELECT last_value FROM <seq>` documented below |
| FETCH-07 | Composite type definition from pg_catalog | pg_type (typtype='c') + pg_class (relkind='c') + pg_attribute documented below |
| FETCH-08 | Domain definition from pg_catalog | pg_type (typtype='d') + pg_constraint + format_type() documented below |
| FETCH-09 | ENUM definition from pg_catalog | pg_type (typtype='e') + pg_enum ordered by enumsortorder documented below |
</phase_requirements>

---

## Summary

Phase 2 implements real `SchemaFetcher` structs for five object kinds: tables (including partitioned tables), sequences, composite types, domains, and ENUMs. All data comes from pg_catalog — no DDL text strings are reconstructed in this phase (that is Phase 4's job). The fetchers return fully populated in-memory `*Def` structs that later phases consume for serialization and DDL generation.

The core challenge is correctly joining the PostgreSQL system catalogs. PostgreSQL stores table structure across at least six catalog tables (`pg_class`, `pg_attribute`, `pg_constraint`, `pg_index`, `pg_attrdef`, `pg_partitioned_table`), and each object kind has its own catalog chain. The pgx v5 `CollectRows` helper with `RowToStructByName` is the correct idiomatic approach for mapping query results to Go structs.

The existing `internal/core/types.go` defines stub `TableDef`, `SequenceDef`, `TypeDef`, `DomainDef`, and `EnumDef` structs with only a `Definition string` placeholder. Phase 2 must **replace** these stubs with fully specified structs containing the fields expected by the spec's section 5.2 YAML format and downstream phases (serialization, DDL generation).

**Primary recommendation:** Place fetcher implementations in `internal/fetch/` subdirectories (one per object kind), expand the `*Def` structs in `internal/core/types.go` with real fields, and use parameterized pg_catalog queries with `$1` placeholders for the schema name throughout.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/jackc/pgx/v5 | v5.8.0 (already in go.mod) | SQL query execution, row scanning | Already decided Phase 1; `CollectRows` + `RowToStructByName` patterns verified |
| database/sql (stdlib) | n/a | Not used — pgx/v5 native API is preferred | pgx native API handles pgx-specific types (oid, int2vector) cleanly |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| context (stdlib) | n/a | Context propagation for all pgx calls | Always pass ctx through; supports cancellation |
| fmt (stdlib) | n/a | Building dynamic sequence last_value query | Sequence name must be interpolated into `SELECT last_value FROM <name>` — cannot use $1 placeholder for table/sequence name |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| pgx.CollectRows + RowToStructByName | manual rows.Next() + rows.Scan() | Manual scan is fine but verbose; CollectRows reduces boilerplate and handles Close() automatically |
| Separate fetcher per object kind | Single mega-fetcher | Single fetcher would create a god object; separate structs follow the command pattern and are individually testable |
| Expanding *Def in core/types.go | New types in fetch/ package | Spec section 10.2 defines TableDef fields in core; serializer and DDL generator (later phases) need the same types — keep all *Def in internal/core |

**Installation:** No new dependencies. All required libraries are already in go.mod from Phase 1.

---

## Architecture Patterns

### Recommended Project Structure

Phase 2 additions (new packages and files):

```
internal/
├── core/
│   └── types.go              # EXPAND: TableDef, SequenceDef, TypeDef, DomainDef, EnumDef with real fields
├── fetch/
│   ├── table/
│   │   └── fetcher.go        # TableSchemaFetcher — implements core.SchemaFetcher
│   ├── sequence/
│   │   └── fetcher.go        # SequenceSchemaFetcher
│   ├── compositetype/
│   │   └── fetcher.go        # CompositeTypeSchemaFetcher (KindType)
│   ├── domain/
│   │   └── fetcher.go        # DomainSchemaFetcher
│   └── enum/
│       └── fetcher.go        # EnumSchemaFetcher
```

Each fetcher package also gets a corresponding `_test.go` file for integration tests that run against a real PostgreSQL instance (skipped when `TEST_DATABASE_URL` is not set).

### Pattern 1: SchemaFetcher Implementation

**What:** Each fetcher is a struct with a `Fetch` method. It takes the schema name as a parameter, runs parameterized pg_catalog queries, and returns `[]core.ObjectDef`.

**When to use:** Every object kind gets its own fetcher struct. Registration into `CommandRegistry` happens at wiring time (e.g., in `cmd/pgbackup/cmd/root.go` or a new `internal/registry/` init file).

**Example:**
```go
// Source: pgx v5 docs https://pkg.go.dev/github.com/jackc/pgx/v5
package table

import (
    "context"
    "github.com/jackc/pgx/v5"
    "github.com/pgbkrs/pgbackup/internal/core"
)

type SchemaFetcher struct{}

func (f *SchemaFetcher) Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]core.ObjectDef, error) {
    tables, err := fetchTables(ctx, conn, schema)
    if err != nil {
        return nil, err
    }
    result := make([]core.ObjectDef, len(tables))
    for i, t := range tables {
        result[i] = t
    }
    return result, nil
}
```

### Pattern 2: Multiple Queries Per Object (N+1 is Acceptable Here)

**What:** For tables, run one query to list all tables, then per-table queries for columns, constraints, and indexes. This is the N+1 pattern — acceptable for a CLI backup tool where total table count is bounded and simplicity wins over query optimization.

**When to use:** When joining all sub-objects into a single complex query would produce duplicated rows (e.g., a table with 10 columns × 3 indexes = 30 rows that need deduplication). Separate queries are cleaner.

**Alternative:** A single query with JSON aggregation (`json_agg`) can fetch all columns in one round-trip per table. Both approaches are correct. The multi-query approach is more readable and easier to test in isolation.

### Pattern 3: Dynamic Query for Sequence Last Value

**What:** The sequence's `last_value` is read directly from the sequence relation — `SELECT last_value FROM schema.seqname`. The sequence name must be interpolated into the SQL string (it is an identifier, not a value, so `$1` placeholders cannot be used for it). Use `pgx.Identifier{schema, seqname}.Sanitize()` to safely quote the identifier.

**When to use:** Only for sequence current-value reads. All other queries use `$1` placeholders.

**Example:**
```go
// Source: pgx v5 docs — pgx.Identifier for safe identifier quoting
quotedName := pgx.Identifier{schema, seqName}.Sanitize()
query := fmt.Sprintf("SELECT last_value FROM %s", quotedName)
var lastValue int64
err := conn.QueryRow(ctx, query).Scan(&lastValue)
```

### Anti-Patterns to Avoid

- **String concatenation for schema/table names without quoting:** Always use `pgx.Identifier{...}.Sanitize()` for identifiers in dynamic SQL. Never `fmt.Sprintf("FROM %s.%s", schema, name)` without quoting.
- **Fetching all schemas at once:** The `Fetch(ctx, conn, schema string)` interface takes a single schema name. Filter all pg_catalog queries with `n.nspname = $1` — do not fetch everything and filter in Go.
- **Not filtering `attnum > 0`:** System columns (oid, ctid) have negative `attnum` values. Always add `AND attnum > 0` when querying `pg_attribute`.
- **Not filtering `NOT attisdropped`:** Dropped columns remain in `pg_attribute` with `attisdropped = true`. Always exclude them.
- **Using `relkind = 'r'` only for tables:** Partitioned tables have `relkind = 'p'`. To get all user tables including partitioned tables, use `relkind IN ('r', 'p')`.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Type name formatting | OID-to-string lookup table | `format_type(atttypid, atttypmod)` | Handles all modifiers (varchar(255), numeric(10,2), etc.) correctly |
| Index DDL reconstruction | Parse pg_index.indkey manually | `pg_get_indexdef(indexrelid)` | Handles expression indexes, partial indexes, covering indexes |
| Constraint text | Decode contype + conkey manually | `pg_get_constraintdef(oid)` | Returns full SQL constraint text including foreign key references |
| Column default expression | Decode pg_attrdef.adbin | `pg_get_expr(adbin, adrelid)` | Converts internal node tree to SQL expression |
| Identifier quoting | fmt.Sprintf | `pgx.Identifier{schema, name}.Sanitize()` | Handles reserved words, special chars, case sensitivity |
| Partition strategy name | Switch on raw char | Map `'r'` → "range", `'l'` → "list", `'h'` → "hash" | Simple but keep it in one place |

**Key insight:** PostgreSQL provides system functions (`format_type`, `pg_get_indexdef`, `pg_get_constraintdef`, `pg_get_expr`) that correctly reconstruct human-readable representations. These are the same functions `pg_dump` uses internally. Use them rather than reimplementing.

---

## Common Pitfalls

### Pitfall 1: Composite Types Include Table Row Types

**What goes wrong:** `pg_type WHERE typtype = 'c'` returns composite types for every table in the schema (PostgreSQL creates a hidden composite type for each table's row structure). Fetching all `typtype='c'` types returns hundreds of spurious entries.

**Why it happens:** PostgreSQL internally represents table rows as composite types. These entries exist in `pg_type` with `typrelid` pointing to the table's `pg_class` entry where `relkind = 'r'` or `'p'`.

**How to avoid:** Join with `pg_class` and filter `relkind = 'c'` — this is the composite type class marker. Or equivalently, exclude entries where `typrelid` points to a table (relkind 'r'/'p').

**Correct filter:**
```sql
WHERE t.typtype = 'c' AND c.relkind = 'c' AND n.nspname = $1
```

**Warning signs:** Fetching composite types returns a list that includes `users`, `orders`, and every other table name in the schema.

### Pitfall 2: Sequence last_value Requires Direct Table Scan

**What goes wrong:** Developer reads `pg_sequence` (which has `seqstart`, `seqmin`, `seqmax`, `seqincrement`, `seqcycle`, `seqcache`) and assumes `last_value` is there — it is not.

**Why it happens:** `pg_sequence` only stores sequence definition metadata. The current value is stored in the sequence relation itself, a special single-row table accessible as `SELECT last_value FROM schema.seqname`.

**How to avoid:** Run two queries for each sequence: one against `pg_sequence` for definition, one `SELECT last_value FROM <quoted_name>` for current value.

**Warning signs:** Code that reads pg_sequence returns a struct with zero for last_value.

### Pitfall 3: pg_index.indkey Is an int2vector, Not int2[]

**What goes wrong:** Scanning `indkey` into `[]int16` fails with a type mismatch because `int2vector` is a PostgreSQL-specific type that pgx does not automatically convert to a Go slice.

**Why it happens:** `int2vector` is an old PostgreSQL internal type distinct from `int2[]`. pgx treats them differently.

**How to avoid:** Use `pg_get_indexdef(indexrelid)` to get the complete CREATE INDEX statement as a string, avoiding the need to parse `indkey` at all. Alternatively, cast: `indkey::int2[]` in the SQL query to convert to a standard array that pgx can scan to `[]int16`.

**Warning signs:** `cannot scan into dest[N] of type *[]int16` error from pgx when querying pg_index directly.

### Pitfall 4: Constraint Columns Array (conkey) Uses Attribute Numbers, Not Names

**What goes wrong:** Reading `conkey` from `pg_constraint` returns column numbers (e.g., `[1, 3]`), not names. Code that stores these numbers needs a secondary join to `pg_attribute` to get the names.

**Why it happens:** pg_constraint stores internal column references by attnum, not by name.

**How to avoid:** Use `pg_get_constraintdef(oid)` to get the complete constraint definition as SQL text (e.g., `PRIMARY KEY (id)`, `UNIQUE (email)`). This avoids the join entirely and returns human-readable output.

**Warning signs:** Constraint definition shows `[1]` instead of column name `id`.

### Pitfall 5: Indexes Supporting Constraints Are Duplicated

**What goes wrong:** Querying both `pg_index` for indexes AND `pg_constraint` for PK/unique constraints returns duplicates — PostgreSQL creates an index for each PK and unique constraint.

**Why it happens:** pg_index contains all indexes including those that back constraints. pg_constraint for primary keys and unique constraints also references the same index via `conindid`.

**How to avoid:** When fetching table indexes, filter out constraint-backing indexes:
```sql
AND NOT ix.indisprimary          -- exclude PK index (captured in constraints)
AND ix.indisunique = false OR    -- either non-unique...
    NOT EXISTS (                  -- ...or not backing a unique constraint
        SELECT 1 FROM pg_constraint c2
        WHERE c2.conindid = ix.indexrelid
        AND c2.contype = 'u'
    )
```
Or: accept duplication and let the serializer deduplicate. The cleaner approach is filtering at the query level.

### Pitfall 6: Schema-Qualified Names for pg_catalog Filters

**What goes wrong:** Using schema name `'public'` works but using `$1` as the schema parameter requires joining through `pg_namespace`. Forgetting this join returns all objects across all schemas.

**Why it happens:** `pg_class.relnamespace` stores an OID, not a name. The schema name lives in `pg_namespace.nspname`.

**How to avoid:** Always join `pg_namespace n ON c.relnamespace = n.oid` and filter `WHERE n.nspname = $1`.

---

## Code Examples

Verified patterns from official PostgreSQL documentation and pgx v5 docs:

### Query: List All Tables in a Schema

```sql
-- Source: PostgreSQL docs pg_class, pg_namespace
SELECT
    c.oid,
    c.relname AS name,
    c.relkind,          -- 'r' = table, 'p' = partitioned table
    c.relrowsecurity AS rls_enabled
FROM pg_class c
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE n.nspname = $1
  AND c.relkind IN ('r', 'p')
ORDER BY c.relname
```

### Query: Columns for a Table

```sql
-- Source: PostgreSQL docs pg_attribute, pg_attrdef, format_type()
SELECT
    a.attname AS name,
    format_type(a.atttypid, a.atttypmod) AS type,
    NOT a.attnotnull AS nullable,
    pg_get_expr(d.adbin, d.adrelid) AS default_expr
FROM pg_attribute a
LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
WHERE a.attrelid = $1   -- table OID
  AND a.attnum > 0
  AND NOT a.attisdropped
ORDER BY a.attnum
```

### Query: Constraints for a Table

```sql
-- Source: PostgreSQL docs pg_constraint, pg_get_constraintdef()
SELECT
    c.conname AS name,
    c.contype,     -- 'p' = primary key, 'u' = unique, 'c' = check, 'f' = foreign key
    pg_get_constraintdef(c.oid) AS definition
FROM pg_constraint c
WHERE c.conrelid = $1   -- table OID
  AND c.contype IN ('p', 'u', 'c')  -- exclude FK (separate object)
ORDER BY c.contype, c.conname
```

### Query: Indexes for a Table (Excluding Constraint-Backing Indexes)

```sql
-- Source: PostgreSQL docs pg_index, pg_class (for index name), pg_am (for method)
SELECT
    ic.relname AS name,
    am.amname AS method,
    pg_get_indexdef(ix.indexrelid) AS definition
FROM pg_index ix
JOIN pg_class ic ON ix.indexrelid = ic.oid
JOIN pg_am am ON ic.relam = am.oid
WHERE ix.indrelid = $1   -- table OID
  AND NOT ix.indisprimary
  AND NOT EXISTS (
      SELECT 1 FROM pg_constraint con
      WHERE con.conindid = ix.indexrelid
        AND con.contype IN ('p', 'u')
  )
ORDER BY ic.relname
```

### Query: Partition Info for a Partitioned Table

```sql
-- Source: PostgreSQL docs pg_partitioned_table, pg_class (for child tables)
-- Step 1: Get partition strategy and key column(s)
SELECT
    pt.partstrat,       -- 'r'=range, 'l'=list, 'h'=hash
    pt.partnatts
FROM pg_partitioned_table pt
WHERE pt.partrelid = $1  -- parent table OID

-- Step 2: Get child table names
SELECT
    child.relname AS child_table
FROM pg_inherits inh
JOIN pg_class child ON child.oid = inh.inhrelid
WHERE inh.inhparent = $1  -- parent table OID
ORDER BY child.relname
```

### Query: All Sequences in a Schema

```sql
-- Source: PostgreSQL docs pg_sequence, pg_class, pg_namespace
SELECT
    c.oid,
    c.relname AS name,
    s.seqstart   AS start_value,
    s.seqmin     AS min_value,
    s.seqmax     AS max_value,
    s.seqincrement AS increment_by,
    s.seqcycle   AS cycle,
    s.seqcache   AS cache
FROM pg_class c
JOIN pg_namespace n ON c.relnamespace = n.oid
JOIN pg_sequence s ON s.seqrelid = c.oid
WHERE n.nspname = $1
  AND c.relkind = 'S'
ORDER BY c.relname
```

### Dynamic Query: Sequence last_value

```go
// Source: pgx v5 docs — pgx.Identifier for safe quoting
// last_value cannot use $1 because it is an identifier (table name), not a value
quotedSeq := pgx.Identifier{schema, seqName}.Sanitize()
var lastValue int64
err := conn.QueryRow(ctx, fmt.Sprintf("SELECT last_value FROM %s", quotedSeq)).Scan(&lastValue)
```

### Query: ENUM Labels for All ENUMs in a Schema

```sql
-- Source: PostgreSQL docs pg_type, pg_enum, pg_namespace
SELECT
    t.oid,
    t.typname AS name,
    array_agg(e.enumlabel ORDER BY e.enumsortorder) AS labels
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
JOIN pg_enum e ON e.enumtypid = t.oid
WHERE t.typtype = 'e'
  AND n.nspname = $1
GROUP BY t.oid, t.typname
ORDER BY t.typname
```

### Query: Composite Types in a Schema

```sql
-- Source: PostgreSQL docs pg_type (typtype='c'), pg_class (relkind='c'), pg_attribute
-- relkind='c' filters to only explicit composite types, excluding table row types
SELECT
    t.oid,
    t.typname AS name,
    a.attname AS field_name,
    format_type(a.atttypid, a.atttypmod) AS field_type
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
JOIN pg_class c ON t.typrelid = c.oid
JOIN pg_attribute a ON a.attrelid = c.oid
WHERE t.typtype = 'c'
  AND c.relkind = 'c'           -- explicit composite type class (not table row type)
  AND n.nspname = $1
  AND a.attnum > 0
  AND NOT a.attisdropped
ORDER BY t.typname, a.attnum
```

### Query: Domains in a Schema

```sql
-- Source: PostgreSQL docs pg_type (typtype='d'), pg_constraint, format_type()
SELECT
    t.oid,
    t.typname AS name,
    format_type(t.typbasetype, t.typtypmod) AS base_type,
    NOT t.typnotnull AS nullable,
    pg_get_expr(t.typdefaultbin, 0) AS default_expr,
    c.conname AS check_name,
    pg_get_constraintdef(c.oid) AS check_definition
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
LEFT JOIN pg_constraint c ON c.contypid = t.oid AND c.contype = 'c'
WHERE t.typtype = 'd'
  AND n.nspname = $1
ORDER BY t.typname
```

### Expanded TypeDef Structs for internal/core/types.go

The stub `*Def` structs in `internal/core/types.go` must be expanded. Below are the complete struct definitions Phase 2 must implement:

```go
// Source: spec section 5.2 + pg_catalog query results

type ColumnDef struct {
    Name        string
    Type        string  // output of format_type()
    Nullable    bool
    Default     string  // output of pg_get_expr(); empty string if no default
}

type ConstraintsDef struct {
    PrimaryKey *PrimaryKeyDef
    Unique     []UniqueConstraintDef
    Check      []CheckConstraintDef
}

type PrimaryKeyDef struct {
    Name    string
    Columns []string
}

type UniqueConstraintDef struct {
    Name       string
    Definition string  // pg_get_constraintdef output, e.g. "UNIQUE (email)"
}

type CheckConstraintDef struct {
    Name       string
    Definition string  // pg_get_constraintdef output, e.g. "CHECK (price > 0)"
}

type IndexDef struct {
    Name       string
    Method     string  // btree, hash, gin, gist, etc.
    Definition string  // full CREATE INDEX statement from pg_get_indexdef()
}

type PartitionDef struct {
    Strategy string   // "range", "list", "hash"
    Children []string // child table names
}

type RLSDef struct {
    Enabled bool
}

type TableDef struct {
    ObjectHeader
    Columns      []ColumnDef
    Constraints  ConstraintsDef
    Indexes      []IndexDef
    Partitioning *PartitionDef  // nil for non-partitioned tables
    RLS          RLSDef
}

type SequenceDef struct {
    ObjectHeader
    StartValue  int64
    MinValue    int64
    MaxValue    int64
    IncrementBy int64
    Cycle       bool
    Cache       int64
    LastValue   int64  // current value at backup time
}

type CompositeField struct {
    Name string
    Type string  // format_type() output
}

type TypeDef struct {
    ObjectHeader
    Fields []CompositeField
}

type DomainDef struct {
    ObjectHeader
    BaseType        string  // format_type() output
    Nullable        bool
    Default         string  // empty if no default
    CheckName       string  // empty if no check constraint
    CheckDefinition string  // pg_get_constraintdef() output; empty if no check
}

type EnumDef struct {
    ObjectHeader
    Labels []string  // ordered by enumsortorder
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `pg_dump` as a subprocess | Direct pg_catalog queries | Project design decision | Full control, no pg_dump dependency |
| Scanning `indkey` as int2vector | Use `pg_get_indexdef()` | pgx v5 API reality | Avoids int2vector scanning complexity |
| SQL string interpolation for identifiers | `pgx.Identifier{}.Sanitize()` | pgx v5 best practice | Prevents SQL injection, handles quoting |

**Not deprecated:** All system catalogs used (pg_class, pg_attribute, pg_constraint, pg_index, pg_type, pg_enum, pg_sequence, pg_namespace, pg_partitioned_table) are stable across all supported PostgreSQL versions (12+). The spec states no PostgreSQL version dependency.

---

## Open Questions

1. **Where should `CommandRegistry` wiring happen?**
   - What we know: Phase 1 left registry empty (stubs). Phase 2 adds real fetchers. The `cmd/pgbackup/cmd/root.go` currently has no registry wiring.
   - What's unclear: Should wiring happen in `root.go`'s `init()`, a new `internal/registry/init.go`, or lazily in `backupCmd.RunE`?
   - Recommendation: Create `internal/registry/registry.go` with a `NewPopulatedRegistry()` function that calls `Register()` for each kind. Call it once in `root.go`'s `PersistentPreRunE` and store the registry in a package-level variable alongside `conn`. This keeps `root.go` clean and makes the registry testable.

2. **Should `TableDef.Columns` include FK column references?**
   - What we know: Spec section 5.2 shows FK defined in the from-table's def.yaml only. ForeignKeyDef is referenced in spec section 10.2 but FK as a standalone object is deferred to Phase 6 (DEPS-03).
   - What's unclear: Should Phase 2 fetch FK constraints into TableDef or defer entirely?
   - Recommendation: Defer FK fetching entirely to Phase 6. Phase 2 should NOT include FK constraints in the `contype IN ('p', 'u', 'c')` filter — exclude `'f'`. This keeps Phase 2 focused and avoids building FK structures before the dependency graph phase defines how they are used.

3. **Behavior when a sequence has never been read (last_value = -1 or start_value)?**
   - What we know: An unread sequence returns `last_value = start_value - 1` or a PostgreSQL version-specific default. `is_called` flag in `pg_sequences` view indicates whether the sequence has been advanced.
   - What's unclear: Should Phase 2 use `pg_sequences` view (which has `last_value` and `is_called`) instead of the direct `SELECT last_value FROM ...` query?
   - Recommendation: Use `SELECT last_value, is_called FROM schema.seqname` to detect unread sequences. If `is_called = false`, record `last_value = start_value - 1` but mark it explicitly. The simpler approach: use `pg_sequences` system view which already exposes `last_value` and `is_called` in one place. **Use `pg_sequences` view instead of the raw sequence table** — it handles the unread-sequence case.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package (already established in Phase 1) |
| Config file | none — `go test ./...` uses standard Go test discovery |
| Quick run command | `go test ./internal/... -short` |
| Full suite command | `go test ./...` |

Integration tests (requiring a live PostgreSQL instance) are gated on `TEST_DATABASE_URL` env var — skip if not set. This pattern is established by `internal/db/connect_test.go` from Phase 1.

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FETCH-01 | `TableSchemaFetcher.Fetch()` returns columns with correct types, nullability, defaults | integration | `go test ./internal/fetch/table/... -run TestFetchTable -v` | ❌ Wave 0 |
| FETCH-01 | Partitioned table fetch returns PartitionDef with strategy and child list | integration | `go test ./internal/fetch/table/... -run TestFetchPartitionedTable -v` | ❌ Wave 0 |
| FETCH-01 | Table with RLS enabled returns `RLSDef.Enabled = true` | integration | `go test ./internal/fetch/table/... -run TestFetchTableRLS -v` | ❌ Wave 0 |
| FETCH-01 | Table fetch returns primary key, unique, and check constraints | integration | `go test ./internal/fetch/table/... -run TestFetchTableConstraints -v` | ❌ Wave 0 |
| FETCH-01 | Table fetch returns indexes (excluding constraint-backing indexes) | integration | `go test ./internal/fetch/table/... -run TestFetchTableIndexes -v` | ❌ Wave 0 |
| FETCH-06 | `SequenceSchemaFetcher.Fetch()` returns start/min/max/increment/cycle/cache and last_value | integration | `go test ./internal/fetch/sequence/... -run TestFetchSequence -v` | ❌ Wave 0 |
| FETCH-07 | `CompositeTypeSchemaFetcher.Fetch()` returns fields with types; excludes table row types | integration | `go test ./internal/fetch/compositetype/... -run TestFetchCompositeType -v` | ❌ Wave 0 |
| FETCH-08 | `DomainSchemaFetcher.Fetch()` returns base type, nullability, default, check constraint | integration | `go test ./internal/fetch/domain/... -run TestFetchDomain -v` | ❌ Wave 0 |
| FETCH-09 | `EnumSchemaFetcher.Fetch()` returns labels in correct enumsortorder | integration | `go test ./internal/fetch/enum/... -run TestFetchEnum -v` | ❌ Wave 0 |
| FETCH-07 | TypeDef satisfies core.ObjectDef interface (compile-time check) | unit | `go build ./internal/fetch/...` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go build ./... && go test ./internal/... -short`
- **Per wave merge:** `go test ./...` (with TEST_DATABASE_URL set)
- **Phase gate:** Full suite green with live PostgreSQL before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/fetch/table/fetcher_test.go` — covers FETCH-01 (all table sub-requirements)
- [ ] `internal/fetch/sequence/fetcher_test.go` — covers FETCH-06
- [ ] `internal/fetch/compositetype/fetcher_test.go` — covers FETCH-07
- [ ] `internal/fetch/domain/fetcher_test.go` — covers FETCH-08
- [ ] `internal/fetch/enum/fetcher_test.go` — covers FETCH-09
- [ ] Test fixture setup function (`setupTestSchema` helper) — creates tables, sequences, types in a test schema then tears down; shared across all fetcher tests

*(No new test framework install needed — stdlib `testing` is sufficient. All tests skip without `TEST_DATABASE_URL`.)*

---

## Sources

### Primary (HIGH confidence)
- [PostgreSQL docs: pg_class](https://www.postgresql.org/docs/current/catalog-pg-class.html) — relkind values ('r','p','c','S'), relrowsecurity, relispartition, relpartbound
- [PostgreSQL docs: pg_attribute](https://www.postgresql.org/docs/current/catalog-pg-attribute.html) — attname, atttypid, attnotnull, atthasdef, attnum, atttypmod, attisdropped
- [PostgreSQL docs: pg_constraint](https://www.postgresql.org/docs/current/catalog-pg-constraint.html) — contype values ('p','u','c','f'), conkey, conindid, pg_get_constraintdef()
- [PostgreSQL docs: pg_index](https://www.postgresql.org/docs/current/catalog-pg-index.html) — indexrelid, indrelid, indkey, indisprimary, indisunique; join with pg_class for name, pg_am for method
- [PostgreSQL docs: pg_type](https://www.postgresql.org/docs/current/catalog-pg-type.html) — typtype ('c','d','e'), typrelid, typbasetype, typdefaultbin
- [PostgreSQL docs: pg_enum](https://www.postgresql.org/docs/current/catalog-pg-enum.html) — enumtypid, enumsortorder, enumlabel
- [PostgreSQL docs: pg_sequence](https://www.postgresql.org/docs/current/catalog-pg-sequence.html) — seqstart, seqmin, seqmax, seqincrement, seqcycle, seqcache (no last_value)
- [PostgreSQL docs: pg_partitioned_table](https://www.postgresql.org/docs/current/catalog-pg-partitioned-table.html) — partstrat ('r','l','h'), partnatts, partattrs
- [PostgreSQL docs: pg_namespace](https://www.postgresql.org/docs/current/catalog-pg-namespace.html) — nspname for schema filtering
- [PostgreSQL docs: pg_attrdef](https://www.postgresql.org/docs/current/catalog-pg-attrdef.html) — adrelid, adnum, adbin; use pg_get_expr(adbin, adrelid) for readable output
- [PostgreSQL docs: system info functions](https://www.postgresql.org/docs/current/functions-info.html) — format_type(), pg_get_expr(), pg_get_indexdef(), pg_get_constraintdef(), pgx.Identifier
- [PostgreSQL docs: composite types](https://www.postgresql.org/docs/current/rowtypes.html) — filter relkind='c' to exclude auto-generated table row composites
- [pgx v5 API docs](https://pkg.go.dev/github.com/jackc/pgx/v5) — CollectRows, RowToStructByName, Identifier.Sanitize(), array scanning, nullable pointer scanning

### Secondary (MEDIUM confidence)
- [pgbackup_spec.md section 5.2](docs/pre/pgbackup_spec.md) — Target YAML format for TableDef (columns, constraints, indexes, rls, partitioning, data)
- [pgbackup_spec.md section 10.2](docs/pre/pgbackup_spec.md) — TableDef Go struct field list (Columns, Indexes, Constraints, ForeignKeys, Triggers, Partitioning, RLS, Grants, Data)

### Tertiary (LOW confidence)
- WebSearch results confirming pg_catalog stability across PostgreSQL versions 12–18 — no specific source URL; treat as established knowledge

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; pgx v5 already in go.mod, query patterns verified against official docs
- pg_catalog queries: HIGH — each query verified against official PostgreSQL documentation pages
- Architecture (struct expansion, package layout): HIGH — follows spec section 10.2 and established Phase 1 patterns
- Pitfalls: HIGH — all pitfalls confirmed against official catalog documentation (int2vector type, composite type filtering, last_value absence from pg_sequence)
- Open questions (wiring location, FK deferral, unread sequences): MEDIUM — recommendations are sound but involve design judgment calls for the planner

**Research date:** 2026-03-11
**Valid until:** 2026-06-11 (pg_catalog is extremely stable; these catalogs have been unchanged for many PostgreSQL major versions)
