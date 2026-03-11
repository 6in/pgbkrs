package restore_test

import (
	"os"
	"testing"
)

func requireDB(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	return dsn
}

// TestPreRestoreBackup verifies REST-01: a pre-restore backup is taken before any mutations.
func TestPreRestoreBackup(t *testing.T) {
	requireDB(t)
	t.Fatal("not implemented")
}

// TestRestoreDropOrder verifies REST-02: objects are dropped in reverse restore_order.
func TestRestoreDropOrder(t *testing.T) {
	requireDB(t)
	t.Fatal("not implemented")
}

// TestRestoreCreateOrder verifies REST-04: objects are created in restore_order (tables/types first).
func TestRestoreCreateOrder(t *testing.T) {
	requireDB(t)
	t.Fatal("not implemented")
}

// TestRestoreCopyFrom verifies REST-05: table data loaded via COPY FROM for regular and partition children.
func TestRestoreCopyFrom(t *testing.T) {
	requireDB(t)
	t.Fatal("not implemented")
}

// TestRestoreSetval verifies REST-06: sequences restored to backed-up value using SETVAL.
func TestRestoreSetval(t *testing.T) {
	requireDB(t)
	t.Fatal("not implemented")
}

// TestRestoreIndexes verifies REST-07: indexes created after data load using IndexDef.Definition.
func TestRestoreIndexes(t *testing.T) {
	requireDB(t)
	t.Fatal("not implemented")
}

// TestRestoreFKBatch verifies REST-08: FK constraints applied as single batch after all tables and data.
func TestRestoreFKBatch(t *testing.T) {
	requireDB(t)
	t.Fatal("not implemented")
}

// TestRestoreViewsFunctions verifies REST-09: views, functions, triggers, policies created in manifest order.
func TestRestoreViewsFunctions(t *testing.T) {
	requireDB(t)
	t.Fatal("not implemented")
}
