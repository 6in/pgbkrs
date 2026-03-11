package restore

import (
	"testing"
)

// TestDetectLeaks_NoLeaks: postDrop contains objects that are NOT in shouldBeDropped;
// detectLeaks returns empty slice (no leaks when remaining objects were never targeted for drop).
func TestDetectLeaks_NoLeaks(t *testing.T) {
	postDrop := []liveObject{
		{Schema: "public", Kind: "table", Name: "users"},
		{Schema: "public", Kind: "table", Name: "orders"},
	}
	// Neither users nor orders were targeted for drop — they were pre-existing
	shouldBeDropped := map[string]bool{}
	leaks := detectLeaks(postDrop, shouldBeDropped)
	if len(leaks) != 0 {
		t.Errorf("expected no leaks, got %d: %v", len(leaks), leaks)
	}
}

// TestDetectLeaks_WithLeak: object in postDrop AND in shouldBeDropped → it's a leak.
func TestDetectLeaks_WithLeak(t *testing.T) {
	postDrop := []liveObject{
		{Schema: "public", Kind: "table", Name: "users"},
		{Schema: "public", Kind: "table", Name: "orphan"},
	}
	shouldBeDropped := map[string]bool{
		"public.users":  true,
		"public.orphan": true,
	}
	leaks := detectLeaks(postDrop, shouldBeDropped)
	if len(leaks) != 2 {
		t.Errorf("expected 2 leaks (both present in shouldBeDropped), got %d", len(leaks))
	}
}

// TestDetectLeaks_PreExisting: object in postDrop but NOT in shouldBeDropped → not a leak.
func TestDetectLeaks_PreExisting(t *testing.T) {
	postDrop := []liveObject{
		{Schema: "public", Kind: "table", Name: "preexisting"},
		{Schema: "public", Kind: "table", Name: "orphan"},
	}
	// preexisting is NOT in shouldBeDropped (it was there before the restore, not in restore_order)
	// orphan IS in shouldBeDropped
	shouldBeDropped := map[string]bool{
		"public.orphan": true,
	}
	leaks := detectLeaks(postDrop, shouldBeDropped)
	if len(leaks) != 1 {
		t.Errorf("expected 1 leak (orphan), got %d: %v", len(leaks), leaks)
	}
	if len(leaks) == 1 && leaks[0].Name != "orphan" {
		t.Errorf("expected leaked object 'orphan', got '%s'", leaks[0].Name)
	}
}
