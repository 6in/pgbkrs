package resolve

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// kindPriority assigns numeric priority to each ObjectKind for deterministic
// tiebreaking when multiple nodes have in-degree 0 during topological sort.
// Lower number = restored earlier.
var kindPriority = map[core.ObjectKind]int{
	core.KindSequence:         0,
	core.KindEnum:             1,
	core.KindType:             2,
	core.KindDomain:           3,
	core.KindTable:            4,
	core.KindView:             5,
	core.KindMaterializedView: 6,
	core.KindFunction:         7,
	core.KindTrigger:          8,
	core.KindPolicy:           9,
}

// nodeSort implements deterministic ordering for candidate nodes:
// primary sort by kindPriority, secondary by schema, tertiary by name.
func nodeSort(nodes []string, headers map[string]core.ObjectHeader) {
	sort.Slice(nodes, func(i, j int) bool {
		hi := headers[nodes[i]]
		hj := headers[nodes[j]]
		pi := kindPriority[hi.Kind]
		pj := kindPriority[hj.Kind]
		if pi != pj {
			return pi < pj
		}
		if hi.Schema != hj.Schema {
			return hi.Schema < hj.Schema
		}
		return hi.Name < hj.Name
	})
}

// TopologicalSort performs Kahn's algorithm on the DAG.
// Returns an ordered list of ObjectHeaders where dependencies appear before dependents.
// Uses kind-priority tiebreaking for deterministic output.
// Returns an error if a cycle is detected.
func (d *DAG) TopologicalSort() ([]core.ObjectHeader, error) {
	// Copy inDegree to avoid mutating DAG state
	inDeg := make(map[string]int, len(d.inDegree))
	for k, v := range d.inDegree {
		inDeg[k] = v
	}

	// Build reverse adjacency: for each edge "from depends on to",
	// we need "to -> [from]" so when we process 'to', we can decrement 'from's in-degree
	reverse := make(map[string][]string)
	for from, deps := range d.edges {
		for _, to := range deps {
			reverse[to] = append(reverse[to], from)
		}
	}

	// Collect initial zero-in-degree candidates
	var candidates []string
	for id := range d.nodes {
		if inDeg[id] == 0 {
			candidates = append(candidates, id)
		}
	}
	nodeSort(candidates, d.nodes)

	var result []core.ObjectHeader

	for len(candidates) > 0 {
		// Pop first (lowest priority) candidate
		current := candidates[0]
		candidates = candidates[1:]

		result = append(result, d.nodes[current])

		// Decrement in-degree for nodes that depend on current
		for _, dependent := range reverse[current] {
			inDeg[dependent]--
			if inDeg[dependent] == 0 {
				candidates = append(candidates, dependent)
			}
		}

		// Re-sort candidates for determinism
		nodeSort(candidates, d.nodes)
	}

	// Cycle detection
	if len(result) != len(d.nodes) {
		var cycleNodes []string
		for id, deg := range inDeg {
			if deg > 0 {
				cycleNodes = append(cycleNodes, id)
			}
		}
		sort.Strings(cycleNodes)
		return nil, fmt.Errorf("circular dependency detected among: [%s]", strings.Join(cycleNodes, ", "))
	}

	return result, nil
}

// BuildRestoreOrder determines the correct restore order for a set of PostgreSQL objects.
// ForeignKeyDef objects are excluded from the DAG and appended at the end.
// Returns an error if a circular dependency is detected among non-FK objects.
func BuildRestoreOrder(objects []core.ObjectDef) ([]core.ObjectHeader, error) {
	if len(objects) == 0 {
		return nil, nil
	}

	// Separate FK objects
	var nonFK []core.ObjectDef
	var fks []core.ObjectHeader
	for _, obj := range objects {
		h := obj.Header()
		if h.Kind == core.KindForeignKey {
			fks = append(fks, h)
		} else {
			nonFK = append(nonFK, obj)
		}
	}

	// Build DAG from non-FK objects
	dag := NewDAG()
	for _, obj := range nonFK {
		dag.AddObject(obj)
	}
	dag.Resolve()

	// Topological sort
	sorted, err := dag.TopologicalSort()
	if err != nil {
		return nil, err
	}

	// Sort FKs deterministically by schema then name
	sort.Slice(fks, func(i, j int) bool {
		if fks[i].Schema != fks[j].Schema {
			return fks[i].Schema < fks[j].Schema
		}
		return fks[i].Name < fks[j].Name
	})

	// Append FKs after all other objects
	result := append(sorted, fks...)
	return result, nil
}
