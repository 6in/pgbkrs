package table_test

import (
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/table"
)

func TestGenerateDDL(t *testing.T) {
	g := &table.DDLGenerator{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Nullable: false, Default: ""},
			{Name: "name", Type: "text", Nullable: true, Default: ""},
		},
		Constraints: core.ConstraintsDef{
			PrimaryKey: &core.PrimaryKeyDef{Name: "users_pkey", Columns: []string{"id"}},
			Unique:     []core.UniqueConstraintDef{{Name: "users_name_key", Definition: "UNIQUE (name)"}},
		},
		Indexes: []core.IndexDef{
			{Name: "idx_users_name", Method: "btree", Definition: "CREATE INDEX idx_users_name ON public.users USING btree (name)"},
		},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) == 0 {
		t.Fatal("expected at least one DDL statement")
	}

	// CREATE TABLE must contain columns and PK but NO FOREIGN KEY
	found := false
	for _, s := range stmts {
		if contains(s, "CREATE TABLE") {
			found = true
			if contains(s, "FOREIGN KEY") {
				t.Error("CREATE TABLE should not contain FOREIGN KEY")
			}
		}
	}
	if !found {
		t.Error("expected CREATE TABLE statement")
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &table.DDLGenerator{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) == 0 {
		t.Fatal("expected at least one DDL statement")
	}

	found := false
	for _, s := range stmts {
		if contains(s, "DROP TABLE IF EXISTS") {
			found = true
		}
	}
	if !found {
		t.Error("expected DROP TABLE IF EXISTS statement")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
