package audit

import (
	"os/exec"
	"testing"
)

func Test_getCommits_WithCommits(t *testing.T) {
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

	commits, err := getCommits(0)
	if err != nil {
		t.Fatalf("getCommits failed: %v", err)
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

func Test_getCommits_WithLimit(t *testing.T) {
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

	commits, err := getCommits(2)
	if err != nil {
		t.Fatalf("getCommits failed: %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("expected 2 commits with limit, got %d", len(commits))
	}
}

func Test_getLocalOnlyHashes_WithUpstream(t *testing.T) {
	// Create a "remote" repo
	remoteDir := t.TempDir()
	for _, args := range [][]string{
		{"git", "init", "--bare", remoteDir},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	// Create local repo cloned from remote
	localDir := t.TempDir()
	t.Chdir(localDir)

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "remote", "add", "origin", remoteDir},
		{"git", "commit", "--allow-empty", "-m", "pushed commit"},
		{"git", "push", "-u", "origin", "HEAD:main"},
		{"git", "commit", "--allow-empty", "-m", "local only commit"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = localDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	hashes := getLocalOnlyHashes()
	if hashes == nil {
		t.Fatal("expected non-nil map with upstream configured")
	}
	// Should have exactly 1 local-only commit
	if len(hashes) != 1 {
		t.Errorf("expected 1 local-only hash, got %d", len(hashes))
	}
}

func Test_getLocalOnlyHashes_NoUpstream(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "commit"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	// No upstream configured — should return nil map
	hashes := getLocalOnlyHashes()
	if hashes != nil {
		t.Errorf("expected nil map when no upstream, got %v", hashes)
	}
}

func Test_getCommits_NotAGitRepo(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := getCommits(0)
	if err == nil {
		t.Fatal("expected error when not in a git repo")
	}
}

func Test_getCommits_EmptyRepo(t *testing.T) {
	repoDir := t.TempDir()
	t.Chdir(repoDir)

	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	commits, err := getCommits(0)
	if err != nil {
		t.Fatalf("getCommits should handle empty repo: %v", err)
	}

	if len(commits) != 0 {
		t.Errorf("expected 0 commits in empty repo, got %d", len(commits))
	}
}
