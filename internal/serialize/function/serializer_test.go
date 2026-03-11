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
	if m["definition"] == nil {
		t.Error("definition is missing")
	}
	if m["arg_types"] != "a integer, b integer" {
		t.Errorf("arg_types = %v, want 'a integer, b integer'", m["arg_types"])
	}
	if m["return_type"] != "integer" {
		t.Errorf("return_type = %v, want integer", m["return_type"])
	}
}

func TestDeserialize(t *testing.T) {
	s := &fn.Serializer{}
	yamlData := []byte(`kind: function
schema: public
name: add_numbers
definition: "CREATE OR REPLACE FUNCTION public.add_numbers(a integer, b integer) RETURNS integer LANGUAGE sql AS $$ SELECT a + b $$"
arg_types: "a integer, b integer"
return_type: integer
`)

	obj, err := s.Deserialize(yamlData)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	fd, ok := obj.(core.FunctionDef)
	if !ok {
		t.Fatalf("expected core.FunctionDef, got %T", obj)
	}

	if fd.Kind != core.KindFunction {
		t.Errorf("Kind = %q, want function", fd.Kind)
	}
	if fd.ArgTypes != "a integer, b integer" {
		t.Errorf("ArgTypes = %q, want 'a integer, b integer'", fd.ArgTypes)
	}
	if fd.ReturnType != "integer" {
		t.Errorf("ReturnType = %q, want integer", fd.ReturnType)
	}
}
