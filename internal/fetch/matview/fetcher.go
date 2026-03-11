package matview

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.SchemaFetcher = (*SchemaFetcher)(nil)

// SchemaFetcher fetches materialized view definitions from pg_catalog.
type SchemaFetcher struct{}

const fetchQuery = `
SELECT matviewname AS name, matviewowner AS owner, definition, ispopulated
FROM pg_matviews
WHERE schemaname = $1
ORDER BY matviewname
`

// Fetch returns MaterializedViewDef objects for all materialized views in the given schema.
// IsPopulated is false when the matview was created WITH NO DATA and has never been refreshed.
func (f *SchemaFetcher) Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]core.ObjectDef, error) {
	rows, err := conn.Query(ctx, fetchQuery, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defs []core.ObjectDef
	for rows.Next() {
		var (
			name        string
			owner       string
			definition  string
			isPopulated bool
		)
		if err := rows.Scan(&name, &owner, &definition, &isPopulated); err != nil {
			return nil, err
		}
		defs = append(defs, core.MaterializedViewDef{
			ObjectHeader: core.ObjectHeader{
				Kind:   core.KindMaterializedView,
				Schema: schema,
				Name:   name,
			},
			Definition:  definition,
			Owner:       owner,
			IsPopulated: isPopulated,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return defs, nil
}
