package audit

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

// Delimiters for parsing git log output
const (
	fieldDelim  = "|||"
	commitDelim = "<<<COMMIT>>>"
)

// Commit represents a single git commit with metadata
type Commit struct {
	Hash        string
	AuthorName  string
	AuthorEmail string
	Date        time.Time
	Subject     string
}

// Result represents an audited commit with mismatch status
type Result struct {
	Commit        Commit
	ExpectedEmail string
	IsMismatched  bool
	IsPushed      bool // true = pushed to remote, false = local-only
}

// GetCommits retrieves commits from git log
// If limit > 0, limits the number of commits returned
// Returns empty slice with nil error for empty repos
func GetCommits(limit int) ([]Commit, error) {
	// Build git log command with custom format
	// Format: <<<COMMIT>>>hash|||name|||email|||date|||subject
	formatArg := fmt.Sprintf("--format=%s%%H%s%%an%s%%ae%s%%ai%s%%s",
		commitDelim, fieldDelim, fieldDelim, fieldDelim, fieldDelim)

	args := []string{"log", formatArg}
	if limit > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", limit))
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		// Check for empty repo or no commits
		errStr := string(output)
		if strings.Contains(errStr, "fatal: your current branch") ||
			strings.Contains(errStr, "does not have any commits") {
			return []Commit{}, nil
		}
		// Also check exit error message
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "fatal: your current branch") ||
				strings.Contains(stderr, "does not have any commits") {
				return []Commit{}, nil
			}
		}
		return nil, fmt.Errorf("failed to run git log: %w", err)
	}

	return parseCommits(string(output))
}

// parseCommits parses the git log output into Commit structs
func parseCommits(output string) ([]Commit, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return []Commit{}, nil
	}

	// Split by commit delimiter
	parts := strings.Split(output, commitDelim)

	var commits []Commit
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		commit, err := parseCommitLine(part)
		if err != nil {
			// Skip malformed commits instead of failing entirely
			continue
		}
		commits = append(commits, commit)
	}

	return commits, nil
}

// parseCommitLine parses a single commit line into a Commit struct
func parseCommitLine(line string) (Commit, error) {
	parts := strings.Split(line, fieldDelim)
	if len(parts) < 5 {
		return Commit{}, fmt.Errorf("malformed commit line: expected 5 fields, got %d", len(parts))
	}

	// Parse the date
	dateStr := strings.TrimSpace(parts[3])
	date, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
	if err != nil {
		return Commit{}, fmt.Errorf("failed to parse date %q: %w", dateStr, err)
	}

	return Commit{
		Hash:        strings.TrimSpace(parts[0]),
		AuthorName:  strings.TrimSpace(parts[1]),
		AuthorEmail: strings.TrimSpace(parts[2]),
		Date:        date,
		Subject:     strings.TrimSpace(parts[4]),
	}, nil
}

// GetLocalOnlyHashes returns a map of commit hashes that exist locally but not on the upstream
// Returns nil, nil if no upstream is configured (cannot determine pushed status)
// Returns empty map if all commits are pushed
func GetLocalOnlyHashes() (map[string]bool, error) {
	// Get upstream ref
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "@{u}")
	upstreamOutput, err := cmd.Output()
	if err != nil {
		// No upstream configured - can't determine pushed status
		return nil, nil
	}

	upstream := strings.TrimSpace(string(upstreamOutput))
	if upstream == "" {
		return nil, nil
	}

	// Get local-only commits (commits in HEAD but not in upstream)
	rangeArg := fmt.Sprintf("%s..HEAD", upstream)
	cmd = exec.Command("git", "log", rangeArg, "--format=%H")
	output, err := cmd.Output()
	if err != nil {
		// If this fails, assume we can't determine status
		return nil, nil
	}

	// Build map of local-only hashes
	localHashes := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		hash := strings.TrimSpace(line)
		if hash != "" {
			localHashes[hash] = true
		}
	}

	return localHashes, nil
}

// ScanOptions configures the Scan function behavior
type ScanOptions struct {
	Limit   int  // Max commits to scan (0 = default 1000)
	ShowAll bool // Include matching commits in results
}

// ScanResult contains the results of an audit scan
type ScanResult struct {
	Results        []Result
	ExpectedEmail  string
	MatchedRule    *rules.Rule
	TotalScanned   int
	MismatchCount  int
	LocalOnlyCount int
	PushedCount    int
	NoUpstream     bool // true if we couldn't determine pushed status
}

// Scan performs an identity audit on the git history
// It compares commit author emails against the expected identity for this repo
func Scan(opts ScanOptions) (*ScanResult, error) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Get remote URL (may be empty)
	remoteURL, _ := rules.GetGitRemoteURL()

	// Find best matching rule
	matchedRule := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)

	// If no rule matches, return empty result (nothing to audit against)
	if matchedRule == nil {
		return &ScanResult{
			Results: []Result{},
		}, nil
	}

	// Get expected identity
	expectedIdentity, err := cfg.GetIdentity(matchedRule.Identity)
	if err != nil {
		return nil, fmt.Errorf("rule references unknown identity %q: %w", matchedRule.Identity, err)
	}

	// Default limit
	limit := opts.Limit
	if limit == 0 {
		limit = 1000
	}

	// Get commits
	commits, err := GetCommits(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	// Get local-only hashes
	localHashes, _ := GetLocalOnlyHashes()
	noUpstream := localHashes == nil

	// Process commits
	var results []Result
	var mismatchCount, localOnlyCount, pushedCount int

	for _, commit := range commits {
		// Determine if pushed
		var isPushed bool
		if noUpstream {
			// Can't determine - assume pushed (conservative)
			isPushed = true
		} else {
			// Not in localHashes means it's pushed
			isPushed = !localHashes[commit.Hash]
		}

		// Count pushed vs local
		if isPushed {
			pushedCount++
		} else {
			localOnlyCount++
		}

		// Check for mismatch (case-insensitive email comparison)
		isMismatched := !strings.EqualFold(commit.AuthorEmail, expectedIdentity.Email)
		if isMismatched {
			mismatchCount++
		}

		// Include in results if mismatch or ShowAll
		if isMismatched || opts.ShowAll {
			results = append(results, Result{
				Commit:        commit,
				ExpectedEmail: expectedIdentity.Email,
				IsMismatched:  isMismatched,
				IsPushed:      isPushed,
			})
		}
	}

	return &ScanResult{
		Results:        results,
		ExpectedEmail:  expectedIdentity.Email,
		MatchedRule:    matchedRule,
		TotalScanned:   len(commits),
		MismatchCount:  mismatchCount,
		LocalOnlyCount: localOnlyCount,
		PushedCount:    pushedCount,
		NoUpstream:     noUpstream,
	}, nil
}

// IsGitRepo checks if the current directory is inside a git repository
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}
