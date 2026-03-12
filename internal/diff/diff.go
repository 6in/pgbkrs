package diff

import (
	"fmt"
	"io"
)

// Run compares two backup directories and writes a schema diff report to w.
func Run(backupA, backupB string, w io.Writer) error {
	a, err := loadBackup(backupA)
	if err != nil {
		return fmt.Errorf("load backup A: %w", err)
	}
	b, err := loadBackup(backupB)
	if err != nil {
		return fmt.Errorf("load backup B: %w", err)
	}
	result := compareObjects(a, b)
	formatReport(result, backupA, backupB, w)
	return nil
}
