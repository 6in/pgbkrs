package table

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.SchemaFetcher = (*SchemaFetcher)(nil)

// SchemaFetcher fetches table definitions from pg_catalog.
type SchemaFetcher struct{}

// Fetch returns all tables (including partitioned tables) in the given schema.
func (f *SchemaFetcher) Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]core.ObjectDef, error) {
	tables, err := listTables(ctx, conn, schema)
	if err != nil {
		return nil, err
	}

	result := make([]core.ObjectDef, 0, len(tables))
	for _, t := range tables {
		def, err := buildTableDef(ctx, conn, schema, t)
		if err != nil {
			return nil, err
		}
		result = append(result, def)
	}
	return result, nil
}

// tableRow holds the raw data from the pg_class query.
type tableRow struct {
	OID        uint32 `db:"oid"`
	Name       string `db:"name"`
	Relkind    string `db:"relkind"`
	RLSEnabled bool   `db:"rls_enabled"`
}

// listTables returns all tables and partitioned tables in the schema.
func listTables(ctx context.Context, conn *pgx.Conn, schema string) ([]tableRow, error) {
	query := `
SELECT c.oid, c.relname AS name, c.relkind::text, c.relrowsecurity AS rls_enabled
FROM pg_class c
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE n.nspname = $1 AND c.relkind IN ('r', 'p')
ORDER BY c.relname`

	rows, err := conn.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[tableRow])
}

// buildTableDef constructs a full core.TableDef for one table.
func buildTableDef(ctx context.Context, conn *pgx.Conn, schema string, t tableRow) (*core.TableDef, error) {
	columns, err := fetchColumns(ctx, conn, t.OID)
	if err != nil {
		return nil, err
	}

	constraints, err := fetchConstraints(ctx, conn, t.OID)
	if err != nil {
		return nil, err
	}

	indexes, err := fetchIndexes(ctx, conn, t.OID)
	if err != nil {
		return nil, err
	}

	var partitioning *core.PartitionDef
	if t.Relkind == "p" {
		partitioning, err = fetchPartitioning(ctx, conn, t.OID)
		if err != nil {
			return nil, err
		}
	}

	return &core.TableDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.KindTable,
			Schema: schema,
			Name:   t.Name,
		},
		Columns:      columns,
		Constraints:  constraints,
		Indexes:      indexes,
		Partitioning: partitioning,
		RLS:          core.RLSDef{Enabled: t.RLSEnabled},
	}, nil
}

// columnRow holds raw column data from pg_attribute.
type columnRow struct {
	Name        string `db:"name"`
	Type        string `db:"type"`
	Nullable    bool   `db:"nullable"`
	DefaultExpr string `db:"default_expr"`
}

// fetchColumns returns all visible columns for the given table OID.
func fetchColumns(ctx context.Context, conn *pgx.Conn, tableOID uint32) ([]core.ColumnDef, error) {
	query := `
SELECT a.attname AS name,
       format_type(a.atttypid, a.atttypmod) AS type,
       NOT a.attnotnull AS nullable,
       COALESCE(pg_get_expr(d.adbin, d.adrelid), '') AS default_expr
FROM pg_attribute a
LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
WHERE a.attrelid = $1 AND a.attnum > 0 AND NOT a.attisdropped
ORDER BY a.attnum`

	rows, err := conn.Query(ctx, query, tableOID)
	if err != nil {
		return nil, err
	}
	raw, err := pgx.CollectRows(rows, pgx.RowToStructByName[columnRow])
	if err != nil {
		return nil, err
	}

	cols := make([]core.ColumnDef, len(raw))
	for i, r := range raw {
		cols[i] = core.ColumnDef{
			Name:     r.Name,
			Type:     r.Type,
			Nullable: r.Nullable,
			Default:  r.DefaultExpr,
		}
	}
	return cols, nil
}

// constraintRow holds raw constraint data from pg_constraint.
type constraintRow struct {
	Name       string `db:"name"`
	Contype    string `db:"contype"`
	Definition string `db:"definition"`
}

// fetchConstraints returns PK, unique, and check constraints for the given table OID.
func fetchConstraints(ctx context.Context, conn *pgx.Conn, tableOID uint32) (core.ConstraintsDef, error) {
	query := `
SELECT c.conname AS name, c.contype::text, pg_get_constraintdef(c.oid) AS definition
FROM pg_constraint c
WHERE c.conrelid = $1 AND c.contype IN ('p', 'u', 'c')
ORDER BY c.contype, c.conname`

	rows, err := conn.Query(ctx, query, tableOID)
	if err != nil {
		return core.ConstraintsDef{}, err
	}
	raw, err := pgx.CollectRows(rows, pgx.RowToStructByName[constraintRow])
	if err != nil {
		return core.ConstraintsDef{}, err
	}

	var result core.ConstraintsDef
	for _, r := range raw {
		switch r.Contype {
		case "p":
			result.PrimaryKey = &core.PrimaryKeyDef{
				Name:    r.Name,
				Columns: parsePKColumns(r.Definition),
			}
		case "u":
			result.Unique = append(result.Unique, core.UniqueConstraintDef{
				Name:       r.Name,
				Definition: r.Definition,
			})
		case "c":
			result.Check = append(result.Check, core.CheckConstraintDef{
				Name:       r.Name,
				Definition: r.Definition,
			})
		}
	}
	return result, nil
}

// parsePKColumns extracts column names from a PRIMARY KEY constraint definition.
// e.g. "PRIMARY KEY (id, tenant_id)" -> ["id", "tenant_id"]
func parsePKColumns(definition string) []string {
	start := strings.Index(definition, "(")
	end := strings.LastIndex(definition, ")")
	if start < 0 || end <= start {
		return nil
	}
	inner := definition[start+1 : end]
	parts := strings.Split(inner, ",")
	cols := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			cols = append(cols, trimmed)
		}
	}
	return cols
}

// indexRow holds raw index data from pg_index.
type indexRow struct {
	Name       string `db:"name"`
	Method     string `db:"method"`
	Definition string `db:"definition"`
}

// fetchIndexes returns non-constraint-backing indexes for the given table OID.
func fetchIndexes(ctx context.Context, conn *pgx.Conn, tableOID uint32) ([]core.IndexDef, error) {
	query := `
SELECT ic.relname AS name, am.amname AS method, pg_get_indexdef(ix.indexrelid) AS definition
FROM pg_index ix
JOIN pg_class ic ON ix.indexrelid = ic.oid
JOIN pg_am am ON ic.relam = am.oid
WHERE ix.indrelid = $1
  AND NOT ix.indisprimary
  AND NOT EXISTS (
      SELECT 1 FROM pg_constraint con
      WHERE con.conindid = ix.indexrelid AND con.contype IN ('p', 'u')
  )
ORDER BY ic.relname`

	rows, err := conn.Query(ctx, query, tableOID)
	if err != nil {
		return nil, err
	}
	raw, err := pgx.CollectRows(rows, pgx.RowToStructByName[indexRow])
	if err != nil {
		return nil, err
	}

	idxs := make([]core.IndexDef, len(raw))
	for i, r := range raw {
		idxs[i] = core.IndexDef{
			Name:       r.Name,
			Method:     r.Method,
			Definition: r.Definition,
		}
	}
	return idxs, nil
}

// partStrategyRow holds partition strategy data.
type partStrategyRow struct {
	Partstrat string `db:"partstrat"`
}

// fetchPartitioning returns partitioning info for a partitioned table (relkind='p').
func fetchPartitioning(ctx context.Context, conn *pgx.Conn, tableOID uint32) (*core.PartitionDef, error) {
	// Step 5a: get strategy
	stratQuery := `SELECT pt.partstrat::text FROM pg_partitioned_table pt WHERE pt.partrelid = $1`
	var strat partStrategyRow
	rows, err := conn.Query(ctx, stratQuery, tableOID)
	if err != nil {
		return nil, err
	}
	strats, err := pgx.CollectRows(rows, pgx.RowToStructByName[partStrategyRow])
	if err != nil {
		return nil, err
	}
	if len(strats) == 0 {
		return nil, nil
	}
	strat = strats[0]

	stratStr := mapPartStrategy(strat.Partstrat)

	// Step 5b: get children
	childQuery := `
SELECT child.relname AS name
FROM pg_inherits inh
JOIN pg_class child ON child.oid = inh.inhrelid
WHERE inh.inhparent = $1 ORDER BY child.relname`

	type childRow struct {
		Name string `db:"name"`
	}
	rows2, err := conn.Query(ctx, childQuery, tableOID)
	if err != nil {
		return nil, err
	}
	childRows, err := pgx.CollectRows(rows2, pgx.RowToStructByName[childRow])
	if err != nil {
		return nil, err
	}

	children := make([]string, len(childRows))
	for i, c := range childRows {
		children[i] = c.Name
	}

	return &core.PartitionDef{
		Strategy: stratStr,
		Children: children,
	}, nil
}

// mapPartStrategy converts pg_partitioned_table.partstrat to a human-readable string.
func mapPartStrategy(s string) string {
	switch s {
	case "r":
		return "range"
	case "l":
		return "list"
	case "h":
		return "hash"
	default:
		return s
	}
}
