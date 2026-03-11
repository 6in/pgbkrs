# Phase 3: Schema Fetch - Views, Functions, Triggers & Policies - Research

**Researched:** 2026-03-11
**Domain:** PostgreSQL pg_catalog introspection for views, materialized views, functions, triggers, and RLS policies; pgx v5 query API; Go struct expansion for schema objects
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FETCH-02 | View definition from pg_catalog (definition SQL and owner) | pg_views system view + pg_class/pg_roles for owner; pg_get_viewdef() returns the SELECT part only; use pg_views.definition directly |
| FETCH-03 | Materialized view definition from pg_catalog (definition SQL and is_populated flag) | pg_matviews system view exposes definition and ispopulated; relkind='m' in pg_class |
| FETCH-04 | Function definition from pg_catalog via pg_get_functiondef(); argument and return types | pg_proc + pg_get_functiondef() + pg_get_function_arguments() + pg_get_function_result() |
| FETCH-05 | Trigger definition from pg_catalog (event, timing, function reference, target table) | pg_trigger + pg_class (for table name) + pg_proc (for function name); tgtype bitmask for event/timing |
| FETCH-10 | RLS policy definition from pg_catalog (name, command, roles, USING/WITH CHECK expressions) | pg_policy + pg_class (for table) + pg_get_expr() for qual and with_check; polroles is oid[] |
</phase_requirements>

---

## Summary

Phase 3 implements five `SchemaFetcher` structs for views, materialized views, functions, triggers, and RLS policies. All data comes from pg_catalog system views and catalog tables. The pattern is identical to Phase 2: one fetcher struct per object kind in `internal/fetch/<kind>/fetcher.go`, implementing `core.SchemaFetcher`.

The current stub `*Def` types in `internal/core/types.go` carry only a `Definition string` field for all five kinds. Phase 3 must expand these structs with the fields required by the success criteria. The expansion is additive — no existing fields are removed. All five object types already have their `ObjectKind` constants defined in `core/types.go` and their `Header()` methods are correct.

The most complex catalog interaction is triggers: `pg_trigger.tgtype` is a bitmask encoding timing (BEFORE/AFTER/INSTEAD OF) and event (INSERT/UPDATE/DELETE/TRUNCATE). Functions require `pg_get_functiondef()` which returns the complete `CREATE OR REPLACE FUNCTION` statement; argument and return types are extracted via `pg_get_function_arguments()` and `pg_get_function_result()`. RLS policies reference role OIDs in `pg_policy.polroles` which must be converted to role names via a join or `pg_get_userbyid()`.

**Primary recommendation:** Use PostgreSQL system views (`pg_views`, `pg_matviews`) where available — they already join the catalog tables and expose human-readable names. For triggers and policies, query `pg_trigger` and `pg_policy` directly with explicit joins to resolve OIDs to names.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/jackc/pgx/v5 | v5.8.0 (in go.mod) | SQL query execution, row scanning | Established in Phase 1; all fetchers use the same API |
| context (stdlib) | n/a | Context propagation | Always pass ctx to pgx calls |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| fmt (stdlib) | n/a | No new dynamic SQL needed in this phase | Views/functions/triggers/policies all use schema-name $1 parameter; no identifier interpolation required |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| pg_views system view | pg_class + pg_rewrite JOIN | pg_views already joins everything and exposes `definition` and `viewowner`; simpler |
| pg_matviews system view | pg_class relkind='m' + pg_rewrite | pg_matviews exposes `ispopulated` directly; no extra join needed |
| pg_get_functiondef() | Reconstruct from pg_proc fields | pg_get_functiondef returns the complete CREATE statement; reconstruction is extremely complex and fragile |
| pg_get_userbyid() for policy roles | JOIN pg_roles | pg_get_userbyid() is simpler for scalar conversion; array expansion requires unnest + JOIN |

**Installation:** No new dependencies. All required libraries are in go.mod from Phase 1.

---

## Architecture Patterns

### Recommended Project Structure

Phase 3 additions (new packages only; no existing files are modified except `internal/core/types.go`):

```
internal/
├── core/
│   └── types.go              # EXPAND: ViewDef, MaterializedViewDef, FunctionDef, TriggerDef, PolicyDef with real fields
├── fetch/
│   ├── view/
│   │   ├── fetcher.go        # ViewSchemaFetcher
│   │   └── fetcher_test.go
│   ├── matview/
│   │   ├── fetcher.go        # MatViewSchemaFetcher
│   │   └── fetcher_test.go
│   ├── function/
│   │   ├── fetcher.go        # FunctionSchemaFetcher
│   │   └── fetcher_test.go
│   ├── trigger/
│   │   ├── fetcher.go        # TriggerSchemaFetcher
│   │   └── fetcher_test.go
│   └── policy/
│       ├── fetcher.go        # PolicySchemaFetcher
│       └── fetcher_test.go
└── fetch/
    └── testhelpers/
        └── setup.go          # EXPAND: add view/matview/function/trigger/policy fixtures
```

### Pattern 1: SchemaFetcher Implementation (Unchanged from Phase 2)

Each fetcher is a struct with a compile-time interface assertion and a `Fetch` method:

```go
// Source: established pattern from internal/fetch/enum/fetcher.go
var _ core.SchemaFetcher = (*SchemaFetcher)(nil)

type SchemaFetcher struct{}

func (f *SchemaFetcher) Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]core.ObjectDef, error) {
    rows, err := conn.Query(ctx, fetchQuery, schema)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var defs []core.ObjectDef
    for rows.Next() {
        // scan fields, append to defs
    }
    return defs, rows.Err()
}
```

### Pattern 2: System Views for Simple Cases (Views and Materialized Views)

`pg_views` and `pg_matviews` are PostgreSQL system views that already join the underlying catalog tables. They expose human-readable names and avoid OID-resolution joins.

```sql
-- pg_views exposes: schemaname, viewname, viewowner, definition
-- pg_matviews exposes: schemaname, matviewname, matviewowner, definition, ispopulated
SELECT viewname AS name, viewowner AS owner, definition
FROM pg_views
WHERE schemaname = $1
ORDER BY viewname
```

### Pattern 3: pg_get_functiondef for Function Bodies

`pg_get_functiondef(oid)` returns the complete `CREATE OR REPLACE FUNCTION` statement as a string. This is what pg_dump uses. Argument and return type summaries are obtained via two more system functions.

```sql
-- Source: PostgreSQL docs functions-info.html
SELECT
    p.proname AS name,
    pg_get_functiondef(p.oid) AS definition,
    pg_get_function_arguments(p.oid) AS arg_types,
    pg_get_function_result(p.oid) AS return_type
FROM pg_proc p
JOIN pg_namespace n ON p.pronamespace = n.oid
WHERE n.nspname = $1
  AND p.prokind = 'f'   -- 'f' = function, 'p' = procedure, 'a' = aggregate, 'w' = window
ORDER BY p.proname
```

### Pattern 4: Trigger tgtype Bitmask Decoding

`pg_trigger.tgtype` is a smallint bitmask. The bits encode timing and event:

| Bit position | Value | Meaning |
|---|---|---|
| 0 (0x01) | 1 = ROW, 0 = STATEMENT | Trigger level |
| 1 (0x02) | 1 = BEFORE | Timing BEFORE |
| 2 (0x04) | 1 = INSERT | Event INSERT |
| 3 (0x08) | 1 = DELETE | Event DELETE |
| 4 (0x10) | 1 = UPDATE | Event UPDATE |
| 5 (0x20) | 1 = TRUNCATE | Event TRUNCATE |
| 6 (0x40) | 1 = INSTEAD OF | Timing INSTEAD OF |

Rather than decoding tgtype in Go, use `pg_get_triggerdef(t.oid)` which returns the complete `CREATE TRIGGER` statement — the same approach as using `pg_get_functiondef`. However the success criterion requires individual fields (event, timing, function reference, target table), not a monolithic DDL string. Decode the bitmask in Go or extract fields individually from pg_catalog.

**Recommended approach:** Fetch raw fields from pg_trigger and decode tgtype in Go. Avoids parsing a DDL string.

```sql
-- Source: PostgreSQL docs catalog-pg-trigger.html
SELECT
    t.tgname AS name,
    t.tgtype::int AS tgtype,
    c.relname AS table_name,
    p.proname AS function_name,
    n2.nspname AS function_schema
FROM pg_trigger t
JOIN pg_class c ON c.oid = t.tgrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_proc p ON p.oid = t.tgfoid
JOIN pg_namespace n2 ON n2.oid = p.pronamespace
WHERE n.nspname = $1
  AND NOT t.tgisinternal    -- exclude constraint-generated triggers
ORDER BY c.relname, t.tgname
```

Decode tgtype in Go:

```go
// Source: PostgreSQL source include/catalog/pg_trigger.h constants
const (
    tgTypeRow       = 0x01
    tgTypeBefore    = 0x02
    tgTypeInsert    = 0x04
    tgTypeDelete    = 0x08
    tgTypeUpdate    = 0x10
    tgTypeTruncate  = 0x20
    tgTypeInsteadOf = 0x40
)

func decodeTiming(tgtype int) string {
    if tgtype&tgTypeInsteadOf != 0 {
        return "INSTEAD OF"
    }
    if tgtype&tgTypeBefore != 0 {
        return "BEFORE"
    }
    return "AFTER"
}

func decodeEvents(tgtype int) []string {
    var events []string
    if tgtype&tgTypeInsert != 0 { events = append(events, "INSERT") }
    if tgtype&tgTypeUpdate != 0 { events = append(events, "UPDATE") }
    if tgtype&tgTypeDelete != 0 { events = append(events, "DELETE") }
    if tgtype&tgTypeTruncate != 0 { events = append(events, "TRUNCATE") }
    return events
}
```

### Pattern 5: RLS Policy Role OID Array

`pg_policy.polroles` is an `oid[]` (array of role OIDs). An empty array (`'{}'`) means the policy applies to `PUBLIC` (all roles). Convert OIDs to role names using `pg_get_userbyid()` applied via `unnest`.

```sql
-- Source: PostgreSQL docs catalog-pg-policy.html
SELECT
    p.polname AS name,
    c.relname AS table_name,
    CASE p.polcmd
        WHEN 'r' THEN 'SELECT'
        WHEN 'a' THEN 'INSERT'
        WHEN 'w' THEN 'UPDATE'
        WHEN 'd' THEN 'DELETE'
        WHEN '*' THEN 'ALL'
    END AS command,
    CASE WHEN p.polroles = '{0}' THEN ARRAY['PUBLIC']
         ELSE array_agg(pg_get_userbyid(r))
    END AS roles,
    pg_get_expr(p.polqual, p.polrelid) AS using_expr,
    pg_get_expr(p.polwithcheck, p.polrelid) AS with_check_expr
FROM pg_policy p
JOIN pg_class c ON c.oid = p.polrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN unnest(p.polroles) AS r ON TRUE
WHERE n.nspname = $1
GROUP BY p.polname, c.relname, p.polcmd, p.polroles, p.polqual, p.polwithcheck, p.polrelid
ORDER BY c.relname, p.polname
```

**Note:** When `polroles = '{0}'` the policy applies to PUBLIC (the sentinel OID 0 is used). The `pg_get_userbyid(0)` returns empty string, so check for this case explicitly.

### Anti-Patterns to Avoid

- **Fetching internal triggers:** PostgreSQL creates internal triggers to enforce deferrable constraints. These have `tgisinternal = true`. Always filter `AND NOT t.tgisinternal` — otherwise every FK constraint adds invisible trigger entries.
- **Forgetting prokind filter for functions:** `pg_proc` contains aggregate functions, window functions, and procedures in addition to regular functions. Use `AND p.prokind = 'f'` to get only regular functions (or adjust based on which proc types are in scope — spec says "functions").
- **Using pg_views.definition raw as DDL:** `pg_views.definition` contains only the SELECT part of the view, not the `CREATE VIEW name AS` prefix. For DDL generation (Phase 4), the full statement must be reconstructed. For Phase 3 (fetch only), store the definition as returned — consistent with how pg_get_functiondef is stored verbatim.
- **Treating polroles='{0}' as a real role:** OID 0 in polroles is the sentinel for PUBLIC. Decode to the string "PUBLIC", not an empty role name.
- **Not filtering pg_trigger by schema:** pg_trigger has no direct schema column. Join through pg_class and pg_namespace to filter by schema.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| View definition text | Query pg_rewrite and reconstruct | `pg_views.definition` or `pg_get_viewdef(oid)` | pg_rewrite stores parse trees, not SQL text |
| Function body reconstruction | Parse pg_proc.prosrc + prolang + arguments manually | `pg_get_functiondef(oid)` | Returns complete CREATE statement; handles all languages (SQL, PL/pgSQL, etc.) |
| Trigger DDL | Decode tgtype + tgfoid + tgrelid manually into DDL text | `pg_get_triggerdef(oid)` for DDL (Phase 4); for field extraction, decode tgtype bitmask directly | Avoids parsing DDL text to extract fields |
| RLS policy expressions | Decode polqual node tree | `pg_get_expr(polqual, polrelid)` | Same pattern as pg_get_expr for column defaults in Phase 2 |
| Role OID to name | pg_roles join per OID | `pg_get_userbyid(oid)` | Handles dropped roles gracefully; standard catalog function |

**Key insight:** PostgreSQL provides `pg_get_*` system functions for every major object type. Phase 3 extends the same pattern used in Phase 2 (pg_get_constraintdef, pg_get_expr, pg_get_indexdef) to views, functions, triggers, and policies.

---

## Common Pitfalls

### Pitfall 1: tgisinternal Triggers

**What goes wrong:** Querying `pg_trigger` without filtering `tgisinternal = false` returns internal triggers created by PostgreSQL for deferrable FK constraints and other system mechanisms. A table with 5 FK constraints has 5 internal triggers that are invisible at the SQL level.

**Why it happens:** PostgreSQL creates triggers internally for constraint enforcement. These are not user-defined triggers and should not be backed up or restored as standalone objects.

**How to avoid:** Always add `AND NOT t.tgisinternal` to pg_trigger queries.

**Warning signs:** A table with 2 user triggers shows 7 triggers in the query result.

### Pitfall 2: pg_views.definition Is Not a Complete DDL Statement

**What goes wrong:** Code stores `pg_views.definition` (e.g., `SELECT id, name FROM users WHERE active`) and tries to replay it as DDL — it fails because there is no `CREATE VIEW name AS` prefix.

**Why it happens:** `pg_views.definition` is only the view's SELECT body. The view name and schema are separate columns.

**How to avoid:** For Phase 3 (fetching), store the definition as returned by `pg_views.definition`. The DDL generation (Phase 4) will prepend `CREATE OR REPLACE VIEW schema.name AS`. Document this in `ViewDef.Definition` Go doc comments.

**Warning signs:** Attempting `conn.Exec(ctx, viewDef.Definition)` fails with SQL syntax error.

### Pitfall 3: Functions with Overloaded Names

**What goes wrong:** Fetching functions by name only returns one when multiple overloaded functions exist (same name, different argument types). PostgreSQL supports function overloading — `calc_total(integer)` and `calc_total(bigint)` are distinct objects.

**Why it happens:** pg_proc uses OID as primary key; the same `proname` appears multiple times for overloaded functions.

**How to avoid:** Always use OID as the primary key when scanning pg_proc. The `FunctionDef.Name` can remain the base name, but tests must account for multiple functions with the same name. Use `pg_get_function_arguments(oid)` to distinguish them (included in `FunctionDef.ArgTypes`).

**Warning signs:** Only one of two overloaded functions appears in test output.

### Pitfall 4: Materialized View ispopulated Flag

**What goes wrong:** Code assumes all materialized views are populated and skips the `ispopulated` field. An unpopulated materialized view (`REFRESH MATERIALIZED VIEW` never run) cannot be queried — attempting to restore data into it will fail silently or with cryptic errors.

**Why it happens:** `CREATE MATERIALIZED VIEW ... WITH NO DATA` creates an unpopulated view. `ispopulated` is `false` until `REFRESH MATERIALIZED VIEW` is called.

**How to avoid:** Always fetch `ispopulated` from `pg_matviews` and store it in `MaterializedViewDef.IsPopulated`. The DDL generator (Phase 4) uses this to decide whether to emit `WITH NO DATA` on restore.

**Warning signs:** All materialized views show `IsPopulated: true` even when created `WITH NO DATA`.

### Pitfall 5: polroles OID Array With PUBLIC Sentinel

**What goes wrong:** `pg_policy.polroles = '{0}'` encodes "applies to PUBLIC" using OID 0. Calling `pg_get_userbyid(0)` returns an empty string. Code that maps this to an empty roles list loses the PUBLIC semantics.

**Why it happens:** PostgreSQL uses OID 0 as a sentinel for "all roles" in pg_policy. This is a catalog-level convention, not an actual role.

**How to avoid:** Check for `polroles = '{0}'` (or equivalently `0 IN (SELECT unnest(polroles))`) before calling pg_get_userbyid. Map to `["PUBLIC"]` explicitly.

**Warning signs:** A policy that should apply to PUBLIC shows an empty roles slice in the struct.

### Pitfall 6: Procedure vs. Function in pg_proc

**What goes wrong:** `pg_get_functiondef` called on a procedure (prokind='p') returns `CREATE PROCEDURE`, not `CREATE FUNCTION`. If the code assumes all pg_proc entries are functions and the schema has stored procedures, the output mix will be confusing.

**Why it happens:** PostgreSQL 11+ added stored procedures as a distinct kind (prokind='p'). They live in pg_proc alongside functions, aggregates, and window functions.

**How to avoid:** FETCH-04 requirement is for "functions". Filter `AND p.prokind = 'f'` to get only regular functions. If procedures need to be backed up in future, add a separate KindProcedure fetcher.

---

## Code Examples

### Query: Views in a Schema

```sql
-- Source: PostgreSQL docs pg_views system view
-- https://www.postgresql.org/docs/current/view-pg-views.html
SELECT viewname AS name, viewowner AS owner, definition
FROM pg_views
WHERE schemaname = $1
ORDER BY viewname
```

### Query: Materialized Views in a Schema

```sql
-- Source: PostgreSQL docs pg_matviews system view
-- https://www.postgresql.org/docs/current/view-pg-matviews.html
SELECT matviewname AS name, matviewowner AS owner, definition, ispopulated
FROM pg_matviews
WHERE schemaname = $1
ORDER BY matviewname
```

### Query: Functions in a Schema

```sql
-- Source: PostgreSQL docs pg_proc catalog and system info functions
-- https://www.postgresql.org/docs/current/catalog-pg-proc.html
-- https://www.postgresql.org/docs/current/functions-info.html
SELECT
    p.proname AS name,
    pg_get_functiondef(p.oid) AS definition,
    pg_get_function_arguments(p.oid) AS arg_types,
    pg_get_function_result(p.oid) AS return_type
FROM pg_proc p
JOIN pg_namespace n ON p.pronamespace = n.oid
WHERE n.nspname = $1
  AND p.prokind = 'f'
ORDER BY p.proname, p.oid
```

### Query: Triggers in a Schema

```sql
-- Source: PostgreSQL docs pg_trigger catalog
-- https://www.postgresql.org/docs/current/catalog-pg-trigger.html
SELECT
    t.tgname AS name,
    t.tgtype::int AS tgtype,
    c.relname AS table_name,
    p.proname AS function_name
FROM pg_trigger t
JOIN pg_class c ON c.oid = t.tgrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_proc p ON p.oid = t.tgfoid
WHERE n.nspname = $1
  AND NOT t.tgisinternal
ORDER BY c.relname, t.tgname
```

### tgtype Decoding in Go

```go
// Source: PostgreSQL source code include/catalog/pg_trigger.h
const (
    tgTypeBefore    = 0x02
    tgTypeInsert    = 0x04
    tgTypeDelete    = 0x08
    tgTypeUpdate    = 0x10
    tgTypeTruncate  = 0x20
    tgTypeInsteadOf = 0x40
)

func decodeTiming(tgtype int) string {
    if tgtype&tgTypeInsteadOf != 0 {
        return "INSTEAD OF"
    }
    if tgtype&tgTypeBefore != 0 {
        return "BEFORE"
    }
    return "AFTER"
}

func decodeEvents(tgtype int) []string {
    var events []string
    if tgtype&tgTypeInsert != 0   { events = append(events, "INSERT") }
    if tgtype&tgTypeUpdate != 0   { events = append(events, "UPDATE") }
    if tgtype&tgTypeDelete != 0   { events = append(events, "DELETE") }
    if tgtype&tgTypeTruncate != 0 { events = append(events, "TRUNCATE") }
    return events
}
```

### Query: RLS Policies in a Schema

```sql
-- Source: PostgreSQL docs pg_policy catalog
-- https://www.postgresql.org/docs/current/catalog-pg-policy.html
SELECT
    p.polname AS name,
    c.relname AS table_name,
    CASE p.polcmd
        WHEN 'r' THEN 'SELECT'
        WHEN 'a' THEN 'INSERT'
        WHEN 'w' THEN 'UPDATE'
        WHEN 'd' THEN 'DELETE'
        WHEN '*' THEN 'ALL'
    END AS command,
    CASE WHEN p.polroles = '{0}' THEN ARRAY['PUBLIC']::text[]
         ELSE array_agg(pg_get_userbyid(r.roloid) ORDER BY r.roloid)
    END AS roles,
    COALESCE(pg_get_expr(p.polqual, p.polrelid), '') AS using_expr,
    COALESCE(pg_get_expr(p.polwithcheck, p.polrelid), '') AS with_check_expr
FROM pg_policy p
JOIN pg_class c ON c.oid = p.polrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN LATERAL unnest(
    CASE WHEN p.polroles = '{0}' THEN '{}'::oid[] ELSE p.polroles END
) AS r(roloid) ON TRUE
WHERE n.nspname = $1
GROUP BY p.polname, c.relname, p.polcmd, p.polroles, p.polqual, p.polwithcheck, p.polrelid
ORDER BY c.relname, p.polname
```

### Expanded Struct Definitions for internal/core/types.go

The current stubs in `types.go` use only `Definition string`. Phase 3 must replace them:

```go
// Source: Phase 3 requirements (FETCH-02 through FETCH-05, FETCH-10)
// + spec section 10.2

// ViewDef represents a view schema object.
// Definition contains the SELECT body only (as returned by pg_views.definition).
// The full CREATE VIEW statement is reconstructed by DDLGenerator in Phase 4.
type ViewDef struct {
    ObjectHeader
    Definition string // SELECT body from pg_views.definition
    Owner      string // viewowner from pg_views
}

// MaterializedViewDef represents a materialized view schema object.
type MaterializedViewDef struct {
    ObjectHeader
    Definition  string // SELECT body from pg_matviews.definition
    Owner       string // matviewowner from pg_matviews
    IsPopulated bool   // ispopulated from pg_matviews; false = created WITH NO DATA
}

// FunctionDef represents a function schema object.
// Definition is the complete CREATE OR REPLACE FUNCTION statement from pg_get_functiondef.
type FunctionDef struct {
    ObjectHeader
    Definition string // complete CREATE OR REPLACE FUNCTION ... statement
    ArgTypes   string // pg_get_function_arguments() output, e.g. "a integer, b text"
    ReturnType string // pg_get_function_result() output, e.g. "integer"
}

// TriggerDef represents a trigger schema object.
// Timing is one of: "BEFORE", "AFTER", "INSTEAD OF".
// Events is one or more of: "INSERT", "UPDATE", "DELETE", "TRUNCATE".
// TableName is the unqualified name of the trigger's target table.
// FunctionName is the unqualified name of the trigger function.
type TriggerDef struct {
    ObjectHeader
    Timing       string   // "BEFORE", "AFTER", or "INSTEAD OF"
    Events       []string // e.g. ["INSERT", "UPDATE"]
    TableName    string   // target table (unqualified)
    FunctionName string   // trigger function name (unqualified)
}

// PolicyDef represents a row-level security policy on a table.
// Command is one of: "SELECT", "INSERT", "UPDATE", "DELETE", "ALL".
// Roles is the list of role names; contains "PUBLIC" if the policy applies to all roles.
// Using is the USING expression (empty string if none).
// WithCheck is the WITH CHECK expression (empty string if none).
type PolicyDef struct {
    ObjectHeader
    TableName  string   // target table name (unqualified)
    Command    string   // "SELECT", "INSERT", "UPDATE", "DELETE", or "ALL"
    Roles      []string // role names; ["PUBLIC"] when polroles={0}
    Using      string   // USING expression or empty string
    WithCheck  string   // WITH CHECK expression or empty string
}
```

### Test Fixture Expansion for testhelpers/setup.go

The shared `SetupTestSchema` function in `internal/fetch/testhelpers/setup.go` must be extended to create test objects for Phase 3 fetchers:

```sql
-- Add to SetupTestSchema DDL:

-- View
CREATE VIEW %s.active_items AS
    SELECT id, email FROM %s.simple_table WHERE price > 0;

-- Materialized view (populated)
CREATE MATERIALIZED VIEW %s.mv_items AS
    SELECT id, email FROM %s.simple_table;

-- Function
CREATE FUNCTION %s.add_values(a integer, b integer) RETURNS integer
    LANGUAGE sql AS 'SELECT a + b';

-- Table for trigger
CREATE TABLE %s.audit_log (
    id         serial PRIMARY KEY,
    table_name text,
    changed_at timestamptz DEFAULT now()
);

-- Trigger function
CREATE FUNCTION %s.record_audit() RETURNS trigger
    LANGUAGE plpgsql AS $$
    BEGIN
        INSERT INTO %[1]s.audit_log(table_name, changed_at) VALUES (TG_TABLE_NAME, now());
        RETURN NEW;
    END;
    $$;

-- Trigger
CREATE TRIGGER trg_audit_simple
    AFTER INSERT ON %s.simple_table
    FOR EACH ROW EXECUTE FUNCTION %s.record_audit();

-- RLS policy (rls_table already has RLS enabled from Phase 2)
CREATE POLICY tenant_isolation ON %s.rls_table
    FOR ALL
    TO PUBLIC
    USING (true)
    WITH CHECK (true);
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Subprocess pg_dump | Direct pg_catalog queries | Project design decision | Full control, version-independent |
| Parsing pg_get_triggerdef output | Decode tgtype bitmask + fetch proc/class names directly | Phase 3 design | Avoids fragile DDL string parsing |
| Role OID arrays decoded manually | pg_get_userbyid() + unnest + LATERAL | PostgreSQL 9.3+ (LATERAL) | Clean single-query role resolution |

**Not deprecated:** All system views (`pg_views`, `pg_matviews`) and catalog tables (`pg_trigger`, `pg_policy`, `pg_proc`) are stable across PostgreSQL 12+. The system info functions `pg_get_functiondef`, `pg_get_function_arguments`, `pg_get_function_result` are stable since PostgreSQL 8.4.

---

## Open Questions

1. **Should TriggerDef.Name use schema-qualified trigger name or bare name?**
   - What we know: Triggers in PostgreSQL are not schema-scoped at the SQL level — they belong to a table, and the trigger name is unique within that table. The `ObjectHeader.Schema` and `ObjectHeader.Name` fields follow the convention for other objects.
   - What's unclear: The trigger name in ObjectHeader.Name is the trigger name (e.g., `trg_audit_simple`). The ObjectHeader.Schema is the schema of the trigger's table. This is consistent with how other objects are keyed.
   - Recommendation: Set ObjectHeader.Name to the trigger name and TriggerDef.TableName to the target table's unqualified name. The composite key for a trigger is `(schema, table_name, name)`.

2. **Should PolicyDef belong to its table or stand alone as a separate object?**
   - What we know: FETCH-10 says "Fetching RLS policies for a table returns each policy's..." implying a per-table fetch. But the existing `KindPolicy` and `PolicySchemaFetcher` pattern treats policies as top-level objects in a schema, like views.
   - What's unclear: The `SchemaFetcher.Fetch(ctx, conn, schema)` interface takes a schema, not a table — so a PolicySchemaFetcher returns all policies across all tables in the schema. The `PolicyDef.TableName` field identifies which table each policy belongs to.
   - Recommendation: Implement `PolicySchemaFetcher` returning all policies in the schema (not per-table). Set `PolicyDef.TableName` to link each policy to its table. This is consistent with the fetcher interface contract.

3. **Do procedures (prokind='p') need to be included alongside functions?**
   - What we know: FETCH-04 says "function definition" and the spec section 3.2 lists "function" as an object type without mentioning procedures. The existing KindFunction constant covers this.
   - What's unclear: Whether stored procedures (PostgreSQL 11+) should be captured under KindFunction or deferred.
   - Recommendation: Filter `prokind = 'f'` for Phase 3. If procedures need capture later, add KindProcedure in a future phase. This matches the spec's intent.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package (established in Phase 1) |
| Config file | none — `go test ./...` uses standard Go test discovery |
| Quick run command | `go test ./internal/... -short` |
| Full suite command | `TEST_DATABASE_URL=<dsn> go test ./...` |

Integration tests are gated on `TEST_DATABASE_URL` env var. Same pattern as Phase 2.

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FETCH-02 | `ViewSchemaFetcher.Fetch()` returns view with non-empty Definition and Owner | integration | `go test ./internal/fetch/view/... -run TestFetchView -v` | ❌ Wave 0 |
| FETCH-03 | `MatViewSchemaFetcher.Fetch()` returns matview with non-empty Definition and IsPopulated=true | integration | `go test ./internal/fetch/matview/... -run TestFetchMatView -v` | ❌ Wave 0 |
| FETCH-03 | MatView created WITH NO DATA returns IsPopulated=false | integration | `go test ./internal/fetch/matview/... -run TestFetchMatViewUnpopulated -v` | ❌ Wave 0 |
| FETCH-04 | `FunctionSchemaFetcher.Fetch()` returns function with non-empty Definition, ArgTypes, ReturnType | integration | `go test ./internal/fetch/function/... -run TestFetchFunction -v` | ❌ Wave 0 |
| FETCH-05 | `TriggerSchemaFetcher.Fetch()` returns trigger with correct Timing, Events, TableName, FunctionName | integration | `go test ./internal/fetch/trigger/... -run TestFetchTrigger -v` | ❌ Wave 0 |
| FETCH-05 | Internal constraint triggers are excluded from results | integration | `go test ./internal/fetch/trigger/... -run TestFetchTriggerExcludesInternal -v` | ❌ Wave 0 |
| FETCH-10 | `PolicySchemaFetcher.Fetch()` returns policy with Command, Roles, Using, WithCheck | integration | `go test ./internal/fetch/policy/... -run TestFetchPolicy -v` | ❌ Wave 0 |
| FETCH-10 | Policy with PUBLIC roles returns Roles=["PUBLIC"] not empty slice | integration | `go test ./internal/fetch/policy/... -run TestFetchPolicyPublicRoles -v` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go build ./... && go test ./internal/... -short`
- **Per wave merge:** `TEST_DATABASE_URL=<dsn> go test ./...`
- **Phase gate:** Full suite green with live PostgreSQL before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/fetch/view/fetcher_test.go` — covers FETCH-02
- [ ] `internal/fetch/matview/fetcher_test.go` — covers FETCH-03 (populated and unpopulated cases)
- [ ] `internal/fetch/function/fetcher_test.go` — covers FETCH-04
- [ ] `internal/fetch/trigger/fetcher_test.go` — covers FETCH-05 (including internal trigger exclusion)
- [ ] `internal/fetch/policy/fetcher_test.go` — covers FETCH-10 (including PUBLIC role sentinel)
- [ ] `internal/fetch/testhelpers/setup.go` — extend with view, matview, function, trigger, policy fixtures

_(No new test framework install needed. All tests skip without `TEST_DATABASE_URL`.)_

---

## Sources

### Primary (HIGH confidence)
- [PostgreSQL docs: pg_views](https://www.postgresql.org/docs/current/view-pg-views.html) — schemaname, viewname, viewowner, definition columns
- [PostgreSQL docs: pg_matviews](https://www.postgresql.org/docs/current/view-pg-matviews.html) — matviewname, matviewowner, definition, ispopulated columns
- [PostgreSQL docs: pg_proc](https://www.postgresql.org/docs/current/catalog-pg-proc.html) — prokind ('f'/'p'/'a'/'w'), proname, pronamespace; pg_get_functiondef, pg_get_function_arguments, pg_get_function_result
- [PostgreSQL docs: pg_trigger](https://www.postgresql.org/docs/current/catalog-pg-trigger.html) — tgname, tgtype, tgrelid, tgfoid, tgisinternal; tgtype bitmask constants
- [PostgreSQL docs: pg_policy](https://www.postgresql.org/docs/current/catalog-pg-policy.html) — polname, polcmd ('r','a','w','d','*'), polroles oid[], polqual, polwithcheck
- [PostgreSQL docs: system information functions](https://www.postgresql.org/docs/current/functions-info.html) — pg_get_functiondef, pg_get_function_arguments, pg_get_function_result, pg_get_expr, pg_get_userbyid, pg_get_triggerdef
- [Phase 2 RESEARCH.md](../02-schema-fetch-tables-sequences/02-RESEARCH.md) — established patterns for SchemaFetcher, pg_catalog query style, pgx v5 scanning, testhelpers
- [internal/core/types.go](../../../internal/core/types.go) — current stub struct definitions that Phase 3 must expand
- [internal/fetch/enum/fetcher.go](../../../internal/fetch/enum/fetcher.go) — canonical fetcher pattern to follow
- [internal/fetch/testhelpers/setup.go](../../../internal/fetch/testhelpers/setup.go) — existing fixture setup to extend

### Secondary (MEDIUM confidence)
- [PostgreSQL source: include/catalog/pg_trigger.h](https://github.com/postgres/postgres/blob/master/src/include/catalog/pg_trigger.h) — tgtype bitmask constants (BEFORE=0x02, INSERT=0x04, DELETE=0x08, UPDATE=0x10, TRUNCATE=0x20, INSTEAD_OF=0x40)
- [pgbackup_spec.md section 10.2](../../../docs/pre/pgbackup_spec.md) — ViewDef, FunctionDef, TriggerDef, PolicyDef struct shapes

### Tertiary (LOW confidence)
- None — all key claims verified against official PostgreSQL documentation

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; same pgx v5 patterns as Phase 2
- pg_catalog queries (views, matviews, functions): HIGH — verified against official PostgreSQL documentation
- pg_catalog queries (triggers, policies): HIGH — catalog tables documented; tgtype bitmask constants confirmed from PostgreSQL source
- Struct expansion design: HIGH — follows spec section 10.2 and success criteria fields verbatim
- Test fixture DDL: MEDIUM — DDL is straightforward but trigger function with plpgsql requires the language to be available in the test database

**Research date:** 2026-03-11
**Valid until:** 2026-06-11 (pg_catalog stable; system views pg_views/pg_matviews stable since PostgreSQL 9.3+)
