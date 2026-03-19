package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestRunLink_AppliesIdentityAndCreatesRuleAndHook(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	keyPath := filepath.Join(env.Dir, "id_test")
	if err := os.WriteFile(keyPath, []byte("not-a-real-key"), 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane", Email: "jane@work.com", SSHKeyPath: keyPath},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	err := runLink(nil, []string{"work"})
	if err != nil {
		t.Fatalf("runLink failed: %v", err)
	}

	// Identity was applied locally.
	localEmail, err := git.GetConfigScoped("user.email", git.ScopeLocal)
	if err != nil {
		t.Fatalf("read local user.email: %v", err)
	}
	if localEmail != "jane@work.com" {
		t.Errorf("local user.email = %q, want %q", localEmail, "jane@work.com")
	}

	// Rule was created.
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if len(cfg.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(cfg.Rules))
	}
	if cfg.Rules[0].Identity != "work" {
		t.Errorf("rule identity = %q, want %q", cfg.Rules[0].Identity, "work")
	}
	if cfg.Rules[0].Type != rules.DirectoryRule {
		t.Errorf("rule type = %q, want %q", cfg.Rules[0].Type, rules.DirectoryRule)
	}

	// Hook was installed.
	hookPath := filepath.Join(env.Dir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Errorf("pre-commit hook was not installed at %s", hookPath)
	}
}

func TestRunLink_SkipsRuleWhenAlreadyMatched(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	keyPath := filepath.Join(env.Dir, "id_test")
	if err := os.WriteFile(keyPath, []byte("not-a-real-key"), 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane", Email: "jane@work.com", SSHKeyPath: keyPath},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir + "/**", Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	err := runLink(nil, []string{"work"})
	if err != nil {
		t.Fatalf("runLink failed: %v", err)
	}

	// Rule count should remain 1 (no duplicate).
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if len(cfg.Rules) != 1 {
		t.Errorf("got %d rules, want 1 (no duplicate)", len(cfg.Rules))
	}
}

func TestRunLink_NotGitRepo(t *testing.T) {
	// Use a completely separate directory that is not inside any git repo.
	nonGitDir := t.TempDir()
	t.Chdir(nonGitDir)
	t.Setenv("GIT_CEILING_DIRECTORIES", nonGitDir)

	err := runLink(nil, []string{"work"})
	if err == nil {
		t.Fatal("expected error for non-git directory, got nil")
	}
	if err != errNotGitRepo {
		t.Errorf("error = %v, want errNotGitRepo", err)
	}
}

func TestRunLink_UnknownIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane", Email: "jane@work.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	err := runLink(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown identity, got nil")
	}
}

func TestRunLink_WarnsOnConflictingRule(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	keyPath := filepath.Join(env.Dir, "id_test")
	if err := os.WriteFile(keyPath, []byte("not-a-real-key"), 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane", Email: "jane@work.com", SSHKeyPath: keyPath},
			{Name: "personal", GitName: "Jane", Email: "jane@home.com", SSHKeyPath: keyPath},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir + "/**", Identity: "personal"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	// Linking "work" when "personal" rule exists should still succeed
	// (adds a new rule, warns about conflict).
	err := runLink(nil, []string{"work"})
	if err != nil {
		t.Fatalf("runLink failed: %v", err)
	}

	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	// Should now have 2 rules.
	if len(cfg.Rules) != 2 {
		t.Errorf("got %d rules, want 2", len(cfg.Rules))
	}
}

func TestGitToplevel(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	top, err := gitToplevel()
	if err != nil {
		t.Fatalf("gitToplevel: %v", err)
	}
	if top != env.Dir {
		t.Errorf("gitToplevel() = %q, want %q", top, env.Dir)
	}
}
