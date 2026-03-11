package core

import "testing"

// TestPhase3StructFields verifies that the five Phase 3 *Def structs
// have the required fields with correct types. Compilation itself is the
// primary assertion; value checks confirm field accessibility.
func TestPhase3StructFields(t *testing.T) {
	// ViewDef: Definition string, Owner string
	vd := ViewDef{
		ObjectHeader: ObjectHeader{Kind: KindView, Schema: "public", Name: "v"},
		Definition:   "SELECT 1",
		Owner:        "alice",
	}
	if vd.Owner == "" {
		t.Error("ViewDef.Owner must be accessible")
	}
	if vd.Header().Kind != KindView {
		t.Errorf("ViewDef.Header().Kind = %q, want %q", vd.Header().Kind, KindView)
	}

	// MaterializedViewDef: Definition string, Owner string, IsPopulated bool
	mv := MaterializedViewDef{
		ObjectHeader: ObjectHeader{Kind: KindMaterializedView, Schema: "public", Name: "mv"},
		Definition:   "SELECT 1",
		Owner:        "bob",
		IsPopulated:  true,
	}
	if !mv.IsPopulated {
		t.Error("MaterializedViewDef.IsPopulated must be accessible")
	}
	if mv.Owner == "" {
		t.Error("MaterializedViewDef.Owner must be accessible")
	}

	// FunctionDef: Definition string, ArgTypes string, ReturnType string
	fd := FunctionDef{
		ObjectHeader: ObjectHeader{Kind: KindFunction, Schema: "public", Name: "f"},
		Definition:   "CREATE OR REPLACE FUNCTION ...",
		ArgTypes:     "a integer, b text",
		ReturnType:   "integer",
	}
	if fd.ArgTypes == "" {
		t.Error("FunctionDef.ArgTypes must be accessible")
	}
	if fd.ReturnType == "" {
		t.Error("FunctionDef.ReturnType must be accessible")
	}

	// TriggerDef: Timing string, Events []string, TableName string, FunctionName string
	td := TriggerDef{
		ObjectHeader: ObjectHeader{Kind: KindTrigger, Schema: "public", Name: "trg"},
		Timing:       "AFTER",
		Events:       []string{"INSERT", "UPDATE"},
		TableName:    "users",
		FunctionName: "audit_fn",
	}
	if td.Timing == "" {
		t.Error("TriggerDef.Timing must be accessible")
	}
	if len(td.Events) != 2 {
		t.Errorf("TriggerDef.Events length = %d, want 2", len(td.Events))
	}
	if td.TableName == "" {
		t.Error("TriggerDef.TableName must be accessible")
	}
	if td.FunctionName == "" {
		t.Error("TriggerDef.FunctionName must be accessible")
	}

	// PolicyDef: TableName string, Command string, Roles []string, Using string, WithCheck string
	pd := PolicyDef{
		ObjectHeader: ObjectHeader{Kind: KindPolicy, Schema: "public", Name: "pol"},
		TableName:    "orders",
		Command:      "ALL",
		Roles:        []string{"PUBLIC"},
		Using:        "(true)",
		WithCheck:    "(true)",
	}
	if pd.TableName == "" {
		t.Error("PolicyDef.TableName must be accessible")
	}
	if pd.Command == "" {
		t.Error("PolicyDef.Command must be accessible")
	}
	if len(pd.Roles) != 1 || pd.Roles[0] != "PUBLIC" {
		t.Errorf("PolicyDef.Roles = %v, want [PUBLIC]", pd.Roles)
	}
	if pd.Using == "" {
		t.Error("PolicyDef.Using must be accessible")
	}
	if pd.WithCheck == "" {
		t.Error("PolicyDef.WithCheck must be accessible")
	}
}
