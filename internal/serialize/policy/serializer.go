package policy

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts PolicyDef to/from YAML bytes.
type Serializer struct{}

type yamlPolicyDef struct {
	Kind      string   `yaml:"kind"`
	Schema    string   `yaml:"schema"`
	Name      string   `yaml:"name"`
	TableName string   `yaml:"table_name"`
	Command   string   `yaml:"command"`
	Roles     []string `yaml:"roles,flow"`
	Using     string   `yaml:"using,omitempty"`
	WithCheck string   `yaml:"with_check,omitempty"`
}

func toYAML(pd core.PolicyDef) yamlPolicyDef {
	return yamlPolicyDef{
		Kind:      string(pd.Kind),
		Schema:    pd.Schema,
		Name:      pd.Name,
		TableName: pd.TableName,
		Command:   pd.Command,
		Roles:     pd.Roles,
		Using:     pd.Using,
		WithCheck: pd.WithCheck,
	}
}

func fromYAML(yd yamlPolicyDef) core.PolicyDef {
	return core.PolicyDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.ObjectKind(yd.Kind),
			Schema: yd.Schema,
			Name:   yd.Name,
		},
		TableName: yd.TableName,
		Command:   yd.Command,
		Roles:     yd.Roles,
		Using:     yd.Using,
		WithCheck: yd.WithCheck,
	}
}

// Serialize converts an ObjectDef to YAML bytes.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	pd, ok := def.(core.PolicyDef)
	if !ok {
		return nil, fmt.Errorf("policy.Serializer.Serialize: expected core.PolicyDef, got %T", def)
	}

	yd := toYAML(pd)
	return yaml.Marshal(&yd)
}

// Deserialize converts YAML bytes to an ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	var yd yamlPolicyDef
	if err := yaml.Unmarshal(data, &yd); err != nil {
		return nil, fmt.Errorf("policy.Serializer.Deserialize: %w", err)
	}

	return fromYAML(yd), nil
}
