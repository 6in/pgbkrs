package compositetype

import (
	"fmt"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for composite type objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE TYPE ... AS (...) DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	td, ok := def.(core.TypeDef)
	if !ok {
		return nil, fmt.Errorf("compositetype.DDLGenerator: expected core.TypeDef, got %T", def)
	}

	fields := make([]string, len(td.Fields))
	for i, f := range td.Fields {
		quotedName := `"` + strings.ReplaceAll(f.Name, `"`, `""`) + `"`
		fields[i] = fmt.Sprintf("    %s %s", quotedName, f.Type)
	}

	ddl := fmt.Sprintf("CREATE TYPE %s.%s AS (\n%s\n)", td.Schema, td.Name, strings.Join(fields, ",\n"))

	return []string{ddl}, nil
}

// GenerateDrop produces DROP TYPE DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	td, ok := def.(core.TypeDef)
	if !ok {
		return nil, fmt.Errorf("compositetype.DDLGenerator: expected core.TypeDef, got %T", def)
	}

	return []string{fmt.Sprintf("DROP TYPE IF EXISTS %s.%s CASCADE", td.Schema, td.Name)}, nil
}
