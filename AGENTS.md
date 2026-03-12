# connect-dbms - Agent Development Guide

## What is this?

A **TUI + CLI database client** that connects to many RDBMS from a single binary.
Built in Go. No CGO needed. Cross-compiles to Linux, macOS, Windows, Termux.

Binary name: `connect-dbms`
Go module name: `oslo` (internal, does not affect binary name)

---

## Supported Databases

| Driver name    | Go package                    | Status      | Notes                                |
|---------------|-------------------------------|-------------|--------------------------------------|
| `mysql`       | `github.com/go-sql-driver/mysql` | Working  | Also handles MariaDB                |
| `mariadb`     | (same as mysql)               | Working     | Alias to mysql driver               |
| `postgres`    | `github.com/jackc/pgx/v5`    | Working     | Pure Go                              |
| `postgresql`  | (same as postgres)            | Working     | Alias                                |
| `oracle`      | `github.com/sijms/go-ora/v2`  | Working     | Pure Go, no Oracle client needed    |
| `sqlite`      | `modernc.org/sqlite`          | Working     | Pure Go (C transpiled), no CGO      |
| `tibero`      | ODBC (build tag)              | Stub        | Build with `-tags tibero`           |
| `cubrid`      | ODBC (build tag)              | Stub        | Build with `-tags cubrid`           |

---

## Architecture

```
main.go                     Entry point → cmd.Execute()
cmd/                        Cobra CLI commands
  root.go                   Root command, global flags
  config.go                 `connect-dbms config` (TUI + CLI subcommands)
  connect.go                `connect-dbms connect` (test connection)
  query.go                  `connect-dbms query` (non-interactive SQL)
  tui.go                    `connect-dbms` (default: launches TUI)
  version.go                `connect-dbms version`
internal/
  db/
    driver.go               Driver interface, registry, RunQuery, RunExec, IsSelectQuery
    mysql.go                MySQL/MariaDB driver + MetaInspector
    postgres.go             PostgreSQL driver + MetaInspector
    oracle.go               Oracle driver + MetaInspector
    sqlite.go               SQLite driver + MetaInspector
    tibero.go               Tibero stub (build tag: tibero)
    cubrid.go               Cubrid stub (build tag: cubrid)
  tui/
    app.go                  Root bubbletea model, screen router
    theme.go                Colors, styles (lipgloss)
    keys.go                 Key bindings
    config.go               TUI config manager (add/edit/delete sessions)
    screen_connect.go       Connection picker + quick connect form
    screen_query.go         SQL editor + results table
    screen_tables.go        Table browser + column inspector
    screen_help.go          Help overlay
  profile/
    profile.go              Session config store (JSON at ~/.config/connect-dbms/config.json)
  export/
    format.go               Output formatters: table, JSON, CSV, TSV
  history/
    history.go              Query history store (SQLite at ~/.config/connect-dbms/history.db)
  dberr/
    dberr.go                Error wrapping with driver-specific codes + stack traces
```

### Key Design Decisions

1. **All default drivers are pure Go** — no CGO, so `CGO_ENABLED=0` cross-compilation works.
   `modernc.org/sqlite` is a C-to-Go transpilation. This adds ~20MB to binary but avoids CGO.

2. **Driver registry pattern** — `db.Register(name, driver)` in each driver's `init()`.
   Adding a new driver: create `internal/db/newdriver.go`, implement `db.Driver` + `db.MetaInspector`, call `Register` in `init()`.

3. **Dual-mode** — TUI (bubbletea) and CLI (cobra) share the same `internal/` packages.
   CLI commands (`query`, `connect`, `config list --json`) are designed for scripts and AI agents.

4. **Config storage** — `~/.config/connect-dbms/config.json` (plaintext JSON).
   Passwords are stored in plaintext. This is intentional (user's request). Do not add encryption unless asked.

5. **Error handling** — `internal/dberr` wraps errors with driver-specific codes, stack traces, timestamps.
   All DB errors go through `dberr.Wrap()` which extracts MySQL error numbers, PG SQLSTATE, ORA codes, SQLite codes.

6. **TUI screens** — Elm architecture via bubbletea. Root `App` model routes between screens.
   Each screen has its own file. Communication between screens uses tea.Msg types defined in `app.go`.

---

## How to Build

```bash
# Development build
go build -ldflags "-s -w" -o connect-dbms .

# Cross-compile (all platforms)
make build-all

# With Tibero/Cubrid ODBC support
go build -tags "tibero cubrid" -o connect-dbms .

# Version injection
make build VERSION=1.0.0
```

---

## Config File Format

Path: `~/.config/connect-dbms/config.json`

```json
{
  "profiles": [
    {
      "name": "my-postgres",
      "driver": "postgres",
      "host": "localhost",
      "port": 5432,
      "user": "admin",
      "password": "secret",
      "database": "mydb"
    },
    {
      "name": "raw-dsn-example",
      "driver": "mysql",
      "dsn": "user:pass@tcp(host:3306)/db?parseTime=true"
    }
  ]
}
```

When `dsn` is set, it overrides `host/port/user/password/database` fields.

---

## Error Code System

Errors are output to stderr with this format:
```
[ERROR] 2026-03-12 13:50:50.827 KST
  Driver : postgres
  Phase  : ping
  Host   : localhost
  Code   : PG-CONN
  Message: connection refused
  Raw    : <original error>
  Trace  :
    <stack frames>
```

Code patterns:
- MySQL: `MYSQL-{errno}` (e.g., `MYSQL-1045`)
- PostgreSQL: `PG-{SQLSTATE}` (e.g., `PG-42P01`)
- Oracle: `ORA-{code}` (e.g., `ORA-12541`)
- SQLite: `SQLITE_{NAME}({code})` (e.g., `SQLITE_ERROR(1)`)
- Network: `CONN-REFUSED`, `CONN-TIMEOUT`, `CONN-DNS`, `NET-DIAL`

---

## What is NOT Implemented Yet (Future Work)

### Medium Priority

### Low Priority
- **Tibero/Cubrid MetaInspector** — the stubs have minimal `Tables()`/`Columns()` but no `Indexes()`/`ForeignKeys()`.

## Release Strategy

- Ship **basic** and **ODBC/full** releases separately.
- Basic release: pure-Go oriented, default DB support.
- ODBC/full release: macOS/Linux only, CGO-enabled, Tibero/Cubrid support behind ODBC.
- Even ODBC/full artifacts still require the vendor ODBC driver on the target host.

---

## Conventions to Follow

### Code Style
- Go standard formatting (`gofmt`).
- No external config libraries (was using `gopkg.in/yaml.v3`, now removed — switched to `encoding/json`).
- Keep `internal/` packages independent from each other where possible. `db` is the foundation, `tui` and `cmd` depend on it.

### Adding a New Database Driver
1. Create `internal/db/newdriver.go`
2. Implement `db.Driver` interface: `Name()`, `Open()`, `BuildDSN()`, `Meta()`
3. Implement `db.MetaInspector` interface for schema introspection
4. Call `db.Register("drivername", &NewDriver{})` in `init()`
5. If the driver needs CGO or external libs, use a build tag (see `tibero.go` as example)

### Adding a New TUI Screen
1. Create `internal/tui/screen_newscreen.go`
2. Add screen constant to `Screen` enum in `app.go`
3. Add screen field to `App` struct
4. Wire routing in `App.Update()` and `App.View()`
5. Use `StyleBorder`, `StyleActiveBorder`, `StyleSelected` etc. from `theme.go`

### Adding a New CLI Command
1. Create `cmd/newcmd.go`
2. Define cobra command
3. Register in `cmd/root.go` `init()` with `rootCmd.AddCommand(newCmd)`
4. For DB operations, use `dberr.Wrap()` for error output

### Testing
- Some tests now exist.
- Prefer SQLite `:memory:` or temp files for integration-style tests.
- Good targets include `internal/db/driver.go`, `internal/export/format.go`, `internal/profile/profile.go`, `internal/history/history.go`, and TUI helper logic.

---

## Dependencies (direct only)

| Package | Purpose | Version |
|---------|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework | v1.3.10 |
| `github.com/charmbracelet/bubbles` | TUI components (textarea, textinput) | v1.0.0 |
| `github.com/charmbracelet/lipgloss` | TUI styling | v1.1.0 |
| `github.com/spf13/cobra` | CLI framework | v1.10.2 |
| `github.com/go-sql-driver/mysql` | MySQL/MariaDB driver | v1.9.3 |
| `github.com/jackc/pgx/v5` | PostgreSQL driver | v5.8.0 |
| `github.com/sijms/go-ora/v2` | Oracle driver (pure Go) | v2.9.0 |
| `modernc.org/sqlite` | SQLite driver (pure Go, no CGO) | v1.46.1 |

---

## Common Pitfalls

1. **Module name is `oslo`, not `connect-dbms`** — the Go module path in `go.mod` is `oslo`.
   All import paths use `oslo/internal/...`. The binary output name is set in Makefile/build command.

2. **Tibero/Cubrid files use build tags** — they won't compile in the default build.
   These files start with `//go:build tibero` or `//go:build cubrid`.

3. **TUI uses Alt Screen** — `tea.WithAltScreen()` is used for both the main TUI and config TUI.
   This means the terminal is fully taken over. stderr output during TUI will appear after TUI exits.

4. **Password masking in TUI** — the config detail view masks passwords with `*`.
   But `config show` CLI outputs the actual password (plaintext JSON). This is by design.

5. **Driver field in config TUI** — uses Left/Right arrow keys to pick from a list, not free text.
   The available drivers are hardcoded in `driverList` in `internal/tui/config.go`.
   If you add a new driver, update this list.

6. **`gopkg.in/yaml.v3` is no longer used** — was removed when switching config from YAML to JSON.
   `go mod tidy` should have cleaned it up, but verify if you see import errors.
