package history

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSaveAndSearch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	first := Entry{
		SessionName: "local-pg",
		Driver:      "postgresql",
		SQL:         "SELECT 1",
		DurationMS:  100,
		RanAt:       time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC),
	}
	second := Entry{
		SessionName: "local-pg",
		Driver:      "postgresql",
		SQL:         "SELECT * FROM users",
		DurationMS:  200,
		RanAt:       time.Date(2026, 3, 12, 10, 1, 0, 0, time.UTC),
	}

	if err := store.Add(first); err != nil {
		t.Fatalf("Add(first) error = %v", err)
	}
	if err := store.Add(second); err != nil {
		t.Fatalf("Add(second) error = %v", err)
	}

	got, err := store.Search("users", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(Search()) = %d, want 1", len(got))
	}
	if got[0].SQL != second.SQL {
		t.Fatalf("Search()[0].SQL = %q, want %q", got[0].SQL, second.SQL)
	}
}

func TestStoreRecentReturnsNewestFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")

	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	entries := []Entry{
		{
			SessionName: "one",
			Driver:      "sqlite",
			SQL:         "SELECT 1",
			RanAt:       time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC),
		},
		{
			SessionName: "two",
			Driver:      "sqlite",
			SQL:         "SELECT 2",
			RanAt:       time.Date(2026, 3, 12, 10, 2, 0, 0, time.UTC),
		},
	}

	for _, entry := range entries {
		if err := store.Add(entry); err != nil {
			t.Fatalf("Add(%q) error = %v", entry.SQL, err)
		}
	}

	got, err := store.Search("", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(Search()) = %d, want 2", len(got))
	}
	if got[0].SQL != "SELECT 2" {
		t.Fatalf("Search()[0].SQL = %q, want newest entry first", got[0].SQL)
	}
}
