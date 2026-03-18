package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestApply_LocalScope(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	keyPath := filepath.Join(env.Dir, "id_test")
	if err := os.WriteFile(keyPath, []byte("not-a-real-key"), 0600); err != nil {
		t.Fatalf("failed to create SSH key fixture: %v", err)
	}

	identity := &config.Identity{
		Name:       "work",
		GitName:    "Jane Doe",
		Email:      "jane@example.com",
		SSHKeyPath: keyPath,
		GPGKeyID:   "ABC123",
	}

	if _, err := Apply(identity, git.ScopeLocal); err != nil {
		t.Fatalf("Apply(local) failed: %v", err)
	}

	localName, err := git.GetConfigScoped("user.name", git.ScopeLocal)
	if err != nil {
		t.Fatalf("failed to read local user.name: %v", err)
	}
	if localName != "Jane Doe" {
		t.Fatalf("local user.name = %q, want %q", localName, "Jane Doe")
	}

	localEmail, err := git.GetConfigScoped("user.email", git.ScopeLocal)
	if err != nil {
		t.Fatalf("failed to read local user.email: %v", err)
	}
	if localEmail != "jane@example.com" {
		t.Fatalf("local user.email = %q, want %q", localEmail, "jane@example.com")
	}

	signingKey, err := git.GetConfigScoped("user.signingkey", git.ScopeLocal)
	if err != nil {
		t.Fatalf("failed to read local signing key: %v", err)
	}
	if signingKey != "ABC123" {
		t.Fatalf("local signing key = %q, want %q", signingKey, "ABC123")
	}

	sshCommand, err := git.GetConfigScoped("core.sshCommand", git.ScopeLocal)
	if err != nil {
		t.Fatalf("failed to read local ssh command: %v", err)
	}
	expectedSSHCommand, err := ssh.GitSSHCommand(keyPath)
	if err != nil {
		t.Fatalf("failed to build expected ssh command: %v", err)
	}
	if sshCommand != expectedSSHCommand {
		t.Fatalf("local core.sshCommand = %q, want %q", sshCommand, expectedSSHCommand)
	}

	globalName, err := git.GetConfigScoped("user.name", git.ScopeGlobal)
	if err != nil {
		t.Fatalf("failed to read global user.name: %v", err)
	}
	if globalName != "" {
		t.Fatalf("global user.name = %q, want empty", globalName)
	}

	match, err := MatchesAtScope(identity, git.ScopeLocal)
	if err != nil {
		t.Fatalf("MatchesAtScope(local) failed: %v", err)
	}
	if !match {
		t.Fatal("expected local scope to match applied identity")
	}
}

func TestMatches_EffectiveScope(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	identity := &config.Identity{
		Name:    "work",
		GitName: "Jane Doe",
		Email:   "jane@example.com",
	}

	if _, err := Apply(identity, git.ScopeLocal); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Matches uses ScopeEffective, which should see local config
	match, err := Matches(identity)
	if err != nil {
		t.Fatalf("Matches failed: %v", err)
	}
	if !match {
		t.Error("expected Matches to return true for applied identity")
	}
}

func TestApply_MinimalIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Apply identity with no SSH key and no GPG key
	identity := &config.Identity{
		Name:    "minimal",
		GitName: "Min User",
		Email:   "min@example.com",
	}

	result, err := Apply(identity, git.ScopeLocal)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should succeed with no warnings (no SSH key to load, no GPG to configure)
	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings for minimal identity, got %v", result.Warnings)
	}

	// Verify identity was applied
	name, err := git.GetConfigScoped("user.name", git.ScopeLocal)
	if err != nil {
		t.Fatalf("failed to read user.name: %v", err)
	}
	if name != "Min User" {
		t.Errorf("user.name = %q, want %q", name, "Min User")
	}
}

func TestApply_ClearsSigningWhenNoGPG(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// First apply identity with GPG
	withGPG := &config.Identity{Name: "signed", GitName: "Signer", Email: "s@e.com", GPGKeyID: "KEY123"}
	if _, err := Apply(withGPG, git.ScopeLocal); err != nil {
		t.Fatalf("Apply with GPG failed: %v", err)
	}

	// Then apply identity without GPG — should clear signing config
	withoutGPG := &config.Identity{Name: "unsigned", GitName: "Plain", Email: "p@e.com"}
	if _, err := Apply(withoutGPG, git.ScopeLocal); err != nil {
		t.Fatalf("Apply without GPG failed: %v", err)
	}

	// Signing key should be cleared
	key, _ := git.GetConfigScoped("user.signingkey", git.ScopeLocal)
	if key != "" {
		t.Errorf("user.signingkey should be cleared, got %q", key)
	}
}

func TestApply_SSHKeyNotFound(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	identity := &config.Identity{
		Name:       "work",
		GitName:    "Jane Doe",
		Email:      "jane@work.com",
		SSHKeyPath: "/nonexistent/ssh/key",
	}

	result, err := Apply(identity, git.ScopeLocal)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Should have a warning about SSH key not found
	if len(result.Warnings) == 0 {
		t.Error("expected warning about missing SSH key")
	}

	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "SSH key not found") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'SSH key not found' warning, got: %v", result.Warnings)
	}
}

func TestApply_ClearsSSHWhenNoKey(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// First apply with SSH key path
	keyPath := filepath.Join(env.Dir, "id_test")
	if err := os.WriteFile(keyPath, []byte("not-a-key"), 0600); err != nil {
		t.Fatalf("failed to create key fixture: %v", err)
	}
	withSSH := &config.Identity{Name: "ssh-user", GitName: "SSH", Email: "ssh@e.com", SSHKeyPath: keyPath}
	if _, err := Apply(withSSH, git.ScopeLocal); err != nil {
		t.Fatalf("Apply with SSH failed: %v", err)
	}

	// Then apply without SSH — should clear core.sshCommand
	noSSH := &config.Identity{Name: "plain", GitName: "Plain", Email: "plain@e.com"}
	if _, err := Apply(noSSH, git.ScopeLocal); err != nil {
		t.Fatalf("Apply without SSH failed: %v", err)
	}

	sshCmd, _ := git.GetConfigScoped("core.sshCommand", git.ScopeLocal)
	if sshCmd != "" {
		t.Errorf("core.sshCommand should be cleared, got %q", sshCmd)
	}
}

func TestMatchesAtScope_NameMismatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Apply one name
	if err := git.ApplyIdentityScoped("Alice", "alice@e.com", git.ScopeLocal); err != nil {
		t.Fatalf("ApplyIdentityScoped failed: %v", err)
	}

	// Check against identity with same email but different name
	identity := &config.Identity{Name: "work", GitName: "Bob", Email: "alice@e.com"}
	match, err := MatchesAtScope(identity, git.ScopeLocal)
	if err != nil {
		t.Fatalf("MatchesAtScope failed: %v", err)
	}
	if match {
		t.Error("expected no match when git name differs")
	}
}

func TestMatches_NoMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Apply one identity
	applied := &config.Identity{Name: "work", GitName: "Jane", Email: "jane@work.com"}
	if _, err := Apply(applied, git.ScopeLocal); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Check against a different identity
	other := &config.Identity{Name: "personal", GitName: "Jane", Email: "jane@personal.com"}
	match, err := Matches(other)
	if err != nil {
		t.Fatalf("Matches failed: %v", err)
	}
	if match {
		t.Error("expected Matches to return false for different identity")
	}
}
