//go:build cubrid

package db

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// CubridDriver uses ODBC to connect to Cubrid databases.
// Build with: go build -tags cubrid
// Requires Cubrid ODBC driver installed on the system.
type CubridDriver struct{}

func init() {
	Register("cubrid", &CubridDriver{})
}

func (d *CubridDriver) Name() string { return "cubrid" }

func (d *CubridDriver) BuildDSN(cfg ConnConfig) string {
	if cfg.DSN != "" {
		return cfg.DSN
	}
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 33000
	}
	return fmt.Sprintf("DRIVER={CUBRID Driver};SERVER=%s;PORT=%d;DB=%s;UID=%s;PWD=%s",
		host, port, cfg.Database, cfg.User, cfg.Password)
}

func (d *CubridDriver) Open(cfg ConnConfig) (*sql.DB, error) {
	dsn := d.BuildDSN(cfg)
	db, err := sql.Open("odbc", dsn)
	if err != nil {
		return nil, fmt.Errorf("cubrid open: %w", err)
	}
	return db, nil
}

func (d *CubridDriver) Meta(db *sql.DB) MetaInspector {
	return &cubridMeta{db: db}
}

type cubridMeta struct {
	db *sql.DB
}

func (m *cubridMeta) Databases(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *cubridMeta) CurrentDatabase(ctx context.Context) (string, error) {
	return "", nil
}

func (m *cubridMeta) Tables(ctx context.Context, schema string) ([]TableInfo, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT class_name, class_type FROM db_class WHERE is_system_class = 'NO' ORDER BY class_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		var classType string
		if err := rows.Scan(&t.Name, &classType); err != nil {
			return nil, err
		}
		if classType == "VCLASS" {
			t.Type = "VIEW"
		} else {
			t.Type = "TABLE"
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (m *cubridMeta) Columns(ctx context.Context, schema, table string) ([]ColumnInfo, error) {
	query := `SELECT attr_name, data_type, is_nullable, COALESCE(default_value,'')
		FROM db_attribute WHERE class_name = ? ORDER BY def_order`
	rows, err := m.db.QueryContext(ctx, query, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []ColumnInfo
	for rows.Next() {
		var c ColumnInfo
		var nullable string
		if err := rows.Scan(&c.Name, &c.Type, &nullable, &c.Default); err != nil {
			return nil, err
		}
		c.Nullable = nullable == "YES"
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func (m *cubridMeta) Indexes(ctx context.Context, schema, table string) ([]IndexInfo, error) {
	query := `SELECT i.index_name, i.is_unique, k.key_attr_name
		FROM db_index i
		JOIN db_index_key k ON i.index_name = k.index_name AND i.class_name = k.class_name
		WHERE i.class_name = ?
		ORDER BY i.index_name, k.key_order`
	rows, err := m.db.QueryContext(ctx, query, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*IndexInfo)
	var order []string
	for rows.Next() {
		var name, isUnique, column string
		if err := rows.Scan(&name, &isUnique, &column); err != nil {
			return nil, err
		}
		if idx, ok := indexMap[name]; ok {
			idx.Columns = append(idx.Columns, column)
		} else {
			indexMap[name] = &IndexInfo{
				Name:    name,
				Columns: []string{column},
				Unique:  strings.EqualFold(isUnique, "YES"),
			}
			order = append(order, name)
		}
	}

	var indexes []IndexInfo
	for _, name := range order {
		indexes = append(indexes, *indexMap[name])
	}
	return indexes, rows.Err()
}

func (m *cubridMeta) PrimaryKey(ctx context.Context, schema, table string) ([]string, error) {
	query := `SELECT k.key_attr_name
		FROM db_index i
		JOIN db_index_key k ON i.index_name = k.index_name AND i.class_name = k.class_name
		WHERE i.class_name = ? AND i.is_primary_key = 'YES'
		ORDER BY k.key_order`
	rows, err := m.db.QueryContext(ctx, query, table)
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

func (m *cubridMeta) ForeignKeys(ctx context.Context, schema, table string) ([]FKInfo, error) {
	var ddl string
	if err := m.db.QueryRowContext(ctx, fmt.Sprintf("SHOW CREATE TABLE [%s]", table)).Scan(&ddl); err != nil {
		return nil, err
	}
	return parseCubridForeignKeys(ddl), nil
}

func (m *cubridMeta) TableRowCount(ctx context.Context, schema, table string) (int64, error) {
	var count int64
	err := m.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM [%s]", table)).Scan(&count)
	return count, err
}

func parseCubridForeignKeys(createSQL string) []FKInfo {
	re := regexp.MustCompile(`(?i)CONSTRAINT\s+\[([^\]]+)\]\s+FOREIGN\s+KEY\s+\(([^)]+)\)\s+REFERENCES\s+\[([^\]]+)\]\s+\(([^)]+)\)`)
	matches := re.FindAllStringSubmatch(createSQL, -1)
	var fks []FKInfo
	for _, match := range matches {
		fks = append(fks, FKInfo{
			Name:       match[1],
			Columns:    splitBracketColumns(match[2]),
			RefTable:   match[3],
			RefColumns: splitBracketColumns(match[4]),
		})
	}
	return fks
}

func splitBracketColumns(s string) []string {
	parts := strings.Split(s, ",")
	cols := make([]string, 0, len(parts))
	for _, part := range parts {
		col := strings.TrimSpace(part)
		col = strings.TrimPrefix(col, "[")
		col = strings.TrimSuffix(col, "]")
		if col != "" {
			cols = append(cols, col)
		}
	}
	return cols
}
