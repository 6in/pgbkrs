package matview

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts MaterializedViewDef to/from YAML bytes.
type Serializer struct{}

type yamlMatViewDef struct {
	Kind        string `yaml:"kind"`
	Schema      string `yaml:"schema"`
	Name        string `yaml:"name"`
	Definition  string `yaml:"definition"`
	Owner       string `yaml:"owner"`
	IsPopulated bool   `yaml:"is_populated"` // NO omitempty: false is meaningful
}

func toYAML(mvd core.MaterializedViewDef) yamlMatViewDef {
	return yamlMatViewDef{
		Kind:        string(mvd.Kind),
		Schema:      mvd.Schema,
		Name:        mvd.Name,
		Definition:  mvd.Definition,
		Owner:       mvd.Owner,
		IsPopulated: mvd.IsPopulated,
	}
}

func fromYAML(yd yamlMatViewDef) core.MaterializedViewDef {
	return core.MaterializedViewDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.ObjectKind(yd.Kind),
			Schema: yd.Schema,
			Name:   yd.Name,
		},
		Definition:  yd.Definition,
		Owner:       yd.Owner,
		IsPopulated: yd.IsPopulated,
	}
}

// Serialize converts an ObjectDef to YAML bytes.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	mvd, ok := def.(core.MaterializedViewDef)
	if !ok {
		return nil, fmt.Errorf("matview.Serializer.Serialize: expected core.MaterializedViewDef, got %T", def)
	}

	yd := toYAML(mvd)
	return yaml.Marshal(&yd)
}

// Deserialize converts YAML bytes to an ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	var yd yamlMatViewDef
	if err := yaml.Unmarshal(data, &yd); err != nil {
		return nil, fmt.Errorf("matview.Serializer.Deserialize: %w", err)
	}

	return fromYAML(yd), nil
}
