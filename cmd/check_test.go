package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func captureCheckOutput(t *testing.T) (string, error) {
	t.Helper()
	return captureCheckOutputWithFlags(t, false, false)
}

func captureCheckOutputWithFlags(t *testing.T, json, fix bool) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	checkJSON = json
	checkFix = fix
	cmd := checkCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func captureCheckJSONOutput(t *testing.T) (string, error) {
	t.Helper()
	return captureCheckOutputWithFlags(t, true, false)
}

func TestCheck_NoRule(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureCheckOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "No rule matches") {
		t.Errorf("expected no-rule message, got: %s", output)
	}
}

func TestCheck_IdentityMatches(t *testing.T) {
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

	output, err := captureCheckOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "OK") {
		t.Errorf("expected OK message, got: %s", output)
	}
	if !strings.Contains(output, "work") {
		t.Errorf("expected identity name in output, got: %s", output)
	}
}

func TestCheck_IdentityMismatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Set wrong identity
	env.Run(t, "git", "config", "user.email", "personal@gmail.com")
	env.Run(t, "git", "config", "user.name", "Personal")

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

	output, err := captureCheckOutput(t)
	if err == nil {
		t.Fatal("expected error for mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "identity does not match rule") {
		t.Errorf("expected mismatch error, got: %v", err)
	}
	if !strings.Contains(output, "FAIL") {
		t.Errorf("expected FAIL in output, got: %s", output)
	}
	if !strings.Contains(output, "personal@gmail.com") {
		t.Errorf("expected current email in output, got: %s", output)
	}
	if !strings.Contains(output, "work@example.com") {
		t.Errorf("expected expected email in output, got: %s", output)
	}
	if !strings.Contains(output, "gitch use work") {
		t.Errorf("expected fix suggestion in output, got: %s", output)
	}
}

func TestCheck_NoIdentityConfigured(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Don't set any git identity — rule expects one
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

	output, err := captureCheckOutput(t)
	if err == nil {
		t.Fatal("expected error for no identity, got nil")
	}
	if !strings.Contains(output, "FAIL") {
		t.Errorf("expected FAIL in output, got: %s", output)
	}
	if !strings.Contains(output, "no identity configured") {
		t.Errorf("expected no-identity message in output, got: %s", output)
	}
}

func TestCheck_JSON_Match(t *testing.T) {
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

	output, err := captureCheckJSONOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, `"ok": true`) {
		t.Errorf("expected ok:true in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"reason": "identity matches rule"`) {
		t.Errorf("expected match reason in JSON, got: %s", output)
	}
}

func TestCheck_JSON_Mismatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "personal@gmail.com")
	env.Run(t, "git", "config", "user.name", "Personal")

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

	output, err := captureCheckJSONOutput(t)
	if err == nil {
		t.Fatal("expected error for mismatch, got nil")
	}
	if !strings.Contains(output, `"ok": false`) {
		t.Errorf("expected ok:false in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"current_email": "personal@gmail.com"`) {
		t.Errorf("expected current email in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"expected_email": "work@example.com"`) {
		t.Errorf("expected expected email in JSON, got: %s", output)
	}
}

func TestCheck_Fix_AppliesCorrectIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Set wrong identity
	env.Run(t, "git", "config", "user.email", "personal@gmail.com")
	env.Run(t, "git", "config", "user.name", "Personal")

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

	output, err := captureCheckOutputWithFlags(t, false, true)
	if err != nil {
		t.Fatalf("expected no error with --fix, got: %v", err)
	}
	if !strings.Contains(output, "FIXED") {
		t.Errorf("expected FIXED in output, got: %s", output)
	}
	if !strings.Contains(output, "work") {
		t.Errorf("expected identity name in output, got: %s", output)
	}

	// Verify git config was actually updated
	cmd := exec.Command("git", "config", "user.email")
	cmd.Dir = env.Dir
	out, _ := cmd.Output()
	if got := strings.TrimSpace(string(out)); got != "work@example.com" {
		t.Errorf("expected git email to be work@example.com after fix, got: %s", got)
	}
}

func TestCheck_Fix_AlreadyCorrect(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Set correct identity
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

	output, err := captureCheckOutputWithFlags(t, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should show OK (not FIXED) since identity already matches
	if !strings.Contains(output, "OK") {
		t.Errorf("expected OK in output, got: %s", output)
	}
	if strings.Contains(output, "FIXED") {
		t.Errorf("should not show FIXED when identity already matches, got: %s", output)
	}
}

func TestCheck_Fix_NoRule(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureCheckOutputWithFlags(t, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No rule = nothing to fix, should be OK
	if !strings.Contains(output, "No rule matches") {
		t.Errorf("expected no-rule message, got: %s", output)
	}
}

func TestCheck_Fix_NoIdentityConfigured(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// No git identity set — rule expects one, --fix should apply it
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

	output, err := captureCheckOutputWithFlags(t, false, true)
	if err != nil {
		t.Fatalf("expected no error with --fix, got: %v", err)
	}
	if !strings.Contains(output, "FIXED") {
		t.Errorf("expected FIXED in output, got: %s", output)
	}

	// Verify git config was set
	cmd := exec.Command("git", "config", "user.email")
	cmd.Dir = env.Dir
	out, _ := cmd.Output()
	if got := strings.TrimSpace(string(out)); got != "work@example.com" {
		t.Errorf("expected git email to be work@example.com after fix, got: %s", got)
	}
}

func TestCheck_Fix_JSON(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Set wrong identity
	env.Run(t, "git", "config", "user.email", "personal@gmail.com")
	env.Run(t, "git", "config", "user.name", "Personal")

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

	output, err := captureCheckOutputWithFlags(t, true, true)
	if err != nil {
		t.Fatalf("expected no error with --fix --json, got: %v", err)
	}
	if !strings.Contains(output, `"ok": true`) {
		t.Errorf("expected ok:true in JSON after fix, got: %s", output)
	}
	if !strings.Contains(output, `"fixed": true`) {
		t.Errorf("expected fixed:true in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"reason": "identity mismatch fixed"`) {
		t.Errorf("expected fix reason in JSON, got: %s", output)
	}
}
