package restore

import (
	"strings"
	"testing"

	"github.com/pgbkrs/pgbackup/internal/resolve"
)

// TestBuildTransitiveClosure_Direct: seed "public.a", which depends_on "public.b"; closure contains both.
func TestBuildTransitiveClosure_Direct(t *testing.T) {
	objects := []resolve.ObjectEntry{
		{ID: "public.a", Kind: "table", DependsOn: []string{"public.b"}},
		{ID: "public.b", Kind: "table", DependsOn: nil},
	}
	closure := buildTransitiveClosure(objects, []string{"public.a"}, nil)
	if !closure["public.a"] {
		t.Error("expected public.a in closure")
	}
	if !closure["public.b"] {
		t.Error("expected public.b in closure (direct dep)")
	}
}

// TestBuildTransitiveClosure_Transitive: "public.a" -> "public.b" -> "public.c"; closure contains all three.
func TestBuildTransitiveClosure_Transitive(t *testing.T) {
	objects := []resolve.ObjectEntry{
		{ID: "public.a", Kind: "table", DependsOn: []string{"public.b"}},
		{ID: "public.b", Kind: "table", DependsOn: []string{"public.c"}},
		{ID: "public.c", Kind: "table", DependsOn: nil},
	}
	closure := buildTransitiveClosure(objects, []string{"public.a"}, nil)
	for _, id := range []string{"public.a", "public.b", "public.c"} {
		if !closure[id] {
			t.Errorf("expected %s in closure", id)
		}
	}
}

// TestBuildTransitiveClosure_Cycle: "public.a" <-> "public.b"; BFS terminates; closure contains both.
func TestBuildTransitiveClosure_Cycle(t *testing.T) {
	objects := []resolve.ObjectEntry{
		{ID: "public.a", Kind: "table", DependsOn: []string{"public.b"}},
		{ID: "public.b", Kind: "table", DependsOn: []string{"public.a"}},
	}
	closure := buildTransitiveClosure(objects, []string{"public.a"}, nil)
	if !closure["public.a"] {
		t.Error("expected public.a in closure")
	}
	if !closure["public.b"] {
		t.Error("expected public.b in closure")
	}
}

// TestBuildTransitiveClosure_FKEntry: FK objects are not seeds; post-BFS pass adds FK entries
// whose full DependsOn set is a subset of the closure.
func TestBuildTransitiveClosure_FKEntry(t *testing.T) {
	objects := []resolve.ObjectEntry{
		{ID: "public.source", Kind: "table", DependsOn: nil},
		{ID: "public.target", Kind: "table", DependsOn: nil},
		{ID: "public.fk_source_target", Kind: "fk", DependsOn: []string{"public.source", "public.target"}},
		{ID: "public.orphan_fk", Kind: "fk", DependsOn: []string{"public.other", "public.source"}},
	}
	// BFS seed is only "public.source" - should not follow fk forward
	closure := buildTransitiveClosure(objects, []string{"public.source"}, nil)
	if !closure["public.source"] {
		t.Error("expected public.source in closure")
	}
	// FK whose ALL deps (source and target) are in closure — but target is not seeded
	// so fk_source_target should NOT be in closure (target missing)
	if closure["public.fk_source_target"] {
		t.Error("expected public.fk_source_target NOT in closure (public.target is not in closure)")
	}
	// orphan_fk should not be in closure (public.other not in closure)
	if closure["public.orphan_fk"] {
		t.Error("expected public.orphan_fk NOT in closure")
	}

	// Now seed both source and target: FK should be included
	closureBoth := buildTransitiveClosure(objects, []string{"public.source", "public.target"}, nil)
	if !closureBoth["public.fk_source_target"] {
		t.Error("expected public.fk_source_target in closure when both deps are present")
	}
	if closureBoth["public.orphan_fk"] {
		t.Error("expected public.orphan_fk NOT in closure (public.other missing)")
	}
}

// TestMissingDependency: verifyDepsPresent returns error for missing IDs, nil when all present.
func TestMissingDependency(t *testing.T) {
	objects := []resolve.ObjectEntry{
		{ID: "public.a", Kind: "table"},
		{ID: "public.b", Kind: "table"},
	}

	// Closure that references a non-existent ID
	closureWithMissing := map[string]bool{
		"public.a": true,
		"public.c": true, // not in objects
	}
	err := verifyDepsPresent(objects, closureWithMissing)
	if err == nil {
		t.Error("expected error for missing dep public.c, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "public.c") {
		t.Errorf("expected error to mention public.c, got: %v", err)
	}

	// Closure with all IDs present
	closureOK := map[string]bool{
		"public.a": true,
		"public.b": true,
	}
	if err := verifyDepsPresent(objects, closureOK); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestFilteredRestoreOrder_Full: empty Schema and Object returns full manifest.RestoreOrder unchanged.
func TestFilteredRestoreOrder_Full(t *testing.T) {
	manifest := &resolve.Manifest{
		Objects: []resolve.ObjectEntry{
			{ID: "public.a", Kind: "table"},
			{ID: "public.b", Kind: "table"},
		},
		RestoreOrder: []resolve.RestoreEntry{
			{Schema: "public", Kind: "table", Name: "a"},
			{Schema: "public", Kind: "table", Name: "b"},
		},
	}
	opts := Options{}
	result, err := filteredRestoreOrder(manifest, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
}

// TestFilteredRestoreOrder_Schema: opts.Schema="myschema" returns only entries in that schema.
func TestFilteredRestoreOrder_Schema(t *testing.T) {
	manifest := &resolve.Manifest{
		Objects: []resolve.ObjectEntry{
			{ID: "myschema.a", Kind: "table"},
			{ID: "public.b", Kind: "table"},
		},
		RestoreOrder: []resolve.RestoreEntry{
			{Schema: "myschema", Kind: "table", Name: "a"},
			{Schema: "public", Kind: "table", Name: "b"},
			{Schema: "myschema", Kind: "table", Name: "c"},
		},
	}
	opts := Options{Schema: "myschema"}
	result, err := filteredRestoreOrder(manifest, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries for myschema, got %d", len(result))
	}
	for _, e := range result {
		if e.Schema != "myschema" {
			t.Errorf("expected schema myschema, got %s", e.Schema)
		}
	}
}

// TestFilteredRestoreOrder_Object: opts.Object="public.a" with transitive dep "public.b"
// returns both entries in original manifest order.
func TestFilteredRestoreOrder_Object(t *testing.T) {
	manifest := &resolve.Manifest{
		Objects: []resolve.ObjectEntry{
			{ID: "public.a", Kind: "table", DependsOn: []string{"public.b"}},
			{ID: "public.b", Kind: "table", DependsOn: nil},
			{ID: "public.c", Kind: "table", DependsOn: nil},
		},
		RestoreOrder: []resolve.RestoreEntry{
			{Schema: "public", Kind: "table", Name: "b"},
			{Schema: "public", Kind: "table", Name: "a"},
			{Schema: "public", Kind: "table", Name: "c"},
		},
	}
	opts := Options{Object: "public.a"}
	result, err := filteredRestoreOrder(manifest, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
	// Original manifest order: b before a
	if result[0].Name != "b" || result[1].Name != "a" {
		t.Errorf("expected order [b, a] (original manifest order), got [%s, %s]", result[0].Name, result[1].Name)
	}
}

// TestFilteredRestoreOrder_InvalidObject: opts.Object="notvalid" (no dot) returns error.
func TestFilteredRestoreOrder_InvalidObject(t *testing.T) {
	manifest := &resolve.Manifest{}
	opts := Options{Object: "notvalid"}
	_, err := filteredRestoreOrder(manifest, opts)
	if err == nil {
		t.Error("expected error for invalid object format, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "--object must be in schema.name format") {
		t.Errorf("expected error to mention format requirement, got: %v", err)
	}
}
