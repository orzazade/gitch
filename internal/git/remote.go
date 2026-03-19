package git

import (
	"os/exec"
	"strings"

	giturls "github.com/whilp/git-urls"
)

// Remote represents a named git remote with its fetch URL.
type Remote struct {
	Name string
	URL  string
}

// Remotes returns all configured remotes for the current git repository.
// Each remote appears once (using its fetch URL).
func Remotes() ([]Remote, error) {
	out, err := exec.Command("git", "remote", "-v").Output()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var remotes []Remote

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		url := fields[1]

		// git remote -v shows each remote twice (fetch/push); take only one.
		if seen[name] {
			continue
		}
		seen[name] = true
		remotes = append(remotes, Remote{Name: name, URL: url})
	}

	return remotes, nil
}

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
