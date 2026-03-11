package policy_test

import (
	"context"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/fetch/policy"
	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
)

func TestFetchPolicy(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &policy.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	var found *core.PolicyDef
	for _, def := range defs {
		if pd, ok := def.(core.PolicyDef); ok && pd.Name == "tenant_isolation" {
			copy := pd
			found = &copy
			break
		}
	}
	if found == nil {
		t.Fatalf("tenant_isolation policy not found in %d defs", len(defs))
	}
	if found.Command != "ALL" {
		t.Errorf("Command: got %q, want %q", found.Command, "ALL")
	}
	if found.TableName != "rls_table" {
		t.Errorf("TableName: got %q, want %q", found.TableName, "rls_table")
	}
	if found.Using == "" {
		t.Error("Using expression must be non-empty (policy has USING (true))")
	}
	if found.WithCheck == "" {
		t.Error("WithCheck expression must be non-empty (policy has WITH CHECK (true))")
	}
}

func TestFetchPolicyPublicRoles(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	// tenant_isolation was created with TO PUBLIC — Roles must be ["PUBLIC"], not empty.
	f := &policy.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	var found *core.PolicyDef
	for _, def := range defs {
		if pd, ok := def.(core.PolicyDef); ok && pd.Name == "tenant_isolation" {
			copy := pd
			found = &copy
			break
		}
	}
	if found == nil {
		t.Fatalf("tenant_isolation not found")
	}
	if len(found.Roles) == 0 {
		t.Fatal("Roles must not be empty for a PUBLIC policy — polroles={0} sentinel must map to [\"PUBLIC\"]")
	}
	if found.Roles[0] != "PUBLIC" {
		t.Errorf("Roles[0]: got %q, want %q", found.Roles[0], "PUBLIC")
	}
}
