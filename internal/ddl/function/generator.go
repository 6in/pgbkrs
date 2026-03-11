package function

import (
	"fmt"

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

	stmt := fmt.Sprintf("DROP FUNCTION IF EXISTS %s.%s(%s) CASCADE", fd.Schema, fd.Name, fd.ArgTypes)
	return []string{stmt}, nil
}
