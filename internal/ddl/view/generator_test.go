package view_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/view"
)

func TestGenerateDDL(t *testing.T) {
	g := &view.DDLGenerator{}
	def := &core.ViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindView, Schema: "public", Name: "active_users"},
		Definition:   "SELECT id, name FROM users",
		Owner:        "admin",
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
		if strings.Contains(s, "CREATE VIEW") && strings.Contains(s, "AS") {
			found = true
		}
	}
	if !found {
		t.Error("expected CREATE VIEW ... AS statement")
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &view.DDLGenerator{}
	def := &core.ViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindView, Schema: "public", Name: "active_users"},
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
		if strings.Contains(s, "DROP VIEW IF EXISTS") {
			found = true
		}
	}
	if !found {
		t.Error("expected DROP VIEW IF EXISTS statement")
	}
}
