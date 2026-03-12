package tui

import (
	"regexp"
	"strings"
)

var sqlKeywordPattern = regexp.MustCompile(`(?i)\b(select|from|where|join|left|right|inner|outer|group|order|by|limit|insert|into|values|update|set|delete|create|table|alter|drop|with|as|having|distinct)\b`)

func renderSQLPreview(sql string) string {
	line := firstNonEmptyLine(sql)
	if line == "" {
		return StyleHelp.Render("No SQL preview yet.")
	}
	return highlightSQLContent(line)
}

func firstNonEmptyLine(sql string) string {
	for _, line := range strings.Split(sql, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func highlightSQLContent(text string) string {
	return sqlKeywordPattern.ReplaceAllStringFunc(text, func(match string) string {
		return StyleSubtitle.Render(strings.ToUpper(match))
	})
}
