package table

import (
	"fmt"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.DDLGenerator = (*DDLGenerator)(nil)

// DDLGenerator generates DDL for table objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE TABLE DDL statements.
// Returns a slice: the CREATE TABLE statement followed by any CREATE INDEX statements.
// FK constraints are deliberately excluded (they are separate objects).
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	td, ok := def.(*core.TableDef)
	if !ok {
		return nil, fmt.Errorf("table.DDLGenerator.GenerateDDL: expected *core.TableDef, got %T", def)
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("CREATE TABLE %s.%s (\n", td.Schema, td.Name))

	// Collect all body items (columns + constraints) for comma separation.
	var items []string

	// Columns
	for _, col := range td.Columns {
		var cb strings.Builder
		cb.WriteString(fmt.Sprintf("    %s %s", col.Name, col.Type))
		if !col.Nullable {
			cb.WriteString(" NOT NULL")
		}
		if col.Default != "" {
			cb.WriteString(fmt.Sprintf(" DEFAULT %s", col.Default))
		}
		items = append(items, cb.String())
	}

	// Primary key constraint
	if td.Constraints.PrimaryKey != nil {
		pk := td.Constraints.PrimaryKey
		items = append(items, fmt.Sprintf("    CONSTRAINT %s PRIMARY KEY (%s)",
			pk.Name, strings.Join(pk.Columns, ", ")))
	}

	// Unique constraints
	for _, uc := range td.Constraints.Unique {
		items = append(items, fmt.Sprintf("    CONSTRAINT %s %s", uc.Name, uc.Definition))
	}

	// Check constraints
	for _, cc := range td.Constraints.Check {
		items = append(items, fmt.Sprintf("    CONSTRAINT %s %s", cc.Name, cc.Definition))
	}

	// Write items with commas (last item has no trailing comma)
	for i, item := range items {
		b.WriteString(item)
		if i < len(items)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}

	b.WriteString(")")

	// Partition clause — KeyExpression comes from pg_get_partkeydef() which already
	// includes the strategy keyword (e.g. "RANGE (col)"), so use it directly.
	if td.Partitioning != nil {
		b.WriteString(fmt.Sprintf(" PARTITION BY %s", td.Partitioning.KeyExpression))
	}

	stmts := []string{b.String()}

	// Indexes as separate statements (pass-through from pg_get_indexdef)
	for _, idx := range td.Indexes {
		stmts = append(stmts, idx.Definition)
	}

	return stmts, nil
}

// GenerateDrop produces DROP TABLE DDL statements.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	td, ok := def.(*core.TableDef)
	if !ok {
		return nil, fmt.Errorf("table.DDLGenerator.GenerateDrop: expected *core.TableDef, got %T", def)
	}

	return []string{fmt.Sprintf("DROP TABLE IF EXISTS %s.%s CASCADE", td.Schema, td.Name)}, nil
}
