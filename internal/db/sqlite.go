package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type SQLiteDriver struct{}

func init() {
	Register("sqlite", &SQLiteDriver{})
}

func (d *SQLiteDriver) Name() string { return "sqlite" }

func (d *SQLiteDriver) BuildDSN(cfg ConnConfig) string {
	if cfg.DSN != "" {
		return cfg.DSN
	}
	if cfg.Database != "" {
		return cfg.Database
	}
	return ":memory:"
}

func (d *SQLiteDriver) Open(cfg ConnConfig) (*sql.DB, error) {
	dsn := d.BuildDSN(cfg)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}
	ApplyPoolSettings(db, ResolvePoolSettings(cfg, 1, 1)) // SQLite is single-writer
	return db, nil
}

func (d *SQLiteDriver) Meta(db *sql.DB) MetaInspector {
	return &sqliteMeta{db: db}
}

type sqliteMeta struct {
	db *sql.DB
}

func (m *sqliteMeta) Databases(ctx context.Context) ([]string, error) {
	return []string{"main"}, nil
}

func (m *sqliteMeta) CurrentDatabase(ctx context.Context) (string, error) {
	return "main", nil
}

func (m *sqliteMeta) Tables(ctx context.Context, schema string) ([]TableInfo, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT name, type FROM sqlite_master WHERE type IN ('table','view') ORDER BY name")
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
		t.Schema = "main"
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (m *sqliteMeta) Columns(ctx context.Context, schema, table string) ([]ColumnInfo, error) {
	rows, err := m.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info('%s')", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		extra := ""
		if pk > 0 {
			extra = "PRIMARY KEY"
		}
		cols = append(cols, ColumnInfo{
			Name:     name,
			Type:     typ,
			Nullable: notnull == 0,
			Default:  dflt.String,
			Extra:    extra,
		})
	}
	return cols, rows.Err()
}

func (m *sqliteMeta) Indexes(ctx context.Context, schema, table string) ([]IndexInfo, error) {
	rows, err := m.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list('%s')", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin, partial string
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}

		// Get columns for this index
		colRows, err := m.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_info('%s')", name))
		if err != nil {
			return nil, err
		}
		var columns []string
		for colRows.Next() {
			var seqno, cid int
			var colName string
			if err := colRows.Scan(&seqno, &cid, &colName); err != nil {
				colRows.Close()
				return nil, err
			}
			columns = append(columns, colName)
		}
		colRows.Close()

		indexes = append(indexes, IndexInfo{
			Name:    name,
			Columns: columns,
			Unique:  unique == 1,
		})
	}
	return indexes, rows.Err()
}

func (m *sqliteMeta) PrimaryKey(ctx context.Context, schema, table string) ([]string, error) {
	cols, err := m.Columns(ctx, schema, table)
	if err != nil {
		return nil, err
	}
	var pks []string
	for _, c := range cols {
		if c.Extra == "PRIMARY KEY" {
			pks = append(pks, c.Name)
		}
	}
	return pks, nil
}

func (m *sqliteMeta) ForeignKeys(ctx context.Context, schema, table string) ([]FKInfo, error) {
	rows, err := m.db.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list('%s')", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkMap := make(map[int]*FKInfo)
	for rows.Next() {
		var id, seq int
		var refTable, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}
		if fk, ok := fkMap[id]; ok {
			fk.Columns = append(fk.Columns, from)
			fk.RefColumns = append(fk.RefColumns, to)
		} else {
			fkMap[id] = &FKInfo{
				Name:       fmt.Sprintf("fk_%d", id),
				Columns:    []string{from},
				RefTable:   refTable,
				RefColumns: []string{to},
			}
		}
	}

	var fks []FKInfo
	for _, fk := range fkMap {
		fks = append(fks, *fk)
	}
	return fks, rows.Err()
}

func (m *sqliteMeta) TableRowCount(ctx context.Context, schema, table string) (int64, error) {
	var count int64
	err := m.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", table)).Scan(&count)
	return count, err
}
