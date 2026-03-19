package cmd

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

// makeCommit creates a git commit with the given author identity.
func makeCommit(t *testing.T, dir, name, email, msg string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "--allow-empty",
		"--author", name+" <"+email+">", "-m", msg)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_COMMITTER_NAME="+name,
		"GIT_COMMITTER_EMAIL="+email,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}
}

func setupStatsEnv(t *testing.T) *testutil.GitEnv {
	t.Helper()
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@company.com"},
			{Name: "personal", GitName: "Jane", Email: "jane@gmail.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create commits with different identities
	makeCommit(t, env.Dir, "Jane Doe", "jane@company.com", "work commit 1")
	makeCommit(t, env.Dir, "Jane Doe", "jane@company.com", "work commit 2")
	makeCommit(t, env.Dir, "Jane Doe", "jane@company.com", "work commit 3")
	makeCommit(t, env.Dir, "Jane", "jane@gmail.com", "personal commit 1")
	makeCommit(t, env.Dir, "Unknown", "unknown@other.org", "unknown commit 1")

	return env
}

func TestRunStats_GroupsByEmail(t *testing.T) {
	setupStatsEnv(t)
	statsJSON = false
	statsLimit = 0
	statsAll = true
	statsSince = ""

	err := runStats(statsCmd, nil)
	if err != nil {
		t.Fatalf("runStats() error = %v", err)
	}
}

func TestRunStats_JSON(t *testing.T) {
	setupStatsEnv(t)
	statsJSON = true
	statsLimit = 0
	statsAll = true
	statsSince = ""

	err := runStats(statsCmd, nil)
	if err != nil {
		t.Fatalf("runStats() error = %v", err)
	}
}

func TestRunStats_EmptyRepo(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@company.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	statsJSON = false
	statsLimit = 0
	statsAll = false
	statsSince = ""

	err := runStats(statsCmd, nil)
	if err != nil {
		t.Fatalf("runStats() error = %v, want nil for empty repo", err)
	}
}

func TestRunStats_NoRules(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@company.com"},
		},
		Rules: []rules.Rule{},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	makeCommit(t, env.Dir, "Jane Doe", "jane@company.com", "test commit")

	statsJSON = false
	statsLimit = 0
	statsAll = true
	statsSince = ""

	err := runStats(statsCmd, nil)
	if err != nil {
		t.Fatalf("runStats() error = %v", err)
	}
}

func TestRunStats_LimitFlag(t *testing.T) {
	setupStatsEnv(t)
	statsJSON = false
	statsLimit = 2
	statsAll = false
	statsSince = ""

	// Should succeed with only 2 commits analyzed
	err := runStats(statsCmd, nil)
	if err != nil {
		t.Fatalf("runStats() error = %v", err)
	}
}

func TestRunStats_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	configDir := t.TempDir()
	t.Setenv("GITCH_CONFIG_PATH", filepath.Join(configDir, "config.yaml"))

	statsJSON = false
	statsLimit = 0
	statsAll = false
	statsSince = ""

	err := runStats(statsCmd, nil)
	if err == nil {
		t.Fatal("expected error when not in git repo")
	}
}
