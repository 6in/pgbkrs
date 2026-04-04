package datadiff

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/resolve"
	serializeTable "github.com/pgbkrs/pgbackup/internal/serialize/table"
)

// tableEntry holds the metadata needed to diff one table's CSV data.
type tableEntry struct {
	Schema   string
	Name     string
	PKCols   []string // primary key column names; empty if no PK
	Columns  []string // CSV column order from DataMeta
	CSVPath  string
	Checksum string // sha256:... from DataMeta; empty string if unavailable
}

// loadBackupTables reads the manifest and returns a map of "schema.name" → tableEntry
// for every table that has an accessible data.csv file.
func loadBackupTables(backupDir string) (map[string]*tableEntry, error) {
	manifestPath := filepath.Join(backupDir, "_manifest.yaml")
	manifest, err := resolve.ReadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	s := &serializeTable.Serializer{}
	result := make(map[string]*tableEntry)

	for _, entry := range manifest.RestoreOrder {
		if entry.Kind != "table" {
			continue
		}

		var defPath, csvDir string
		if entry.FromTable != "" {
			// Partition child
			base := filepath.Join(backupDir, entry.Schema, "tables", entry.FromTable, "partitions", entry.Name)
			defPath = filepath.Join(base, "def.yaml")
			csvDir = base
		} else {
			base := filepath.Join(backupDir, entry.Schema, "tables", entry.Name)
			defPath = filepath.Join(base, "def.yaml")
			csvDir = base
		}

		data, err := os.ReadFile(defPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read def.yaml for %s.%s: %w", entry.Schema, entry.Name, err)
		}

		def, err := s.Deserialize(data)
		if err != nil {
			return nil, fmt.Errorf("deserialize def.yaml for %s.%s: %w", entry.Schema, entry.Name, err)
		}

		td, ok := def.(*core.TableDef)
		if !ok {
			continue
		}
		if td.DataMeta == nil {
			continue // partitioned parent — no CSV
		}

		csvPath := filepath.Join(csvDir, td.DataMeta.File)
		if _, err := os.Stat(csvPath); os.IsNotExist(err) {
			continue
		}

		var pkCols []string
		if td.Constraints.PrimaryKey != nil {
			pkCols = make([]string, len(td.Constraints.PrimaryKey.Columns))
			copy(pkCols, td.Constraints.PrimaryKey.Columns)
		}

		key := entry.Schema + "." + entry.Name
		result[key] = &tableEntry{
			Schema:   entry.Schema,
			Name:     entry.Name,
			PKCols:   pkCols,
			Columns:  td.DataMeta.Columns,
			CSVPath:  csvPath,
			Checksum: td.DataMeta.Checksum,
		}
	}

	return result, nil
}
