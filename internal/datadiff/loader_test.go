package datadiff

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBackupTables_NonExistentDir(t *testing.T) {
	_, err := loadBackupTables("/tmp/does-not-exist-pgbkrs-datadiff-test")
	if err == nil {
		t.Fatal("expected error for non-existent backup directory, got nil")
	}
}

func TestLoadBackupTables_EmptyManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "_manifest.yaml"), []byte("RestoreOrder: []\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	tables, err := loadBackupTables(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("expected empty map for empty manifest, got %d entries", len(tables))
	}
}

func TestLoadBackupTables_TableWithData(t *testing.T) {
	dir := t.TempDir()

	// Write manifest referencing one table
	manifest := `restore_order:
  - schema: public
    kind: table
    name: users
`
	if err := os.WriteFile(filepath.Join(dir, "_manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// Write def.yaml for the table with DataMeta and a primary key
	tableDir := filepath.Join(dir, "public", "tables", "users")
	if err := os.MkdirAll(tableDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defYAML := `kind: table
schema: public
name: users
columns:
  - name: id
    type: integer
    nullable: false
  - name: email
    type: text
    nullable: false
constraints:
  primary_key:
    name: users_pkey
    columns: [id]
data:
  file: data.csv
  columns: [id, email]
  row_count: 2
  checksum: sha256:deadbeef
`
	if err := os.WriteFile(filepath.Join(tableDir, "def.yaml"), []byte(defYAML), 0o644); err != nil {
		t.Fatalf("write def.yaml: %v", err)
	}

	// Write the CSV file so stat succeeds
	if err := os.WriteFile(filepath.Join(tableDir, "data.csv"), []byte("1,a@example.com\n2,b@example.com\n"), 0o644); err != nil {
		t.Fatalf("write data.csv: %v", err)
	}

	tables, err := loadBackupTables(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry, ok := tables["public.users"]
	if !ok {
		t.Fatal("expected 'public.users' in result map")
	}
	if entry.Schema != "public" || entry.Name != "users" {
		t.Errorf("unexpected schema/name: %s.%s", entry.Schema, entry.Name)
	}
	if len(entry.PKCols) != 1 || entry.PKCols[0] != "id" {
		t.Errorf("expected PKCols=[id], got %v", entry.PKCols)
	}
	if entry.Checksum != "sha256:deadbeef" {
		t.Errorf("expected checksum 'sha256:deadbeef', got %q", entry.Checksum)
	}
}

func TestLoadBackupTables_TableWithoutCSV(t *testing.T) {
	dir := t.TempDir()

	manifest := `restore_order:
  - schema: public
    kind: table
    name: partitioned_parent
`
	if err := os.WriteFile(filepath.Join(dir, "_manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// def.yaml with no data: block (partitioned parent)
	tableDir := filepath.Join(dir, "public", "tables", "partitioned_parent")
	os.MkdirAll(tableDir, 0o755)
	defYAML := `kind: table
schema: public
name: partitioned_parent
columns:
  - name: id
    type: integer
    nullable: false
partitioning:
  strategy: range
  key_expression: id
  children: []
`
	os.WriteFile(filepath.Join(tableDir, "def.yaml"), []byte(defYAML), 0o644)

	tables, err := loadBackupTables(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("expected no tables (partitioned parent has no data.csv), got %d", len(tables))
	}
}
