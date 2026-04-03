package function_test

import (
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/function"
)

func TestGenerateDDL_Passthrough(t *testing.T) {
	g := &function.DDLGenerator{}
	definition := "CREATE OR REPLACE FUNCTION public.add_numbers(a integer, b integer)\n RETURNS integer\n LANGUAGE sql\nAS $function$ SELECT a + b $function$"
	def := core.FunctionDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "add_numbers"},
		Definition:   definition,
		ArgTypes:     "a integer, b integer",
		ReturnType:   "integer",
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	// Must be exact passthrough — no modification
	if stmts[0] != definition {
		t.Errorf("expected exact passthrough of Definition\ngot:  %q\nwant: %q", stmts[0], definition)
	}
}

func TestGenerateDrop_WithArgs(t *testing.T) {
	g := &function.DDLGenerator{}
	def := core.FunctionDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "add_numbers"},
		ArgTypes:     "a integer, b integer",
		ReturnType:   "integer",
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP FUNCTION IF EXISTS public.add_numbers(a integer, b integer) CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDrop_NoArgs(t *testing.T) {
	g := &function.DDLGenerator{}
	def := core.FunctionDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "now_utc"},
		ArgTypes:     "",
		ReturnType:   "timestamp with time zone",
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 DDL statement, got %d", len(stmts))
	}

	expected := "DROP FUNCTION IF EXISTS public.now_utc() CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDrop_WithDefaults(t *testing.T) {
	g := &function.DDLGenerator{}
	tests := []struct {
		name     string
		argTypes string
		expected string
	}{
		{
			name:     "single arg with DEFAULT",
			argTypes: "p_table_name text DEFAULT NULL::text",
			expected: "DROP FUNCTION IF EXISTS art_qc.sf_retire_table_after(p_table_name text) CASCADE",
		},
		{
			name:     "multiple args some with DEFAULT",
			argTypes: "a integer, b text DEFAULT 'hello'",
			expected: "DROP FUNCTION IF EXISTS public.myfunc(a integer, b text) CASCADE",
		},
		{
			name:     "complex type with parens and DEFAULT",
			argTypes: "a numeric(10, 2) DEFAULT 0.0, b integer",
			expected: "DROP FUNCTION IF EXISTS public.myfunc(a numeric(10, 2), b integer) CASCADE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parts := splitSchemaName(tc.expected)
			def := core.FunctionDef{
				ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: parts[0], Name: parts[1]},
				ArgTypes:     tc.argTypes,
			}
			stmts, err := g.GenerateDrop(def)
			if err != nil {
				t.Fatal(err)
			}
			if len(stmts) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(stmts))
			}
			if stmts[0] != tc.expected {
				t.Errorf("expected %q\ngot     %q", tc.expected, stmts[0])
			}
		})
	}
}

// splitSchemaName extracts schema and function name from a DROP FUNCTION statement.
// Only used in tests to avoid duplicating schema/name in test cases.
func splitSchemaName(dropStmt string) [2]string {
	// "DROP FUNCTION IF EXISTS schema.name(...) CASCADE"
	after := dropStmt[len("DROP FUNCTION IF EXISTS "):]
	dot := 0
	for i, c := range after {
		if c == '.' {
			dot = i
			break
		}
	}
	schema := after[:dot]
	rest := after[dot+1:]
	paren := 0
	for i, c := range rest {
		if c == '(' {
			paren = i
			break
		}
	}
	return [2]string{schema, rest[:paren]}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &function.DDLGenerator{}
	def := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Fatal("expected error for wrong type input")
	}
}

func TestGenerateDrop_WrongType(t *testing.T) {
	g := &function.DDLGenerator{}
	def := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	_, err := g.GenerateDrop(def)
	if err == nil {
		t.Fatal("expected error for wrong type input")
	}
}
