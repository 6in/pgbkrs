package restore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// liveObject represents a live database object captured during a snapshot.
type liveObject struct {
	Schema string
	Kind   string
	Name   string
}

// snapshotLiveObjects queries pg_catalog for all live objects in the given schemas.
// It returns tables, views, materialized_views, sequences, functions, types, domains, and enums.
func snapshotLiveObjects(ctx context.Context, conn *pgx.Conn, schemas []string) ([]liveObject, error) {
	const query = `
SELECT n.nspname AS schema, 'table' AS kind, c.relname AS name
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'r' AND n.nspname = ANY($1)

UNION ALL

SELECT n.nspname, 'view', c.relname
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'v' AND n.nspname = ANY($1)

UNION ALL

SELECT n.nspname, 'materialized_view', c.relname
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'm' AND n.nspname = ANY($1)

UNION ALL

SELECT n.nspname, 'sequence', c.relname
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'S' AND n.nspname = ANY($1)

UNION ALL

SELECT n.nspname, 'function', p.proname
FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE n.nspname = ANY($1)

UNION ALL

SELECT n.nspname, 'type', t.typname
FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE t.typtype = 'c' AND n.nspname = ANY($1)

UNION ALL

SELECT n.nspname, 'domain', t.typname
FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE t.typtype = 'd' AND n.nspname = ANY($1)

UNION ALL

SELECT n.nspname, 'enum', t.typname
FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE t.typtype = 'e' AND n.nspname = ANY($1)
`
	rows, err := conn.Query(ctx, query, schemas)
	if err != nil {
		return nil, fmt.Errorf("snapshot live objects: %w", err)
	}
	defer rows.Close()

	var result []liveObject
	for rows.Next() {
		var obj liveObject
		if err := rows.Scan(&obj.Schema, &obj.Kind, &obj.Name); err != nil {
			return nil, fmt.Errorf("scan live object: %w", err)
		}
		result = append(result, obj)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate live objects: %w", err)
	}
	return result, nil
}

// detectLeaks returns live objects that are still present after the DROP wave
// and were in the shouldBeDropped set. These represent objects that failed to drop.
func detectLeaks(postDrop []liveObject, shouldBeDropped map[string]bool) []liveObject {
	var leaked []liveObject
	for _, obj := range postDrop {
		id := obj.Schema + "." + obj.Name
		if shouldBeDropped[id] {
			leaked = append(leaked, obj)
		}
	}
	return leaked
}
