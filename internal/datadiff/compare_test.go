package datadiff

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareTables_ChecksumFastPath(t *testing.T) {
	// Identical checksums → no diff, CSV is never read
	a := &tableEntry{
		Schema:   "public",
		Name:     "users",
		PKCols:   []string{"id"},
		Columns:  []string{"id", "name"},
		CSVPath:  "/does/not/exist.csv", // would fail if read
		Checksum: "sha256:abc123",
	}
	b := &tableEntry{
		Schema:   "public",
		Name:     "users",
		PKCols:   []string{"id"},
		Columns:  []string{"id", "name"},
		CSVPath:  "/does/not/exist.csv",
		Checksum: "sha256:abc123",
	}

	res := compareTables(a, b)
	if res.Added != 0 || res.Removed != 0 || res.Changed != 0 {
		t.Errorf("expected no diff for identical checksums, got added=%d removed=%d changed=%d",
			res.Added, res.Removed, res.Changed)
	}
	if res.Err != nil {
		t.Errorf("unexpected error: %v", res.Err)
	}
}

func TestCompareTables_NoPK(t *testing.T) {
	// Differing checksums but no PK → NoPK flag set, no error
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "data.csv")
	os.WriteFile(csvPath, []byte("1,alice\n"), 0o644)

	a := &tableEntry{Schema: "public", Name: "logs", PKCols: nil, Columns: []string{"id", "msg"}, CSVPath: csvPath, Checksum: "sha256:aaa"}
	b := &tableEntry{Schema: "public", Name: "logs", PKCols: nil, Columns: []string{"id", "msg"}, CSVPath: csvPath, Checksum: "sha256:bbb"}

	res := compareTables(a, b)
	if !res.NoPK {
		t.Error("expected NoPK=true for table without primary key")
	}
	if res.Err != nil {
		t.Errorf("unexpected error: %v", res.Err)
	}
}

func TestCompareTables_AddedRemovedChanged(t *testing.T) {
	dir := t.TempDir()

	// backup A: rows 1,2,3
	csvA := filepath.Join(dir, "a.csv")
	os.WriteFile(csvA, []byte("1,alice\n2,bob\n3,carol\n"), 0o644)

	// backup B: row 2 changed, row 3 removed, row 4 added
	csvB := filepath.Join(dir, "b.csv")
	os.WriteFile(csvB, []byte("1,alice\n2,bobby\n4,dave\n"), 0o644)

	a := &tableEntry{Schema: "public", Name: "users", PKCols: []string{"id"}, Columns: []string{"id", "name"}, CSVPath: csvA, Checksum: "sha256:aaa"}
	b := &tableEntry{Schema: "public", Name: "users", PKCols: []string{"id"}, Columns: []string{"id", "name"}, CSVPath: csvB, Checksum: "sha256:bbb"}

	res := compareTables(a, b)
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.Added != 1 {
		t.Errorf("expected Added=1, got %d", res.Added)
	}
	if res.Removed != 1 {
		t.Errorf("expected Removed=1, got %d", res.Removed)
	}
	if res.Changed != 1 {
		t.Errorf("expected Changed=1, got %d", res.Changed)
	}
}

func TestCompareTables_CompositePK(t *testing.T) {
	dir := t.TempDir()

	// PK = (tenant_id, user_id)
	csvA := filepath.Join(dir, "a.csv")
	os.WriteFile(csvA, []byte("t1,u1,alice\nt1,u2,bob\n"), 0o644)

	csvB := filepath.Join(dir, "b.csv")
	os.WriteFile(csvB, []byte("t1,u1,alice\nt2,u1,carol\n"), 0o644) // t1/u2 removed, t2/u1 added

	a := &tableEntry{Schema: "public", Name: "memberships", PKCols: []string{"tenant_id", "user_id"}, Columns: []string{"tenant_id", "user_id", "name"}, CSVPath: csvA, Checksum: "sha256:aaa"}
	b := &tableEntry{Schema: "public", Name: "memberships", PKCols: []string{"tenant_id", "user_id"}, Columns: []string{"tenant_id", "user_id", "name"}, CSVPath: csvB, Checksum: "sha256:bbb"}

	res := compareTables(a, b)
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.Added != 1 {
		t.Errorf("expected Added=1, got %d", res.Added)
	}
	if res.Removed != 1 {
		t.Errorf("expected Removed=1, got %d", res.Removed)
	}
	if res.Changed != 0 {
		t.Errorf("expected Changed=0, got %d", res.Changed)
	}
}

func TestCompareTables_NoDiff(t *testing.T) {
	dir := t.TempDir()
	csv := filepath.Join(dir, "data.csv")
	os.WriteFile(csv, []byte("1,alice\n2,bob\n"), 0o644)

	// Different checksums (simulating unknown) but identical data
	a := &tableEntry{Schema: "public", Name: "users", PKCols: []string{"id"}, Columns: []string{"id", "name"}, CSVPath: csv, Checksum: "sha256:aaa"}
	b := &tableEntry{Schema: "public", Name: "users", PKCols: []string{"id"}, Columns: []string{"id", "name"}, CSVPath: csv, Checksum: "sha256:bbb"}

	res := compareTables(a, b)
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.Added != 0 || res.Removed != 0 || res.Changed != 0 {
		t.Errorf("expected no diff, got added=%d removed=%d changed=%d", res.Added, res.Removed, res.Changed)
	}
}

func TestLoadCSVByPK_UnknownPKColumn(t *testing.T) {
	dir := t.TempDir()
	csv := filepath.Join(dir, "data.csv")
	os.WriteFile(csv, []byte("1,alice\n"), 0o644)

	te := &tableEntry{
		Columns: []string{"id", "name"},
		PKCols:  []string{"nonexistent"},
		CSVPath: csv,
	}

	_, err := loadCSVByPK(te)
	if err == nil {
		t.Error("expected error for unknown PK column, got nil")
	}
}
