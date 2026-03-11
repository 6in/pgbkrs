package trigger

import (
	"fmt"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for trigger objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE TRIGGER DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateDrop produces DROP TRIGGER DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}
