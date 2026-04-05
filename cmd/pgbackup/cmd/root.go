package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mattn/go-isatty"
	"github.com/pgbkrs/pgbackup/internal/db"
	"github.com/spf13/cobra"
)

var (
	host     string
	port     int
	user     string
	password string
	dbname   string
	conn     *pgx.Conn
)

var rootCmd = &cobra.Command{
	Use:           "pgbackup",
	Short:         "PostgreSQL schema backup, restore, and diff tool",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "diff" || cmd.Name() == "data-diff" {
			return nil // these commands read backup dirs from disk; no DB connection needed
		}
		connStr := db.BuildConnString(host, port, user, password, dbname)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var err error
		conn, err = db.ConnectAndPing(ctx, connStr)
		if err != nil {
			printError(err)
			return err
		}
		return nil
	},
}

// Execute is the entry point called by main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&host, "host", "localhost", "PostgreSQL host")
	rootCmd.PersistentFlags().IntVar(&port, "port", 5432, "PostgreSQL port")
	rootCmd.PersistentFlags().StringVar(&user, "user", "", "PostgreSQL user")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "PostgreSQL password")
	rootCmd.PersistentFlags().StringVar(&dbname, "dbname", "", "Target database name")

	rootCmd.AddCommand(backupCmd, backupAICmd, restoreCmd, restoreTUICmd, diffCmd, dataDiffCmd)
}

func printError(err error) {
	if isatty.IsTerminal(os.Stderr.Fd()) {
		fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}
