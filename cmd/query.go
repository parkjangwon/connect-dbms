package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"oslo/internal/db"
	"oslo/internal/dberr"
	"oslo/internal/export"
	"oslo/internal/profile"
	"oslo/internal/sshtunnel"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Run a SQL query (non-interactive)",
	Long: `Run SQL against a database and print results.
Useful for scripts, pipes, and AI agents.

Examples:
  connect-dbms query --driver sqlite --dsn test.db --sql "SELECT * FROM users"
  connect-dbms query --profile mydb --sql "SELECT 1" --format json
  echo "SELECT 1" | connect-dbms query --profile mydb --format csv
  connect-dbms query --profile mydb --file query.sql --format tsv`,
	RunE: runQuery,
}

var (
	qDriver  string
	qDSN     string
	qProfile string
	qSQL     string
	qFile    string
	qFormat  string
	qNoHead  bool
	qTimeout int
	qQuiet   bool
	qMaxOpen int
	qMaxIdle int
	qMaxLife int
)

func init() {
	queryCmd.Flags().StringVar(&qDriver, "driver", "", "database driver (mysql, postgres, sqlite, oracle, mariadb)")
	queryCmd.Flags().StringVar(&qDSN, "dsn", "", "data source name / connection string")
	queryCmd.Flags().StringVarP(&qProfile, "profile", "p", "", "session name from config")
	queryCmd.Flags().StringVar(&qSQL, "sql", "", "SQL to run")
	queryCmd.Flags().StringVarP(&qFile, "file", "f", "", "file with SQL to run")
	queryCmd.Flags().StringVar(&qFormat, "format", "table", "output format: table, json, csv, tsv")
	queryCmd.Flags().BoolVar(&qNoHead, "no-header", false, "hide column headers (csv/tsv)")
	queryCmd.Flags().IntVar(&qTimeout, "timeout", 30, "query timeout in seconds")
	queryCmd.Flags().BoolVarP(&qQuiet, "quiet", "q", false, "suppress info messages")
	queryCmd.Flags().IntVar(&qMaxOpen, "max-open-conns", 0, "max open connections for the DB pool")
	queryCmd.Flags().IntVar(&qMaxIdle, "max-idle-conns", 0, "max idle connections for the DB pool")
	queryCmd.Flags().IntVar(&qMaxLife, "conn-max-lifetime-seconds", 0, "connection max lifetime in seconds")
}

func runQuery(cmd *cobra.Command, args []string) error {
	sql, err := resolveSQL()
	if err != nil {
		return err
	}

	cfg, driverName, err := resolveConnection()
	if err != nil {
		return err
	}

	format, err := export.ParseFormat(qFormat)
	if err != nil {
		return err
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(qTimeout)*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		dbe := dberr.Wrap(driverName, "ping", host, err)
		fmt.Fprintln(os.Stderr, dbe.Format())
		return fmt.Errorf("ping failed")
	}

	if !qQuiet {
		fmt.Fprintf(os.Stderr, "Connected to %s\n", driverName)
	}

	var result *db.QueryResult
	if db.IsSelectQuery(sql) {
		result, err = db.RunQuery(ctx, conn, sql)
	} else {
		result, err = db.RunExec(ctx, conn, sql)
	}
	if err != nil {
		dbe := dberr.Wrap(driverName, "query", host, err)
		fmt.Fprintln(os.Stderr, dbe.Format())
		return fmt.Errorf("query failed")
	}

	if !qQuiet {
		if result.Columns != nil {
			fmt.Fprintf(os.Stderr, "%d rows (%s)\n", result.RowCount, result.Duration.Round(time.Millisecond))
		} else {
			fmt.Fprintf(os.Stderr, "%d rows affected (%s)\n", result.RowCount, result.Duration.Round(time.Millisecond))
		}
	}

	if result.Columns != nil {
		return export.Write(os.Stdout, result, format, qNoHead)
	}
	return nil
}

func resolveSQL() (string, error) {
	if qSQL != "" {
		return qSQL, nil
	}
	if qFile != "" {
		data, err := os.ReadFile(qFile)
		if err != nil {
			return "", fmt.Errorf("read sql file: %w", err)
		}
		return string(data), nil
	}
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		s := strings.TrimSpace(string(data))
		if s != "" {
			return s, nil
		}
	}
	return "", fmt.Errorf("no SQL provided (use --sql, --file, or pipe stdin)")
}

func resolveConnection() (db.ConnConfig, string, error) {
	if qProfile != "" {
		store, err := profile.Load(cfgFile)
		if err != nil {
			return db.ConnConfig{}, "", err
		}
		p, err := store.Get(qProfile)
		if err != nil {
			return db.ConnConfig{}, "", err
		}
		return p.ToConnConfig(), p.Driver, nil
	}

	if qDriver == "" {
		return db.ConnConfig{}, "", fmt.Errorf("specify --driver and --dsn, or --profile")
	}

	return db.ConnConfig{
		Driver:                 qDriver,
		DSN:                    qDSN,
		MaxOpenConns:           qMaxOpen,
		MaxIdleConns:           qMaxIdle,
		ConnMaxLifetimeSeconds: qMaxLife,
	}, qDriver, nil
}
