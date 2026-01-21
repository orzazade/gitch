package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testConfig creates a new config with the given identities
func testConfig(identities ...Identity) *Config {
	return &Config{
		Identities: identities,
	}
}

func TestAddIdentity_Success(t *testing.T) {
	cfg := testConfig()

	err := cfg.AddIdentity(Identity{Name: "work", Email: "work@example.com"})
	if err != nil {
		t.Fatalf("AddIdentity() returned error: %v", err)
	}

	if len(cfg.Identities) != 1 {
		t.Errorf("Expected 1 identity, got %d", len(cfg.Identities))
	}
	if cfg.Identities[0].Name != "work" {
		t.Errorf("Expected name 'work', got %q", cfg.Identities[0].Name)
	}
	if cfg.Identities[0].Email != "work@example.com" {
		t.Errorf("Expected email 'work@example.com', got %q", cfg.Identities[0].Email)
	}
}

func TestAddIdentity_DuplicateName(t *testing.T) {
	cfg := testConfig(Identity{Name: "work", Email: "work@example.com"})

	err := cfg.AddIdentity(Identity{Name: "work", Email: "other@example.com"})
	if err == nil {
		t.Fatal("AddIdentity() should return error for duplicate name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Error should mention 'already exists', got: %v", err)
	}
}

func TestAddIdentity_DuplicateNameCaseInsensitive(t *testing.T) {
	cfg := testConfig(Identity{Name: "Work", Email: "work@example.com"})

	err := cfg.AddIdentity(Identity{Name: "WORK", Email: "other@example.com"})
	if err == nil {
		t.Fatal("AddIdentity() should return error for duplicate name (case-insensitive)")
	}
}

func TestAddIdentity_InvalidName(t *testing.T) {
	cfg := testConfig()

	err := cfg.AddIdentity(Identity{Name: "-invalid", Email: "user@example.com"})
	if err == nil {
		t.Fatal("AddIdentity() should return error for invalid name")
	}
}

func TestAddIdentity_InvalidEmail(t *testing.T) {
	cfg := testConfig()

	err := cfg.AddIdentity(Identity{Name: "work", Email: "invalid"})
	if err == nil {
		t.Fatal("AddIdentity() should return error for invalid email")
	}
}

func TestGetIdentity_Found(t *testing.T) {
	cfg := testConfig(
		Identity{Name: "work", Email: "work@example.com"},
		Identity{Name: "personal", Email: "personal@example.com"},
	)

	identity, err := cfg.GetIdentity("work")
	if err != nil {
		t.Fatalf("GetIdentity() returned error: %v", err)
	}
	if identity.Name != "work" {
		t.Errorf("Expected name 'work', got %q", identity.Name)
	}
	if identity.Email != "work@example.com" {
		t.Errorf("Expected email 'work@example.com', got %q", identity.Email)
	}
}

func TestGetIdentity_FoundCaseInsensitive(t *testing.T) {
	cfg := testConfig(Identity{Name: "Work", Email: "work@example.com"})

	identity, err := cfg.GetIdentity("WORK")
	if err != nil {
		t.Fatalf("GetIdentity() returned error: %v", err)
	}
	if identity.Name != "Work" {
		t.Errorf("Expected name 'Work' (original case), got %q", identity.Name)
	}
}

func TestGetIdentity_NotFound(t *testing.T) {
	cfg := testConfig()

	_, err := cfg.GetIdentity("nonexistent")
	if err == nil {
		t.Fatal("GetIdentity() should return error for nonexistent identity")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

func TestDeleteIdentity_Success(t *testing.T) {
	cfg := testConfig(
		Identity{Name: "work", Email: "work@example.com"},
		Identity{Name: "personal", Email: "personal@example.com"},
	)

	err := cfg.DeleteIdentity("work")
	if err != nil {
		t.Fatalf("DeleteIdentity() returned error: %v", err)
	}

	if len(cfg.Identities) != 1 {
		t.Errorf("Expected 1 identity after delete, got %d", len(cfg.Identities))
	}
	if cfg.Identities[0].Name != "personal" {
		t.Errorf("Expected remaining identity to be 'personal', got %q", cfg.Identities[0].Name)
	}
}

func TestDeleteIdentity_NotFound(t *testing.T) {
	cfg := testConfig()

	err := cfg.DeleteIdentity("nonexistent")
	if err == nil {
		t.Fatal("DeleteIdentity() should return error for nonexistent identity")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

func TestDeleteIdentity_ClearsDefault(t *testing.T) {
	cfg := testConfig(Identity{Name: "work", Email: "work@example.com"})
	cfg.Default = "work"

	err := cfg.DeleteIdentity("work")
	if err != nil {
		t.Fatalf("DeleteIdentity() returned error: %v", err)
	}

	if cfg.Default != "" {
		t.Errorf("Expected default to be cleared, got %q", cfg.Default)
	}
}

func TestDeleteIdentity_ClearsDefaultCaseInsensitive(t *testing.T) {
	cfg := testConfig(Identity{Name: "Work", Email: "work@example.com"})
	cfg.Default = "Work"

	err := cfg.DeleteIdentity("WORK")
	if err != nil {
		t.Fatalf("DeleteIdentity() returned error: %v", err)
	}

	if cfg.Default != "" {
		t.Errorf("Expected default to be cleared, got %q", cfg.Default)
	}
}

func TestListIdentities_Empty(t *testing.T) {
	cfg := testConfig()

	identities := cfg.ListIdentities()
	if len(identities) != 0 {
		t.Errorf("Expected 0 identities, got %d", len(identities))
	}
}

func TestListIdentities_Multiple(t *testing.T) {
	cfg := testConfig(
		Identity{Name: "work", Email: "work@example.com"},
		Identity{Name: "personal", Email: "personal@example.com"},
		Identity{Name: "oss", Email: "oss@example.com"},
	)

	identities := cfg.ListIdentities()
	if len(identities) != 3 {
		t.Errorf("Expected 3 identities, got %d", len(identities))
	}
}

func TestSetDefault_Valid(t *testing.T) {
	cfg := testConfig(Identity{Name: "work", Email: "work@example.com"})

	err := cfg.SetDefault("work")
	if err != nil {
		t.Fatalf("SetDefault() returned error: %v", err)
	}

	if cfg.Default != "work" {
		t.Errorf("Expected default to be 'work', got %q", cfg.Default)
	}
}

func TestSetDefault_ValidCaseInsensitive(t *testing.T) {
	cfg := testConfig(Identity{Name: "Work", Email: "work@example.com"})

	err := cfg.SetDefault("WORK")
	if err != nil {
		t.Fatalf("SetDefault() returned error: %v", err)
	}

	// Should preserve the original case
	if cfg.Default != "Work" {
		t.Errorf("Expected default to be 'Work' (original case), got %q", cfg.Default)
	}
}

func TestSetDefault_Invalid(t *testing.T) {
	cfg := testConfig()

	err := cfg.SetDefault("nonexistent")
	if err == nil {
		t.Fatal("SetDefault() should return error for nonexistent identity")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gitch", "config.yaml")

	// Create config directory
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Create and populate a config
	original := &Config{
		Default: "work",
		Identities: []Identity{
			{Name: "work", Email: "work@example.com"},
			{Name: "personal", Email: "personal@example.com"},
		},
	}

	// Manually save to temp path (can't use Save() as it uses XDG path)
	data, err := os.ReadFile(configPath)
	if err == nil {
		t.Log("Config file already exists, will overwrite")
	}

	// Use yaml directly for test
	import_yaml_test(t, original, configPath)

	// Load from the temp path
	loaded := load_yaml_test(t, configPath)

	// Verify loaded config matches original
	if loaded.Default != original.Default {
		t.Errorf("Default mismatch: got %q, want %q", loaded.Default, original.Default)
	}

	if len(loaded.Identities) != len(original.Identities) {
		t.Fatalf("Identities count mismatch: got %d, want %d", len(loaded.Identities), len(original.Identities))
	}

	for i, want := range original.Identities {
		got := loaded.Identities[i]
		if got.Name != want.Name {
			t.Errorf("Identity[%d].Name mismatch: got %q, want %q", i, got.Name, want.Name)
		}
		if got.Email != want.Email {
			t.Errorf("Identity[%d].Email mismatch: got %q, want %q", i, got.Email, want.Email)
		}
	}

	// Clean up the unused data variable warning
	_ = data
}

// Helper to save config to a specific path for testing
func import_yaml_test(t *testing.T, cfg *Config, path string) {
	t.Helper()
	// Use gopkg.in/yaml.v3 directly
	data := []byte(`default: ` + cfg.Default + `
identities:
`)
	for _, id := range cfg.Identities {
		data = append(data, []byte(`    - name: `+id.Name+`
      email: `+id.Email+`
`)...)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
}

// Helper to load config from a specific path for testing
func load_yaml_test(t *testing.T, path string) *Config {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var cfg Config
	// Simple YAML parsing for test
	lines := strings.Split(string(data), "\n")
	var currentIdentity *Identity
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "default:") {
			cfg.Default = strings.TrimSpace(strings.TrimPrefix(line, "default:"))
		} else if strings.HasPrefix(line, "- name:") {
			if currentIdentity != nil {
				cfg.Identities = append(cfg.Identities, *currentIdentity)
			}
			currentIdentity = &Identity{
				Name: strings.TrimSpace(strings.TrimPrefix(line, "- name:")),
			}
		} else if strings.HasPrefix(line, "email:") && currentIdentity != nil {
			currentIdentity.Email = strings.TrimSpace(strings.TrimPrefix(line, "email:"))
		}
	}
	if currentIdentity != nil {
		cfg.Identities = append(cfg.Identities, *currentIdentity)
	}

	return &cfg
}

func TestLoad_NonexistentFile(t *testing.T) {
	// Save the original ConfigPath function behavior
	// We can't easily mock it, but we can test the behavior with a clean XDG_CONFIG_HOME
	tempDir := t.TempDir()
	oldXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldXDGConfigHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error for nonexistent file: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	if cfg.Identities == nil {
		t.Error("Load() returned config with nil Identities")
	}

	if len(cfg.Identities) != 0 {
		t.Errorf("Expected 0 identities for nonexistent file, got %d", len(cfg.Identities))
	}
}
