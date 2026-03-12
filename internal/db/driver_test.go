package db

import (
	"testing"
	"time"
)

func TestResolvePoolSettingsUsesDefaults(t *testing.T) {
	got := ResolvePoolSettings(ConnConfig{}, 5, 2)

	if got.MaxOpenConns != 5 {
		t.Fatalf("MaxOpenConns = %d, want 5", got.MaxOpenConns)
	}
	if got.MaxIdleConns != 2 {
		t.Fatalf("MaxIdleConns = %d, want 2", got.MaxIdleConns)
	}
	if got.ConnMaxLifetime != 0 {
		t.Fatalf("ConnMaxLifetime = %s, want 0", got.ConnMaxLifetime)
	}
}

func TestResolvePoolSettingsUsesOverridesAndClampsIdle(t *testing.T) {
	got := ResolvePoolSettings(ConnConfig{
		MaxOpenConns:           3,
		MaxIdleConns:           8,
		ConnMaxLifetimeSeconds: 30,
	}, 5, 2)

	if got.MaxOpenConns != 3 {
		t.Fatalf("MaxOpenConns = %d, want 3", got.MaxOpenConns)
	}
	if got.MaxIdleConns != 3 {
		t.Fatalf("MaxIdleConns = %d, want 3", got.MaxIdleConns)
	}
	if got.ConnMaxLifetime != 30*time.Second {
		t.Fatalf("ConnMaxLifetime = %s, want 30s", got.ConnMaxLifetime)
	}
}
