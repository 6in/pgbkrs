package trigger_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/trigger"
)

func TestGenerateDDL(t *testing.T) {
	g := &trigger.DDLGenerator{}
	def := &core.TriggerDef{
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
	if len(stmts) == 0 {
		t.Fatal("expected at least one DDL statement")
	}

	combined := strings.Join(stmts, "\n")
	if !strings.Contains(combined, "CREATE TRIGGER") {
		t.Error("expected CREATE TRIGGER statement")
	}
	if !strings.Contains(combined, "BEFORE INSERT OR UPDATE ON") {
		t.Error("expected BEFORE INSERT OR UPDATE ON in trigger DDL")
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &trigger.DDLGenerator{}
	def := &core.TriggerDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTrigger, Schema: "public", Name: "trg_audit"},
		TableName:    "orders",
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) == 0 {
		t.Fatal("expected at least one DDL statement")
	}

	found := false
	for _, s := range stmts {
		if strings.Contains(s, "DROP TRIGGER IF EXISTS") {
			found = true
		}
	}
	if !found {
		t.Error("expected DROP TRIGGER IF EXISTS statement")
	}
}
