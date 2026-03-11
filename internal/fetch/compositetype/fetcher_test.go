package compositetype_test

import (
	"context"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/fetch/compositetype"
	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
)

func TestFetchCompositeType(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &compositetype.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Find address_type in results
	var addressType *core.TypeDef
	for _, def := range defs {
		td, ok := def.(core.TypeDef)
		if !ok {
			t.Fatalf("expected core.TypeDef, got %T", def)
		}
		if td.Name == "address_type" {
			td := td
			addressType = &td
		}
		// Assert that table-row composite types are excluded
		if td.Name == "simple_table" {
			t.Errorf("simple_table should NOT appear in composite type results (table row types must be excluded via relkind='c' filter)")
		}
	}

	if addressType == nil {
		t.Fatalf("address_type not found in results; got %d defs", len(defs))
	}

	// Verify exactly 3 fields in order
	if len(addressType.Fields) != 3 {
		t.Fatalf("address_type.Fields: want 3, got %d", len(addressType.Fields))
	}

	wantFields := []core.CompositeField{
		{Name: "street", Type: "text"},
		{Name: "city", Type: "text"},
		{Name: "zip", Type: "text"},
	}
	for i, want := range wantFields {
		got := addressType.Fields[i]
		if got.Name != want.Name {
			t.Errorf("Fields[%d].Name: want %q, got %q", i, want.Name, got.Name)
		}
		if got.Type != want.Type {
			t.Errorf("Fields[%d].Type: want %q, got %q", i, want.Type, got.Type)
		}
	}
}
