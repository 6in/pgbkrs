package matview

import (
	"fmt"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for materialized view objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE MATERIALIZED VIEW DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	mvd, ok := def.(core.MaterializedViewDef)
	if !ok {
		return nil, fmt.Errorf("matview.DDLGenerator.GenerateDDL: expected core.MaterializedViewDef, got %T", def)
	}

	stmt := fmt.Sprintf("CREATE MATERIALIZED VIEW %s.%s AS\n%s", mvd.Schema, mvd.Name, mvd.Definition)
	if !mvd.IsPopulated {
		stmt += "\nWITH NO DATA"
	}
	return []string{stmt}, nil
}

// GenerateDrop produces DROP MATERIALIZED VIEW DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	mvd, ok := def.(core.MaterializedViewDef)
	if !ok {
		return nil, fmt.Errorf("matview.DDLGenerator.GenerateDrop: expected core.MaterializedViewDef, got %T", def)
	}

	stmt := fmt.Sprintf("DROP MATERIALIZED VIEW IF EXISTS %s.%s CASCADE", mvd.Schema, mvd.Name)
	return []string{stmt}, nil
}
