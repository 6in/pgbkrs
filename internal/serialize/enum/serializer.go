package enum

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts EnumDef to/from YAML bytes.
type Serializer struct{}

type yamlEnumDef struct {
	Kind   string   `yaml:"kind"`
	Schema string   `yaml:"schema"`
	Name   string   `yaml:"name"`
	Labels []string `yaml:"labels,flow"`
}

func toYAML(ed core.EnumDef) yamlEnumDef {
	return yamlEnumDef{
		Kind:   string(ed.Kind),
		Schema: ed.Schema,
		Name:   ed.Name,
		Labels: ed.Labels,
	}
}

func fromYAML(yd yamlEnumDef) core.EnumDef {
	return core.EnumDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.ObjectKind(yd.Kind),
			Schema: yd.Schema,
			Name:   yd.Name,
		},
		Labels: yd.Labels,
	}
}

// Serialize converts an ObjectDef to YAML bytes.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	ed, ok := def.(core.EnumDef)
	if !ok {
		return nil, fmt.Errorf("enum.Serializer.Serialize: expected core.EnumDef, got %T", def)
	}

	yd := toYAML(ed)
	return yaml.Marshal(&yd)
}

// Deserialize converts YAML bytes to an ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	var yd yamlEnumDef
	if err := yaml.Unmarshal(data, &yd); err != nil {
		return nil, fmt.Errorf("enum.Serializer.Deserialize: %w", err)
	}

	return fromYAML(yd), nil
}
