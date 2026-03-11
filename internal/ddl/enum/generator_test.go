package enum_test

import (
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/enum"
)

func TestGenerateDDL_MultipleLabels(t *testing.T) {
	g := &enum.DDLGenerator{}
	def := core.EnumDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status"},
		Labels:       []string{"active", "inactive", "deleted"},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	ddl := stmts[0]
	expected := "CREATE TYPE public.status AS ENUM ('active', 'inactive', 'deleted')"
	if ddl != expected {
		t.Errorf("expected:\n  %s\ngot:\n  %s", expected, ddl)
	}
}

func TestGenerateDDL_SingleLabel(t *testing.T) {
	g := &enum.DDLGenerator{}
	def := core.EnumDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "myschema", Name: "single_enum"},
		Labels:       []string{"only"},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	ddl := stmts[0]
	expected := "CREATE TYPE myschema.single_enum AS ENUM ('only')"
	if ddl != expected {
		t.Errorf("expected:\n  %s\ngot:\n  %s", expected, ddl)
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &enum.DDLGenerator{}
	def := core.EnumDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status"},
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP TYPE IF EXISTS public.status CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &enum.DDLGenerator{}
	def := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "seq"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestGenerateDrop_WrongType(t *testing.T) {
	g := &enum.DDLGenerator{}
	def := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "seq"},
	}

	_, err := g.GenerateDrop(def)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}
