package tui

import (
	"database/sql"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"oslo/internal/db"
	"oslo/internal/profile"
)

type Screen int

const (
	ScreenConnect Screen = iota
	ScreenQuery
	ScreenTables
	ScreenHelp
)

type App struct {
	width  int
	height int

	screen     Screen
	prevScreen Screen

	// Connection state
	store      *profile.Store
	profile    *profile.Profile
	driver     db.Driver
	conn       *sql.DB
	tunnel     io.Closer
	connStatus string

	// Sub-models
	connectScreen  *ConnectScreen
	queryTabs      []*QueryScreen
	activeQueryTab int
	tablesScreen   *TablesScreen
	helpScreen     *HelpScreen
}

func NewApp(store *profile.Store) *App {
	a := &App{
		store:  store,
		screen: ScreenConnect,
	}
	a.connectScreen = NewConnectScreen(store)
	a.helpScreen = NewHelpScreen()
	return a
}

func (a *App) Init() tea.Cmd {
	return a.connectScreen.Init()
}

// Messages
type ConnectedMsg struct {
	Profile *profile.Profile
	Driver  db.Driver
	Conn    *sql.DB
	Tunnel  io.Closer
}

type ErrorMsg struct {
	Err error
}

type StatusMsg string

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		// Global keys
		if key.Matches(msg, Keys.Quit) {
			a.closeQueryTabs()
			if a.conn != nil {
				a.conn.Close()
			}
			if a.tunnel != nil {
				a.tunnel.Close()
			}
			return a, tea.Quit
		}

		if a.screen == ScreenConnect && msg.String() == "q" {
			a.closeQueryTabs()
			if a.conn != nil {
				a.conn.Close()
			}
			if a.tunnel != nil {
				a.tunnel.Close()
			}
			return a, tea.Quit
		}

		if key.Matches(msg, Keys.Help) && a.screen != ScreenHelp {
			a.prevScreen = a.screen
			a.screen = ScreenHelp
			return a, nil
		}

		if a.screen == ScreenHelp && (key.Matches(msg, Keys.Escape) || key.Matches(msg, Keys.Help)) {
			a.screen = a.prevScreen
			return a, nil
		}

		// Screen-specific navigation (only when connected)
		if a.conn != nil {
			if key.Matches(msg, Keys.Tables) && a.screen != ScreenTables {
				a.screen = ScreenTables
				if a.tablesScreen != nil {
					return a, a.tablesScreen.Refresh()
				}
				return a, nil
			}
			if a.screen == ScreenQuery && key.Matches(msg, Keys.NewTab) {
				a.newQueryTab("")
				return a, nil
			}
			if a.screen == ScreenQuery && key.Matches(msg, Keys.NextTab) {
				a.nextQueryTab()
				return a, nil
			}
			if a.screen == ScreenQuery && key.Matches(msg, Keys.PrevTab) {
				a.prevQueryTab()
				return a, nil
			}
			if key.Matches(msg, Keys.NewConn) {
				if a.conn != nil {
					a.conn.Close()
					a.conn = nil
				}
				if a.tunnel != nil {
					a.tunnel.Close()
					a.tunnel = nil
				}
				a.closeQueryTabs()
				a.queryTabs = nil
				a.activeQueryTab = 0
				a.screen = ScreenConnect
				a.connectScreen = NewConnectScreen(a.store)
				return a, a.connectScreen.Init()
			}
		}

	case ConnectedMsg:
		a.closeQueryTabs()
		a.profile = msg.Profile
		a.driver = msg.Driver
		a.conn = msg.Conn
		a.tunnel = msg.Tunnel
		a.connStatus = fmt.Sprintf("%s @ %s", msg.Profile.Name, msg.Profile.Driver)
		a.queryTabs = []*QueryScreen{
			NewQueryScreen(a.conn, a.driver, a.profile),
		}
		a.activeQueryTab = 0
		a.tablesScreen = NewTablesScreen(a.conn, a.driver)
		a.screen = ScreenQuery
		return a, a.currentQueryScreen().Init()

	case ErrorMsg:
		if a.screen == ScreenConnect {
			a.connectScreen.SetError(msg.Err.Error())
		}
		return a, nil
	}

	// Delegate to current screen
	var cmd tea.Cmd
	switch a.screen {
	case ScreenConnect:
		cmd = a.connectScreen.Update(msg)
	case ScreenQuery:
		if queryScreen := a.currentQueryScreen(); queryScreen != nil {
			cmd = queryScreen.Update(msg)
		}
	case ScreenTables:
		if a.tablesScreen != nil {
			cmd = a.tablesScreen.Update(msg)
			// Check if tables screen wants to switch to query
			if a.tablesScreen.switchToQuery {
				a.tablesScreen.switchToQuery = false
				a.screen = ScreenQuery
				if queryScreen := a.currentQueryScreen(); queryScreen != nil {
					queryScreen.SetSQL(a.tablesScreen.selectedSQL)
				}
			}
		}
	case ScreenHelp:
		// handled above
	}

	return a, cmd
}

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	var content string
	mainH := a.height - 2 // status bar

	switch a.screen {
	case ScreenConnect:
		a.connectScreen.width = a.width
		a.connectScreen.height = mainH
		content = a.connectScreen.View()
	case ScreenQuery:
		if queryScreen := a.currentQueryScreen(); queryScreen != nil {
			tabBarHeight := 0
			if len(a.queryTabs) > 1 {
				tabBarHeight = 1
			}
			queryScreen.width = a.width
			queryScreen.height = mainH - tabBarHeight
			content = queryScreen.View()
			if len(a.queryTabs) > 1 {
				content = lipgloss.JoinVertical(lipgloss.Left, a.queryTabBar(), content)
			}
		}
	case ScreenTables:
		if a.tablesScreen != nil {
			a.tablesScreen.width = a.width
			a.tablesScreen.height = mainH
			content = a.tablesScreen.View()
		}
	case ScreenHelp:
		a.helpScreen.width = a.width
		a.helpScreen.height = mainH
		content = a.helpScreen.View()
	}

	// Status bar
	status := a.statusBar()

	return lipgloss.JoinVertical(lipgloss.Left, content, status)
}

func (a *App) statusBar() string {
	left := " connect-dbms"
	if a.connStatus != "" {
		left += " | " + a.connStatus
	}

	right := "F1:Help Ctrl+Q:Quit"
	if a.conn != nil {
		if a.screen == ScreenQuery {
			right = "Ctrl+H:History Ctrl+S:Export Ctrl+Space/F9:Auto F6:NewTab F7/F8:Tabs Ctrl+T:Tables Ctrl+N:NewConn | " + right
		} else {
			right = "Ctrl+T:Tables Ctrl+N:NewConn | " + right
		}
	}

	pad := a.width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}

	bar := left + fmt.Sprintf("%*s", pad, "") + right
	return StyleStatusBar.Width(a.width).Render(bar)
}

func (a *App) currentQueryScreen() *QueryScreen {
	if len(a.queryTabs) == 0 || a.activeQueryTab >= len(a.queryTabs) {
		return nil
	}
	return a.queryTabs[a.activeQueryTab]
}

func (a *App) newQueryTab(initialSQL string) {
	tab := NewQueryScreen(a.conn, a.driver, a.profile)
	if initialSQL != "" {
		tab.SetSQL(initialSQL)
	}
	a.queryTabs = append(a.queryTabs, tab)
	a.activeQueryTab = len(a.queryTabs) - 1
}

func (a *App) nextQueryTab() {
	if len(a.queryTabs) < 2 {
		return
	}
	a.activeQueryTab = (a.activeQueryTab + 1) % len(a.queryTabs)
}

func (a *App) prevQueryTab() {
	if len(a.queryTabs) < 2 {
		return
	}
	a.activeQueryTab--
	if a.activeQueryTab < 0 {
		a.activeQueryTab = len(a.queryTabs) - 1
	}
}

func (a *App) closeQueryTabs() {
	for _, tab := range a.queryTabs {
		if tab != nil {
			tab.Close()
		}
	}
}

func (a *App) queryTabBar() string {
	var tabs []string
	for i := range a.queryTabs {
		label := fmt.Sprintf(" Tab %d ", i+1)
		if i == a.activeQueryTab {
			tabs = append(tabs, StyleSelected.Render(label))
		} else {
			tabs = append(tabs, StyleBorder.Render(label))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, tabs...)
}
