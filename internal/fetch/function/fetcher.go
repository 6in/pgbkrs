package function

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.SchemaFetcher = (*SchemaFetcher)(nil)

// SchemaFetcher fetches function definitions from pg_catalog.
// Only regular functions (prokind='f') are returned.
// Procedures (prokind='p'), aggregates ('a'), and window functions ('w') are excluded.
type SchemaFetcher struct{}

const fetchQuery = `
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
`

// Fetch returns FunctionDef objects for all regular functions in the given schema.
// Definition is the complete CREATE OR REPLACE FUNCTION statement from pg_get_functiondef.
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
			definition string
			argTypes   string
			returnType string
		)
		if err := rows.Scan(&name, &definition, &argTypes, &returnType); err != nil {
			return nil, err
		}
		defs = append(defs, core.FunctionDef{
			ObjectHeader: core.ObjectHeader{
				Kind:   core.KindFunction,
				Schema: schema,
				Name:   name,
			},
			Definition: definition,
			ArgTypes:   argTypes,
			ReturnType: returnType,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return defs, nil
}
