package testhelpers

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

// ConnectTestDB opens a pgx connection from TEST_DATABASE_URL or skips the test.
func ConnectTestDB(t *testing.T) (*pgx.Conn, func()) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run integration tests")
	}
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return conn, func() { conn.Close(context.Background()) }
}

// SetupTestSchema creates a temporary schema with fixtures for all fetcher tests.
// Returns the schema name and a teardown func that drops the schema.
// Creates:
//   - Table: simple_table (id serial pk, email text unique not null, price numeric check >0)
//   - Table: partitioned_table (id int, created_at date) PARTITION BY RANGE (created_at)
//   - Table: partitioned_table_2024 as partition child
//   - Table: rls_table with ROW LEVEL SECURITY enabled
//   - Sequence: test_seq
//   - Composite type: address_type (street text, city text, zip text)
//   - Domain: positive_int over integer with CHECK (VALUE > 0)
//   - ENUM: status_enum ('active', 'inactive', 'pending')
func SetupTestSchema(t *testing.T, conn *pgx.Conn) (schema string, teardown func()) {
	t.Helper()
	schema = fmt.Sprintf("pgbkrs_test_%d", os.Getpid())
	ctx := context.Background()

	ddl := fmt.Sprintf(`
        CREATE SCHEMA %s;

        CREATE TABLE %s.simple_table (
            id    serial PRIMARY KEY,
            email text   NOT NULL UNIQUE,
            price numeric CHECK (price > 0)
        );

        CREATE INDEX idx_simple_email ON %s.simple_table (email);

        CREATE TABLE %s.rls_table (
            id   serial PRIMARY KEY,
            data text
        );
        ALTER TABLE %s.rls_table ENABLE ROW LEVEL SECURITY;

        CREATE TABLE %s.partitioned_table (
            id         int,
            created_at date
        ) PARTITION BY RANGE (created_at);

        CREATE TABLE %s.partitioned_table_2024
            PARTITION OF %s.partitioned_table
            FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');

        CREATE SEQUENCE %s.test_seq
            START 1 MINVALUE 1 MAXVALUE 9223372036854775807
            INCREMENT BY 1 NO CYCLE CACHE 1;

        CREATE TYPE %s.address_type AS (
            street text,
            city   text,
            zip    text
        );

        CREATE DOMAIN %s.positive_int AS integer
            NOT NULL
            CHECK (VALUE > 0);

        CREATE TYPE %s.status_enum AS ENUM ('active', 'inactive', 'pending');
    `, schema,
		schema, schema,
		schema, schema,
		schema, schema, schema,
		schema,
		schema,
		schema,
		schema)

	if _, err := conn.Exec(ctx, ddl); err != nil {
		t.Fatalf("SetupTestSchema: %v", err)
	}

	return schema, func() {
		conn.Exec(context.Background(), fmt.Sprintf("DROP SCHEMA %s CASCADE", schema))
	}
}
