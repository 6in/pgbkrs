package compositetype

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// SchemaFetcher fetches composite type definitions from pg_catalog.
// It excludes table row types (relkind='c' filter) so that only
// explicitly created composite types are returned.
type SchemaFetcher struct{}

// Fetch returns all composite types in the given schema as []core.ObjectDef.
// Each entry is a core.TypeDef with Fields listing attributes in attnum order.
// Table-row composite types (automatically created by PostgreSQL for every table)
// are excluded by filtering c.relkind = 'c'.
func (f *SchemaFetcher) Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]core.ObjectDef, error) {
	const query = `
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
  AND c.relkind = 'c'
  AND n.nspname = $1
  AND a.attnum > 0
  AND NOT a.attisdropped
ORDER BY t.typname, a.attnum`

	rows, err := conn.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type rowResult struct {
		OID       uint32
		Name      string
		FieldName string
		FieldType string
	}

	// Group rows by OID to build TypeDef.Fields slice per type.
	// The ORDER BY t.typname, a.attnum guarantees rows are grouped together.
	var (
		result  []core.ObjectDef
		current *core.TypeDef
		lastOID uint32
	)

	for rows.Next() {
		var r rowResult
		if err := rows.Scan(&r.OID, &r.Name, &r.FieldName, &r.FieldType); err != nil {
			return nil, err
		}

		if r.OID != lastOID {
			// Save previous type if any
			if current != nil {
				result = append(result, *current)
			}
			// Start a new TypeDef
			current = &core.TypeDef{
				ObjectHeader: core.ObjectHeader{
					Kind:   core.KindType,
					Schema: schema,
					Name:   r.Name,
				},
			}
			lastOID = r.OID
		}

		current.Fields = append(current.Fields, core.CompositeField{
			Name: r.FieldName,
			Type: r.FieldType,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Don't forget the last type
	if current != nil {
		result = append(result, *current)
	}

	return result, nil
}

// Compile-time check: SchemaFetcher must implement core.SchemaFetcher.
var _ core.SchemaFetcher = (*SchemaFetcher)(nil)
