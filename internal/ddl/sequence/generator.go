package sequence

import (
	"fmt"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for sequence objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE SEQUENCE DDL statements.
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	sd, ok := def.(core.SequenceDef)
	if !ok {
		return nil, fmt.Errorf("sequence.DDLGenerator: expected core.SequenceDef, got %T", def)
	}

	cycle := "NO CYCLE"
	if sd.Cycle {
		cycle = "CYCLE"
	}

	ddl := fmt.Sprintf("CREATE SEQUENCE %s.%s START %d MINVALUE %d MAXVALUE %d INCREMENT BY %d %s CACHE %d",
		sd.Schema, sd.Name, sd.StartValue, sd.MinValue, sd.MaxValue, sd.IncrementBy, cycle, sd.Cache)

	return []string{ddl}, nil
}

// GenerateDrop produces DROP SEQUENCE DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	sd, ok := def.(core.SequenceDef)
	if !ok {
		return nil, fmt.Errorf("sequence.DDLGenerator: expected core.SequenceDef, got %T", def)
	}

	return []string{fmt.Sprintf("DROP SEQUENCE IF EXISTS %s.%s CASCADE", sd.Schema, sd.Name)}, nil
}
