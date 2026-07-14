package store

import (
	"path/filepath"
	"testing"
)

func TestNewUsesJumpStartConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	s, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	want := filepath.Join(home, ".jumpstart", "config.json")
	if s.path != want {
		t.Fatalf("Store path = %q, want %q", s.path, want)
	}
}
