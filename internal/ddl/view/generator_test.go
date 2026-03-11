package view_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/view"
)

func TestGenerateDDL(t *testing.T) {
	g := &view.DDLGenerator{}
	def := core.ViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindView, Schema: "public", Name: "active_users"},
		Definition:   "SELECT id, name FROM users WHERE active = true",
		Owner:        "admin",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	stmt := stmts[0]
	// Must contain CREATE VIEW schema.name AS
	if !strings.Contains(stmt, "CREATE VIEW public.active_users AS") {
		t.Errorf("expected CREATE VIEW public.active_users AS, got: %s", stmt)
	}
	// Must contain the definition body
	if !strings.Contains(stmt, "SELECT id, name FROM users WHERE active = true") {
		t.Errorf("expected SELECT body in DDL, got: %s", stmt)
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &view.DDLGenerator{}
	def := core.ViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindView, Schema: "public", Name: "active_users"},
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP VIEW IF EXISTS public.active_users CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &view.DDLGenerator{}
	def := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Fatal("expected error for wrong type input")
	}
}

func TestGenerateDrop_WrongType(t *testing.T) {
	g := &view.DDLGenerator{}
	def := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDrop(def)
	if err == nil {
		t.Fatal("expected error for wrong type input")
	}
}
