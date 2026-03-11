package restore

import (
	"fmt"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/resolve"
)

// Options holds parameters for a restore operation.
type Options struct {
	BackupDir    string
	PreBackupDir string
	Schema       string // REST-10: filter to schema (e.g. "myschema")
	Object       string // REST-10: filter to "schema.name"; implies BFS (REST-11)
	LogDir       string // REST-13: parent dir for restore_YYYYMMDD_HHMMSS/ logs
}

// buildTransitiveClosure computes the set of all objects reachable from seeds via BFS.
// After BFS, a post-BFS pass adds FK entries whose full DependsOn set is a subset of the closure.
func buildTransitiveClosure(objects []resolve.ObjectEntry, seeds []string) map[string]bool {
	// Build byID map
	byID := make(map[string]resolve.ObjectEntry, len(objects))
	for _, obj := range objects {
		byID[obj.ID] = obj
	}

	visited := make(map[string]bool)
	queue := make([]string, 0, len(seeds))

	// Enqueue seeds (non-FK only for BFS seeding)
	for _, s := range seeds {
		if !visited[s] {
			visited[s] = true
			queue = append(queue, s)
		}
	}

	// Standard BFS — only follow non-FK edges
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		obj, ok := byID[cur]
		if !ok {
			continue
		}
		// Don't follow forward from FK entries
		if obj.Kind == "fk" {
			continue
		}
		for _, dep := range obj.DependsOn {
			if !visited[dep] {
				visited[dep] = true
				queue = append(queue, dep)
			}
		}
	}

	// Post-BFS pass: add FK entries whose ALL deps are in the closure
	for _, obj := range objects {
		if obj.Kind != "fk" {
			continue
		}
		if visited[obj.ID] {
			continue
		}
		allPresent := true
		for _, dep := range obj.DependsOn {
			if !visited[dep] {
				allPresent = false
				break
			}
		}
		if allPresent && len(obj.DependsOn) > 0 {
			visited[obj.ID] = true
		}
	}

	return visited
}

// verifyDepsPresent checks that every ID in the closure is present in manifest objects.
// Returns an error naming the first missing ID.
func verifyDepsPresent(objects []resolve.ObjectEntry, closure map[string]bool) error {
	present := make(map[string]bool, len(objects))
	for _, obj := range objects {
		present[obj.ID] = true
	}
	for id := range closure {
		if !present[id] {
			return fmt.Errorf("dependency %q is required but not present in backup", id)
		}
	}
	return nil
}

// filteredRestoreOrder returns the subset of manifest.RestoreOrder matching opts.
// If both Schema and Object are empty, returns the full restore order unchanged.
func filteredRestoreOrder(manifest *resolve.Manifest, opts Options) ([]resolve.RestoreEntry, error) {
	// Validate Object format
	if opts.Object != "" && !strings.Contains(opts.Object, ".") {
		return nil, fmt.Errorf("--object must be in schema.name format, got %q", opts.Object)
	}

	if opts.Object != "" {
		// BFS closure from the named object
		closure := buildTransitiveClosure(manifest.Objects, []string{opts.Object})
		if err := verifyDepsPresent(manifest.Objects, closure); err != nil {
			return nil, err
		}
		// Filter keeping entries whose schema.name is in closure, in original manifest order
		var result []resolve.RestoreEntry
		for _, entry := range manifest.RestoreOrder {
			id := entry.Schema + "." + entry.Name
			if closure[id] {
				result = append(result, entry)
			}
		}
		return result, nil
	}

	if opts.Schema != "" {
		var result []resolve.RestoreEntry
		for _, entry := range manifest.RestoreOrder {
			if entry.Schema == opts.Schema {
				result = append(result, entry)
			}
		}
		return result, nil
	}

	return manifest.RestoreOrder, nil
}
