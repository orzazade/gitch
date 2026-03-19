package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func captureUnuseOutput(t *testing.T) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := unuseCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func TestUnuse_ClearsLocalIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Set a local identity manually.
	env.Run(t, "git", "config", "--local", "user.name", "Worker")
	env.Run(t, "git", "config", "--local", "user.email", "work@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureUnuseOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Cleared") {
		t.Errorf("expected 'Cleared' message, got: %s", output)
	}
	if !strings.Contains(output, "work@example.com") {
		t.Errorf("expected email in output, got: %s", output)
	}

	// Verify local config was actually cleared.
	localName, localEmail, _ := gitpkg.GetCurrentIdentityScoped(gitpkg.ScopeLocal)
	if localName != "" {
		t.Errorf("expected local user.name to be empty, got: %s", localName)
	}
	if localEmail != "" {
		t.Errorf("expected local user.email to be empty, got: %s", localEmail)
	}
}

func TestUnuse_NoLocalOverride(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureUnuseOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "No local identity override") {
		t.Errorf("expected no-override message, got: %s", output)
	}
}

func TestUnuse_NotGitRepo(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := captureUnuseOutput(t)
	if err == nil {
		t.Fatal("expected error outside git repo, got nil")
	}
	if !strings.Contains(err.Error(), "not in a git repository") {
		t.Errorf("expected not-in-repo error, got: %v", err)
	}
}

func TestUnuse_ShowsRuleMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Set a local identity manually.
	env.Run(t, "git", "config", "--local", "user.name", "Worker")
	env.Run(t, "git", "config", "--local", "user.email", "work@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureUnuseOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Cleared") {
		t.Errorf("expected 'Cleared' message, got: %s", output)
	}
	if !strings.Contains(output, "Rule match") {
		t.Errorf("expected rule match info, got: %s", output)
	}
	if !strings.Contains(output, "work") {
		t.Errorf("expected identity name in rule match, got: %s", output)
	}
}
