package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type HelpScreen struct {
	width, height int
}

func NewHelpScreen() *HelpScreen {
	return &HelpScreen{}
}

func (s *HelpScreen) View() string {
	var sb strings.Builder

	sb.WriteString(StyleTitle.Render("  Keys") + "\n\n")

	keys := []struct{ key, desc string }{
		{"F5 / Ctrl+E", "Run SQL query"},
		{"F1", "Show / hide help"},
		{"Tab", "Switch between panels"},
		{"Ctrl+H", "Open query history"},
		{"Ctrl+S", "Export query results"},
		{"Ctrl+Space / F9", "Autocomplete"},
		{"F6", "Open new query tab"},
		{"F7 / F8", "Next / previous tab"},
		{"Ctrl+T", "Open table list"},
		{"Ctrl+N", "New connection"},
		{"Ctrl+Q", "Quit"},
		{"Up / Down", "Move cursor"},
		{"PgUp / PgDn", "Scroll results"},
		{"Enter", "Select / confirm"},
		{"Esc", "Go back"},
	}

	for _, k := range keys {
		sb.WriteString("  ")
		sb.WriteString(StyleSubtitle.Render(padRight(k.key, 16)))
		sb.WriteString("  ")
		sb.WriteString(k.desc)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(StyleTitle.Render("  Drivers") + "\n\n")

	for _, d := range availableDrivers() {
		sb.WriteString("  ")
		sb.WriteString(StyleSubtitle.Render(padRight(d, 12)))
		sb.WriteString("  ")
		sb.WriteString(driverDescription(d))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(StyleHelp.Render("  Press F1 or Esc to close"))

	content := sb.String()
	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center,
		StyleBorder.Width(50).Padding(1, 2).Render(content))
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func driverDescription(driver string) string {
	switch driver {
	case "mysql":
		return "MySQL 5.x - 8.x"
	case "mariadb":
		return "MariaDB 5.x - 12.x"
	case "postgres", "postgresql":
		return "PostgreSQL 9+"
	case "oracle":
		return "Oracle 11g - 23c"
	case "sqlite":
		return "SQLite 3"
	case "tibero":
		return "Tibero (ODBC, build tag)"
	case "cubrid":
		return "Cubrid (ODBC, build tag)"
	default:
		return "Registered driver"
	}
}
