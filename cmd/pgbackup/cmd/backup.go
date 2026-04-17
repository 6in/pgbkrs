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
	Short: "Back up PostgreSQL schema and data to YAML+CSV files (use --output to choose destination)",
	Long: `backup writes the full schema (YAML) and table data (CSV) into a timestamped
directory. Use --output to choose where that directory is created (defaults to
the current working directory). Use --snapshot for a REPEATABLE READ transaction
that keeps all reads consistent.`,
	Example: `  # Write the backup into ./2026-...-... under the current directory
  pgbackup backup --dbname mydb --user postgres

  # Write the backup under /var/backups
  pgbackup backup --dbname mydb --user postgres --output /var/backups

  # Snapshot-consistent backup
  pgbackup backup --dbname mydb --user postgres --snapshot --output /var/backups`,
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
