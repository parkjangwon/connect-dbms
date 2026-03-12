package tui

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"oslo/internal/db"
	"oslo/internal/dberr"
	"oslo/internal/export"
	"oslo/internal/history"
	"oslo/internal/profile"
)

type queryFocus int

const (
	focusEditor queryFocus = iota
	focusResults
)

type queryOverlay int

const (
	overlayNone queryOverlay = iota
	overlayHistory
	overlayExport
	overlayAutocomplete
)

type QueryScreen struct {
	width, height int
	conn          *sql.DB
	driver        db.Driver
	prof          *profile.Profile

	focus   queryFocus
	editor  textarea.Model
	results *db.QueryResult
	errMsg  string
	status  string

	// Results scroll
	resultOffset int
	resultCursor int

	historyStore     *history.Store
	overlay          queryOverlay
	historyInput     textinput.Model
	historyItems     []history.Entry
	historyCursor    int
	historyErr       string
	exportInput      textinput.Model
	exportErr        string
	tableCompletions []string
	completionItems  []string
	completionCursor int
	completionErr    string
}

type queryDoneMsg struct {
	sql      string
	result   *db.QueryResult
	err      error
	duration time.Duration
}

type historyLoadedMsg struct {
	items []history.Entry
	err   error
}

type historySavedMsg struct {
	err error
}

type completionsLoadedMsg struct {
	items []string
	err   error
}

type exportDoneMsg struct {
	path string
	err  error
}

func NewQueryScreen(conn *sql.DB, drv db.Driver, prof *profile.Profile) *QueryScreen {
	ta := textarea.New()
	ta.Placeholder = "Type your SQL here... (F5 to run)"
	ta.Focus()
	ta.CharLimit = 100000
	ta.ShowLineNumbers = true
	ta.SetHeight(8)

	historyInput := textinput.New()
	historyInput.Placeholder = "Search query history"
	historyInput.Width = 42

	exportInput := textinput.New()
	exportInput.Placeholder = "./query-results.csv"
	exportInput.Width = 42

	historyStore, histErr := history.Open("")
	status := "Ready"
	if histErr != nil {
		status = "Ready (history unavailable)"
	}

	return &QueryScreen{
		conn:         conn,
		driver:       drv,
		prof:         prof,
		focus:        focusEditor,
		editor:       ta,
		status:       status,
		historyStore: historyStore,
		historyErr:   errorString(histErr),
		historyInput: historyInput,
		exportInput:  exportInput,
	}
}

func (s *QueryScreen) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, s.loadCompletions())
}

func (s *QueryScreen) Close() error {
	if s.historyStore != nil {
		return s.historyStore.Close()
	}
	return nil
}

func (s *QueryScreen) SetSQL(sql string) {
	s.editor.SetValue(sql)
}

func (s *QueryScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s.overlay != overlayNone {
			return s.updateOverlay(msg)
		}
		if key.Matches(msg, Keys.RunQuery) {
			return s.runQuery()
		}
		if key.Matches(msg, Keys.History) {
			return s.openHistory()
		}
		if key.Matches(msg, Keys.Autocomplete) {
			return s.openAutocomplete()
		}
		if key.Matches(msg, Keys.Export) {
			return s.openExport()
		}
		if key.Matches(msg, Keys.Tab) {
			if s.focus == focusEditor {
				s.focus = focusResults
				s.editor.Blur()
			} else {
				s.focus = focusEditor
				s.editor.Focus()
			}
			return nil
		}

		if s.focus == focusResults {
			return s.updateResults(msg)
		}

	case queryDoneMsg:
		if msg.err != nil {
			host := s.prof.Host
			if host == "" && s.prof.DSN != "" {
				host = "(dsn)"
			}
			dbe := dberr.Wrap(s.prof.Driver, "query", host, msg.err)
			fmt.Fprintln(os.Stderr, dbe.Format())
			s.errMsg = fmt.Sprintf("[%s] %s", dbe.Code, dbe.Message)
			s.status = "Error"
			s.results = nil
		} else {
			s.errMsg = ""
			s.results = msg.result
			s.resultOffset = 0
			s.resultCursor = 0
			if msg.result.Columns != nil {
				s.status = fmt.Sprintf("%d rows (%s)", msg.result.RowCount, msg.result.Duration.Round(time.Millisecond))
			} else {
				s.status = fmt.Sprintf("%d affected (%s)", msg.result.RowCount, msg.result.Duration.Round(time.Millisecond))
			}
		}
		return s.saveHistory(msg)

	case historyLoadedMsg:
		if msg.err != nil {
			s.historyErr = msg.err.Error()
			s.historyItems = nil
			s.historyCursor = 0
		} else {
			s.historyErr = ""
			s.historyItems = msg.items
			if s.historyCursor >= len(s.historyItems) {
				s.historyCursor = maxInt(len(s.historyItems)-1, 0)
			}
		}
		return nil

	case historySavedMsg:
		return nil

	case completionsLoadedMsg:
		if msg.err != nil {
			s.completionErr = msg.err.Error()
			s.tableCompletions = nil
		} else {
			s.completionErr = ""
			s.tableCompletions = msg.items
		}
		return nil

	case exportDoneMsg:
		if msg.err != nil {
			s.exportErr = msg.err.Error()
		} else {
			s.exportErr = ""
			s.overlay = overlayNone
			s.exportInput.Blur()
			s.status = fmt.Sprintf("Exported to %s", msg.path)
		}
		return nil
	}

	if s.focus == focusEditor {
		var cmd tea.Cmd
		s.editor, cmd = s.editor.Update(msg)
		return cmd
	}
	return nil
}

func (s *QueryScreen) updateOverlay(msg tea.KeyMsg) tea.Cmd {
	switch s.overlay {
	case overlayHistory:
		return s.updateHistoryOverlay(msg)
	case overlayExport:
		return s.updateExportOverlay(msg)
	case overlayAutocomplete:
		return s.updateAutocompleteOverlay(msg)
	default:
		return nil
	}
}

func (s *QueryScreen) updateResults(msg tea.KeyMsg) tea.Cmd {
	if s.results == nil || len(s.results.Rows) == 0 {
		return nil
	}
	maxRow := len(s.results.Rows) - 1

	switch {
	case key.Matches(msg, Keys.Up):
		if s.resultCursor > 0 {
			s.resultCursor--
		}
	case key.Matches(msg, Keys.Down):
		if s.resultCursor < maxRow {
			s.resultCursor++
		}
	case key.Matches(msg, Keys.NextPage):
		s.resultCursor += 10
		if s.resultCursor > maxRow {
			s.resultCursor = maxRow
		}
	case key.Matches(msg, Keys.PrevPage):
		s.resultCursor -= 10
		if s.resultCursor < 0 {
			s.resultCursor = 0
		}
	}
	return nil
}

func (s *QueryScreen) runQuery() tea.Cmd {
	sqlText := strings.TrimSpace(s.editor.Value())
	if sqlText == "" {
		return nil
	}
	s.status = "Running..."
	s.errMsg = ""
	start := time.Now()

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var result *db.QueryResult
		var err error
		if db.IsSelectQuery(sqlText) {
			result, err = db.RunQuery(ctx, s.conn, sqlText)
		} else {
			result, err = db.RunExec(ctx, s.conn, sqlText)
		}
		return queryDoneMsg{
			sql:      sqlText,
			result:   result,
			err:      err,
			duration: time.Since(start),
		}
	}
}

func (s *QueryScreen) View() string {
	if s.overlay == overlayHistory {
		return s.viewHistory()
	}
	if s.overlay == overlayExport {
		return s.viewExport()
	}
	if s.overlay == overlayAutocomplete {
		return s.viewAutocomplete()
	}

	editorH := 10
	previewH := 3
	resultsH := s.height - editorH - previewH - 5
	if resultsH < 6 {
		resultsH = 6
	}

	s.editor.SetWidth(s.width - 4)
	s.editor.SetHeight(editorH - 2)
	editorBorder := StyleBorder
	if s.focus == focusEditor {
		editorBorder = StyleActiveBorder
	}
	editorView := editorBorder.Width(s.width - 2).Render(
		StyleSubtitle.Render(" SQL ") + "\n" + highlightSQLContent(s.editor.View()),
	)

	previewView := StyleBorder.Width(s.width - 2).Height(previewH).Render(
		StyleSubtitle.Render(" Preview ") + "\n" + renderSQLPreview(s.editor.Value()),
	)

	resultsBorder := StyleBorder
	if s.focus == focusResults {
		resultsBorder = StyleActiveBorder
	}

	var resultsContent string
	if s.errMsg != "" {
		resultsContent = StyleError.Render(s.errMsg)
	} else if s.results != nil && s.results.Columns != nil {
		resultsContent = s.renderTable(s.width-6, resultsH-3)
	} else if s.results != nil {
		resultsContent = StyleSuccess.Render(s.status)
	} else {
		resultsContent = StyleHelp.Render("Press F5 to run query")
	}

	resultsView := resultsBorder.Width(s.width - 2).Height(resultsH).Render(
		StyleSubtitle.Render(" Results ") + " " + StyleHelp.Render(s.status) + "\n" + resultsContent,
	)

	return lipgloss.JoinVertical(lipgloss.Left, editorView, previewView, resultsView)
}

func (s *QueryScreen) renderTable(maxW, maxH int) string {
	if s.results == nil || len(s.results.Columns) == 0 {
		return ""
	}

	cols := s.results.Columns
	rows := s.results.Rows

	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c)
	}
	for _, row := range rows {
		for i, v := range row {
			w := len(fmt.Sprintf("%v", v))
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	maxColW := (maxW - len(cols)*3) / len(cols)
	if maxColW < 8 {
		maxColW = 8
	}
	for i := range widths {
		if widths[i] > maxColW {
			widths[i] = maxColW
		}
	}

	var sb strings.Builder
	for i, c := range cols {
		if i > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString(StyleTableHeader.Render(truncPad(c, widths[i])))
	}
	sb.WriteString("\n")

	for i, w := range widths {
		if i > 0 {
			sb.WriteString("-+-")
		}
		sb.WriteString(strings.Repeat("-", w))
	}
	sb.WriteString("\n")

	visibleRows := maxH - 2
	if visibleRows < 1 {
		visibleRows = 1
	}

	if s.resultCursor < s.resultOffset {
		s.resultOffset = s.resultCursor
	}
	if s.resultCursor >= s.resultOffset+visibleRows {
		s.resultOffset = s.resultCursor - visibleRows + 1
	}

	end := s.resultOffset + visibleRows
	if end > len(rows) {
		end = len(rows)
	}

	for ri := s.resultOffset; ri < end; ri++ {
		row := rows[ri]
		for i, v := range row {
			if i > 0 {
				sb.WriteString(" | ")
			}
			val := fmt.Sprintf("%v", v)
			cell := truncPad(val, widths[i])
			if ri == s.resultCursor && s.focus == focusResults {
				sb.WriteString(StyleSelected.Render(cell))
			} else {
				sb.WriteString(cell)
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func truncPad(s string, w int) string {
	if len(s) > w {
		if w > 2 {
			return s[:w-2] + ".."
		}
		return s[:w]
	}
	return s + strings.Repeat(" ", w-len(s))
}

func (s *QueryScreen) openHistory() tea.Cmd {
	s.overlay = overlayHistory
	s.historyCursor = 0
	s.historyInput.Focus()
	s.exportInput.Blur()
	return s.loadHistory()
}

func (s *QueryScreen) openExport() tea.Cmd {
	if s.results == nil || len(s.results.Columns) == 0 {
		s.status = "No query results to export"
		return nil
	}

	s.overlay = overlayExport
	s.exportErr = ""
	if strings.TrimSpace(s.exportInput.Value()) == "" {
		s.exportInput.SetValue(defaultExportPath(s.prof.Name))
	}
	s.exportInput.Focus()
	s.historyInput.Blur()
	return nil
}

func (s *QueryScreen) openAutocomplete() tea.Cmd {
	lineInfo := s.editor.LineInfo()
	row := s.editor.Line()
	col := lineInfo.StartColumn + lineInfo.ColumnOffset
	prefix := completionPrefix(s.editor.Value(), row, col)
	matches := filterCompletions(prefix, s.tableCompletions)
	if len(matches) == 0 {
		if s.completionErr != "" {
			s.status = "Autocomplete unavailable"
		} else {
			s.status = "No autocomplete matches"
		}
		return nil
	}
	s.overlay = overlayAutocomplete
	s.completionItems = matches
	s.completionCursor = 0
	return nil
}

func (s *QueryScreen) updateHistoryOverlay(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, Keys.Escape) || key.Matches(msg, Keys.History):
		s.overlay = overlayNone
		s.historyInput.Blur()
		return nil
	case key.Matches(msg, Keys.Up):
		if s.historyCursor > 0 {
			s.historyCursor--
		}
		return nil
	case key.Matches(msg, Keys.Down):
		if s.historyCursor < len(s.historyItems)-1 {
			s.historyCursor++
		}
		return nil
	case key.Matches(msg, Keys.Enter):
		if s.historyCursor < len(s.historyItems) {
			s.editor.SetValue(s.historyItems[s.historyCursor].SQL)
			s.overlay = overlayNone
			s.historyInput.Blur()
			s.focus = focusEditor
			s.editor.Focus()
		}
		return nil
	}

	var cmd tea.Cmd
	s.historyInput, cmd = s.historyInput.Update(msg)
	return tea.Batch(cmd, s.loadHistory())
}

func (s *QueryScreen) updateExportOverlay(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, Keys.Escape) || key.Matches(msg, Keys.Export):
		s.overlay = overlayNone
		s.exportInput.Blur()
		return nil
	case key.Matches(msg, Keys.Enter):
		return s.exportResults()
	}

	var cmd tea.Cmd
	s.exportInput, cmd = s.exportInput.Update(msg)
	return cmd
}

func (s *QueryScreen) updateAutocompleteOverlay(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, Keys.Escape), key.Matches(msg, Keys.Autocomplete):
		s.overlay = overlayNone
		return nil
	case key.Matches(msg, Keys.Up):
		if s.completionCursor > 0 {
			s.completionCursor--
		}
		return nil
	case key.Matches(msg, Keys.Down):
		if s.completionCursor < len(s.completionItems)-1 {
			s.completionCursor++
		}
		return nil
	case key.Matches(msg, Keys.Enter):
		if s.completionCursor < len(s.completionItems) {
			lineInfo := s.editor.LineInfo()
			row := s.editor.Line()
			col := lineInfo.StartColumn + lineInfo.ColumnOffset
			prefix := completionPrefix(s.editor.Value(), row, col)
			selected := s.completionItems[s.completionCursor]
			if strings.HasPrefix(strings.ToLower(selected), strings.ToLower(prefix)) {
				s.editor.InsertString(selected[len(prefix):])
			}
			s.overlay = overlayNone
		}
		return nil
	}
	return nil
}

func (s *QueryScreen) loadHistory() tea.Cmd {
	if s.historyStore == nil {
		return func() tea.Msg {
			return historyLoadedMsg{err: fmt.Errorf("history store unavailable")}
		}
	}

	term := strings.TrimSpace(s.historyInput.Value())
	return func() tea.Msg {
		items, err := s.historyStore.Search(term, 50)
		return historyLoadedMsg{items: items, err: err}
	}
}

func (s *QueryScreen) loadCompletions() tea.Cmd {
	return func() tea.Msg {
		meta := s.driver.Meta(s.conn)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tables, err := meta.Tables(ctx, "")
		if err != nil {
			return completionsLoadedMsg{err: err}
		}

		tableColumns := make(map[string][]string, len(tables))
		for _, table := range tables {
			tableColumns[table.Name] = nil
			cols, err := meta.Columns(ctx, table.Schema, table.Name)
			if err != nil {
				continue
			}
			for _, col := range cols {
				tableColumns[table.Name] = append(tableColumns[table.Name], col.Name)
			}
		}
		return completionsLoadedMsg{items: buildCompletionItems(tableColumns)}
	}
}

func (s *QueryScreen) saveHistory(msg queryDoneMsg) tea.Cmd {
	if s.historyStore == nil || strings.TrimSpace(msg.sql) == "" {
		return nil
	}

	entry := history.Entry{
		SessionName: s.prof.Name,
		Driver:      s.prof.Driver,
		Database:    s.prof.Database,
		SQL:         msg.sql,
		DurationMS:  msg.duration.Milliseconds(),
		Success:     msg.err == nil,
		RanAt:       time.Now().UTC(),
	}
	if msg.result != nil {
		entry.RowCount = msg.result.RowCount
		if entry.DurationMS == 0 {
			entry.DurationMS = msg.result.Duration.Milliseconds()
		}
	}
	if msg.err != nil {
		entry.ErrorText = msg.err.Error()
	}

	return func() tea.Msg {
		return historySavedMsg{err: s.historyStore.Add(entry)}
	}
}

func (s *QueryScreen) exportResults() tea.Cmd {
	if s.results == nil || len(s.results.Columns) == 0 {
		s.exportErr = "No tabular results to export"
		return nil
	}

	path := strings.TrimSpace(s.exportInput.Value())
	if path == "" {
		s.exportErr = "Export file path is required"
		return nil
	}

	result := s.results
	format := inferExportFormat(path)

	return func() tea.Msg {
		dir := filepath.Dir(path)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return exportDoneMsg{err: fmt.Errorf("create export dir: %w", err)}
			}
		}

		f, err := os.Create(path)
		if err != nil {
			return exportDoneMsg{err: fmt.Errorf("create export file: %w", err)}
		}
		defer f.Close()

		if err := export.Write(f, result, format, false); err != nil {
			return exportDoneMsg{err: fmt.Errorf("write export: %w", err)}
		}

		return exportDoneMsg{path: path}
	}
}

func (s *QueryScreen) viewHistory() string {
	var sb strings.Builder
	sb.WriteString(StyleTitle.Render("  Query History") + "\n\n")
	sb.WriteString("  Search: " + s.historyInput.View() + "\n\n")

	if s.historyErr != "" {
		sb.WriteString(StyleError.Render("  "+s.historyErr) + "\n")
	} else if len(s.historyItems) == 0 {
		sb.WriteString(StyleHelp.Render("  No saved queries yet.") + "\n")
	} else {
		for i, item := range s.historyItems {
			line := fmt.Sprintf(
				"  %s  %-14s  %s",
				item.RanAt.Local().Format("01-02 15:04"),
				truncPad(item.SessionName, 14),
				truncPad(oneLine(item.SQL), 48),
			)
			if i == s.historyCursor {
				sb.WriteString(StyleSelected.Render(line) + "\n")
			} else {
				sb.WriteString(line + "\n")
			}
		}
	}

	sb.WriteString("\n")
	sb.WriteString(StyleHelp.Render("  Type: search  Up/Down: move  Enter: load query  Esc: close"))

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center,
		StyleActiveBorder.Width(minInt(s.width-4, 84)).Padding(1, 2).Render(sb.String()))
}

func (s *QueryScreen) viewExport() string {
	format := inferExportFormat(strings.TrimSpace(s.exportInput.Value()))

	var sb strings.Builder
	sb.WriteString(StyleTitle.Render("  Export Results") + "\n\n")
	sb.WriteString("  File Path: " + s.exportInput.View() + "\n")
	sb.WriteString("  Format:    " + string(format) + "\n")
	sb.WriteString("  Tip: .csv, .tsv, .json, .txt (or no extension for table)\n")

	if s.exportErr != "" {
		sb.WriteString("\n" + StyleError.Render("  "+s.exportErr))
	}

	sb.WriteString("\n\n")
	sb.WriteString(StyleHelp.Render("  Enter: export  Esc: cancel"))

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center,
		StyleActiveBorder.Width(minInt(s.width-4, 72)).Padding(1, 2).Render(sb.String()))
}

func (s *QueryScreen) viewAutocomplete() string {
	var sb strings.Builder
	sb.WriteString(StyleTitle.Render("  Autocomplete") + "\n\n")

	if len(s.completionItems) == 0 {
		sb.WriteString(StyleHelp.Render("  No suggestions available.") + "\n")
	} else {
		limit := len(s.completionItems)
		if limit > 12 {
			limit = 12
		}
		for i := 0; i < limit; i++ {
			line := "  " + s.completionItems[i]
			if i == s.completionCursor {
				sb.WriteString(StyleSelected.Render(line) + "\n")
			} else {
				sb.WriteString(line + "\n")
			}
		}
	}

	if s.completionErr != "" {
		sb.WriteString("\n" + StyleError.Render("  "+s.completionErr))
	}
	sb.WriteString("\n\n")
	sb.WriteString(StyleHelp.Render("  Up/Down: move  Enter: insert  Esc/Ctrl+Space/F9: close"))

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center,
		StyleActiveBorder.Width(minInt(s.width-4, 64)).Padding(1, 2).Render(sb.String()))
}

func inferExportFormat(path string) export.Format {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return export.FormatJSON
	case ".csv":
		return export.FormatCSV
	case ".tsv":
		return export.FormatTSV
	default:
		return export.FormatTable
	}
}

func defaultExportPath(session string) string {
	base := sanitizeFileName(session)
	if base == "" {
		base = "query-results"
	} else {
		base += "-results"
	}
	return "./" + base + ".csv"
}

func sanitizeFileName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func errorString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
