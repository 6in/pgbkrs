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
		Timing:       "BEFORE",
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
	if m["timing"] != "BEFORE" {
		t.Errorf("timing = %v, want BEFORE", m["timing"])
	}
	events, ok := m["events"].([]interface{})
	if !ok {
		t.Fatalf("events is not a list: %T", m["events"])
	}
	if len(events) != 2 {
		t.Errorf("events length = %d, want 2", len(events))
	}
	if m["table_name"] != "users" {
		t.Errorf("table_name = %v, want users", m["table_name"])
	}
	if m["function_name"] != "audit_func" {
		t.Errorf("function_name = %v, want audit_func", m["function_name"])
	}
}

func TestDeserialize(t *testing.T) {
	s := &trigger.Serializer{}
	yamlData := []byte(`kind: trigger
schema: public
name: trg_audit
timing: BEFORE
events: [INSERT, UPDATE]
table_name: users
function_name: audit_func
`)

	obj, err := s.Deserialize(yamlData)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	td, ok := obj.(core.TriggerDef)
	if !ok {
		t.Fatalf("expected core.TriggerDef, got %T", obj)
	}

	if td.Timing != "BEFORE" {
		t.Errorf("Timing = %q, want BEFORE", td.Timing)
	}
	if len(td.Events) != 2 || td.Events[0] != "INSERT" || td.Events[1] != "UPDATE" {
		t.Errorf("Events = %v, want [INSERT UPDATE]", td.Events)
	}
	if td.TableName != "users" {
		t.Errorf("TableName = %q, want users", td.TableName)
	}
	if td.FunctionName != "audit_func" {
		t.Errorf("FunctionName = %q, want audit_func", td.FunctionName)
	}
}
