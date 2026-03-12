package sshtunnel

import (
	"testing"

	"oslo/internal/db"
)

func TestAuthMethodRequiresPasswordOrKey(t *testing.T) {
	_, err := authMethod(db.SSHConfig{
		Host: "jump.example.com",
		User: "tester",
	})
	if err == nil {
		t.Fatal("expected authMethod() error, got nil")
	}
}

func TestPrepareConnConfigWithoutSSHReturnsOriginalConfig(t *testing.T) {
	cfg := db.ConnConfig{
		Driver: "postgresql",
		Host:   "db.example.com",
		Port:   5432,
	}

	got, closer, err := PrepareConnConfig(cfg)
	if err != nil {
		t.Fatalf("PrepareConnConfig() error = %v", err)
	}
	if closer != nil {
		t.Fatal("expected nil closer for non-SSH config")
	}
	if got.Host != cfg.Host || got.Port != cfg.Port {
		t.Fatalf("PrepareConnConfig() = %#v, want original host/port", got)
	}
}
