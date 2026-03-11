package export_test

import (
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/export"
)

func TestExportTableData(t *testing.T) {
	t.Skip("requires live database")
}

func TestChecksum(t *testing.T) {
	t.Skip("requires live database")
}

func TestShouldExportData_RegularTable(t *testing.T) {
	td := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
		Columns:      []core.ColumnDef{{Name: "id", Type: "integer"}},
	}

	if !export.ShouldExportData(td) {
		t.Error("expected ShouldExportData to return true for non-partitioned table")
	}
}

func TestShouldExportData_PartitionedTable(t *testing.T) {
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
		t.Error("expected ShouldExportData to return false for partitioned table")
	}
}
