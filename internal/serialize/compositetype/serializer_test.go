package compositetype_test

import (
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/serialize/compositetype"
)

func TestSerialize(t *testing.T) {
	s := &compositetype.Serializer{}
	def := core.TypeDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "public", Name: "address"},
		Fields: []core.CompositeField{
			{Name: "street", Type: "text"},
			{Name: "city", Type: "text"},
		},
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["kind"] != "type" {
		t.Errorf("kind = %v, want type", m["kind"])
	}
	if m["schema"] != "public" {
		t.Errorf("schema = %v, want public", m["schema"])
	}
	if m["name"] != "address" {
		t.Errorf("name = %v, want address", m["name"])
	}
}
