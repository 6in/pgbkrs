package domain_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/domain"
)

func TestGenerateDDL(t *testing.T) {
	g := &domain.DDLGenerator{}
	def := &core.DomainDef{
		ObjectHeader:    core.ObjectHeader{Kind: core.KindDomain, Schema: "public", Name: "positive_int"},
		BaseType:        "integer",
		Nullable:        false,
		Default:         "0",
		CheckName:       "positive_int_check",
		CheckDefinition: "CHECK (VALUE > 0)",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) == 0 {
		t.Fatal("expected at least one DDL statement")
	}

	combined := strings.Join(stmts, "\n")
	if !strings.Contains(combined, "CREATE DOMAIN") {
		t.Error("expected CREATE DOMAIN statement")
	}
	if !strings.Contains(combined, "CONSTRAINT") {
		t.Error("expected CONSTRAINT in domain DDL")
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &domain.DDLGenerator{}
	def := &core.DomainDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindDomain, Schema: "public", Name: "positive_int"},
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
		if strings.Contains(s, "DROP DOMAIN IF EXISTS") {
			found = true
		}
	}
	if !found {
		t.Error("expected DROP DOMAIN IF EXISTS statement")
	}
}
