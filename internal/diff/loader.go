package diff

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/resolve"
	serializeCompositetype "github.com/pgbkrs/pgbackup/internal/serialize/compositetype"
	serializeDomain "github.com/pgbkrs/pgbackup/internal/serialize/domain"
	serializeEnum "github.com/pgbkrs/pgbackup/internal/serialize/enum"
	serializeForeignkey "github.com/pgbkrs/pgbackup/internal/serialize/foreignkey"
	serializeFunction "github.com/pgbkrs/pgbackup/internal/serialize/function"
	serializeMatview "github.com/pgbkrs/pgbackup/internal/serialize/matview"
	serializePolicy "github.com/pgbkrs/pgbackup/internal/serialize/policy"
	serializeSequence "github.com/pgbkrs/pgbackup/internal/serialize/sequence"
	serializeTable "github.com/pgbkrs/pgbackup/internal/serialize/table"
	serializeTrigger "github.com/pgbkrs/pgbackup/internal/serialize/trigger"
	serializeView "github.com/pgbkrs/pgbackup/internal/serialize/view"
)

// kindToDir maps object kind strings to their backup directory names.
// Duplicated from internal/restore/loader.go — do NOT import restore package.
var kindToDir = map[string]string{
	"table":             "tables",
	"view":              "views",
	"materialized_view": "materialized_views",
	"function":          "functions",
	"sequence":          "sequences",
	"trigger":           "triggers",
	"type":              "types",
	"domain":            "domains",
	"enum":              "enums",
	"policy":            "policies",
	"fk":               "foreignkeys",
}

// defYAMLPath returns the path to def.yaml for a restore entry.
// Partition children: <schema>/tables/<parent>/partitions/<child>/def.yaml
// Regular tables:     <schema>/tables/<name>/def.yaml
// Non-table objects:  <schema>/<kindDir>/<name>.yaml
// Duplicated from internal/restore/loader.go — do NOT import restore package.
func defYAMLPath(backupDir string, entry resolve.RestoreEntry) string {
	if entry.Kind == "table" {
		if entry.FromTable != "" {
			return filepath.Join(backupDir, entry.Schema, "tables", entry.FromTable, "partitions", entry.Name, "def.yaml")
		}
		return filepath.Join(backupDir, entry.Schema, "tables", entry.Name, "def.yaml")
	}
	dir, ok := kindToDir[entry.Kind]
	if !ok {
		return ""
	}
	return filepath.Join(backupDir, entry.Schema, dir, entry.Name+".yaml")
}

// serializerFor returns the Serializer for a given object kind string.
// Duplicated from internal/restore/loader.go — do NOT import restore package.
func serializerFor(kind string) core.Serializer {
	// The manifest uses "fk" for foreign keys; core uses "foreign_key".
	if kind == "fk" {
		return &serializeForeignkey.Serializer{}
	}
	switch core.ObjectKind(kind) {
	case core.KindTable:
		return &serializeTable.Serializer{}
	case core.KindView:
		return &serializeView.Serializer{}
	case core.KindMaterializedView:
		return &serializeMatview.Serializer{}
	case core.KindFunction:
		return &serializeFunction.Serializer{}
	case core.KindSequence:
		return &serializeSequence.Serializer{}
	case core.KindTrigger:
		return &serializeTrigger.Serializer{}
	case core.KindType:
		return &serializeCompositetype.Serializer{}
	case core.KindDomain:
		return &serializeDomain.Serializer{}
	case core.KindEnum:
		return &serializeEnum.Serializer{}
	case core.KindPolicy:
		return &serializePolicy.Serializer{}
	case core.KindForeignKey:
		return &serializeForeignkey.Serializer{}
	default:
		return nil
	}
}

// loadBackup reads a backup directory and returns all schema objects keyed by
// "schema.kind.name". FKs use kind="fk".
func loadBackup(backupDir string) (map[string]core.ObjectDef, error) {
	manifestPath := filepath.Join(backupDir, "_manifest.yaml")
	manifest, err := resolve.ReadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("loadBackup: read manifest: %w", err)
	}

	result := make(map[string]core.ObjectDef)
	for _, entry := range manifest.RestoreOrder {
		yamlPath := defYAMLPath(backupDir, entry)
		if yamlPath == "" {
			continue
		}
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("loadBackup: read def.yaml for %s.%s: %w", entry.Schema, entry.Name, err)
		}
		s := serializerFor(entry.Kind)
		if s == nil {
			continue
		}
		def, err := s.Deserialize(data)
		if err != nil {
			return nil, fmt.Errorf("loadBackup: deserialize %s.%s: %w", entry.Schema, entry.Name, err)
		}
		key := entry.Schema + "." + entry.Kind + "." + entry.Name
		result[key] = def
	}
	return result, nil
}
