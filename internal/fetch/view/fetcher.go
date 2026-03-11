package view

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.SchemaFetcher = (*SchemaFetcher)(nil)

// SchemaFetcher fetches view definitions from pg_catalog.
type SchemaFetcher struct{}

const fetchQuery = `
SELECT viewname AS name, viewowner AS owner, definition
FROM pg_views
WHERE schemaname = $1
ORDER BY viewname
`

// Fetch returns ViewDef objects for all views in the given schema.
// Definition contains the SELECT body only (as returned by pg_views.definition).
// The full CREATE VIEW DDL is reconstructed by DDLGenerator in Phase 4.
func (f *SchemaFetcher) Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]core.ObjectDef, error) {
	rows, err := conn.Query(ctx, fetchQuery, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defs []core.ObjectDef
	for rows.Next() {
		var (
			name       string
			owner      string
			definition string
		)
		if err := rows.Scan(&name, &owner, &definition); err != nil {
			return nil, err
		}
		defs = append(defs, core.ViewDef{
			ObjectHeader: core.ObjectHeader{
				Kind:   core.KindView,
				Schema: schema,
				Name:   name,
			},
			Definition: definition,
			Owner:      owner,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return defs, nil
}
