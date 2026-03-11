package enum_test

import (
	"context"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
)

func TestFetchEnum(t *testing.T) {
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
	t.Log("TestFetchEnum: STUB — implement assertions in Wave 2")
}
