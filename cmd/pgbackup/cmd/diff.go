package cmd

import "github.com/spf13/cobra"

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Diff two PostgreSQL schemas and show structural differences",
	RunE: func(cmd *cobra.Command, args []string) error {
		// conn available here from PersistentPreRunE
		// Phase 10+ will add real diff logic
		return nil
	},
}
