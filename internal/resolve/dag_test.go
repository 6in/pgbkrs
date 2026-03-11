package resolve

import (
	"testing"

	"github.com/pgbkrs/pgbackup/internal/core"
)

func TestNewDAG(t *testing.T) {
	d := NewDAG()
	if d == nil {
		t.Fatal("NewDAG returned nil")
	}
	if len(d.Nodes()) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(d.Nodes()))
	}
}

func TestNodeID(t *testing.T) {
	h := core.ObjectHeader{Schema: "public", Kind: core.KindTable, Name: "users"}
	got := NodeID(h)
	want := "public.table.users"
	if got != want {
		t.Errorf("NodeID = %q, want %q", got, want)
	}
}

func TestAddNode(t *testing.T) {
	d := NewDAG()
	h := core.ObjectHeader{Schema: "public", Kind: core.KindTable, Name: "users"}
	d.AddNode(h)

	nodes := d.Nodes()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0] != h {
		t.Errorf("node = %+v, want %+v", nodes[0], h)
	}
	// in-degree should be 0 for new node
	if deg := d.InDegree(NodeID(h)); deg != 0 {
		t.Errorf("in-degree = %d, want 0", deg)
	}
}

func TestAddObject_Table_SequenceDep(t *testing.T) {
	d := NewDAG()

	seq := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "user_id_seq"},
	}
	table := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
		Columns: []core.ColumnDef{
			{Name: "id", Type: "integer", Default: "nextval('public.user_id_seq'::regclass)"},
		},
	}

	d.AddObject(seq)
	d.AddObject(table)
	d.Resolve()

	tableID := NodeID(table.Header())
	seqID := NodeID(seq.Header())
	deps := d.DepsOf(tableID)
	if !contains(deps, seqID) {
		t.Errorf("table %q should depend on sequence %q; deps = %v", tableID, seqID, deps)
	}
}

func TestAddObject_Table_TypeDep(t *testing.T) {
	d := NewDAG()

	enumDef := core.EnumDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "status_type"},
		Labels:       []string{"active", "inactive"},
	}
	table := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
		Columns: []core.ColumnDef{
			{Name: "status", Type: "status_type"},
		},
	}

	d.AddObject(enumDef)
	d.AddObject(table)
	d.Resolve()

	tableID := NodeID(table.Header())
	enumID := NodeID(enumDef.Header())
	deps := d.DepsOf(tableID)
	if !contains(deps, enumID) {
		t.Errorf("table %q should depend on enum %q; deps = %v", tableID, enumID, deps)
	}
}

func TestAddObject_Trigger(t *testing.T) {
	d := NewDAG()

	table := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}
	fn := core.FunctionDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "audit_func"},
	}
	trigger := core.TriggerDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTrigger, Schema: "public", Name: "audit_trigger"},
		TableName:    "users",
		FunctionName: "audit_func",
	}

	d.AddObject(table)
	d.AddObject(fn)
	d.AddObject(trigger)
	d.Resolve()

	trigID := NodeID(trigger.Header())
	tableID := NodeID(table.Header())
	fnID := NodeID(fn.Header())
	deps := d.DepsOf(trigID)
	if !contains(deps, tableID) {
		t.Errorf("trigger should depend on table; deps = %v", deps)
	}
	if !contains(deps, fnID) {
		t.Errorf("trigger should depend on function; deps = %v", deps)
	}
}

func TestAddObject_Policy(t *testing.T) {
	d := NewDAG()

	table := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "users"},
	}
	policy := core.PolicyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindPolicy, Schema: "public", Name: "users_policy"},
		TableName:    "users",
	}

	d.AddObject(table)
	d.AddObject(policy)
	d.Resolve()

	polID := NodeID(policy.Header())
	tableID := NodeID(table.Header())
	deps := d.DepsOf(polID)
	if !contains(deps, tableID) {
		t.Errorf("policy should depend on table; deps = %v", deps)
	}
}

func TestAddObject_Function(t *testing.T) {
	d := NewDAG()

	customType := core.TypeDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "public", Name: "address_type"},
	}
	fn := core.FunctionDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindFunction, Schema: "public", Name: "get_address"},
		ArgTypes:     "id integer, addr address_type",
		ReturnType:   "address_type",
	}

	d.AddObject(customType)
	d.AddObject(fn)
	d.Resolve()

	fnID := NodeID(fn.Header())
	typeID := NodeID(customType.Header())
	deps := d.DepsOf(fnID)
	if !contains(deps, typeID) {
		t.Errorf("function should depend on type; deps = %v", deps)
	}
}

func TestAddObject_Domain(t *testing.T) {
	d := NewDAG()

	baseType := core.TypeDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "public", Name: "base_type"},
	}
	domain := core.DomainDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindDomain, Schema: "public", Name: "email"},
		BaseType:     "base_type",
	}

	d.AddObject(baseType)
	d.AddObject(domain)
	d.Resolve()

	domID := NodeID(domain.Header())
	typeID := NodeID(baseType.Header())
	deps := d.DepsOf(domID)
	if !contains(deps, typeID) {
		t.Errorf("domain should depend on type; deps = %v", deps)
	}
}

func TestAddObject_Type(t *testing.T) {
	d := NewDAG()

	inner := core.TypeDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "public", Name: "inner_type"},
	}
	outer := core.TypeDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindType, Schema: "public", Name: "outer_type"},
		Fields: []core.CompositeField{
			{Name: "data", Type: "inner_type"},
		},
	}

	d.AddObject(inner)
	d.AddObject(outer)
	d.Resolve()

	outerID := NodeID(outer.Header())
	innerID := NodeID(inner.Header())
	deps := d.DepsOf(outerID)
	if !contains(deps, innerID) {
		t.Errorf("outer type should depend on inner type; deps = %v", deps)
	}
}

func TestAddObject_Sequence(t *testing.T) {
	d := NewDAG()

	seq := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "my_seq"},
	}
	d.AddObject(seq)
	d.Resolve()

	seqID := NodeID(seq.Header())
	deps := d.DepsOf(seqID)
	if len(deps) != 0 {
		t.Errorf("sequence should have no deps; got %v", deps)
	}
}

func TestAddObject_Enum(t *testing.T) {
	d := NewDAG()

	e := core.EnumDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindEnum, Schema: "public", Name: "color"},
		Labels:       []string{"red", "green", "blue"},
	}
	d.AddObject(e)
	d.Resolve()

	enumID := NodeID(e.Header())
	deps := d.DepsOf(enumID)
	if len(deps) != 0 {
		t.Errorf("enum should have no deps; got %v", deps)
	}
}

func TestAddObject_View(t *testing.T) {
	d := NewDAG()

	v := core.ViewDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindView, Schema: "public", Name: "active_users"},
		Definition:   "SELECT * FROM users WHERE active",
	}
	d.AddObject(v)
	d.Resolve()

	viewID := NodeID(v.Header())
	deps := d.DepsOf(viewID)
	// Views rely on type-priority tiebreaking, not explicit edges
	if len(deps) != 0 {
		t.Errorf("view should have no explicit deps; got %v", deps)
	}
}

func TestAddObject_ForeignKey_Excluded(t *testing.T) {
	d := NewDAG()

	fk := core.ForeignKeyDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindForeignKey, Schema: "public", Name: "orders_user_fk"},
		SourceTable:  "orders",
		TargetSchema: "public",
		TargetTable:  "users",
		Definition:   "FOREIGN KEY (user_id) REFERENCES users(id)",
	}
	added := d.AddObject(fk)
	if added {
		t.Error("ForeignKeyDef should not be added to DAG (AddObject should return false)")
	}
	if len(d.Nodes()) != 0 {
		t.Errorf("DAG should have 0 nodes after FK add; got %d", len(d.Nodes()))
	}
}

func TestIsolatedNodes(t *testing.T) {
	d := NewDAG()

	t1 := core.TableDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindTable, Schema: "public", Name: "standalone"},
	}
	s1 := core.SequenceDef{
		ObjectHeader: core.ObjectHeader{Kind: core.KindSequence, Schema: "public", Name: "lone_seq"},
	}

	d.AddObject(t1)
	d.AddObject(s1)
	d.Resolve()

	nodes := d.Nodes()
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	// Both should have in-degree 0
	if deg := d.InDegree(NodeID(t1.Header())); deg != 0 {
		t.Errorf("table in-degree = %d, want 0", deg)
	}
	if deg := d.InDegree(NodeID(s1.Header())); deg != 0 {
		t.Errorf("sequence in-degree = %d, want 0", deg)
	}
}

// helper
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
