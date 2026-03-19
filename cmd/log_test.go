package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestLogCmd_ShowsCommits(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)

	// Create a git repo with commits
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "alice@work.com"},
		{"git", "config", "user.name", "Alice"},
		{"git", "commit", "--allow-empty", "-m", "first commit"},
		{"git", "commit", "--allow-empty", "-m", "second commit"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	// Run gitch log --json
	logJSON = true
	logLimit = 10
	defer func() { logJSON = false }()

	err := runLog(logCmd, nil)
	if err != nil {
		t.Fatalf("runLog failed: %v", err)
	}
}

func TestLogCmd_JSONFormat(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "test commit"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	logJSON = true
	logLimit = 5
	defer func() { logJSON = false }()

	// Capture would need stdout redirect; just verify no error
	err := runLog(logCmd, nil)
	if err != nil {
		t.Fatalf("runLog --json failed: %v", err)
	}
}

func TestLogCmd_NotGitRepo(t *testing.T) {
	t.Chdir(t.TempDir())

	logJSON = false
	logLimit = 10

	err := runLog(logCmd, nil)
	if err == nil {
		t.Fatal("expected error when not in a git repo")
	}
}

func TestLogCmd_EmptyRepo(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)

	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	logJSON = false
	logLimit = 10

	err := runLog(logCmd, nil)
	if err != nil {
		t.Fatalf("runLog should handle empty repo: %v", err)
	}
}

// setupLogEnv creates a git repo with mixed-identity commits and a config with rules.
func setupLogEnv(t *testing.T) *testutil.GitEnv {
	t.Helper()
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Alice", Email: "alice@company.com"},
			{Name: "personal", GitName: "Alice", Email: "alice@gmail.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	makeCommit(t, env.Dir, "Alice", "alice@company.com", "correct work commit")
	makeCommit(t, env.Dir, "Alice", "alice@gmail.com", "wrong identity commit")
	makeCommit(t, env.Dir, "Alice", "alice@company.com", "another work commit")

	return env
}

func resetLogFlags() {
	logJSON = false
	logIdentity = ""
	logMismatches = false
	logLimit = 10
}

func TestLogCmd_IdentityFilter(t *testing.T) {
	setupLogEnv(t)
	defer resetLogFlags()

	logJSON = true
	logIdentity = "work"
	logLimit = 10

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLog(logCmd, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("runLog --identity work failed: %v", err)
	}

	var entries []logEntry
	if err := json.NewDecoder(r).Decode(&entries); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// Should only contain commits by alice@company.com (work identity)
	for _, e := range entries {
		if e.Email != "alice@company.com" {
			t.Errorf("expected only work commits, got email %q", e.Email)
		}
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 work commits, got %d", len(entries))
	}
}

func TestLogCmd_IdentityFilter_Personal(t *testing.T) {
	setupLogEnv(t)
	defer resetLogFlags()

	logJSON = true
	logIdentity = "personal"
	logLimit = 10

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLog(logCmd, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("runLog --identity personal failed: %v", err)
	}

	var entries []logEntry
	if err := json.NewDecoder(r).Decode(&entries); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 personal commit, got %d", len(entries))
	}
}

func TestLogCmd_MismatchesFilter(t *testing.T) {
	setupLogEnv(t)
	defer resetLogFlags()

	logJSON = true
	logMismatches = true
	logLimit = 10

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLog(logCmd, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("runLog --mismatches failed: %v", err)
	}

	var entries []logEntry
	if err := json.NewDecoder(r).Decode(&entries); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// Only the personal commit is a mismatch (rule expects work)
	if len(entries) != 1 {
		t.Errorf("expected 1 mismatch commit, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Match != "mismatch" {
			t.Errorf("expected match=mismatch, got %q", e.Match)
		}
	}
}

func TestLogCmd_MismatchesFilter_NoMatches(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)
	defer resetLogFlags()

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Alice", Email: "alice@company.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// All commits match the expected identity
	makeCommit(t, env.Dir, "Alice", "alice@company.com", "good commit")

	logJSON = true
	logMismatches = true
	logLimit = 10

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLog(logCmd, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("runLog --mismatches with no mismatches failed: %v", err)
	}

	var entries []logEntry
	if err := json.NewDecoder(r).Decode(&entries); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 mismatch commits, got %d", len(entries))
	}
}

func TestLogCmd_IdentityFilter_Unknown(t *testing.T) {
	setupLogEnv(t)
	defer resetLogFlags()

	logIdentity = "nonexistent"
	logLimit = 10

	err := runLog(logCmd, nil)
	if err == nil {
		t.Fatal("expected error for unknown identity name")
	}
}

func TestLogCmd_IdentityAndMismatches(t *testing.T) {
	setupLogEnv(t)
	defer resetLogFlags()

	// Combine both filters: --identity personal --mismatches
	// The personal commit IS a mismatch (rule expects work), so it should appear
	logJSON = true
	logIdentity = "personal"
	logMismatches = true
	logLimit = 10

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runLog(logCmd, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("runLog --identity personal --mismatches failed: %v", err)
	}

	var entries []logEntry
	if err := json.NewDecoder(r).Decode(&entries); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 commit (personal AND mismatch), got %d", len(entries))
	}
}

func TestLogEntry_JSONFields(t *testing.T) {
	entry := logEntry{
		Hash:     "abc12345",
		Date:     "2026-01-15",
		Email:    "alice@work.com",
		Subject:  "fix login bug",
		Identity: "work",
		Expected: "work",
		Match:    "ok",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal logEntry: %v", err)
	}

	var decoded map[string]string
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded["hash"] != "abc12345" {
		t.Errorf("expected hash abc12345, got %q", decoded["hash"])
	}
	if decoded["identity"] != "work" {
		t.Errorf("expected identity work, got %q", decoded["identity"])
	}
	if decoded["match"] != "ok" {
		t.Errorf("expected match ok, got %q", decoded["match"])
	}
}
