package rules

import (
	"path/filepath"
	"strings"
)

// Specificity calculates the specificity score for a rule
// Higher scores indicate more specific rules
// Directory rules: count path segments (*10), penalize wildcards (*-2)
// Remote rules: count parts (*10), exact repo match bonus (+50)
func (r Rule) Specificity() int {
	switch r.Type {
	case DirectoryRule:
		return directorySpecificity(r.Pattern)
	case RemoteRule:
		return remoteSpecificity(r.Pattern)
	default:
		return 0
	}
}

// directorySpecificity calculates specificity for directory patterns
func directorySpecificity(pattern string) int {
	// Expand and clean the pattern
	expanded := expandTilde(pattern)
	expanded = filepath.Clean(expanded)

	// Count path segments
	segments := strings.Split(expanded, string(filepath.Separator))
	score := len(segments) * 10

	// Penalize wildcards
	wildcardCount := strings.Count(pattern, "*")
	score -= wildcardCount * 2

	// Double star is more general, penalize more
	doubleStarCount := strings.Count(pattern, "**")
	score -= doubleStarCount * 3

	return score
}

// remoteSpecificity calculates specificity for remote patterns
func remoteSpecificity(pattern string) int {
	// Count path parts
	parts := strings.Split(pattern, "/")
	score := len(parts) * 10

	// Check for wildcards
	hasWildcard := strings.Contains(pattern, "*")
	if !hasWildcard {
		// Exact repo match bonus
		score += 50
	} else {
		// Penalize wildcards
		wildcardCount := strings.Count(pattern, "*")
		score -= wildcardCount * 2
	}

	return score
}

// Matches checks if a rule matches the given context
func (r Rule) Matches(cwd, remoteURL string) bool {
	switch r.Type {
	case DirectoryRule:
		matched, err := MatchDirectory(r.Pattern, cwd)
		if err != nil {
			return false
		}
		return matched
	case RemoteRule:
		if remoteURL == "" {
			return false
		}
		parsed, err := ParseRemote(remoteURL)
		if err != nil {
			return false
		}
		return MatchRemote(r.Pattern, parsed)
	default:
		return false
	}
}

// FindBestMatch finds the rule with the highest specificity that matches the context
// Returns nil if no rules match
func FindBestMatch(rules []Rule, cwd, remoteURL string) *Rule {
	var bestMatch *Rule
	bestScore := -1

	for i := range rules {
		rule := &rules[i]
		if !rule.Matches(cwd, remoteURL) {
			continue
		}

		score := rule.Specificity()
		if score > bestScore {
			bestScore = score
			bestMatch = rule
		}
	}

	return bestMatch
}
