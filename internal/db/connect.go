package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// BuildConnString assembles a keyword/value connstring from individual parameters.
// pgx handles quoting; no manual escaping needed.
func BuildConnString(host string, port int, user, password, dbname string) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)
}

// ConnectAndPing opens a pgx.Conn with a 10-second timeout and verifies
// the connection is live. Returns the connection or an error.
func ConnectAndPing(ctx context.Context, connStr string) (*pgx.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to database: %w", err)
	}
	return conn, nil
}
