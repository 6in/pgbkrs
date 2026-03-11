package view_test

import (
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/serialize/view"
)

func TestSerialize(t *testing.T) {
	s := &view.Serializer{}
	def := core.ViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindView, Schema: "public", Name: "active_users"},
		Definition:   "SELECT id, email FROM users WHERE active = true",
		Owner:        "postgres",
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["kind"] != "view" {
		t.Errorf("kind = %v, want view", m["kind"])
	}
	if m["schema"] != "public" {
		t.Errorf("schema = %v, want public", m["schema"])
	}
	if m["name"] != "active_users" {
		t.Errorf("name = %v, want active_users", m["name"])
	}
	if m["definition"] != "SELECT id, email FROM users WHERE active = true" {
		t.Errorf("definition = %v, want SELECT body", m["definition"])
	}
	if m["owner"] != "postgres" {
		t.Errorf("owner = %v, want postgres", m["owner"])
	}
}

func TestDeserialize(t *testing.T) {
	s := &view.Serializer{}
	yamlData := []byte(`kind: view
schema: public
name: active_users
definition: "SELECT id, email FROM users WHERE active = true"
owner: postgres
`)

	obj, err := s.Deserialize(yamlData)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	vd, ok := obj.(core.ViewDef)
	if !ok {
		t.Fatalf("expected core.ViewDef, got %T", obj)
	}

	if vd.Kind != core.KindView {
		t.Errorf("Kind = %q, want view", vd.Kind)
	}
	if vd.Schema != "public" {
		t.Errorf("Schema = %q, want public", vd.Schema)
	}
	if vd.Name != "active_users" {
		t.Errorf("Name = %q, want active_users", vd.Name)
	}
	if vd.Definition != "SELECT id, email FROM users WHERE active = true" {
		t.Errorf("Definition = %q, want SELECT body", vd.Definition)
	}
	if vd.Owner != "postgres" {
		t.Errorf("Owner = %q, want postgres", vd.Owner)
	}
}
