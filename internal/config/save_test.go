package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	t.Setenv("GITCH_CONFIG_PATH", configPath)

	cfg := &Config{
		Default: "work",
		Identities: []Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@work.com", HookMode: "block"},
			{Name: "personal", Email: "jane@personal.com"},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load it back
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Default != "work" {
		t.Errorf("expected default 'work', got %q", loaded.Default)
	}
	if len(loaded.Identities) != 2 {
		t.Fatalf("expected 2 identities, got %d", len(loaded.Identities))
	}
	if loaded.Identities[0].Name != "work" {
		t.Errorf("expected first identity 'work', got %q", loaded.Identities[0].Name)
	}
	if loaded.Identities[0].GitName != "Jane Doe" {
		t.Errorf("expected git name 'Jane Doe', got %q", loaded.Identities[0].GitName)
	}
	if loaded.Identities[0].HookMode != "block" {
		t.Errorf("expected hook mode 'block', got %q", loaded.Identities[0].HookMode)
	}
	if loaded.Identities[1].Email != "jane@personal.com" {
		t.Errorf("expected email 'jane@personal.com', got %q", loaded.Identities[1].Email)
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent", "config.yaml")

	t.Setenv("GITCH_CONFIG_PATH", configPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should not error for missing file: %v", err)
	}
	if len(cfg.Identities) != 0 {
		t.Errorf("expected empty identities, got %d", len(cfg.Identities))
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "dir", "config.yaml")

	t.Setenv("GITCH_CONFIG_PATH", configPath)

	cfg := &Config{
		Identities: []Identity{{Name: "test", Email: "t@t.com"}},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save should create nested directories: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created in nested directory")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	if err := os.WriteFile(configPath, []byte("{{invalid yaml: [unclosed"), 0644); err != nil {
		t.Fatalf("failed to write invalid yaml: %v", err)
	}

	t.Setenv("GITCH_CONFIG_PATH", configPath)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestConfigPath_RespectsGitchConfigPath(t *testing.T) {
	t.Setenv("GITCH_CONFIG_PATH", "/custom/path/config.yaml")

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath failed: %v", err)
	}
	if path != "/custom/path/config.yaml" {
		t.Errorf("expected /custom/path/config.yaml, got %q", path)
	}
}

func TestConfigPath_RespectsXDGConfigHome(t *testing.T) {
	t.Setenv("GITCH_CONFIG_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath failed: %v", err)
	}
	if path != "/tmp/xdg-test/gitch/config.yaml" {
		t.Errorf("expected /tmp/xdg-test/gitch/config.yaml, got %q", path)
	}
}
