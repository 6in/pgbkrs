package datadiff

import (
	"fmt"
	"io"
)

// Run compares table data CSV files in two backup directories and writes a summary to w.
// Only tables present in both backups are compared; tables added or removed are left to
// schema diff (pgbackup diff) to report.
func Run(backupA, backupB string, w io.Writer) error {
	aTables, err := loadBackupTables(backupA)
	if err != nil {
		return fmt.Errorf("load backup A: %w", err)
	}
	bTables, err := loadBackupTables(backupB)
	if err != nil {
		return fmt.Errorf("load backup B: %w", err)
	}

	var results []TableDiffResult
	for key, aEntry := range aTables {
		bEntry, ok := bTables[key]
		if !ok {
			continue // table not in B; schema diff will report it
		}
		results = append(results, compareTables(aEntry, bEntry))
	}

	formatDataReport(results, backupA, backupB, w)
	return nil
}
