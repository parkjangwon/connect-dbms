package tui

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"oslo/internal/db"
	"oslo/internal/dberr"
	"oslo/internal/profile"
	"oslo/internal/sshtunnel"
)

type connectFocus int

const (
	focusProfiles connectFocus = iota
	focusForm
)

type ConnectScreen struct {
	width, height int
	store         *profile.Store
	focus         connectFocus
	cursor        int
	errMsg        string

	// Quick connect form
	formFocus int
	inputs    []textinput.Model
}

func NewConnectScreen(store *profile.Store) *ConnectScreen {
	s := &ConnectScreen{
		store: store,
	}

	// Form inputs: driver, host, port, user, password, database
	labels := []string{"Driver", "Host", "Port", "User", "Password", "Database"}
	defaults := []string{"postgres", "127.0.0.1", "5432", "", "", ""}

	s.inputs = make([]textinput.Model, len(labels))
	for i, label := range labels {
		ti := textinput.New()
		ti.Placeholder = label
		ti.SetValue(defaults[i])
		if label == "Password" {
			ti.EchoMode = textinput.EchoPassword
		}
		ti.CharLimit = 256
		ti.Width = 30
		s.inputs[i] = ti
	}

	if len(store.List()) > 0 {
		s.focus = focusProfiles
	} else {
		s.focus = focusForm
		s.inputs[0].Focus()
	}

	return s
}

func (s *ConnectScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (s *ConnectScreen) SetError(msg string) {
	s.errMsg = msg
}

func (s *ConnectScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s.errMsg = ""

		if key.Matches(msg, Keys.Tab) {
			if s.focus == focusProfiles {
				s.focus = focusForm
				s.inputs[0].Focus()
			} else {
				s.focus = focusProfiles
				for i := range s.inputs {
					s.inputs[i].Blur()
				}
			}
			return nil
		}

		if s.focus == focusProfiles {
			return s.updateProfiles(msg)
		}
		return s.updateForm(msg)
	}

	// Update text inputs
	if s.focus == focusForm {
		var cmds []tea.Cmd
		for i := range s.inputs {
			var cmd tea.Cmd
			s.inputs[i], cmd = s.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		return tea.Batch(cmds...)
	}
	return nil
}

func (s *ConnectScreen) updateProfiles(msg tea.KeyMsg) tea.Cmd {
	profiles := s.store.List()
	switch {
	case key.Matches(msg, Keys.Up):
		if s.cursor > 0 {
			s.cursor--
		}
	case key.Matches(msg, Keys.Down):
		if s.cursor < len(profiles)-1 {
			s.cursor++
		}
	case key.Matches(msg, Keys.Enter):
		if s.cursor < len(profiles) {
			return s.connectProfile(&profiles[s.cursor])
		}
	case key.Matches(msg, Keys.Delete):
		if s.cursor < len(profiles) {
			name := profiles[s.cursor].Name
			_ = s.store.Remove(name)
			if s.cursor > 0 {
				s.cursor--
			}
		}
	}
	return nil
}

func (s *ConnectScreen) updateForm(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, Keys.Up):
		if s.formFocus > 0 {
			s.inputs[s.formFocus].Blur()
			s.formFocus--
			s.inputs[s.formFocus].Focus()
		}
	case key.Matches(msg, Keys.Down):
		if s.formFocus < len(s.inputs)-1 {
			s.inputs[s.formFocus].Blur()
			s.formFocus++
			s.inputs[s.formFocus].Focus()
		}
	case key.Matches(msg, Keys.Enter):
		if s.formFocus < len(s.inputs)-1 {
			s.inputs[s.formFocus].Blur()
			s.formFocus++
			s.inputs[s.formFocus].Focus()
			return nil
		}
		return s.connectForm()
	}
	return nil
}

func (s *ConnectScreen) connectProfile(p *profile.Profile) tea.Cmd {
	return func() tea.Msg {
		drv, err := db.Get(p.Driver)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		host := p.Host
		if host == "" && p.DSN != "" {
			host = "(dsn)"
		}
		cfg := p.ToConnConfig()
		tunnelCfg, tunnel, err := sshtunnel.PrepareConnConfig(cfg)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		conn, err := drv.Open(tunnelCfg)
		if err != nil {
			if tunnel != nil {
				tunnel.Close()
			}
			dbe := dberr.Wrap(p.Driver, "open", host, err)
			fmt.Fprintln(os.Stderr, dbe.Format())
			return ErrorMsg{Err: fmt.Errorf("[%s] %s", dbe.Code, dbe.Message)}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := conn.PingContext(ctx); err != nil {
			conn.Close()
			if tunnel != nil {
				tunnel.Close()
			}
			dbe := dberr.Wrap(p.Driver, "ping", host, err)
			fmt.Fprintln(os.Stderr, dbe.Format())
			return ErrorMsg{Err: fmt.Errorf("[%s] %s", dbe.Code, dbe.Message)}
		}
		return ConnectedMsg{Profile: p, Driver: drv, Conn: conn, Tunnel: tunnel}
	}
}

func (s *ConnectScreen) connectForm() tea.Cmd {
	driverName := s.inputs[0].Value()
	host := s.inputs[1].Value()
	port := s.inputs[2].Value()
	user := s.inputs[3].Value()
	password := s.inputs[4].Value()
	database := s.inputs[5].Value()

	p := &profile.Profile{
		Name:     fmt.Sprintf("%s@%s", user, host),
		Driver:   driverName,
		Host:     host,
		User:     user,
		Password: password,
		Database: database,
	}
	if port != "" {
		fmt.Sscanf(port, "%d", &p.Port)
	}

	return s.connectProfile(p)
}

func (s *ConnectScreen) View() string {
	logo := StyleLogo.Render(`
                          _            _ _
  ___ ___  _ __  _ __   ___  ___| |_      __| | |__  _ __ ___  ___
 / __/ _ \| '_ \| '_ \ / _ \/ __| __|____/ _' | '_ \| '_ ' _ \/ __|
| (_| (_) | | | | | | |  __/ (__| ||_____| (_| | |_) | | | | | \__ \
 \___\___/|_| |_|_| |_|\___|\___|\__|     \__,_|_.__/|_| |_| |_|___/  v1.0.0`)

	profiles := s.store.List()

	// Left: saved profiles
	var leftContent string
	leftContent = StyleSubtitle.Render("Saved Sessions") + "\n\n"
	if len(profiles) == 0 {
		leftContent += StyleHelp.Render("  (no sessions)")
	} else {
		for i, p := range profiles {
			line := fmt.Sprintf("  %s (%s)", p.Name, p.Driver)
			if s.focus == focusProfiles && i == s.cursor {
				leftContent += StyleSelected.Render(line) + "\n"
			} else {
				leftContent += line + "\n"
			}
		}
	}
	leftContent += "\n" + StyleHelp.Render("  Enter: connect  Del: remove")

	// Right: quick connect form
	var rightContent string
	rightContent = StyleSubtitle.Render("Quick Connect") + "\n\n"
	labels := []string{"Driver  ", "Host    ", "Port    ", "User    ", "Password", "Database"}
	for i, label := range labels {
		prefix := "  "
		if s.focus == focusForm && i == s.formFocus {
			prefix = "> "
		}
		rightContent += fmt.Sprintf("%s%s: %s\n", prefix, label, s.inputs[i].View())
	}
	rightContent += "\n" + StyleHelp.Render("  Enter: connect  Tab: switch panel")

	leftW := 36
	rightW := s.width - leftW - 6
	if rightW < 30 {
		rightW = 30
	}

	left := StyleBorder.Width(leftW).Height(s.height - 10).Render(leftContent)
	right := StyleBorder.Width(rightW).Height(s.height - 10).Render(rightContent)

	main := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	var errView string
	if s.errMsg != "" {
		errView = "\n" + StyleError.Render("  Error: "+s.errMsg)
	}

	return lipgloss.JoinVertical(lipgloss.Left, logo, "", main, errView)
}
