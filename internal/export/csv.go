package export

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
)

// DataMeta holds metadata about exported table data (CSV file reference in def.yaml).
type DataMeta struct {
	File     string
	Columns  []string
	RowCount int64
	Checksum string
}

// ShouldExportData returns true if the table should have its data exported.
// Partitioned parent tables have no data of their own; only leaf partitions hold rows.
func ShouldExportData(td *core.TableDef) bool {
	return td.Partitioning == nil
}

// ExportTableData streams table data to a CSV file via PostgreSQL COPY TO protocol.
// It computes a SHA256 checksum in a single pass using io.MultiWriter and extracts
// the row count from the COPY CommandTag.
func ExportTableData(ctx context.Context, conn *pgx.Conn, schema, table, filePath string, columns []string) (*DataMeta, error) {
	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("create data file: %w", err)
	}
	defer f.Close()

	hash := sha256.New()
	w := io.MultiWriter(f, hash)

	sql := fmt.Sprintf(`COPY "%s"."%s" TO STDOUT WITH (FORMAT CSV)`, schema, table)
	tag, err := conn.PgConn().CopyTo(ctx, w, sql)
	if err != nil {
		// Attempt to remove partial file on failure.
		f.Close()
		os.Remove(filePath)
		return nil, fmt.Errorf("COPY TO: %w", err)
	}

	// Sync file to disk for data safety.
	if err := f.Sync(); err != nil {
		return nil, fmt.Errorf("sync data file: %w", err)
	}

	checksum := "sha256:" + hex.EncodeToString(hash.Sum(nil))

	return &DataMeta{
		File:     "data.csv",
		Columns:  columns,
		RowCount: tag.RowsAffected(),
		Checksum: checksum,
	}, nil
}
