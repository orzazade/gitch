package git

import (
	"os/exec"
	"strings"

	giturls "github.com/whilp/git-urls"
)

// IsAzureDevOpsRemote checks if the given remote URL is an Azure DevOps repository.
// Returns true for both modern (dev.azure.com) and legacy (visualstudio.com) URLs.
// Supports HTTPS, SSH, and SCP-style URL formats.
func IsAzureDevOpsRemote(remoteURL string) bool {
	if remoteURL == "" {
		return false
	}

	// Parse the URL
	u, err := giturls.Parse(remoteURL)
	if err != nil {
		return false
	}

	// Normalize host to lowercase for comparison
	host := strings.ToLower(u.Host)

	// Check for Azure DevOps patterns
	// Modern: dev.azure.com, ssh.dev.azure.com
	// Legacy: *.visualstudio.com, vs-ssh.visualstudio.com
	return strings.Contains(host, "dev.azure.com") ||
		strings.Contains(host, "visualstudio.com")
}

// GetCurrentRemoteType detects if the current git repository's origin remote
// is an Azure DevOps repository.
// Returns (true, nil) if Azure DevOps is detected.
// Returns (false, nil) if not Azure DevOps or no origin remote exists.
// Returns (false, error) only if the git command fails for other reasons.
func GetCurrentRemoteType() (isAzureDevOps bool, err error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		// If the command fails (e.g., no origin remote), return false without error
		// This handles: not in a git repo, no origin remote, etc.
		return false, nil
	}

	remoteURL := strings.TrimSpace(string(output))
	if remoteURL == "" {
		return false, nil
	}

	return IsAzureDevOpsRemote(remoteURL), nil
}
