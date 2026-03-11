package function_test

import (
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
	fn "github.com/pgbkrs/pgbackup/internal/serialize/function"
)

func TestSerialize(t *testing.T) {
	s := &fn.Serializer{}
	def := core.FunctionDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "add_numbers"},
		Definition:   "CREATE OR REPLACE FUNCTION public.add_numbers(a integer, b integer) RETURNS integer LANGUAGE sql AS $$ SELECT a + b $$",
		ArgTypes:     "a integer, b integer",
		ReturnType:   "integer",
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["kind"] != "function" {
		t.Errorf("kind = %v, want function", m["kind"])
	}
	if m["schema"] != "public" {
		t.Errorf("schema = %v, want public", m["schema"])
	}
	if m["name"] != "add_numbers" {
		t.Errorf("name = %v, want add_numbers", m["name"])
	}
}
