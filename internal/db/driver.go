package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"
)

type ConnConfig struct {
	Driver   string
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Options  map[string]string
	DSN      string // raw DSN override

	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeSeconds int

	SSH *SSHConfig
}

type PoolSettings struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type SSHConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	KeyPath  string
}

type Driver interface {
	Name() string
	Open(cfg ConnConfig) (*sql.DB, error)
	BuildDSN(cfg ConnConfig) string
	Meta(db *sql.DB) MetaInspector
}

type MetaInspector interface {
	Databases(ctx context.Context) ([]string, error)
	CurrentDatabase(ctx context.Context) (string, error)
	Tables(ctx context.Context, schema string) ([]TableInfo, error)
	Columns(ctx context.Context, schema, table string) ([]ColumnInfo, error)
	Indexes(ctx context.Context, schema, table string) ([]IndexInfo, error)
	PrimaryKey(ctx context.Context, schema, table string) ([]string, error)
	ForeignKeys(ctx context.Context, schema, table string) ([]FKInfo, error)
	TableRowCount(ctx context.Context, schema, table string) (int64, error)
}

type TableInfo struct {
	Name   string
	Schema string
	Type   string // TABLE, VIEW
}

type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
	Extra    string
}

type IndexInfo struct {
	Name    string
	Columns []string
	Unique  bool
}

type FKInfo struct {
	Name       string
	Columns    []string
	RefTable   string
	RefColumns []string
}

type QueryResult struct {
	Columns  []string
	Rows     [][]interface{}
	Duration time.Duration
	RowCount int64
}

func RunQuery(ctx context.Context, db *sql.DB, query string) (*QueryResult, error) {
	start := time.Now()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("columns error: %w", err)
	}

	var result [][]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		row := make([]interface{}, len(cols))
		for i, v := range vals {
			switch val := v.(type) {
			case []byte:
				row[i] = string(val)
			default:
				row[i] = val
			}
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &QueryResult{
		Columns:  cols,
		Rows:     result,
		Duration: time.Since(start),
		RowCount: int64(len(result)),
	}, nil
}

func RunExec(ctx context.Context, db *sql.DB, query string) (*QueryResult, error) {
	start := time.Now()
	res, err := db.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("exec error: %w", err)
	}
	affected, _ := res.RowsAffected()
	return &QueryResult{
		Duration: time.Since(start),
		RowCount: affected,
	}, nil
}

func ResolvePoolSettings(cfg ConnConfig, defaultOpen, defaultIdle int) PoolSettings {
	settings := PoolSettings{
		MaxOpenConns: defaultOpen,
		MaxIdleConns: defaultIdle,
	}

	if cfg.MaxOpenConns > 0 {
		settings.MaxOpenConns = cfg.MaxOpenConns
	}
	if cfg.MaxIdleConns > 0 {
		settings.MaxIdleConns = cfg.MaxIdleConns
	}
	if settings.MaxOpenConns > 0 && settings.MaxIdleConns > settings.MaxOpenConns {
		settings.MaxIdleConns = settings.MaxOpenConns
	}
	if cfg.ConnMaxLifetimeSeconds > 0 {
		settings.ConnMaxLifetime = time.Duration(cfg.ConnMaxLifetimeSeconds) * time.Second
	}
	return settings
}

func ApplyPoolSettings(db *sql.DB, settings PoolSettings) {
	if settings.MaxOpenConns > 0 {
		db.SetMaxOpenConns(settings.MaxOpenConns)
	}
	if settings.MaxIdleConns >= 0 {
		db.SetMaxIdleConns(settings.MaxIdleConns)
	}
	if settings.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(settings.ConnMaxLifetime)
	}
}

// IsSelectQuery does a rough check if the query is a SELECT-like statement.
func IsSelectQuery(q string) bool {
	for i := 0; i < len(q); i++ {
		c := q[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		// Check first non-whitespace word
		rest := q[i:]
		if len(rest) >= 6 {
			w := rest[:6]
			if w == "SELECT" || w == "select" || w == "Select" {
				return true
			}
		}
		if len(rest) >= 4 {
			w := rest[:4]
			if w == "SHOW" || w == "show" || w == "Show" || w == "DESC" || w == "desc" || w == "Desc" || w == "WITH" || w == "with" || w == "With" {
				return true
			}
		}
		if len(rest) >= 7 {
			w := rest[:7]
			if w == "EXPLAIN" || w == "explain" || w == "Explain" {
				return true
			}
		}
		if len(rest) >= 8 {
			w := rest[:8]
			if w == "DESCRIBE" || w == "describe" || w == "Describe" || w == "PRAGMA s" || w == "pragma s" {
				return true
			}
		}
		return false
	}
	return false
}

// Driver registry
var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

func Register(name string, d Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	drivers[name] = d
}

func Get(name string) (Driver, error) {
	driversMu.RLock()
	defer driversMu.RUnlock()
	d, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf("unknown driver: %s (available: %s)", name, AvailableDrivers())
	}
	return d, nil
}

func AvailableDrivers() string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	var names []string
	for name := range drivers {
		names = append(names, name)
	}
	sort.Strings(names)
	return fmt.Sprintf("%v", names)
}

func ListDrivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	var names []string
	for name := range drivers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
