package compositetype_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/compositetype"
)

func TestGenerateDDL(t *testing.T) {
	g := &compositetype.DDLGenerator{}
	def := &core.TypeDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "public", Name: "address"},
		Fields: []core.CompositeField{
			{Name: "street", Type: "text"},
			{Name: "city", Type: "text"},
		},
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
	if !strings.Contains(combined, "AS (") || (!strings.Contains(combined, "street text") && !strings.Contains(combined, "street  text")) {
		t.Error("expected AS (field1 type1, field2 type2) in output")
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &compositetype.DDLGenerator{}
	def := &core.TypeDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "public", Name: "address"},
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
