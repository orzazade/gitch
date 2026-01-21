package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// testGitEnv sets up an isolated git environment for testing.
// It creates a temp directory with a git repo and uses GIT_CONFIG_GLOBAL
// to point to a temp config file, ensuring tests don't modify user's real config.
type testGitEnv struct {
	dir          string
	globalConfig string
	origEnv      map[string]string
}

// setupTestEnv creates an isolated git testing environment.
func setupTestEnv(t *testing.T) *testGitEnv {
	t.Helper()

	// Create temp directory
	dir, err := os.MkdirTemp("", "gitch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create temp global config file
	globalConfig := filepath.Join(dir, ".gitconfig")
	if err := os.WriteFile(globalConfig, []byte{}, 0644); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to create temp gitconfig: %v", err)
	}

	// Save original environment
	origEnv := make(map[string]string)
	for _, key := range []string{"GIT_CONFIG_GLOBAL", "HOME", "XDG_CONFIG_HOME"} {
		origEnv[key] = os.Getenv(key)
	}

	// Set isolated environment
	// GIT_CONFIG_GLOBAL points git to our temp config file for --global operations
	os.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	// Set HOME to temp dir to prevent git from reading user's real config
	os.Setenv("HOME", dir)
	// Clear XDG_CONFIG_HOME to avoid any XDG-based config loading
	os.Setenv("XDG_CONFIG_HOME", dir)

	// Initialize a git repo in the temp directory for local config tests
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to init git repo: %v", err)
	}

	return &testGitEnv{
		dir:          dir,
		globalConfig: globalConfig,
		origEnv:      origEnv,
	}
}

// cleanup restores the original environment and removes temp files.
func (e *testGitEnv) cleanup(t *testing.T) {
	t.Helper()

	// Restore original environment
	for key, val := range e.origEnv {
		if val == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, val)
		}
	}

	// Remove temp directory
	os.RemoveAll(e.dir)
}

func TestGetConfig_ExistingKey(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Set a value first
	if err := SetConfig("user.name", "Test User", true); err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	// Read it back
	value, err := GetConfig("user.name", true)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if value != "Test User" {
		t.Errorf("expected 'Test User', got '%s'", value)
	}
}

func TestGetConfig_MissingKey(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Read a key that doesn't exist
	value, err := GetConfig("user.nonexistent", true)
	if err != nil {
		t.Fatalf("GetConfig for missing key should not error: %v", err)
	}

	if value != "" {
		t.Errorf("expected empty string for missing key, got '%s'", value)
	}
}

func TestSetConfig_Success(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Set a value
	if err := SetConfig("user.email", "test@example.com", true); err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	// Verify it was set by reading it back
	value, err := GetConfig("user.email", true)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if value != "test@example.com" {
		t.Errorf("expected 'test@example.com', got '%s'", value)
	}
}

func TestGetCurrentIdentity_BothSet(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Set both values
	if err := SetConfig("user.name", "Jane Doe", true); err != nil {
		t.Fatalf("failed to set name: %v", err)
	}
	if err := SetConfig("user.email", "jane@example.com", true); err != nil {
		t.Fatalf("failed to set email: %v", err)
	}

	// Get identity
	name, email, err := GetCurrentIdentity()
	if err != nil {
		t.Fatalf("GetCurrentIdentity failed: %v", err)
	}

	if name != "Jane Doe" {
		t.Errorf("expected name 'Jane Doe', got '%s'", name)
	}
	if email != "jane@example.com" {
		t.Errorf("expected email 'jane@example.com', got '%s'", email)
	}
}

func TestGetCurrentIdentity_PartiallySet(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Set only name (email will be missing)
	if err := SetConfig("user.name", "Partial User", true); err != nil {
		t.Fatalf("failed to set name: %v", err)
	}

	// Get identity - should succeed with empty email
	name, email, err := GetCurrentIdentity()
	if err != nil {
		t.Fatalf("GetCurrentIdentity failed: %v", err)
	}

	if name != "Partial User" {
		t.Errorf("expected name 'Partial User', got '%s'", name)
	}
	if email != "" {
		t.Errorf("expected empty email, got '%s'", email)
	}
}

func TestGetCurrentIdentity_NoneSet(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Get identity with nothing set
	name, email, err := GetCurrentIdentity()
	if err != nil {
		t.Fatalf("GetCurrentIdentity failed: %v", err)
	}

	if name != "" {
		t.Errorf("expected empty name, got '%s'", name)
	}
	if email != "" {
		t.Errorf("expected empty email, got '%s'", email)
	}
}

func TestApplyIdentity_Success(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Apply identity
	if err := ApplyIdentity("Alice Smith", "alice@example.com"); err != nil {
		t.Fatalf("ApplyIdentity failed: %v", err)
	}

	// Verify values were set
	name, err := GetConfig("user.name", true)
	if err != nil {
		t.Fatalf("failed to get name: %v", err)
	}
	email, err := GetConfig("user.email", true)
	if err != nil {
		t.Fatalf("failed to get email: %v", err)
	}

	if name != "Alice Smith" {
		t.Errorf("expected name 'Alice Smith', got '%s'", name)
	}
	if email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got '%s'", email)
	}
}

func TestApplyIdentity_VerifyPersistence(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Apply identity
	if err := ApplyIdentity("Bob Jones", "bob@example.com"); err != nil {
		t.Fatalf("ApplyIdentity failed: %v", err)
	}

	// Use GetCurrentIdentity to verify (different code path)
	name, email, err := GetCurrentIdentity()
	if err != nil {
		t.Fatalf("GetCurrentIdentity failed: %v", err)
	}

	if name != "Bob Jones" {
		t.Errorf("expected name 'Bob Jones', got '%s'", name)
	}
	if email != "bob@example.com" {
		t.Errorf("expected email 'bob@example.com', got '%s'", email)
	}
}

func TestGetConfig_LocalScope(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Change to temp directory for local config operations
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(env.dir)

	// Set local config (not global)
	if err := SetConfig("user.name", "Local User", false); err != nil {
		t.Fatalf("failed to set local config: %v", err)
	}

	// Read local config
	value, err := GetConfig("user.name", false)
	if err != nil {
		t.Fatalf("GetConfig local failed: %v", err)
	}

	if value != "Local User" {
		t.Errorf("expected 'Local User', got '%s'", value)
	}

	// Global should still be empty
	globalValue, err := GetConfig("user.name", true)
	if err != nil {
		t.Fatalf("GetConfig global failed: %v", err)
	}

	if globalValue != "" {
		t.Errorf("expected empty global value, got '%s'", globalValue)
	}
}
