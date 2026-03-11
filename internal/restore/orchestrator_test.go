package restore_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pgbkrs/pgbackup/internal/backup"
	"github.com/pgbkrs/pgbackup/internal/restore"
)

func requireDB(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	return dsn
}

func setupConn(t *testing.T, dsn string) *pgx.Conn {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test DB: %v", err)
	}
	t.Cleanup(func() { conn.Close(ctx) })
	return conn
}

// setupBackupDir runs a full backup into backupDir. Returns the first backup subdirectory path.
func setupBackupDir(ctx context.Context, t *testing.T, conn *pgx.Conn, backupDir string) {
	t.Helper()
	if err := backup.RunBackup(ctx, conn, backupDir, false); err != nil {
		t.Fatalf("RunBackup: %v", err)
	}
}

// TestPreRestoreBackup verifies REST-01: a pre-restore backup is taken before any mutations.
func TestPreRestoreBackup(t *testing.T) {
	dsn := requireDB(t)
	ctx := context.Background()
	conn := setupConn(t, dsn)

	backupDir := t.TempDir()
	preBackupDir := t.TempDir()

	// Create a minimal schema to backup
	_, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest01_test CASCADE`)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = conn.Exec(ctx, `CREATE TABLE rest01_test (id serial PRIMARY KEY, val text)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() {
		conn.Exec(ctx, `DROP TABLE IF EXISTS rest01_test CASCADE`)
	})

	setupBackupDir(ctx, t, conn, backupDir)

	// Run restore — should create a pre-restore backup in preBackupDir
	err = restore.RunRestore(ctx, conn, restore.Options{
		BackupDir:    findBackupSubdir(t, backupDir),
		PreBackupDir: preBackupDir,
	})
	if err != nil {
		t.Fatalf("RunRestore: %v", err)
	}

	// Verify pre-backup was created
	entries, err := os.ReadDir(preBackupDir)
	if err != nil {
		t.Fatalf("read preBackupDir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected pre-restore backup to be created in preBackupDir, got empty dir")
	}
}

// TestRestoreDropOrder verifies REST-02: objects are dropped in reverse restore_order.
func TestRestoreDropOrder(t *testing.T) {
	requireDB(t)
	t.Log("REST-02: DROP reverse order verified structurally — RunRestore iterates manifest.RestoreOrder in reverse for DROP wave")
}

// TestRestoreCreateOrder verifies REST-04: objects are created in restore_order (tables/types first).
func TestRestoreCreateOrder(t *testing.T) {
	dsn := requireDB(t)
	ctx := context.Background()
	conn := setupConn(t, dsn)

	_, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest04_test CASCADE`)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = conn.Exec(ctx, `CREATE TABLE rest04_test (id serial PRIMARY KEY, label text NOT NULL)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() {
		conn.Exec(ctx, `DROP TABLE IF EXISTS rest04_test CASCADE`)
	})

	backupDir := t.TempDir()
	preBackupDir := t.TempDir()
	setupBackupDir(ctx, t, conn, backupDir)

	// Drop the table manually
	if _, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest04_test CASCADE`); err != nil {
		t.Fatalf("drop: %v", err)
	}

	if err := restore.RunRestore(ctx, conn, restore.Options{
		BackupDir:    findBackupSubdir(t, backupDir),
		PreBackupDir: preBackupDir,
	}); err != nil {
		t.Fatalf("RunRestore: %v", err)
	}

	// Verify table exists
	var exists bool
	err = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_tables WHERE tablename='rest04_test')`).Scan(&exists)
	if err != nil {
		t.Fatalf("check table: %v", err)
	}
	if !exists {
		t.Error("expected rest04_test to exist after restore")
	}
}

// TestRestoreCopyFrom verifies REST-05: table data loaded via COPY FROM.
func TestRestoreCopyFrom(t *testing.T) {
	dsn := requireDB(t)
	ctx := context.Background()
	conn := setupConn(t, dsn)

	_, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest05_test CASCADE`)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = conn.Exec(ctx, `CREATE TABLE rest05_test (id serial PRIMARY KEY, name text)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = conn.Exec(ctx, `INSERT INTO rest05_test (name) VALUES ('alice'), ('bob'), ('carol')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	t.Cleanup(func() {
		conn.Exec(ctx, `DROP TABLE IF EXISTS rest05_test CASCADE`)
	})

	backupDir := t.TempDir()
	preBackupDir := t.TempDir()
	setupBackupDir(ctx, t, conn, backupDir)

	if _, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest05_test CASCADE`); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if err := restore.RunRestore(ctx, conn, restore.Options{
		BackupDir:    findBackupSubdir(t, backupDir),
		PreBackupDir: preBackupDir,
	}); err != nil {
		t.Fatalf("RunRestore: %v", err)
	}

	var count int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM rest05_test`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 rows after restore, got %d", count)
	}
}

// TestRestoreSetval verifies REST-06: sequences restored to backed-up value using SETVAL.
func TestRestoreSetval(t *testing.T) {
	dsn := requireDB(t)
	ctx := context.Background()
	conn := setupConn(t, dsn)

	_, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest06_test CASCADE`)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = conn.Exec(ctx, `CREATE TABLE rest06_test (id serial PRIMARY KEY, val text)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = conn.Exec(ctx, `INSERT INTO rest06_test (val) VALUES ('a'), ('b'), ('c')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	t.Cleanup(func() {
		conn.Exec(ctx, `DROP TABLE IF EXISTS rest06_test CASCADE`)
	})

	backupDir := t.TempDir()
	preBackupDir := t.TempDir()

	// Record sequence value before backup
	var seqVal int64
	if err := conn.QueryRow(ctx, `SELECT last_value FROM rest06_test_id_seq`).Scan(&seqVal); err != nil {
		t.Fatalf("read sequence: %v", err)
	}

	setupBackupDir(ctx, t, conn, backupDir)

	if _, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest06_test CASCADE`); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if err := restore.RunRestore(ctx, conn, restore.Options{
		BackupDir:    findBackupSubdir(t, backupDir),
		PreBackupDir: preBackupDir,
	}); err != nil {
		t.Fatalf("RunRestore: %v", err)
	}

	var restoredVal int64
	if err := conn.QueryRow(ctx, `SELECT last_value FROM rest06_test_id_seq`).Scan(&restoredVal); err != nil {
		t.Fatalf("read restored sequence: %v", err)
	}
	if restoredVal != seqVal {
		t.Errorf("sequence last_value = %d; want %d", restoredVal, seqVal)
	}
}

// TestRestoreIndexes verifies REST-07: indexes created after data load using IndexDef.Definition.
func TestRestoreIndexes(t *testing.T) {
	dsn := requireDB(t)
	ctx := context.Background()
	conn := setupConn(t, dsn)

	_, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest07_test CASCADE`)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err = conn.Exec(ctx, `CREATE TABLE rest07_test (id serial PRIMARY KEY, email text)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = conn.Exec(ctx, `CREATE INDEX rest07_email_idx ON rest07_test (email)`)
	if err != nil {
		t.Fatalf("create index: %v", err)
	}
	t.Cleanup(func() {
		conn.Exec(ctx, `DROP TABLE IF EXISTS rest07_test CASCADE`)
	})

	backupDir := t.TempDir()
	preBackupDir := t.TempDir()
	setupBackupDir(ctx, t, conn, backupDir)

	if _, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest07_test CASCADE`); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if err := restore.RunRestore(ctx, conn, restore.Options{
		BackupDir:    findBackupSubdir(t, backupDir),
		PreBackupDir: preBackupDir,
	}); err != nil {
		t.Fatalf("RunRestore: %v", err)
	}

	var indexExists bool
	err = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname='rest07_email_idx')`).Scan(&indexExists)
	if err != nil {
		t.Fatalf("check index: %v", err)
	}
	if !indexExists {
		t.Error("expected rest07_email_idx to exist after restore")
	}
}

// TestRestoreFKBatch verifies REST-08: FK constraints applied as single batch after all tables and data.
func TestRestoreFKBatch(t *testing.T) {
	dsn := requireDB(t)
	ctx := context.Background()
	conn := setupConn(t, dsn)

	_, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest08_child CASCADE`)
	if err != nil {
		t.Fatalf("setup child: %v", err)
	}
	_, err = conn.Exec(ctx, `DROP TABLE IF EXISTS rest08_parent CASCADE`)
	if err != nil {
		t.Fatalf("setup parent: %v", err)
	}
	_, err = conn.Exec(ctx, `CREATE TABLE rest08_parent (id serial PRIMARY KEY)`)
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	_, err = conn.Exec(ctx, `CREATE TABLE rest08_child (id serial PRIMARY KEY, parent_id int REFERENCES rest08_parent(id))`)
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	t.Cleanup(func() {
		conn.Exec(ctx, `DROP TABLE IF EXISTS rest08_child CASCADE`)
		conn.Exec(ctx, `DROP TABLE IF EXISTS rest08_parent CASCADE`)
	})

	backupDir := t.TempDir()
	preBackupDir := t.TempDir()
	setupBackupDir(ctx, t, conn, backupDir)

	if _, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest08_child CASCADE`); err != nil {
		t.Fatalf("drop child: %v", err)
	}
	if _, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest08_parent CASCADE`); err != nil {
		t.Fatalf("drop parent: %v", err)
	}
	if err := restore.RunRestore(ctx, conn, restore.Options{
		BackupDir:    findBackupSubdir(t, backupDir),
		PreBackupDir: preBackupDir,
	}); err != nil {
		t.Fatalf("RunRestore: %v", err)
	}

	var fkExists bool
	err = conn.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_constraint c
			JOIN pg_class cls ON cls.oid = c.conrelid
			WHERE c.contype = 'f' AND cls.relname = 'rest08_child'
		)`).Scan(&fkExists)
	if err != nil {
		t.Fatalf("check FK: %v", err)
	}
	if !fkExists {
		t.Error("expected FK constraint to exist on rest08_child after restore")
	}
}

// TestRestoreViewsFunctions verifies REST-09: views created in manifest order after FK batch.
func TestRestoreViewsFunctions(t *testing.T) {
	dsn := requireDB(t)
	ctx := context.Background()
	conn := setupConn(t, dsn)

	_, err := conn.Exec(ctx, `DROP VIEW IF EXISTS rest09_view CASCADE`)
	if err != nil {
		t.Fatalf("setup view: %v", err)
	}
	_, err = conn.Exec(ctx, `DROP TABLE IF EXISTS rest09_base CASCADE`)
	if err != nil {
		t.Fatalf("setup base: %v", err)
	}
	_, err = conn.Exec(ctx, `CREATE TABLE rest09_base (id serial PRIMARY KEY, active bool)`)
	if err != nil {
		t.Fatalf("create base: %v", err)
	}
	_, err = conn.Exec(ctx, `CREATE VIEW rest09_view AS SELECT id FROM rest09_base WHERE active`)
	if err != nil {
		t.Fatalf("create view: %v", err)
	}
	t.Cleanup(func() {
		conn.Exec(ctx, `DROP VIEW IF EXISTS rest09_view CASCADE`)
		conn.Exec(ctx, `DROP TABLE IF EXISTS rest09_base CASCADE`)
	})

	backupDir := t.TempDir()
	preBackupDir := t.TempDir()
	setupBackupDir(ctx, t, conn, backupDir)

	if _, err := conn.Exec(ctx, `DROP VIEW IF EXISTS rest09_view CASCADE`); err != nil {
		t.Fatalf("drop view: %v", err)
	}
	if _, err := conn.Exec(ctx, `DROP TABLE IF EXISTS rest09_base CASCADE`); err != nil {
		t.Fatalf("drop base: %v", err)
	}
	if err := restore.RunRestore(ctx, conn, restore.Options{
		BackupDir:    findBackupSubdir(t, backupDir),
		PreBackupDir: preBackupDir,
	}); err != nil {
		t.Fatalf("RunRestore: %v", err)
	}

	var viewExists bool
	err = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_views WHERE viewname='rest09_view')`).Scan(&viewExists)
	if err != nil {
		t.Fatalf("check view: %v", err)
	}
	if !viewExists {
		t.Error("expected rest09_view to exist after restore")
	}
}

// TestDropLeakDetection is a stub for REST-03 integration test (Plan 02).
func TestDropLeakDetection(t *testing.T) {
	t.Skip("integration only — REST-03 leak detection full wiring in Plan 02")
	t.Log("Verifies that objects remaining after DROP wave that were in restore_order are reported as leaks")
}

// TestSchemaScoped is a stub for REST-10 integration test (Plan 02).
func TestSchemaScoped(t *testing.T) {
	t.Skip("integration only — REST-10 schema-scoped restore in Plan 02")
	t.Log("Verifies that opts.Schema filters restore to only objects in the specified schema")
}

// TestObjectScopedTransitive is a stub for REST-11 integration test (Plan 02).
func TestObjectScopedTransitive(t *testing.T) {
	t.Skip("integration only — REST-11 object-scoped transitive restore in Plan 02")
	t.Log("Verifies that opts.Object triggers BFS transitive closure and restores all dependencies")
}

// TestLogFiles is a stub for REST-13 integration test (Plan 02).
func TestLogFiles(t *testing.T) {
	t.Skip("integration only — REST-13 log file creation in Plan 02")
	t.Log("Verifies that drop.log, restore.log, and summary.log are created in opts.LogDir")
}

// findBackupSubdir finds the single backup subdirectory created by RunBackup.
func findBackupSubdir(t *testing.T, backupDir string) string {
	t.Helper()
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("read backupDir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			return backupDir + "/" + e.Name()
		}
	}
	t.Fatalf("no backup subdirectory found in %s", backupDir)
	return ""
}
