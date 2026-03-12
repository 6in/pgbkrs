package diff

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// TestLoadBackup verifies that loadBackup returns an error for a non-existent
// directory and an empty (but non-nil) map for a valid backup dir.
func TestLoadBackup(t *testing.T) {
	t.Run("non-existent directory returns error", func(t *testing.T) {
		_, err := loadBackup("/tmp/does-not-exist-pgbkrs-diff-test")
		if err == nil {
			t.Fatal("expected non-nil error for non-existent backup dir, got nil")
		}
	})

	t.Run("empty directory returns empty map", func(t *testing.T) {
		dir := t.TempDir()
		// Create a minimal manifest so loadBackup can parse it.
		manifestContent := "RestoreOrder: []\n"
		if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(manifestContent), 0o644); err != nil {
			t.Fatalf("failed to write manifest: %v", err)
		}
		result, err := loadBackup(dir)
		if err != nil {
			t.Fatalf("unexpected error for valid backup dir: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil map for valid backup dir")
		}
	})
}

// TestDiffAddedRemoved verifies that compareObjects detects objects present in
// only one of the two maps (DIFF-02: Added and Removed detection).
func TestDiffAddedRemoved(t *testing.T) {
	headerA := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"}
	headerB := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "orders"}

	// Both maps share "public.table.users", A also has "public.table.accounts",
	// B also has "public.table.orders".
	sharedKey := "public.table.users"
	onlyInA := "public.table.accounts"
	onlyInB := "public.table.orders"

	mapA := map[string]core.ObjectDef{
		sharedKey: core.TableDef{ObjectHeader: headerA},
		onlyInA: core.TableDef{ObjectHeader: core.ObjectHeader{
			Kind: core.KindTable, Schema: "public", Name: "accounts",
		}},
	}
	mapB := map[string]core.ObjectDef{
		sharedKey: core.TableDef{ObjectHeader: headerA},
		onlyInB:   core.TableDef{ObjectHeader: headerB},
	}

	result := compareObjects(mapA, mapB)

	if len(result.Added) != 1 {
		t.Errorf("expected 1 Added entry, got %d", len(result.Added))
	} else if result.Added[0].Name != "orders" {
		t.Errorf("expected Added entry name 'orders', got %q", result.Added[0].Name)
	}

	if len(result.Removed) != 1 {
		t.Errorf("expected 1 Removed entry, got %d", len(result.Removed))
	} else if result.Removed[0].Name != "accounts" {
		t.Errorf("expected Removed entry name 'accounts', got %q", result.Removed[0].Name)
	}
}

// TestDiffTable verifies that compareObjects detects column changes within a
// TableDef (DIFF-03: table-level structural diff).
func TestDiffTable(t *testing.T) {
	header := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "products"}
	key := "public.table.products"

	tableA := core.TableDef{
		ObjectHeader: header,
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Nullable: false},
		},
	}
	tableB := core.TableDef{
		ObjectHeader: header,
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Nullable: false},
			{Name: "price", Type: "numeric", Nullable: true}, // new column
		},
	}

	mapA := map[string]core.ObjectDef{key: tableA}
	mapB := map[string]core.ObjectDef{key: tableB}

	result := compareObjects(mapA, mapB)

	if len(result.Changed) != 1 {
		t.Fatalf("expected 1 Changed entry, got %d", len(result.Changed))
	}
	change := result.Changed[0]
	if change.Header.Name != "products" {
		t.Errorf("expected Changed entry name 'products', got %q", change.Header.Name)
	}
	// At least one detail line must contain "カラム追加" (column added indicator).
	found := false
	for _, d := range change.Details {
		if strings.Contains(d, "カラム追加") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected at least one Details line containing 'カラム追加', got: %v", change.Details)
	}
}

// TestDiffNonTable verifies that compareObjects detects changes in non-table
// objects (DIFF-04: non-table object diff, e.g. view definition change).
func TestDiffNonTable(t *testing.T) {
	header := core.ObjectHeader{Kind: core.KindView, Schema: "public", Name: "active_users"}
	key := "public.view.active_users"

	viewA := core.ViewDef{ObjectHeader: header, Definition: "SELECT id FROM users WHERE active = true"}
	viewB := core.ViewDef{ObjectHeader: header, Definition: "SELECT id, name FROM users WHERE active = true"}

	mapA := map[string]core.ObjectDef{key: viewA}
	mapB := map[string]core.ObjectDef{key: viewB}

	result := compareObjects(mapA, mapB)

	if len(result.Changed) != 1 {
		t.Fatalf("expected 1 Changed entry for view definition change, got %d", len(result.Changed))
	}
	if result.Changed[0].Header.Name != "active_users" {
		t.Errorf("expected Changed entry name 'active_users', got %q", result.Changed[0].Header.Name)
	}
}

// TestFormatReport verifies that formatReport emits the three required section
// headers in spec 9.3 format (DIFF-05: report formatting).
func TestFormatReport(t *testing.T) {
	result := DiffResult{
		Added: []core.ObjectHeader{
			{Kind: core.KindTable, Schema: "public", Name: "new_table"},
		},
		Removed: []core.ObjectHeader{
			{Kind: core.KindTable, Schema: "public", Name: "old_table"},
		},
		Changed: []ObjectChange{
			{
				Header:  core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
				Details: []string{"カラム追加: email text"},
			},
		},
	}

	var buf bytes.Buffer
	formatReport(result, "backup-2024-01-01", "backup-2024-01-02", &buf)
	output := buf.String()

	if !strings.Contains(output, "[追加オブジェクト]") {
		t.Errorf("expected output to contain '[追加オブジェクト]', got:\n%s", output)
	}
	if !strings.Contains(output, "[削除オブジェクト]") {
		t.Errorf("expected output to contain '[削除オブジェクト]', got:\n%s", output)
	}
	if !strings.Contains(output, "[変更オブジェクト]") {
		t.Errorf("expected output to contain '[変更オブジェクト]', got:\n%s", output)
	}
}
