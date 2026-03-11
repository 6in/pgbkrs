package domain_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/fetch/domain"
	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
)

func TestFetchDomain(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &domain.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Find positive_int in results
	var positiveInt *core.DomainDef
	for _, def := range defs {
		dd, ok := def.(core.DomainDef)
		if !ok {
			t.Fatalf("expected core.DomainDef, got %T", def)
		}
		if dd.Name == "positive_int" {
			dd := dd
			positiveInt = &dd
		}
	}

	if positiveInt == nil {
		t.Fatalf("positive_int not found in results; got %d defs", len(defs))
	}

	// positive_int AS integer NOT NULL CHECK (VALUE > 0)
	if positiveInt.BaseType != "integer" {
		t.Errorf("BaseType: want %q, got %q", "integer", positiveInt.BaseType)
	}

	// NOT NULL means Nullable = false
	if positiveInt.Nullable {
		t.Errorf("Nullable: want false (positive_int is NOT NULL), got true")
	}

	// No default was specified in the fixture
	if positiveInt.Default != "" {
		t.Errorf("Default: want empty string, got %q", positiveInt.Default)
	}

	// PostgreSQL auto-generates a check constraint name — must be non-empty
	if positiveInt.CheckName == "" {
		t.Errorf("CheckName: want non-empty (PostgreSQL auto-generates constraint name), got empty")
	}

	// pg_get_constraintdef returns something like CHECK ((VALUE > 0))
	if !strings.Contains(positiveInt.CheckDefinition, "VALUE > 0") {
		t.Errorf("CheckDefinition: want to contain %q, got %q", "VALUE > 0", positiveInt.CheckDefinition)
	}
}
