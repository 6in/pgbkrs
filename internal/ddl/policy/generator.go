package policy

import (
	"fmt"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for row-level security policy objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE POLICY DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	pd, ok := def.(core.PolicyDef)
	if !ok {
		return nil, fmt.Errorf("policy.DDLGenerator: expected core.PolicyDef, got %T", def)
	}

	ddl := fmt.Sprintf("CREATE POLICY %s ON %s.%s", pd.Name, pd.Schema, pd.TableName)
	ddl += fmt.Sprintf(" FOR %s", pd.Command)
	ddl += fmt.Sprintf(" TO %s", strings.Join(pd.Roles, ", "))

	if pd.Using != "" {
		ddl += fmt.Sprintf("\n    USING (%s)", pd.Using)
	}
	if pd.WithCheck != "" {
		ddl += fmt.Sprintf("\n    WITH CHECK (%s)", pd.WithCheck)
	}

	return []string{ddl}, nil
}

// GenerateDrop produces DROP POLICY DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	pd, ok := def.(core.PolicyDef)
	if !ok {
		return nil, fmt.Errorf("policy.DDLGenerator: expected core.PolicyDef, got %T", def)
	}

	return []string{fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s.%s", pd.Name, pd.Schema, pd.TableName)}, nil
}
