package table_test

import (
	"context"
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/fetch/table"
	"github.com/pgbkrs/pgbackup/internal/fetch/testhelpers"
)

// findTable locates a core.TableDef by name in the returned ObjectDef slice.
func findTable(t *testing.T, defs []core.ObjectDef, name string) *core.TableDef {
	t.Helper()
	for _, d := range defs {
		if td, ok := d.(*core.TableDef); ok && td.Name == name {
			return td
		}
	}
	t.Fatalf("table %q not found in fetched defs", name)
	return nil
}

func TestFetchTable(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &table.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	st := findTable(t, defs, "simple_table")

	// Find column "id" — type should contain "integer" (serial is stored as integer)
	var colID, colEmail, colPrice *core.ColumnDef
	for i := range st.Columns {
		switch st.Columns[i].Name {
		case "id":
			colID = &st.Columns[i]
		case "email":
			colEmail = &st.Columns[i]
		case "price":
			colPrice = &st.Columns[i]
		}
	}

	if colID == nil {
		t.Fatal("column 'id' not found in simple_table")
	}
	if !strings.Contains(colID.Type, "integer") {
		t.Errorf("id column type %q should contain 'integer'", colID.Type)
	}

	if colEmail == nil {
		t.Fatal("column 'email' not found in simple_table")
	}
	if colEmail.Type != "text" {
		t.Errorf("email column type = %q, want 'text'", colEmail.Type)
	}
	if colEmail.Nullable {
		t.Error("email column should not be nullable (NOT NULL)")
	}

	if colPrice == nil {
		t.Fatal("column 'price' not found in simple_table")
	}
	if colPrice.Type != "numeric" {
		t.Errorf("price column type = %q, want 'numeric'", colPrice.Type)
	}
	if !colPrice.Nullable {
		t.Error("price column should be nullable (no NOT NULL constraint)")
	}
	if colPrice.Default != "" {
		t.Errorf("price column default = %q, want empty string", colPrice.Default)
	}
}

func TestFetchTableConstraints(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &table.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	st := findTable(t, defs, "simple_table")

	// Primary key
	if st.Constraints.PrimaryKey == nil {
		t.Fatal("simple_table should have a primary key")
	}
	if st.Constraints.PrimaryKey.Name == "" {
		t.Error("primary key name should not be empty")
	}
	if len(st.Constraints.PrimaryKey.Columns) != 1 || st.Constraints.PrimaryKey.Columns[0] != "id" {
		t.Errorf("primary key columns = %v, want [id]", st.Constraints.PrimaryKey.Columns)
	}

	// Unique constraint on email
	if len(st.Constraints.Unique) != 1 {
		t.Errorf("expected 1 unique constraint, got %d", len(st.Constraints.Unique))
	} else {
		uc := st.Constraints.Unique[0]
		if uc.Name == "" {
			t.Error("unique constraint name should not be empty")
		}
		if !strings.Contains(strings.ToLower(uc.Definition), "email") {
			t.Errorf("unique constraint definition %q should mention 'email'", uc.Definition)
		}
	}

	// Check constraint on price
	if len(st.Constraints.Check) != 1 {
		t.Errorf("expected 1 check constraint, got %d", len(st.Constraints.Check))
	} else {
		cc := st.Constraints.Check[0]
		if cc.Name == "" {
			t.Error("check constraint name should not be empty")
		}
		if !strings.Contains(strings.ToLower(cc.Definition), "price") {
			t.Errorf("check constraint definition %q should mention 'price'", cc.Definition)
		}
	}
}

func TestFetchTableIndexes(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &table.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	st := findTable(t, defs, "simple_table")

	// simple_table should have exactly one non-constraint index: idx_simple_email
	if len(st.Indexes) != 1 {
		t.Errorf("expected 1 index (idx_simple_email), got %d: %+v", len(st.Indexes), st.Indexes)
		return
	}
	idx := st.Indexes[0]
	if idx.Method != "btree" {
		t.Errorf("index method = %q, want 'btree'", idx.Method)
	}
	if !strings.Contains(idx.Definition, "idx_simple_email") {
		t.Errorf("index definition %q should contain 'idx_simple_email'", idx.Definition)
	}
}

func TestFetchTableRLS(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &table.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	rls := findTable(t, defs, "rls_table")
	if !rls.RLS.Enabled {
		t.Error("rls_table should have RLS.Enabled = true")
	}

	st := findTable(t, defs, "simple_table")
	if st.RLS.Enabled {
		t.Error("simple_table should have RLS.Enabled = false")
	}
}

func TestFetchPartitionedTable(t *testing.T) {
	conn, closeConn := testhelpers.ConnectTestDB(t)
	defer closeConn()
	schema, teardown := testhelpers.SetupTestSchema(t, conn)
	defer teardown()

	f := &table.SchemaFetcher{}
	defs, err := f.Fetch(context.Background(), conn, schema)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	pt := findTable(t, defs, "partitioned_table")

	if pt.Partitioning == nil {
		t.Fatal("partitioned_table should have non-nil Partitioning")
	}
	if pt.Partitioning.Strategy != "range" {
		t.Errorf("partitioned_table strategy = %q, want 'range'", pt.Partitioning.Strategy)
	}
	found := false
	for _, child := range pt.Partitioning.Children {
		if child == "partitioned_table_2024" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("partitioned_table children %v should contain 'partitioned_table_2024'", pt.Partitioning.Children)
	}

	// partitioned_table_2024 is a child — it should have nil Partitioning
	child := findTable(t, defs, "partitioned_table_2024")
	if child.Partitioning != nil {
		t.Errorf("partitioned_table_2024 is a leaf partition, Partitioning should be nil, got %+v", child.Partitioning)
	}
}
