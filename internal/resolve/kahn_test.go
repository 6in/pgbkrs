package resolve

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
)

func TestTopologicalSort_Simple(t *testing.T) {
	// Linear chain: A depends on B, B depends on C => order [C, B, A]
	d := NewDAG()
	a := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "a"}
	b := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "b"}
	c := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "c"}

	d.AddNode(a)
	d.AddNode(b)
	d.AddNode(c)
	d.AddEdge(NodeID(a), NodeID(b))
	d.AddEdge(NodeID(b), NodeID(c))

	result, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	order := headerNames(result)
	idxC := indexOf(order, "c")
	idxB := indexOf(order, "b")
	idxA := indexOf(order, "a")
	if idxC > idxB || idxB > idxA {
		t.Errorf("expected C before B before A; got order %v", order)
	}
}

func TestTopologicalSort_Diamond(t *testing.T) {
	// A->B, A->C, B->D, C->D => D before B and C, both before A
	d := NewDAG()
	a := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "a"}
	b := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "b"}
	c := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "c"}
	dd := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "d"}

	d.AddNode(a)
	d.AddNode(b)
	d.AddNode(c)
	d.AddNode(dd)
	d.AddEdge(NodeID(a), NodeID(b))
	d.AddEdge(NodeID(a), NodeID(c))
	d.AddEdge(NodeID(b), NodeID(dd))
	d.AddEdge(NodeID(c), NodeID(dd))

	result, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	order := headerNames(result)
	idxD := indexOf(order, "d")
	idxB := indexOf(order, "b")
	idxC := indexOf(order, "c")
	idxA := indexOf(order, "a")

	if idxD > idxB || idxD > idxC {
		t.Errorf("D should come before B and C; got %v", order)
	}
	if idxB > idxA || idxC > idxA {
		t.Errorf("B and C should come before A; got %v", order)
	}
}

func TestTopologicalSort_Deterministic(t *testing.T) {
	// Run multiple times, expect same result
	for i := 0; i < 10; i++ {
		d := NewDAG()
		for _, name := range []string{"z", "a", "m", "b", "x"} {
			d.AddNode(core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: name})
		}
		result, err := d.TopologicalSort()
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}
		order := headerNames(result)
		expected := []string{"a", "b", "m", "x", "z"} // alphabetical, same kind+schema
		for j, name := range order {
			if name != expected[j] {
				t.Fatalf("run %d: expected %v, got %v", i, expected, order)
			}
		}
	}
}

func TestTopologicalSort_TypePriority(t *testing.T) {
	// Multiple kinds with in-degree 0: should sort by kind priority
	d := NewDAG()
	table := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"}
	seq := core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "user_id_seq"}
	view := core.ObjectHeader{Kind: core.KindView, Schema: "public", Name: "active_users"}
	enum := core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status"}

	d.AddNode(table)
	d.AddNode(seq)
	d.AddNode(view)
	d.AddNode(enum)

	result, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	order := headerNames(result)
	// Expected priority: sequence(0) < enum(1) < table(4) < view(5)
	idxSeq := indexOf(order, "user_id_seq")
	idxEnum := indexOf(order, "status")
	idxTable := indexOf(order, "users")
	idxView := indexOf(order, "active_users")

	if idxSeq > idxEnum || idxEnum > idxTable || idxTable > idxView {
		t.Errorf("expected seq < enum < table < view; got %v", order)
	}
}

func TestTopologicalSort_CycleDetection(t *testing.T) {
	// A->B->C->A (circular)
	d := NewDAG()
	a := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "a"}
	b := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "b"}
	c := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "c"}

	d.AddNode(a)
	d.AddNode(b)
	d.AddNode(c)
	d.AddEdge(NodeID(a), NodeID(b))
	d.AddEdge(NodeID(b), NodeID(c))
	d.AddEdge(NodeID(c), NodeID(a))

	_, err := d.TopologicalSort()
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("error should mention circular dependency; got: %v", err)
	}
}

func TestTopologicalSort_AllIsolated(t *testing.T) {
	d := NewDAG()
	seq := core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "s1"}
	table := core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "t1"}
	enum := core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "e1"}

	d.AddNode(seq)
	d.AddNode(table)
	d.AddNode(enum)

	result, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	order := headerNames(result)
	// priority: seq(0), enum(1), table(4)
	if order[0] != "s1" || order[1] != "e1" || order[2] != "t1" {
		t.Errorf("expected [s1 e1 t1], got %v", order)
	}
}

func TestBuildRestoreOrder_FKIsolation(t *testing.T) {
	objects := []core.ObjectDef{
		core.TableDef{ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"}},
		core.TableDef{ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "orders"}},
		core.ForeignKeyDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "orders_user_fk"},
			SourceTable:  "orders", TargetSchema: "public", TargetTable: "users",
			Definition: "FOREIGN KEY (user_id) REFERENCES users(id)",
		},
	}

	result, err := BuildRestoreOrder(objects)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	// FK should be last
	last := result[len(result)-1]
	if last.Kind != core.KindForeignKey {
		t.Errorf("last item should be FK; got %+v", last)
	}
}

func TestBuildRestoreOrder_FKNotInDAG(t *testing.T) {
	objects := []core.ObjectDef{
		core.ForeignKeyDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "some_fk"},
			SourceTable:  "a", TargetSchema: "public", TargetTable: "b",
		},
	}

	result, err := BuildRestoreOrder(objects)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result (FK only), got %d", len(result))
	}
	if result[0].Kind != core.KindForeignKey {
		t.Errorf("expected FK, got %+v", result[0])
	}
}

func TestBuildRestoreOrder_Empty(t *testing.T) {
	result, err := BuildRestoreOrder(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}

func TestBuildRestoreOrder_Integration(t *testing.T) {
	objects := []core.ObjectDef{
		// Sequences
		core.SequenceDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "user_id_seq"},
		},
		core.SequenceDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "order_id_seq"},
		},
		// Enum
		core.EnumDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status_type"},
			Labels:       []string{"active", "inactive"},
		},
		// Tables
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
		// View
		core.ViewDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindView, Schema: "public", Name: "active_users"},
			Definition:   "SELECT * FROM users WHERE active",
		},
		// Function
		core.FunctionDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "audit_func"},
		},
		// Trigger (depends on users table and audit_func)
		core.TriggerDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindTrigger, Schema: "public", Name: "audit_trigger"},
			TableName:    "users",
			FunctionName: "audit_func",
		},
		// Policy (depends on users table)
		core.PolicyDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "users_policy"},
			TableName:    "users",
		},
		// Foreign keys (excluded from DAG, appended at end)
		core.ForeignKeyDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "orders_user_fk"},
			SourceTable:  "orders", TargetSchema: "public", TargetTable: "users",
			Definition: "FOREIGN KEY (user_id) REFERENCES users(id)",
		},
		core.ForeignKeyDef{
			ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "orders_self_fk"},
			SourceTable:  "orders", TargetSchema: "public", TargetTable: "orders",
			Definition: "FOREIGN KEY (parent_id) REFERENCES orders(id)",
		},
	}

	result, err := BuildRestoreOrder(objects)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 11 {
		t.Fatalf("expected 11 results, got %d", len(result))
	}

	// Build index map
	idx := make(map[string]int)
	for i, h := range result {
		idx[NodeID(h)] = i
	}

	// Sequences should come first (priority 0)
	// Enum should come after sequences (priority 1)
	// Tables after enum (priority 4), but tables depend on seqs/enum so definitely after
	// View after tables (priority 5)
	// Function after tables (priority 7 -- but no deps, so after view by priority)
	// Trigger after function and table (has explicit deps)
	// Policy after table (has explicit dep)
	// FKs at the very end

	// Verify sequences before tables
	if idx["public.sequence.user_id_seq"] > idx["public.table.users"] {
		t.Error("user_id_seq should come before users table")
	}
	if idx["public.sequence.order_id_seq"] > idx["public.table.orders"] {
		t.Error("order_id_seq should come before orders table")
	}

	// Verify enum before orders table (orders uses status_type)
	if idx["public.enum.status_type"] > idx["public.table.orders"] {
		t.Error("status_type enum should come before orders table")
	}

	// Verify trigger after table and function
	if idx["public.trigger.audit_trigger"] < idx["public.table.users"] {
		t.Error("trigger should come after users table")
	}
	if idx["public.trigger.audit_trigger"] < idx["public.function.audit_func"] {
		t.Error("trigger should come after audit_func")
	}

	// Verify policy after table
	if idx["public.policy.users_policy"] < idx["public.table.users"] {
		t.Error("policy should come after users table")
	}

	// Verify FKs are last two items
	fk1 := result[len(result)-2]
	fk2 := result[len(result)-1]
	if fk1.Kind != core.KindForeignKey || fk2.Kind != core.KindForeignKey {
		t.Errorf("last two items should be FKs; got %+v, %+v", fk1, fk2)
	}

	// Verify FK deterministic ordering (alphabetical by name)
	if fk1.Name > fk2.Name {
		t.Errorf("FKs should be sorted by name; got %q before %q", fk1.Name, fk2.Name)
	}
}

// helpers
func headerNames(headers []core.ObjectHeader) []string {
	names := make([]string, len(headers))
	for i, h := range headers {
		names[i] = h.Name
	}
	return names
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}
