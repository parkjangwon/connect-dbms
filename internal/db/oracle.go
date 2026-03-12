package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/sijms/go-ora/v2"
)

type OracleDriver struct{}

func init() {
	Register("oracle", &OracleDriver{})
}

func (d *OracleDriver) Name() string { return "oracle" }

func (d *OracleDriver) BuildDSN(cfg ConnConfig) string {
	if cfg.DSN != "" {
		return cfg.DSN
	}
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 1521
	}
	service := cfg.Database
	if service == "" {
		service = "ORCL"
	}
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		cfg.User, cfg.Password, host, port, service)
}

func (d *OracleDriver) Open(cfg ConnConfig) (*sql.DB, error) {
	dsn := d.BuildDSN(cfg)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("oracle open: %w", err)
	}
	ApplyPoolSettings(db, ResolvePoolSettings(cfg, 5, 2))
	return db, nil
}

func (d *OracleDriver) Meta(db *sql.DB) MetaInspector {
	return &oracleMeta{db: db}
}

type oracleMeta struct {
	db *sql.DB
}

func (m *oracleMeta) Databases(ctx context.Context) ([]string, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT username FROM all_users ORDER BY username")
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

func (m *oracleMeta) CurrentDatabase(ctx context.Context) (string, error) {
	var name string
	err := m.db.QueryRowContext(ctx, "SELECT SYS_CONTEXT('USERENV','CURRENT_SCHEMA') FROM DUAL").Scan(&name)
	return name, err
}

func (m *oracleMeta) Tables(ctx context.Context, schema string) ([]TableInfo, error) {
	query := `SELECT table_name, 'TABLE' FROM all_tables WHERE owner = NVL(:1, SYS_CONTEXT('USERENV','CURRENT_SCHEMA'))
		UNION ALL
		SELECT view_name, 'VIEW' FROM all_views WHERE owner = NVL(:2, SYS_CONTEXT('USERENV','CURRENT_SCHEMA'))
		ORDER BY 1`
	if schema == "" {
		schema = ""
	}
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

func (m *oracleMeta) Columns(ctx context.Context, schema, table string) ([]ColumnInfo, error) {
	query := `SELECT column_name, data_type, nullable, NVL(data_default,' ')
		FROM all_tab_columns
		WHERE owner = NVL(:1, SYS_CONTEXT('USERENV','CURRENT_SCHEMA')) AND table_name = :2
		ORDER BY column_id`
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

func (m *oracleMeta) Indexes(ctx context.Context, schema, table string) ([]IndexInfo, error) {
	query := `SELECT i.index_name, i.uniqueness, LISTAGG(c.column_name, ',') WITHIN GROUP (ORDER BY c.column_position)
		FROM all_indexes i
		JOIN all_ind_columns c ON i.index_name = c.index_name AND i.owner = c.index_owner
		WHERE i.table_owner = NVL(:1, SYS_CONTEXT('USERENV','CURRENT_SCHEMA')) AND i.table_name = :2
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

func (m *oracleMeta) PrimaryKey(ctx context.Context, schema, table string) ([]string, error) {
	query := `SELECT cols.column_name
		FROM all_constraints cons
		JOIN all_cons_columns cols ON cons.constraint_name = cols.constraint_name AND cons.owner = cols.owner
		WHERE cons.owner = NVL(:1, SYS_CONTEXT('USERENV','CURRENT_SCHEMA'))
		AND cons.table_name = :2 AND cons.constraint_type = 'P'
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

func (m *oracleMeta) ForeignKeys(ctx context.Context, schema, table string) ([]FKInfo, error) {
	query := `SELECT a.constraint_name, a.column_name, c_pk.table_name, b.column_name
		FROM all_cons_columns a
		JOIN all_constraints c ON a.constraint_name = c.constraint_name AND a.owner = c.owner
		JOIN all_constraints c_pk ON c.r_constraint_name = c_pk.constraint_name AND c.r_owner = c_pk.owner
		JOIN all_cons_columns b ON c_pk.constraint_name = b.constraint_name AND b.position = a.position AND c_pk.owner = b.owner
		WHERE c.constraint_type = 'R'
		AND a.owner = NVL(:1, SYS_CONTEXT('USERENV','CURRENT_SCHEMA'))
		AND a.table_name = :2
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

func (m *oracleMeta) TableRowCount(ctx context.Context, schema, table string) (int64, error) {
	q := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", table)
	if schema != "" {
		q = fmt.Sprintf("SELECT COUNT(*) FROM \"%s\".\"%s\"", schema, table)
	}
	var count int64
	err := m.db.QueryRowContext(ctx, q).Scan(&count)
	return count, err
}
