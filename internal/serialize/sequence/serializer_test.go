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
}
