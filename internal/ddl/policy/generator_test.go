package policy_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/policy"
)

func TestGenerateDDL_Full(t *testing.T) {
	g := &policy.DDLGenerator{}
	def := core.PolicyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "tenant_isolation"},
		TableName:    "orders",
		Command:      "ALL",
		Roles:        []string{"app_user", "admin"},
		Using:        "(tenant_id = current_setting('app.tenant_id')::int)",
		WithCheck:    "(tenant_id = current_setting('app.tenant_id')::int)",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	ddl := stmts[0]
	if !strings.HasPrefix(ddl, "CREATE POLICY tenant_isolation ON public.orders") {
		t.Errorf("expected CREATE POLICY tenant_isolation ON public.orders prefix, got: %s", ddl)
	}
	if !strings.Contains(ddl, "FOR ALL") {
		t.Errorf("expected FOR ALL, got: %s", ddl)
	}
	if !strings.Contains(ddl, "TO app_user, admin") {
		t.Errorf("expected TO app_user, admin, got: %s", ddl)
	}
	if !strings.Contains(ddl, "USING ((tenant_id = current_setting('app.tenant_id')::int))") {
		t.Errorf("expected USING clause, got: %s", ddl)
	}
	if !strings.Contains(ddl, "WITH CHECK ((tenant_id = current_setting('app.tenant_id')::int))") {
		t.Errorf("expected WITH CHECK clause, got: %s", ddl)
	}
}

func TestGenerateDDL_UsingOnly(t *testing.T) {
	g := &policy.DDLGenerator{}
	def := core.PolicyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "read_policy"},
		TableName:    "documents",
		Command:      "SELECT",
		Roles:        []string{"reader"},
		Using:        "(visible = true)",
		WithCheck:    "",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	ddl := stmts[0]
	if !strings.Contains(ddl, "USING") {
		t.Errorf("expected USING clause, got: %s", ddl)
	}
	if strings.Contains(ddl, "WITH CHECK") {
		t.Errorf("expected no WITH CHECK clause, got: %s", ddl)
	}
}

func TestGenerateDDL_WithCheckOnly(t *testing.T) {
	g := &policy.DDLGenerator{}
	def := core.PolicyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "insert_policy"},
		TableName:    "records",
		Command:      "INSERT",
		Roles:        []string{"writer"},
		Using:        "",
		WithCheck:    "(owner_id = current_user_id())",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	ddl := stmts[0]
	if strings.Contains(ddl, "USING") {
		t.Errorf("expected no USING clause, got: %s", ddl)
	}
	if !strings.Contains(ddl, "WITH CHECK") {
		t.Errorf("expected WITH CHECK clause, got: %s", ddl)
	}
}

func TestGenerateDDL_PublicRole(t *testing.T) {
	g := &policy.DDLGenerator{}
	def := core.PolicyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "public_read"},
		TableName:    "announcements",
		Command:      "SELECT",
		Roles:        []string{"PUBLIC"},
		Using:        "(published = true)",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	ddl := stmts[0]
	if !strings.Contains(ddl, "TO PUBLIC") {
		t.Errorf("expected TO PUBLIC, got: %s", ddl)
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &policy.DDLGenerator{}
	def := core.PolicyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "tenant_isolation"},
		TableName:    "orders",
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP POLICY IF EXISTS tenant_isolation ON public.orders"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}

	// Policy DROP must NOT have CASCADE
	if strings.Contains(stmts[0], "CASCADE") {
		t.Error("policy DROP should not include CASCADE")
	}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &policy.DDLGenerator{}
	def := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "seq"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestGenerateDrop_WrongType(t *testing.T) {
	g := &policy.DDLGenerator{}
	def := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "seq"},
	}

	_, err := g.GenerateDrop(def)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}
