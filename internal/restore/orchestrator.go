package restore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/pgbkrs/pgbackup/internal/backup"
	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/resolve"
)

// RunRestore executes a full restore from opts.BackupDir to the connected database.
// opts.PreBackupDir is the directory where the pre-restore safety backup will be written.
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
func RunRestore(ctx context.Context, conn *pgx.Conn, opts Options) error {
	// Resolve log directory default
	logDir := opts.LogDir
	if logDir == "" {
		logDir = "."
	}

	// Initialize logger
	logger, err := NewLogger(logDir)
	if err != nil {
		// Non-fatal: log to stderr and continue without file logging
		fmt.Fprintf(os.Stderr, "restore: failed to create logger: %v\n", err)
		logger = nil
	}

	var restoreErr error
	startTime := time.Now()
	defer func() {
		if logger != nil {
			elapsed := time.Since(startTime)
			logger.WriteSummary(logger.dropCount, logger.restoreCount, logger.warnCount, elapsed, restoreErr)
			_ = logger.Close()
		}
	}()

	// Wave 0: Pre-restore backup (REST-01)
	if err := backup.RunBackup(ctx, conn, opts.PreBackupDir, false); err != nil {
		restoreErr = fmt.Errorf("pre-restore backup: %w", err)
		return restoreErr
	}

	// Load manifest
	manifestPath := filepath.Join(opts.BackupDir, "_manifest.yaml")
	manifest, err := resolve.ReadManifest(manifestPath)
	if err != nil {
		restoreErr = fmt.Errorf("load manifest: %w", err)
		return restoreErr
	}

	// Determine the scoped restore order
	restoreOrder, err := filteredRestoreOrder(manifest, opts)
	if err != nil {
		restoreErr = fmt.Errorf("filter restore order: %w", err)
		return restoreErr
	}

	// Build shouldBeDropped set from restore order
	shouldBeDropped := make(map[string]bool, len(restoreOrder))
	for _, entry := range restoreOrder {
		shouldBeDropped[entry.Schema+"."+entry.Name] = true
	}

	// Snapshot live objects before DROP wave (REST-03 leak detection stub)
	// Collect unique schemas from restoreOrder
	schemaSet := make(map[string]bool)
	for _, entry := range restoreOrder {
		schemaSet[entry.Schema] = true
	}
	schemas := make([]string, 0, len(schemaSet))
	for s := range schemaSet {
		schemas = append(schemas, s)
	}

	preDrop, snapErr := snapshotLiveObjects(ctx, conn, schemas)
	if snapErr != nil {
		fmt.Fprintf(os.Stderr, "restore: pre-drop snapshot failed (continuing): %v\n", snapErr)
		preDrop = nil
	}

	// Wave 1: DROP in reverse restore_order (REST-02)
	for i := len(restoreOrder) - 1; i >= 0; i-- {
		entry := restoreOrder[i]
		if err := execDrop(ctx, conn, opts.BackupDir, entry); err != nil {
			restoreErr = fmt.Errorf("drop %s %s.%s: %w", entry.Kind, entry.Schema, entry.Name, err)
			return restoreErr
		}
		if logger != nil {
			logger.LogDrop(entry.Schema, entry.Kind, entry.Name, nil)
		}
	}

	// Snapshot after DROP wave and detect leaks (REST-03)
	if preDrop != nil {
		postDrop, snapErr := snapshotLiveObjects(ctx, conn, schemas)
		if snapErr != nil {
			fmt.Fprintf(os.Stderr, "restore: post-drop snapshot failed (continuing): %v\n", snapErr)
		} else {
			leaks := detectLeaks(postDrop, shouldBeDropped)
			for _, leak := range leaks {
				fmt.Fprintf(os.Stderr, "restore: LEAK detected: %s.%s (kind=%s)\n", leak.Schema, leak.Name, leak.Kind)
				if logger != nil {
					logger.LogLeakWarning(leak.Schema, leak.Kind, leak.Name)
				}
			}
		}
	}

	// Wave 1.5: CREATE SCHEMA IF NOT EXISTS for all schemas in restoreOrder
	// (required when schemas were dropped or never existed on the target DB)
	createdSchemas := make(map[string]bool)
	for _, entry := range restoreOrder {
		if createdSchemas[entry.Schema] {
			continue
		}
		createdSchemas[entry.Schema] = true
		if _, err := conn.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, entry.Schema)); err != nil {
			restoreErr = fmt.Errorf("create schema %s: %w", entry.Schema, err)
			return restoreErr
		}
	}

	// Wave 2: CREATE schema-only objects (tables, sequences, types, domains, enums) (REST-04)
	schemaWaveKinds := map[string]bool{
		"table": true, "sequence": true, "type": true, "domain": true, "enum": true,
	}
	for _, entry := range restoreOrder {
		if !schemaWaveKinds[entry.Kind] {
			continue
		}
		def, err := loadObjectDef(opts.BackupDir, entry)
		if err != nil {
			restoreErr = fmt.Errorf("load def for %s.%s: %w", entry.Schema, entry.Name, err)
			return restoreErr
		}
		gen := ddlGeneratorFor(entry.Kind)
		if gen == nil {
			continue
		}
		stmts, err := gen.GenerateDDL(def)
		if err != nil {
			restoreErr = fmt.Errorf("generate DDL for %s.%s: %w", entry.Schema, entry.Name, err)
			return restoreErr
		}
		// For tables: GenerateDDL returns CREATE TABLE + possibly index statements.
		// In Wave 2, execute only the first statement (CREATE TABLE).
		// Indexes are handled separately in Wave 5.
		if entry.Kind == "table" {
			if len(stmts) > 0 {
				if _, err := conn.Exec(ctx, stmts[0]); err != nil {
					restoreErr = fmt.Errorf("create table %s.%s: %w", entry.Schema, entry.Name, err)
					return restoreErr
				}
			}
			if logger != nil {
				logger.LogRestore(entry.Schema, entry.Kind, entry.Name, nil)
			}
			continue
		}
		for _, stmt := range stmts {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				restoreErr = fmt.Errorf("exec DDL %s.%s: %w", entry.Schema, entry.Name, err)
				return restoreErr
			}
		}
		if logger != nil {
			logger.LogRestore(entry.Schema, entry.Kind, entry.Name, nil)
		}
	}

	// Wave 3: COPY FROM for table data (REST-05)
	for _, entry := range restoreOrder {
		if entry.Kind != "table" {
			continue
		}
		if err := copyFromTable(ctx, conn, opts.BackupDir, entry); err != nil {
			restoreErr = fmt.Errorf("copy from %s.%s: %w", entry.Schema, entry.Name, err)
			return restoreErr
		}
	}

	// Wave 4: SETVAL for sequences after COPY FROM (REST-06)
	for _, entry := range restoreOrder {
		if entry.Kind != "sequence" {
			continue
		}
		def, err := loadObjectDef(opts.BackupDir, entry)
		if err != nil {
			restoreErr = fmt.Errorf("load sequence def %s.%s: %w", entry.Schema, entry.Name, err)
			return restoreErr
		}
		sd, ok := def.(core.SequenceDef)
		if !ok {
			restoreErr = fmt.Errorf("expected SequenceDef for %s.%s, got %T", entry.Schema, entry.Name, def)
			return restoreErr
		}
		// Skip setval for sequences that were never used (LastValue=0, IsCalled=false).
		// PostgreSQL sequences have a minimum of 1; setval(seq, 0) is invalid.
		if sd.LastValue == 0 && !sd.IsCalled {
			continue
		}
		sql := fmt.Sprintf(`SELECT setval('%s.%s', %d, %v)`,
			sd.Schema, sd.Name, sd.LastValue, sd.IsCalled)
		if _, err := conn.Exec(ctx, sql); err != nil {
			restoreErr = fmt.Errorf("setval %s.%s: %w", entry.Schema, entry.Name, err)
			return restoreErr
		}
	}

	// Wave 5: CREATE non-PK indexes from IndexDef.Definition (REST-07)
	for _, entry := range restoreOrder {
		if entry.Kind != "table" {
			continue
		}
		def, err := loadObjectDef(opts.BackupDir, entry)
		if err != nil {
			restoreErr = fmt.Errorf("load table def for indexes %s.%s: %w", entry.Schema, entry.Name, err)
			return restoreErr
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
				restoreErr = fmt.Errorf("create index %s on %s.%s: %w", idx.Name, entry.Schema, entry.Name, err)
				return restoreErr
			}
		}
	}

	// Wave 6: FK batch — all "fk" entries (REST-08)
	for _, entry := range restoreOrder {
		if entry.Kind != "fk" {
			continue
		}
		if err := execFKCreate(ctx, conn, opts.BackupDir, entry); err != nil {
			restoreErr = fmt.Errorf("apply FK %s.%s: %w", entry.Schema, entry.Name, err)
			return restoreErr
		}
	}

	// Wave 7: Views, materialized views, functions, triggers, policies (REST-09)
	remainderKinds := map[string]bool{
		"view": true, "materialized_view": true, "function": true, "trigger": true, "policy": true,
	}
	for _, entry := range restoreOrder {
		if !remainderKinds[entry.Kind] {
			continue
		}
		def, err := loadObjectDef(opts.BackupDir, entry)
		if err != nil {
			restoreErr = fmt.Errorf("load def for %s.%s: %w", entry.Schema, entry.Name, err)
			return restoreErr
		}
		gen := ddlGeneratorFor(entry.Kind)
		if gen == nil {
			continue
		}
		stmts, err := gen.GenerateDDL(def)
		if err != nil {
			restoreErr = fmt.Errorf("generate DDL for %s.%s: %w", entry.Schema, entry.Name, err)
			return restoreErr
		}
		for _, stmt := range stmts {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				restoreErr = fmt.Errorf("exec DDL %s.%s: %w", entry.Schema, entry.Name, err)
				return restoreErr
			}
		}
		if logger != nil {
			logger.LogRestore(entry.Schema, entry.Kind, entry.Name, nil)
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

// execFKCreate loads the FK definition from the backup directory and executes
// the ALTER TABLE ... ADD CONSTRAINT DDL for the given FK entry.
func execFKCreate(ctx context.Context, conn *pgx.Conn, backupDir string, entry resolve.RestoreEntry) error {
	def, err := loadObjectDef(backupDir, entry)
	if err != nil {
		return fmt.Errorf("load FK def %s.%s: %w", entry.Schema, entry.Name, err)
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
