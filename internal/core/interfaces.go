package core

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// SchemaFetcher fetches object definitions from pg_catalog.
type SchemaFetcher interface {
	Fetch(ctx context.Context, conn *pgx.Conn, schema string) ([]ObjectDef, error)
}

// Serializer converts ObjectDef to/from YAML bytes.
type Serializer interface {
	Serialize(def ObjectDef) ([]byte, error)
	Deserialize(data []byte) (ObjectDef, error)
}

// DDLGenerator produces executable DDL strings from an ObjectDef.
type DDLGenerator interface {
	GenerateDDL(def ObjectDef) ([]string, error)
	GenerateDrop(def ObjectDef) ([]string, error)
}
