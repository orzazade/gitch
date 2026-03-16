package audit

import (
	"os/exec"
)

// IsFilterRepoAvailable checks if git-filter-repo is installed and accessible.
// Returns true if git-filter-repo is available, false otherwise.
func IsFilterRepoAvailable() bool {
	cmd := exec.Command("git", "filter-repo", "--version")
	err := cmd.Run()
	return err == nil
}
