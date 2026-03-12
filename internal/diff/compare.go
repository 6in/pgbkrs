package diff

import "github.com/pgbkrs/pgbackup/internal/core"

// DiffResult holds the schema comparison output.
type DiffResult struct {
	Added   []core.ObjectHeader
	Removed []core.ObjectHeader
	Changed []ObjectChange
}

// ObjectChange describes a schema object that exists in both backups but differs.
type ObjectChange struct {
	Header  core.ObjectHeader
	Details []string // human-readable change lines per spec 9.3
}

// compareObjects compares two object maps and returns a DiffResult.
func compareObjects(a, b map[string]core.ObjectDef) DiffResult {
	return DiffResult{}
}
