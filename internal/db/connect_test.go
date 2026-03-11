package db_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/db"
)

func TestBuildConnString(t *testing.T) {
	result := db.BuildConnString("localhost", 5432, "u", "p", "d")
	if !strings.Contains(result, "host=localhost") {
		t.Errorf("expected result to contain 'host=localhost', got: %s", result)
	}
	if !strings.Contains(result, "dbname=d") {
		t.Errorf("expected result to contain 'dbname=d', got: %s", result)
	}
}

func TestConnect(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	conn, err := db.ConnectAndPing(context.Background(), url)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if conn == nil {
		t.Fatal("expected conn to be non-nil")
	}
	conn.Close(context.Background())
}
