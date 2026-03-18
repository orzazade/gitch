package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func captureVerifyOutput(t *testing.T) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := verifyCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

// setupFakeUpstream configures a fake upstream so getLocalOnlyHashes works.
// Must be called after at least one commit exists.
func setupFakeUpstream(t *testing.T, env *testutil.GitEnv) {
	t.Helper()
	branch := strings.TrimSpace(env.Run(t, "git", "rev-parse", "--abbrev-ref", "HEAD"))
	env.Run(t, "git", "branch", "fake-upstream")
	env.Run(t, "git", "config", fmt.Sprintf("branch.%s.remote", branch), ".")
	env.Run(t, "git", "config", fmt.Sprintf("branch.%s.merge", branch), "refs/heads/fake-upstream")
}

func TestVerify_NoRule(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Empty config — no rules
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureVerifyOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "No identity rule matches") {
		t.Errorf("expected no-rule message, got: %s", output)
	}
}

func TestVerify_NoUpstream(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")
	env.Run(t, "git", "commit", "--allow-empty", "-m", "init")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureVerifyOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "No upstream branch configured") {
		t.Errorf("expected no-upstream message, got: %s", output)
	}
}

func TestVerify_AllMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Create initial commit and set up a fake upstream
	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")
	env.Run(t, "git", "commit", "--allow-empty", "-m", "init")

	// Create a fake upstream at init, then add an unpushed commit
	setupFakeUpstream(t, env)
	env.Run(t, "git", "commit", "--allow-empty", "-m", "new work")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureVerifyOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "OK") {
		t.Errorf("expected OK message, got: %s", output)
	}
	if !strings.Contains(output, "work@example.com") {
		t.Errorf("expected email in output, got: %s", output)
	}
}

func TestVerify_Mismatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Initial commit with correct identity
	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")
	env.Run(t, "git", "commit", "--allow-empty", "-m", "init")

	// Set up fake upstream at this point
	setupFakeUpstream(t, env)

	// Now commit with WRONG identity (simulating forgotten switch)
	env.Run(t, "git", "config", "user.email", "personal@gmail.com")
	env.Run(t, "git", "config", "user.name", "Personal")
	env.Run(t, "git", "commit", "--allow-empty", "-m", "oops wrong identity")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureVerifyOutput(t)
	if err == nil {
		t.Fatal("expected error for mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "identity mismatch") {
		t.Errorf("expected identity mismatch error, got: %v", err)
	}
	if !strings.Contains(output, "FAIL") {
		t.Errorf("expected FAIL in output, got: %s", output)
	}
	if !strings.Contains(output, "personal@gmail.com") {
		t.Errorf("expected wrong email in output, got: %s", output)
	}
	if !strings.Contains(output, "gitch audit --fix") {
		t.Errorf("expected fix suggestion in output, got: %s", output)
	}
}

func TestVerify_NoUnpushedCommits(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")
	env.Run(t, "git", "commit", "--allow-empty", "-m", "init")

	// Upstream points to same commit — nothing unpushed
	setupFakeUpstream(t, env)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureVerifyOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "No unpushed commits") {
		t.Errorf("expected no-unpushed message, got: %s", output)
	}
}
