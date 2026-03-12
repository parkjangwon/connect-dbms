# CLAUDE.md

## Project
connect-dbms: TUI + CLI database client for MySQL, MariaDB, PostgreSQL, Oracle, SQLite, Tibero, Cubrid.

## Build
```bash
go build -ldflags "-s -w" -o connect-dbms .
```

## Release Split
- Basic release: `.goreleaser.yaml`
- ODBC/full release: `.goreleaser-odbc.yaml`
- GitHub Actions workflow: `.github/workflows/release.yml`

## Quick Test
```bash
./connect-dbms query --driver sqlite --dsn ":memory:" --sql "SELECT 1" --format json -q
```

## Implemented TUI Extras
- Query history is stored at `~/.config/connect-dbms/history.db`
- `Ctrl+H` opens query history search in the query screen
- `Ctrl+S` exports current tabular results from the query screen
- `F6` opens a new query tab and `F7`/`F8` switch tabs
- `Ctrl+Space` or `F9` opens table-name autocomplete
- SQL preview highlighting is shown below the editor
- SSH tunneling is supported through saved session settings

## Rules
- Go module name is `oslo` (legacy). Binary name is `connect-dbms`.
- Config path: `~/.config/connect-dbms/config.json` (JSON, plaintext).
- Missing config files are auto-created on first run.
- All DB drivers must be pure Go (no CGO) unless gated behind build tags.
- Use `dberr.Wrap()` for all DB error handling — provides error codes + stack traces.
- TUI text must be in English with simple words.
- Do not add YAML dependencies. Config is JSON only.
