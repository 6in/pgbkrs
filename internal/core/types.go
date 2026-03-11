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

// TableDef represents a table schema object.
type TableDef struct {
	ObjectHeader
	// Columns, Indexes, Constraints, etc. — populated in Phase 2+
}

func (d TableDef) Header() ObjectHeader { return d.ObjectHeader }

// ViewDef represents a view schema object.
type ViewDef struct {
	ObjectHeader
	Definition string
}

func (d ViewDef) Header() ObjectHeader { return d.ObjectHeader }

// MaterializedViewDef represents a materialized view schema object.
type MaterializedViewDef struct {
	ObjectHeader
	Definition string
}

func (d MaterializedViewDef) Header() ObjectHeader { return d.ObjectHeader }

// FunctionDef represents a function schema object.
type FunctionDef struct {
	ObjectHeader
	Definition string
}

func (d FunctionDef) Header() ObjectHeader { return d.ObjectHeader }

// SequenceDef represents a sequence schema object.
type SequenceDef struct {
	ObjectHeader
	Definition string
}

func (d SequenceDef) Header() ObjectHeader { return d.ObjectHeader }

// TriggerDef represents a trigger schema object.
type TriggerDef struct {
	ObjectHeader
	Definition string
}

func (d TriggerDef) Header() ObjectHeader { return d.ObjectHeader }

// TypeDef represents a composite/base type schema object.
type TypeDef struct {
	ObjectHeader
	Definition string
}

func (d TypeDef) Header() ObjectHeader { return d.ObjectHeader }

// DomainDef represents a domain schema object.
type DomainDef struct {
	ObjectHeader
	Definition string
}

func (d DomainDef) Header() ObjectHeader { return d.ObjectHeader }

// EnumDef represents an enum type schema object.
type EnumDef struct {
	ObjectHeader
	Definition string
}

func (d EnumDef) Header() ObjectHeader { return d.ObjectHeader }

// PolicyDef represents a row-level security policy schema object.
type PolicyDef struct {
	ObjectHeader
	Definition string
}

func (d PolicyDef) Header() ObjectHeader { return d.ObjectHeader }
