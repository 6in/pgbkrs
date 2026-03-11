package matview_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/matview"
)

func TestGenerateDDL(t *testing.T) {
	g := &matview.DDLGenerator{}

	t.Run("populated", func(t *testing.T) {
		def := &core.MaterializedViewDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindMaterializedView, Schema: "public", Name: "mv_stats"},
			Definition:   "SELECT count(*) FROM orders",
			Owner:        "admin",
			IsPopulated:  true,
		}

		stmts, err := g.GenerateDDL(def)
		if err != nil {
			t.Fatal(err)
		}
		if len(stmts) == 0 {
			t.Fatal("expected at least one DDL statement")
		}

		combined := strings.Join(stmts, "\n")
		if !strings.Contains(combined, "CREATE MATERIALIZED VIEW") {
			t.Error("expected CREATE MATERIALIZED VIEW statement")
		}
		if strings.Contains(combined, "WITH NO DATA") {
			t.Error("populated matview should not contain WITH NO DATA")
		}
	})

	t.Run("not_populated", func(t *testing.T) {
		def := &core.MaterializedViewDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindMaterializedView, Schema: "public", Name: "mv_stats"},
			Definition:   "SELECT count(*) FROM orders",
			Owner:        "admin",
			IsPopulated:  false,
		}

		stmts, err := g.GenerateDDL(def)
		if err != nil {
			t.Fatal(err)
		}
		if len(stmts) == 0 {
			t.Fatal("expected at least one DDL statement")
		}

		combined := strings.Join(stmts, "\n")
		if !strings.Contains(combined, "CREATE MATERIALIZED VIEW") {
			t.Error("expected CREATE MATERIALIZED VIEW statement")
		}
		if !strings.Contains(combined, "WITH NO DATA") {
			t.Error("non-populated matview should contain WITH NO DATA")
		}
	})
}

func TestGenerateDrop(t *testing.T) {
	g := &matview.DDLGenerator{}
	def := &core.MaterializedViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindMaterializedView, Schema: "public", Name: "mv_stats"},
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
		if strings.Contains(s, "DROP MATERIALIZED VIEW IF EXISTS") {
			found = true
		}
	}
	if !found {
		t.Error("expected DROP MATERIALIZED VIEW IF EXISTS statement")
	}
}
