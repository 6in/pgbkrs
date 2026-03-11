package core

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// stubSchemaFetcher satisfies SchemaFetcher for compile-time verification.
type stubSchemaFetcher struct{}

func (s *stubSchemaFetcher) Fetch(_ context.Context, _ *pgx.Conn, _ string) ([]ObjectDef, error) {
	return nil, nil
}

// stubSerializer satisfies Serializer for compile-time verification.
type stubSerializer struct{}

func (s *stubSerializer) Serialize(_ ObjectDef) ([]byte, error)   { return nil, nil }
func (s *stubSerializer) Deserialize(_ []byte) (ObjectDef, error) { return nil, nil }

// stubDDLGenerator satisfies DDLGenerator for compile-time verification.
type stubDDLGenerator struct{}

func (s *stubDDLGenerator) GenerateDDL(_ ObjectDef) ([]string, error)  { return nil, nil }
func (s *stubDDLGenerator) GenerateDrop(_ ObjectDef) ([]string, error) { return nil, nil }

// Compile-time interface compliance assertions.
// If any of these fail to compile, the corresponding interface is not satisfied.
var _ SchemaFetcher = (*stubSchemaFetcher)(nil)
var _ Serializer = (*stubSerializer)(nil)
var _ DDLGenerator = (*stubDDLGenerator)(nil)
var _ ObjectDef = TableDef{}

// TestInterfaceCompliance documents that compile-time assertions above are in place.
// If this test runs, it means all stubs satisfy their respective interfaces.
func TestInterfaceCompliance(t *testing.T) {
	// The compile-time var _ assertions above are the real test.
	// This function body is intentionally minimal — the proof is at compile time.
	t.Log("All interface compliance assertions passed at compile time")
}

// TestObjectDefHeader verifies that TableDef.Header() returns the embedded ObjectHeader.
func TestObjectDefHeader(t *testing.T) {
	header := ObjectHeader{
		Kind:   KindTable,
		Schema: "public",
		Name:   "users",
	}
	def := TableDef{ObjectHeader: header}

	got := def.Header()
	if got.Kind != KindTable {
		t.Errorf("Header().Kind = %q, want %q", got.Kind, KindTable)
	}
	if got.Schema != "public" {
		t.Errorf("Header().Schema = %q, want %q", got.Schema, "public")
	}
	if got.Name != "users" {
		t.Errorf("Header().Name = %q, want %q", got.Name, "users")
	}
}
