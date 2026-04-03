package function

import (
	"fmt"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for function objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE OR REPLACE FUNCTION DDL statements.
// The Definition field from pg_get_functiondef() is already a complete statement,
// so this is a pure passthrough.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	fd, ok := def.(core.FunctionDef)
	if !ok {
		return nil, fmt.Errorf("function.DDLGenerator.GenerateDDL: expected core.FunctionDef, got %T", def)
	}

	return []string{fd.Definition}, nil
}

// GenerateDrop produces DROP FUNCTION DDL statements.
// Includes argument signature for overload disambiguation.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	fd, ok := def.(core.FunctionDef)
	if !ok {
		return nil, fmt.Errorf("function.DDLGenerator.GenerateDrop: expected core.FunctionDef, got %T", def)
	}

	stmt := fmt.Sprintf("DROP FUNCTION IF EXISTS %s.%s(%s) CASCADE", fd.Schema, fd.Name, stripDefaults(fd.ArgTypes))
	return []string{stmt}, nil
}

// stripDefaults removes DEFAULT clauses from a pg_get_function_arguments() string.
// DROP FUNCTION does not accept DEFAULT values in the argument list.
// e.g. "a integer DEFAULT 5, b text DEFAULT 'hello'" → "a integer, b text"
// Splits at top-level commas only to handle types like numeric(10, 2).
func stripDefaults(argTypes string) string {
	if argTypes == "" {
		return ""
	}
	args := splitTopLevelArgs(argTypes)
	for i, arg := range args {
		if idx := strings.Index(strings.ToUpper(arg), " DEFAULT "); idx >= 0 {
			args[i] = strings.TrimRight(arg[:idx], " ")
		}
	}
	return strings.Join(args, ", ")
}

// splitTopLevelArgs splits a comma-separated argument list at top-level commas only
// (i.e. not inside parentheses), then trims whitespace from each element.
func splitTopLevelArgs(s string) []string {
	var args []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	args = append(args, strings.TrimSpace(s[start:]))
	return args
}
