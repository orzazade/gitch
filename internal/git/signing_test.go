package git

import (
	"os"
	"testing"
)

func TestApplySigningConfigScoped_SetsKeys(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)
	if err := os.Chdir(env.dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	keyID := "ABCD1234"
	if err := ApplySigningConfigScoped(keyID, ScopeLocal); err != nil {
		t.Fatalf("ApplySigningConfigScoped failed: %v", err)
	}

	gotKey, err := GetConfigScoped("user.signingkey", ScopeLocal)
	if err != nil {
		t.Fatalf("GetConfigScoped(user.signingkey) failed: %v", err)
	}
	if gotKey != keyID {
		t.Errorf("user.signingkey = %q, want %q", gotKey, keyID)
	}

	gotSign, err := GetConfigScoped("commit.gpgsign", ScopeLocal)
	if err != nil {
		t.Fatalf("GetConfigScoped(commit.gpgsign) failed: %v", err)
	}
	if gotSign != "true" {
		t.Errorf("commit.gpgsign = %q, want \"true\"", gotSign)
	}
}

func TestClearSigningConfigScoped_RemovesKeys(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)
	if err := os.Chdir(env.dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	if err := ApplySigningConfigScoped("ABCD1234", ScopeLocal); err != nil {
		t.Fatalf("ApplySigningConfigScoped failed: %v", err)
	}
	if err := ClearSigningConfigScoped(ScopeLocal); err != nil {
		t.Fatalf("ClearSigningConfigScoped failed: %v", err)
	}

	gotKey, err := GetConfigScoped("user.signingkey", ScopeLocal)
	if err != nil {
		t.Fatalf("GetConfigScoped failed after clear: %v", err)
	}
	if gotKey != "" {
		t.Errorf("user.signingkey should be empty after clear, got %q", gotKey)
	}
}

func TestClearSigningConfigScoped_Idempotent(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)
	if err := os.Chdir(env.dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Clear when nothing is set — should not error
	if err := ClearSigningConfigScoped(ScopeLocal); err != nil {
		t.Fatalf("ClearSigningConfigScoped on empty config returned error: %v", err)
	}
}

func TestIsGitRepository_InRepo(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup(t)
	if err := os.Chdir(env.dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	if !IsGitRepository() {
		t.Error("expected IsGitRepository() = true inside git repo")
	}
}
