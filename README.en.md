# connect-dbms

[한글 README](./README.md)

`connect-dbms` is a terminal-first database client for multiple RDBMS.  
Run it without arguments to launch the TUI, or use the subcommands as a script-friendly CLI.

## What It Does

- TUI-based database browsing and querying
- Saved session management
- SQL execution
- Query history search
- Result export
- Multiple query tabs
- SSH tunnel based connections

## Supported Databases

| Driver | Status | Notes |
| --- | --- | --- |
| `mysql` | supported | includes MariaDB usage |
| `mariadb` | supported | MySQL alias |
| `postgres` | supported | PostgreSQL |
| `postgresql` | supported | PostgreSQL alias |
| `oracle` | supported | pure Go |
| `sqlite` | supported | file or in-memory |
| `tibero` | supported in ODBC/full build | requires ODBC |
| `cubrid` | supported in ODBC/full build | requires ODBC |

## Release Tracks

### Basic

- pure-Go oriented
- easiest default install path
- intended for MySQL, MariaDB, PostgreSQL, Oracle, and SQLite

### ODBC/Full

- macOS/Linux release with ODBC-enabled features
- includes Tibero and Cubrid support
- still requires the vendor ODBC driver on the target machine

## Quick Start

### 1. Run

```bash
./connect-dbms
```

One-line install/update:

```bash
curl -fsSL https://raw.githubusercontent.com/parkjangwon/connect-dbms/master/install.sh | sh
```

Install the ODBC/full channel:

```bash
curl -fsSL https://raw.githubusercontent.com/parkjangwon/connect-dbms/master/install.sh | CONNECT_DBMS_CHANNEL=odbc sh
```

Uninstall:

```bash
curl -fsSL https://raw.githubusercontent.com/parkjangwon/connect-dbms/master/install.sh | sh -s -- uninstall
```

### 2. Add a session

PostgreSQL:

```bash
./connect-dbms config add \
  --name local-pg \
  --driver postgresql \
  --host 127.0.0.1 \
  --port 5432 \
  --user postgres \
  --database mydb
```

SQLite:

```bash
./connect-dbms config add \
  --name local-sqlite \
  --driver sqlite \
  --database ./data.db
```

SSH tunnel example:

```bash
./connect-dbms config add \
  --name prod-pg \
  --driver postgresql \
  --host 10.0.0.12 \
  --port 5432 \
  --user app \
  --database appdb \
  --ssh-host bastion.example.com \
  --ssh-user ec2-user \
  --ssh-key-path ~/.ssh/id_rsa
```

### 3. Test the connection

```bash
./connect-dbms connect local-pg
```

### 4. Run SQL

```bash
./connect-dbms query --profile local-pg --sql "SELECT now()" --format table
```

## Config File

- Path: `~/.config/connect-dbms/config.json`
- Auto-created on first run if missing
- Passwords are stored as plaintext JSON

For SQLite sessions, put the DB file path in the `database` field.

## TUI Keys

### Global

- `F1`: help
- `Ctrl+Q`: quit
- `Ctrl+N`: new connection

### Query Screen

- `F5` or `Ctrl+E`: run SQL
- `Ctrl+H`: query history
- `Ctrl+S`: export results
- `Ctrl+Space` or `F9`: autocomplete
- `F6`: new tab
- `F7`, `F8`: switch tabs
- `Ctrl+T`: table browser

## Build

Basic build:

```bash
make build
```

ODBC/full build:

```bash
make build-odbc
```

ODBC tests:

```bash
make test-odbc
```

## Release

- Basic config: `.goreleaser.yaml`
- ODBC/full config: `.goreleaser-odbc.yaml`
- GitHub Actions workflow: `.github/workflows/release.yml`

## Notes

- Autocomplete and highlighting are practical first-pass implementations.
- Tibero/Cubrid require the ODBC/full build plus the vendor ODBC driver on the target system.
