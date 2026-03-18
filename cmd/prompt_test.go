package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func capturePromptOutput(t *testing.T) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := promptCmd
	cmd.Run(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestPrompt_ManagedIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Work User")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	output := capturePromptOutput(t)
	if output != "work" {
		t.Errorf("expected %q, got %q", "work", output)
	}
}

func TestPrompt_RuleMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Work User")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	output := capturePromptOutput(t)
	if output != "work" {
		t.Errorf("expected %q, got %q", "work", output)
	}
}

func TestPrompt_RuleMismatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Current identity is personal, but rule expects work
	env.Run(t, "git", "config", "user.email", "personal@example.com")
	env.Run(t, "git", "config", "user.name", "Personal")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
			{Name: "personal", Email: "personal@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	output := capturePromptOutput(t)
	if output != "work!" {
		t.Errorf("expected %q, got %q", "work!", output)
	}
}

func TestPrompt_UnmanagedIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "random@example.com")
	env.Run(t, "git", "config", "user.name", "Random")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	output := capturePromptOutput(t)
	if output != "?" {
		t.Errorf("expected %q, got %q", "?", output)
	}
}

func TestPrompt_NoGitIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// No git config set — empty email
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	output := capturePromptOutput(t)
	if output != "" {
		t.Errorf("expected empty output, got %q", output)
	}
}

func TestPrompt_NotGitRepo(t *testing.T) {
	// Use a temp dir that is NOT a git repo
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("HOME", dir)

	output := capturePromptOutput(t)
	if output != "" {
		t.Errorf("expected empty output outside git repo, got %q", output)
	}
}

func TestPrompt_CaseInsensitiveEmailMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "Work@Example.COM")
	env.Run(t, "git", "config", "user.name", "Work User")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	output := capturePromptOutput(t)
	if output != "work" {
		t.Errorf("expected %q, got %q", "work", output)
	}
}
