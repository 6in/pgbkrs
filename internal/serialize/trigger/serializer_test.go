package trigger_test

import (
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/serialize/trigger"
)

func TestSerialize(t *testing.T) {
	s := &trigger.Serializer{}
	def := core.TriggerDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTrigger, Schema: "public", Name: "trg_audit"},
		Timing:       "AFTER",
		Events:       []string{"INSERT", "UPDATE"},
		TableName:    "users",
		FunctionName: "audit_func",
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["kind"] != "trigger" {
		t.Errorf("kind = %v, want trigger", m["kind"])
	}
	if m["schema"] != "public" {
		t.Errorf("schema = %v, want public", m["schema"])
	}
	if m["name"] != "trg_audit" {
		t.Errorf("name = %v, want trg_audit", m["name"])
	}
}
