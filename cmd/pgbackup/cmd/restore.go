package cmd

import "github.com/spf13/cobra"

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore PostgreSQL schema and data from YAML+CSV files",
	RunE: func(cmd *cobra.Command, args []string) error {
		// conn available here from PersistentPreRunE
		// Phase 8+ will add real restore logic
		return nil
	},
}
