package domain_test

import (
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/serialize/domain"
)

func TestSerialize(t *testing.T) {
	s := &domain.Serializer{}
	def := core.DomainDef{
		ObjectHeader:    core.ObjectHeader{Kind: core.KindDomain, Schema: "public", Name: "email_address"},
		BaseType:        "character varying(255)",
		Nullable:        false,
		Default:         "",
		CheckName:       "email_check",
		CheckDefinition: "CHECK ((VALUE)::text ~~ '%@%'::text)",
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["kind"] != "domain" {
		t.Errorf("kind = %v, want domain", m["kind"])
	}
	if m["schema"] != "public" {
		t.Errorf("schema = %v, want public", m["schema"])
	}
	if m["name"] != "email_address" {
		t.Errorf("name = %v, want email_address", m["name"])
	}
	if m["base_type"] != "character varying(255)" {
		t.Errorf("base_type = %v, want 'character varying(255)'", m["base_type"])
	}
	// nullable=false must still appear (no omitempty)
	if m["nullable"] != false {
		t.Errorf("nullable = %v, want false", m["nullable"])
	}
	if m["check_name"] != "email_check" {
		t.Errorf("check_name = %v, want email_check", m["check_name"])
	}
	if m["check_definition"] != "CHECK ((VALUE)::text ~~ '%@%'::text)" {
		t.Errorf("check_definition = %v, want CHECK expression", m["check_definition"])
	}
}

func TestSerializeNullableTrue(t *testing.T) {
	s := &domain.Serializer{}
	def := core.DomainDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindDomain, Schema: "public", Name: "nullable_int"},
		BaseType:     "integer",
		Nullable:     true,
	}

	data, err := s.Serialize(def)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	if m["nullable"] != true {
		t.Errorf("nullable = %v, want true", m["nullable"])
	}
}

func TestDeserialize(t *testing.T) {
	s := &domain.Serializer{}
	yamlData := []byte(`kind: domain
schema: public
name: email_address
base_type: "character varying(255)"
nullable: false
check_name: email_check
check_definition: "CHECK ((VALUE)::text ~~ '%@%'::text)"
`)

	obj, err := s.Deserialize(yamlData)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	dd, ok := obj.(core.DomainDef)
	if !ok {
		t.Fatalf("expected core.DomainDef, got %T", obj)
	}

	if dd.BaseType != "character varying(255)" {
		t.Errorf("BaseType = %q, want 'character varying(255)'", dd.BaseType)
	}
	if dd.Nullable != false {
		t.Errorf("Nullable = %v, want false", dd.Nullable)
	}
	if dd.CheckName != "email_check" {
		t.Errorf("CheckName = %q, want email_check", dd.CheckName)
	}
	if dd.CheckDefinition != "CHECK ((VALUE)::text ~~ '%@%'::text)" {
		t.Errorf("CheckDefinition = %q, want CHECK expression", dd.CheckDefinition)
	}
}
