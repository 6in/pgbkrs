package enum_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/enum"
)

func TestGenerateDDL(t *testing.T) {
	g := &enum.DDLGenerator{}
	def := &core.EnumDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status"},
		Labels:       []string{"a", "b", "c"},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) == 0 {
		t.Fatal("expected at least one DDL statement")
	}

	combined := strings.Join(stmts, "\n")
	if !strings.Contains(combined, "CREATE TYPE") {
		t.Error("expected CREATE TYPE statement")
	}
	if !strings.Contains(combined, "AS ENUM") {
		t.Error("expected AS ENUM in output")
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &enum.DDLGenerator{}
	def := &core.EnumDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status"},
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
		if strings.Contains(s, "DROP TYPE IF EXISTS") {
			found = true
		}
	}
	if !found {
		t.Error("expected DROP TYPE IF EXISTS statement")
	}
}
