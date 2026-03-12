package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesMissingConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connect-dbms", "config.json")

	store, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if store.Path != path {
		t.Fatalf("store.Path = %q, want %q", store.Path, path)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file to exist, stat error = %v", err)
	}
}
