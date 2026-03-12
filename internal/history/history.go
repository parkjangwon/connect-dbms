package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Entry struct {
	ID          int64
	SessionName string
	Driver      string
	Database    string
	SQL         string
	DurationMS  int64
	RowCount    int64
	Success     bool
	ErrorText   string
	RanAt       time.Time
}

type Store struct {
	db *sql.DB
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "connect-dbms", "history.db")
}

func Open(path string) (*Store, error) {
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create history dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open history db: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.init(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Add(entry Entry) error {
	if entry.RanAt.IsZero() {
		entry.RanAt = time.Now().UTC()
	}

	_, err := s.db.Exec(`
		INSERT INTO query_history (
			session_name, driver, database_name, sql_text,
			duration_ms, row_count, success, error_text, ran_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		entry.SessionName,
		entry.Driver,
		entry.Database,
		entry.SQL,
		entry.DurationMS,
		entry.RowCount,
		boolToInt(entry.Success),
		entry.ErrorText,
		entry.RanAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert history entry: %w", err)
	}
	return nil
}

func (s *Store) Search(term string, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 50
	}

	term = strings.TrimSpace(strings.ToLower(term))
	args := []interface{}{limit}
	query := `
		SELECT id, session_name, driver, database_name, sql_text,
		       duration_ms, row_count, success, COALESCE(error_text, ''), ran_at
		FROM query_history
	`

	if term != "" {
		pattern := "%" + term + "%"
		query += `
			WHERE lower(session_name) LIKE ? OR lower(driver) LIKE ? OR
			      lower(database_name) LIKE ? OR lower(sql_text) LIKE ? OR
			      lower(COALESCE(error_text, '')) LIKE ?
		`
		args = []interface{}{pattern, pattern, pattern, pattern, pattern, limit}
	}

	query += ` ORDER BY ran_at DESC LIMIT ?`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("search history: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		var success int
		var ranAt string
		if err := rows.Scan(
			&entry.ID,
			&entry.SessionName,
			&entry.Driver,
			&entry.Database,
			&entry.SQL,
			&entry.DurationMS,
			&entry.RowCount,
			&success,
			&entry.ErrorText,
			&ranAt,
		); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		entry.Success = success == 1
		entry.RanAt, _ = time.Parse(time.RFC3339Nano, ranAt)
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (s *Store) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS query_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_name TEXT NOT NULL,
			driver TEXT NOT NULL,
			database_name TEXT,
			sql_text TEXT NOT NULL,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			row_count INTEGER NOT NULL DEFAULT 0,
			success INTEGER NOT NULL DEFAULT 1,
			error_text TEXT,
			ran_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("create history table: %w", err)
	}

	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_query_history_ran_at
		ON query_history (ran_at DESC)
	`)
	if err != nil {
		return fmt.Errorf("create history index: %w", err)
	}

	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
