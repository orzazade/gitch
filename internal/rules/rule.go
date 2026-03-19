package rules

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// RuleType indicates whether a rule matches by directory or remote
type RuleType string

const (
	// DirectoryRule matches based on working directory path
	DirectoryRule RuleType = "directory"
	// RemoteRule matches based on git remote URL
	RemoteRule RuleType = "remote"
)

// Rule represents an auto-switch rule that maps a pattern to an identity
type Rule struct {
	Type     RuleType `yaml:"type"`
	Pattern  string   `yaml:"pattern"`
	Identity string   `yaml:"identity"`
}

// ValidatePattern validates the rule pattern
// For directory rules, it expands tilde and validates with doublestar
// For remote rules, it validates the pattern format
func (r Rule) ValidatePattern() error {
	if r.Pattern == "" {
		return errors.New("pattern cannot be empty")
	}

	switch r.Type {
	case DirectoryRule:
		return validateDirectoryPattern(r.Pattern)
	case RemoteRule:
		return validateRemotePattern(r.Pattern)
	default:
		return fmt.Errorf("unknown rule type: %s", r.Type)
	}
}

// validateDirectoryPattern validates a directory glob pattern
func validateDirectoryPattern(pattern string) error {
	// Expand tilde for validation
	expanded := expandTilde(pattern)

	// Validate the pattern with doublestar
	if !doublestar.ValidatePathPattern(expanded) {
		return fmt.Errorf("invalid glob pattern: %s", pattern)
	}

	return nil
}

// validateRemotePattern validates a remote URL pattern
func validateRemotePattern(pattern string) error {
	// Remote patterns should be in format: host/org/* or host/org/repo
	// They should contain at least host and one path segment
	parts := strings.Split(pattern, "/")
	if len(parts) < 2 {
		return fmt.Errorf("remote pattern must be in format: host/org/* or host/org/repo, got: %s", pattern)
	}

	// First part should be a hostname
	if parts[0] == "" || strings.ContainsAny(parts[0], " \t\n") {
		return fmt.Errorf("invalid host in remote pattern: %s", pattern)
	}

	// Validate glob syntax so malformed patterns (e.g. unclosed '[') are
	// caught at rule creation time rather than silently failing to match.
	if strings.ContainsAny(pattern, "*?[") {
		if _, err := filepath.Match(pattern, ""); err != nil {
			return fmt.Errorf("invalid glob syntax in remote pattern: %w", err)
		}
	}

	return nil
}

// expandTilde expands ~ to the user's home directory
func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	// Handle ~/path format
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}

	// Handle ~user format (not supported, return as-is)
	return path
}
