package function_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/function"
)

func TestGenerateDDL(t *testing.T) {
	g := &function.DDLGenerator{}
	def := &core.FunctionDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "add_numbers"},
		Definition:   "CREATE OR REPLACE FUNCTION public.add_numbers(a integer, b integer) RETURNS integer LANGUAGE sql AS $$ SELECT a + b $$",
		ArgTypes:     "a integer, b integer",
		ReturnType:   "integer",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) == 0 {
		t.Fatal("expected at least one DDL statement")
	}

	// Function DDL should pass through the Definition field
	combined := strings.Join(stmts, "\n")
	if !strings.Contains(combined, "CREATE OR REPLACE FUNCTION") {
		t.Error("expected CREATE OR REPLACE FUNCTION in output")
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &function.DDLGenerator{}
	def := &core.FunctionDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "add_numbers"},
		ArgTypes:     "a integer, b integer",
		ReturnType:   "integer",
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
		if strings.Contains(s, "DROP FUNCTION IF EXISTS") {
			found = true
		}
	}
	if !found {
		t.Error("expected DROP FUNCTION IF EXISTS statement")
	}
}
