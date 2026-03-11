package backup

import (
	"fmt"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// skipTypes lists column type substrings that cannot be safely exported to CSV.
// Tables containing any of these column types are skipped during backup.
var skipTypes = []string{"bytea", "xml", "pg_lsn", "txid_snapshot"}

// skipReason returns a non-empty reason string if the table should be skipped
// due to containing an unsupported column type. Returns "" if the table is safe
// to export. Detection uses strings.Contains to catch array variants like "bytea[]".
func skipReason(td *core.TableDef) string {
	for _, col := range td.Columns {
		for _, bad := range skipTypes {
			if strings.Contains(col.Type, bad) {
				return fmt.Sprintf("contains %s column: %s", bad, col.Name)
			}
		}
	}
	return ""
}
