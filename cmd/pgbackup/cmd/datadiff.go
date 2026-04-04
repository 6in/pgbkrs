package cmd

import (
	"os"

	"github.com/pgbkrs/pgbackup/internal/datadiff"
	"github.com/spf13/cobra"
)

var dataDiffCmd = &cobra.Command{
	Use:   "data-diff <backup-a> <backup-b>",
	Short: "Compare table data (CSV) between two backup directories",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return datadiff.Run(args[0], args[1], os.Stdout)
	},
}
