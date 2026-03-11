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
	if m["table_name"] != "users" {
		t.Errorf("table_name = %v, want users", m["table_name"])
	}
	if m["command"] != "SELECT" {
		t.Errorf("command = %v, want SELECT", m["command"])
	}
	roles, ok := m["roles"].([]interface{})
	if !ok {
		t.Fatalf("roles is not a list: %T", m["roles"])
	}
	if len(roles) != 1 || roles[0] != "app_user" {
		t.Errorf("roles = %v, want [app_user]", roles)
	}
	if m["using"] != "(user_id = current_user_id())" {
		t.Errorf("using = %v, want USING expression", m["using"])
	}
}

func TestSerializePublicRole(t *testing.T) {
	s := &policy.Serializer{}
	def := core.PolicyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "all_read"},
		TableName:    "posts",
		Command:      "ALL",
		Roles:        []string{"PUBLIC"},
		Using:        "true",
		WithCheck:    "true",
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["with_check"] != "true" {
		t.Errorf("with_check = %v, want true", m["with_check"])
	}
}

func TestDeserialize(t *testing.T) {
	s := &policy.Serializer{}
	yamlData := []byte(`kind: policy
schema: public
name: user_isolation
table_name: users
command: SELECT
roles: [app_user]
using: "(user_id = current_user_id())"
`)

	obj, err := s.Deserialize(yamlData)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	pd, ok := obj.(core.PolicyDef)
	if !ok {
		t.Fatalf("expected core.PolicyDef, got %T", obj)
	}

	if pd.TableName != "users" {
		t.Errorf("TableName = %q, want users", pd.TableName)
	}
	if pd.Command != "SELECT" {
		t.Errorf("Command = %q, want SELECT", pd.Command)
	}
	if len(pd.Roles) != 1 || pd.Roles[0] != "app_user" {
		t.Errorf("Roles = %v, want [app_user]", pd.Roles)
	}
	if pd.Using != "(user_id = current_user_id())" {
		t.Errorf("Using = %q, want USING expression", pd.Using)
	}
}
