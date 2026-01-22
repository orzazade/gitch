package rules

import (
	"os/exec"
	"strings"

	giturls "github.com/whilp/git-urls"
)

// ParsedRemote represents a parsed git remote URL
type ParsedRemote struct {
	Host string // e.g., "github.com"
	Org  string // e.g., "company"
	Repo string // e.g., "project"
}

// ParseRemote parses a git remote URL and extracts host, org, and repo
// Supports SSH (git@host:path), HTTPS, and SCP-style URLs
func ParseRemote(rawURL string) (*ParsedRemote, error) {
	u, err := giturls.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	// Normalize host to lowercase
	host := strings.ToLower(u.Host)

	// Get path and clean it
	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	// Split path into org and repo
	parts := strings.Split(path, "/")

	result := &ParsedRemote{
		Host: host,
	}

	if len(parts) >= 1 && parts[0] != "" {
		result.Org = parts[0]
	}
	if len(parts) >= 2 && parts[1] != "" {
		result.Repo = parts[1]
	}

	return result, nil
}

// GetGitRemoteURL retrieves the origin remote URL from the current git repository
func GetGitRemoteURL() (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
