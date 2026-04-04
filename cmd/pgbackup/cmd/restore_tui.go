package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pgbkrs/pgbackup/internal/restore"
	"github.com/pgbkrs/pgbackup/internal/resolve"
	"github.com/pgbkrs/pgbackup/internal/tui"
	"github.com/spf13/cobra"
)

var (
	restoreTUIInputDir     string
	restoreTUIPreBackupDir string
	restoreTUILogDir       string
)

var restoreTUICmd = &cobra.Command{
	Use:   "restore-tui",
	Short: "Interactively select schemas/tables to restore from a backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		manifestPath := filepath.Join(restoreTUIInputDir, "_manifest.yaml")
		manifest, err := resolve.ReadManifest(manifestPath)
		if err != nil {
			return fmt.Errorf("load manifest: %w", err)
		}

		result, err := tui.RunSelector(manifest)
		if err != nil {
			return fmt.Errorf("tui: %w", err)
		}
		if result.Cancelled {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}
		if len(result.Schemas) == 0 && len(result.Objects) == 0 {
			fmt.Fprintln(os.Stderr, "Nothing selected.")
			return nil
		}

		opts := restore.Options{
			BackupDir:    restoreTUIInputDir,
			PreBackupDir: restoreTUIPreBackupDir,
			Schemas:      result.Schemas,
			Objects:      result.Objects,
			LogDir:       restoreTUILogDir,
		}
		return restore.RunRestore(cmd.Context(), conn, opts)
	},
}

func init() {
	restoreTUICmd.Flags().StringVar(&restoreTUIInputDir, "input", "",
		"Path to the backup directory to restore from (required)")
	restoreTUICmd.Flags().StringVar(&restoreTUIPreBackupDir, "pre-backup-dir", ".",
		"Directory where the pre-restore safety backup will be written")
	restoreTUICmd.Flags().StringVar(&restoreTUILogDir, "log-dir", "",
		"Directory for restore log files (default: current working directory)")
	_ = restoreTUICmd.MarkFlagRequired("input")
}
