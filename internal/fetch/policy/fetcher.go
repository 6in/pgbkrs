package policy

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.SchemaFetcher = (*SchemaFetcher)(nil)

// SchemaFetcher fetches RLS policy definitions from pg_catalog.
// Returns all policies across all tables in the given schema.
// Each PolicyDef identifies its table via the TableName field.
type SchemaFetcher struct{}

// fetchQuery resolves polroles oid[] to role name strings.
// When polroles = '{0}', the policy applies to PUBLIC (OID 0 is the sentinel, not a real role).
// LATERAL unnest is used to expand polroles for non-PUBLIC policies; pg_get_userbyid converts OID to name.
const fetchQuery = `
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
`

// Fetch returns PolicyDef objects for all RLS policies in the given schema.
// Roles contains ["PUBLIC"] for policies created with TO PUBLIC (polroles sentinel OID 0).
func (f *SchemaFetcher) Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]core.ObjectDef, error) {
	rows, err := conn.Query(ctx, fetchQuery, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defs []core.ObjectDef
	for rows.Next() {
		var (
			name      string
			tableName string
			command   string
			roles     []string
			using     string
			withCheck string
		)
		if err := rows.Scan(&name, &tableName, &command, &roles, &using, &withCheck); err != nil {
			return nil, err
		}
		defs = append(defs, core.PolicyDef{
			ObjectHeader: core.ObjectHeader{
				Kind:   core.KindPolicy,
				Schema: schema,
				Name:   name,
			},
			TableName: tableName,
			Command:   command,
			Roles:     roles,
			Using:     using,
			WithCheck: withCheck,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return defs, nil
}
