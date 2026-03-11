package view

import (
	"fmt"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for view objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE VIEW DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	vd, ok := def.(core.ViewDef)
	if !ok {
		return nil, fmt.Errorf("view.DDLGenerator.GenerateDDL: expected core.ViewDef, got %T", def)
	}

	stmt := fmt.Sprintf("CREATE VIEW %s.%s AS\n%s", vd.Schema, vd.Name, vd.Definition)
	return []string{stmt}, nil
}

// GenerateDrop produces DROP VIEW DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	vd, ok := def.(core.ViewDef)
	if !ok {
		return nil, fmt.Errorf("view.DDLGenerator.GenerateDrop: expected core.ViewDef, got %T", def)
	}

	stmt := fmt.Sprintf("DROP VIEW IF EXISTS %s.%s CASCADE", vd.Schema, vd.Name)
	return []string{stmt}, nil
}
