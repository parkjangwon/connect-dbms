package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresDriver struct{}

func init() {
	Register("postgres", &PostgresDriver{})
	Register("postgresql", &PostgresDriver{})
}

func (d *PostgresDriver) Name() string { return "postgres" }

func (d *PostgresDriver) BuildDSN(cfg ConnConfig) string {
	if cfg.DSN != "" {
		return cfg.DSN
	}
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 5432
	}
	sslmode := "disable"
	if v, ok := cfg.Options["sslmode"]; ok {
		sslmode = v
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, host, port, cfg.Database, sslmode)
}

func (d *PostgresDriver) Open(cfg ConnConfig) (*sql.DB, error) {
	dsn := d.BuildDSN(cfg)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}
	ApplyPoolSettings(db, ResolvePoolSettings(cfg, 5, 2))
	return db, nil
}

func (d *PostgresDriver) Meta(db *sql.DB) MetaInspector {
	return &pgMeta{db: db}
}

type pgMeta struct {
	db *sql.DB
}

func (m *pgMeta) Databases(ctx context.Context) ([]string, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname")
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

func (m *pgMeta) CurrentDatabase(ctx context.Context) (string, error) {
	var name string
	err := m.db.QueryRowContext(ctx, "SELECT current_database()").Scan(&name)
	return name, err
}

func (m *pgMeta) Tables(ctx context.Context, schema string) ([]TableInfo, error) {
	if schema == "" {
		schema = "public"
	}
	query := `SELECT table_name, table_type FROM information_schema.tables
		WHERE table_schema = $1 ORDER BY table_name`
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

func (m *pgMeta) Columns(ctx context.Context, schema, table string) ([]ColumnInfo, error) {
	if schema == "" {
		schema = "public"
	}
	query := `SELECT column_name, data_type, is_nullable, COALESCE(column_default,'')
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position`
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
		c.Nullable = nullable == "YES"
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func (m *pgMeta) Indexes(ctx context.Context, schema, table string) ([]IndexInfo, error) {
	if schema == "" {
		schema = "public"
	}
	query := `SELECT i.relname, ix.indisunique, array_to_string(ARRAY(
			SELECT a.attname FROM unnest(ix.indkey) WITH ORDINALITY AS k(attnum, ord)
			JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
			ORDER BY k.ord
		), ',')
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = $1 AND t.relname = $2`
	rows, err := m.db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		var colStr string
		if err := rows.Scan(&idx.Name, &idx.Unique, &colStr); err != nil {
			return nil, err
		}
		if colStr != "" {
			for _, c := range splitComma(colStr) {
				idx.Columns = append(idx.Columns, c)
			}
		}
		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

func splitComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	return result
}

func (m *pgMeta) PrimaryKey(ctx context.Context, schema, table string) ([]string, error) {
	if schema == "" {
		schema = "public"
	}
	query := `SELECT a.attname
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1 AND t.relname = $2 AND ix.indisprimary
		ORDER BY array_position(ix.indkey, a.attnum)`
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

func (m *pgMeta) ForeignKeys(ctx context.Context, schema, table string) ([]FKInfo, error) {
	if schema == "" {
		schema = "public"
	}
	query := `SELECT
		tc.constraint_name,
		kcu.column_name,
		ccu.table_name AS ref_table,
		ccu.column_name AS ref_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu ON tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_schema = $1 AND tc.table_name = $2
		ORDER BY tc.constraint_name, kcu.ordinal_position`
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

func (m *pgMeta) TableRowCount(ctx context.Context, schema, table string) (int64, error) {
	if schema == "" {
		schema = "public"
	}
	var count int64
	err := m.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM "%s"."%s"`, schema, table)).Scan(&count)
	return count, err
}
