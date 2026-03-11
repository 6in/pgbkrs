package restore

import "github.com/pgbkrs/pgbackup/internal/resolve"

// defYAMLPath returns the path to def.yaml for a given restore entry within backupDir.
// Partition children are routed to <schema>/tables/<parent>/partitions/<child>/def.yaml.
func defYAMLPath(backupDir string, entry resolve.RestoreEntry) string {
	// TODO: implement in 08-02
	return ""
}
