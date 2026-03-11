package policy_test

import (
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/serialize/policy"
)

func TestSerialize(t *testing.T) {
	s := &policy.Serializer{}
	def := core.PolicyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "user_isolation"},
		TableName:    "users",
		Command:      "SELECT",
		Roles:        []string{"app_user"},
		Using:        "(user_id = current_user_id())",
		WithCheck:    "",
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["kind"] != "policy" {
		t.Errorf("kind = %v, want policy", m["kind"])
	}
	if m["schema"] != "public" {
		t.Errorf("schema = %v, want public", m["schema"])
	}
	if m["name"] != "user_isolation" {
		t.Errorf("name = %v, want user_isolation", m["name"])
	}
}
