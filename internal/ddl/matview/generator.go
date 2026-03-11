package matview

import (
	"fmt"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for materialized view objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE MATERIALIZED VIEW DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateDrop produces DROP MATERIALIZED VIEW DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}
