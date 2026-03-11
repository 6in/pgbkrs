package domain

import (
	"fmt"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for domain objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE DOMAIN DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	dd, ok := def.(core.DomainDef)
	if !ok {
		return nil, fmt.Errorf("domain.DDLGenerator: expected core.DomainDef, got %T", def)
	}

	ddl := fmt.Sprintf("CREATE DOMAIN %s.%s AS %s", dd.Schema, dd.Name, dd.BaseType)

	if !dd.Nullable {
		ddl += " NOT NULL"
	}
	if dd.Default != "" {
		ddl += fmt.Sprintf(" DEFAULT %s", dd.Default)
	}
	if dd.CheckName != "" {
		ddl += fmt.Sprintf(" CONSTRAINT %s %s", dd.CheckName, dd.CheckDefinition)
	}

	return []string{ddl}, nil
}

// GenerateDrop produces DROP DOMAIN DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	dd, ok := def.(core.DomainDef)
	if !ok {
		return nil, fmt.Errorf("domain.DDLGenerator: expected core.DomainDef, got %T", def)
	}

	return []string{fmt.Sprintf("DROP DOMAIN IF EXISTS %s.%s CASCADE", dd.Schema, dd.Name)}, nil
}
