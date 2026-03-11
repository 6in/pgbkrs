package foreignkey_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/foreignkey"
)

func TestGenerateDDL(t *testing.T) {
	g := &foreignkey.DDLGenerator{}
	def := &core.ForeignKeyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "fk_orders_customer"},
		SourceTable:  "orders",
		TargetSchema: "public",
		TargetTable:  "customers",
		Definition:   "FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}

	expected := "ALTER TABLE public.orders ADD CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, stmts[0])
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &foreignkey.DDLGenerator{}
	def := &core.ForeignKeyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "fk_orders_customer"},
		SourceTable:  "orders",
		TargetSchema: "public",
		TargetTable:  "customers",
		Definition:   "FOREIGN KEY (customer_id) REFERENCES public.customers(id)",
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}

	expected := "ALTER TABLE public.orders DROP CONSTRAINT IF EXISTS fk_orders_customer CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, stmts[0])
	}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &foreignkey.DDLGenerator{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
	if !strings.Contains(err.Error(), "ForeignKeyDef") {
		t.Errorf("expected error mentioning ForeignKeyDef, got: %s", err)
	}
}

func TestGenerateDrop_WrongType(t *testing.T) {
	g := &foreignkey.DDLGenerator{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDrop(def)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
	if !strings.Contains(err.Error(), "ForeignKeyDef") {
		t.Errorf("expected error mentioning ForeignKeyDef, got: %s", err)
	}
}
