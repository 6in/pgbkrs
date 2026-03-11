package matview_test

import (
	"context"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/fetch/matview"
	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
)

func TestFetchMatView(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &matview.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	var found *core.MaterializedViewDef
	for _, def := range defs {
		if mv, ok := def.(core.MaterializedViewDef); ok && mv.Name == "mv_items" {
			copy := mv
			found = &copy
			break
		}
	}
	if found == nil {
		t.Fatalf("mv_items matview not found in %d defs", len(defs))
	}
	if found.Definition == "" {
		t.Error("MaterializedViewDef.Definition must be non-empty")
	}
	if found.Owner == "" {
		t.Error("MaterializedViewDef.Owner must be non-empty")
	}
	if !found.IsPopulated {
		t.Error("mv_items was created without WITH NO DATA so IsPopulated must be true")
	}
}

func TestFetchMatViewUnpopulated(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	// Create an unpopulated matview directly in the test
	ctx := context.Background()
	_, err := conn.Exec(ctx,
		"CREATE MATERIALIZED VIEW "+schema+".mv_unpopulated AS SELECT 1 AS n WITH NO DATA")
	if err != nil {
		t.Fatalf("create unpopulated matview: %v", err)
	}

	f := &matview.SchemaFetcher{}
	defs, err := f.Fetch(ctx, conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	var found *core.MaterializedViewDef
	for _, def := range defs {
		if mv, ok := def.(core.MaterializedViewDef); ok && mv.Name == "mv_unpopulated" {
			copy := mv
			found = &copy
			break
		}
	}
	if found == nil {
		t.Fatalf("mv_unpopulated not found in %d defs", len(defs))
	}
	if found.IsPopulated {
		t.Error("mv_unpopulated was created WITH NO DATA so IsPopulated must be false")
	}
}
