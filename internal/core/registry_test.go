package core_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// stubFetcher is a minimal SchemaFetcher implementation for testing.
type stubFetcher struct{}

func (s *stubFetcher) Fetch(_ context.Context, _ *pgx.Conn, _ string) ([]core.ObjectDef, error) {
	return nil, nil
}

// stubSerializer is a minimal Serializer implementation for testing.
type stubSerializer struct{}

func (s *stubSerializer) Serialize(_ core.ObjectDef) ([]byte, error) {
	return nil, nil
}

func (s *stubSerializer) Deserialize(_ []byte) (core.ObjectDef, error) {
	return nil, nil
}

// stubGenerator is a minimal DDLGenerator implementation for testing.
type stubGenerator struct{}

func (s *stubGenerator) GenerateDDL(_ core.ObjectDef) ([]string, error) {
	return nil, nil
}

func (s *stubGenerator) GenerateDrop(_ core.ObjectDef) ([]string, error) {
	return nil, nil
}

// TestCommandRegistry verifies basic registry behavior:
// - NewCommandRegistry returns non-nil
// - After Register, all three resolvers return non-nil for the registered kind
// - Unregistered kinds return nil
func TestCommandRegistry(t *testing.T) {
	r := core.NewCommandRegistry()
	if r == nil {
		t.Fatal("NewCommandRegistry() returned nil")
	}

	f := &stubFetcher{}
	s := &stubSerializer{}
	g := &stubGenerator{}

	r.Register(core.KindTable, f, s, g)

	if got := r.Fetcher(core.KindTable); got == nil {
		t.Error("Fetcher(KindTable) returned nil after Register")
	}
	if got := r.Serializer(core.KindTable); got == nil {
		t.Error("Serializer(KindTable) returned nil after Register")
	}
	if got := r.Generator(core.KindTable); got == nil {
		t.Error("Generator(KindTable) returned nil after Register")
	}

	// Unregistered kind should return nil
	if got := r.Fetcher(core.KindView); got != nil {
		t.Error("Fetcher(KindView) should be nil for unregistered kind")
	}
}

// TestRegistrySingleLine verifies FOUND-04: a single Register() call makes all
// three components (Fetcher, Serializer, Generator) immediately resolvable.
func TestRegistrySingleLine(t *testing.T) {
	r := core.NewCommandRegistry()

	r.Register(core.KindSequence, &stubFetcher{}, &stubSerializer{}, &stubGenerator{})

	if r.Fetcher(core.KindSequence) == nil {
		t.Error("Fetcher(KindSequence) should be non-nil after single Register() call")
	}
	if r.Serializer(core.KindSequence) == nil {
		t.Error("Serializer(KindSequence) should be non-nil after single Register() call")
	}
	if r.Generator(core.KindSequence) == nil {
		t.Error("Generator(KindSequence) should be non-nil after single Register() call")
	}
}
