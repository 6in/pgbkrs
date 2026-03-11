package policy_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/policy"
)

func TestGenerateDDL(t *testing.T) {
	g := &policy.DDLGenerator{}
	def := &core.PolicyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "tenant_isolation"},
		TableName:    "orders",
		Command:      "ALL",
		Roles:        []string{"app_user"},
		Using:        "(tenant_id = current_setting('app.tenant_id')::int)",
		WithCheck:    "(tenant_id = current_setting('app.tenant_id')::int)",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) == 0 {
		t.Fatal("expected at least one DDL statement")
	}

	combined := strings.Join(stmts, "\n")
	if !strings.Contains(combined, "CREATE POLICY") {
		t.Error("expected CREATE POLICY statement")
	}
	if !strings.Contains(combined, "ON") {
		t.Error("expected ON table in policy DDL")
	}
	if !strings.Contains(combined, "USING") {
		t.Error("expected USING clause in policy DDL")
	}
	if !strings.Contains(combined, "WITH CHECK") {
		t.Error("expected WITH CHECK clause in policy DDL")
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &policy.DDLGenerator{}
	def := &core.PolicyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "tenant_isolation"},
		TableName:    "orders",
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) == 0 {
		t.Fatal("expected at least one DDL statement")
	}

	combined := strings.Join(stmts, "\n")
	if !strings.Contains(combined, "DROP POLICY IF EXISTS") {
		t.Error("expected DROP POLICY IF EXISTS statement")
	}
	if !strings.Contains(combined, "ON") {
		t.Error("expected ON table in DROP POLICY statement")
	}
}
