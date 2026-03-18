package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestApply_LocalScope(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
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
	defer env.Cleanup(t)
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

func TestMatches_NoMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
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
