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

func captureRuleCheckOutput(t *testing.T) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := ruleCheckCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func TestRuleCheck_AllGood(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

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

	output, err := captureRuleCheckOutput(t)
	if err != nil {
		t.Fatalf("expected no error, got: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "OK") {
		t.Errorf("expected OK message, got: %s", output)
	}
	if !strings.Contains(output, "1 rule(s)") {
		t.Errorf("expected rule count, got: %s", output)
	}
}

func TestRuleCheck_NoRules(t *testing.T) {
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

	output, err := captureRuleCheckOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "No rules configured") {
		t.Errorf("expected no-rules message, got: %s", output)
	}
}

func TestRuleCheck_UnknownIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "nonexistent"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRuleCheckOutput(t)
	if err == nil {
		t.Fatal("expected error for unknown identity, got nil")
	}
	if !strings.Contains(output, "unknown identity") {
		t.Errorf("expected unknown identity message, got: %s", output)
	}
	if !strings.Contains(output, "nonexistent") {
		t.Errorf("expected identity name in output, got: %s", output)
	}
}

func TestRuleCheck_MissingDirectory(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "/nonexistent/path/**", Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRuleCheckOutput(t)
	if err == nil {
		t.Fatal("expected error for missing directory, got nil")
	}
	if !strings.Contains(output, "does not exist") {
		t.Errorf("expected missing-directory message, got: %s", output)
	}
}

func TestRuleCheck_ConflictingOverlap(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
			{Name: "personal", Email: "personal@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir + "/**", Identity: "work"},
			{Type: rules.DirectoryRule, Pattern: env.Dir + "/sub/**", Identity: "personal"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRuleCheckOutput(t)
	if err == nil {
		t.Fatal("expected error for overlapping rules, got nil")
	}
	if !strings.Contains(output, "overlaps") {
		t.Errorf("expected overlap warning, got: %s", output)
	}
}

func TestRuleCheck_SameIdentityOverlapNoIssue(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Create the sub directory so the directory-exists check passes.
	if err := os.MkdirAll(env.Dir+"/sub", 0755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir + "/**", Identity: "work"},
			{Type: rules.DirectoryRule, Pattern: env.Dir + "/sub/**", Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRuleCheckOutput(t)
	if err != nil {
		t.Fatalf("expected no error for same-identity overlap, got: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "OK") {
		t.Errorf("expected OK, got: %s", output)
	}
}

func TestRuleCheck_RemoteRuleSkipsDirectoryCheck(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.RemoteRule, Pattern: "github.com/myorg/*", Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRuleCheckOutput(t)
	if err != nil {
		t.Fatalf("expected no error for remote rule, got: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "OK") {
		t.Errorf("expected OK, got: %s", output)
	}
}

func TestRuleCheck_DidYouMeanSuggestion(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "wrk"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRuleCheckOutput(t)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(output, "Did you mean") {
		t.Errorf("expected did-you-mean suggestion, got: %s", output)
	}
	if !strings.Contains(output, "work") {
		t.Errorf("expected suggested identity name, got: %s", output)
	}
}

func TestDirectoryRuleBasePath(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{"/home/user/work/**", "/home/user/work"},
		{"/home/user/work/*", "/home/user/work"},
		{"/home/user/work", "/home/user/work"},
		{"**/foo", ""},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := directoryRuleBasePath(tt.pattern)
			if got != tt.want {
				t.Errorf("directoryRuleBasePath(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}
