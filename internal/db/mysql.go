package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLDriver struct{}

func init() {
	Register("mysql", &MySQLDriver{})
	Register("mariadb", &MySQLDriver{})
}

func (d *MySQLDriver) Name() string { return "mysql" }

func (d *MySQLDriver) BuildDSN(cfg ConnConfig) string {
	if cfg.DSN != "" {
		return cfg.DSN
	}
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 3306
	}
	// user:password@tcp(host:port)/dbname?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
		cfg.User, cfg.Password, host, port, cfg.Database)
	if charset, ok := cfg.Options["charset"]; ok {
		dsn += "&charset=" + charset
	}
	return dsn
}

func (d *MySQLDriver) Open(cfg ConnConfig) (*sql.DB, error) {
	dsn := d.BuildDSN(cfg)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}
	ApplyPoolSettings(db, ResolvePoolSettings(cfg, 5, 2))
	return db, nil
}

func (d *MySQLDriver) Meta(db *sql.DB) MetaInspector {
	return &mysqlMeta{db: db}
}

type mysqlMeta struct {
	db *sql.DB
}

func (m *mysqlMeta) Databases(ctx context.Context) ([]string, error) {
	rows, err := m.db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dbs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		dbs = append(dbs, name)
	}
	return dbs, rows.Err()
}

func (m *mysqlMeta) CurrentDatabase(ctx context.Context) (string, error) {
	var name string
	err := m.db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&name)
	return name, err
}

func (m *mysqlMeta) Tables(ctx context.Context, schema string) ([]TableInfo, error) {
	query := `SELECT TABLE_NAME, TABLE_TYPE FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME`
	if schema == "" {
		// Use current database
		var err error
		schema, err = m.CurrentDatabase(ctx)
		if err != nil {
			return nil, err
		}
	}
	rows, err := m.db.QueryContext(ctx, query, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Name, &t.Type); err != nil {
			return nil, err
		}
		t.Schema = schema
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (m *mysqlMeta) Columns(ctx context.Context, schema, table string) ([]ColumnInfo, error) {
	if schema == "" {
		var err error
		schema, err = m.CurrentDatabase(ctx)
		if err != nil {
			return nil, err
		}
	}
	query := `SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COALESCE(COLUMN_DEFAULT,''), COALESCE(EXTRA,'')
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`
	rows, err := m.db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var c ColumnInfo
		var nullable string
		if err := rows.Scan(&c.Name, &c.Type, &nullable, &c.Default, &c.Extra); err != nil {
			return nil, err
		}
		c.Nullable = nullable == "YES"
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func (m *mysqlMeta) Indexes(ctx context.Context, schema, table string) ([]IndexInfo, error) {
	if schema == "" {
		var err error
		schema, err = m.CurrentDatabase(ctx)
		if err != nil {
			return nil, err
		}
	}
	query := `SELECT INDEX_NAME, COLUMN_NAME, NON_UNIQUE
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX`
	rows, err := m.db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	idxMap := make(map[string]*IndexInfo)
	var order []string
	for rows.Next() {
		var name, col string
		var nonUnique int
		if err := rows.Scan(&name, &col, &nonUnique); err != nil {
			return nil, err
		}
		if idx, ok := idxMap[name]; ok {
			idx.Columns = append(idx.Columns, col)
		} else {
			idxMap[name] = &IndexInfo{
				Name:    name,
				Columns: []string{col},
				Unique:  nonUnique == 0,
			}
			order = append(order, name)
		}
	}

	var indexes []IndexInfo
	for _, name := range order {
		indexes = append(indexes, *idxMap[name])
	}
	return indexes, rows.Err()
}

func (m *mysqlMeta) PrimaryKey(ctx context.Context, schema, table string) ([]string, error) {
	if schema == "" {
		var err error
		schema, err = m.CurrentDatabase(ctx)
		if err != nil {
			return nil, err
		}
	}
	query := `SELECT COLUMN_NAME FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND CONSTRAINT_NAME = 'PRIMARY'
		ORDER BY ORDINAL_POSITION`
	rows, err := m.db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pks []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		pks = append(pks, col)
	}
	return pks, rows.Err()
}

func (m *mysqlMeta) ForeignKeys(ctx context.Context, schema, table string) ([]FKInfo, error) {
	if schema == "" {
		var err error
		schema, err = m.CurrentDatabase(ctx)
		if err != nil {
			return nil, err
		}
	}
	query := `SELECT CONSTRAINT_NAME, COLUMN_NAME, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY CONSTRAINT_NAME, ORDINAL_POSITION`
	rows, err := m.db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkMap := make(map[string]*FKInfo)
	var order []string
	for rows.Next() {
		var name, col, refTable, refCol string
		if err := rows.Scan(&name, &col, &refTable, &refCol); err != nil {
			return nil, err
		}
		if fk, ok := fkMap[name]; ok {
			fk.Columns = append(fk.Columns, col)
			fk.RefColumns = append(fk.RefColumns, refCol)
		} else {
			fkMap[name] = &FKInfo{
				Name:       name,
				Columns:    []string{col},
				RefTable:   refTable,
				RefColumns: []string{refCol},
			}
			order = append(order, name)
		}
	}

	var fks []FKInfo
	for _, name := range order {
		fks = append(fks, *fkMap[name])
	}
	return fks, rows.Err()
}

func (m *mysqlMeta) TableRowCount(ctx context.Context, schema, table string) (int64, error) {
	if schema == "" {
		var err error
		schema, err = m.CurrentDatabase(ctx)
		if err != nil {
			return 0, err
		}
	}
	var count int64
	err := m.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`", schema, table)).Scan(&count)
	return count, err
}
