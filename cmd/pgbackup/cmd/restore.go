package cmd

import (
	"github.com/pgbkrs/pgbackup/internal/restore"
	"github.com/spf13/cobra"
)

var (
	restoreInputDir     string
	restorePreBackupDir string
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore PostgreSQL schema and data from YAML+CSV files",
	RunE: func(cmd *cobra.Command, args []string) error {
		return restore.RunRestore(cmd.Context(), conn, restoreInputDir, restorePreBackupDir)
	},
}

func init() {
	restoreCmd.Flags().StringVar(&restoreInputDir, "input", "",
		"Path to the backup directory to restore from (required)")
	restoreCmd.Flags().StringVar(&restorePreBackupDir, "pre-backup-dir", ".",
		"Directory where the pre-restore safety backup will be written")
	_ = restoreCmd.MarkFlagRequired("input")
}
