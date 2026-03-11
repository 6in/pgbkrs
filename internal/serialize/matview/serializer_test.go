package matview_test

import (
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/serialize/matview"
)

func TestSerialize(t *testing.T) {
	s := &matview.Serializer{}
	def := core.MaterializedViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindMaterializedView, Schema: "public", Name: "user_stats"},
		Definition:   "SELECT count(*) FROM users",
		Owner:        "postgres",
		IsPopulated:  true,
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["kind"] != "materialized_view" {
		t.Errorf("kind = %v, want materialized_view", m["kind"])
	}
	if m["schema"] != "public" {
		t.Errorf("schema = %v, want public", m["schema"])
	}
	if m["name"] != "user_stats" {
		t.Errorf("name = %v, want user_stats", m["name"])
	}
	if m["definition"] != "SELECT count(*) FROM users" {
		t.Errorf("definition = %v, want SELECT body", m["definition"])
	}
	if m["owner"] != "postgres" {
		t.Errorf("owner = %v, want postgres", m["owner"])
	}
	if m["is_populated"] != true {
		t.Errorf("is_populated = %v, want true", m["is_populated"])
	}
}

func TestSerializeUnpopulated(t *testing.T) {
	s := &matview.Serializer{}
	def := core.MaterializedViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindMaterializedView, Schema: "public", Name: "empty_mv"},
		Definition:   "SELECT 1",
		Owner:        "postgres",
		IsPopulated:  false,
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	// is_populated=false must still appear (no omitempty)
	if m["is_populated"] != false {
		t.Errorf("is_populated = %v, want false", m["is_populated"])
	}
}

func TestDeserialize(t *testing.T) {
	s := &matview.Serializer{}
	yamlData := []byte(`kind: materialized_view
schema: public
name: user_stats
definition: "SELECT count(*) FROM users"
owner: postgres
is_populated: true
`)

	obj, err := s.Deserialize(yamlData)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	mvd, ok := obj.(core.MaterializedViewDef)
	if !ok {
		t.Fatalf("expected core.MaterializedViewDef, got %T", obj)
	}

	if mvd.Kind != core.KindMaterializedView {
		t.Errorf("Kind = %q, want materialized_view", mvd.Kind)
	}
	if mvd.IsPopulated != true {
		t.Errorf("IsPopulated = %v, want true", mvd.IsPopulated)
	}
}
