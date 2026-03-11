# Phase 6: Dependency Resolution & Manifest - Research

**Researched:** 2026-03-11
**Domain:** Graph algorithms (DAG, topological sort), YAML manifest generation
**Confidence:** HIGH

## Summary

Phase 6 builds the dependency resolution engine and manifest writer for pgbackup. The core problem is: given a flat set of fetched `ObjectDef` values (tables, views, functions, triggers, sequences, types, domains, enums, policies, foreign keys), determine the correct restore order by constructing an in-memory DAG and running Kahn's topological sort. FK constraints are already modeled as independent `ForeignKeyDef` objects (Phase 4 decision), so the circular FK problem is architecturally solved -- the implementation must place FK objects after all tables in the restore order.

This phase is pure Go algorithmic code with no external dependencies beyond `go.yaml.in/yaml/v3` (already in go.mod). The DAG and Kahn's algorithm are textbook graph operations that should be hand-rolled in Go (no graph library needed for this scale). The output is `_manifest.yaml` matching the spec in section 5.1 of `docs/pre/pgbackup_spec.md`.

**Primary recommendation:** Create a new `internal/resolve` package containing the DAG builder, Kahn's sort, and manifest writer. Keep graph logic pure (no DB, no I/O) for easy unit testing.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DEPS-01 | In-memory DAG construction (inter-object dependencies) | DAG builder analyzes ObjectDef headers + type-specific fields to extract edges; see Architecture Patterns |
| DEPS-02 | Kahn's algorithm topological sort for restore order | Standard Kahn's with in-degree tracking; see Code Examples |
| DEPS-03 | FK constraint isolation as independent objects (circular FK avoidance) | ForeignKeyDef already exists as independent ObjectDef; DAG builder excludes FK->table edges and places FKs at end |
| DEPS-04 | _manifest.yaml generation (dependencies, restore order, metadata, skip info) | YAML struct matching spec 5.1 serialized with go.yaml.in/yaml/v3; see Manifest Schema |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (`sort`, `fmt`) | 1.24 | Graph algorithm, string formatting | Pure algorithmic code, no external deps needed |
| go.yaml.in/yaml/v3 | v3.0.4 | _manifest.yaml serialization | Already in go.mod, project standard for YAML |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `internal/core` | project | ObjectDef types, ObjectKind, ObjectHeader | Input to DAG builder |
| `testing` (stdlib) | 1.24 | Unit tests for graph logic | All tests |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled DAG | gonum/graph | Massive overkill for ~100 nodes; adds dependency for trivial operations |
| Kahn's algorithm | DFS-based topo sort | Kahn's is spec requirement; also better for cycle detection reporting |

**Installation:**
```bash
# No new dependencies needed -- all already in go.mod
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── resolve/              # NEW: Phase 6
│   ├── dag.go            # DAG type, AddNode, AddEdge, dependency extraction
│   ├── dag_test.go       # Unit tests for DAG construction
│   ├── kahn.go           # Kahn's topological sort
│   ├── kahn_test.go      # Unit tests including cycle detection
│   ├── manifest.go       # ManifestWriter, _manifest.yaml struct + serialization
│   └── manifest_test.go  # Manifest YAML output verification
```

### Pattern 1: DAG as Adjacency List with In-Degree Map

**What:** The DAG is a struct holding `map[string][]string` (adjacency list: node -> dependencies) and `map[string]int` (in-degree count). Node IDs are `"schema.kind.name"` strings derived from `ObjectHeader`.

**When to use:** Always -- this is the only DAG representation needed.

**Key design:**
```go
// NodeID generates a unique identifier for an ObjectDef.
// Format: "schema.kind.name" e.g. "public.table.users"
func NodeID(h core.ObjectHeader) string {
    return fmt.Sprintf("%s.%s.%s", h.Schema, string(h.Kind), h.Name)
}

type DAG struct {
    nodes    map[string]core.ObjectHeader  // nodeID -> header
    edges    map[string][]string           // nodeID -> list of dependency nodeIDs
    inDegree map[string]int                // nodeID -> count of incoming edges
}
```

### Pattern 2: Dependency Extraction by ObjectKind

**What:** Each ObjectKind has specific dependency rules. The DAG builder inspects ObjectDef fields to determine edges.

**Dependency rules (exhaustive for this project):**

| Object Kind | Depends On | How to Extract |
|-------------|-----------|----------------|
| `table` | `sequence` (via column defaults containing `nextval`), `type`/`domain`/`enum` (via column types) | Parse ColumnDef.Default for `nextval('seq_name')`, check ColumnDef.Type against known types/domains/enums |
| `view` | `table`, `view`, `function` (via definition) | View definition text analysis or pg_depend query; **recommend: accept ObjectDef list and build deps from pg_catalog at fetch time** |
| `materialized_view` | Same as view | Same approach |
| `function` | `type`, `domain`, `enum` (via arg/return types) | Parse ArgTypes and ReturnType |
| `trigger` | `table` (TableName), `function` (FunctionName) | Direct field access on TriggerDef |
| `sequence` | nothing | Leaf node |
| `type` | `type` (via composite field types) | Parse Fields[].Type |
| `domain` | `type` (base type) | Parse BaseType |
| `enum` | nothing | Leaf node |
| `policy` | `table` (TableName) | Direct field access on PolicyDef |
| `foreign_key` | EXCLUDED from normal DAG | Always placed after ALL tables; see Pattern 3 |

**Critical insight:** View/matview dependencies on other objects within their SELECT body are the hardest to extract statically from the current ObjectDef fields (ViewDef.Definition is just the SELECT body text). Two approaches:

1. **Simple approach (recommended for v1):** Use pg_depend catalog to fetch dependencies at backup time (add a DependencyFetcher), or hardcode a conservative ordering where views always come after all tables/functions/types.
2. **Complex approach:** Parse SQL text to extract referenced objects. Fragile and error-prone.

**Recommendation:** For DEPS-01, add a lightweight `pg_depend` query that returns `(source_oid, dep_oid)` pairs for all objects in a schema. This gives accurate dependency data without SQL parsing. Alternatively, if avoiding DB queries in Phase 6 (pure algorithmic phase), use a type-priority ordering with field-based extraction for known cases (trigger->table, trigger->function, policy->table, table->sequence).

### Pattern 3: FK Constraint Isolation

**What:** ForeignKeyDef objects are never added as dependencies of tables in the DAG. Instead, they are collected separately and appended to the restore order after all other objects.

**Why:** This is the project's core architectural decision for circular FK avoidance. Tables A and B can reference each other via FKs. By treating FK constraints as independent objects applied after all tables exist + data loaded, the circular dependency vanishes.

**Implementation:**
```go
func BuildRestoreOrder(objects []core.ObjectDef) ([]core.ObjectHeader, error) {
    var fks []core.ObjectHeader
    var others []core.ObjectDef

    // Separate FK constraints from other objects
    for _, obj := range objects {
        if obj.Header().Kind == core.KindForeignKey {
            fks = append(fks, obj.Header())
        } else {
            others = append(others, obj)
        }
    }

    // Build DAG from non-FK objects only
    dag := NewDAG()
    for _, obj := range others {
        dag.AddObject(obj)
    }

    // Kahn's sort on non-FK objects
    order, err := dag.TopologicalSort()
    if err != nil {
        return nil, err // true circular dependency (non-FK)
    }

    // Append FK constraints at the end
    order = append(order, fks...)
    return order, nil
}
```

### Pattern 4: Manifest YAML Structure

**What:** The `_manifest.yaml` file follows the spec (section 5.1). Use intermediate structs with yaml tags.

```go
type Manifest struct {
    BackupAt    string          `yaml:"backup_at"`
    PgVersion   string          `yaml:"pg_version"`
    ToolVersion string          `yaml:"tool_version"`
    Snapshot    bool            `yaml:"snapshot"`
    SkippedTables []SkipEntry  `yaml:"skipped_tables"`
    Objects     []ObjectEntry   `yaml:"objects"`
    RestoreOrder []RestoreEntry `yaml:"restore_order"`
}

type SkipEntry struct {
    Schema string `yaml:"schema"`
    Name   string `yaml:"name"`
    Reason string `yaml:"reason"`
}

type ObjectEntry struct {
    ID        string   `yaml:"id"`
    Kind      string   `yaml:"kind"`
    DependsOn []string `yaml:"depends_on"`
}

type RestoreEntry struct {
    Schema    string `yaml:"schema"`
    Kind      string `yaml:"kind"`
    Name      string `yaml:"name"`
    FromTable string `yaml:"from_table,omitempty"` // only for FK
}
```

### Anti-Patterns to Avoid
- **Adding ForeignKeyDef to the DAG as table dependency:** This reintroduces the circular FK problem. FKs must be excluded from the DAG entirely.
- **Parsing SQL text to find view dependencies:** Fragile, complex, and unnecessary when pg_depend or conservative ordering works.
- **Global singleton DAG:** The DAG should be created per-backup-run, passed as a value. No package-level state.
- **Mixing I/O with graph logic:** Keep DAG construction and Kahn's sort pure (accept `[]ObjectDef`, return `[]ObjectHeader`). Manifest file writing is a separate concern.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| YAML serialization | Custom YAML formatter | go.yaml.in/yaml/v3 with struct tags | Handles quoting, escaping, flow style correctly |

**Key insight:** The DAG and Kahn's algorithm SHOULD be hand-rolled. They are 50-80 lines of straightforward Go code. Importing a graph library would add unnecessary dependency for this scale (~100 nodes max).

## Common Pitfalls

### Pitfall 1: Forgetting to Add Isolated Nodes
**What goes wrong:** Objects with no dependencies AND no dependents are missing from the DAG and thus missing from restore_order.
**Why it happens:** Only adding nodes when processing edges.
**How to avoid:** Always add every object as a node first, then add edges. Two-pass: `AddNode` for all, then `AddEdge` for dependencies.
**Warning signs:** restore_order has fewer items than the object list.

### Pitfall 2: Non-Deterministic Output
**What goes wrong:** Kahn's algorithm with a simple queue produces different orderings on different runs when multiple nodes have in-degree 0 simultaneously.
**Why it happens:** Map iteration order in Go is randomized.
**How to avoid:** Use a sorted slice (or `sort.Strings`) for the zero-in-degree candidates at each step. This produces stable, reproducible manifest files.
**Warning signs:** Tests pass intermittently, manifest diffs appear with no code changes.

### Pitfall 3: Confusing "No Cycle Found" with "Correct"
**What goes wrong:** Kahn's algorithm finishes but the output length != input node count, indicating an undetected cycle.
**Why it happens:** Not checking the completion condition.
**How to avoid:** After Kahn's completes, verify `len(result) == len(nodes)`. If not equal, there is a cycle. Report the remaining nodes (those with in-degree > 0) in the error message.
**Warning signs:** Silent data loss in restore_order.

### Pitfall 4: Schema-Qualified Name Collisions
**What goes wrong:** Two objects in different schemas with the same name collide in the DAG.
**Why it happens:** Using `kind.name` instead of `schema.kind.name` as node ID.
**How to avoid:** Always include schema in the node ID.
**Warning signs:** Fewer nodes in DAG than objects provided.

### Pitfall 5: Sequence Dependency Extraction from nextval
**What goes wrong:** Table depends on sequence via `nextval('schema.seq_name'::regclass)` but the dependency is not detected.
**Why it happens:** The default value string format varies: `nextval('seq')`, `nextval('public.seq'::regclass)`, `nextval('"Seq"'::regclass)`.
**How to avoid:** Use a regex like `nextval\('([^']+)'` and normalize the extracted name (strip `::regclass`, handle schema prefix).
**Warning signs:** Sequences appear after tables in restore_order, causing runtime errors.

## Code Examples

### Kahn's Algorithm (Topological Sort)
```go
// Source: Standard algorithm, Go implementation
func (d *DAG) TopologicalSort() ([]string, error) {
    inDeg := make(map[string]int)
    for id := range d.nodes {
        inDeg[id] = d.inDegree[id]
    }

    // Collect initial zero-indegree nodes, sorted for determinism
    var queue []string
    for id, deg := range inDeg {
        if deg == 0 {
            queue = append(queue, id)
        }
    }
    sort.Strings(queue)

    var result []string
    for len(queue) > 0 {
        // Pop first (sorted order)
        node := queue[0]
        queue = queue[1:]
        result = append(result, node)

        // Find nodes that depend on this node (reverse adjacency)
        for id, deps := range d.edges {
            for _, dep := range deps {
                if dep == node {
                    inDeg[id]--
                    if inDeg[id] == 0 {
                        queue = append(queue, id)
                    }
                }
            }
        }
        sort.Strings(queue) // Re-sort for determinism
    }

    if len(result) != len(d.nodes) {
        // Cycle detected: remaining nodes form the cycle
        var cycleNodes []string
        for id, deg := range inDeg {
            if deg > 0 {
                cycleNodes = append(cycleNodes, id)
            }
        }
        sort.Strings(cycleNodes)
        return nil, fmt.Errorf("circular dependency detected among: %v", cycleNodes)
    }

    return result, nil
}
```

**Performance note:** The naive reverse-adjacency scan above is O(V*E). For the expected scale (~100 nodes), this is fine. If needed, maintain a reverse adjacency list for O(V+E). But premature optimization is unnecessary here.

### Manifest YAML Writing
```go
// Source: go.yaml.in/yaml/v3 standard usage
func WriteManifest(path string, m *Manifest) error {
    data, err := yaml.Marshal(m)
    if err != nil {
        return fmt.Errorf("marshal manifest: %w", err)
    }
    return os.WriteFile(path, data, 0644)
}
```

### nextval Sequence Extraction
```go
// Source: Project-specific pattern
var nextvalRe = regexp.MustCompile(`nextval\('([^']+)'`)

func extractSequenceDep(defaultExpr string) string {
    m := nextvalRe.FindStringSubmatch(defaultExpr)
    if m == nil {
        return ""
    }
    // Strip ::regclass suffix and schema-qualify if needed
    name := strings.TrimSuffix(m[1], "'::regclass")
    return name
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| gopkg.in/yaml.v3 | go.yaml.in/yaml/v3 | Project decision (Phase 1) | Same API, maintained fork |
| pg_dump dependency tracking | pg_depend catalog queries | Project decision | Direct control over dependency extraction |

**Deprecated/outdated:**
- None relevant -- Kahn's algorithm and DAG construction are timeless CS fundamentals

## Open Questions

1. **View/Matview dependency extraction without pg_depend**
   - What we know: ViewDef.Definition contains the SELECT body but no structured dependency info
   - What's unclear: Whether to add a pg_depend query in this phase or use conservative type-priority ordering
   - Recommendation: Use type-priority ordering for v1 (sequences -> enums -> types -> domains -> tables -> views/matviews -> functions -> triggers -> policies -> FKs). This is safe because views/matviews are created after all tables. If more precise ordering is needed later, add pg_depend in a future phase.

2. **Type/Domain/Enum dependency on column types**
   - What we know: Tables can use custom types, domains, and enums as column types
   - What's unclear: How to match ColumnDef.Type strings to known type/domain/enum ObjectDefs
   - Recommendation: Build a name set from fetched type/domain/enum objects. For each table column, check if its Type matches a known custom type name. This handles the common case. Complex cases (arrays of custom types, etc.) can be deferred.

3. **Manifest metadata (pg_version, tool_version)**
   - What we know: The manifest spec requires `pg_version` and `tool_version`
   - What's unclear: Where these values come from at Phase 6 (no orchestration yet)
   - Recommendation: ManifestWriter accepts these as parameters. The orchestration phase (Phase 7) will provide them.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) 1.24 |
| Config file | None needed (go test convention) |
| Quick run command | `go test ./internal/resolve/... -v -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DEPS-01 | DAG correctly built from ObjectDef slice (nodes + edges) | unit | `go test ./internal/resolve/ -run TestDAG -v -count=1` | Wave 0 |
| DEPS-02 | Kahn's sort produces correct topological order | unit | `go test ./internal/resolve/ -run TestTopologicalSort -v -count=1` | Wave 0 |
| DEPS-02 | Kahn's detects true (non-FK) circular dependency | unit | `go test ./internal/resolve/ -run TestCycleDetection -v -count=1` | Wave 0 |
| DEPS-03 | FK constraints excluded from DAG and appended after all tables | unit | `go test ./internal/resolve/ -run TestFKIsolation -v -count=1` | Wave 0 |
| DEPS-04 | _manifest.yaml output matches spec structure | unit | `go test ./internal/resolve/ -run TestManifest -v -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/resolve/... -v -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/resolve/dag_test.go` -- covers DEPS-01, DEPS-03
- [ ] `internal/resolve/kahn_test.go` -- covers DEPS-02
- [ ] `internal/resolve/manifest_test.go` -- covers DEPS-04

## Sources

### Primary (HIGH confidence)
- Project spec: `docs/pre/pgbackup_spec.md` sections 5.1, 6.1, 6.2 -- manifest format, dependency algorithm, FK handling
- Project source: `internal/core/types.go` -- all ObjectDef types including ForeignKeyDef
- Project source: `internal/core/interfaces.go` -- SchemaFetcher/Serializer/DDLGenerator interfaces
- Project source: `internal/core/registry.go` -- CommandRegistry pattern
- Project decisions: STATE.md -- FK constraint isolation, Kahn's algorithm selection

### Secondary (MEDIUM confidence)
- Kahn's algorithm: Standard CS textbook algorithm (Kahn 1962), well-known and deterministic

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - no new dependencies, uses existing go.yaml.in/yaml/v3
- Architecture: HIGH - project spec is explicit about DAG + Kahn's + FK isolation; ForeignKeyDef already exists
- Pitfalls: HIGH - standard graph algorithm pitfalls are well-documented in CS literature
- Dependency extraction: MEDIUM - view/matview dependency extraction without pg_depend is an open question

**Research date:** 2026-03-11
**Valid until:** 2026-04-11 (stable domain, no external dependency changes expected)
