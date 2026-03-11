package foreignkey

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts ForeignKeyDef to/from YAML bytes.
type Serializer struct{}

type yamlForeignKeyDef struct {
	Kind         string `yaml:"kind"`
	Schema       string `yaml:"schema"`
	Name         string `yaml:"name"`
	SourceTable  string `yaml:"source_table"`
	TargetSchema string `yaml:"target_schema"`
	TargetTable  string `yaml:"target_table"`
	Definition   string `yaml:"definition"`
}

func toYAML(fkd *core.ForeignKeyDef) yamlForeignKeyDef {
	return yamlForeignKeyDef{
		Kind:         string(fkd.Kind),
		Schema:       fkd.Schema,
		Name:         fkd.Name,
		SourceTable:  fkd.SourceTable,
		TargetSchema: fkd.TargetSchema,
		TargetTable:  fkd.TargetTable,
		Definition:   fkd.Definition,
	}
}

func fromYAML(yd yamlForeignKeyDef) *core.ForeignKeyDef {
	return &core.ForeignKeyDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.ObjectKind(yd.Kind),
			Schema: yd.Schema,
			Name:   yd.Name,
		},
		SourceTable:  yd.SourceTable,
		TargetSchema: yd.TargetSchema,
		TargetTable:  yd.TargetTable,
		Definition:   yd.Definition,
	}
}

// Serialize converts an ObjectDef to YAML bytes.
// Uses POINTER type assertion (*core.ForeignKeyDef) matching DDL generator convention.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	fkd, ok := def.(*core.ForeignKeyDef)
	if !ok {
		return nil, fmt.Errorf("foreignkey.Serializer.Serialize: expected *core.ForeignKeyDef, got %T", def)
	}

	yd := toYAML(fkd)
	return yaml.Marshal(&yd)
}

// Deserialize converts YAML bytes to an ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	var yd yamlForeignKeyDef
	if err := yaml.Unmarshal(data, &yd); err != nil {
		return nil, fmt.Errorf("foreignkey.Serializer.Deserialize: %w", err)
	}

	return fromYAML(yd), nil
}
