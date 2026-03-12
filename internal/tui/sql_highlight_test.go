package tui

import (
	"strings"
	"testing"
)

func TestFirstNonEmptyLine(t *testing.T) {
	sql := "\n   \nSELECT * FROM users\nWHERE id = 1"
	got := firstNonEmptyLine(sql)
	want := "SELECT * FROM users"
	if got != want {
		t.Fatalf("firstNonEmptyLine() = %q, want %q", got, want)
	}
}

func TestHighlightSQLContentPreservesNonKeywordText(t *testing.T) {
	got := highlightSQLContent("select users.email from users")
	if !strings.Contains(got, "users.email") {
		t.Fatalf("highlightSQLContent() should preserve identifiers, got %q", got)
	}
}
