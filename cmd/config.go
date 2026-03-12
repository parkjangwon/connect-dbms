package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"oslo/internal/db"
	"oslo/internal/profile"
	"oslo/internal/tui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage DB session configs",
	Long: `Manage saved database connection sessions.
Config is stored at ~/.config/connect-dbms/config.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := profile.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		app := tui.NewConfigApp(store)
		p := tea.NewProgram(app, tea.WithAltScreen())
		_, err = p.Run()
		return err
	},
}

// --- config list ---

var configListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List saved sessions",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := profile.Load(cfgFile)
		if err != nil {
			return err
		}
		profiles := store.List()
		if len(profiles) == 0 {
			fmt.Println("No sessions saved. Use 'connect-dbms config add' to create one.")
			return nil
		}

		if configJSON {
			data, err := json.MarshalIndent(profiles, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "NAME\tDRIVER\tHOST\tPORT\tDATABASE")
		for _, p := range profiles {
			host := p.Host
			if p.DSN != "" {
				host = "(dsn)"
			}
			port := ""
			if p.Port > 0 {
				port = fmt.Sprintf("%d", p.Port)
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", p.Name, p.Driver, host, port, p.Database)
		}
		return tw.Flush()
	},
}

// --- config show ---

var configShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show session details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := profile.Load(cfgFile)
		if err != nil {
			return err
		}
		p, err := store.Get(args[0])
		if err != nil {
			return err
		}
		data, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	},
}

// --- config add ---

var (
	caName     string
	caDriver   string
	caHost     string
	caPort     int
	caUser     string
	caPassword string
	caDatabase string
	caDSN      string
	caMaxOpen  int
	caMaxIdle  int
	caMaxLife  int
	caSSHHost  string
	caSSHPort  int
	caSSHUser  string
	caSSHPass  string
	caSSHKey   string
	configJSON bool
)

var configAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new session",
	Long: `Add a database connection session.

Examples:
  connect-dbms config add --name local-pg --driver postgres --host localhost --port 5432 --user postgres --database mydb
  connect-dbms config add --name local-sqlite --driver sqlite --database ./data.db
  connect-dbms config add --name prod-mysql --driver mysql --dsn "user:pass@tcp(host:3306)/db"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if caName == "" {
			return fmt.Errorf("--name is required")
		}
		if caDriver == "" {
			return fmt.Errorf("--driver is required")
		}
		if _, err := db.Get(caDriver); err != nil {
			return err
		}

		store, err := profile.Load(cfgFile)
		if err != nil {
			return err
		}

		p := profile.Profile{
			Name:                   caName,
			Driver:                 caDriver,
			Host:                   caHost,
			Port:                   caPort,
			User:                   caUser,
			Password:               caPassword,
			Database:               caDatabase,
			DSN:                    caDSN,
			MaxOpenConns:           caMaxOpen,
			MaxIdleConns:           caMaxIdle,
			ConnMaxLifetimeSeconds: caMaxLife,
			SSHHost:                caSSHHost,
			SSHPort:                caSSHPort,
			SSHUser:                caSSHUser,
			SSHPassword:            caSSHPass,
			SSHKeyPath:             caSSHKey,
		}

		if err := store.Add(p); err != nil {
			return err
		}
		fmt.Printf("Session '%s' saved.\n", caName)
		return nil
	},
}

// --- config edit ---

var configEditCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Edit a saved session",
	Long: `Edit an existing session. Only provided flags are changed.

Examples:
  connect-dbms config edit mydb --host newhost.example.com --port 3307
  connect-dbms config edit mydb --password newpass
  connect-dbms config edit mydb --database otherdb`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := profile.Load(cfgFile)
		if err != nil {
			return err
		}
		p, err := store.Get(args[0])
		if err != nil {
			return err
		}

		// Only update fields that were explicitly set
		if cmd.Flags().Changed("driver") {
			if _, err := db.Get(caDriver); err != nil {
				return err
			}
			p.Driver = caDriver
		}
		if cmd.Flags().Changed("host") {
			p.Host = caHost
		}
		if cmd.Flags().Changed("port") {
			p.Port = caPort
		}
		if cmd.Flags().Changed("user") {
			p.User = caUser
		}
		if cmd.Flags().Changed("password") {
			p.Password = caPassword
		}
		if cmd.Flags().Changed("database") {
			p.Database = caDatabase
		}
		if cmd.Flags().Changed("dsn") {
			p.DSN = caDSN
		}
		if cmd.Flags().Changed("max-open-conns") {
			p.MaxOpenConns = caMaxOpen
		}
		if cmd.Flags().Changed("max-idle-conns") {
			p.MaxIdleConns = caMaxIdle
		}
		if cmd.Flags().Changed("conn-max-lifetime-seconds") {
			p.ConnMaxLifetimeSeconds = caMaxLife
		}
		if cmd.Flags().Changed("ssh-host") {
			p.SSHHost = caSSHHost
		}
		if cmd.Flags().Changed("ssh-port") {
			p.SSHPort = caSSHPort
		}
		if cmd.Flags().Changed("ssh-user") {
			p.SSHUser = caSSHUser
		}
		if cmd.Flags().Changed("ssh-password") {
			p.SSHPassword = caSSHPass
		}
		if cmd.Flags().Changed("ssh-key-path") {
			p.SSHKeyPath = caSSHKey
		}

		if err := store.Update(args[0], *p); err != nil {
			return err
		}
		fmt.Printf("Session '%s' updated.\n", args[0])
		return nil
	},
}

// --- config remove ---

var configRemoveCmd = &cobra.Command{
	Use:     "remove [name]",
	Short:   "Remove a session",
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := profile.Load(cfgFile)
		if err != nil {
			return err
		}
		if err := store.Remove(args[0]); err != nil {
			return err
		}
		fmt.Printf("Session '%s' removed.\n", args[0])
		return nil
	},
}

// --- config path ---

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file path",
	Run: func(cmd *cobra.Command, args []string) {
		path := cfgFile
		if path == "" {
			path = profile.DefaultPath()
		}
		fmt.Println(path)
	},
}

func init() {
	// add flags
	configAddCmd.Flags().StringVar(&caName, "name", "", "session name (required)")
	configAddCmd.Flags().StringVar(&caDriver, "driver", "", "database driver (required)")
	configAddCmd.Flags().StringVar(&caHost, "host", "", "database host")
	configAddCmd.Flags().IntVar(&caPort, "port", 0, "database port")
	configAddCmd.Flags().StringVar(&caUser, "user", "", "database user")
	configAddCmd.Flags().StringVar(&caPassword, "password", "", "database password")
	configAddCmd.Flags().StringVar(&caDatabase, "database", "", "database name or path")
	configAddCmd.Flags().StringVar(&caDSN, "dsn", "", "raw connection string")
	configAddCmd.Flags().IntVar(&caMaxOpen, "max-open-conns", 0, "max open connections for the DB pool")
	configAddCmd.Flags().IntVar(&caMaxIdle, "max-idle-conns", 0, "max idle connections for the DB pool")
	configAddCmd.Flags().IntVar(&caMaxLife, "conn-max-lifetime-seconds", 0, "connection max lifetime in seconds")
	configAddCmd.Flags().StringVar(&caSSHHost, "ssh-host", "", "SSH jump host")
	configAddCmd.Flags().IntVar(&caSSHPort, "ssh-port", 22, "SSH jump port")
	configAddCmd.Flags().StringVar(&caSSHUser, "ssh-user", "", "SSH user")
	configAddCmd.Flags().StringVar(&caSSHPass, "ssh-password", "", "SSH password")
	configAddCmd.Flags().StringVar(&caSSHKey, "ssh-key-path", "", "SSH private key path")

	// edit flags (reuse same vars)
	configEditCmd.Flags().StringVar(&caDriver, "driver", "", "database driver")
	configEditCmd.Flags().StringVar(&caHost, "host", "", "database host")
	configEditCmd.Flags().IntVar(&caPort, "port", 0, "database port")
	configEditCmd.Flags().StringVar(&caUser, "user", "", "database user")
	configEditCmd.Flags().StringVar(&caPassword, "password", "", "database password")
	configEditCmd.Flags().StringVar(&caDatabase, "database", "", "database name or path")
	configEditCmd.Flags().StringVar(&caDSN, "dsn", "", "raw connection string")
	configEditCmd.Flags().IntVar(&caMaxOpen, "max-open-conns", 0, "max open connections for the DB pool")
	configEditCmd.Flags().IntVar(&caMaxIdle, "max-idle-conns", 0, "max idle connections for the DB pool")
	configEditCmd.Flags().IntVar(&caMaxLife, "conn-max-lifetime-seconds", 0, "connection max lifetime in seconds")
	configEditCmd.Flags().StringVar(&caSSHHost, "ssh-host", "", "SSH jump host")
	configEditCmd.Flags().IntVar(&caSSHPort, "ssh-port", 22, "SSH jump port")
	configEditCmd.Flags().StringVar(&caSSHUser, "ssh-user", "", "SSH user")
	configEditCmd.Flags().StringVar(&caSSHPass, "ssh-password", "", "SSH password")
	configEditCmd.Flags().StringVar(&caSSHKey, "ssh-key-path", "", "SSH private key path")

	// list flags
	configListCmd.Flags().BoolVar(&configJSON, "json", false, "output as JSON")

	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configRemoveCmd)
	configCmd.AddCommand(configPathCmd)
}
