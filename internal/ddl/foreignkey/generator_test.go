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
		Definition:   "FOREIGN KEY (customer_id) REFERENCES customers(id)",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) == 0 {
		t.Fatal("expected at least one DDL statement")
	}

	found := false
	for _, s := range stmts {
		if strings.Contains(s, "ALTER TABLE") && strings.Contains(s, "ADD CONSTRAINT") {
			found = true
		}
	}
	if !found {
		t.Error("expected ALTER TABLE ... ADD CONSTRAINT statement")
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &foreignkey.DDLGenerator{}
	def := &core.ForeignKeyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "fk_orders_customer"},
		SourceTable:  "orders",
		TargetSchema: "public",
		TargetTable:  "customers",
		Definition:   "FOREIGN KEY (customer_id) REFERENCES customers(id)",
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
		if strings.Contains(s, "ALTER TABLE") && strings.Contains(s, "DROP CONSTRAINT") {
			found = true
		}
	}
	if !found {
		t.Error("expected ALTER TABLE ... DROP CONSTRAINT statement")
	}
}
