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
}
