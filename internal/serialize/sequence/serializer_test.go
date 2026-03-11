package sequence_test

import (
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/serialize/sequence"
)

func TestSerialize(t *testing.T) {
	s := &sequence.Serializer{}
	def := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "users_id_seq"},
		StartValue:   1,
		MinValue:     1,
		MaxValue:     9223372036854775807,
		IncrementBy:  1,
		Cycle:        false,
		Cache:        1,
		LastValue:    42,
		IsCalled:     true,
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["kind"] != "sequence" {
		t.Errorf("kind = %v, want sequence", m["kind"])
	}
	if m["schema"] != "public" {
		t.Errorf("schema = %v, want public", m["schema"])
	}
	if m["name"] != "users_id_seq" {
		t.Errorf("name = %v, want users_id_seq", m["name"])
	}
	// YAML int64 values come back as int
	if m["start_value"] == nil {
		t.Error("start_value is missing")
	}
	if m["last_value"] == nil {
		t.Error("last_value is missing")
	}
	if m["is_called"] != true {
		t.Errorf("is_called = %v, want true", m["is_called"])
	}
	if m["cycle"] != false {
		t.Errorf("cycle = %v, want false", m["cycle"])
	}
}

func TestSerializeCycleFalse(t *testing.T) {
	s := &sequence.Serializer{}
	def := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "test_seq"},
		Cycle:        false,
		IsCalled:     false,
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	// cycle=false and is_called=false must still appear (no omitempty on bools)
	if _, ok := m["cycle"]; !ok {
		t.Error("cycle field missing from YAML output")
	}
	if _, ok := m["is_called"]; !ok {
		t.Error("is_called field missing from YAML output")
	}
}

func TestDeserialize(t *testing.T) {
	s := &sequence.Serializer{}
	yamlData := []byte(`kind: sequence
schema: public
name: users_id_seq
start_value: 1
min_value: 1
max_value: 9223372036854775807
increment_by: 1
cycle: false
cache: 1
last_value: 42
is_called: true
`)

	obj, err := s.Deserialize(yamlData)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	sd, ok := obj.(core.SequenceDef)
	if !ok {
		t.Fatalf("expected core.SequenceDef, got %T", obj)
	}

	if sd.StartValue != 1 {
		t.Errorf("StartValue = %d, want 1", sd.StartValue)
	}
	if sd.MaxValue != 9223372036854775807 {
		t.Errorf("MaxValue = %d, want max int64", sd.MaxValue)
	}
	if sd.LastValue != 42 {
		t.Errorf("LastValue = %d, want 42", sd.LastValue)
	}
	if sd.IsCalled != true {
		t.Errorf("IsCalled = %v, want true", sd.IsCalled)
	}
	if sd.Cycle != false {
		t.Errorf("Cycle = %v, want false", sd.Cycle)
	}
}
