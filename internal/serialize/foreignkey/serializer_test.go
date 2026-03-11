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
	if m["source_table"] != "orders" {
		t.Errorf("source_table = %v, want orders", m["source_table"])
	}
	if m["target_schema"] != "public" {
		t.Errorf("target_schema = %v, want public", m["target_schema"])
	}
	if m["target_table"] != "customers" {
		t.Errorf("target_table = %v, want customers", m["target_table"])
	}
	if m["definition"] != "FOREIGN KEY (customer_id) REFERENCES customers(id)" {
		t.Errorf("definition = %v, want FK definition", m["definition"])
	}
}

func TestDeserialize(t *testing.T) {
	s := &foreignkey.Serializer{}
	yamlData := []byte(`kind: foreign_key
schema: public
name: orders_customer_id_fkey
source_table: orders
target_schema: public
target_table: customers
definition: "FOREIGN KEY (customer_id) REFERENCES customers(id)"
`)

	obj, err := s.Deserialize(yamlData)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	fkd, ok := obj.(*core.ForeignKeyDef)
	if !ok {
		t.Fatalf("expected *core.ForeignKeyDef, got %T", obj)
	}

	if fkd.Kind != core.KindForeignKey {
		t.Errorf("Kind = %q, want foreign_key", fkd.Kind)
	}
	if fkd.SourceTable != "orders" {
		t.Errorf("SourceTable = %q, want orders", fkd.SourceTable)
	}
	if fkd.TargetSchema != "public" {
		t.Errorf("TargetSchema = %q, want public", fkd.TargetSchema)
	}
	if fkd.TargetTable != "customers" {
		t.Errorf("TargetTable = %q, want customers", fkd.TargetTable)
	}
	if fkd.Definition != "FOREIGN KEY (customer_id) REFERENCES customers(id)" {
		t.Errorf("Definition = %q, want FK definition", fkd.Definition)
	}
}
