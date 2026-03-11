package cmd

import (
	"github.com/pgbkrs/pgbackup/internal/restore"
	"github.com/spf13/cobra"
)

var (
	restoreInputDir     string
	restorePreBackupDir string
	// Placeholders for Plan 03 flag wiring (REST-10, REST-11, REST-13)
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
	_ = restoreCmd.MarkFlagRequired("input")
	// Note: --schema, --object, --log-dir flags will be registered in Plan 03
}
