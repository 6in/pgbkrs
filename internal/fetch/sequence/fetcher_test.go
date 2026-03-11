package sequence_test

import (
	"context"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
	"github.com/pgbkrs/pgbackup/internal/fetch/sequence"
)

func TestFetchSequence(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &sequence.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Find test_seq in returned defs
	var found *core.SequenceDef
	for _, def := range defs {
		if sd, ok := def.(core.SequenceDef); ok && sd.Name == "test_seq" {
			copy := sd
			found = &copy
			break
		}
	}
	if found == nil {
		t.Fatalf("test_seq not found in %d defs", len(defs))
	}

	if found.StartValue != 1 {
		t.Errorf("StartValue: got %d, want 1", found.StartValue)
	}
	if found.MinValue != 1 {
		t.Errorf("MinValue: got %d, want 1", found.MinValue)
	}
	if found.MaxValue != 9223372036854775807 {
		t.Errorf("MaxValue: got %d, want 9223372036854775807", found.MaxValue)
	}
	if found.IncrementBy != 1 {
		t.Errorf("IncrementBy: got %d, want 1", found.IncrementBy)
	}
	if found.Cycle {
		t.Errorf("Cycle: got true, want false")
	}
	if found.Cache != 1 {
		t.Errorf("Cache: got %d, want 1", found.Cache)
	}
	if found.IsCalled {
		t.Errorf("IsCalled: got true, want false (sequence has never been advanced)")
	}
}
