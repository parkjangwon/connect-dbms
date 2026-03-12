package tui

import (
	"reflect"
	"testing"

	"oslo/internal/db"
	"oslo/internal/profile"
)

func TestAvailableDriversMatchesRegistry(t *testing.T) {
	got := availableDrivers()
	want := expectedConfigDrivers()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("availableDrivers() = %v, want %v", got, want)
	}
}

func TestValidateProfileForSaveRequiresSQLitePathOrDSN(t *testing.T) {
	field, err := validateProfileForSave(profile.Profile{
		Name:   "local-sqlite",
		Driver: "sqlite",
	})
	if err == nil {
		t.Fatal("expected sqlite validation error, got nil")
	}
	if field != fieldDatabase {
		t.Fatalf("field = %d, want %d", field, fieldDatabase)
	}
}

func TestValidateProfileForSaveAcceptsSQLiteFilePath(t *testing.T) {
	field, err := validateProfileForSave(profile.Profile{
		Name:     "local-sqlite",
		Driver:   "sqlite",
		Database: "./data.db",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if field != -1 {
		t.Fatalf("field = %d, want -1", field)
	}
}

func TestFormFieldLabelForSQLiteUsesFilePath(t *testing.T) {
	got := formFieldLabel(fieldDatabase, "sqlite")
	want := "File Path"
	if got != want {
		t.Fatalf("formFieldLabel(fieldDatabase, sqlite) = %q, want %q", got, want)
	}
}

func expectedConfigDrivers() []string {
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
