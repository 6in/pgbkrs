package view

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts ViewDef to/from YAML bytes.
type Serializer struct{}

type yamlViewDef struct {
	Kind       string `yaml:"kind"`
	Schema     string `yaml:"schema"`
	Name       string `yaml:"name"`
	Definition string `yaml:"definition"`
	Owner      string `yaml:"owner"`
}

func toYAML(vd core.ViewDef) yamlViewDef {
	return yamlViewDef{
		Kind:       string(vd.Kind),
		Schema:     vd.Schema,
		Name:       vd.Name,
		Definition: vd.Definition,
		Owner:      vd.Owner,
	}
}

func fromYAML(yd yamlViewDef) core.ViewDef {
	return core.ViewDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.ObjectKind(yd.Kind),
			Schema: yd.Schema,
			Name:   yd.Name,
		},
		Definition: yd.Definition,
		Owner:      yd.Owner,
	}
}

// Serialize converts an ObjectDef to YAML bytes.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	vd, ok := def.(core.ViewDef)
	if !ok {
		return nil, fmt.Errorf("view.Serializer.Serialize: expected core.ViewDef, got %T", def)
	}

	yd := toYAML(vd)
	return yaml.Marshal(&yd)
}

// Deserialize converts YAML bytes to an ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	var yd yamlViewDef
	if err := yaml.Unmarshal(data, &yd); err != nil {
		return nil, fmt.Errorf("view.Serializer.Deserialize: %w", err)
	}

	return fromYAML(yd), nil
}
