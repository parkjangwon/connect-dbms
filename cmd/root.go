package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "connect-dbms",
	Short: "A TUI database client for many RDBMS",
	Long: `connect-dbms is a TUI-based database client that works with
MySQL, MariaDB, PostgreSQL, Oracle, SQLite, Tibero, and Cubrid.

Run without arguments to start the TUI, or use subcommands
for non-interactive use (great for scripts and AI agents).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI(cmd, args)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/connect-dbms/config.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(driversCmd)
}

func exitErr(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}
