package restore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/resolve"
	serializeTable "github.com/pgbkrs/pgbackup/internal/serialize/table"
)

// TestDefYAMLPath verifies defYAMLPath returns correct paths for regular and partition child tables.
func TestDefYAMLPath(t *testing.T) {
	tests := []struct {
		name      string
		backupDir string
		entry     resolve.RestoreEntry
		want      string
	}{
		{
			name:      "regular table",
			backupDir: "backup",
			entry:     resolve.RestoreEntry{Schema: "pub", Kind: "table", Name: "users"},
			want:      filepath.Join("backup", "pub", "tables", "users", "def.yaml"),
		},
		{
			name:      "partition child table",
			backupDir: "backup",
			entry:     resolve.RestoreEntry{Schema: "pub", Kind: "table", Name: "orders_2024", FromTable: "orders"},
			want:      filepath.Join("backup", "pub", "tables", "orders", "partitions", "orders_2024", "def.yaml"),
		},
		{
			name:      "view",
			backupDir: "backup",
			entry:     resolve.RestoreEntry{Schema: "pub", Kind: "view", Name: "active_users"},
			want:      filepath.Join("backup", "pub", "views", "active_users.yaml"),
		},
		{
			name:      "fk returns foreignkeys path",
			backupDir: "backup",
			entry:     resolve.RestoreEntry{Schema: "pub", Kind: "fk", Name: "fk_order_user"},
			want:      filepath.Join("backup", "pub", "foreignkeys", "fk_order_user.yaml"),
		},
		{
			name:      "sequence",
			backupDir: "/data/bkp",
			entry:     resolve.RestoreEntry{Schema: "public", Kind: "sequence", Name: "users_id_seq"},
			want:      filepath.Join("/data/bkp", "public", "sequences", "users_id_seq.yaml"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := defYAMLPath(tc.backupDir, tc.entry)
			if got != tc.want {
				t.Errorf("defYAMLPath(%q, %+v) = %q; want %q", tc.backupDir, tc.entry, got, tc.want)
			}
		})
	}
}

// TestSerializerFor verifies serializerFor returns non-nil for supported kinds and nil for unsupported.
func TestSerializerFor(t *testing.T) {
	supported := []string{"table", "view", "materialized_view", "function", "sequence", "trigger", "type", "domain", "enum", "policy", "fk", "foreign_key"}
	for _, kind := range supported {
		s := serializerFor(kind)
		if s == nil {
			t.Errorf("serializerFor(%q) = nil; want non-nil", kind)
		}
	}

	unsupported := []string{"unknown"}
	for _, kind := range unsupported {
		s := serializerFor(kind)
		if s != nil {
			t.Errorf("serializerFor(%q) = %T; want nil", kind, s)
		}
	}
}

// TestDDLGeneratorFor verifies ddlGeneratorFor returns non-nil for all 11 supported kinds including fk.
func TestDDLGeneratorFor(t *testing.T) {
	supported := []string{"table", "view", "materialized_view", "function", "sequence", "trigger", "type", "domain", "enum", "policy", "fk"}
	for _, kind := range supported {
		g := ddlGeneratorFor(kind)
		if g == nil {
			t.Errorf("ddlGeneratorFor(%q) = nil; want non-nil", kind)
		}
	}
}

// TestDeserializeRoundTrip verifies that a TableDef can be serialized and deserialized back.
func TestDeserializeRoundTrip(t *testing.T) {
	// Create a temp directory with a def.yaml for a table
	dir := t.TempDir()
	tableDir := filepath.Join(dir, "pub", "tables", "users")
	if err := os.MkdirAll(tableDir, 0755); err != nil {
		t.Fatal(err)
	}

	td := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "pub", Name: "users"},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Nullable: false},
			{Name: "email", Type: "text", Nullable: true},
		},
	}

	s := &serializeTable.Serializer{}
	data, err := s.Serialize(td)
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tableDir, "def.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}

	entry := resolve.RestoreEntry{Schema: "pub", Kind: "table", Name: "users"}
	def, err := loadObjectDef(dir, entry)
	if err != nil {
		t.Fatalf("loadObjectDef: %v", err)
	}

	result, ok := def.(*core.TableDef)
	if !ok {
		t.Fatalf("expected *core.TableDef, got %T", def)
	}
	if result.Name != "users" {
		t.Errorf("Name = %q; want %q", result.Name, "users")
	}
	if len(result.Columns) != 2 {
		t.Errorf("len(Columns) = %d; want 2", len(result.Columns))
	}
}

// TestReadManifestRoundTrip verifies ReadManifest can parse a manifest written by WriteManifest.
func TestReadManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "_manifest.yaml")

	m := &resolve.Manifest{
		BackupAt:    "2026-01-01T00:00:00Z",
		PgVersion:   "16.0",
		ToolVersion: "0.1.0",
		Snapshot:    false,
		RestoreOrder: []resolve.RestoreEntry{
			{Schema: "public", Kind: "table", Name: "users"},
			{Schema: "public", Kind: "sequence", Name: "users_id_seq"},
			{Schema: "public", Kind: "fk", Name: "fk_orders_users", FromTable: "orders"},
		},
	}

	if err := resolve.WriteManifest(manifestPath, m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	got, err := resolve.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	if len(got.RestoreOrder) != len(m.RestoreOrder) {
		t.Fatalf("RestoreOrder len = %d; want %d", len(got.RestoreOrder), len(m.RestoreOrder))
	}
	for i, want := range m.RestoreOrder {
		if got.RestoreOrder[i] != want {
			t.Errorf("RestoreOrder[%d] = %+v; want %+v", i, got.RestoreOrder[i], want)
		}
	}
	if got.PgVersion != m.PgVersion {
		t.Errorf("PgVersion = %q; want %q", got.PgVersion, m.PgVersion)
	}
}
