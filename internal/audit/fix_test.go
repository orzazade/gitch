package audit

import (
	"os/exec"
	"sort"
	"strings"
	"testing"
)

func TestGenerateMailmap_SingleMismatch(t *testing.T) {
	mismatches := []Result{
		{
			Commit:        Commit{AuthorEmail: "wrong@example.com"},
			ExpectedEmail: "correct@example.com",
			IsMismatched:  true,
		},
	}

	result := GenerateMailmap(mismatches, "correct@example.com")
	expected := "<correct@example.com> <wrong@example.com>"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestGenerateMailmap_MultipleDifferentEmails(t *testing.T) {
	mismatches := []Result{
		{Commit: Commit{AuthorEmail: "wrong1@example.com"}, IsMismatched: true},
		{Commit: Commit{AuthorEmail: "wrong2@example.com"}, IsMismatched: true},
	}

	result := GenerateMailmap(mismatches, "correct@example.com")
	lines := strings.Split(result, "\n")
	sort.Strings(lines)

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), result)
	}

	// Sorted order
	if lines[0] != "<correct@example.com> <wrong1@example.com>" {
		t.Errorf("unexpected line[0]: %q", lines[0])
	}
	if lines[1] != "<correct@example.com> <wrong2@example.com>" {
		t.Errorf("unexpected line[1]: %q", lines[1])
	}
}

func TestGenerateMailmap_DeduplicatesSameEmail(t *testing.T) {
	mismatches := []Result{
		{Commit: Commit{AuthorEmail: "wrong@example.com"}, IsMismatched: true},
		{Commit: Commit{AuthorEmail: "wrong@example.com"}, IsMismatched: true},
		{Commit: Commit{AuthorEmail: "wrong@example.com"}, IsMismatched: true},
	}

	result := GenerateMailmap(mismatches, "correct@example.com")
	lines := strings.Split(result, "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 unique line, got %d: %q", len(lines), result)
	}
}

func TestGenerateMailmap_SkipsNonMismatched(t *testing.T) {
	mismatches := []Result{
		{Commit: Commit{AuthorEmail: "correct@example.com"}, IsMismatched: false},
		{Commit: Commit{AuthorEmail: "wrong@example.com"}, IsMismatched: true},
	}

	result := GenerateMailmap(mismatches, "correct@example.com")
	if strings.Contains(result, "correct@example.com> <correct@example.com>") {
		t.Error("should not include non-mismatched email in mailmap")
	}
	if !strings.Contains(result, "<wrong@example.com>") {
		t.Error("should include mismatched email in mailmap")
	}
}

func TestGenerateMailmap_EmptyInput(t *testing.T) {
	result := GenerateMailmap([]Result{}, "correct@example.com")
	if result != "" {
		t.Errorf("expected empty string for no mismatches, got %q", result)
	}
}

func TestGenerateMailmap_AllNonMismatched(t *testing.T) {
	mismatches := []Result{
		{Commit: Commit{AuthorEmail: "a@example.com"}, IsMismatched: false},
		{Commit: Commit{AuthorEmail: "b@example.com"}, IsMismatched: false},
	}

	result := GenerateMailmap(mismatches, "correct@example.com")
	if result != "" {
		t.Errorf("expected empty string when no commits are mismatched, got %q", result)
	}
}

// initTempGitRepo creates a temp git repo and changes into it.
// Uses t.TempDir() for automatic cleanup and t.Chdir() for automatic cwd restore.
func initTempGitRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	cmd := exec.Command("git", "init", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	t.Chdir(tmpDir)
	return tmpDir
}

func TestGetRemotes_InRepoWithOrigin(t *testing.T) {
	tmpDir := initTempGitRepo(t)

	cmd := exec.Command("git", "-C", tmpDir, "remote", "add", "origin", "https://example.com/repo.git")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git remote add failed: %v", err)
	}

	remotes, err := GetRemotes()
	if err != nil {
		t.Fatalf("GetRemotes failed: %v", err)
	}

	if len(remotes) != 1 || remotes[0] != "origin" {
		t.Errorf("expected [origin], got %v", remotes)
	}
}

func TestGetRemotes_NoRemotes(t *testing.T) {
	initTempGitRepo(t)

	remotes, err := GetRemotes()
	if err != nil {
		t.Fatalf("GetRemotes failed: %v", err)
	}

	if len(remotes) != 0 {
		t.Errorf("expected no remotes, got %v", remotes)
	}
}

func TestRemoveRemotes_RemovesAll(t *testing.T) {
	tmpDir := initTempGitRepo(t)

	for _, name := range []string{"origin", "upstream"} {
		cmd := exec.Command("git", "-C", tmpDir, "remote", "add", name, "https://example.com/"+name+".git")
		if err := cmd.Run(); err != nil {
			t.Fatalf("git remote add %s failed: %v", name, err)
		}
	}

	if err := RemoveRemotes(); err != nil {
		t.Fatalf("RemoveRemotes failed: %v", err)
	}

	remotes, err := GetRemotes()
	if err != nil {
		t.Fatalf("GetRemotes after removal failed: %v", err)
	}

	if len(remotes) != 0 {
		t.Errorf("expected 0 remotes after removal, got %v", remotes)
	}
}

func TestRemoveRemotes_NoRemotes(t *testing.T) {
	initTempGitRepo(t)

	if err := RemoveRemotes(); err != nil {
		t.Fatalf("RemoveRemotes with no remotes should not error: %v", err)
	}
}
