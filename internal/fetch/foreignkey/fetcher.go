package foreignkey

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.SchemaFetcher = (*SchemaFetcher)(nil)

// SchemaFetcher fetches foreign key constraint definitions from pg_catalog.
type SchemaFetcher struct{}

// fkRow holds the raw data from the pg_constraint query for foreign keys.
type fkRow struct {
	ConName      string `db:"conname"`
	SourceTable  string `db:"source_table"`
	TargetSchema string `db:"target_schema"`
	TargetTable  string `db:"target_table"`
	Definition   string `db:"definition"`
}

// Fetch returns all foreign key constraints in the given schema.
func (f *SchemaFetcher) Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]core.ObjectDef, error) {
	query := `
SELECT c.conname, cls.relname AS source_table,
       ns2.nspname AS target_schema, cls2.relname AS target_table,
       pg_get_constraintdef(c.oid) AS definition
FROM pg_constraint c
JOIN pg_class cls ON cls.oid = c.conrelid
JOIN pg_namespace ns ON ns.oid = cls.relnamespace
JOIN pg_class cls2 ON cls2.oid = c.confrelid
JOIN pg_namespace ns2 ON ns2.oid = cls2.relnamespace
WHERE c.contype = 'f' AND ns.nspname = $1
ORDER BY cls.relname, c.conname`

	rows, err := conn.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}
	raw, err := pgx.CollectRows(rows, pgx.RowToStructByName[fkRow])
	if err != nil {
		return nil, err
	}

	result := make([]core.ObjectDef, len(raw))
	for i, r := range raw {
		result[i] = &core.ForeignKeyDef{
			ObjectHeader: core.ObjectHeader{
				Kind:   core.KindForeignKey,
				Schema: schema,
				Name:   r.ConName,
			},
			SourceTable:  r.SourceTable,
			TargetSchema: r.TargetSchema,
			TargetTable:  r.TargetTable,
			Definition:   r.Definition,
		}
	}
	return result, nil
}
