package backup

import (
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// TestSkipUnsupportedColumns verifies that skipReason correctly identifies all
// unsupported column types including array variants.
func TestSkipUnsupportedColumns(t *testing.T) {
	tests := []struct {
		name        string
		columns     []core.ColumnDef
		wantSkipped bool
		wantContains string // substring expected in reason when wantSkipped=true
	}{
		{
			name: "no problematic columns",
			columns: []core.ColumnDef{
				{Name: "id", Type: "integer"},
				{Name: "name", Type: "text"},
				{Name: "created_at", Type: "timestamp with time zone"},
			},
			wantSkipped: false,
		},
		{
			name: "bytea column",
			columns: []core.ColumnDef{
				{Name: "id", Type: "integer"},
				{Name: "data", Type: "bytea"},
			},
			wantSkipped:  true,
			wantContains: "data",
		},
		{
			name: "bytea array variant",
			columns: []core.ColumnDef{
				{Name: "id", Type: "integer"},
				{Name: "chunks", Type: "bytea[]"},
			},
			wantSkipped:  true,
			wantContains: "chunks",
		},
		{
			name: "xml column",
			columns: []core.ColumnDef{
				{Name: "id", Type: "integer"},
				{Name: "doc", Type: "xml"},
			},
			wantSkipped:  true,
			wantContains: "doc",
		},
		{
			name: "pg_lsn column",
			columns: []core.ColumnDef{
				{Name: "id", Type: "integer"},
				{Name: "lsn", Type: "pg_lsn"},
			},
			wantSkipped:  true,
			wantContains: "lsn",
		},
		{
			name: "txid_snapshot column",
			columns: []core.ColumnDef{
				{Name: "id", Type: "integer"},
				{Name: "snap", Type: "txid_snapshot"},
			},
			wantSkipped:  true,
			wantContains: "snap",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			td := &core.TableDef{
				ObjectHeader: core.ObjectHeader{
					Kind:   core.KindTable,
					Schema: "public",
					Name:   "test_table",
				},
				Columns: tc.columns,
			}

			reason := skipReason(td)
			if tc.wantSkipped {
				if reason == "" {
					t.Errorf("skipReason() returned empty string; want non-empty for %q", tc.name)
				}
				if tc.wantContains != "" && !containsStr(reason, tc.wantContains) {
					t.Errorf("skipReason() = %q; want it to contain %q", reason, tc.wantContains)
				}
			} else {
				if reason != "" {
					t.Errorf("skipReason() = %q; want empty string for %q", reason, tc.name)
				}
			}
		})
	}
}

// TestSkipWarning verifies skipReason returns a descriptive message for a bytea column.
func TestSkipWarning(t *testing.T) {
	td := &core.TableDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.KindTable,
			Schema: "public",
			Name:   "binary_data",
		},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer"},
			{Name: "payload", Type: "bytea"},
		},
	}

	reason := skipReason(td)
	if reason == "" {
		t.Fatal("skipReason() returned empty string for table with bytea column; want non-empty")
	}
	if !containsStr(reason, "payload") {
		t.Errorf("skipReason() = %q; want it to contain the column name 'payload'", reason)
	}
}

// TestOrchestratorDirectoryLayout is an integration test requiring a live database.
func TestOrchestratorDirectoryLayout(t *testing.T) {
	t.Skip("integration: requires live DB")
}

// TestSnapshotMode is an integration test for snapshot backup mode.
func TestSnapshotMode(t *testing.T) {
	t.Skip("integration: requires live DB")
}

// TestNonSnapshotMode is an integration test for non-snapshot backup mode.
func TestNonSnapshotMode(t *testing.T) {
	t.Skip("integration: requires live DB")
}

// containsStr is a helper to check substring presence.
func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
