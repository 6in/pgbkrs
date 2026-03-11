package matview_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/matview"
)

func TestGenerateDDL_Populated(t *testing.T) {
	g := &matview.DDLGenerator{}
	def := core.MaterializedViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindMaterializedView, Schema: "public", Name: "mv_stats"},
		Definition:   "SELECT count(*) FROM orders",
		Owner:        "admin",
		IsPopulated:  true,
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	stmt := stmts[0]
	if !strings.Contains(stmt, "CREATE MATERIALIZED VIEW public.mv_stats AS") {
		t.Errorf("expected CREATE MATERIALIZED VIEW public.mv_stats AS, got: %s", stmt)
	}
	if !strings.Contains(stmt, "SELECT count(*) FROM orders") {
		t.Errorf("expected SELECT body in DDL, got: %s", stmt)
	}
	if strings.Contains(stmt, "WITH NO DATA") {
		t.Error("populated matview should not contain WITH NO DATA")
	}
}

func TestGenerateDDL_NotPopulated(t *testing.T) {
	g := &matview.DDLGenerator{}
	def := core.MaterializedViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindMaterializedView, Schema: "public", Name: "mv_stats"},
		Definition:   "SELECT count(*) FROM orders",
		Owner:        "admin",
		IsPopulated:  false,
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	stmt := stmts[0]
	if !strings.Contains(stmt, "CREATE MATERIALIZED VIEW public.mv_stats AS") {
		t.Errorf("expected CREATE MATERIALIZED VIEW public.mv_stats AS, got: %s", stmt)
	}
	if !strings.Contains(stmt, "WITH NO DATA") {
		t.Error("non-populated matview should contain WITH NO DATA")
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &matview.DDLGenerator{}
	def := core.MaterializedViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindMaterializedView, Schema: "public", Name: "mv_stats"},
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP MATERIALIZED VIEW IF EXISTS public.mv_stats CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &matview.DDLGenerator{}
	def := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Fatal("expected error for wrong type input")
	}
}

func TestGenerateDrop_WrongType(t *testing.T) {
	g := &matview.DDLGenerator{}
	def := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDrop(def)
	if err == nil {
		t.Fatal("expected error for wrong type input")
	}
}
