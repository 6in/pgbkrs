package enum_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/fetch/enum"
	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
)

func TestFetchEnum(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &enum.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Find status_enum in returned defs
	var found *core.EnumDef
	for _, def := range defs {
		if ed, ok := def.(core.EnumDef); ok && ed.Name == "status_enum" {
			copy := ed
			found = &copy
			break
		}
	}
	if found == nil {
		t.Fatalf("status_enum not found in %d defs", len(defs))
	}

	want := []string{"active", "inactive", "pending"}
	if !reflect.DeepEqual(found.Labels, want) {
		t.Errorf("Labels: got %v, want %v", found.Labels, want)
	}

	// Verify only the test schema's ENUMs are returned
	for _, def := range defs {
		if ed, ok := def.(core.EnumDef); ok {
			if ed.Schema != schema {
				t.Errorf("unexpected enum from schema %q: %q", ed.Schema, ed.Name)
			}
		}
	}
}
