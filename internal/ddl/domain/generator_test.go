package domain_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/domain"
)

func TestGenerateDDL_FullDomain(t *testing.T) {
	g := &domain.DDLGenerator{}
	def := core.DomainDef{
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
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	ddl := stmts[0]
	if !strings.HasPrefix(ddl, "CREATE DOMAIN public.positive_int AS integer") {
		t.Errorf("expected CREATE DOMAIN public.positive_int AS integer prefix, got: %s", ddl)
	}
	if !strings.Contains(ddl, "NOT NULL") {
		t.Errorf("expected NOT NULL, got: %s", ddl)
	}
	if !strings.Contains(ddl, "DEFAULT 0") {
		t.Errorf("expected DEFAULT 0, got: %s", ddl)
	}
	if !strings.Contains(ddl, "CONSTRAINT positive_int_check CHECK (VALUE > 0)") {
		t.Errorf("expected CONSTRAINT positive_int_check CHECK (VALUE > 0), got: %s", ddl)
	}
}

func TestGenerateDDL_NullableDomain(t *testing.T) {
	g := &domain.DDLGenerator{}
	def := core.DomainDef{
		ObjectHeader:    core.ObjectHeader{Kind: core.KindDomain, Schema: "public", Name: "nullable_text"},
		BaseType:        "text",
		Nullable:        true,
		Default:         "",
		CheckName:       "",
		CheckDefinition: "",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	ddl := stmts[0]
	if strings.Contains(ddl, "NOT NULL") {
		t.Errorf("nullable domain should not have NOT NULL, got: %s", ddl)
	}
}

func TestGenerateDDL_NoCheckConstraint(t *testing.T) {
	g := &domain.DDLGenerator{}
	def := core.DomainDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindDomain, Schema: "public", Name: "nonnull_text"},
		BaseType:     "text",
		Nullable:     false,
		Default:      "",
		CheckName:    "",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	ddl := stmts[0]
	if strings.Contains(ddl, "CONSTRAINT") {
		t.Errorf("domain without check should not have CONSTRAINT, got: %s", ddl)
	}
}

func TestGenerateDDL_NoDefault(t *testing.T) {
	g := &domain.DDLGenerator{}
	def := core.DomainDef{
		ObjectHeader:    core.ObjectHeader{Kind: core.KindDomain, Schema: "public", Name: "checked_int"},
		BaseType:        "integer",
		Nullable:        false,
		Default:         "",
		CheckName:       "checked_int_check",
		CheckDefinition: "CHECK (VALUE >= 0)",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	ddl := stmts[0]
	if strings.Contains(ddl, "DEFAULT") {
		t.Errorf("domain without default should not have DEFAULT, got: %s", ddl)
	}
	if !strings.Contains(ddl, "CONSTRAINT checked_int_check") {
		t.Errorf("expected CONSTRAINT checked_int_check, got: %s", ddl)
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &domain.DDLGenerator{}
	def := core.DomainDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindDomain, Schema: "public", Name: "positive_int"},
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP DOMAIN IF EXISTS public.positive_int CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &domain.DDLGenerator{}
	def := core.EnumDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}
