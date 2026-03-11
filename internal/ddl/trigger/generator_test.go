package trigger_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/trigger"
)

func TestGenerateDDL_MultipleEvents(t *testing.T) {
	g := &trigger.DDLGenerator{}
	def := core.TriggerDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTrigger, Schema: "public", Name: "trg_audit"},
		Timing:       "BEFORE",
		Events:       []string{"INSERT", "UPDATE"},
		TableName:    "orders",
		FunctionName: "audit_fn",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	stmt := stmts[0]
	if !strings.Contains(stmt, "CREATE TRIGGER trg_audit") {
		t.Errorf("expected CREATE TRIGGER trg_audit, got: %s", stmt)
	}
	if !strings.Contains(stmt, "BEFORE INSERT OR UPDATE ON public.orders") {
		t.Errorf("expected BEFORE INSERT OR UPDATE ON public.orders, got: %s", stmt)
	}
	if !strings.Contains(stmt, "FOR EACH ROW EXECUTE FUNCTION public.audit_fn()") {
		t.Errorf("expected FOR EACH ROW EXECUTE FUNCTION public.audit_fn(), got: %s", stmt)
	}
}

func TestGenerateDDL_SingleEvent(t *testing.T) {
	g := &trigger.DDLGenerator{}
	def := core.TriggerDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTrigger, Schema: "public", Name: "trg_delete"},
		Timing:       "AFTER",
		Events:       []string{"DELETE"},
		TableName:    "orders",
		FunctionName: "cleanup_fn",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	stmt := stmts[0]
	if !strings.Contains(stmt, "AFTER DELETE ON public.orders") {
		t.Errorf("expected AFTER DELETE ON public.orders (no OR), got: %s", stmt)
	}
	// Must NOT contain " OR " for single event
	if strings.Contains(stmt, " OR ") {
		t.Errorf("single event should not contain OR, got: %s", stmt)
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &trigger.DDLGenerator{}
	def := core.TriggerDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTrigger, Schema: "public", Name: "trg_audit"},
		TableName:    "orders",
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP TRIGGER IF EXISTS trg_audit ON public.orders CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &trigger.DDLGenerator{}
	def := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Fatal("expected error for wrong type input")
	}
}

func TestGenerateDrop_WrongType(t *testing.T) {
	g := &trigger.DDLGenerator{}
	def := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDrop(def)
	if err == nil {
		t.Fatal("expected error for wrong type input")
	}
}
