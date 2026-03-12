package cmd

import (
	"os"

	"github.com/pgbkrs/pgbackup/internal/diff"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff <backup-a> <backup-b>",
	Short: "Compare two backup directories at the schema level",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return diff.Run(args[0], args[1], os.Stdout)
	},
}
