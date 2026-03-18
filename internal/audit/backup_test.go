package audit

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func Test_createMirrorBackup_Success(t *testing.T) {
	// Create a temp git repo with at least one commit
	repoDir := t.TempDir()
	t.Chdir(repoDir)

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	backupDir := filepath.Join(t.TempDir(), "backup.git")

	if err := createMirrorBackup(backupDir); err != nil {
		t.Fatalf("createMirrorBackup failed: %v", err)
	}

	// Verify backup directory exists and is a git repo
	if _, err := os.Stat(backupDir); errors.Is(err, os.ErrNotExist) {
		t.Fatal("backup directory was not created")
	}

	// Check it's a bare repo (mirror clones are bare)
	cmd := exec.Command("git", "-C", backupDir, "rev-parse", "--is-bare-repository")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to check bare repo: %v", err)
	}
	if string(output) != "true\n" {
		t.Errorf("expected bare repository, got %q", string(output))
	}
}

func Test_createMirrorBackup_NotInRepo(t *testing.T) {
	t.Chdir(t.TempDir())

	err := createMirrorBackup(filepath.Join(t.TempDir(), "backup.git"))
	if err == nil {
		t.Fatal("expected error when not in a git repo")
	}
}
