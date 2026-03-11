package restore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/pgbkrs/pgbackup/internal/backup"
	"github.com/pgbkrs/pgbackup/internal/core"
	fetchForeignkey "github.com/pgbkrs/pgbackup/internal/fetch/foreignkey"
	"github.com/pgbkrs/pgbackup/internal/resolve"
)

// RunRestore executes a full restore from backupDir to the connected database.
// preBackupDir is the directory where the pre-restore safety backup will be written.
//
// Pipeline waves:
//
//	0. Pre-restore backup (REST-01)
//	1. DROP in reverse restore_order (REST-02)
//	2. CREATE schema objects in restore_order: tables, sequences, types, domains, enums (REST-04)
//	3. COPY FROM for table data and partition children (REST-05)
//	4. SETVAL for sequences after data load (REST-06)
//	5. CREATE non-PK indexes from IndexDef.Definition (REST-07)
//	6. ADD FK constraints as a batch (REST-08)
//	7. CREATE views, mat views, functions, triggers, policies (REST-09)
func RunRestore(ctx context.Context, conn *pgx.Conn, backupDir, preBackupDir string) error {
	// Wave 0: Pre-restore backup (REST-01)
	if err := backup.RunBackup(ctx, conn, preBackupDir, false); err != nil {
		return fmt.Errorf("pre-restore backup: %w", err)
	}

	// Load manifest
	manifestPath := filepath.Join(backupDir, "_manifest.yaml")
	manifest, err := resolve.ReadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// Wave 1: DROP in reverse restore_order (REST-02)
	for i := len(manifest.RestoreOrder) - 1; i >= 0; i-- {
		entry := manifest.RestoreOrder[i]
		if err := execDrop(ctx, conn, backupDir, entry); err != nil {
			return fmt.Errorf("drop %s %s.%s: %w", entry.Kind, entry.Schema, entry.Name, err)
		}
	}

	// Wave 2: CREATE schema-only objects (tables, sequences, types, domains, enums) (REST-04)
	schemaWaveKinds := map[string]bool{
		"table": true, "sequence": true, "type": true, "domain": true, "enum": true,
	}
	for _, entry := range manifest.RestoreOrder {
		if !schemaWaveKinds[entry.Kind] {
			continue
		}
		def, err := loadObjectDef(backupDir, entry)
		if err != nil {
			return fmt.Errorf("load def for %s.%s: %w", entry.Schema, entry.Name, err)
		}
		gen := ddlGeneratorFor(entry.Kind)
		if gen == nil {
			continue
		}
		stmts, err := gen.GenerateDDL(def)
		if err != nil {
			return fmt.Errorf("generate DDL for %s.%s: %w", entry.Schema, entry.Name, err)
		}
		// For tables: GenerateDDL returns CREATE TABLE + possibly index statements.
		// In Wave 2, execute only the first statement (CREATE TABLE).
		// Indexes are handled separately in Wave 5.
		if entry.Kind == "table" {
			if len(stmts) > 0 {
				if _, err := conn.Exec(ctx, stmts[0]); err != nil {
					return fmt.Errorf("create table %s.%s: %w", entry.Schema, entry.Name, err)
				}
			}
			continue
		}
		for _, stmt := range stmts {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("exec DDL %s.%s: %w", entry.Schema, entry.Name, err)
			}
		}
	}

	// Wave 3: COPY FROM for table data (REST-05)
	for _, entry := range manifest.RestoreOrder {
		if entry.Kind != "table" {
			continue
		}
		if err := copyFromTable(ctx, conn, backupDir, entry); err != nil {
			return fmt.Errorf("copy from %s.%s: %w", entry.Schema, entry.Name, err)
		}
	}

	// Wave 4: SETVAL for sequences after COPY FROM (REST-06)
	for _, entry := range manifest.RestoreOrder {
		if entry.Kind != "sequence" {
			continue
		}
		def, err := loadObjectDef(backupDir, entry)
		if err != nil {
			return fmt.Errorf("load sequence def %s.%s: %w", entry.Schema, entry.Name, err)
		}
		sd, ok := def.(core.SequenceDef)
		if !ok {
			return fmt.Errorf("expected SequenceDef for %s.%s, got %T", entry.Schema, entry.Name, def)
		}
		sql := fmt.Sprintf(`SELECT setval('%s.%s', %d, %v)`,
			sd.Schema, sd.Name, sd.LastValue, sd.IsCalled)
		if _, err := conn.Exec(ctx, sql); err != nil {
			return fmt.Errorf("setval %s.%s: %w", entry.Schema, entry.Name, err)
		}
	}

	// Wave 5: CREATE non-PK indexes from IndexDef.Definition (REST-07)
	for _, entry := range manifest.RestoreOrder {
		if entry.Kind != "table" {
			continue
		}
		def, err := loadObjectDef(backupDir, entry)
		if err != nil {
			return fmt.Errorf("load table def for indexes %s.%s: %w", entry.Schema, entry.Name, err)
		}
		td, ok := def.(*core.TableDef)
		if !ok {
			continue
		}
		for _, idx := range td.Indexes {
			if idx.Definition == "" {
				continue
			}
			if _, err := conn.Exec(ctx, idx.Definition); err != nil {
				return fmt.Errorf("create index %s on %s.%s: %w", idx.Name, entry.Schema, entry.Name, err)
			}
		}
	}

	// Wave 6: FK batch — all "fk" entries (REST-08)
	for _, entry := range manifest.RestoreOrder {
		if entry.Kind != "fk" {
			continue
		}
		if err := execFKCreate(ctx, conn, entry); err != nil {
			return fmt.Errorf("apply FK %s.%s: %w", entry.Schema, entry.Name, err)
		}
	}

	// Wave 7: Views, materialized views, functions, triggers, policies (REST-09)
	remainderKinds := map[string]bool{
		"view": true, "materialized_view": true, "function": true, "trigger": true, "policy": true,
	}
	for _, entry := range manifest.RestoreOrder {
		if !remainderKinds[entry.Kind] {
			continue
		}
		def, err := loadObjectDef(backupDir, entry)
		if err != nil {
			return fmt.Errorf("load def for %s.%s: %w", entry.Schema, entry.Name, err)
		}
		gen := ddlGeneratorFor(entry.Kind)
		if gen == nil {
			continue
		}
		stmts, err := gen.GenerateDDL(def)
		if err != nil {
			return fmt.Errorf("generate DDL for %s.%s: %w", entry.Schema, entry.Name, err)
		}
		for _, stmt := range stmts {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("exec DDL %s.%s: %w", entry.Schema, entry.Name, err)
			}
		}
	}

	return nil
}

// execDrop generates and executes a DROP statement for the given restore entry.
// For FK entries (no def.yaml), builds a minimal ForeignKeyDef.
// For all other entries, loads def.yaml and calls the DDL generator.
func execDrop(ctx context.Context, conn *pgx.Conn, backupDir string, entry resolve.RestoreEntry) error {
	if entry.Kind == "fk" {
		// FK has no def.yaml; build minimal ForeignKeyDef for GenerateDrop
		fkDef := &core.ForeignKeyDef{
			ObjectHeader: core.ObjectHeader{
				Kind:   core.KindForeignKey,
				Schema: entry.Schema,
				Name:   entry.Name,
			},
			SourceTable: entry.FromTable,
		}
		gen := ddlGeneratorFor("fk")
		stmts, err := gen.GenerateDrop(fkDef)
		if err != nil {
			return fmt.Errorf("generate drop for FK %s: %w", entry.Name, err)
		}
		for _, stmt := range stmts {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("drop FK %s: %w", entry.Name, err)
			}
		}
		return nil
	}

	def, err := loadObjectDef(backupDir, entry)
	if err != nil {
		// Object may not exist in backup dir — skip gracefully
		return nil
	}
	gen := ddlGeneratorFor(entry.Kind)
	if gen == nil {
		return nil
	}
	stmts, err := gen.GenerateDrop(def)
	if err != nil {
		return fmt.Errorf("generate drop for %s.%s: %w", entry.Schema, entry.Name, err)
	}
	for _, stmt := range stmts {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("exec drop %s.%s: %w", entry.Schema, entry.Name, err)
		}
	}
	return nil
}

// copyFromTable opens the table's data.csv and streams it via COPY FROM STDIN.
func copyFromTable(ctx context.Context, conn *pgx.Conn, backupDir string, entry resolve.RestoreEntry) error {
	def, err := loadObjectDef(backupDir, entry)
	if err != nil {
		return nil // partition child with no def.yaml is skipped
	}
	td, ok := def.(*core.TableDef)
	if !ok || td.DataMeta == nil || td.DataMeta.File == "" {
		return nil // no data to load (partitioned parent or table without data export)
	}

	// Determine the table directory (matches backup layout)
	var tableDir string
	if entry.FromTable != "" {
		tableDir = filepath.Join(backupDir, entry.Schema, "tables", entry.FromTable, "partitions", entry.Name)
	} else {
		tableDir = filepath.Join(backupDir, entry.Schema, "tables", entry.Name)
	}
	csvPath := filepath.Join(tableDir, td.DataMeta.File)

	f, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open data file %s: %w", csvPath, err)
	}
	defer f.Close()

	cols := quotedColList(td.DataMeta.Columns)
	sql := fmt.Sprintf(`COPY "%s"."%s" (%s) FROM STDIN WITH (FORMAT CSV)`,
		entry.Schema, entry.Name, cols)
	if _, err := conn.PgConn().CopyFrom(ctx, f, sql); err != nil {
		return fmt.Errorf("COPY FROM %s.%s: %w", entry.Schema, entry.Name, err)
	}
	return nil
}

// quotedColList returns a comma-separated list of double-quoted column names.
func quotedColList(cols []string) string {
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = `"` + c + `"`
	}
	return strings.Join(quoted, ", ")
}

// execFKCreate re-fetches FK definitions from the live database (post-table-creation)
// and executes the ALTER TABLE ... ADD CONSTRAINT DDL for the given FK entry.
func execFKCreate(ctx context.Context, conn *pgx.Conn, entry resolve.RestoreEntry) error {
	// FKs are not serialized to disk. Re-fetch from the database post-table-creation.
	fetcher := &fetchForeignkey.SchemaFetcher{}
	defs, err := fetcher.Fetch(ctx, conn, entry.Schema)
	if err != nil {
		return fmt.Errorf("fetch FKs for schema %s: %w", entry.Schema, err)
	}
	for _, def := range defs {
		h := def.Header()
		if h.Name != entry.Name {
			continue
		}
		gen := ddlGeneratorFor("fk")
		stmts, err := gen.GenerateDDL(def)
		if err != nil {
			return fmt.Errorf("generate FK DDL %s: %w", entry.Name, err)
		}
		for _, stmt := range stmts {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("exec FK DDL %s: %w", entry.Name, err)
			}
		}
		return nil
	}
	return fmt.Errorf("FK %s.%s not found in database after table creation", entry.Schema, entry.Name)
}
