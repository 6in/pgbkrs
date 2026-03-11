package view_test

import (
	"context"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
	"github.com/pgbkrs/pgbackup/internal/fetch/view"
)

func TestFetchView(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &view.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	var found *core.ViewDef
	for _, def := range defs {
		if vd, ok := def.(core.ViewDef); ok && vd.Name == "active_items" {
			copy := vd
			found = &copy
			break
		}
	}
	if found == nil {
		t.Fatalf("active_items view not found in %d defs", len(defs))
	}
	if found.Definition == "" {
		t.Error("ViewDef.Definition must be non-empty")
	}
	if found.Owner == "" {
		t.Error("ViewDef.Owner must be non-empty")
	}
}
