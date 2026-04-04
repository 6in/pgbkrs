package datadiff

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// TableDiffResult holds the row-level diff summary for a single table.
type TableDiffResult struct {
	Schema  string
	Name    string
	Added   int64
	Removed int64
	Changed int64
	NoPK    bool  // true when the table has no primary key; row-level diff was skipped
	Err     error
}

// compareTables compares the CSV data of a table between two backups.
// It uses checksums as a fast path: identical checksums mean no diff without loading the CSV.
// For tables without a primary key the checksum mismatch is reported but row details are skipped.
func compareTables(a, b *tableEntry) TableDiffResult {
	res := TableDiffResult{Schema: a.Schema, Name: a.Name}

	// Fast path: checksums match → no diff
	if a.Checksum != "" && a.Checksum == b.Checksum {
		return res
	}

	if len(a.PKCols) == 0 {
		res.NoPK = true
		return res
	}

	aRows, err := loadCSVByPK(a)
	if err != nil {
		res.Err = fmt.Errorf("load CSV A (%s.%s): %w", a.Schema, a.Name, err)
		return res
	}
	bRows, err := loadCSVByPK(b)
	if err != nil {
		res.Err = fmt.Errorf("load CSV B (%s.%s): %w", b.Schema, b.Name, err)
		return res
	}

	for key, bRow := range bRows {
		aRow, ok := aRows[key]
		if !ok {
			res.Added++
		} else if joinRow(aRow) != joinRow(bRow) {
			res.Changed++
		}
	}
	for key := range aRows {
		if _, ok := bRows[key]; !ok {
			res.Removed++
		}
	}

	return res
}

// loadCSVByPK reads a CSV file and returns rows indexed by their primary key.
// The key is a null-byte-separated concatenation of PK column values.
func loadCSVByPK(te *tableEntry) (map[string][]string, error) {
	f, err := os.Open(te.CSVPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	colIndex := make(map[string]int, len(te.Columns))
	for i, c := range te.Columns {
		colIndex[c] = i
	}
	pkIndexes := make([]int, 0, len(te.PKCols))
	for _, pk := range te.PKCols {
		idx, ok := colIndex[pk]
		if !ok {
			return nil, fmt.Errorf("PK column %q not found in CSV columns %v", pk, te.Columns)
		}
		pkIndexes = append(pkIndexes, idx)
	}

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	result := make(map[string][]string)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV: %w", err)
		}

		parts := make([]string, len(pkIndexes))
		for i, idx := range pkIndexes {
			if idx < len(record) {
				parts[i] = record[idx]
			}
		}
		key := strings.Join(parts, "\x00")
		result[key] = record
	}

	return result, nil
}

// joinRow joins a CSV record into a comparable string.
func joinRow(row []string) string {
	return strings.Join(row, "\x00")
}
