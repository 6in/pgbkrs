package datadiff

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// formatDataReport writes the data diff summary in a format consistent with schema diff output.
func formatDataReport(results []TableDiffResult, backupA, backupB string, w io.Writer) {
	fmt.Fprintf(w, "比較(データ): %s → %s\n", filepath.Base(backupA), filepath.Base(backupB))

	var changed, nopk, errored []TableDiffResult
	for _, r := range results {
		switch {
		case r.Err != nil:
			errored = append(errored, r)
		case r.NoPK:
			nopk = append(nopk, r)
		case r.Added > 0 || r.Removed > 0 || r.Changed > 0:
			changed = append(changed, r)
		}
	}

	byKey := func(s []TableDiffResult) func(i, j int) bool {
		return func(i, j int) bool {
			return s[i].Schema+"."+s[i].Name < s[j].Schema+"."+s[j].Name
		}
	}
	sort.Slice(changed, byKey(changed))
	sort.Slice(nopk, byKey(nopk))
	sort.Slice(errored, byKey(errored))

	if len(changed) == 0 && len(nopk) == 0 && len(errored) == 0 {
		fmt.Fprintf(w, "\n(差分なし)\n")
		return
	}

	if len(changed) > 0 {
		fmt.Fprintf(w, "\n[データ変更テーブル]\n")
		for _, r := range changed {
			var parts []string
			if r.Added > 0 {
				parts = append(parts, fmt.Sprintf("追加 %d行", r.Added))
			}
			if r.Removed > 0 {
				parts = append(parts, fmt.Sprintf("削除 %d行", r.Removed))
			}
			if r.Changed > 0 {
				parts = append(parts, fmt.Sprintf("変更 %d行", r.Changed))
			}
			fmt.Fprintf(w, "  %s.%s: %s\n", r.Schema, r.Name, strings.Join(parts, ", "))
		}
	}

	if len(nopk) > 0 {
		fmt.Fprintf(w, "\n[主キーなし - データ変更あり]\n")
		for _, r := range nopk {
			fmt.Fprintf(w, "  %s.%s\n", r.Schema, r.Name)
		}
	}

	if len(errored) > 0 {
		fmt.Fprintf(w, "\n[エラー]\n")
		for _, r := range errored {
			fmt.Fprintf(w, "  %s.%s: %v\n", r.Schema, r.Name, r.Err)
		}
	}
}
