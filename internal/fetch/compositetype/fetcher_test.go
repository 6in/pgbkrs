package compositetype_test

import (
	"context"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
)

func TestFetchCompositeType(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	_ = defs
	t.Log("TestFetchCompositeType: STUB — implement assertions in Wave 2")
}
