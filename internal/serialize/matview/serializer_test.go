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
}
