package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/core"
	"go.yaml.in/yaml/v3"
)

// TestSkipUnsupportedColumns verifies that skipReason correctly identifies all
// unsupported column types including array variants.
func TestSkipUnsupportedColumns(t *testing.T) {
	tests := []struct {
		name         string
		columns      []core.ColumnDef
		wantSkipped  bool
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

// connectTestDB connects to the database specified by TEST_DATABASE_URL.
// It skips the test if the env var is not set.
func connectTestDB(t *testing.T) *pgx.Conn {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("pgx.Connect: %v", err)
	}
	t.Cleanup(func() { conn.Close(ctx) })
	return conn
}

// parseManifest reads the _manifest.yaml from backupRoot and returns it as a
// map[string]interface{} for field assertions.
func parseManifest(t *testing.T, backupRoot string) map[string]interface{} {
	t.Helper()
	manifestPath := filepath.Join(backupRoot, "_manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read _manifest.yaml: %v", err)
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse _manifest.yaml: %v", err)
	}
	return m
}

// findBackupRoot uses filepath.Glob to find the single backup_* subdirectory
// created inside outDir by RunBackup.
func findBackupRoot(t *testing.T, outDir string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(outDir, "backup_*"))
	if err != nil {
		t.Fatalf("glob backup_*: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no backup_* directory found in %s", outDir)
	}
	if len(matches) > 1 {
		t.Fatalf("expected 1 backup_* directory, got %d: %v", len(matches), matches)
	}
	return matches[0]
}

// TestBackupIntegration is an integration test that runs RunBackup (non-snapshot)
// against a live database and verifies the backup directory structure.
func TestBackupIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: skipping in -short mode")
	}
	conn := connectTestDB(t)
	outDir := t.TempDir()
	ctx := context.Background()

	if err := RunBackup(ctx, conn, outDir, false); err != nil {
		t.Fatalf("RunBackup: %v", err)
	}

	backupRoot := findBackupRoot(t, outDir)

	// _manifest.yaml must exist and be valid YAML.
	m := parseManifest(t, backupRoot)

	// Verify mandatory manifest fields are non-empty.
	if v, ok := m["backup_at"]; !ok || v == "" {
		t.Errorf("manifest missing or empty backup_at field; got %v", v)
	}
	if v, ok := m["pg_version"]; !ok || v == "" {
		t.Errorf("manifest missing or empty pg_version field; got %v", v)
	}

}

// TestBackupSnapshotMode is an integration test that runs RunBackup with
// snapshot=true and verifies the manifest records snapshot: true.
func TestBackupSnapshotMode(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: skipping in -short mode")
	}
	conn := connectTestDB(t)
	outDir := t.TempDir()
	ctx := context.Background()

	if err := RunBackup(ctx, conn, outDir, true); err != nil {
		t.Fatalf("RunBackup(snapshot=true): %v", err)
	}

	backupRoot := findBackupRoot(t, outDir)
	m := parseManifest(t, backupRoot)

	// Snapshot field must be present and true.
	v, ok := m["snapshot"]
	if !ok {
		t.Fatalf("manifest missing snapshot field")
	}
	snapBool, ok := v.(bool)
	if !ok {
		t.Fatalf("manifest snapshot field is not bool: %T(%v)", v, v)
	}
	if !snapBool {
		t.Errorf("manifest snapshot = false; want true for snapshot=true run")
	}

	// Verify mandatory manifest fields are non-empty.
	if v, ok := m["backup_at"]; !ok || v == "" {
		t.Errorf("manifest missing or empty backup_at field; got %v", v)
	}
	if v, ok := m["pg_version"]; !ok || v == "" {
		t.Errorf("manifest missing or empty pg_version field; got %v", v)
	}
}

// TestBackupNonSnapshotMode is an integration test that runs RunBackup with
// snapshot=false and verifies the manifest records snapshot: false.
func TestBackupNonSnapshotMode(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: skipping in -short mode")
	}
	conn := connectTestDB(t)
	outDir := t.TempDir()
	ctx := context.Background()

	if err := RunBackup(ctx, conn, outDir, false); err != nil {
		t.Fatalf("RunBackup(snapshot=false): %v", err)
	}

	backupRoot := findBackupRoot(t, outDir)
	m := parseManifest(t, backupRoot)

	// Snapshot field must be present and false.
	v, ok := m["snapshot"]
	if !ok {
		t.Fatalf("manifest missing snapshot field")
	}
	snapBool, ok := v.(bool)
	if !ok {
		t.Fatalf("manifest snapshot field is not bool: %T(%v)", v, v)
	}
	if snapBool {
		t.Errorf("manifest snapshot = true; want false for snapshot=false run")
	}

	// Verify mandatory manifest fields are non-empty.
	if v, ok := m["backup_at"]; !ok || v == "" {
		t.Errorf("manifest missing or empty backup_at field; got %v", v)
	}
	if v, ok := m["pg_version"]; !ok || v == "" {
		t.Errorf("manifest missing or empty pg_version field; got %v", v)
	}
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
