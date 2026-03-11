package function_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/fetch/function"
	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
)

func TestFetchFunction(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &function.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Find add_values (prokind='f' regular function)
	var found *core.FunctionDef
	for _, def := range defs {
		if fd, ok := def.(core.FunctionDef); ok && fd.Name == "add_values" {
			copy := fd
			found = &copy
			break
		}
	}
	if found == nil {
		t.Fatalf("add_values not found in %d defs", len(defs))
	}
	if found.Definition == "" {
		t.Error("FunctionDef.Definition must be non-empty")
	}
	if !strings.Contains(found.Definition, "CREATE") {
		t.Errorf("FunctionDef.Definition should start with CREATE, got: %q", found.Definition[:min(80, len(found.Definition))])
	}
	if found.ArgTypes == "" {
		t.Error("FunctionDef.ArgTypes must be non-empty for add_values(a integer, b integer)")
	}
	if found.ReturnType == "" {
		t.Error("FunctionDef.ReturnType must be non-empty for add_values returning integer")
	}
	// record_audit is a trigger function (returns trigger), it should also appear
	var foundAudit bool
	for _, def := range defs {
		if fd, ok := def.(core.FunctionDef); ok && fd.Name == "record_audit" {
			foundAudit = true
			break
		}
	}
	if !foundAudit {
		t.Error("record_audit trigger function not found — prokind='f' functions returning trigger are still regular functions")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
