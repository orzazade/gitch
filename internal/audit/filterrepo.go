package audit

import (
	"os/exec"
)

// isFilterRepoAvailable checks if git-filter-repo is installed and accessible.
// Returns true if git-filter-repo is available, false otherwise.
func isFilterRepoAvailable() bool {
	return exec.Command("git", "filter-repo", "--version").Run() == nil
}
