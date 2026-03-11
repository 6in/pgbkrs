package core

import "testing"

// TestPhase4ForeignKeyDef verifies ForeignKeyDef struct compiles and Header()
// returns ObjectHeader with Kind=KindForeignKey.
func TestPhase4ForeignKeyDef(t *testing.T) {
	fk := ForeignKeyDef{
		ObjectHeader: ObjectHeader{Kind: KindForeignKey, Schema: "public", Name: "fk_orders_customer"},
		SourceTable:  "orders",
		TargetSchema: "public",
		TargetTable:  "customers",
		Definition:   "FOREIGN KEY (customer_id) REFERENCES customers(id)",
	}

	hdr := fk.Header()
	if hdr.Kind != KindForeignKey {
		t.Errorf("ForeignKeyDef.Header().Kind = %q, want %q", hdr.Kind, KindForeignKey)
	}
	if hdr.Schema != "public" {
		t.Errorf("ForeignKeyDef.Header().Schema = %q, want %q", hdr.Schema, "public")
	}
	if hdr.Name != "fk_orders_customer" {
		t.Errorf("ForeignKeyDef.Header().Name = %q, want %q", hdr.Name, "fk_orders_customer")
	}
}

// TestPhase4ForeignKeyDefFields verifies all ForeignKeyDef fields are accessible.
func TestPhase4ForeignKeyDefFields(t *testing.T) {
	fk := ForeignKeyDef{
		ObjectHeader: ObjectHeader{Kind: KindForeignKey, Schema: "public", Name: "fk_test"},
		SourceTable:  "orders",
		TargetSchema: "other_schema",
		TargetTable:  "customers",
		Definition:   "FOREIGN KEY (cid) REFERENCES customers(id)",
	}

	if fk.SourceTable != "orders" {
		t.Errorf("ForeignKeyDef.SourceTable = %q, want %q", fk.SourceTable, "orders")
	}
	if fk.TargetSchema != "other_schema" {
		t.Errorf("ForeignKeyDef.TargetSchema = %q, want %q", fk.TargetSchema, "other_schema")
	}
	if fk.TargetTable != "customers" {
		t.Errorf("ForeignKeyDef.TargetTable = %q, want %q", fk.TargetTable, "customers")
	}
	if fk.Definition == "" {
		t.Error("ForeignKeyDef.Definition must be accessible")
	}
}

// TestPhase4PartitionDefKeyExpression verifies PartitionDef.KeyExpression field is accessible.
func TestPhase4PartitionDefKeyExpression(t *testing.T) {
	pd := PartitionDef{
		Strategy:      "range",
		KeyExpression: "created_at",
		Children:      []string{"t_2024", "t_2025"},
	}
	if pd.KeyExpression != "created_at" {
		t.Errorf("PartitionDef.KeyExpression = %q, want %q", pd.KeyExpression, "created_at")
	}
}

// TestPhase4KindForeignKeyConstant verifies KindForeignKey equals "foreign_key".
func TestPhase4KindForeignKeyConstant(t *testing.T) {
	if KindForeignKey != "foreign_key" {
		t.Errorf("KindForeignKey = %q, want %q", KindForeignKey, "foreign_key")
	}
}
