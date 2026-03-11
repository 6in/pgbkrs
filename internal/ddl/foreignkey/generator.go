package foreignkey

import (
	"fmt"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for foreign key constraints.
type DDLGenerator struct{}

// GenerateDDL produces ALTER TABLE ADD CONSTRAINT DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateDrop produces ALTER TABLE DROP CONSTRAINT DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}
