package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"oslo/internal/db"
	"oslo/internal/profile"
)

func TestConnectedMsgCreatesFirstQueryTab(t *testing.T) {
	store := &profile.Store{}
	app := NewApp(store)

	drv, err := db.Get("sqlite")
	if err != nil {
		t.Fatalf("Get(sqlite) error = %v", err)
	}
	conn, err := drv.Open(db.ConnConfig{DSN: ":memory:"})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	_, _ = app.Update(ConnectedMsg{
		Profile: &profile.Profile{Name: "local-sqlite", Driver: "sqlite", Database: ":memory:"},
		Driver:  drv,
		Conn:    conn,
	})

	if len(app.queryTabs) != 1 {
		t.Fatalf("len(queryTabs) = %d, want 1", len(app.queryTabs))
	}
	if app.activeQueryTab != 0 {
		t.Fatalf("activeQueryTab = %d, want 0", app.activeQueryTab)
	}
}

func TestNewTabKeyAddsQueryTab(t *testing.T) {
	store := &profile.Store{}
	app := NewApp(store)

	drv, err := db.Get("sqlite")
	if err != nil {
		t.Fatalf("Get(sqlite) error = %v", err)
	}
	conn, err := drv.Open(db.ConnConfig{DSN: ":memory:"})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer conn.Close()

	_, _ = app.Update(ConnectedMsg{
		Profile: &profile.Profile{Name: "local-sqlite", Driver: "sqlite", Database: ":memory:"},
		Driver:  drv,
		Conn:    conn,
	})

	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyF6})

	if len(app.queryTabs) != 2 {
		t.Fatalf("len(queryTabs) = %d, want 2", len(app.queryTabs))
	}
	if app.activeQueryTab != 1 {
		t.Fatalf("activeQueryTab = %d, want 1", app.activeQueryTab)
	}
}
