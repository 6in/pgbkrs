package core

// ObjectKind identifies the type of a PostgreSQL schema object.
type ObjectKind string

const (
	KindTable            ObjectKind = "table"
	KindView             ObjectKind = "view"
	KindMaterializedView ObjectKind = "materialized_view"
	KindFunction         ObjectKind = "function"
	KindSequence         ObjectKind = "sequence"
	KindTrigger          ObjectKind = "trigger"
	KindType             ObjectKind = "type"
	KindDomain           ObjectKind = "domain"
	KindEnum             ObjectKind = "enum"
	KindPolicy           ObjectKind = "policy"
)

// ObjectHeader holds the identifying metadata for any PostgreSQL schema object.
type ObjectHeader struct {
	Kind   ObjectKind
	Schema string
	Name   string
}

// ObjectDef is the common interface for all schema object definitions.
type ObjectDef interface {
	Header() ObjectHeader
}

// ColumnDef describes a single column of a table.
type ColumnDef struct {
	Name     string
	Type     string // output of format_type()
	Nullable bool
	Default  string // output of pg_get_expr(); empty string if no default
}

// PrimaryKeyDef describes a primary key constraint.
type PrimaryKeyDef struct {
	Name    string
	Columns []string
}

// UniqueConstraintDef describes a unique constraint.
type UniqueConstraintDef struct {
	Name       string
	Definition string // pg_get_constraintdef output e.g. "UNIQUE (email)"
}

// CheckConstraintDef describes a check constraint.
type CheckConstraintDef struct {
	Name       string
	Definition string // pg_get_constraintdef output e.g. "CHECK (price > 0)"
}

// ConstraintsDef groups all constraints for a table.
type ConstraintsDef struct {
	PrimaryKey *PrimaryKeyDef
	Unique     []UniqueConstraintDef
	Check      []CheckConstraintDef
}

// IndexDef describes a table index.
type IndexDef struct {
	Name       string
	Method     string // btree, hash, gin, gist, etc.
	Definition string // full CREATE INDEX statement from pg_get_indexdef()
}

// PartitionDef describes a partitioned table's partitioning strategy.
type PartitionDef struct {
	Strategy string   // "range", "list", "hash"
	Children []string // child table names
}

// RLSDef describes row-level security settings for a table.
type RLSDef struct {
	Enabled bool
}

// CompositeField describes a single field of a composite type.
type CompositeField struct {
	Name string
	Type string // format_type() output
}

// TableDef represents a table schema object.
type TableDef struct {
	ObjectHeader
	Columns      []ColumnDef
	Constraints  ConstraintsDef
	Indexes      []IndexDef
	Partitioning *PartitionDef // nil for non-partitioned tables
	RLS          RLSDef
}

func (d TableDef) Header() ObjectHeader { return d.ObjectHeader }

// ViewDef represents a view schema object.
// Definition contains the SELECT body only (as returned by pg_views.definition).
// The full CREATE VIEW statement is reconstructed by DDLGenerator in Phase 4.
type ViewDef struct {
	ObjectHeader
	Definition string // SELECT body from pg_views.definition; NOT a complete DDL statement
	Owner      string // viewowner from pg_views
}

func (d ViewDef) Header() ObjectHeader { return d.ObjectHeader }

// MaterializedViewDef represents a materialized view schema object.
// IsPopulated is false when created WITH NO DATA (REFRESH MATERIALIZED VIEW never run).
type MaterializedViewDef struct {
	ObjectHeader
	Definition  string // SELECT body from pg_matviews.definition
	Owner       string // matviewowner from pg_matviews
	IsPopulated bool   // ispopulated from pg_matviews
}

func (d MaterializedViewDef) Header() ObjectHeader { return d.ObjectHeader }

// FunctionDef represents a function schema object (prokind='f' only; procedures excluded).
// Definition is the complete CREATE OR REPLACE FUNCTION statement from pg_get_functiondef.
type FunctionDef struct {
	ObjectHeader
	Definition string // complete CREATE OR REPLACE FUNCTION ... statement
	ArgTypes   string // pg_get_function_arguments() output e.g. "a integer, b text"
	ReturnType string // pg_get_function_result() output e.g. "integer"
}

func (d FunctionDef) Header() ObjectHeader { return d.ObjectHeader }

// SequenceDef represents a sequence schema object.
type SequenceDef struct {
	ObjectHeader
	StartValue  int64
	MinValue    int64
	MaxValue    int64
	IncrementBy int64
	Cycle       bool
	Cache       int64
	LastValue   int64 // current value at backup time; 0 if sequence never used
	IsCalled    bool  // false if sequence has never been advanced
}

func (d SequenceDef) Header() ObjectHeader { return d.ObjectHeader }

// TriggerDef represents a trigger schema object (user-defined only; tgisinternal=false).
// Timing is one of: "BEFORE", "AFTER", "INSTEAD OF".
// Events contains one or more of: "INSERT", "UPDATE", "DELETE", "TRUNCATE".
// TableName is the unqualified name of the trigger's target table.
// FunctionName is the unqualified name of the trigger function.
type TriggerDef struct {
	ObjectHeader
	Timing       string   // "BEFORE", "AFTER", or "INSTEAD OF"
	Events       []string // e.g. ["INSERT", "UPDATE"]
	TableName    string   // target table (unqualified)
	FunctionName string   // trigger function name (unqualified)
}

func (d TriggerDef) Header() ObjectHeader { return d.ObjectHeader }

// TypeDef represents a composite/base type schema object.
type TypeDef struct {
	ObjectHeader
	Fields []CompositeField
}

func (d TypeDef) Header() ObjectHeader { return d.ObjectHeader }

// DomainDef represents a domain schema object.
type DomainDef struct {
	ObjectHeader
	BaseType        string // format_type() output
	Nullable        bool
	Default         string // empty if no default
	CheckName       string // empty if no check constraint
	CheckDefinition string // pg_get_constraintdef() output; empty if no check
}

func (d DomainDef) Header() ObjectHeader { return d.ObjectHeader }

// EnumDef represents an enum type schema object.
type EnumDef struct {
	ObjectHeader
	Labels []string // ordered by enumsortorder
}

func (d EnumDef) Header() ObjectHeader { return d.ObjectHeader }

// PolicyDef represents a row-level security policy.
// Command is one of: "SELECT", "INSERT", "UPDATE", "DELETE", "ALL".
// Roles contains role names; contains "PUBLIC" when polroles={0}.
// Using is the USING expression (empty string if none).
// WithCheck is the WITH CHECK expression (empty string if none).
type PolicyDef struct {
	ObjectHeader
	TableName string   // target table name (unqualified)
	Command   string   // "SELECT", "INSERT", "UPDATE", "DELETE", or "ALL"
	Roles     []string // role names; ["PUBLIC"] when polroles={0}
	Using     string   // USING expression or empty string
	WithCheck string   // WITH CHECK expression or empty string
}

func (d PolicyDef) Header() ObjectHeader { return d.ObjectHeader }
