package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"go.yaml.in/yaml/v3"
)

func TestManifestStruct(t *testing.T) {
	m := Manifest{
		BackupAt:    "2024-01-01T12:00:00Z",
		PgVersion:   "16.1",
		ToolVersion: "1.0.0",
		Snapshot:    true,
		SkippedTables: []SkipEntry{
			{Schema: "public", Name: "big_data", Reason: "contains bytea column: payload"},
		},
		Objects: []ObjectEntry{
			{ID: "public.users", Kind: "table", DependsOn: []string{}},
		},
		RestoreOrder: []RestoreEntry{
			{Schema: "public", Kind: "table", Name: "users"},
		},
	}
	data, err := yaml.Marshal(&m)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}
	out := string(data)

	// Verify spec 5.1 field names
	for _, key := range []string{"backup_at:", "pg_version:", "tool_version:", "snapshot:", "skipped_tables:", "objects:", "restore_order:"} {
		if !strings.Contains(out, key) {
			t.Errorf("YAML output missing field %q", key)
		}
	}
}

func TestObjectEntry_DependsOn(t *testing.T) {
	// Non-empty depends_on
	obj := ObjectEntry{ID: "public.orders", Kind: "table", DependsOn: []string{"public.users"}}
	data, err := yaml.Marshal(&obj)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, "depends_on:") {
		t.Error("YAML missing depends_on field")
	}
	if !strings.Contains(out, "public.users") {
		t.Error("depends_on should contain public.users")
	}

	// Empty depends_on should serialize as [] not null
	obj2 := ObjectEntry{ID: "public.users", Kind: "table", DependsOn: []string{}}
	data2, err := yaml.Marshal(&obj2)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}
	out2 := string(data2)
	if strings.Contains(out2, "null") {
		t.Error("empty depends_on should not serialize as null")
	}
	if !strings.Contains(out2, "depends_on: []") {
		t.Errorf("empty depends_on should serialize as 'depends_on: []', got: %s", out2)
	}
}

func TestRestoreEntry_FK(t *testing.T) {
	// FK entry should include from_table
	fk := RestoreEntry{Schema: "public", Kind: "fk", Name: "orders_user_fk", FromTable: "orders"}
	data, err := yaml.Marshal(&fk)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, "from_table: orders") {
		t.Errorf("FK RestoreEntry should include from_table, got: %s", out)
	}

	// Non-FK entry should omit from_table
	tbl := RestoreEntry{Schema: "public", Kind: "table", Name: "users"}
	data2, err := yaml.Marshal(&tbl)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}
	out2 := string(data2)
	if strings.Contains(out2, "from_table") {
		t.Errorf("non-FK RestoreEntry should omit from_table, got: %s", out2)
	}
}

func TestSkipEntry(t *testing.T) {
	s := SkipEntry{Schema: "public", Name: "binary_data", Reason: "contains bytea column: payload"}
	data, err := yaml.Marshal(&s)
	if err != nil {
		t.Fatalf("yaml.Marshal failed: %v", err)
	}
	out := string(data)
	for _, key := range []string{"schema:", "name:", "reason:"} {
		if !strings.Contains(out, key) {
			t.Errorf("SkipEntry YAML missing field %q", key)
		}
	}
}

func TestBuildManifest_Basic(t *testing.T) {
	objects := []core.ObjectDef{
		core.SequenceDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "user_id_seq"},
		},
		core.TableDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
			Columns: []core.ColumnDef{
				{Name: "id", Type: "integer", Default: "nextval('public.user_id_seq'::regclass)"},
			},
		},
		core.TableDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "orders"},
			Columns: []core.ColumnDef{
				{Name: "id", Type: "integer"},
			},
		},
		core.ForeignKeyDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "orders_user_fk"},
			SourceTable:  "orders",
			TargetSchema: "public",
			TargetTable:  "users",
			Definition:   "FOREIGN KEY (user_id) REFERENCES users(id)",
		},
	}

	params := ManifestParams{
		BackupAt:    "2024-01-01T12:00:00Z",
		PgVersion:   "16.1",
		ToolVersion: "1.0.0",
		Snapshot:    true,
		Skipped: []SkipEntry{
			{Schema: "public", Name: "big_data", Reason: "contains bytea"},
		},
	}

	m, err := BuildManifest(objects, params)
	if err != nil {
		t.Fatalf("BuildManifest failed: %v", err)
	}

	if m.BackupAt != "2024-01-01T12:00:00Z" {
		t.Errorf("BackupAt = %q, want 2024-01-01T12:00:00Z", m.BackupAt)
	}
	if m.PgVersion != "16.1" {
		t.Errorf("PgVersion = %q, want 16.1", m.PgVersion)
	}
	if m.ToolVersion != "1.0.0" {
		t.Errorf("ToolVersion = %q, want 1.0.0", m.ToolVersion)
	}
	if !m.Snapshot {
		t.Error("Snapshot should be true")
	}
	if len(m.SkippedTables) != 1 {
		t.Errorf("SkippedTables count = %d, want 1", len(m.SkippedTables))
	}
	// 3 non-FK objects + 1 FK = 4 objects
	if len(m.Objects) != 4 {
		t.Errorf("Objects count = %d, want 4", len(m.Objects))
	}
	// 3 non-FK + 1 FK = 4 restore order entries
	if len(m.RestoreOrder) != 4 {
		t.Errorf("RestoreOrder count = %d, want 4", len(m.RestoreOrder))
	}
}

func TestBuildManifest_ObjectIDs(t *testing.T) {
	objects := []core.ObjectDef{
		core.TableDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
		},
		core.SequenceDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "user_id_seq"},
		},
	}

	m, err := BuildManifest(objects, ManifestParams{})
	if err != nil {
		t.Fatalf("BuildManifest failed: %v", err)
	}

	for _, obj := range m.Objects {
		// Object IDs should be "schema.name" format, not "schema.kind.name"
		parts := strings.Split(obj.ID, ".")
		if len(parts) != 2 {
			t.Errorf("Object ID %q should be schema.name format (2 parts), got %d parts", obj.ID, len(parts))
		}
	}
}

func TestBuildManifest_RestoreOrderMatchesKahn(t *testing.T) {
	objects := []core.ObjectDef{
		core.SequenceDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "user_id_seq"},
		},
		core.TableDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
			Columns: []core.ColumnDef{
				{Name: "id", Type: "integer", Default: "nextval('public.user_id_seq'::regclass)"},
			},
		},
		core.ForeignKeyDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "orders_user_fk"},
			SourceTable:  "orders",
			TargetSchema: "public",
			TargetTable:  "users",
		},
	}

	m, err := BuildManifest(objects, ManifestParams{})
	if err != nil {
		t.Fatalf("BuildManifest failed: %v", err)
	}

	// Sequence should come before table
	seqIdx := -1
	tblIdx := -1
	fkIdx := -1
	for i, entry := range m.RestoreOrder {
		switch {
		case entry.Name == "user_id_seq":
			seqIdx = i
		case entry.Name == "users":
			tblIdx = i
		case entry.Name == "orders_user_fk":
			fkIdx = i
		}
	}

	if seqIdx >= tblIdx {
		t.Errorf("sequence (idx=%d) should come before table (idx=%d)", seqIdx, tblIdx)
	}
	if fkIdx < tblIdx {
		t.Errorf("FK (idx=%d) should come after table (idx=%d)", fkIdx, tblIdx)
	}
	// FK should be last
	if fkIdx != len(m.RestoreOrder)-1 {
		t.Errorf("FK should be last in restore order, idx=%d of %d", fkIdx, len(m.RestoreOrder)-1)
	}
}

func TestBuildManifest_DependsOnFromDAG(t *testing.T) {
	objects := []core.ObjectDef{
		core.SequenceDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "user_id_seq"},
		},
		core.TableDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
			Columns: []core.ColumnDef{
				{Name: "id", Type: "integer", Default: "nextval('public.user_id_seq'::regclass)"},
			},
		},
	}

	m, err := BuildManifest(objects, ManifestParams{})
	if err != nil {
		t.Fatalf("BuildManifest failed: %v", err)
	}

	// Find the users object
	var usersObj *ObjectEntry
	for i, obj := range m.Objects {
		if obj.ID == "public.users" {
			usersObj = &m.Objects[i]
			break
		}
	}
	if usersObj == nil {
		t.Fatal("public.users not found in Objects")
	}

	// users depends on user_id_seq via nextval
	found := false
	for _, dep := range usersObj.DependsOn {
		if dep == "public.user_id_seq" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("users.DependsOn should contain public.user_id_seq, got: %v", usersObj.DependsOn)
	}

	// Sequence should have empty depends_on
	var seqObj *ObjectEntry
	for i, obj := range m.Objects {
		if obj.ID == "public.user_id_seq" {
			seqObj = &m.Objects[i]
			break
		}
	}
	if seqObj == nil {
		t.Fatal("public.user_id_seq not found in Objects")
	}
	if len(seqObj.DependsOn) != 0 {
		t.Errorf("user_id_seq.DependsOn should be empty, got: %v", seqObj.DependsOn)
	}
}

func TestBuildManifest_SkippedTables(t *testing.T) {
	objects := []core.ObjectDef{
		core.TableDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
		},
	}

	skipped := []SkipEntry{
		{Schema: "public", Name: "binary_data", Reason: "contains bytea column: payload"},
		{Schema: "audit", Name: "logs", Reason: "too large"},
	}

	m, err := BuildManifest(objects, ManifestParams{Skipped: skipped})
	if err != nil {
		t.Fatalf("BuildManifest failed: %v", err)
	}

	if len(m.SkippedTables) != 2 {
		t.Errorf("SkippedTables count = %d, want 2", len(m.SkippedTables))
	}
	if m.SkippedTables[0].Schema != "public" || m.SkippedTables[0].Name != "binary_data" {
		t.Errorf("first skipped table unexpected: %+v", m.SkippedTables[0])
	}
}

func TestBuildManifest_FKRestoreEntry(t *testing.T) {
	objects := []core.ObjectDef{
		core.TableDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
		},
		core.TableDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "orders"},
		},
		core.ForeignKeyDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "orders_user_fk"},
			SourceTable:  "orders",
			TargetSchema: "public",
			TargetTable:  "users",
			Definition:   "FOREIGN KEY (user_id) REFERENCES users(id)",
		},
	}

	m, err := BuildManifest(objects, ManifestParams{})
	if err != nil {
		t.Fatalf("BuildManifest failed: %v", err)
	}

	// Find FK in restore order
	var fkEntry *RestoreEntry
	for i, entry := range m.RestoreOrder {
		if entry.Name == "orders_user_fk" {
			fkEntry = &m.RestoreOrder[i]
			break
		}
	}
	if fkEntry == nil {
		t.Fatal("FK not found in RestoreOrder")
	}
	if fkEntry.Kind != "fk" {
		t.Errorf("FK kind = %q, want 'fk'", fkEntry.Kind)
	}
	if fkEntry.FromTable != "orders" {
		t.Errorf("FK from_table = %q, want 'orders'", fkEntry.FromTable)
	}
}

func TestWriteManifest(t *testing.T) {
	m := &Manifest{
		BackupAt:    "2024-01-01T12:00:00Z",
		PgVersion:   "16.1",
		ToolVersion: "1.0.0",
		Snapshot:    true,
		Objects: []ObjectEntry{
			{ID: "public.users", Kind: "table", DependsOn: []string{}},
		},
		RestoreOrder: []RestoreEntry{
			{Schema: "public", Kind: "table", Name: "users"},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "_manifest.yaml")

	err := WriteManifest(path, m)
	if err != nil {
		t.Fatalf("WriteManifest failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "backup_at:") {
		t.Error("written file missing backup_at field")
	}

	// Verify round-trip
	var m2 Manifest
	if err := yaml.Unmarshal(data, &m2); err != nil {
		t.Fatalf("failed to unmarshal written YAML: %v", err)
	}
	if m2.BackupAt != m.BackupAt {
		t.Errorf("round-trip BackupAt mismatch: %q vs %q", m2.BackupAt, m.BackupAt)
	}
}
