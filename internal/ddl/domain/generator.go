package domain

import (
	"fmt"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for domain objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE DOMAIN DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateDrop produces DROP DOMAIN DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}
