package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func captureAmendOutput(t *testing.T, force bool) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	amendJSON = false
	amendForce = force
	cmd := amendCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func captureAmendJSONOutput(t *testing.T, force bool) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	amendJSON = true
	amendForce = force
	cmd := amendCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

// setupAmendEnv creates a git repo with a commit using the "wrong" identity.
func setupAmendEnv(t *testing.T) *testutil.GitEnv {
	t.Helper()
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Configure git identity for the commit
	env.Run(t, "git", "config", "user.email", "personal@gmail.com")
	env.Run(t, "git", "config", "user.name", "Personal")

	// Create a commit
	env.Run(t, "git", "commit", "--allow-empty", "-m", "test commit")

	// Save gitch config with a rule pointing to "work"
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
			{Name: "personal", Email: "personal@gmail.com", GitName: "Personal"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	return env
}

func TestAmend_FixesMismatchedCommit(t *testing.T) {
	setupAmendEnv(t)

	output, err := captureAmendOutput(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "OK") {
		t.Errorf("expected OK in output, got: %s", output)
	}
	if !strings.Contains(output, "Amended") {
		t.Errorf("expected Amended in output, got: %s", output)
	}
	if !strings.Contains(output, "personal@gmail.com") {
		t.Errorf("expected old email in output, got: %s", output)
	}
	if !strings.Contains(output, "work@example.com") {
		t.Errorf("expected new email in output, got: %s", output)
	}

	// Verify the commit was actually amended
	env := &testutil.GitEnv{Dir: "."}
	authorEmail := strings.TrimSpace(env.Run(t, "git", "log", "-1", "--format=%ae"))
	if authorEmail != "work@example.com" {
		t.Errorf("commit author email not amended: got %s, want work@example.com", authorEmail)
	}
	authorName := strings.TrimSpace(env.Run(t, "git", "log", "-1", "--format=%an"))
	if authorName != "Worker" {
		t.Errorf("commit author name not amended: got %s, want Worker", authorName)
	}
}

func TestAmend_AlreadyCorrect(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Commit with the correct identity
	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")
	env.Run(t, "git", "commit", "--allow-empty", "-m", "correct commit")

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

	output, err := captureAmendOutput(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "already matches") {
		t.Errorf("expected already-matches message, got: %s", output)
	}
}

func TestAmend_NoRule(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "user@example.com")
	env.Run(t, "git", "config", "user.name", "User")
	env.Run(t, "git", "commit", "--allow-empty", "-m", "test commit")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	_, err := captureAmendOutput(t, false)
	if err == nil {
		t.Fatal("expected error when no rule matches, got nil")
	}
	if !strings.Contains(err.Error(), "no rule matches") {
		t.Errorf("expected no-rule error, got: %v", err)
	}
}

func TestAmend_JSON_Amended(t *testing.T) {
	setupAmendEnv(t)

	output, err := captureAmendJSONOutput(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, `"ok": true`) {
		t.Errorf("expected ok:true in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"action": "amended"`) {
		t.Errorf("expected action:amended in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"old_email": "personal@gmail.com"`) {
		t.Errorf("expected old email in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"new_email": "work@example.com"`) {
		t.Errorf("expected new email in JSON, got: %s", output)
	}
}

func TestAmend_JSON_AlreadyCorrect(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")
	env.Run(t, "git", "commit", "--allow-empty", "-m", "correct commit")

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

	output, err := captureAmendJSONOutput(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, `"ok": true`) {
		t.Errorf("expected ok:true in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"action": "none"`) {
		t.Errorf("expected action:none in JSON, got: %s", output)
	}
}
