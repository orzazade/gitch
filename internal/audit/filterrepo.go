package audit

import (
	"os/exec"
)

// IsFilterRepoAvailable checks if git-filter-repo is installed and accessible.
// Returns true if git-filter-repo is available, false otherwise.
func IsFilterRepoAvailable() bool {
	return exec.Command("git", "filter-repo", "--version").Run() == nil
}
