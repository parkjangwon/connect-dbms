package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"oslo/internal/db"
	"oslo/internal/dberr"
	"oslo/internal/profile"
	"oslo/internal/sshtunnel"
)

var connectCmd = &cobra.Command{
	Use:   "connect [session]",
	Short: "Test a database connection",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var cfg db.ConnConfig
		var driverName string

		if len(args) > 0 {
			store, err := profile.Load(cfgFile)
			if err != nil {
				return err
			}
			p, err := store.Get(args[0])
			if err != nil {
				return err
			}
			cfg = p.ToConnConfig()
			driverName = p.Driver
		} else if connDriver != "" {
			cfg = db.ConnConfig{
				Driver:                 connDriver,
				DSN:                    connDSN,
				MaxOpenConns:           connMaxOpen,
				MaxIdleConns:           connMaxIdle,
				ConnMaxLifetimeSeconds: connMaxLife,
			}
			driverName = connDriver
		} else {
			return fmt.Errorf("specify a session name or use --driver/--dsn")
		}

		drv, err := db.Get(driverName)
		if err != nil {
			return err
		}

		host := cfg.Host
		if host == "" && cfg.DSN != "" {
			host = "(dsn)"
		}

		cfg, tunnel, err := sshtunnel.PrepareConnConfig(cfg)
		if err != nil {
			return err
		}
		if tunnel != nil {
			defer tunnel.Close()
		}

		conn, err := drv.Open(cfg)
		if err != nil {
			dbe := dberr.Wrap(driverName, "open", host, err)
			fmt.Fprintln(os.Stderr, dbe.Format())
			return fmt.Errorf("connect failed")
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		start := time.Now()
		if err := conn.PingContext(ctx); err != nil {
			dbe := dberr.Wrap(driverName, "ping", host, err)
			fmt.Fprintln(os.Stderr, dbe.Format())
			return fmt.Errorf("ping failed")
		}

		fmt.Printf("OK - connected to %s (%s)\n", driverName, time.Since(start).Round(time.Millisecond))
		return nil
	},
}

var (
	connDriver  string
	connDSN     string
	connMaxOpen int
	connMaxIdle int
	connMaxLife int
)

func init() {
	connectCmd.Flags().StringVar(&connDriver, "driver", "", "database driver")
	connectCmd.Flags().StringVar(&connDSN, "dsn", "", "connection string")
	connectCmd.Flags().IntVar(&connMaxOpen, "max-open-conns", 0, "max open connections for the DB pool")
	connectCmd.Flags().IntVar(&connMaxIdle, "max-idle-conns", 0, "max idle connections for the DB pool")
	connectCmd.Flags().IntVar(&connMaxLife, "conn-max-lifetime-seconds", 0, "connection max lifetime in seconds")
}

var driversCmd = &cobra.Command{
	Use:   "drivers",
	Short: "List available database drivers",
	Run: func(cmd *cobra.Command, args []string) {
		for _, name := range db.ListDrivers() {
			fmt.Println(name)
		}
	},
}
