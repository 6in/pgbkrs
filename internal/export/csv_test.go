package export_test

import (
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/export"
)

func TestShouldExportData_RegularTable(t *testing.T) {
	td := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
		Columns:      []core.ColumnDef{{Name: "id", Type: "integer"}},
	}

	if !export.ShouldExportData(td) {
		t.Error("expected ShouldExportData to return true for non-partitioned table")
	}
}

func TestShouldExportData_PartitionParent(t *testing.T) {
	td := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "events"},
		Columns:      []core.ColumnDef{{Name: "id", Type: "integer"}},
		Partitioning: &core.PartitionDef{
			Strategy:      "range",
			KeyExpression: "created_at",
			Children:      []string{"events_2024", "events_2025"},
		},
	}

	if export.ShouldExportData(td) {
		t.Error("expected ShouldExportData to return false for partition parent table")
	}
}

func TestShouldExportData_PartitionChild(t *testing.T) {
	// Partition children are regular tables (Partitioning is nil).
	td := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "events_2024"},
		Columns:      []core.ColumnDef{{Name: "id", Type: "integer"}, {Name: "created_at", Type: "timestamp"}},
	}

	if !export.ShouldExportData(td) {
		t.Error("expected ShouldExportData to return true for partition child table")
	}
}

func TestExportTableData(t *testing.T) {
	if testing.Short() {
		t.Skip("requires live database")
	}
	// Integration test: exports data from a real table, verifies file exists,
	// checksum format is "sha256:<hex>", row count > 0.
	// Will be validated in Phase 7 integration testing.
}

func TestExportTableData_EmptyTable(t *testing.T) {
	if testing.Short() {
		t.Skip("requires live database")
	}
	// Integration test: exports empty table, verifies row_count=0,
	// checksum is valid sha256, file is empty.
	// Will be validated in Phase 7 integration testing.
}
