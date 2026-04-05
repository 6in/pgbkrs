package cmd

import (
	"github.com/pgbkrs/pgbackup/internal/backup"
	"github.com/spf13/cobra"
)

var (
	backupAISnapshotFlag bool
	backupAIOutputDir    string
)

var backupAICmd = &cobra.Command{
	Use:   "backup-ai",
	Short: "Back up PostgreSQL schema and sampled data optimised for AI consumption",
	Long: `backup-ai produces the same schema files as backup, but exports at most
10 rows per table (with column headers) instead of full data. A README.md is
written at the backup root describing the database structure for AI reading.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return backup.RunBackupAI(cmd.Context(), conn, backupAIOutputDir, backupAISnapshotFlag)
	},
}

func init() {
	backupAICmd.Flags().BoolVar(&backupAISnapshotFlag, "snapshot", false,
		"Use REPEATABLE READ transaction for a consistent snapshot across all reads")
	backupAICmd.Flags().StringVar(&backupAIOutputDir, "output", ".",
		"Directory where the timestamped backup folder will be created")
}
