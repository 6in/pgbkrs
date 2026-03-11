package table_test

import (
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/serialize/table"
)

func TestSerialize(t *testing.T) {
	s := &table.Serializer{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "bigint", Nullable: false, Default: "nextval('users_id_seq')"},
			{Name: "email", Type: "character varying(255)", Nullable: false},
		},
		Constraints: core.ConstraintsDef{
			PrimaryKey: &core.PrimaryKeyDef{Name: "users_pkey", Columns: []string{"id"}},
			Unique:     []core.UniqueConstraintDef{{Name: "users_email_key", Definition: "UNIQUE (email)"}},
		},
		Indexes: []core.IndexDef{
			{Name: "idx_users_created_at", Method: "btree", Definition: "CREATE INDEX idx_users_created_at ON public.users USING btree (created_at)"},
		},
		RLS: core.RLSDef{Enabled: false},
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["kind"] != "table" {
		t.Errorf("kind = %v, want table", m["kind"])
	}
	if m["schema"] != "public" {
		t.Errorf("schema = %v, want public", m["schema"])
	}
	if m["name"] != "users" {
		t.Errorf("name = %v, want users", m["name"])
	}

	cols, ok := m["columns"].([]interface{})
	if !ok || len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %v", m["columns"])
	}

	constraints, ok := m["constraints"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected constraints map, got %T", m["constraints"])
	}
	pk, ok := constraints["primary_key"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected primary_key map, got %T", constraints["primary_key"])
	}
	if pk["name"] != "users_pkey" {
		t.Errorf("pk name = %v, want users_pkey", pk["name"])
	}
}

func TestDeserialize(t *testing.T) {
	s := &table.Serializer{}

	yamlData := []byte(`kind: table
schema: public
name: users
columns:
  - name: id
    type: bigint
    nullable: false
    default: "nextval('users_id_seq')"
  - name: email
    type: "character varying(255)"
    nullable: false
constraints:
  primary_key:
    name: users_pkey
    columns: [id]
  unique:
    - name: users_email_key
      definition: "UNIQUE (email)"
indexes:
  - name: idx_users_created_at
    method: btree
rls:
  enabled: false
`)

	obj, err := s.Deserialize(yamlData)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	td, ok := obj.(*core.TableDef)
	if !ok {
		t.Fatalf("expected *core.TableDef, got %T", obj)
	}

	if td.Schema != "public" {
		t.Errorf("Schema = %q, want public", td.Schema)
	}
	if td.Name != "users" {
		t.Errorf("Name = %q, want users", td.Name)
	}
	if len(td.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(td.Columns))
	}
	if td.Columns[0].Name != "id" || td.Columns[0].Type != "bigint" {
		t.Errorf("column[0] = %+v, want id/bigint", td.Columns[0])
	}
	if td.Constraints.PrimaryKey == nil || td.Constraints.PrimaryKey.Name != "users_pkey" {
		t.Errorf("PK = %+v, want users_pkey", td.Constraints.PrimaryKey)
	}
}

func TestIndexDefinitionRoundTrip(t *testing.T) {
	s := &table.Serializer{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "t"},
		Columns:      []core.ColumnDef{{Name: "id", Type: "integer", Nullable: false}},
		Indexes: []core.IndexDef{
			{Name: "foo", Method: "btree", Definition: "CREATE INDEX foo ON t(id)"},
		},
		RLS: core.RLSDef{Enabled: false},
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	obj, err := s.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	td, ok := obj.(*core.TableDef)
	if !ok {
		t.Fatalf("expected *core.TableDef, got %T", obj)
	}
	if len(td.Indexes) != 1 {
		t.Fatalf("expected 1 index, got %d", len(td.Indexes))
	}
	if td.Indexes[0].Definition != "CREATE INDEX foo ON t(id)" {
		t.Errorf("Definition = %q, want 'CREATE INDEX foo ON t(id)'", td.Indexes[0].Definition)
	}
}

func TestSerializeWithPartitioning(t *testing.T) {
	s := &table.Serializer{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "orders"},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Nullable: false},
			{Name: "created_at", Type: "timestamp", Nullable: false},
		},
		Constraints: core.ConstraintsDef{},
		Partitioning: &core.PartitionDef{
			Strategy:      "range",
			KeyExpression: "created_at",
			Children:      []string{"orders_2024", "orders_2025"},
		},
		RLS: core.RLSDef{Enabled: false},
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	part, ok := m["partitioning"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected partitioning map, got %T", m["partitioning"])
	}
	if part["strategy"] != "range" {
		t.Errorf("strategy = %v, want range", part["strategy"])
	}
	if part["key_expression"] != "created_at" {
		t.Errorf("key_expression = %v, want created_at", part["key_expression"])
	}
	children, ok := part["children"].([]interface{})
	if !ok || len(children) != 2 {
		t.Fatalf("expected 2 children, got %v", part["children"])
	}
}
