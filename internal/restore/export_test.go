package restore

import "github.com/pgbkrs/pgbackup/internal/resolve"

// FilteredRestoreOrder exposes filteredRestoreOrder for black-box testing in package restore_test.
func FilteredRestoreOrder(manifest *resolve.Manifest, opts Options) ([]resolve.RestoreEntry, error) {
	return filteredRestoreOrder(manifest, opts)
}
