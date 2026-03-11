package function

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts FunctionDef to/from YAML bytes.
type Serializer struct{}

type yamlFunctionDef struct {
	Kind       string `yaml:"kind"`
	Schema     string `yaml:"schema"`
	Name       string `yaml:"name"`
	Definition string `yaml:"definition"`
	ArgTypes   string `yaml:"arg_types"`
	ReturnType string `yaml:"return_type"`
}

func toYAML(fd core.FunctionDef) yamlFunctionDef {
	return yamlFunctionDef{
		Kind:       string(fd.Kind),
		Schema:     fd.Schema,
		Name:       fd.Name,
		Definition: fd.Definition,
		ArgTypes:   fd.ArgTypes,
		ReturnType: fd.ReturnType,
	}
}

func fromYAML(yd yamlFunctionDef) core.FunctionDef {
	return core.FunctionDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.ObjectKind(yd.Kind),
			Schema: yd.Schema,
			Name:   yd.Name,
		},
		Definition: yd.Definition,
		ArgTypes:   yd.ArgTypes,
		ReturnType: yd.ReturnType,
	}
}

// Serialize converts an ObjectDef to YAML bytes.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	fd, ok := def.(core.FunctionDef)
	if !ok {
		return nil, fmt.Errorf("function.Serializer.Serialize: expected core.FunctionDef, got %T", def)
	}

	yd := toYAML(fd)
	return yaml.Marshal(&yd)
}

// Deserialize converts YAML bytes to an ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	var yd yamlFunctionDef
	if err := yaml.Unmarshal(data, &yd); err != nil {
		return nil, fmt.Errorf("function.Serializer.Deserialize: %w", err)
	}

	return fromYAML(yd), nil
}
