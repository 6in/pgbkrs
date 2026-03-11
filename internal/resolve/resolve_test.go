package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"go.yaml.in/yaml/v3"
)

func TestFullPipeline_RealisticScenario(t *testing.T) {
	objects := []core.ObjectDef{
		core.SequenceDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "user_id_seq"},
		},
		core.SequenceDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "order_id_seq"},
		},
		core.EnumDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status_type"},
			Labels:       []string{"active", "inactive"},
		},
		core.TableDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
			Columns: []core.ColumnDef{
				{Name: "id", Type: "integer", Default: "nextval('public.user_id_seq'::regclass)"},
				{Name: "name", Type: "text"},
			},
		},
		core.TableDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "orders"},
			Columns: []core.ColumnDef{
				{Name: "id", Type: "integer", Default: "nextval('public.order_id_seq'::regclass)"},
				{Name: "status", Type: "status_type"},
			},
		},
		core.ViewDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindView, Schema: "public", Name: "active_users"},
			Definition:   "SELECT * FROM users WHERE active = true",
		},
		core.FunctionDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "audit_func"},
			Definition:   "CREATE OR REPLACE FUNCTION public.audit_func() RETURNS trigger ...",
			ReturnType:   "trigger",
		},
		core.TriggerDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTrigger, Schema: "public", Name: "audit_trigger"},
			TableName:    "users",
			FunctionName: "audit_func",
			Timing:       "AFTER",
			Events:       []string{"INSERT", "UPDATE"},
		},
		core.PolicyDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "users_policy"},
			TableName:    "users",
			Command:      "SELECT",
			Roles:        []string{"PUBLIC"},
			Using:        "true",
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
	}

	m, err := BuildManifest(objects, params)
	if err != nil {
		t.Fatalf("BuildManifest failed: %v", err)
	}

	// 1. No error (already checked above)

	// 2. Verify ordering constraints in RestoreOrder
	indexOf := func(name string) int {
		for i, e := range m.RestoreOrder {
			if e.Name == name {
				return i
			}
		}
		t.Fatalf("entry %q not found in RestoreOrder", name)
		return -1
	}

	seqUserIdx := indexOf("user_id_seq")
	seqOrderIdx := indexOf("order_id_seq")
	enumIdx := indexOf("status_type")
	usersIdx := indexOf("users")
	ordersIdx := indexOf("orders")
	viewIdx := indexOf("active_users")
	funcIdx := indexOf("audit_func")
	triggerIdx := indexOf("audit_trigger")
	policyIdx := indexOf("users_policy")
	fkIdx := indexOf("orders_user_fk")

	// Sequences before tables
	if seqUserIdx >= usersIdx {
		t.Errorf("user_id_seq (idx=%d) should come before users (idx=%d)", seqUserIdx, usersIdx)
	}
	if seqOrderIdx >= ordersIdx {
		t.Errorf("order_id_seq (idx=%d) should come before orders (idx=%d)", seqOrderIdx, ordersIdx)
	}
	// Enum before tables that use it
	if enumIdx >= ordersIdx {
		t.Errorf("status_type (idx=%d) should come before orders (idx=%d)", enumIdx, ordersIdx)
	}
	// Tables before views
	if usersIdx >= viewIdx {
		t.Errorf("users (idx=%d) should come before active_users (idx=%d)", usersIdx, viewIdx)
	}
	// Function before trigger
	if funcIdx >= triggerIdx {
		t.Errorf("audit_func (idx=%d) should come before audit_trigger (idx=%d)", funcIdx, triggerIdx)
	}
	// Table before trigger and policy
	if usersIdx >= triggerIdx {
		t.Errorf("users (idx=%d) should come before audit_trigger (idx=%d)", usersIdx, triggerIdx)
	}
	if usersIdx >= policyIdx {
		t.Errorf("users (idx=%d) should come before users_policy (idx=%d)", usersIdx, policyIdx)
	}
	// FK at the very end
	if fkIdx != len(m.RestoreOrder)-1 {
		t.Errorf("FK should be last in restore order, idx=%d of %d", fkIdx, len(m.RestoreOrder)-1)
	}

	// 3. Verify objects[].depends_on for orders
	var ordersObj *ObjectEntry
	for i, obj := range m.Objects {
		if obj.ID == "public.orders" {
			ordersObj = &m.Objects[i]
			break
		}
	}
	if ordersObj == nil {
		t.Fatal("public.orders not found in Objects")
	}
	// orders depends on order_id_seq (via nextval) and status_type (via column type)
	depSet := make(map[string]bool)
	for _, d := range ordersObj.DependsOn {
		depSet[d] = true
	}
	if !depSet["public.order_id_seq"] {
		t.Errorf("orders should depend on public.order_id_seq, got: %v", ordersObj.DependsOn)
	}
	if !depSet["public.status_type"] {
		t.Errorf("orders should depend on public.status_type, got: %v", ordersObj.DependsOn)
	}

	// 4. Verify total count
	if len(m.RestoreOrder) != 10 {
		t.Errorf("RestoreOrder count = %d, want 10", len(m.RestoreOrder))
	}

	// 5. Last entry is FK with kind="fk" and from_table="orders"
	last := m.RestoreOrder[len(m.RestoreOrder)-1]
	if last.Kind != "fk" {
		t.Errorf("last entry kind = %q, want 'fk'", last.Kind)
	}
	if last.FromTable != "orders" {
		t.Errorf("last entry from_table = %q, want 'orders'", last.FromTable)
	}
}

func TestFullPipeline_WriteAndRead(t *testing.T) {
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

	params := ManifestParams{
		BackupAt:    "2024-01-01T12:00:00Z",
		PgVersion:   "16.1",
		ToolVersion: "1.0.0",
		Snapshot:    true,
		Skipped: []SkipEntry{
			{Schema: "public", Name: "big_blob", Reason: "contains bytea column: data"},
		},
	}

	m, err := BuildManifest(objects, params)
	if err != nil {
		t.Fatalf("BuildManifest failed: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "_manifest.yaml")
	if err := WriteManifest(path, m); err != nil {
		t.Fatalf("WriteManifest failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// Verify valid YAML
	var m2 Manifest
	if err := yaml.Unmarshal(data, &m2); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	// Round-trip fidelity
	if m2.BackupAt != m.BackupAt {
		t.Errorf("BackupAt mismatch: %q vs %q", m2.BackupAt, m.BackupAt)
	}
	if m2.PgVersion != m.PgVersion {
		t.Errorf("PgVersion mismatch: %q vs %q", m2.PgVersion, m.PgVersion)
	}
	if m2.ToolVersion != m.ToolVersion {
		t.Errorf("ToolVersion mismatch: %q vs %q", m2.ToolVersion, m.ToolVersion)
	}
	if m2.Snapshot != m.Snapshot {
		t.Errorf("Snapshot mismatch: %v vs %v", m2.Snapshot, m.Snapshot)
	}
	if len(m2.Objects) != len(m.Objects) {
		t.Errorf("Objects count mismatch: %d vs %d", len(m2.Objects), len(m.Objects))
	}
	if len(m2.RestoreOrder) != len(m.RestoreOrder) {
		t.Errorf("RestoreOrder count mismatch: %d vs %d", len(m2.RestoreOrder), len(m.RestoreOrder))
	}
	if len(m2.SkippedTables) != 1 {
		t.Errorf("SkippedTables count = %d, want 1", len(m2.SkippedTables))
	}

	// Verify determinism: write again and compare bytes
	path2 := filepath.Join(dir, "_manifest2.yaml")
	if err := WriteManifest(path2, m); err != nil {
		t.Fatalf("WriteManifest second call failed: %v", err)
	}
	data2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatalf("failed to read second file: %v", err)
	}
	if string(data) != string(data2) {
		t.Error("WriteManifest is not deterministic: two calls produced different output")
	}
}

func TestFullPipeline_CycleError(t *testing.T) {
	// Create a cycle: type A has a field of type B, type B has a field of type A
	objects := []core.ObjectDef{
		core.TypeDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "public", Name: "type_a"},
			Fields: []core.CompositeField{
				{Name: "b_ref", Type: "type_b"},
			},
		},
		core.TypeDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "public", Name: "type_b"},
			Fields: []core.CompositeField{
				{Name: "a_ref", Type: "type_a"},
			},
		},
	}

	_, err := BuildManifest(objects, ManifestParams{})
	if err == nil {
		t.Fatal("expected error for circular dependency, got nil")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("error should mention 'circular dependency', got: %v", err)
	}
}
