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

func TestWhy_ShowsMatchedRule(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Set up an identity and rule that matches the test directory
	cfg := &config.Config{
		Identities: []config.Identity{
			{
				Name:  "work",
				Email: "work@example.com",
			},
		},
		Rules: []rules.Rule{
			{
				Type:     rules.DirectoryRule,
				Pattern:  env.Dir,
				Identity: "work",
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Apply the identity so it's active
	result, err := tryAutoSwitch(cfg)
	if err != nil {
		t.Fatalf("tryAutoSwitch failed: %v", err)
	}
	if !result.Switched {
		t.Fatal("expected switch to happen")
	}

	// Capture output of gitch why
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := whyCmd
	err = cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runWhy failed: %v\nOutput: %s", err, output)
	}

	// Verify key sections are present
	if !strings.Contains(output, "Current Identity") {
		t.Error("output missing 'Current Identity' section")
	}
	if !strings.Contains(output, "work@example.com") {
		t.Error("output missing identity email")
	}
	if !strings.Contains(output, "Rule Evaluation") {
		t.Error("output missing 'Rule Evaluation' section")
	}
	if !strings.Contains(output, "Reason") {
		t.Error("output missing 'Reason' section")
	}
	if !strings.Contains(output, env.Dir) {
		t.Errorf("output missing test directory %s", env.Dir)
	}
}

func TestWhy_NoRules(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Set a git identity so we get past the "no identity" check
	env.Run(t, "git", "config", "user.email", "me@example.com")
	env.Run(t, "git", "config", "user.name", "Personal")

	// Set up identity without rules
	cfg := &config.Config{
		Identities: []config.Identity{
			{
				Name:  "personal",
				Email: "me@example.com",
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := whyCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runWhy failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "No rules configured") {
		t.Error("output should mention no rules configured")
	}
}

func TestWhy_NoIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Empty config
	cfg := &config.Config{}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := whyCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runWhy failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "No git identity configured") {
		t.Error("output should indicate no identity configured")
	}
	if !strings.Contains(output, "gitch use") {
		t.Error("output should suggest 'gitch use' as fix")
	}
}
