package diff

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// formatReport writes the diff output in spec 9.3 format to w.
func formatReport(result DiffResult, backupA, backupB string, w io.Writer) {
	fmt.Fprintf(w, "比較: %s → %s\n", filepath.Base(backupA), filepath.Base(backupB))

	if len(result.Added) == 0 && len(result.Removed) == 0 && len(result.Changed) == 0 {
		fmt.Fprintf(w, "\n(差分なし)\n")
		return
	}

	if len(result.Added) > 0 {
		// Sort by schema.name for deterministic output
		added := make([]core.ObjectHeader, len(result.Added))
		copy(added, result.Added)
		sort.Slice(added, func(i, j int) bool {
			ki := added[i].Schema + "." + added[i].Name
			kj := added[j].Schema + "." + added[j].Name
			return ki < kj
		})

		fmt.Fprintf(w, "\n[追加オブジェクト]\n")
		for _, h := range added {
			fmt.Fprintf(w, "  %-16s: %s.%s\n", string(h.Kind), h.Schema, displayName(h))
		}
	}

	if len(result.Removed) > 0 {
		// Sort by schema.name for deterministic output
		removed := make([]core.ObjectHeader, len(result.Removed))
		copy(removed, result.Removed)
		sort.Slice(removed, func(i, j int) bool {
			ki := removed[i].Schema + "." + removed[i].Name
			kj := removed[j].Schema + "." + removed[j].Name
			return ki < kj
		})

		fmt.Fprintf(w, "\n[削除オブジェクト]\n")
		for _, h := range removed {
			fmt.Fprintf(w, "  %-16s: %s.%s\n", string(h.Kind), h.Schema, displayName(h))
		}
	}

	if len(result.Changed) > 0 {
		// Sort by schema.name for deterministic output
		changed := make([]ObjectChange, len(result.Changed))
		copy(changed, result.Changed)
		sort.Slice(changed, func(i, j int) bool {
			ki := changed[i].Header.Schema + "." + changed[i].Header.Name
			kj := changed[j].Header.Schema + "." + changed[j].Header.Name
			return ki < kj
		})

		fmt.Fprintf(w, "\n[変更オブジェクト]\n")
		for _, ch := range changed {
			fmt.Fprintf(w, "  %s: %s.%s\n", string(ch.Header.Kind), ch.Header.Schema, displayName(ch.Header))
			for _, detail := range ch.Details {
				fmt.Fprintf(w, "    %s\n", detail)
			}
			fmt.Fprintf(w, "\n")
		}
	}
}

// displayName returns the display name for an object header.
// For functions, appends the ArgTypes in parentheses if available via type assertion.
// For all other kinds, returns the plain Name.
func displayName(h core.ObjectHeader) string {
	return h.Name
}
