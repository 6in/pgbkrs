package domain

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts DomainDef to/from YAML bytes.
type Serializer struct{}

type yamlDomainDef struct {
	Kind            string `yaml:"kind"`
	Schema          string `yaml:"schema"`
	Name            string `yaml:"name"`
	BaseType        string `yaml:"base_type"`
	Nullable        bool   `yaml:"nullable"` // NO omitempty: false is meaningful
	Default         string `yaml:"default,omitempty"`
	CheckName       string `yaml:"check_name,omitempty"`
	CheckDefinition string `yaml:"check_definition,omitempty"`
}

func toYAML(dd core.DomainDef) yamlDomainDef {
	return yamlDomainDef{
		Kind:            string(dd.Kind),
		Schema:          dd.Schema,
		Name:            dd.Name,
		BaseType:        dd.BaseType,
		Nullable:        dd.Nullable,
		Default:         dd.Default,
		CheckName:       dd.CheckName,
		CheckDefinition: dd.CheckDefinition,
	}
}

func fromYAML(yd yamlDomainDef) core.DomainDef {
	return core.DomainDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.ObjectKind(yd.Kind),
			Schema: yd.Schema,
			Name:   yd.Name,
		},
		BaseType:        yd.BaseType,
		Nullable:        yd.Nullable,
		Default:         yd.Default,
		CheckName:       yd.CheckName,
		CheckDefinition: yd.CheckDefinition,
	}
}

// Serialize converts an ObjectDef to YAML bytes.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	dd, ok := def.(core.DomainDef)
	if !ok {
		return nil, fmt.Errorf("domain.Serializer.Serialize: expected core.DomainDef, got %T", def)
	}

	yd := toYAML(dd)
	return yaml.Marshal(&yd)
}

// Deserialize converts YAML bytes to an ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	var yd yamlDomainDef
	if err := yaml.Unmarshal(data, &yd); err != nil {
		return nil, fmt.Errorf("domain.Serializer.Deserialize: %w", err)
	}

	return fromYAML(yd), nil
}
