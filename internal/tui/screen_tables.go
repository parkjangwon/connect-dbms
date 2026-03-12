package tui

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"oslo/internal/db"
)

type TablesScreen struct {
	width, height int
	conn          *sql.DB
	driver        db.Driver

	tables      []db.TableInfo
	columns     []db.ColumnInfo
	cursor      int
	errMsg      string
	loading     bool
	selectedSQL string
	switchToQuery bool
}

type tablesLoadedMsg struct {
	tables []db.TableInfo
	err    error
}

type columnsLoadedMsg struct {
	columns []db.ColumnInfo
	err     error
}

func NewTablesScreen(conn *sql.DB, drv db.Driver) *TablesScreen {
	return &TablesScreen{
		conn:   conn,
		driver: drv,
	}
}

func (s *TablesScreen) Refresh() tea.Cmd {
	s.loading = true
	meta := s.driver.Meta(s.conn)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		tables, err := meta.Tables(ctx, "")
		return tablesLoadedMsg{tables: tables, err: err}
	}
}

func (s *TablesScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tablesLoadedMsg:
		s.loading = false
		if msg.err != nil {
			s.errMsg = msg.err.Error()
		} else {
			s.tables = msg.tables
			s.errMsg = ""
			if len(s.tables) > 0 {
				return s.loadColumns()
			}
		}
		return nil

	case columnsLoadedMsg:
		if msg.err != nil {
			s.errMsg = msg.err.Error()
		} else {
			s.columns = msg.columns
		}
		return nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Up):
			if s.cursor > 0 {
				s.cursor--
				return s.loadColumns()
			}
		case key.Matches(msg, Keys.Down):
			if s.cursor < len(s.tables)-1 {
				s.cursor++
				return s.loadColumns()
			}
		case key.Matches(msg, Keys.Enter):
			if s.cursor < len(s.tables) {
				t := s.tables[s.cursor]
				s.selectedSQL = fmt.Sprintf("SELECT * FROM %s LIMIT 100", t.Name)
				s.switchToQuery = true
			}
		case key.Matches(msg, Keys.Escape):
			s.switchToQuery = true
			s.selectedSQL = ""
		}
	}
	return nil
}

func (s *TablesScreen) loadColumns() tea.Cmd {
	if s.cursor >= len(s.tables) {
		return nil
	}
	t := s.tables[s.cursor]
	meta := s.driver.Meta(s.conn)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cols, err := meta.Columns(ctx, t.Schema, t.Name)
		return columnsLoadedMsg{columns: cols, err: err}
	}
}

func (s *TablesScreen) View() string {
	if s.loading {
		return StyleHelp.Render("  Loading tables...")
	}

	// Left: table list
	var left strings.Builder
	left.WriteString(StyleSubtitle.Render(" Tables") + "\n\n")

	if s.errMsg != "" {
		left.WriteString(StyleError.Render("  " + s.errMsg) + "\n")
	}

	for i, t := range s.tables {
		icon := "T"
		if t.Type == "VIEW" || t.Type == "view" {
			icon = "V"
		}
		line := fmt.Sprintf("  [%s] %s", icon, t.Name)
		if i == s.cursor {
			left.WriteString(StyleSelected.Render(line) + "\n")
		} else {
			left.WriteString(line + "\n")
		}
	}

	if len(s.tables) == 0 {
		left.WriteString(StyleHelp.Render("  (no tables)"))
	}

	left.WriteString("\n" + StyleHelp.Render("  Enter: query  Esc: back"))

	// Right: columns of selected table
	var right strings.Builder
	if s.cursor < len(s.tables) {
		t := s.tables[s.cursor]
		right.WriteString(StyleSubtitle.Render(fmt.Sprintf(" %s", t.Name)) + "\n\n")

		if len(s.columns) > 0 {
			right.WriteString(StyleTableHeader.Render(
				fmt.Sprintf("  %-20s %-15s %-5s %s", "Column", "Type", "Null", "Extra"),
			) + "\n")
			right.WriteString("  " + strings.Repeat("-", 55) + "\n")
			for _, c := range s.columns {
				null := "NO"
				if c.Nullable {
					null = "YES"
				}
				right.WriteString(fmt.Sprintf("  %-20s %-15s %-5s %s\n",
					truncate(c.Name, 20),
					truncate(c.Type, 15),
					null,
					c.Extra,
				))
			}
		}
	}

	leftW := 32
	rightW := s.width - leftW - 4
	if rightW < 20 {
		rightW = 20
	}

	leftView := StyleBorder.Width(leftW).Height(s.height - 2).Render(left.String())
	rightView := StyleBorder.Width(rightW).Height(s.height - 2).Render(right.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-2] + ".."
	}
	return s
}
