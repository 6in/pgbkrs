package foreignkey_test

import (
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/serialize/foreignkey"
)

func TestSerialize(t *testing.T) {
	s := &foreignkey.Serializer{}
	def := &core.ForeignKeyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "orders_customer_id_fkey"},
		SourceTable:  "orders",
		TargetSchema: "public",
		TargetTable:  "customers",
		Definition:   "FOREIGN KEY (customer_id) REFERENCES customers(id)",
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["kind"] != "foreign_key" {
		t.Errorf("kind = %v, want foreign_key", m["kind"])
	}
	if m["schema"] != "public" {
		t.Errorf("schema = %v, want public", m["schema"])
	}
	if m["name"] != "orders_customer_id_fkey" {
		t.Errorf("name = %v, want orders_customer_id_fkey", m["name"])
	}
}
