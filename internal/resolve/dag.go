package resolve

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// nextvalRe matches nextval('schema.seqname'::regclass) or nextval('seqname'::regclass)
var nextvalRe = regexp.MustCompile(`nextval\('([^']+)'`)

// NodeID returns a deterministic string identifier for an ObjectHeader.
// Format: "schema.kind.name"
func NodeID(h core.ObjectHeader) string {
	return fmt.Sprintf("%s.%s.%s", h.Schema, string(h.Kind), h.Name)
}

// DAG represents a directed acyclic graph of PostgreSQL object dependencies.
type DAG struct {
	nodes       map[string]core.ObjectHeader // nodeID -> header
	edges       map[string][]string          // nodeID -> list of nodeIDs this node depends ON
	inDegree    map[string]int               // nodeID -> count of incoming edges (how many things depend on this)
	customTypes map[string]string            // type name -> nodeID (for column type matching)
	objects     []core.ObjectDef             // stored for deferred resolution
}

// NewDAG creates an empty DAG with initialized maps.
func NewDAG() *DAG {
	return &DAG{
		nodes:       make(map[string]core.ObjectHeader),
		edges:       make(map[string][]string),
		inDegree:    make(map[string]int),
		customTypes: make(map[string]string),
	}
}

// Nodes returns all registered ObjectHeaders.
func (d *DAG) Nodes() []core.ObjectHeader {
	result := make([]core.ObjectHeader, 0, len(d.nodes))
	for _, h := range d.nodes {
		result = append(result, h)
	}
	return result
}

// InDegree returns the in-degree for a node (how many nodes depend on it is NOT this;
// this is how many dependencies this node has = len(edges[nodeID])).
// Actually, inDegree tracks how many other nodes list this node as a dependency target.
// Wait -- let's clarify: in Kahn's algorithm, inDegree[v] = number of edges pointing TO v.
// An edge "A depends on B" means A->B in dependency sense. In graph terms, B->A (B must come before A).
// So inDegree[A] = number of things A depends on.
// For Kahn's: we process nodes with inDegree 0 first (no dependencies).
func (d *DAG) InDegree(nodeID string) int {
	return d.inDegree[nodeID]
}

// DepsOf returns the list of nodeIDs that the given node depends on.
func (d *DAG) DepsOf(nodeID string) []string {
	return d.edges[nodeID]
}

// AddNode registers a node in the DAG if not already present.
func (d *DAG) AddNode(h core.ObjectHeader) {
	id := NodeID(h)
	if _, exists := d.nodes[id]; !exists {
		d.nodes[id] = h
		d.inDegree[id] = 0
	}
}

// AddEdge declares that 'from' depends on 'to'.
// In Kahn's terms: 'to' must be restored before 'from'.
// This increments inDegree[from].
func (d *DAG) AddEdge(from, to string) {
	// Avoid duplicate edges
	for _, dep := range d.edges[from] {
		if dep == to {
			return
		}
	}
	d.edges[from] = append(d.edges[from], to)
	d.inDegree[from]++
}

// AddObject registers an object in the DAG. Returns false for ForeignKeyDef (excluded).
// After all objects are added, call Resolve() to build dependency edges.
func (d *DAG) AddObject(obj core.ObjectDef) bool {
	h := obj.Header()
	if h.Kind == core.KindForeignKey {
		return false
	}

	d.AddNode(h)
	d.objects = append(d.objects, obj)

	// Register custom types for later column-type matching
	if h.Kind == core.KindType || h.Kind == core.KindDomain || h.Kind == core.KindEnum {
		id := NodeID(h)
		// Register both qualified "schema.name" and unqualified "name"
		d.customTypes[h.Schema+"."+h.Name] = id
		d.customTypes[h.Name] = id
	}

	return true
}

// Resolve builds dependency edges based on the full set of registered nodes.
// Must be called after all AddObject calls.
func (d *DAG) Resolve() {
	for _, obj := range d.objects {
		h := obj.Header()
		fromID := NodeID(h)

		switch h.Kind {
		case core.KindTable:
			d.resolveTable(fromID, obj)
		case core.KindTrigger:
			d.resolveTrigger(fromID, obj)
		case core.KindPolicy:
			d.resolvePolicy(fromID, obj)
		case core.KindFunction:
			d.resolveFunction(fromID, obj)
		case core.KindDomain:
			d.resolveDomain(fromID, obj)
		case core.KindType:
			d.resolveType(fromID, obj)
		// sequence, enum, view, materialized_view: no explicit edges
		}
	}
}

func (d *DAG) resolveTable(fromID string, obj core.ObjectDef) {
	table, ok := obj.(core.TableDef)
	if !ok {
		if tp, ok2 := obj.(*core.TableDef); ok2 {
			table = *tp
		} else {
			return
		}
	}
	h := table.Header()

	for _, col := range table.Columns {
		// Check for nextval sequence dependency
		if matches := nextvalRe.FindStringSubmatch(col.Default); len(matches) > 1 {
			seqRef := matches[1]
			// Strip ::regclass suffix if present
			seqRef = strings.TrimSuffix(seqRef, "::regclass")
			var seqSchema, seqName string
			if parts := strings.SplitN(seqRef, ".", 2); len(parts) == 2 {
				seqSchema = parts[0]
				seqName = parts[1]
			} else {
				seqSchema = h.Schema
				seqName = seqRef
			}
			targetID := fmt.Sprintf("%s.%s.%s", seqSchema, string(core.KindSequence), seqName)
			if _, exists := d.nodes[targetID]; exists {
				d.AddEdge(fromID, targetID)
			}
		}

		// Check column type against custom types
		d.addCustomTypeDep(fromID, col.Type)
	}
}

func (d *DAG) resolveTrigger(fromID string, obj core.ObjectDef) {
	trig, ok := obj.(core.TriggerDef)
	if !ok {
		return
	}
	h := trig.Header()

	// Trigger depends on its table
	tableID := fmt.Sprintf("%s.%s.%s", h.Schema, string(core.KindTable), trig.TableName)
	if _, exists := d.nodes[tableID]; exists {
		d.AddEdge(fromID, tableID)
	}

	// Trigger depends on its function
	fnID := fmt.Sprintf("%s.%s.%s", h.Schema, string(core.KindFunction), trig.FunctionName)
	if _, exists := d.nodes[fnID]; exists {
		d.AddEdge(fromID, fnID)
	}
}

func (d *DAG) resolvePolicy(fromID string, obj core.ObjectDef) {
	pol, ok := obj.(core.PolicyDef)
	if !ok {
		return
	}
	h := pol.Header()

	tableID := fmt.Sprintf("%s.%s.%s", h.Schema, string(core.KindTable), pol.TableName)
	if _, exists := d.nodes[tableID]; exists {
		d.AddEdge(fromID, tableID)
	}
}

func (d *DAG) resolveFunction(fromID string, obj core.ObjectDef) {
	fn, ok := obj.(core.FunctionDef)
	if !ok {
		return
	}

	// Check arg types
	if fn.ArgTypes != "" {
		for _, arg := range strings.Split(fn.ArgTypes, ",") {
			arg = strings.TrimSpace(arg)
			// arg format: "name type" or just "type"
			parts := strings.Fields(arg)
			if len(parts) >= 1 {
				typeName := parts[len(parts)-1]
				d.addCustomTypeDep(fromID, typeName)
			}
		}
	}

	// Check return type
	if fn.ReturnType != "" {
		d.addCustomTypeDep(fromID, fn.ReturnType)
	}
}

func (d *DAG) resolveDomain(fromID string, obj core.ObjectDef) {
	dom, ok := obj.(core.DomainDef)
	if !ok {
		return
	}
	d.addCustomTypeDep(fromID, dom.BaseType)
}

func (d *DAG) resolveType(fromID string, obj core.ObjectDef) {
	typ, ok := obj.(core.TypeDef)
	if !ok {
		return
	}
	for _, f := range typ.Fields {
		d.addCustomTypeDep(fromID, f.Type)
	}
}

// addCustomTypeDep checks if a type name matches a registered custom type and adds an edge.
func (d *DAG) addCustomTypeDep(fromID, typeName string) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return
	}
	// Don't create self-dependencies
	if targetID, ok := d.customTypes[typeName]; ok && targetID != fromID {
		if _, exists := d.nodes[targetID]; exists {
			d.AddEdge(fromID, targetID)
		}
	}
}
