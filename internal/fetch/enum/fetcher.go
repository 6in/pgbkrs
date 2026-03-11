package enum

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.SchemaFetcher = (*SchemaFetcher)(nil)

// SchemaFetcher fetches enum type definitions from pg_catalog.
type SchemaFetcher struct{}

const fetchQuery = `
SELECT
    t.typname AS name,
    array_agg(e.enumlabel ORDER BY e.enumsortorder) AS labels
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
JOIN pg_enum e ON e.enumtypid = t.oid
WHERE t.typtype = 'e'
  AND n.nspname = $1
GROUP BY t.oid, t.typname
ORDER BY t.typname
`

// Fetch returns EnumDef objects for all enum types in the given schema.
func (f *SchemaFetcher) Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]core.ObjectDef, error) {
	rows, err := conn.Query(ctx, fetchQuery, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defs []core.ObjectDef
	for rows.Next() {
		var (
			name   string
			labels []string
		)
		if err := rows.Scan(&name, &labels); err != nil {
			return nil, err
		}
		defs = append(defs, core.EnumDef{
			ObjectHeader: core.ObjectHeader{
				Kind:   core.KindEnum,
				Schema: schema,
				Name:   name,
			},
			Labels: labels,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return defs, nil
}
