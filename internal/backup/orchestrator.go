package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/pgbkrs/pgbackup/internal/core"
	"github.com/pgbkrs/pgbackup/internal/export"
	fetchCompositetype "github.com/pgbkrs/pgbackup/internal/fetch/compositetype"
	fetchDomain "github.com/pgbkrs/pgbackup/internal/fetch/domain"
	fetchEnum "github.com/pgbkrs/pgbackup/internal/fetch/enum"
	fetchForeignkey "github.com/pgbkrs/pgbackup/internal/fetch/foreignkey"
	fetchFunction "github.com/pgbkrs/pgbackup/internal/fetch/function"
	fetchMatview "github.com/pgbkrs/pgbackup/internal/fetch/matview"
	fetchPolicy "github.com/pgbkrs/pgbackup/internal/fetch/policy"
	fetchSequence "github.com/pgbkrs/pgbackup/internal/fetch/sequence"
	fetchTable "github.com/pgbkrs/pgbackup/internal/fetch/table"
	fetchTrigger "github.com/pgbkrs/pgbackup/internal/fetch/trigger"
	fetchView "github.com/pgbkrs/pgbackup/internal/fetch/view"
	"github.com/pgbkrs/pgbackup/internal/resolve"
	serializeCompositetype "github.com/pgbkrs/pgbackup/internal/serialize/compositetype"
	serializeDomain "github.com/pgbkrs/pgbackup/internal/serialize/domain"
	serializeEnum "github.com/pgbkrs/pgbackup/internal/serialize/enum"
	serializeFunction "github.com/pgbkrs/pgbackup/internal/serialize/function"
	serializeMatview "github.com/pgbkrs/pgbackup/internal/serialize/matview"
	serializePolicy "github.com/pgbkrs/pgbackup/internal/serialize/policy"
	serializeSequence "github.com/pgbkrs/pgbackup/internal/serialize/sequence"
	serializeTable "github.com/pgbkrs/pgbackup/internal/serialize/table"
	serializeTrigger "github.com/pgbkrs/pgbackup/internal/serialize/trigger"
	serializeView "github.com/pgbkrs/pgbackup/internal/serialize/view"
)

const toolVersion = "0.1.0"

// kindToDir maps ObjectKind to the subdirectory name under a schema directory.
// This implements the spec section 4 directory layout.
var kindToDir = map[core.ObjectKind]string{
	core.KindTable:            "tables",
	core.KindView:             "views",
	core.KindMaterializedView: "materialized_views",
	core.KindFunction:         "functions",
	core.KindSequence:         "sequences",
	core.KindTrigger:          "triggers",
	core.KindType:             "types",
	core.KindDomain:           "domains",
	core.KindEnum:             "enums",
	core.KindPolicy:           "policies",
}

// kindFetcher pairs an ObjectKind with its corresponding SchemaFetcher.
type kindFetcher struct {
	kind    core.ObjectKind
	fetcher core.SchemaFetcher
}

// RunBackup executes a full backup of all user schemas to outDir.
// snapshot=false: each fetch/export uses conn directly.
// snapshot=true: TODO(Plan 02): snapshot mode — wrap in RepeatableRead transaction.
func RunBackup(ctx context.Context, conn *pgx.Conn, outDir string, snapshot bool) error {
	// 1. Create timestamped backup root
	backupRoot := filepath.Join(outDir, time.Now().UTC().Format("backup_20060102_150405"))
	if err := os.MkdirAll(backupRoot, 0755); err != nil {
		return fmt.Errorf("create backup root: %w", err)
	}

	// 2. Fetch pg version
	var pgVersion string
	if err := conn.QueryRow(ctx, "SELECT current_setting('server_version')").Scan(&pgVersion); err != nil {
		return fmt.Errorf("fetch pg_version: %w", err)
	}

	// 3. Discover user schemas
	schemas, err := discoverSchemas(ctx, conn)
	if err != nil {
		return fmt.Errorf("discover schemas: %w", err)
	}

	// 4. All fetchers in iteration order
	fetchers := []kindFetcher{
		{core.KindTable, &fetchTable.SchemaFetcher{}},
		{core.KindView, &fetchView.SchemaFetcher{}},
		{core.KindMaterializedView, &fetchMatview.SchemaFetcher{}},
		{core.KindFunction, &fetchFunction.SchemaFetcher{}},
		{core.KindSequence, &fetchSequence.SchemaFetcher{}},
		{core.KindTrigger, &fetchTrigger.SchemaFetcher{}},
		{core.KindType, &fetchCompositetype.SchemaFetcher{}},
		{core.KindDomain, &fetchDomain.SchemaFetcher{}},
		{core.KindEnum, &fetchEnum.SchemaFetcher{}},
		{core.KindPolicy, &fetchPolicy.SchemaFetcher{}},
		{core.KindForeignKey, &fetchForeignkey.SchemaFetcher{}},
	}

	var allObjects []core.ObjectDef
	var skipped []resolve.SkipEntry

	for _, schema := range schemas {
		// First pass: fetch tables to build childOf map for partition routing.
		rawTables, err := (&fetchTable.SchemaFetcher{}).Fetch(ctx, conn, schema)
		if err != nil {
			return fmt.Errorf("fetch tables in schema %s: %w", schema, err)
		}
		childOf := buildChildOf(rawTables)

		for _, kf := range fetchers {
			if kf.kind == core.KindTable {
				// Tables already fetched above — process each one.
				for _, def := range rawTables {
					td, ok := def.(*core.TableDef)
					if !ok {
						continue
					}

					reason := skipReason(td)
					if reason != "" {
						fmt.Fprintf(os.Stderr, "WARNING: skipping table %s.%s: %s\n", schema, td.Name, reason)
						skipped = append(skipped, resolve.SkipEntry{Schema: schema, Name: td.Name, Reason: reason})
						continue
					}

					tableDir := tableObjectDir(backupRoot, schema, td, childOf)
					if err := os.MkdirAll(tableDir, 0755); err != nil {
						return fmt.Errorf("create table dir: %w", err)
					}

					// Export data for non-partitioned tables only.
					if export.ShouldExportData(td) {
						cols := columnNames(td)
						csvPath := filepath.Join(tableDir, "data.csv")
						meta, err := export.ExportTableData(ctx, conn, schema, td.Name, csvPath, cols)
						if err != nil {
							return fmt.Errorf("export %s.%s: %w", schema, td.Name, err)
						}
						td.DataMeta = &core.TableDataMeta{
							File:     "data.csv",
							Columns:  meta.Columns,
							RowCount: meta.RowCount,
							Checksum: meta.Checksum,
						}
					}

					// Serialize def.yaml.
					s := &serializeTable.Serializer{}
					data, err := s.Serialize(td)
					if err != nil {
						return fmt.Errorf("serialize table %s.%s: %w", schema, td.Name, err)
					}
					if err := os.WriteFile(filepath.Join(tableDir, "def.yaml"), data, 0644); err != nil {
						return fmt.Errorf("write def.yaml for %s.%s: %w", schema, td.Name, err)
					}

					allObjects = append(allObjects, td)
				}
				continue
			}

			// Non-table objects
			defs, err := kf.fetcher.Fetch(ctx, conn, schema)
			if err != nil {
				return fmt.Errorf("fetch %s in schema %s: %w", kf.kind, schema, err)
			}
			for _, def := range defs {
				// ForeignKey objects are tracked for manifest but not written as standalone YAML files.
				if kf.kind != core.KindForeignKey {
					s := serializerFor(kf.kind)
					if s != nil {
						b, err := s.Serialize(def)
						if err != nil {
							return fmt.Errorf("serialize %s %s: %w", kf.kind, def.Header().Name, err)
						}
						dir := filepath.Join(backupRoot, schema, kindToDir[kf.kind])
						if err := os.MkdirAll(dir, 0755); err != nil {
							return fmt.Errorf("create dir %s: %w", dir, err)
						}
						if err := os.WriteFile(filepath.Join(dir, def.Header().Name+".yaml"), b, 0644); err != nil {
							return fmt.Errorf("write yaml for %s: %w", def.Header().Name, err)
						}
					}
				}
				allObjects = append(allObjects, def)
			}
		}
	}

	// 5. Build and write manifest.
	params := resolve.ManifestParams{
		BackupAt:    time.Now().UTC().Format(time.RFC3339),
		PgVersion:   pgVersion,
		ToolVersion: toolVersion,
		Snapshot:    snapshot,
		Skipped:     skipped,
	}
	manifest, err := resolve.BuildManifest(allObjects, params)
	if err != nil {
		return fmt.Errorf("build manifest: %w", err)
	}
	if err := resolve.WriteManifest(filepath.Join(backupRoot, "_manifest.yaml"), manifest); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

// discoverSchemas returns all user schemas, excluding PostgreSQL system namespaces.
func discoverSchemas(ctx context.Context, conn *pgx.Conn) ([]string, error) {
	const q = `
		SELECT nspname
		FROM pg_namespace
		WHERE nspname NOT IN ('pg_catalog', 'information_schema')
		  AND nspname NOT LIKE 'pg_%'
		ORDER BY nspname
	`
	rows, err := conn.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query schemas: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan schema: %w", err)
		}
		schemas = append(schemas, s)
	}
	return schemas, rows.Err()
}

// buildChildOf constructs a map from child table name to parent table name
// by inspecting Partitioning.Children on each table definition.
func buildChildOf(defs []core.ObjectDef) map[string]string {
	childOf := make(map[string]string)
	for _, def := range defs {
		td, ok := def.(*core.TableDef)
		if !ok || td.Partitioning == nil {
			continue
		}
		for _, child := range td.Partitioning.Children {
			childOf[child] = td.Name
		}
	}
	return childOf
}

// tableObjectDir returns the directory path for a table's backup artifacts,
// routing partition children into <parent>/partitions/<child>/ per spec section 4.
func tableObjectDir(backupRoot, schema string, td *core.TableDef, childOf map[string]string) string {
	if parent, ok := childOf[td.Name]; ok {
		return filepath.Join(backupRoot, schema, "tables", parent, "partitions", td.Name)
	}
	return filepath.Join(backupRoot, schema, "tables", td.Name)
}

// columnNames returns an ordered slice of column names for a table.
func columnNames(td *core.TableDef) []string {
	names := make([]string, 0, len(td.Columns))
	for _, col := range td.Columns {
		names = append(names, col.Name)
	}
	return names
}

// serializerFor returns the appropriate Serializer for the given ObjectKind.
// Returns nil for ForeignKey and unknown kinds.
func serializerFor(kind core.ObjectKind) core.Serializer {
	switch kind {
	case core.KindView:
		return &serializeView.Serializer{}
	case core.KindMaterializedView:
		return &serializeMatview.Serializer{}
	case core.KindFunction:
		return &serializeFunction.Serializer{}
	case core.KindSequence:
		return &serializeSequence.Serializer{}
	case core.KindTrigger:
		return &serializeTrigger.Serializer{}
	case core.KindType:
		return &serializeCompositetype.Serializer{}
	case core.KindDomain:
		return &serializeDomain.Serializer{}
	case core.KindEnum:
		return &serializeEnum.Serializer{}
	case core.KindPolicy:
		return &serializePolicy.Serializer{}
	default:
		return nil
	}
}
