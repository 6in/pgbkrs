package sequence_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/sequence"
)

func TestGenerateDDL_AllAttributes(t *testing.T) {
	g := &sequence.DDLGenerator{}
	def := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "orders_id_seq"},
		StartValue:   1,
		MinValue:     1,
		MaxValue:     9223372036854775807,
		IncrementBy:  1,
		Cycle:        false,
		Cache:        1,
		LastValue:    42,
		IsCalled:     true,
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	ddl := stmts[0]
	if !strings.HasPrefix(ddl, "CREATE SEQUENCE public.orders_id_seq") {
		t.Errorf("expected CREATE SEQUENCE public.orders_id_seq prefix, got: %s", ddl)
	}
	for _, kw := range []string{"START 1", "MINVALUE 1", "MAXVALUE 9223372036854775807", "INCREMENT BY 1", "NO CYCLE", "CACHE 1"} {
		if !strings.Contains(ddl, kw) {
			t.Errorf("expected %q in DDL, got: %s", kw, ddl)
		}
	}
}

func TestGenerateDDL_CycleTrue(t *testing.T) {
	g := &sequence.DDLGenerator{}
	def := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "myschema", Name: "cyclic_seq"},
		StartValue:   100,
		MinValue:     1,
		MaxValue:     1000,
		IncrementBy:  10,
		Cycle:        true,
		Cache:        5,
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	ddl := stmts[0]
	if !strings.Contains(ddl, "CYCLE") {
		t.Errorf("expected CYCLE keyword, got: %s", ddl)
	}
	if strings.Contains(ddl, "NO CYCLE") {
		t.Errorf("expected CYCLE (not NO CYCLE), got: %s", ddl)
	}
	if !strings.Contains(ddl, "START 100") {
		t.Errorf("expected START 100, got: %s", ddl)
	}
	if !strings.Contains(ddl, "INCREMENT BY 10") {
		t.Errorf("expected INCREMENT BY 10, got: %s", ddl)
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &sequence.DDLGenerator{}
	def := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "orders_id_seq"},
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP SEQUENCE IF EXISTS public.orders_id_seq CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &sequence.DDLGenerator{}
	def := core.EnumDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}
