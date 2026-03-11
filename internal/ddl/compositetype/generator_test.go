package compositetype_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/compositetype"
)

func TestGenerateDDL_MultipleFields(t *testing.T) {
	g := &compositetype.DDLGenerator{}
	def := core.TypeDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "public", Name: "address"},
		Fields: []core.CompositeField{
			{Name: "street", Type: "text"},
			{Name: "city", Type: "text"},
			{Name: "zip", Type: "character varying(10)"},
		},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	ddl := stmts[0]
	if !strings.HasPrefix(ddl, "CREATE TYPE public.address AS (") {
		t.Errorf("expected CREATE TYPE public.address AS ( prefix, got: %s", ddl)
	}
	if !strings.Contains(ddl, "street text") {
		t.Errorf("expected 'street text' in DDL, got: %s", ddl)
	}
	if !strings.Contains(ddl, "city text") {
		t.Errorf("expected 'city text' in DDL, got: %s", ddl)
	}
	if !strings.Contains(ddl, "zip character varying(10)") {
		t.Errorf("expected 'zip character varying(10)' in DDL, got: %s", ddl)
	}
}

func TestGenerateDDL_SingleField(t *testing.T) {
	g := &compositetype.DDLGenerator{}
	def := core.TypeDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "myschema", Name: "singleton"},
		Fields: []core.CompositeField{
			{Name: "value", Type: "integer"},
		},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	ddl := stmts[0]
	// Single field should not have trailing comma
	if strings.Contains(ddl, "integer,") {
		t.Errorf("single field should not have trailing comma, got: %s", ddl)
	}
	if !strings.Contains(ddl, "value integer") {
		t.Errorf("expected 'value integer' in DDL, got: %s", ddl)
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &compositetype.DDLGenerator{}
	def := core.TypeDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "public", Name: "address"},
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP TYPE IF EXISTS public.address CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &compositetype.DDLGenerator{}
	def := core.EnumDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}
