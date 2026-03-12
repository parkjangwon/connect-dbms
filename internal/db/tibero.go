//go:build tibero

package db

import (
	"context"
	"database/sql"
	"fmt"
)

// TiberoDriver uses ODBC to connect to Tibero databases.
// Build with: go build -tags tibero
// Requires Tibero ODBC driver installed on the system.
type TiberoDriver struct{}

func init() {
	Register("tibero", &TiberoDriver{})
}

func (d *TiberoDriver) Name() string { return "tibero" }

func (d *TiberoDriver) BuildDSN(cfg ConnConfig) string {
	if cfg.DSN != "" {
		return cfg.DSN
	}
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 8629
	}
	return fmt.Sprintf("DRIVER={Tibero 6 ODBC Driver};SERVER=%s;PORT=%d;DB=%s;UID=%s;PWD=%s",
		host, port, cfg.Database, cfg.User, cfg.Password)
}

func (d *TiberoDriver) Open(cfg ConnConfig) (*sql.DB, error) {
	dsn := d.BuildDSN(cfg)
	db, err := sql.Open("odbc", dsn)
	if err != nil {
		return nil, fmt.Errorf("tibero open: %w", err)
	}
	return db, nil
}

func (d *TiberoDriver) Meta(db *sql.DB) MetaInspector {
	return &tiberoMeta{db: db}
}

type tiberoMeta struct {
	db *sql.DB
}

func (m *tiberoMeta) Databases(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *tiberoMeta) CurrentDatabase(ctx context.Context) (string, error) {
	var name string
	err := m.db.QueryRowContext(ctx, "SELECT SYS_CONTEXT('USERENV','CURRENT_SCHEMA') FROM DUAL").Scan(&name)
	return name, err
}

func (m *tiberoMeta) Tables(ctx context.Context, schema string) ([]TableInfo, error) {
	query := `SELECT TABLE_NAME, 'TABLE' FROM ALL_TABLES WHERE OWNER = NVL(?, USER)
		UNION ALL
		SELECT VIEW_NAME, 'VIEW' FROM ALL_VIEWS WHERE OWNER = NVL(?, USER)
		ORDER BY 1`
	rows, err := m.db.QueryContext(ctx, query, schema, schema)
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

func (m *tiberoMeta) Columns(ctx context.Context, schema, table string) ([]ColumnInfo, error) {
	query := `SELECT COLUMN_NAME, DATA_TYPE, NULLABLE, NVL(DATA_DEFAULT,' ')
		FROM ALL_TAB_COLUMNS
		WHERE OWNER = NVL(?, USER) AND TABLE_NAME = ?
		ORDER BY COLUMN_ID`
	rows, err := m.db.QueryContext(ctx, query, schema, table)
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
		c.Nullable = nullable == "Y"
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func (m *tiberoMeta) Indexes(ctx context.Context, schema, table string) ([]IndexInfo, error) {
	query := `SELECT i.index_name, i.uniqueness, LISTAGG(c.column_name, ',') WITHIN GROUP (ORDER BY c.column_position)
		FROM all_indexes i
		JOIN all_ind_columns c ON i.index_name = c.index_name AND i.owner = c.index_owner
		WHERE i.table_owner = NVL(?, USER) AND i.table_name = ?
		GROUP BY i.index_name, i.uniqueness
		ORDER BY i.index_name`
	rows, err := m.db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		var uniqueness, colStr string
		if err := rows.Scan(&idx.Name, &uniqueness, &colStr); err != nil {
			return nil, err
		}
		idx.Unique = uniqueness == "UNIQUE"
		idx.Columns = splitComma(colStr)
		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

func (m *tiberoMeta) PrimaryKey(ctx context.Context, schema, table string) ([]string, error) {
	query := `SELECT cols.column_name
		FROM all_constraints cons
		JOIN all_cons_columns cols ON cons.constraint_name = cols.constraint_name AND cons.owner = cols.owner
		WHERE cons.owner = NVL(?, USER)
		AND cons.table_name = ? AND cons.constraint_type = 'P'
		ORDER BY cols.position`
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

func (m *tiberoMeta) ForeignKeys(ctx context.Context, schema, table string) ([]FKInfo, error) {
	query := `SELECT a.constraint_name, a.column_name, c_pk.table_name, b.column_name
		FROM all_cons_columns a
		JOIN all_constraints c ON a.constraint_name = c.constraint_name AND a.owner = c.owner
		JOIN all_constraints c_pk ON c.r_constraint_name = c_pk.constraint_name AND c.r_owner = c_pk.owner
		JOIN all_cons_columns b ON c_pk.constraint_name = b.constraint_name AND b.position = a.position AND c_pk.owner = b.owner
		WHERE c.constraint_type = 'R'
		AND a.owner = NVL(?, USER)
		AND a.table_name = ?
		ORDER BY a.constraint_name, a.position`
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

func (m *tiberoMeta) TableRowCount(ctx context.Context, schema, table string) (int64, error) {
	var count int64
	q := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", table)
	if schema != "" {
		q = fmt.Sprintf("SELECT COUNT(*) FROM \"%s\".\"%s\"", schema, table)
	}
	err := m.db.QueryRowContext(ctx, q).Scan(&count)
	return count, err
}
