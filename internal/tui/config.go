package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"oslo/internal/db"
	"oslo/internal/profile"
)

// configView tracks which view is active
type configView int

const (
	viewList configView = iota
	viewAdd
	viewEdit
	viewDelete
	viewDetail
)

// field indices
const (
	fieldName = iota
	fieldDriver
	fieldHost
	fieldPort
	fieldUser
	fieldPassword
	fieldDatabase
	fieldDSN
	fieldMaxOpenConns
	fieldMaxIdleConns
	fieldConnMaxLifetime
	fieldSSHHost
	fieldSSHPort
	fieldSSHUser
	fieldSSHPassword
	fieldSSHKeyPath
	fieldCount
)

var fieldLabels = [fieldCount]string{
	"Name    ",
	"Driver  ",
	"Host    ",
	"Port    ",
	"User    ",
	"Password",
	"Database",
	"DSN     ",
	"MaxOpen ",
	"MaxIdle ",
	"MaxLifeS",
	"SSH Host",
	"SSH Port",
	"SSH User",
	"SSH Pass",
	"SSH Key ",
}

var configDriverOrder = []string{
	"mysql",
	"mariadb",
	"oracle",
	"postgresql",
	"tibero",
	"cubrid",
	"sqlite",
}

var driverDefaultPorts = map[string]string{
	"mysql":      "3306",
	"mariadb":    "3306",
	"postgres":   "5432",
	"postgresql": "5432",
	"oracle":     "1521",
	"sqlite":     "",
	"tibero":     "8629",
	"cubrid":     "33000",
}

// ConfigApp is the top-level TUI model for `connect-dbms config`
type ConfigApp struct {
	width, height int
	store         *profile.Store
	drivers       []string
	view          configView

	// list view
	cursor  int
	message string // success/info message
	errMsg  string

	// form (add/edit)
	inputs       [fieldCount]textinput.Model
	formFocus    int
	editName     string // original name when editing
	driverCursor int    // for driver picker

	// delete confirm
	deleteName string
}

func NewConfigApp(store *profile.Store) *ConfigApp {
	c := &ConfigApp{
		store:   store,
		drivers: availableDrivers(),
		view:    viewList,
	}
	c.initInputs()
	return c
}

func (c *ConfigApp) initInputs() {
	for i := 0; i < fieldCount; i++ {
		ti := textinput.New()
		ti.CharLimit = 256
		ti.Width = 40
		if i == fieldPassword {
			ti.EchoMode = textinput.EchoPassword
		}
		if i == fieldSSHPassword {
			ti.EchoMode = textinput.EchoPassword
		}
		ti.Placeholder = strings.TrimSpace(fieldLabels[i])
		c.inputs[i] = ti
	}
	c.applyDriverInputState("")
}

func (c *ConfigApp) clearForm() {
	for i := 0; i < fieldCount; i++ {
		c.inputs[i].SetValue("")
		c.inputs[i].Blur()
	}
	c.formFocus = 0
	c.driverCursor = 0
}

func (c *ConfigApp) loadProfile(p *profile.Profile) {
	driver := normalizeDriverName(p.Driver)
	c.inputs[fieldName].SetValue(p.Name)
	c.inputs[fieldDriver].SetValue(driver)
	c.inputs[fieldHost].SetValue(p.Host)
	if p.Port > 0 {
		c.inputs[fieldPort].SetValue(strconv.Itoa(p.Port))
	} else {
		c.inputs[fieldPort].SetValue("")
	}
	c.inputs[fieldUser].SetValue(p.User)
	c.inputs[fieldPassword].SetValue(p.Password)
	c.inputs[fieldDatabase].SetValue(p.Database)
	c.inputs[fieldDSN].SetValue(p.DSN)
	if p.MaxOpenConns > 0 {
		c.inputs[fieldMaxOpenConns].SetValue(strconv.Itoa(p.MaxOpenConns))
	} else {
		c.inputs[fieldMaxOpenConns].SetValue("")
	}
	if p.MaxIdleConns > 0 {
		c.inputs[fieldMaxIdleConns].SetValue(strconv.Itoa(p.MaxIdleConns))
	} else {
		c.inputs[fieldMaxIdleConns].SetValue("")
	}
	if p.ConnMaxLifetimeSeconds > 0 {
		c.inputs[fieldConnMaxLifetime].SetValue(strconv.Itoa(p.ConnMaxLifetimeSeconds))
	} else {
		c.inputs[fieldConnMaxLifetime].SetValue("")
	}
	c.inputs[fieldSSHHost].SetValue(p.SSHHost)
	if p.SSHPort > 0 {
		c.inputs[fieldSSHPort].SetValue(strconv.Itoa(p.SSHPort))
	} else {
		c.inputs[fieldSSHPort].SetValue("")
	}
	c.inputs[fieldSSHUser].SetValue(p.SSHUser)
	c.inputs[fieldSSHPassword].SetValue(p.SSHPassword)
	c.inputs[fieldSSHKeyPath].SetValue(p.SSHKeyPath)

	// Set driver cursor
	for i, d := range c.drivers {
		if d == driver {
			c.driverCursor = i
			break
		}
	}
	c.applyDriverInputState(driver)
}

func (c *ConfigApp) buildProfile() profile.Profile {
	port, _ := strconv.Atoi(c.inputs[fieldPort].Value())
	maxOpenConns, _ := strconv.Atoi(c.inputs[fieldMaxOpenConns].Value())
	maxIdleConns, _ := strconv.Atoi(c.inputs[fieldMaxIdleConns].Value())
	connMaxLifetimeSeconds, _ := strconv.Atoi(c.inputs[fieldConnMaxLifetime].Value())
	sshPort, _ := strconv.Atoi(c.inputs[fieldSSHPort].Value())
	return profile.Profile{
		Name:                   strings.TrimSpace(c.inputs[fieldName].Value()),
		Driver:                 normalizeDriverName(strings.TrimSpace(c.inputs[fieldDriver].Value())),
		Host:                   strings.TrimSpace(c.inputs[fieldHost].Value()),
		Port:                   port,
		User:                   strings.TrimSpace(c.inputs[fieldUser].Value()),
		Password:               c.inputs[fieldPassword].Value(),
		Database:               strings.TrimSpace(c.inputs[fieldDatabase].Value()),
		DSN:                    strings.TrimSpace(c.inputs[fieldDSN].Value()),
		MaxOpenConns:           maxOpenConns,
		MaxIdleConns:           maxIdleConns,
		ConnMaxLifetimeSeconds: connMaxLifetimeSeconds,
		SSHHost:                strings.TrimSpace(c.inputs[fieldSSHHost].Value()),
		SSHPort:                sshPort,
		SSHUser:                strings.TrimSpace(c.inputs[fieldSSHUser].Value()),
		SSHPassword:            c.inputs[fieldSSHPassword].Value(),
		SSHKeyPath:             strings.TrimSpace(c.inputs[fieldSSHKeyPath].Value()),
	}
}

func (c *ConfigApp) Init() tea.Cmd {
	return nil
}

func (c *ConfigApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		return c, nil

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" || msg.String() == "ctrl+q" {
			return c, tea.Quit
		}

		switch c.view {
		case viewList:
			return c, c.updateList(msg)
		case viewAdd, viewEdit:
			return c, c.updateForm(msg)
		case viewDelete:
			return c, c.updateDelete(msg)
		case viewDetail:
			return c, c.updateDetail(msg)
		}
	}

	// pass to text inputs when in form
	if c.view == viewAdd || c.view == viewEdit {
		if c.formFocus != fieldDriver {
			var cmd tea.Cmd
			c.inputs[c.formFocus], cmd = c.inputs[c.formFocus].Update(msg)
			return c, cmd
		}
	}

	return c, nil
}

// --- List view ---

func (c *ConfigApp) updateList(msg tea.KeyMsg) tea.Cmd {
	profiles := c.store.List()
	c.errMsg = ""
	c.message = ""

	switch msg.String() {
	case "up", "k":
		if c.cursor > 0 {
			c.cursor--
		}
	case "down", "j":
		if c.cursor < len(profiles)-1 {
			c.cursor++
		}
	case "home":
		c.cursor = 0
	case "end":
		if len(profiles) > 0 {
			c.cursor = len(profiles) - 1
		}
	case "enter":
		if len(profiles) > 0 && c.cursor < len(profiles) {
			c.view = viewDetail
		}
	case "a":
		c.clearForm()
		// Set reasonable defaults
		driver := c.defaultDriver()
		c.inputs[fieldDriver].SetValue(driver)
		c.driverCursor = c.indexOfDriver(driver)
		c.inputs[fieldHost].SetValue("127.0.0.1")
		c.applyDriverInputState(driver)
		c.view = viewAdd
		c.formFocus = fieldName
		c.inputs[fieldName].Focus()
	case "e":
		if len(profiles) > 0 && c.cursor < len(profiles) {
			p := profiles[c.cursor]
			c.clearForm()
			c.loadProfile(&p)
			c.editName = p.Name
			c.view = viewEdit
			c.formFocus = fieldDriver
		}
	case "d", "delete":
		if len(profiles) > 0 && c.cursor < len(profiles) {
			c.deleteName = profiles[c.cursor].Name
			c.view = viewDelete
		}
	case "esc", "q":
		return tea.Quit
	}
	return nil
}

// --- Detail view ---

func (c *ConfigApp) updateDetail(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "q", "enter":
		c.view = viewList
	case "e":
		profiles := c.store.List()
		if c.cursor < len(profiles) {
			p := profiles[c.cursor]
			c.clearForm()
			c.loadProfile(&p)
			c.editName = p.Name
			c.view = viewEdit
			c.formFocus = fieldDriver
		}
	case "d", "delete":
		profiles := c.store.List()
		if c.cursor < len(profiles) {
			c.deleteName = profiles[c.cursor].Name
			c.view = viewDelete
		}
	}
	return nil
}

// --- Form view (add/edit) ---

func (c *ConfigApp) updateForm(msg tea.KeyMsg) tea.Cmd {
	c.errMsg = ""

	switch msg.String() {
	case "esc":
		c.view = viewList
		return nil
	case "up":
		c.moveFocus(-1)
		return nil
	case "down":
		c.moveFocus(1)
		return nil
	case "tab":
		c.moveFocus(1)
		return nil
	case "shift+tab":
		c.moveFocus(-1)
		return nil
	case "enter":
		// On driver field, enter selects the driver
		if c.formFocus == fieldDriver && len(c.drivers) > 0 {
			c.applyDriverInputState(c.drivers[c.driverCursor])
			c.moveFocus(1)
			return nil
		}
		// If on last field or explicitly saving
		if c.formFocus == fieldCount-1 {
			return c.saveForm()
		}
		// Move to next field
		c.moveFocus(1)
		return nil
	case "ctrl+s":
		return c.saveForm()
	case "left":
		if c.formFocus == fieldDriver {
			if c.driverCursor > 0 {
				c.driverCursor--
				c.applyDriverInputState(c.drivers[c.driverCursor])
			}
			return nil
		}
	case "right":
		if c.formFocus == fieldDriver {
			if c.driverCursor < len(c.drivers)-1 {
				c.driverCursor++
				c.applyDriverInputState(c.drivers[c.driverCursor])
			}
			return nil
		}
	}

	// Update current input (not for driver field which uses arrow keys)
	if c.formFocus != fieldDriver {
		var cmd tea.Cmd
		c.inputs[c.formFocus], cmd = c.inputs[c.formFocus].Update(msg)
		return cmd
	}
	return nil
}

func (c *ConfigApp) moveFocus(dir int) {
	c.inputs[c.formFocus].Blur()
	c.formFocus += dir
	if c.formFocus < 0 {
		c.formFocus = 0
	}
	if c.formFocus >= fieldCount {
		c.formFocus = fieldCount - 1
	}
	if c.formFocus != fieldDriver {
		c.inputs[c.formFocus].Focus()
	}
}

func (c *ConfigApp) updatePortDefault() {
	if len(c.drivers) == 0 || c.driverCursor >= len(c.drivers) {
		return
	}
	driver := c.drivers[c.driverCursor]
	if port, ok := driverDefaultPorts[driver]; ok {
		// Only update if port is empty or matches another default
		cur := c.inputs[fieldPort].Value()
		for _, v := range driverDefaultPorts {
			if cur == v || cur == "" {
				c.inputs[fieldPort].SetValue(port)
				break
			}
		}
	}
}

func (c *ConfigApp) saveForm() tea.Cmd {
	p := c.buildProfile()

	if field, err := validateProfileForSave(p); err != nil {
		c.errMsg = err.Error()
		c.formFocus = field
		if field != fieldDriver {
			c.inputs[field].Focus()
		}
		return nil
	}

	var err error
	if c.view == viewAdd {
		err = c.store.Add(p)
		if err == nil {
			c.message = fmt.Sprintf("Session '%s' saved", p.Name)
			c.cursor = len(c.store.List()) - 1
		}
	} else {
		err = c.store.Update(c.editName, p)
		if err == nil {
			c.message = fmt.Sprintf("Session '%s' updated", p.Name)
		}
	}

	if err != nil {
		c.errMsg = err.Error()
		return nil
	}

	c.view = viewList
	return nil
}

// --- Delete confirm ---

func (c *ConfigApp) updateDelete(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "enter":
		if err := c.store.Remove(c.deleteName); err != nil {
			c.errMsg = err.Error()
		} else {
			c.message = fmt.Sprintf("Session '%s' removed", c.deleteName)
		}
		if c.cursor > 0 && c.cursor >= len(c.store.List()) {
			c.cursor = len(c.store.List()) - 1
		}
		c.view = viewList
	case "n", "esc", "q":
		c.view = viewList
	}
	return nil
}

// --- Views ---

func (c *ConfigApp) View() string {
	if c.width == 0 {
		return "Loading..."
	}

	var content string
	switch c.view {
	case viewList:
		content = c.viewList()
	case viewAdd:
		content = c.viewForm("Add Session")
	case viewEdit:
		content = c.viewForm("Edit Session")
	case viewDelete:
		content = c.viewDelete()
	case viewDetail:
		content = c.viewDetail()
	}

	// Status bar
	bar := c.statusBar()
	return lipgloss.JoinVertical(lipgloss.Left, content, bar)
}

func (c *ConfigApp) viewList() string {
	var sb strings.Builder

	sb.WriteString(StyleTitle.Render("  DB Sessions") + "\n\n")

	profiles := c.store.List()

	if len(profiles) == 0 {
		sb.WriteString(StyleHelp.Render("  No sessions saved yet.") + "\n\n")
		sb.WriteString(StyleHelp.Render("  Press ") + StyleSubtitle.Render("a") + StyleHelp.Render(" to add one.") + "\n")
	} else {
		// Header
		header := fmt.Sprintf("  %-20s %-12s %-20s %-6s %s", "NAME", "DRIVER", "HOST", "PORT", "DATABASE")
		sb.WriteString(StyleTableHeader.Render(header) + "\n")
		sb.WriteString("  " + strings.Repeat("-", 72) + "\n")

		for i, p := range profiles {
			host := p.Host
			if p.DSN != "" && host == "" {
				host = "(dsn)"
			}
			port := ""
			if p.Port > 0 {
				port = strconv.Itoa(p.Port)
			}
			line := fmt.Sprintf("  %-20s %-12s %-20s %-6s %s",
				truncStr(p.Name, 20),
				truncStr(p.Driver, 12),
				truncStr(host, 20),
				port,
				truncStr(p.Database, 20),
			)
			if i == c.cursor {
				sb.WriteString(StyleSelected.Render(line) + "\n")
			} else {
				sb.WriteString(line + "\n")
			}
		}
	}

	// Messages
	if c.message != "" {
		sb.WriteString("\n" + StyleSuccess.Render("  "+c.message))
	}
	if c.errMsg != "" {
		sb.WriteString("\n" + StyleError.Render("  "+c.errMsg))
	}

	sb.WriteString("\n\n")
	sb.WriteString(StyleHelp.Render("  Up/Down: move  Enter: details  a: add  e: edit  d: delete  q: quit"))

	return sb.String()
}

func (c *ConfigApp) viewDetail() string {
	profiles := c.store.List()
	if c.cursor >= len(profiles) {
		return "No session selected"
	}
	p := profiles[c.cursor]

	var sb strings.Builder
	sb.WriteString(StyleTitle.Render(fmt.Sprintf("  Session: %s", p.Name)) + "\n\n")

	fields := []struct{ label, value string }{
		{"Driver", p.Driver},
		{"Host", p.Host},
		{"Port", fmtPort(p.Port)},
		{"User", p.User},
		{"Password", maskPassword(p.Password)},
		{"Database", p.Database},
		{"DSN", p.DSN},
		{"MaxOpen", fmtPort(p.MaxOpenConns)},
		{"MaxIdle", fmtPort(p.MaxIdleConns)},
		{"MaxLifeS", fmtPort(p.ConnMaxLifetimeSeconds)},
		{"SSH Host", p.SSHHost},
		{"SSH Port", fmtPort(p.SSHPort)},
		{"SSH User", p.SSHUser},
		{"SSH Pass", maskPassword(p.SSHPassword)},
		{"SSH Key", p.SSHKeyPath},
	}

	for _, f := range fields {
		if f.value == "" {
			f.value = StyleHelp.Render("(empty)")
		}
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			StyleSubtitle.Render(padRight(f.label, 10)),
			f.value,
		))
	}

	sb.WriteString("\n")
	sb.WriteString(StyleHelp.Render("  e: edit  d: delete  Esc: back"))

	boxW := 60
	if boxW > c.width-4 {
		boxW = c.width - 4
	}

	return lipgloss.Place(c.width, c.height-2, lipgloss.Center, lipgloss.Center,
		StyleBorder.Width(boxW).Padding(1, 2).Render(sb.String()))
}

func (c *ConfigApp) viewForm(title string) string {
	var sb strings.Builder

	sb.WriteString(StyleTitle.Render("  "+title) + "\n\n")
	currentDriver := c.currentDriver()

	for i := 0; i < fieldCount; i++ {
		isActive := i == c.formFocus
		prefix := "  "
		if isActive {
			prefix = "> "
		}

		label := padRight(formFieldLabel(i, currentDriver), 14)

		if i == fieldDriver {
			// Driver picker with arrow keys
			var driverView string
			for di, d := range c.drivers {
				if di == c.driverCursor {
					if isActive {
						driverView += StyleSelected.Render(" " + d + " ")
					} else {
						driverView += StyleSubtitle.Render("[" + d + "]")
					}
				} else {
					driverView += StyleHelp.Render(" " + d + " ")
				}
			}
			if isActive {
				sb.WriteString(StyleSubtitle.Render(prefix+label+": ") + driverView + "\n")
				sb.WriteString(StyleHelp.Render("           Left/Right: pick driver  Enter: confirm") + "\n")
			} else {
				sb.WriteString(prefix + label + ": " + driverView + "\n")
			}
		} else {
			inputView := c.inputs[i].View()
			if isActive {
				sb.WriteString(StyleSubtitle.Render(prefix+label+": ") + inputView + "\n")
			} else {
				sb.WriteString(prefix + label + ": " + inputView + "\n")
			}
		}
	}

	if hint := formDriverHelp(currentDriver); hint != "" {
		sb.WriteString("\n" + StyleHelp.Render("  "+hint))
	}

	if c.errMsg != "" {
		sb.WriteString("\n" + StyleError.Render("  "+c.errMsg))
	}

	sb.WriteString("\n\n")
	sb.WriteString(StyleHelp.Render("  Up/Down: move  Enter: next/save  Ctrl+S: save  Esc: cancel"))

	boxW := 70
	if boxW > c.width-4 {
		boxW = c.width - 4
	}

	return lipgloss.Place(c.width, c.height-2, lipgloss.Center, lipgloss.Center,
		StyleActiveBorder.Width(boxW).Padding(1, 2).Render(sb.String()))
}

func (c *ConfigApp) viewDelete() string {
	var sb strings.Builder

	sb.WriteString(StyleError.Render("  Delete Session") + "\n\n")
	sb.WriteString(fmt.Sprintf("  Remove session '%s'?\n\n", c.deleteName))
	sb.WriteString("  " + StyleError.Render("[y]") + " Yes, delete\n")
	sb.WriteString("  " + StyleHelp.Render("[n]") + " No, cancel\n")

	boxW := 45
	if boxW > c.width-4 {
		boxW = c.width - 4
	}

	return lipgloss.Place(c.width, c.height-2, lipgloss.Center, lipgloss.Center,
		StyleBorder.Width(boxW).Padding(1, 2).Render(sb.String()))
}

func (c *ConfigApp) statusBar() string {
	left := " connect-dbms config"
	right := fmt.Sprintf("%d sessions | %s", len(c.store.List()), profile.DefaultPath())

	pad := c.width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}

	bar := left + fmt.Sprintf("%*s", pad, "") + right
	return StyleStatusBar.Width(c.width).Render(bar)
}

// helpers

func truncStr(s string, max int) string {
	if len(s) > max {
		return s[:max-2] + ".."
	}
	return s
}

func fmtPort(port int) string {
	if port > 0 {
		return strconv.Itoa(port)
	}
	return ""
}

func maskPassword(s string) string {
	if s == "" {
		return ""
	}
	return strings.Repeat("*", len(s))
}

func availableDrivers() []string {
	registered := make(map[string]struct{})
	for _, name := range db.ListDrivers() {
		registered[name] = struct{}{}
	}

	var drivers []string
	for _, name := range configDriverOrder {
		switch name {
		case "postgresql":
			if _, ok := registered["postgresql"]; ok {
				drivers = append(drivers, "postgresql")
				continue
			}
			if _, ok := registered["postgres"]; ok {
				drivers = append(drivers, "postgresql")
			}
		default:
			if _, ok := registered[name]; ok {
				drivers = append(drivers, name)
			}
		}
	}

	return drivers
}

func defaultPortForDriver(driver string) string {
	return driverDefaultPorts[driver]
}

func (c *ConfigApp) defaultDriver() string {
	if len(c.drivers) == 0 {
		return ""
	}
	return c.drivers[0]
}

func (c *ConfigApp) indexOfDriver(driver string) int {
	for i, d := range c.drivers {
		if d == normalizeDriverName(driver) {
			return i
		}
	}
	return 0
}

func (c *ConfigApp) currentDriver() string {
	driver := normalizeDriverName(strings.TrimSpace(c.inputs[fieldDriver].Value()))
	if driver != "" {
		return driver
	}
	if len(c.drivers) == 0 || c.driverCursor >= len(c.drivers) {
		return ""
	}
	return c.drivers[c.driverCursor]
}

func (c *ConfigApp) applyDriverInputState(driver string) {
	driver = normalizeDriverName(driver)
	c.inputs[fieldDriver].SetValue(driver)
	c.updatePortDefault()

	if isSQLiteDriver(driver) {
		c.inputs[fieldDatabase].Placeholder = "./data.db"
		c.inputs[fieldDSN].Placeholder = "Optional raw DSN"
		if c.inputs[fieldHost].Value() == "127.0.0.1" {
			c.inputs[fieldHost].SetValue("")
		}
		if isDefaultPort(c.inputs[fieldPort].Value()) {
			c.inputs[fieldPort].SetValue("")
		}
		return
	}

	c.inputs[fieldDatabase].Placeholder = "Database"
	c.inputs[fieldDSN].Placeholder = "DSN"
	if c.inputs[fieldHost].Value() == "" {
		c.inputs[fieldHost].SetValue("127.0.0.1")
	}
}

func validateProfileForSave(p profile.Profile) (int, error) {
	if p.Name == "" {
		return fieldName, fmt.Errorf("Name is required")
	}
	if p.Driver == "" {
		return fieldDriver, fmt.Errorf("Driver is required")
	}
	if _, err := db.Get(p.Driver); err != nil {
		return fieldDriver, fmt.Errorf("Unknown driver: %s", p.Driver)
	}
	if isSQLiteDriver(p.Driver) && p.Database == "" && p.DSN == "" {
		return fieldDatabase, fmt.Errorf("SQLite requires a file path in File Path or a DSN")
	}
	return -1, nil
}

func normalizeDriverName(driver string) string {
	if driver == "postgres" {
		return "postgresql"
	}
	return driver
}

func isSQLiteDriver(driver string) bool {
	return normalizeDriverName(driver) == "sqlite"
}

func formFieldLabel(field int, driver string) string {
	if isSQLiteDriver(driver) {
		switch field {
		case fieldHost:
			return "Host (unused)"
		case fieldPort:
			return "Port (unused)"
		case fieldUser:
			return "User (unused)"
		case fieldPassword:
			return "Password (unused)"
		case fieldDatabase:
			return "File Path"
		case fieldDSN:
			return "DSN (optional)"
		}
	}
	return strings.TrimSpace(fieldLabels[field])
}

func formDriverHelp(driver string) string {
	if isSQLiteDriver(driver) {
		return "SQLite: put the database file path in File Path, for example ./data.db or /tmp/app.db. Host, Port, User, and Password are ignored."
	}
	return ""
}

func isDefaultPort(port string) bool {
	for _, v := range driverDefaultPorts {
		if port == v {
			return true
		}
	}
	return false
}
