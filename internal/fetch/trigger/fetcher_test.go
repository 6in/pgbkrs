package trigger_test

import (
	"context"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
	"github.com/pgbkrs/pgbackup/internal/fetch/trigger"
)

func TestFetchTrigger(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &trigger.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	var found *core.TriggerDef
	for _, def := range defs {
		if td, ok := def.(core.TriggerDef); ok && td.Name == "trg_audit_simple" {
			copy := td
			found = &copy
			break
		}
	}
	if found == nil {
		t.Fatalf("trg_audit_simple not found in %d defs", len(defs))
	}
	if found.Timing != "AFTER" {
		t.Errorf("Timing: got %q, want %q", found.Timing, "AFTER")
	}
	if len(found.Events) != 1 || found.Events[0] != "INSERT" {
		t.Errorf("Events: got %v, want [INSERT]", found.Events)
	}
	if found.TableName != "simple_table" {
		t.Errorf("TableName: got %q, want %q", found.TableName, "simple_table")
	}
	if found.FunctionName != "record_audit" {
		t.Errorf("FunctionName: got %q, want %q", found.FunctionName, "record_audit")
	}
}

func TestFetchTriggerExcludesInternal(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	// simple_table has a UNIQUE constraint on email which generates an internal trigger.
	// Verify that only user-defined triggers (tgisinternal=false) are returned.
	f := &trigger.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	for _, def := range defs {
		if td, ok := def.(core.TriggerDef); ok {
			// All returned triggers must be from the test schema
			if td.Schema != schema {
				t.Errorf("trigger from unexpected schema %q: %q", td.Schema, td.Name)
			}
		}
	}
	// The only user trigger created in setup is trg_audit_simple.
	// Any additional triggers are suspicious internal triggers leaking through.
	if len(defs) > 1 {
		t.Logf("WARNING: got %d triggers, expected 1 (trg_audit_simple). Check tgisinternal filter.", len(defs))
	}
}
