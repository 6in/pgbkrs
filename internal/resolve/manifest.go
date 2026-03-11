package resolve

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/core"
	"go.yaml.in/yaml/v3"
)

// Manifest represents the _manifest.yaml structure per spec section 5.1.
type Manifest struct {
	BackupAt      string         `yaml:"backup_at"`
	PgVersion     string         `yaml:"pg_version"`
	ToolVersion   string         `yaml:"tool_version"`
	Snapshot      bool           `yaml:"snapshot"`
	SkippedTables []SkipEntry    `yaml:"skipped_tables"`
	Objects       []ObjectEntry  `yaml:"objects"`
	RestoreOrder  []RestoreEntry `yaml:"restore_order"`
}

// SkipEntry represents a table that was skipped during backup.
type SkipEntry struct {
	Schema string `yaml:"schema"`
	Name   string `yaml:"name"`
	Reason string `yaml:"reason"`
}

// ObjectEntry represents an object in the manifest's objects section.
type ObjectEntry struct {
	ID        string   `yaml:"id"`
	Kind      string   `yaml:"kind"`
	DependsOn []string `yaml:"depends_on"`
}

// RestoreEntry represents an entry in the manifest's restore_order section.
type RestoreEntry struct {
	Schema    string `yaml:"schema"`
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	FromTable string `yaml:"from_table,omitempty"`
}

// ManifestParams holds metadata input for BuildManifest.
type ManifestParams struct {
	BackupAt    string
	PgVersion   string
	ToolVersion string
	Snapshot    bool
	Skipped     []SkipEntry
}

// objectID returns "schema.name" format for an ObjectHeader (not "schema.kind.name").
func objectID(h core.ObjectHeader) string {
	return fmt.Sprintf("%s.%s", h.Schema, h.Name)
}

// nodeIDToObjectID converts "schema.kind.name" to "schema.name".
func nodeIDToObjectID(nodeID string) string {
	parts := strings.SplitN(nodeID, ".", 3)
	if len(parts) == 3 {
		return parts[0] + "." + parts[2]
	}
	return nodeID
}

// BuildManifest constructs a Manifest from a set of ObjectDef values and metadata params.
//
// Step A: Calls BuildRestoreOrder (from kahn.go) as the single authoritative source for restore order.
// Step B: Builds a separate DAG to read dependency edges for objects[].depends_on.
// Step C: Builds RestoreOrder slice from BuildRestoreOrder result.
// Step D: Assembles the final Manifest.
func BuildManifest(objects []core.ObjectDef, params ManifestParams) (*Manifest, error) {
	// Step A: Get restore order via BuildRestoreOrder
	restoreHeaders, err := BuildRestoreOrder(objects)
	if err != nil {
		return nil, err
	}

	// Separate FKs from non-FK objects
	var nonFK []core.ObjectDef
	var fkDefs []core.ForeignKeyDef
	for _, obj := range objects {
		h := obj.Header()
		if h.Kind == core.KindForeignKey {
			if fk, ok := obj.(core.ForeignKeyDef); ok {
				fkDefs = append(fkDefs, fk)
			} else if fk, ok := obj.(*core.ForeignKeyDef); ok {
				fkDefs = append(fkDefs, *fk)
			}
		} else {
			nonFK = append(nonFK, obj)
		}
	}

	// Step B: Build DAG to read dependency edges
	dag := NewDAG()
	for _, obj := range nonFK {
		dag.AddObject(obj)
	}
	dag.Resolve()

	// Build ObjectEntry list for non-FK objects
	var entries []ObjectEntry
	for _, obj := range nonFK {
		h := obj.Header()
		nid := NodeID(h)
		deps := dag.DepsOf(nid)

		// Convert dependency nodeIDs from "schema.kind.name" to "schema.name"
		depIDs := make([]string, 0, len(deps))
		for _, dep := range deps {
			depIDs = append(depIDs, nodeIDToObjectID(dep))
		}
		sort.Strings(depIDs)

		entries = append(entries, ObjectEntry{
			ID:        objectID(h),
			Kind:      string(h.Kind),
			DependsOn: depIDs,
		})
	}

	// Add FK objects to entries
	for _, fk := range fkDefs {
		h := fk.Header()
		// FK depends on source and target tables
		depIDs := []string{
			fmt.Sprintf("%s.%s", h.Schema, fk.SourceTable),
			fmt.Sprintf("%s.%s", fk.TargetSchema, fk.TargetTable),
		}
		sort.Strings(depIDs)
		// Deduplicate (in case source and target are the same table)
		deduped := make([]string, 0, len(depIDs))
		for i, d := range depIDs {
			if i == 0 || d != depIDs[i-1] {
				deduped = append(deduped, d)
			}
		}

		entries = append(entries, ObjectEntry{
			ID:        objectID(h),
			Kind:      "fk",
			DependsOn: deduped,
		})
	}

	// Step C: Build RestoreOrder from BuildRestoreOrder result
	// Build FK lookup by header for FromTable
	fkByKey := make(map[string]core.ForeignKeyDef)
	for _, fk := range fkDefs {
		h := fk.Header()
		key := fmt.Sprintf("%s.%s.%s", h.Schema, string(h.Kind), h.Name)
		fkByKey[key] = fk
	}

	restoreOrder := make([]RestoreEntry, 0, len(restoreHeaders))
	for _, h := range restoreHeaders {
		entry := RestoreEntry{
			Schema: h.Schema,
			Kind:   string(h.Kind),
			Name:   h.Name,
		}
		if h.Kind == core.KindForeignKey {
			entry.Kind = "fk"
			key := fmt.Sprintf("%s.%s.%s", h.Schema, string(h.Kind), h.Name)
			if fk, ok := fkByKey[key]; ok {
				entry.FromTable = fk.SourceTable
			}
		}
		restoreOrder = append(restoreOrder, entry)
	}

	// Step D: Assemble Manifest
	m := &Manifest{
		BackupAt:      params.BackupAt,
		PgVersion:     params.PgVersion,
		ToolVersion:   params.ToolVersion,
		Snapshot:      params.Snapshot,
		SkippedTables: params.Skipped,
		Objects:       entries,
		RestoreOrder:  restoreOrder,
	}

	return m, nil
}

// WriteManifest serializes a Manifest to YAML and writes it to the given path.
func WriteManifest(path string, m *Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
