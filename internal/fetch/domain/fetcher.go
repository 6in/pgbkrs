package domain

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// SchemaFetcher fetches domain definitions from pg_catalog.
type SchemaFetcher struct{}

// Fetch returns all domain definitions in the given schema as []core.ObjectDef.
// Each entry is a core.DomainDef with BaseType, Nullable, Default, CheckName,
// and CheckDefinition populated from pg_catalog.
func (f *SchemaFetcher) Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]core.ObjectDef, error) {
	const query = `
SELECT
    t.oid,
    t.typname AS name,
    format_type(t.typbasetype, t.typtypmod) AS base_type,
    NOT t.typnotnull AS nullable,
    COALESCE(pg_get_expr(t.typdefaultbin, 0), '') AS default_expr,
    COALESCE(c.conname, '') AS check_name,
    COALESCE(pg_get_constraintdef(c.oid), '') AS check_definition
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
LEFT JOIN pg_constraint c ON c.contypid = t.oid AND c.contype = 'c'
WHERE t.typtype = 'd'
  AND n.nspname = $1
ORDER BY t.typname`

	rows, err := conn.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []core.ObjectDef

	for rows.Next() {
		var (
			oid             uint32
			name            string
			baseType        string
			nullable        bool
			defaultExpr     string
			checkName       string
			checkDefinition string
		)
		if err := rows.Scan(&oid, &name, &baseType, &nullable, &defaultExpr, &checkName, &checkDefinition); err != nil {
			return nil, err
		}

		result = append(result, core.DomainDef{
			ObjectHeader: core.ObjectHeader{
				Kind:   core.KindDomain,
				Schema: schema,
				Name:   name,
			},
			BaseType:        baseType,
			Nullable:        nullable,
			Default:         defaultExpr,
			CheckName:       checkName,
			CheckDefinition: checkDefinition,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Compile-time check: SchemaFetcher must implement core.SchemaFetcher.
var _ core.SchemaFetcher = (*SchemaFetcher)(nil)
