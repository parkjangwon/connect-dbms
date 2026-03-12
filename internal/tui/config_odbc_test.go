//go:build tibero && cubrid

package tui

import (
	"reflect"
	"testing"
)

func TestAvailableDriversIncludesODBCDriversInConfiguredOrder(t *testing.T) {
	got := availableDrivers()
	want := []string{"mysql", "mariadb", "oracle", "postgresql", "tibero", "cubrid", "sqlite"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("availableDrivers() = %v, want %v", got, want)
	}
}
