package table_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/table"
)

func TestGenerateDDL_SimpleTableWithPK(t *testing.T) {
	g := &table.DDLGenerator{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Nullable: false, Default: ""},
			{Name: "name", Type: "text", Nullable: true, Default: ""},
		},
		Constraints: core.ConstraintsDef{
			PrimaryKey: &core.PrimaryKeyDef{Name: "users_pkey", Columns: []string{"id"}},
		},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) < 1 {
		t.Fatal("expected at least one DDL statement")
	}

	create := stmts[0]
	if !strings.Contains(create, "CREATE TABLE public.users") {
		t.Errorf("expected CREATE TABLE public.users, got:\n%s", create)
	}
	if !strings.Contains(create, "id integer NOT NULL") {
		t.Errorf("expected 'id integer NOT NULL' in:\n%s", create)
	}
	if !strings.Contains(create, "name text") {
		t.Errorf("expected 'name text' in:\n%s", create)
	}
	if !strings.Contains(create, "CONSTRAINT users_pkey PRIMARY KEY (id)") {
		t.Errorf("expected PK constraint in:\n%s", create)
	}
}

func TestGenerateDDL_UniqueAndCheckConstraints(t *testing.T) {
	g := &table.DDLGenerator{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "products"},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Nullable: false},
			{Name: "sku", Type: "text", Nullable: false},
			{Name: "price", Type: "numeric", Nullable: false},
		},
		Constraints: core.ConstraintsDef{
			PrimaryKey: &core.PrimaryKeyDef{Name: "products_pkey", Columns: []string{"id"}},
			Unique:     []core.UniqueConstraintDef{{Name: "products_sku_key", Definition: "UNIQUE (sku)"}},
			Check:      []core.CheckConstraintDef{{Name: "products_price_check", Definition: "CHECK (price > 0)"}},
		},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	create := stmts[0]
	if !strings.Contains(create, "CONSTRAINT products_sku_key UNIQUE (sku)") {
		t.Errorf("expected unique constraint in:\n%s", create)
	}
	if !strings.Contains(create, "CONSTRAINT products_price_check CHECK (price > 0)") {
		t.Errorf("expected check constraint in:\n%s", create)
	}
}

func TestGenerateDDL_IndexesAsSeparateStatements(t *testing.T) {
	g := &table.DDLGenerator{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Nullable: false},
			{Name: "name", Type: "text", Nullable: true},
		},
		Constraints: core.ConstraintsDef{
			PrimaryKey: &core.PrimaryKeyDef{Name: "users_pkey", Columns: []string{"id"}},
		},
		Indexes: []core.IndexDef{
			{Name: "idx_users_name", Method: "btree", Definition: "CREATE INDEX idx_users_name ON public.users USING btree (name)"},
			{Name: "idx_users_id", Method: "btree", Definition: "CREATE INDEX idx_users_id ON public.users USING btree (id)"},
		},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	// Expect: CREATE TABLE + 2 CREATE INDEX = 3 statements
	if len(stmts) != 3 {
		t.Fatalf("expected 3 statements, got %d: %v", len(stmts), stmts)
	}
	if !strings.Contains(stmts[1], "CREATE INDEX idx_users_name") {
		t.Errorf("expected first index statement, got: %s", stmts[1])
	}
	if !strings.Contains(stmts[2], "CREATE INDEX idx_users_id") {
		t.Errorf("expected second index statement, got: %s", stmts[2])
	}
}

func TestGenerateDDL_PartitionedTable(t *testing.T) {
	g := &table.DDLGenerator{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "events"},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Nullable: false},
			{Name: "created_at", Type: "timestamp", Nullable: false},
		},
		Constraints: core.ConstraintsDef{},
		Partitioning: &core.PartitionDef{
			Strategy:      "range",
			KeyExpression: "RANGE (created_at)",
			Children:      []string{"events_2024", "events_2025"},
		},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	create := stmts[0]
	if !strings.Contains(create, "PARTITION BY RANGE (created_at)") {
		t.Errorf("expected PARTITION BY RANGE (created_at) in:\n%s", create)
	}
}

func TestGenerateDDL_NotNullAndDefault(t *testing.T) {
	g := &table.DDLGenerator{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "accounts"},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Nullable: false, Default: "nextval('accounts_id_seq'::regclass)"},
			{Name: "email", Type: "text", Nullable: false, Default: ""},
			{Name: "bio", Type: "text", Nullable: true, Default: "''::text"},
		},
		Constraints: core.ConstraintsDef{},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	create := stmts[0]
	// id: NOT NULL + DEFAULT
	if !strings.Contains(create, "id integer NOT NULL DEFAULT nextval('accounts_id_seq'::regclass)") {
		t.Errorf("expected id with NOT NULL DEFAULT, got:\n%s", create)
	}
	// email: NOT NULL, no default
	if !strings.Contains(create, "email text NOT NULL") {
		t.Errorf("expected email NOT NULL, got:\n%s", create)
	}
	// bio: nullable with default (no NOT NULL)
	if strings.Contains(create, "bio text NOT NULL") {
		t.Error("bio should be nullable (no NOT NULL)")
	}
	if !strings.Contains(create, "bio text DEFAULT ''::text") {
		t.Errorf("expected bio with DEFAULT, got:\n%s", create)
	}
}

func TestGenerateDDL_NoForeignKeyInOutput(t *testing.T) {
	g := &table.DDLGenerator{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "orders"},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Nullable: false},
			{Name: "customer_id", Type: "integer", Nullable: false},
		},
		Constraints: core.ConstraintsDef{
			PrimaryKey: &core.PrimaryKeyDef{Name: "orders_pkey", Columns: []string{"id"}},
		},
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range stmts {
		if strings.Contains(s, "FOREIGN KEY") {
			t.Error("CREATE TABLE should not contain FOREIGN KEY")
		}
		if strings.Contains(s, "REFERENCES") {
			t.Error("CREATE TABLE should not contain REFERENCES")
		}
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &table.DDLGenerator{}
	def := &core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "myschema", Name: "orders"},
	}

	stmts, err := g.GenerateDrop(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	expected := "DROP TABLE IF EXISTS myschema.orders CASCADE"
	if stmts[0] != expected {
		t.Errorf("expected %q, got %q", expected, stmts[0])
	}
}

func TestGenerateDDL_WrongType(t *testing.T) {
	g := &table.DDLGenerator{}
	def := &core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "seq1"},
	}

	_, err := g.GenerateDDL(def)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
	if !strings.Contains(err.Error(), "TableDef") {
		t.Errorf("expected error mentioning TableDef, got: %s", err)
	}
}

func TestGenerateDrop_WrongType(t *testing.T) {
	g := &table.DDLGenerator{}
	def := &core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "seq1"},
	}

	_, err := g.GenerateDrop(def)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
	if !strings.Contains(err.Error(), "TableDef") {
		t.Errorf("expected error mentioning TableDef, got: %s", err)
	}
}
