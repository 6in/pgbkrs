package cmd

import (
	"github.com/pgbkrs/pgbackup/internal/restore"
	"github.com/spf13/cobra"
)

var (
	restoreInputDir     string
	restorePreBackupDir string
	restoreSchema string
	restoreObject string
	restoreLogDir string
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore PostgreSQL schema and data from YAML+CSV files",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := restore.Options{
			BackupDir:    restoreInputDir,
			PreBackupDir: restorePreBackupDir,
			Schema:       restoreSchema,
			Object:       restoreObject,
			LogDir:       restoreLogDir,
		}
		return restore.RunRestore(cmd.Context(), conn, opts)
	},
}

func init() {
	restoreCmd.Flags().StringVar(&restoreInputDir, "input", "",
		"Path to the backup directory to restore from (required)")
	restoreCmd.Flags().StringVar(&restorePreBackupDir, "pre-backup-dir", ".",
		"Directory where the pre-restore safety backup will be written")
	restoreCmd.Flags().StringVar(&restoreSchema, "schema", "",
		"Restore only objects in this schema (e.g. myschema)")
	restoreCmd.Flags().StringVar(&restoreObject, "object", "",
		"Restore this object and its transitive dependencies (schema.name format, e.g. public.mytable)")
	restoreCmd.Flags().StringVar(&restoreLogDir, "log-dir", "",
		"Directory for restore log files (default: current working directory)")
	_ = restoreCmd.MarkFlagRequired("input")
}
