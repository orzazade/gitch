package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	if err := SetConfigScoped("user.name", "Test User", ScopeGlobal); err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	// Read it back
	value, err := GetConfigScoped("user.name", ScopeGlobal)
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
	value, err := GetConfigScoped("user.nonexistent", ScopeGlobal)
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
	if err := SetConfigScoped("user.email", "test@example.com", ScopeGlobal); err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	// Verify it was set by reading it back
	value, err := GetConfigScoped("user.email", ScopeGlobal)
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
	if err := SetConfigScoped("user.name", "Jane Doe", ScopeGlobal); err != nil {
		t.Fatalf("failed to set name: %v", err)
	}
	if err := SetConfigScoped("user.email", "jane@example.com", ScopeGlobal); err != nil {
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
	if err := SetConfigScoped("user.name", "Partial User", ScopeGlobal); err != nil {
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

func TestGetCurrentIdentity_EffectivePrefersLocal(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	t.Chdir(env.dir)

	if err := SetConfigScoped("user.name", "Global User", ScopeGlobal); err != nil {
		t.Fatalf("failed to set global name: %v", err)
	}
	if err := SetConfigScoped("user.email", "global@example.com", ScopeGlobal); err != nil {
		t.Fatalf("failed to set global email: %v", err)
	}
	if err := SetConfigScoped("user.name", "Local User", ScopeLocal); err != nil {
		t.Fatalf("failed to set local name: %v", err)
	}
	if err := SetConfigScoped("user.email", "local@example.com", ScopeLocal); err != nil {
		t.Fatalf("failed to set local email: %v", err)
	}

	name, email, err := GetCurrentIdentity()
	if err != nil {
		t.Fatalf("GetCurrentIdentity failed: %v", err)
	}
	if name != "Local User" {
		t.Fatalf("effective user.name = %q, want %q", name, "Local User")
	}
	if email != "local@example.com" {
		t.Fatalf("effective user.email = %q, want %q", email, "local@example.com")
	}
}

func TestApplyIdentity_Success(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Apply identity
	if err := ApplyIdentityScoped("Alice Smith", "alice@example.com", ScopeGlobal); err != nil {
		t.Fatalf("ApplyIdentity failed: %v", err)
	}

	// Verify values were set
	name, err := GetConfigScoped("user.name", ScopeGlobal)
	if err != nil {
		t.Fatalf("failed to get name: %v", err)
	}
	email, err := GetConfigScoped("user.email", ScopeGlobal)
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
	if err := ApplyIdentityScoped("Bob Jones", "bob@example.com", ScopeGlobal); err != nil {
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
	t.Chdir(env.dir)

	// Set local config (not global)
	if err := SetConfigScoped("user.name", "Local User", ScopeLocal); err != nil {
		t.Fatalf("failed to set local config: %v", err)
	}

	// Read local config
	value, err := GetConfigScoped("user.name", ScopeLocal)
	if err != nil {
		t.Fatalf("GetConfig local failed: %v", err)
	}

	if value != "Local User" {
		t.Errorf("expected 'Local User', got '%s'", value)
	}

	// Global should still be empty
	globalValue, err := GetConfigScoped("user.name", ScopeGlobal)
	if err != nil {
		t.Fatalf("GetConfig global failed: %v", err)
	}

	if globalValue != "" {
		t.Errorf("expected empty global value, got '%s'", globalValue)
	}
}

func TestGitPath_HooksDir(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	t.Chdir(env.dir)

	path, err := GitPath("hooks/pre-commit")
	if err != nil {
		t.Fatalf("GitPath failed: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	if !strings.Contains(path, "hooks") || !strings.Contains(path, "pre-commit") {
		t.Errorf("expected path containing hooks/pre-commit, got %q", path)
	}
}

func TestUnsetConfigScoped_Idempotent(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Unsetting a key that was never set should not error
	err := UnsetConfigScoped("user.nonexistent", ScopeGlobal)
	if err != nil {
		t.Fatalf("UnsetConfigScoped should be idempotent: %v", err)
	}
}

func TestUnsetConfigScoped_RemovesKey(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)

	// Set a value
	if err := SetConfigScoped("user.name", "To Remove", ScopeGlobal); err != nil {
		t.Fatalf("SetConfigScoped failed: %v", err)
	}

	// Unset it
	if err := UnsetConfigScoped("user.name", ScopeGlobal); err != nil {
		t.Fatalf("UnsetConfigScoped failed: %v", err)
	}

	// Verify it's gone
	value, err := GetConfigScoped("user.name", ScopeGlobal)
	if err != nil {
		t.Fatalf("GetConfigScoped failed: %v", err)
	}
	if value != "" {
		t.Errorf("expected empty value after unset, got %q", value)
	}
}
