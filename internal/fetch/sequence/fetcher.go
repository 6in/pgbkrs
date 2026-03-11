package sequence

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.SchemaFetcher = (*SchemaFetcher)(nil)

// SchemaFetcher fetches sequence definitions from pg_sequences.
type SchemaFetcher struct{}

const fetchQuery = `
SELECT
    s.sequencename  AS name,
    s.start_value,
    s.min_value,
    s.max_value,
    s.increment_by,
    s.cycle,
    s.cache_size     AS cache,
    COALESCE(s.last_value, s.start_value - s.increment_by) AS last_value,
    s.last_value IS NOT NULL AS is_called
FROM pg_sequences s
WHERE s.schemaname = $1
ORDER BY s.sequencename
`

// Fetch returns SequenceDef objects for all sequences in the given schema.
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
			startValue  int64
			minValue    int64
			maxValue    int64
			incrementBy int64
			cycle       bool
			cache       int64
			lastValue   int64
			isCalled    bool
		)
		if err := rows.Scan(
			&name,
			&startValue,
			&minValue,
			&maxValue,
			&incrementBy,
			&cycle,
			&cache,
			&lastValue,
			&isCalled,
		); err != nil {
			return nil, err
		}
		defs = append(defs, core.SequenceDef{
			ObjectHeader: core.ObjectHeader{
				Kind:   core.KindSequence,
				Schema: schema,
				Name:   name,
			},
			StartValue:  startValue,
			MinValue:    minValue,
			MaxValue:    maxValue,
			IncrementBy: incrementBy,
			Cycle:       cycle,
			Cache:       cache,
			LastValue:   lastValue,
			IsCalled:    isCalled,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return defs, nil
}
