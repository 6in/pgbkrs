package function_test

import (
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/function"
)

func TestGenerateDDL_Passthrough(t *testing.T) {
	g := &function.DDLGenerator{}
	definition := "CREATE OR REPLACE FUNCTION public.add_numbers(a integer, b integer)\n RETURNS integer\n LANGUAGE sql\nAS $function$ SELECT a + b $function$"
	def := core.FunctionDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "add_numbers"},
		Definition:   definition,
		ArgTypes:     "a integer, b integer",
		ReturnType:   "integer",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	// Must be exact passthrough — no modification
	if stmts[0] != definition {
		t.Errorf("expected exact passthrough of Definition\ngot:  %q\nwant: %q", stmts[0], definition)
	}
}

func TestGenerateDrop_WithArgs(t *testing.T) {
	g := &function.DDLGenerator{}
	def := core.FunctionDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "add_numbers"},
		ArgTypes:     "a integer, b integer",
		ReturnType:   "integer",
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP FUNCTION IF EXISTS public.add_numbers(a integer, b integer) CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDrop_NoArgs(t *testing.T) {
	g := &function.DDLGenerator{}
	def := core.FunctionDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "now_utc"},
		ArgTypes:     "",
		ReturnType:   "timestamp with time zone",
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP FUNCTION IF EXISTS public.now_utc() CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &function.DDLGenerator{}
	def := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Fatal("expected error for wrong type input")
	}
}

func TestGenerateDrop_WrongType(t *testing.T) {
	g := &function.DDLGenerator{}
	def := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDrop(def)
	if err == nil {
		t.Fatal("expected error for wrong type input")
	}
}
