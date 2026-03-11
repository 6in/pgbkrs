package trigger

import (
	"fmt"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// DDLGenerator generates DDL for trigger objects.
type DDLGenerator struct{}

// GenerateDDL produces CREATE TRIGGER DDL statements.
// Assembles timing, events, table, and function into standard SQL syntax.
// Defaults to FOR EACH ROW (TriggerDef does not store row vs statement granularity).
func (g *DDLGenerator) GenerateDDL(def core.ObjectDef) ([]string, error) {
	td, ok := def.(core.TriggerDef)
	if !ok {
		return nil, fmt.Errorf("trigger.DDLGenerator.GenerateDDL: expected core.TriggerDef, got %T", def)
	}

	events := strings.Join(td.Events, " OR ")
	stmt := fmt.Sprintf("CREATE TRIGGER %s\n    %s %s ON %s.%s\n    FOR EACH ROW EXECUTE FUNCTION %s.%s()",
		td.Name, td.Timing, events, td.Schema, td.TableName, td.Schema, td.FunctionName)
	return []string{stmt}, nil
}

// GenerateDrop produces DROP TRIGGER DDL statements.
// Trigger names are scoped to their table, so ON table_name is required.
func (g *DDLGenerator) GenerateDrop(def core.ObjectDef) ([]string, error) {
	td, ok := def.(core.TriggerDef)
	if !ok {
		return nil, fmt.Errorf("trigger.DDLGenerator.GenerateDrop: expected core.TriggerDef, got %T", def)
	}

	stmt := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s.%s CASCADE", td.Name, td.Schema, td.TableName)
	return []string{stmt}, nil
}
