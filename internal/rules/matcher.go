package rules

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// matchDirectory checks if the current working directory matches the given pattern.
// Pattern should be a glob pattern, optionally starting with ~ for home directory.
func matchDirectory(pattern, cwd string) (bool, error) {
	expandedPattern := normalizeDirectoryPattern(pattern)
	expandedCwd := normalizeDirectoryPath(cwd)

	// Use doublestar.PathMatch for OS-native path separators
	return doublestar.PathMatch(expandedPattern, expandedCwd)
}

func normalizeDirectoryPattern(pattern string) string {
	expanded := filepath.Clean(expandTilde(pattern))
	if expanded == "." {
		return expanded
	}

	absPattern, err := filepath.Abs(expanded)
	if err == nil {
		expanded = absPattern
	}

	prefix, suffix := splitStaticPrefix(expanded)
	if prefix == "" {
		return expanded
	}

	resolvedPrefix, err := filepath.EvalSymlinks(prefix)
	if err != nil {
		return expanded
	}

	if suffix == "" {
		return resolvedPrefix
	}

	return filepath.Join(resolvedPrefix, suffix)
}

func normalizeDirectoryPath(path string) string {
	expanded := filepath.Clean(expandTilde(path))
	if expanded == "." {
		return expanded
	}

	absPath, err := filepath.Abs(expanded)
	if err == nil {
		expanded = absPath
	}

	resolved, err := filepath.EvalSymlinks(expanded)
	if err == nil {
		return resolved
	}

	return expanded
}

func splitStaticPrefix(pattern string) (prefix string, suffix string) {
	cleaned := filepath.Clean(pattern)
	volume := filepath.VolumeName(cleaned)
	trimmed := strings.TrimPrefix(cleaned, volume)
	isAbs := filepath.IsAbs(cleaned)

	sep := string(filepath.Separator)
	trimmed = strings.TrimPrefix(trimmed, sep)
	if trimmed == "" {
		return cleaned, ""
	}

	parts := strings.Split(trimmed, sep)
	staticParts := make([]string, 0, len(parts))
	for idx, part := range parts {
		if hasGlobMeta(part) {
			prefix = joinPatternPrefix(volume, isAbs, staticParts)
			suffix = filepath.Join(parts[idx:]...)
			return prefix, suffix
		}
		staticParts = append(staticParts, part)
	}

	return joinPatternPrefix(volume, isAbs, staticParts), ""
}

func joinPatternPrefix(volume string, isAbs bool, parts []string) string {
	sep := string(filepath.Separator)
	switch {
	case len(parts) == 0 && volume != "" && isAbs:
		return volume + sep
	case len(parts) == 0 && volume != "":
		return volume
	case len(parts) == 0 && isAbs:
		return sep
	case len(parts) == 0:
		return ""
	}

	prefix := filepath.Join(parts...)
	if volume != "" {
		prefix = filepath.Join(volume+sep, prefix)
	} else if isAbs {
		prefix = filepath.Join(sep, prefix)
	}

	return prefix
}

func hasGlobMeta(part string) bool {
	return strings.ContainsAny(part, "*?[")
}

// matchRemote checks if a parsed remote matches the given pattern.
// Pattern format: "host/org/*" or "host/org/repo"
func matchRemote(pattern string, remote *parsedRemote) bool {
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
	return strings.HasPrefix(remotePath, pattern) &&
		(len(remotePath) == len(pattern) || remotePath[len(pattern)] == '/')
}
