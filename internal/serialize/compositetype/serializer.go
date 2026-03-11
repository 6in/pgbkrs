package compositetype

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts TypeDef to/from YAML bytes.
type Serializer struct{}

type yamlTypeDef struct {
	Kind   string      `yaml:"kind"`
	Schema string      `yaml:"schema"`
	Name   string      `yaml:"name"`
	Fields []yamlField `yaml:"fields"`
}

type yamlField struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

func toYAML(td core.TypeDef) yamlTypeDef {
	yd := yamlTypeDef{
		Kind:   string(td.Kind),
		Schema: td.Schema,
		Name:   td.Name,
	}
	for _, f := range td.Fields {
		yd.Fields = append(yd.Fields, yamlField{
			Name: f.Name,
			Type: f.Type,
		})
	}
	return yd
}

func fromYAML(yd yamlTypeDef) core.TypeDef {
	td := core.TypeDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.ObjectKind(yd.Kind),
			Schema: yd.Schema,
			Name:   yd.Name,
		},
	}
	for _, f := range yd.Fields {
		td.Fields = append(td.Fields, core.CompositeField{
			Name: f.Name,
			Type: f.Type,
		})
	}
	return td
}

// Serialize converts an ObjectDef to YAML bytes.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	td, ok := def.(core.TypeDef)
	if !ok {
		return nil, fmt.Errorf("compositetype.Serializer.Serialize: expected core.TypeDef, got %T", def)
	}

	yd := toYAML(td)
	return yaml.Marshal(&yd)
}

// Deserialize converts YAML bytes to an ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	var yd yamlTypeDef
	if err := yaml.Unmarshal(data, &yd); err != nil {
		return nil, fmt.Errorf("compositetype.Serializer.Deserialize: %w", err)
	}

	return fromYAML(yd), nil
}
