package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/resolve"
)

// writeAIReadme generates a README.md at backupRoot summarising the database
// structure in a format optimised for AI consumption.
func writeAIReadme(backupRoot, dbName, pgVersion, backupAt string, allObjects []core.ObjectDef, skipped []resolve.SkipEntry) error {
	var b strings.Builder

	fmt.Fprintf(&b, "# pgbackup AI Export — %s\n\n", dbName)
	fmt.Fprintf(&b, "**Backed up at:** %s  \n", backupAt)
	fmt.Fprintf(&b, "**PostgreSQL version:** %s  \n", pgVersion)
	fmt.Fprintf(&b, "**Tool version:** %s\n\n", toolVersion)

	b.WriteString("## How to Use This Backup\n\n")
	b.WriteString("This backup was created with `pgbackup backup-ai` and is optimised for reading with an AI assistant.\n\n")
	b.WriteString("- **Schema definitions** (`def.yaml`): Full schema for each object\n")
	b.WriteString("- **Sample data** (`data.csv`): Up to 10 rows per table, including a header row with column names\n")
	b.WriteString("- **Manifest** (`_manifest.yaml`): Full object inventory and restore order\n\n")
	b.WriteString("**Suggested reading order:** Start with this README, then `_manifest.yaml`, then explore schema directories.\n\n")

	// Collect tables and FKs from allObjects.
	var tables []*core.TableDef
	var fks []*core.ForeignKeyDef
	schemaSet := make(map[string]struct{})

	for _, obj := range allObjects {
		h := obj.Header()
		schemaSet[h.Schema] = struct{}{}
		switch v := obj.(type) {
		case *core.TableDef:
			tables = append(tables, v)
		case *core.ForeignKeyDef:
			fks = append(fks, v)
		}
	}

	// Schemas section.
	b.WriteString("## Schemas\n\n")
	for schema := range schemaSet {
		fmt.Fprintf(&b, "- %s\n", schema)
	}
	b.WriteString("\n")

	// Tables section.
	fmt.Fprintf(&b, "## Tables (%d)\n\n", len(tables))
	if len(tables) > 0 {
		b.WriteString("| Schema | Table | Columns | Primary Key | Sample Rows |\n")
		b.WriteString("|--------|-------|---------|-------------|-------------|\n")
		for _, td := range tables {
			pkCols := ""
			if td.Constraints.PrimaryKey != nil {
				pkCols = strings.Join(td.Constraints.PrimaryKey.Columns, ", ")
			}
			sampleRows := int64(0)
			if td.DataMeta != nil {
				sampleRows = td.DataMeta.RowCount
			}
			fmt.Fprintf(&b, "| %s | %s | %d | %s | %d |\n",
				td.Schema, td.Name, len(td.Columns), pkCols, sampleRows)
		}
		b.WriteString("\n")
	}

	// Foreign key relationships.
	if len(fks) > 0 {
		fmt.Fprintf(&b, "## Foreign Key Relationships (%d)\n\n", len(fks))
		b.WriteString("| Schema | Constraint | Source Table | Target Schema | Target Table | Definition |\n")
		b.WriteString("|--------|-----------|-------------|---------------|--------------|------------|\n")
		for _, fk := range fks {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n",
				fk.Schema, fk.Name, fk.SourceTable, fk.TargetSchema, fk.TargetTable, fk.Definition)
		}
		b.WriteString("\n")
	}

	// Skipped tables.
	if len(skipped) > 0 {
		fmt.Fprintf(&b, "## Skipped Tables (%d)\n\n", len(skipped))
		b.WriteString("| Schema | Table | Reason |\n")
		b.WriteString("|--------|-------|--------|\n")
		for _, s := range skipped {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", s.Schema, s.Name, s.Reason)
		}
		b.WriteString("\n")
	}

	return os.WriteFile(filepath.Join(backupRoot, "README.md"), []byte(b.String()), 0644)
}
