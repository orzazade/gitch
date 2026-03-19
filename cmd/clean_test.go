package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func captureCleanOutput(t *testing.T) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cleanJSON = false
	cleanFix = false
	cmd := cleanCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func captureCleanJSONOutput(t *testing.T) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cleanJSON = true
	cleanFix = false
	cmd := cleanCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func captureCleanFixOutput(t *testing.T) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cleanJSON = false
	cleanFix = true
	cmd := cleanCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func TestClean_NoIssues(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")

	cfg := &config.Config{
		Default: "work",
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

	output, err := captureCleanOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "clean") {
		t.Errorf("expected clean message, got: %s", output)
	}
}

func TestClean_OrphanedDirectoryRule(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")

	nonExistentDir := filepath.Join(t.TempDir(), "does-not-exist", "subdir")

	cfg := &config.Config{
		Default: "work",
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
			{Type: rules.DirectoryRule, Pattern: nonExistentDir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureCleanOutput(t)
	if err == nil {
		t.Fatal("expected error for orphaned rule, got nil")
	}
	if !strings.Contains(output, "orphaned") {
		t.Errorf("expected orphaned in output, got: %s", output)
	}
	if !strings.Contains(output, "does not exist") {
		t.Errorf("expected 'does not exist' in output, got: %s", output)
	}
}

func TestClean_UnusedIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")

	cfg := &config.Config{
		Default: "work",
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
			{Name: "stale", Email: "stale@example.com", GitName: "Stale"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureCleanOutput(t)
	if err == nil {
		t.Fatal("expected error for unused identity, got nil")
	}
	if !strings.Contains(output, "unused") {
		t.Errorf("expected unused in output, got: %s", output)
	}
	if !strings.Contains(output, "stale") {
		t.Errorf("expected stale identity in output, got: %s", output)
	}
}

func TestClean_OrphanedRuleRef(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "deleted-identity"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureCleanOutput(t)
	if err == nil {
		t.Fatal("expected error for orphaned rule ref, got nil")
	}
	if !strings.Contains(output, "non-existent identity") {
		t.Errorf("expected non-existent identity message, got: %s", output)
	}
	if !strings.Contains(output, "deleted-identity") {
		t.Errorf("expected deleted-identity in output, got: %s", output)
	}
}

func TestClean_Fix_RemovesOrphanedRules(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")

	nonExistentDir := filepath.Join(t.TempDir(), "gone")

	cfg := &config.Config{
		Default: "work",
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
			{Type: rules.DirectoryRule, Pattern: nonExistentDir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureCleanFixOutput(t)
	if err != nil {
		t.Fatalf("unexpected error after fix: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "removed") {
		t.Errorf("expected 'removed' in output, got: %s", output)
	}

	// Verify rule was actually removed from config
	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if len(reloaded.ListRules()) != 1 {
		t.Errorf("expected 1 rule after fix, got %d", len(reloaded.ListRules()))
	}
}

func TestClean_JSON_Output(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	nonExistentDir := filepath.Join(t.TempDir(), "nope")

	cfg := &config.Config{
		Default: "work",
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
			{Name: "orphan", Email: "orphan@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: nonExistentDir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureCleanJSONOutput(t)
	if err == nil {
		t.Fatal("expected error for issues, got nil")
	}
	if !strings.Contains(output, `"total_issues"`) {
		t.Errorf("expected total_issues in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"orphaned_rules"`) {
		t.Errorf("expected orphaned_rules in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"unused_identities"`) {
		t.Errorf("expected unused_identities in JSON, got: %s", output)
	}
}

func TestClean_RemoteRulesNotOrphaned(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Default: "work",
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
		Rules: []rules.Rule{
			{Type: rules.RemoteRule, Pattern: "github.com/myorg/*", Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureCleanOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "clean") {
		t.Errorf("expected clean message for remote-only rules, got: %s", output)
	}
}

func TestClean_GlobPatternNotOrphaned(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Default: "work",
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "/tmp/projects/*", Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureCleanOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "clean") {
		t.Errorf("expected clean message for glob patterns, got: %s", output)
	}
}
