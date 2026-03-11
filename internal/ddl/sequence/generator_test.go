package sequence_test

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/ddl/sequence"
)

func TestGenerateDDL(t *testing.T) {
	g := &sequence.DDLGenerator{}
	def := &core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "orders_id_seq"},
		StartValue:   1,
		MinValue:     1,
		MaxValue:     9223372036854775807,
		IncrementBy:  1,
		Cycle:        false,
		Cache:        1,
		LastValue:    42,
		IsCalled:     true,
	}

	stmts, err := g.GenerateDDL(def)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) == 0 {
		t.Fatal("expected at least one DDL statement")
	}

	combined := strings.Join(stmts, "\n")
	for _, kw := range []string{"CREATE SEQUENCE", "START", "MINVALUE", "MAXVALUE", "INCREMENT", "CACHE"} {
		if !strings.Contains(combined, kw) {
			t.Errorf("expected %q in CREATE SEQUENCE output", kw)
		}
	}
}

func TestGenerateDrop(t *testing.T) {
	g := &sequence.DDLGenerator{}
	def := &core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "orders_id_seq"},
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
		if strings.Contains(s, "DROP SEQUENCE IF EXISTS") {
			found = true
		}
	}
	if !found {
		t.Error("expected DROP SEQUENCE IF EXISTS statement")
	}
}
