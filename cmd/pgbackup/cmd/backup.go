package cmd

import (
	"github.com/pgbkrs/pgbackup/internal/backup"
	"github.com/spf13/cobra"
)

var (
	snapshotFlag bool
	outputDir    string
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Back up PostgreSQL schema and data to YAML+CSV files",
	RunE: func(cmd *cobra.Command, args []string) error {
		return backup.RunBackup(cmd.Context(), conn, outputDir, snapshotFlag)
	},
}

func init() {
	backupCmd.Flags().BoolVar(&snapshotFlag, "snapshot", false,
		"Use REPEATABLE READ transaction for a consistent snapshot across all reads")
	backupCmd.Flags().StringVar(&outputDir, "output", ".",
		"Directory where the timestamped backup folder will be created")
}
