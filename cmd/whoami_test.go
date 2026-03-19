package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func captureWhoami(t *testing.T, jsonFlag bool) (string, string, error) {
	t.Helper()
	oldOut := os.Stdout
	oldErr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	whoamiJSON = jsonFlag
	cmd := whoamiCmd
	err := cmd.RunE(cmd, []string{})

	wOut.Close()
	wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	var bufOut, bufErr bytes.Buffer
	bufOut.ReadFrom(rOut)
	bufErr.ReadFrom(rErr)
	return bufOut.String(), bufErr.String(), err
}

func TestWhoami_ManagedIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)
	env.Run(t, "git", "config", "user.name", "Alice Smith")
	env.Run(t, "git", "config", "user.email", "alice@work.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "alice@work.com", GitName: "Alice Smith"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	out, _, err := captureWhoami(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Alice Smith <alice@work.com>") {
		t.Errorf("expected git author format, got: %q", out)
	}
}

func TestWhoami_NoIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	_, _, err := captureWhoami(t, false)
	if err == nil {
		t.Fatal("expected error for no identity")
	}
	if !strings.Contains(err.Error(), "no identity configured") {
		t.Errorf("expected 'no identity configured' error, got: %v", err)
	}
}

func TestWhoami_JSON(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)
	env.Run(t, "git", "config", "user.name", "Bob Jones")
	env.Run(t, "git", "config", "user.email", "bob@personal.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "personal", Email: "bob@personal.com", GitName: "Bob Jones"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	out, _, err := captureWhoami(t, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result whoamiOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Name != "personal" {
		t.Errorf("expected name 'personal', got %q", result.Name)
	}
	if result.Email != "bob@personal.com" {
		t.Errorf("expected email 'bob@personal.com', got %q", result.Email)
	}
	if !result.Managed {
		t.Error("expected managed=true")
	}
}

func TestWhoami_Unmanaged(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)
	env.Run(t, "git", "config", "user.name", "Random User")
	env.Run(t, "git", "config", "user.email", "random@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@company.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	out, _, err := captureWhoami(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Random User <random@example.com>") {
		t.Errorf("expected author format, got: %q", out)
	}
}

func TestWhoami_RuleMismatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)
	env.Run(t, "git", "config", "user.name", "Alice Smith")
	env.Run(t, "git", "config", "user.email", "alice@personal.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "personal", Email: "alice@personal.com", GitName: "Alice Smith"},
			{Name: "work", Email: "alice@work.com", GitName: "Alice Smith"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	out, stderr, err := captureWhoami(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Alice Smith <alice@personal.com>") {
		t.Errorf("expected current identity in output, got: %q", out)
	}
	if !strings.Contains(stderr, "expected 'work'") {
		t.Errorf("expected mismatch warning on stderr, got: %q", stderr)
	}
}

func TestWhoami_RuleMismatchJSON(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)
	env.Run(t, "git", "config", "user.name", "Alice")
	env.Run(t, "git", "config", "user.email", "alice@personal.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "personal", Email: "alice@personal.com"},
			{Name: "work", Email: "alice@work.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	out, _, err := captureWhoami(t, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result whoamiOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !result.Mismatch {
		t.Error("expected mismatch=true")
	}
	if result.ExpectedBy != "work" {
		t.Errorf("expected expected_by='work', got %q", result.ExpectedBy)
	}
}

func TestWhoami_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	// Clear git env vars that might interfere
	t.Setenv("GIT_DIR", "")

	_, _, err := captureWhoami(t, false)
	if err == nil {
		t.Fatal("expected error outside git repo")
	}
}

func TestWhoami_EmailOnly(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)
	// Set only email, no name
	cmd := exec.Command("git", "config", "user.email", "noname@example.com")
	cmd.Dir = env.Dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("set email: %v", err)
	}

	out, _, err := captureWhoami(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "<noname@example.com>") {
		t.Errorf("expected <email> format, got: %q", out)
	}
	if strings.Contains(out, " <") && !strings.HasPrefix(out, "<") {
		t.Errorf("expected no name prefix, got: %q", out)
	}
}
