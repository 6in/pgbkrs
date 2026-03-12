package diff

import (
	"fmt"
	"sort"

	"github.com/pgbkrs/pgbackup/internal/core"
	"go.yaml.in/yaml/v3"
)

// DiffResult holds the schema comparison output.
type DiffResult struct {
	Added   []core.ObjectHeader
	Removed []core.ObjectHeader
	Changed []ObjectChange
}

// ObjectChange describes a schema object that exists in both backups but differs.
type ObjectChange struct {
	Header  core.ObjectHeader
	Details []string // human-readable change lines per spec 9.3
}

// compareObjects compares two object maps and returns a DiffResult.
func compareObjects(a, b map[string]core.ObjectDef) DiffResult {
	var result DiffResult

	// Detect added (in b but not in a)
	for key, bObj := range b {
		if _, ok := a[key]; !ok {
			result.Added = append(result.Added, bObj.Header())
		}
	}

	// Detect removed (in a but not in b)
	for key, aObj := range a {
		if _, ok := b[key]; !ok {
			result.Removed = append(result.Removed, aObj.Header())
		}
	}

	// Detect changed (in both with different content)
	for key, aObj := range a {
		bObj, ok := b[key]
		if !ok {
			continue
		}
		details := diffObject(aObj, bObj)
		if len(details) > 0 {
			result.Changed = append(result.Changed, ObjectChange{
				Header:  aObj.Header(),
				Details: details,
			})
		}
	}

	// Sort Added and Removed by key for deterministic output
	sort.Slice(result.Added, func(i, j int) bool {
		ki := result.Added[i].Schema + "." + result.Added[i].Name
		kj := result.Added[j].Schema + "." + result.Added[j].Name
		return ki < kj
	})
	sort.Slice(result.Removed, func(i, j int) bool {
		ki := result.Removed[i].Schema + "." + result.Removed[i].Name
		kj := result.Removed[j].Schema + "." + result.Removed[j].Name
		return ki < kj
	})
	sort.Slice(result.Changed, func(i, j int) bool {
		ki := result.Changed[i].Header.Schema + "." + result.Changed[i].Header.Name
		kj := result.Changed[j].Header.Schema + "." + result.Changed[j].Header.Name
		return ki < kj
	})

	// Post-processing: move trigger/policy Added/Removed entries into their table's Changed.Details
	result = moveTriggerPolicyToTable(result, a, b)

	return result
}

// moveTriggerPolicyToTable moves trigger and policy Added/Removed entries into
// their owning table's Changed.Details when the table exists in both backups.
func moveTriggerPolicyToTable(result DiffResult, a, b map[string]core.ObjectDef) DiffResult {
	// Build a map of tables present in both a and b
	tablesInBoth := make(map[string]bool)
	for key := range a {
		if _, ok := b[key]; ok {
			tablesInBoth[key] = true
		}
	}

	// Helper to find or create a Changed entry for a table
	changedByKey := make(map[string]*ObjectChange)
	for i := range result.Changed {
		h := result.Changed[i].Header
		k := h.Schema + "." + string(h.Kind) + "." + h.Name
		changedByKey[k] = &result.Changed[i]
	}

	var newAdded []core.ObjectHeader
	for _, h := range result.Added {
		tableKey := ""
		label := ""
		switch h.Kind {
		case core.KindTrigger:
			// Find the trigger def from b
			trigKey := h.Schema + ".trigger." + h.Name
			if def, ok := b[trigKey]; ok {
				td := def.(core.TriggerDef)
				tableKey = h.Schema + ".table." + td.TableName
				label = fmt.Sprintf("トリガー追加: %s", h.Name)
			}
		case core.KindPolicy:
			// Find the policy def from b
			polKey := h.Schema + ".policy." + h.Name
			if def, ok := b[polKey]; ok {
				pd := def.(core.PolicyDef)
				tableKey = h.Schema + ".table." + pd.TableName
				label = fmt.Sprintf("ポリシー追加: %s", h.Name)
			}
		}

		if tableKey != "" && tablesInBoth[tableKey] && label != "" {
			// Move to table's Changed.Details
			if ch, ok := changedByKey[tableKey]; ok {
				ch.Details = append(ch.Details, label)
			} else {
				// Create new Changed entry for the table
				tableObj := a[tableKey]
				newChange := ObjectChange{
					Header:  tableObj.Header(),
					Details: []string{label},
				}
				result.Changed = append(result.Changed, newChange)
				changedByKey[tableKey] = &result.Changed[len(result.Changed)-1]
			}
			// Don't add to newAdded
			continue
		}
		newAdded = append(newAdded, h)
	}
	result.Added = newAdded

	var newRemoved []core.ObjectHeader
	for _, h := range result.Removed {
		tableKey := ""
		label := ""
		switch h.Kind {
		case core.KindTrigger:
			// Find the trigger def from a
			trigKey := h.Schema + ".trigger." + h.Name
			if def, ok := a[trigKey]; ok {
				td := def.(core.TriggerDef)
				tableKey = h.Schema + ".table." + td.TableName
				label = fmt.Sprintf("トリガー削除: %s", h.Name)
			}
		case core.KindPolicy:
			// Find the policy def from a
			polKey := h.Schema + ".policy." + h.Name
			if def, ok := a[polKey]; ok {
				pd := def.(core.PolicyDef)
				tableKey = h.Schema + ".table." + pd.TableName
				label = fmt.Sprintf("ポリシー削除: %s", h.Name)
			}
		}

		if tableKey != "" && tablesInBoth[tableKey] && label != "" {
			if ch, ok := changedByKey[tableKey]; ok {
				ch.Details = append(ch.Details, label)
			} else {
				tableObj := a[tableKey]
				newChange := ObjectChange{
					Header:  tableObj.Header(),
					Details: []string{label},
				}
				result.Changed = append(result.Changed, newChange)
				changedByKey[tableKey] = &result.Changed[len(result.Changed)-1]
			}
			continue
		}
		newRemoved = append(newRemoved, h)
	}
	result.Removed = newRemoved

	// Re-sort Changed after any additions
	sort.Slice(result.Changed, func(i, j int) bool {
		ki := result.Changed[i].Header.Schema + "." + result.Changed[i].Header.Name
		kj := result.Changed[j].Header.Schema + "." + result.Changed[j].Header.Name
		return ki < kj
	})

	return result
}

// diffObject compares two ObjectDef values and returns human-readable change lines.
// Returns nil if no differences.
func diffObject(a, b core.ObjectDef) []string {
	// Try value type assertion for TableDef
	aTable, aIsTable := a.(core.TableDef)
	bTable, bIsTable := b.(core.TableDef)
	if aIsTable && bIsTable {
		return diffTable(&aTable, &bTable)
	}
	// Try pointer type assertion for TableDef
	aPtrTable, aIsPtrTable := a.(*core.TableDef)
	bPtrTable, bIsPtrTable := b.(*core.TableDef)
	if aIsPtrTable && bIsPtrTable {
		return diffTable(aPtrTable, bPtrTable)
	}
	// Non-table path
	return diffNonTable(a, b)
}

// diffTable compares two TableDef values and returns human-readable change lines.
func diffTable(a, b *core.TableDef) []string {
	var details []string

	// Column comparison
	details = append(details, diffColumns(a.Columns, b.Columns)...)

	// Index comparison
	details = append(details, diffIndexes(a.Indexes, b.Indexes)...)

	// Unique constraint comparison
	details = append(details, diffUniqueConstraints(a.Constraints.Unique, b.Constraints.Unique)...)

	// Check constraint comparison
	details = append(details, diffCheckConstraints(a.Constraints.Check, b.Constraints.Check)...)

	// RLS comparison
	if a.RLS.Enabled != b.RLS.Enabled {
		details = append(details, fmt.Sprintf("RLS変更    : enabled %v → %v", a.RLS.Enabled, b.RLS.Enabled))
	}

	return details
}

// diffColumns compares column slices and returns human-readable change lines.
func diffColumns(aCols, bCols []core.ColumnDef) []string {
	var details []string

	aByName := make(map[string]core.ColumnDef)
	for _, c := range aCols {
		aByName[c.Name] = c
	}
	bByName := make(map[string]core.ColumnDef)
	for _, c := range bCols {
		bByName[c.Name] = c
	}

	// Added columns (in b but not in a) — in order of b
	for _, bc := range bCols {
		if _, ok := aByName[bc.Name]; !ok {
			nullStr := ""
			if bc.Nullable {
				nullStr = " nullable"
			}
			details = append(details, fmt.Sprintf("カラム追加  : %s %s%s", bc.Name, bc.Type, nullStr))
		}
	}

	// Removed columns (in a but not in b) — in order of a
	for _, ac := range aCols {
		if _, ok := bByName[ac.Name]; !ok {
			details = append(details, fmt.Sprintf("カラム削除  : %s", ac.Name))
		}
	}

	// Changed columns
	for _, ac := range aCols {
		bc, ok := bByName[ac.Name]
		if !ok {
			continue
		}
		if ac.Type != bc.Type {
			details = append(details, fmt.Sprintf("カラム変更  : %s type %s → %s", ac.Name, ac.Type, bc.Type))
		}
		if ac.Nullable != bc.Nullable {
			details = append(details, fmt.Sprintf("カラム変更  : %s nullable %v → %v", ac.Name, ac.Nullable, bc.Nullable))
		}
		if ac.Default != bc.Default {
			details = append(details, fmt.Sprintf("カラム変更  : %s default %q → %q", ac.Name, ac.Default, bc.Default))
		}
	}

	return details
}

// diffIndexes compares index slices and returns human-readable change lines.
func diffIndexes(aIdxs, bIdxs []core.IndexDef) []string {
	var details []string

	aByName := make(map[string]core.IndexDef)
	for _, idx := range aIdxs {
		aByName[idx.Name] = idx
	}
	bByName := make(map[string]core.IndexDef)
	for _, idx := range bIdxs {
		bByName[idx.Name] = idx
	}

	// Added indexes (in b but not in a)
	for _, bidx := range bIdxs {
		if _, ok := aByName[bidx.Name]; !ok {
			details = append(details, fmt.Sprintf("インデックス追加: %s", bidx.Name))
		}
	}

	// Removed indexes (in a but not in b)
	for _, aidx := range aIdxs {
		if _, ok := bByName[aidx.Name]; !ok {
			details = append(details, fmt.Sprintf("インデックス削除: %s", aidx.Name))
		}
	}

	// Changed indexes: same name but different definition = remove + add
	for _, aidx := range aIdxs {
		bidx, ok := bByName[aidx.Name]
		if !ok {
			continue
		}
		if aidx.Definition != bidx.Definition {
			details = append(details, fmt.Sprintf("インデックス削除: %s", aidx.Name))
			details = append(details, fmt.Sprintf("インデックス追加: %s", bidx.Name))
		}
	}

	return details
}

// diffUniqueConstraints compares unique constraint slices.
func diffUniqueConstraints(aU, bU []core.UniqueConstraintDef) []string {
	var details []string

	aByName := make(map[string]bool)
	for _, u := range aU {
		aByName[u.Name] = true
	}
	bByName := make(map[string]bool)
	for _, u := range bU {
		bByName[u.Name] = true
	}

	for _, bu := range bU {
		if !aByName[bu.Name] {
			details = append(details, fmt.Sprintf("UNIQUE追加  : %s", bu.Name))
		}
	}
	for _, au := range aU {
		if !bByName[au.Name] {
			details = append(details, fmt.Sprintf("UNIQUE削除  : %s", au.Name))
		}
	}

	return details
}

// diffCheckConstraints compares check constraint slices.
func diffCheckConstraints(aC, bC []core.CheckConstraintDef) []string {
	var details []string

	aByName := make(map[string]bool)
	for _, c := range aC {
		aByName[c.Name] = true
	}
	bByName := make(map[string]bool)
	for _, c := range bC {
		bByName[c.Name] = true
	}

	for _, bc := range bC {
		if !aByName[bc.Name] {
			details = append(details, fmt.Sprintf("CHECK追加   : %s", bc.Name))
		}
	}
	for _, ac := range aC {
		if !bByName[ac.Name] {
			details = append(details, fmt.Sprintf("CHECK削除   : %s", ac.Name))
		}
	}

	return details
}

// diffNonTable compares two non-table ObjectDef values via serializer byte comparison.
// Returns ["本体変更あり"] for functions/triggers, ["定義変更あり"] for views/matviews,
// ["設定変更あり"] for sequences/types/domains/enums/policies/fks.
// Returns nil if no differences.
func diffNonTable(a, b core.ObjectDef) []string {
	aHeader := a.Header()
	bHeader := b.Header()

	// Use YAML marshal for byte comparison (avoids needing a serializer lookup by ObjectDef kind)
	aBytes, err := yaml.Marshal(a)
	if err != nil {
		// Fall back to fmt.Sprintf comparison
		aBytes = []byte(fmt.Sprintf("%v", a))
	}
	bBytes, err := yaml.Marshal(b)
	if err != nil {
		bBytes = []byte(fmt.Sprintf("%v", b))
	}

	if string(aBytes) == string(bBytes) {
		return nil
	}

	_ = bHeader
	switch aHeader.Kind {
	case core.KindFunction, core.KindTrigger:
		return []string{"本体変更あり"}
	case core.KindView, core.KindMaterializedView:
		return []string{"定義変更あり"}
	default:
		return []string{"設定変更あり"}
	}
}
