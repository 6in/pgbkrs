package restore

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/resolve"
	ddlCompositetype "github.com/pgbkrs/pgbackup/internal/ddl/compositetype"
	ddlDomain "github.com/pgbkrs/pgbackup/internal/ddl/domain"
	ddlEnum "github.com/pgbkrs/pgbackup/internal/ddl/enum"
	ddlForeignkey "github.com/pgbkrs/pgbackup/internal/ddl/foreignkey"
	ddlFunction "github.com/pgbkrs/pgbackup/internal/ddl/function"
	ddlMatview "github.com/pgbkrs/pgbackup/internal/ddl/matview"
	ddlPolicy "github.com/pgbkrs/pgbackup/internal/ddl/policy"
	ddlSequence "github.com/pgbkrs/pgbackup/internal/ddl/sequence"
	ddlTable "github.com/pgbkrs/pgbackup/internal/ddl/table"
	ddlTrigger "github.com/pgbkrs/pgbackup/internal/ddl/trigger"
	ddlView "github.com/pgbkrs/pgbackup/internal/ddl/view"
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

// ddlGeneratorFor returns the DDLGenerator for a given object kind string.
// Returns nil for unknown kinds.
// Note: the manifest uses "fk" for foreign keys; this is handled explicitly.
func ddlGeneratorFor(kind string) core.DDLGenerator {
	// Handle the manifest "fk" string which differs from core.KindForeignKey ("foreign_key").
	if kind == "fk" {
		return &ddlForeignkey.DDLGenerator{}
	}
	switch core.ObjectKind(kind) {
	case core.KindTable:
		return &ddlTable.DDLGenerator{}
	case core.KindView:
		return &ddlView.DDLGenerator{}
	case core.KindMaterializedView:
		return &ddlMatview.DDLGenerator{}
	case core.KindFunction:
		return &ddlFunction.DDLGenerator{}
	case core.KindSequence:
		return &ddlSequence.DDLGenerator{}
	case core.KindTrigger:
		return &ddlTrigger.DDLGenerator{}
	case core.KindType:
		return &ddlCompositetype.DDLGenerator{}
	case core.KindDomain:
		return &ddlDomain.DDLGenerator{}
	case core.KindEnum:
		return &ddlEnum.DDLGenerator{}
	case core.KindPolicy:
		return &ddlPolicy.DDLGenerator{}
	case core.KindForeignKey:
		return &ddlForeignkey.DDLGenerator{}
	default:
		return nil
	}
}

// loadObjectDef reads def.yaml from disk and deserializes it into a core.ObjectDef.
// Returns an error for FK entries (no YAML file exists).
func loadObjectDef(backupDir string, entry resolve.RestoreEntry) (core.ObjectDef, error) {
	yamlPath := defYAMLPath(backupDir, entry)
	if yamlPath == "" {
		return nil, fmt.Errorf("no def.yaml for kind %q (entry: %s.%s)", entry.Kind, entry.Schema, entry.Name)
	}
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read def.yaml for %s.%s: %w", entry.Schema, entry.Name, err)
	}
	s := serializerFor(entry.Kind)
	if s == nil {
		return nil, fmt.Errorf("no serializer for kind %q", entry.Kind)
	}
	def, err := s.Deserialize(data)
	if err != nil {
		return nil, fmt.Errorf("deserialize %s.%s: %w", entry.Schema, entry.Name, err)
	}
	return def, nil
}
