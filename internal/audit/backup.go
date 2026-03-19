package audit

import (
	"fmt"
	"os/exec"

	"github.com/orzazade/gitch/internal/git"
)

// createMirrorBackup creates a full mirror backup of the current git repository.
// The destPath should be an absolute path where the mirror will be created.
// Uses --no-local to avoid hardlink issues that could cause data loss.
// Returns error if not in a git repository or if backup fails.
func createMirrorBackup(destPath string) error {
	root, err := git.Toplevel()
	if err != nil {
		return err
	}

	// CRITICAL: Use --no-local to avoid hardlink issues (Pitfall 4 from research)
	// Hardlinks would cause changes to backup to affect original repo
	cmd := exec.Command("git", "clone", "--mirror", "--no-local", root, destPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mirror backup failed: %w", err)
	}

	return nil
}
