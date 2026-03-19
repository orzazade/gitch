package cmd

import (
	"path/filepath"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestSavePrevIdentity_SavesCurrentMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@example.com"},
			{Name: "personal", GitName: "Jane", Email: "jane@personal.dev"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Set the current identity to "work".
	if err := git.ApplyIdentityScoped("Jane Doe", "jane@example.com", git.ScopeLocal); err != nil {
		t.Fatalf("failed to apply identity: %v", err)
	}

	savePrevIdentity(cfg)

	got, err := config.LoadPreviousIdentity()
	if err != nil {
		t.Fatalf("LoadPreviousIdentity() error: %v", err)
	}
	if got != "work" {
		t.Errorf("expected prev identity %q, got %q", "work", got)
	}
}

func TestSavePrevIdentity_NoIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// No identity is configured in git — savePrevIdentity should be a no-op.
	savePrevIdentity(cfg)

	got, err := config.LoadPreviousIdentity()
	if err != nil {
		t.Fatalf("LoadPreviousIdentity() error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty prev identity, got %q", got)
	}
}

func TestRunPrev_NoPreviousIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// No prev-identity file exists — runPrev should error.
	err := runPrev(prevCmd, nil)
	if err == nil {
		t.Fatal("expected error when no previous identity exists")
	}
	if err.Error() != "no previous identity — switch identities first with 'gitch use'" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunPrev_DeletedIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Save a prev identity that doesn't exist in config.
	if err := config.SavePreviousIdentity("deleted"); err != nil {
		t.Fatalf("SavePreviousIdentity() error: %v", err)
	}

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	err := runPrev(prevCmd, nil)
	if err == nil {
		t.Fatal("expected error when previous identity no longer exists")
	}
}

func TestRunPrev_SwitchesBack(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@example.com"},
			{Name: "personal", GitName: "Jane", Email: "jane@personal.dev"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Set current identity to "work".
	if err := git.ApplyIdentityScoped("Jane Doe", "jane@example.com", git.ScopeLocal); err != nil {
		t.Fatalf("failed to apply identity: %v", err)
	}

	// Simulate: user was on "work", save it as prev, then switch to "personal".
	if err := config.SavePreviousIdentity("work"); err != nil {
		t.Fatalf("SavePreviousIdentity() error: %v", err)
	}
	if err := git.ApplyIdentityScoped("Jane", "jane@personal.dev", git.ScopeLocal); err != nil {
		t.Fatalf("failed to apply identity: %v", err)
	}

	// Now run prev — should switch back to "work".
	err := runPrev(prevCmd, nil)
	if err != nil {
		t.Fatalf("runPrev() error: %v", err)
	}

	// Verify git config is now "work".
	_, email, err := git.GetCurrentIdentity()
	if err != nil {
		t.Fatalf("GetCurrentIdentity() error: %v", err)
	}
	if email != "jane@example.com" {
		t.Errorf("expected email %q after prev, got %q", "jane@example.com", email)
	}

	// Verify the prev file now points to "personal" (toggle behavior).
	prev, err := config.LoadPreviousIdentity()
	if err != nil {
		t.Fatalf("LoadPreviousIdentity() error: %v", err)
	}
	if prev != "personal" {
		t.Errorf("expected prev to be %q after toggle, got %q", "personal", prev)
	}
}

func TestPrevIdentityPath_UsesConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITCH_CONFIG_PATH", filepath.Join(dir, "config.yaml"))

	path, err := config.PrevIdentityPath()
	if err != nil {
		t.Fatalf("PrevIdentityPath() error: %v", err)
	}
	expected := filepath.Join(dir, "prev-identity")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}
