package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadPreviousIdentity(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITCH_CONFIG_PATH", filepath.Join(dir, "config.yaml"))

	// Initially no prev file — should return empty.
	name, err := LoadPreviousIdentity()
	if err != nil {
		t.Fatalf("LoadPreviousIdentity() error: %v", err)
	}
	if name != "" {
		t.Errorf("expected empty, got %q", name)
	}

	// Save and reload.
	if err := SavePreviousIdentity("work"); err != nil {
		t.Fatalf("SavePreviousIdentity() error: %v", err)
	}

	name, err = LoadPreviousIdentity()
	if err != nil {
		t.Fatalf("LoadPreviousIdentity() error: %v", err)
	}
	if name != "work" {
		t.Errorf("expected %q, got %q", "work", name)
	}

	// Overwrite with a new name.
	if err := SavePreviousIdentity("personal"); err != nil {
		t.Fatalf("SavePreviousIdentity() error: %v", err)
	}

	name, err = LoadPreviousIdentity()
	if err != nil {
		t.Fatalf("LoadPreviousIdentity() error: %v", err)
	}
	if name != "personal" {
		t.Errorf("expected %q, got %q", "personal", name)
	}
}

func TestPrevIdentityPath(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	t.Setenv("GITCH_CONFIG_PATH", configFile)

	path, err := PrevIdentityPath()
	if err != nil {
		t.Fatalf("PrevIdentityPath() error: %v", err)
	}
	expected := filepath.Join(dir, "prev-identity")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestLoadPreviousIdentity_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITCH_CONFIG_PATH", filepath.Join(dir, "config.yaml"))

	// Write a file with extra whitespace/newlines.
	prevPath := filepath.Join(dir, "prev-identity")
	if err := os.WriteFile(prevPath, []byte("  spacey-name  \n\n"), 0644); err != nil {
		t.Fatal(err)
	}

	name, err := LoadPreviousIdentity()
	if err != nil {
		t.Fatalf("LoadPreviousIdentity() error: %v", err)
	}
	if name != "spacey-name" {
		t.Errorf("expected %q, got %q", "spacey-name", name)
	}
}
