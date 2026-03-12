package foreignkey

import (
	"fmt"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.DDLGenerator = (*DDLGenerator)(nil)

// DDLGenerator generates DDL for foreign key constraints.
type DDLGenerator struct{}

// GenerateDDL produces ALTER TABLE ... ADD CONSTRAINT DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	fk, ok := def.(*core.ForeignKeyDef)
	if !ok {
		return nil, fmt.Errorf("foreignkey.DDLGenerator.GenerateDDL: expected *core.ForeignKeyDef, got %T", def)
	}

	stmt := fmt.Sprintf("ALTER TABLE %s.%s ADD CONSTRAINT %s %s",
		fk.Schema, fk.SourceTable, fk.Name, fk.Definition)

	return []string{stmt}, nil
}

// GenerateDrop produces ALTER TABLE ... DROP CONSTRAINT DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	fk, ok := def.(*core.ForeignKeyDef)
	if !ok {
		return nil, fmt.Errorf("foreignkey.DDLGenerator.GenerateDrop: expected *core.ForeignKeyDef, got %T", def)
	}

	stmt := fmt.Sprintf("ALTER TABLE IF EXISTS %s.%s DROP CONSTRAINT IF EXISTS %s CASCADE",
		fk.Schema, fk.SourceTable, fk.Name)

	return []string{stmt}, nil
}
