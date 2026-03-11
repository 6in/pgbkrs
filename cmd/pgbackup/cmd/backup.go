package cmd

import "github.com/spf13/cobra"

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Back up PostgreSQL schema and data to YAML+CSV files",
	RunE: func(cmd *cobra.Command, args []string) error {
		// conn available here from PersistentPreRunE
		// Phase 2+ will add real backup logic
		return nil
	},
}
