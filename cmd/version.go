package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"oslo/internal/db"
)

var (
	Version   = "1.0.0"
	BuildDate = "dev"
	GitCommit = "none"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("connect-dbms %s\n", Version)
		fmt.Printf("  build:   %s\n", BuildDate)
		fmt.Printf("  commit:  %s\n", GitCommit)
		fmt.Printf("  go:      %s\n", runtime.Version())
		fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  drivers: %s\n", db.AvailableDrivers())
	},
}
