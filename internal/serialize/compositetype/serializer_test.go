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

	fields, ok := m["fields"].([]interface{})
	if !ok || len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %v", m["fields"])
	}
	f0, ok := fields[0].(map[string]interface{})
	if !ok {
		t.Fatalf("field[0] not a map: %T", fields[0])
	}
	if f0["name"] != "street" {
		t.Errorf("field[0].name = %v, want street", f0["name"])
	}
	if f0["type"] != "text" {
		t.Errorf("field[0].type = %v, want text", f0["type"])
	}
}

func TestDeserialize(t *testing.T) {
	s := &compositetype.Serializer{}
	yamlData := []byte(`kind: type
schema: public
name: address
fields:
  - name: street
    type: text
  - name: city
    type: text
`)

	obj, err := s.Deserialize(yamlData)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	td, ok := obj.(core.TypeDef)
	if !ok {
		t.Fatalf("expected core.TypeDef, got %T", obj)
	}

	if td.Kind != core.KindType {
		t.Errorf("Kind = %q, want type", td.Kind)
	}
	if len(td.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(td.Fields))
	}
	if td.Fields[0].Name != "street" || td.Fields[0].Type != "text" {
		t.Errorf("field[0] = %+v, want street/text", td.Fields[0])
	}
}
