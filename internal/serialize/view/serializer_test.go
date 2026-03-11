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
}
