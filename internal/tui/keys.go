package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	RunQuery     key.Binding
	Quit         key.Binding
	Help         key.Binding
	Tab          key.Binding
	ShiftTab     key.Binding
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Escape       key.Binding
	Tables       key.Binding
	History      key.Binding
	NewConn      key.Binding
	NewTab       key.Binding
	NextTab      key.Binding
	PrevTab      key.Binding
	Autocomplete key.Binding
	Export       key.Binding
	Delete       key.Binding
	NextPage     key.Binding
	PrevPage     key.Binding
}

var Keys = KeyMap{
	RunQuery: key.NewBinding(
		key.WithKeys("f5", "ctrl+e"),
		key.WithHelp("F5/Ctrl+E", "run query"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+q"),
		key.WithHelp("Ctrl+Q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("f1", "ctrl+?"),
		key.WithHelp("F1", "help"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "next panel"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("Shift+Tab", "prev panel"),
	),
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("Up", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("Down", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "select"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "back"),
	),
	Tables: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("Ctrl+T", "tables"),
	),
	History: key.NewBinding(
		key.WithKeys("ctrl+h"),
		key.WithHelp("Ctrl+H", "history"),
	),
	NewConn: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("Ctrl+N", "new conn"),
	),
	NewTab: key.NewBinding(
		key.WithKeys("f6"),
		key.WithHelp("F6", "new tab"),
	),
	NextTab: key.NewBinding(
		key.WithKeys("f7"),
		key.WithHelp("F7", "next tab"),
	),
	PrevTab: key.NewBinding(
		key.WithKeys("f8"),
		key.WithHelp("F8", "prev tab"),
	),
	Autocomplete: key.NewBinding(
		key.WithKeys("ctrl+space", "f9"),
		key.WithHelp("Ctrl+Space/F9", "autocomplete"),
	),
	Export: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("Ctrl+S", "export"),
	),
	Delete: key.NewBinding(
		key.WithKeys("ctrl+d", "delete"),
		key.WithHelp("Ctrl+D", "delete"),
	),
	NextPage: key.NewBinding(
		key.WithKeys("pgdown", "ctrl+f"),
		key.WithHelp("PgDn", "next page"),
	),
	PrevPage: key.NewBinding(
		key.WithKeys("pgup", "ctrl+b"),
		key.WithHelp("PgUp", "prev page"),
	),
}
