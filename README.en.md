# connect-dbms

[한글 README](./README.md)

`connect-dbms` is a Go-based TUI + CLI database client that connects to multiple RDBMS from a single binary.

## Overview

`connect-dbms` is a terminal-first database client. Run it without arguments to launch the TUI, or use subcommands for non-interactive workflows in scripts, automation, and agent environments.

Core design goals:

- The default build uses mostly pure-Go drivers, so `CGO_ENABLED=0` builds work.
- The TUI and CLI share the same internal packages.
- Configuration is stored as human-readable JSON.
- Database errors are wrapped into a structured terminal-friendly format.

## Implemented Features

These are available in the current codebase.

- Multiple database drivers: MySQL, MariaDB, PostgreSQL, Oracle, SQLite
- Default TUI flow: saved sessions, quick connect, SQL execution, result browsing
- Query history: automatic persistence plus search/load flow
- TUI export for current query results
- Multiple query tabs: open with F6 and switch with F7/F8
- Autocomplete: open table-name suggestions with Ctrl+Space/F9
- SQL preview highlighting: keyword-highlighted preview panel below the editor
- SSH tunnel support: connect through an SSH jump host from saved sessions
- Table browser: table list, column inspection, `SELECT * ... LIMIT 100` starter query generation
- Non-interactive SQL execution with `query` in `table`, `json`, `csv`, and `tsv` formats
- Session management with `config add`, `edit`, `remove`, `list`, `show`, and `path`
- Connection testing with `connect`
- Driver discovery with `drivers`
- Version and build info with `version`
- Structured driver-specific and network-aware error formatting

## Supported Databases

| Driver | Status | Notes |
| --- | --- | --- |
| `mysql` | Working | `github.com/go-sql-driver/mysql` |
| `mariadb` | Working | Alias to the MySQL driver |
| `postgres` | Working | Based on `pgx/v5` |
| `postgresql` | Working | PostgreSQL alias |
| `oracle` | Working | Pure Go via `go-ora` |
| `sqlite` | Working | `modernc.org/sqlite`, no CGO |
| `tibero` | Stub / Optional | Build tag required, ODBC-based |
| `cubrid` | Stub / Optional | Build tag required, ODBC-based |

## Why This Project

The project is designed for teams and individuals who work with multiple relational databases and want one binary, one config format, and one consistent workflow for connecting, querying, and testing.

## Quick Start

Recommended environment:

- Go 1.24 or newer
- macOS, Linux, Windows, or Termux
- No native database client libraries needed for the default drivers

### 1. Build

```bash
go build -ldflags "-s -w" -o connect-dbms .
```

Or use the Makefile:

```bash
make build
```

### 2. List drivers

```bash
./connect-dbms drivers
```

### 3. Add a session

PostgreSQL example:

```bash
./connect-dbms config add \
  --name local-pg \
  --driver postgres \
  --host 127.0.0.1 \
  --port 5432 \
  --user postgres \
  --database mydb
```

SQLite example:

```bash
./connect-dbms config add \
  --name local-sqlite \
  --driver sqlite \
  --database ./data.db
```

Raw DSN example:

```bash
./connect-dbms config add \
  --name prod-mysql \
  --driver mysql \
  --dsn "user:pass@tcp(db.example.com:3306)/app?parseTime=true"
```

### 4. Test a connection

```bash
./connect-dbms connect local-pg
```

Or connect directly without a saved session:

```bash
./connect-dbms connect --driver sqlite --dsn ./data.db
```

### 5. Run SQL

```bash
./connect-dbms query --profile local-pg --sql "SELECT now()" --format table
```

From a file:

```bash
./connect-dbms query --profile local-pg --file query.sql --format json
```

From stdin:

```bash
echo "SELECT 1" | ./connect-dbms query --profile local-pg --format csv
```

### 6. Launch the TUI

```bash
./connect-dbms
```

Running the binary without arguments opens the full-screen Alt Screen TUI.

## TUI Usage

The current TUI centers on three flows.

### Connect Screen

- Left: saved session list
- Right: quick connect form
- `Enter`: connect
- `Tab`: switch panels
- `Delete`: remove saved session

### Query Screen

- Top: SQL editor
- Bottom: result table or execution status
- `F5` or `Ctrl+E`: run SQL
- `Ctrl+H`: search and load saved query history
- `Ctrl+S`: export the current result
- `Ctrl+Space` or `F9`: open autocomplete
- `F6`: open a new query tab
- `F7`, `F8`: move to the next or previous query tab
- `Tab`: switch between editor and results
- `PgUp`, `PgDn`: scroll results

### Tables Screen

- `Ctrl+T`: open the table browser
- Inspect table names and columns side by side
- `Enter`: generate a starter query for the selected table
- `Esc`: return to the query screen

### Global Keys

- `F1`: help
- `Ctrl+N`: new connection
- `Ctrl+Q`: quit

## CLI Usage

### Basic help

```bash
./connect-dbms --help
```

### Common commands

```bash
./connect-dbms
./connect-dbms config
./connect-dbms config list --json
./connect-dbms config show local-pg
./connect-dbms connect local-pg
./connect-dbms query --profile local-pg --sql "SELECT * FROM users"
./connect-dbms version
./connect-dbms drivers
```

Notes:

- Running `connect-dbms config` without subcommands opens the config-management TUI.
- `connect-dbms query` runs with either `--profile` or `--driver` plus `--dsn`.
- The config TUI shows drivers in this order: `mysql`, `mariadb`, `oracle`, `postgresql`, `tibero`, `cubrid`, `sqlite`.
- In the config TUI, `sqlite` uses the `File Path` field instead of a host-based database setup. Typical values are `./data.db` or `/tmp/app.db`.
- For `sqlite`, `Host`, `Port`, `User`, and `Password` are ignored.
- Pooling can be configured via `max_open_conns`, `max_idle_conns`, and `conn_max_lifetime_seconds` in saved sessions or CLI flags.
- SSH can be configured with `ssh_host`, `ssh_port`, `ssh_user`, `ssh_password`, and `ssh_key_path` in saved sessions or via config CLI/TUI.

### `query` output formats

- `table`: human-readable table output
- `json`: automation-friendly JSON array
- `csv`: comma-separated output
- `tsv`: tab-separated output

## Configuration File

Default path:

```text
~/.config/connect-dbms/config.json
```

Use the global `--config` flag to point to a custom file.

If the config file does not exist on first run, an empty `config.json` is created automatically.

Example:

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

Important:

- If `dsn` is set, it takes precedence over `host`, `port`, `user`, `password`, and `database`.
- Passwords are intentionally stored in plaintext JSON in the current design.

## Build

### Development build

```bash
make build
```

### Cross compilation

```bash
make build-all
```

Artifacts are generated under `dist/`.

### Version injection

```bash
make build VERSION=1.0.0
```

### GoReleaser

```bash
goreleaser release --snapshot --clean
```

The basic pure-Go release config lives in `.goreleaser.yaml`.

For the ODBC/full release:

```bash
goreleaser release --snapshot --clean --config .goreleaser-odbc.yaml
```

Files:
- basic: `.goreleaser.yaml`
- ODBC/full: `.goreleaser-odbc.yaml`

Release split:
- basic: pure-Go oriented, default database support
- ODBC/full: macOS/Linux only, Tibero/Cubrid-enabled build

GitHub Actions:
- `.github/workflows/release.yml` publishes the basic and ODBC/full releases separately on `v*` tag pushes.

### Optional build tags

```bash
make build-odbc
```

Use this only when you need the optional ODBC-based integrations.

Additional requirements:

- `tibero` and `cubrid` require an ODBC driver plus ODBC development headers.
- On macOS and Linux, this usually means installing a system library such as `unixODBC`.
- This path is not compatible with `CGO_ENABLED=0`.
- Even the ODBC/full binary still requires the vendor ODBC driver to be installed on the target machine.

macOS + Homebrew example:

```bash
brew install unixodbc
make build-odbc
make test-odbc
```

Linux (Fedora / RHEL family) example:

```bash
sudo dnf install unixODBC unixODBC-devel -y
make build-odbc
make test-odbc
```

Notes:

- `unixODBC` provides the runtime library.
- `unixODBC-devel` provides build headers such as `sql.h`.
- You still need the vendor-specific ODBC driver for the target database.
```

## Architecture Summary

The project uses Cobra for the CLI and Bubble Tea for the TUI, with shared packages under `internal/`.

```text
main.go
cmd/
  root.go
  config.go
  connect.go
  query.go
  tui.go
  version.go
internal/
  db/
  tui/
  profile/
  export/
  dberr/
  history/
```

Key packages:

- `internal/db`: driver registry, shared query execution, metadata interfaces
- `internal/tui`: screen routing and TUI screens for connect, query, tables, and help
- `internal/profile`: JSON-backed config storage
- `internal/export`: `table`, `json`, `csv`, `tsv` output writers
- `internal/history`: SQLite-backed query history store
- `internal/dberr`: structured database error wrapping

## Error Output Format

Database failures are printed to stderr in a readable multi-line format.

Example:

```text
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

Typical code patterns:

- MySQL: `MYSQL-{errno}`
- PostgreSQL: `PG-{SQLSTATE}`
- Oracle: `ORA-{code}`
- SQLite: `SQLITE_{NAME}({code})`
- Network: `CONN-REFUSED`, `CONN-TIMEOUT`, `CONN-DNS`, `NET-*`

## Current Gaps and Planned Work

These are still missing or only partially implemented.

- Tibero and Cubrid metadata: partially implemented
- Autocomplete refinement: stronger column/context-aware suggestions
- Syntax-highlighting refinement: improve in-editor rendering fidelity
- Test coverage: still needs to grow

## Tests

The repository now has some focused tests for history, pooling helpers, export helpers, and TUI config behavior. The Makefile also includes:

```bash
make test
```

## Development Notes

- The Go module name is `oslo`, not `connect-dbms`.
- Internal imports use `oslo/internal/...`.
- SQLite uses `modernc.org/sqlite`, so CGO is not required.
- The TUI runs in Alt Screen mode and takes over the terminal.

## License

There is currently no license file in the repository. It would be a good idea to define the license policy before public distribution.
