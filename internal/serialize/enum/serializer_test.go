package enum_test

import (
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/serialize/enum"
)

func TestSerialize(t *testing.T) {
	s := &enum.Serializer{}
	def := core.EnumDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status"},
		Labels:       []string{"active", "inactive", "deleted"},
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["kind"] != "enum" {
		t.Errorf("kind = %v, want enum", m["kind"])
	}
	if m["schema"] != "public" {
		t.Errorf("schema = %v, want public", m["schema"])
	}
	if m["name"] != "status" {
		t.Errorf("name = %v, want status", m["name"])
	}

	labels, ok := m["labels"].([]interface{})
	if !ok {
		t.Fatalf("labels is not a list: %T", m["labels"])
	}
	if len(labels) != 3 {
		t.Errorf("labels length = %d, want 3", len(labels))
	}
	if labels[0] != "active" || labels[1] != "inactive" || labels[2] != "deleted" {
		t.Errorf("labels = %v, want [active inactive deleted]", labels)
	}
}

func TestDeserialize(t *testing.T) {
	s := &enum.Serializer{}
	yamlData := []byte(`kind: enum
schema: public
name: status
labels: [active, inactive, deleted]
`)

	obj, err := s.Deserialize(yamlData)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	ed, ok := obj.(core.EnumDef)
	if !ok {
		t.Fatalf("expected core.EnumDef, got %T", obj)
	}

	if len(ed.Labels) != 3 {
		t.Fatalf("expected 3 labels, got %d", len(ed.Labels))
	}
	if ed.Labels[0] != "active" || ed.Labels[1] != "inactive" || ed.Labels[2] != "deleted" {
		t.Errorf("Labels = %v, want [active inactive deleted]", ed.Labels)
	}
}
