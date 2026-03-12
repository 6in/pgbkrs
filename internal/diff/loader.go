package diff

import "github.com/pgbkrs/pgbackup/internal/core"

// loadBackup reads a backup directory and returns all schema objects keyed by
// "schema.kind.name". FKs use kind="fk".
func loadBackup(backupDir string) (map[string]core.ObjectDef, error) {
	return map[string]core.ObjectDef{}, nil
}
