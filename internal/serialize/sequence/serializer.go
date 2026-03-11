package sequence

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts SequenceDef to/from YAML bytes.
type Serializer struct{}

type yamlSequenceDef struct {
	Kind        string `yaml:"kind"`
	Schema      string `yaml:"schema"`
	Name        string `yaml:"name"`
	StartValue  int64  `yaml:"start_value"`
	MinValue    int64  `yaml:"min_value"`
	MaxValue    int64  `yaml:"max_value"`
	IncrementBy int64  `yaml:"increment_by"`
	Cycle       bool   `yaml:"cycle"`       // NO omitempty: false is meaningful
	Cache       int64  `yaml:"cache"`
	LastValue   int64  `yaml:"last_value"`
	IsCalled    bool   `yaml:"is_called"`   // NO omitempty: false is meaningful
}

func toYAML(sd core.SequenceDef) yamlSequenceDef {
	return yamlSequenceDef{
		Kind:        string(sd.Kind),
		Schema:      sd.Schema,
		Name:        sd.Name,
		StartValue:  sd.StartValue,
		MinValue:    sd.MinValue,
		MaxValue:    sd.MaxValue,
		IncrementBy: sd.IncrementBy,
		Cycle:       sd.Cycle,
		Cache:       sd.Cache,
		LastValue:   sd.LastValue,
		IsCalled:    sd.IsCalled,
	}
}

func fromYAML(yd yamlSequenceDef) core.SequenceDef {
	return core.SequenceDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.ObjectKind(yd.Kind),
			Schema: yd.Schema,
			Name:   yd.Name,
		},
		StartValue:  yd.StartValue,
		MinValue:    yd.MinValue,
		MaxValue:    yd.MaxValue,
		IncrementBy: yd.IncrementBy,
		Cycle:       yd.Cycle,
		Cache:       yd.Cache,
		LastValue:   yd.LastValue,
		IsCalled:    yd.IsCalled,
	}
}

// Serialize converts an ObjectDef to YAML bytes.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	sd, ok := def.(core.SequenceDef)
	if !ok {
		return nil, fmt.Errorf("sequence.Serializer.Serialize: expected core.SequenceDef, got %T", def)
	}

	yd := toYAML(sd)
	return yaml.Marshal(&yd)
}

// Deserialize converts YAML bytes to an ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	var yd yamlSequenceDef
	if err := yaml.Unmarshal(data, &yd); err != nil {
		return nil, fmt.Errorf("sequence.Serializer.Deserialize: %w", err)
	}

	return fromYAML(yd), nil
}
