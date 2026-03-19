package rules

import (
	"strings"

	giturls "github.com/whilp/git-urls"

	"github.com/orzazade/gitch/internal/git"
)

// ParsedRemote represents a parsed git remote URL with host, org, and repo components.
type ParsedRemote struct {
	Host string // e.g., "github.com"
	Org  string // e.g., "company"
	Repo string // e.g., "project"
}

// ParseRemote parses a git remote URL and extracts host, org, and repo.
// Supports SSH (git@host:path), HTTPS, and SCP-style URLs.
func ParseRemote(rawURL string) (*ParsedRemote, error) {
	u, err := giturls.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	// Split path into org and repo
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), ".git"), "/")

	result := &ParsedRemote{
		Host: strings.ToLower(u.Host),
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
	return git.GetConfigScoped("remote.origin.url", git.ScopeEffective)
}
