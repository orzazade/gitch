package audit

import (
	"os/exec"
	"testing"
)

func TestGetCommits_WithCommits(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test User"},
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

	commits, err := GetCommits(0)
	if err != nil {
		t.Fatalf("GetCommits failed: %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	if commits[0].AuthorEmail != "test@test.com" {
		t.Errorf("expected email test@test.com, got %q", commits[0].AuthorEmail)
	}
	if commits[0].AuthorName != "Test User" {
		t.Errorf("expected name Test User, got %q", commits[0].AuthorName)
	}
}

func TestGetCommits_WithLimit(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "first"},
		{"git", "commit", "--allow-empty", "-m", "second"},
		{"git", "commit", "--allow-empty", "-m", "third"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	commits, err := GetCommits(2)
	if err != nil {
		t.Fatalf("GetCommits failed: %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("expected 2 commits with limit, got %d", len(commits))
	}
}

func TestGetCommits_EmptyRepo(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)

	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	commits, err := GetCommits(0)
	if err != nil {
		t.Fatalf("GetCommits should handle empty repo: %v", err)
	}

	if len(commits) != 0 {
		t.Errorf("expected 0 commits in empty repo, got %d", len(commits))
	}
}
