package tui

import (
	"reflect"
	"testing"
)

func TestCompletionPrefix(t *testing.T) {
	sql := "SELECT * FROM us"
	got := completionPrefix(sql, 0, len("SELECT * FROM us"))
	want := "us"
	if got != want {
		t.Fatalf("completionPrefix() = %q, want %q", got, want)
	}
}

func TestFilterCompletionsMatchesPrefix(t *testing.T) {
	got := filterCompletions("us", []string{"users", "user_logs", "orders"})
	want := []string{"users", "user_logs"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterCompletions() = %v, want %v", got, want)
	}
}

func TestCompletionPrefixSupportsTableDotColumn(t *testing.T) {
	sql := "SELECT users.em FROM users"
	got := completionPrefix(sql, 0, len("SELECT users.em"))
	want := "users.em"
	if got != want {
		t.Fatalf("completionPrefix() = %q, want %q", got, want)
	}
}

func TestFilterCompletionsMatchesTableDotColumn(t *testing.T) {
	got := filterCompletions("users.em", []string{"users.email", "users.created_at", "orders.email"})
	want := []string{"users.email"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterCompletions() = %v, want %v", got, want)
	}
}

func TestFilterCompletionsReturnsAllForEmptyPrefix(t *testing.T) {
	got := filterCompletions("", []string{"users", "orders"})
	want := []string{"users", "orders"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterCompletions() = %v, want %v", got, want)
	}
}

func TestBuildCompletionItemsIncludesTableColumnForms(t *testing.T) {
	got := buildCompletionItems(map[string][]string{
		"users":  {"id", "email"},
		"orders": {"id"},
	})
	want := []string{
		"orders",
		"orders.id",
		"users",
		"users.email",
		"users.id",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildCompletionItems() = %v, want %v", got, want)
	}
}
