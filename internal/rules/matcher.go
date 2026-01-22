package rules

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// MatchDirectory checks if the current working directory matches the given pattern
// Pattern should be a glob pattern, optionally starting with ~ for home directory
func MatchDirectory(pattern, cwd string) (bool, error) {
	// Expand tilde in both pattern and cwd
	expandedPattern := expandTilde(pattern)
	expandedCwd := expandTilde(cwd)

	// Clean paths for consistent matching
	expandedPattern = filepath.Clean(expandedPattern)
	expandedCwd = filepath.Clean(expandedCwd)

	// Use doublestar.PathMatch for OS-native path separators
	match, err := doublestar.PathMatch(expandedPattern, expandedCwd)
	if err != nil {
		return false, err
	}

	return match, nil
}

// MatchRemote checks if a parsed remote matches the given pattern
// Pattern format: "host/org/*" or "host/org/repo"
func MatchRemote(pattern string, remote *ParsedRemote) bool {
	if remote == nil {
		return false
	}

	// Build the remote path for matching: host/org/repo
	remotePath := remote.Host
	if remote.Org != "" {
		remotePath += "/" + remote.Org
	}
	if remote.Repo != "" {
		remotePath += "/" + remote.Repo
	}

	// Normalize both for comparison (lowercase)
	pattern = strings.ToLower(pattern)
	remotePath = strings.ToLower(remotePath)

	// Check if pattern contains wildcard
	if strings.Contains(pattern, "*") {
		// Use simple glob matching
		matched, _ := filepath.Match(pattern, remotePath)
		return matched
	}

	// Exact match (with potential partial path match)
	// Pattern "github.com/org" should match remote "github.com/org/repo"
	if strings.HasPrefix(remotePath, pattern) {
		// Ensure we match at a path boundary
		if len(remotePath) == len(pattern) {
			return true
		}
		if len(remotePath) > len(pattern) && remotePath[len(pattern)] == '/' {
			return true
		}
	}

	return pattern == remotePath
}
