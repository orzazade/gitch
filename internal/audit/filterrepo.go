package audit

import (
	"os/exec"
	"strings"
)

// IsFilterRepoAvailable checks if git-filter-repo is installed and accessible.
// Returns true if git-filter-repo is available, false otherwise.
func IsFilterRepoAvailable() bool {
	cmd := exec.Command("git", "filter-repo", "--version")
	err := cmd.Run()
	return err == nil
}

// GetFilterRepoVersion returns the version string of git-filter-repo.
// Returns error if git-filter-repo is not installed or version check fails.
func GetFilterRepoVersion() (string, error) {
	cmd := exec.Command("git", "filter-repo", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Output format: "git-filter-repo X.Y.Z"
	return strings.TrimSpace(string(output)), nil
}
