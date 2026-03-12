package tui

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // purple
	ColorSecondary = lipgloss.Color("#06B6D4") // cyan
	ColorSuccess   = lipgloss.Color("#10B981") // green
	ColorWarning   = lipgloss.Color("#F59E0B") // amber
	ColorError     = lipgloss.Color("#EF4444") // red
	ColorMuted     = lipgloss.Color("#6B7280") // gray
	ColorBg        = lipgloss.Color("#1F2937") // dark bg
	ColorBorder    = lipgloss.Color("#374151") // border gray

	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	StyleSubtitle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	StyleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(lipgloss.Color("#E5E7EB")).
			Padding(0, 1)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StyleSelected = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	StyleNormal = lipgloss.NewStyle()

	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder)

	StyleActiveBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary)

	StyleTableHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary)

	StyleTableCell = lipgloss.NewStyle()

	StyleSidebar = lipgloss.NewStyle().
			Width(24)

	StyleLogo = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)
)
