package enum

import (
	"fmt"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for enum type objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE TYPE ... AS ENUM DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	ed, ok := def.(core.EnumDef)
	if !ok {
		return nil, fmt.Errorf("enum.DDLGenerator: expected core.EnumDef, got %T", def)
	}

	quoted := make([]string, len(ed.Labels))
	for i, l := range ed.Labels {
		quoted[i] = fmt.Sprintf("'%s'", l)
	}

	ddl := fmt.Sprintf("CREATE TYPE %s.%s AS ENUM (%s)", ed.Schema, ed.Name, strings.Join(quoted, ", "))

	return []string{ddl}, nil
}

// GenerateDrop produces DROP TYPE DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	ed, ok := def.(core.EnumDef)
	if !ok {
		return nil, fmt.Errorf("enum.DDLGenerator: expected core.EnumDef, got %T", def)
	}

	return []string{fmt.Sprintf("DROP TYPE IF EXISTS %s.%s CASCADE", ed.Schema, ed.Name)}, nil
}
