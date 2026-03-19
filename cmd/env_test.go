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

func captureEnvOutput(t *testing.T) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	envJSON = false
	cmd := envCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func captureEnvJSONOutput(t *testing.T) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	envJSON = true
	cmd := envCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func TestEnv_NoVarsSet(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Ensure identity env vars are not set
	t.Setenv("GIT_AUTHOR_NAME", "")
	os.Unsetenv("GIT_AUTHOR_NAME")
	t.Setenv("GIT_AUTHOR_EMAIL", "")
	os.Unsetenv("GIT_AUTHOR_EMAIL")
	t.Setenv("GIT_COMMITTER_NAME", "")
	os.Unsetenv("GIT_COMMITTER_NAME")
	t.Setenv("GIT_COMMITTER_EMAIL", "")
	os.Unsetenv("GIT_COMMITTER_EMAIL")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureEnvOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "No git identity environment variables set") {
		t.Errorf("expected no-vars message, got: %s", output)
	}
	if !strings.Contains(output, "No environment overrides") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestEnv_ConflictingEmail(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")

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

	// Set conflicting env var
	t.Setenv("GIT_AUTHOR_EMAIL", "personal@gmail.com")

	output, err := captureEnvOutput(t)
	if err == nil {
		t.Fatal("expected error for conflict, got nil")
	}
	if !strings.Contains(output, "conflict") {
		t.Errorf("expected conflict in output, got: %s", output)
	}
	if !strings.Contains(output, "personal@gmail.com") {
		t.Errorf("expected conflicting email in output, got: %s", output)
	}
	if !strings.Contains(output, "override git config") {
		t.Errorf("expected override warning, got: %s", output)
	}
}

func TestEnv_MatchingVars(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")

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

	// Set matching env vars
	t.Setenv("GIT_AUTHOR_EMAIL", "work@example.com")
	t.Setenv("GIT_AUTHOR_NAME", "Worker")

	output, err := captureEnvOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "ok") {
		t.Errorf("expected ok status, got: %s", output)
	}
	if !strings.Contains(output, "match the expected identity") {
		t.Errorf("expected match message, got: %s", output)
	}
}

func TestEnv_JSON_Conflict(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")

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

	t.Setenv("GIT_COMMITTER_EMAIL", "wrong@example.com")

	output, err := captureEnvJSONOutput(t)
	if err == nil {
		t.Fatal("expected error for conflict, got nil")
	}
	if !strings.Contains(output, `"has_conflict": true`) {
		t.Errorf("expected has_conflict:true in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"conflict": true`) {
		t.Errorf("expected conflict:true in variable, got: %s", output)
	}
	if !strings.Contains(output, `"expected": "work@example.com"`) {
		t.Errorf("expected expected email in JSON, got: %s", output)
	}
}

func TestEnv_JSON_NoConflict(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	// Unset all identity env vars
	for _, v := range identityEnvVars {
		t.Setenv(v, "")
		os.Unsetenv(v)
	}

	output, err := captureEnvJSONOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, `"has_conflict": false`) {
		t.Errorf("expected has_conflict:false in JSON, got: %s", output)
	}
}
