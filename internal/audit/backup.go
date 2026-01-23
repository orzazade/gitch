package audit

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreateMirrorBackup creates a full mirror backup of the current git repository.
// The destPath should be an absolute path where the mirror will be created.
// Uses --no-local to avoid hardlink issues that could cause data loss.
// Returns error if not in a git repository or if backup fails.
func CreateMirrorBackup(destPath string) error {
	// Get git repo root
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("not in a git repository")
	}

	repoRoot := strings.TrimSpace(string(output))
	if repoRoot == "" {
		return fmt.Errorf("not in a git repository")
	}

	// Create mirror backup
	// CRITICAL: Use --no-local to avoid hardlink issues (Pitfall 4 from research)
	// Hardlinks would cause changes to backup to affect original repo
	cmd = exec.Command("git", "clone", "--mirror", "--no-local", repoRoot, destPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mirror backup failed: %w", err)
	}

	return nil
}
