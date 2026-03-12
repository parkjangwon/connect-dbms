package tui

import (
	"sort"
	"strings"
)

func completionPrefix(sql string, row, col int) string {
	lines := strings.Split(sql, "\n")
	if row < 0 || row >= len(lines) {
		return ""
	}
	line := lines[row]
	if col < 0 {
		col = 0
	}
	if col > len(line) {
		col = len(line)
	}

	start := col
	for start > 0 {
		if isCompletionByte(line[start-1]) {
			start--
			continue
		}
		break
	}
	return line[start:col]
}

func filterCompletions(prefix string, items []string) []string {
	prefix = strings.ToLower(prefix)
	var matches []string
	for _, item := range items {
		if prefix == "" || strings.HasPrefix(strings.ToLower(item), prefix) {
			matches = append(matches, item)
		}
	}
	return matches
}

func buildCompletionItems(tableColumns map[string][]string) []string {
	var items []string
	for table, columns := range tableColumns {
		items = append(items, table)
		for _, column := range columns {
			items = append(items, table+"."+column)
		}
	}
	sort.Strings(items)
	return items
}

func isCompletionByte(ch byte) bool {
	return ch == '_' || ch == '.' ||
		(ch >= '0' && ch <= '9') ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z')
}
