package export

import "github.com/pgbkrs/pgbackup/internal/core"

// DataMeta holds metadata about exported table data (CSV file reference in def.yaml).
type DataMeta struct {
	File     string
	Columns  []string
	RowCount int64
	Checksum string
}

// ShouldExportData returns true if the table should have its data exported.
// Partitioned parent tables have no data of their own; only leaf partitions hold rows.
func ShouldExportData(td *core.TableDef) bool {
	return td.Partitioning == nil
}
