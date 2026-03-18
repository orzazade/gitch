package cmd

import (
	"encoding/json"
	"os/exec"
	"testing"
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
