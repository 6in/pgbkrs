package trigger

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts TriggerDef to/from YAML bytes.
type Serializer struct{}

type yamlTriggerDef struct {
	Kind         string   `yaml:"kind"`
	Schema       string   `yaml:"schema"`
	Name         string   `yaml:"name"`
	Timing       string   `yaml:"timing"`
	Events       []string `yaml:"events,flow"`
	TableName    string   `yaml:"table_name"`
	FunctionName string   `yaml:"function_name"`
}

func toYAML(td core.TriggerDef) yamlTriggerDef {
	return yamlTriggerDef{
		Kind:         string(td.Kind),
		Schema:       td.Schema,
		Name:         td.Name,
		Timing:       td.Timing,
		Events:       td.Events,
		TableName:    td.TableName,
		FunctionName: td.FunctionName,
	}
}

func fromYAML(yd yamlTriggerDef) core.TriggerDef {
	return core.TriggerDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.ObjectKind(yd.Kind),
			Schema: yd.Schema,
			Name:   yd.Name,
		},
		Timing:       yd.Timing,
		Events:       yd.Events,
		TableName:    yd.TableName,
		FunctionName: yd.FunctionName,
	}
}

// Serialize converts an ObjectDef to YAML bytes.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	td, ok := def.(core.TriggerDef)
	if !ok {
		return nil, fmt.Errorf("trigger.Serializer.Serialize: expected core.TriggerDef, got %T", def)
	}

	yd := toYAML(td)
	return yaml.Marshal(&yd)
}

// Deserialize converts YAML bytes to an ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	var yd yamlTriggerDef
	if err := yaml.Unmarshal(data, &yd); err != nil {
		return nil, fmt.Errorf("trigger.Serializer.Deserialize: %w", err)
	}

	return fromYAML(yd), nil
}
