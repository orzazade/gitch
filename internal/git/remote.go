package git

import (
	"strings"

	giturls "github.com/whilp/git-urls"
)

// isAzureDevOpsRemote checks if the given remote URL is an Azure DevOps repository.
// Returns true for both modern (dev.azure.com) and legacy (visualstudio.com) URLs.
// Supports HTTPS, SSH, and SCP-style URL formats.
func isAzureDevOpsRemote(remoteURL string) bool {
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

// IsAzureDevOps reports whether the current git repository's origin remote
// is an Azure DevOps repository.
// Returns false when not in a git repo or no origin remote exists.
func IsAzureDevOps() bool {
	url, err := GetConfigScoped("remote.origin.url", ScopeEffective)
	if err != nil {
		return false
	}

	return isAzureDevOpsRemote(url)
}
