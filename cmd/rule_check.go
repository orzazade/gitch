package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var ruleCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate all rules for problems",
	Long: `Check all configured rules for common problems:

- Rules that reference identities not in your config
- Directory rules whose base path does not exist
- Overlapping rules that assign different identities to the same directories

Exit code 0 if no problems found, 1 if issues detected.

Examples:
  gitch rule check`,
	Args: cobra.NoArgs,
	RunE: runRuleCheck,
}

func init() {
	ruleCmd.AddCommand(ruleCheckCmd)
}

// ruleIssue describes a single problem found with a rule.
type ruleIssue struct {
	Rule    rules.Rule
	Problem string
	Fix     string
}

func runRuleCheck(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	allRules := cfg.ListRules()
	if len(allRules) == 0 {
		fmt.Println("No rules configured. Use 'gitch rule add' to create one.")
		return nil
	}

	var issues []ruleIssue

	// Check each rule individually.
	for _, rule := range allRules {
		issues = checkIdentityExists(cfg, rule, issues)
		if rule.Type == rules.DirectoryRule {
			issues = checkDirectoryExists(rule, issues)
		}
	}

	// Check for overlapping rules with different identities.
	issues = checkConflicts(allRules, issues)

	// Report results.
	if len(issues) == 0 {
		fmt.Printf("%s All %d rule(s) look good.\n",
			ui.SuccessStyle.Render("OK"), len(allRules))
		return nil
	}

	fmt.Printf("Found %d issue(s) across %d rule(s):\n\n", len(issues), len(allRules))
	for i, issue := range issues {
		fmt.Printf("%s %s rule %q: %s\n",
			ui.WarningStyle.Render(fmt.Sprintf("[%d]", i+1)),
			issue.Rule.Type,
			issue.Rule.Pattern,
			issue.Problem)
		if issue.Fix != "" {
			fmt.Printf("    Fix: %s\n", issue.Fix)
		}
	}

	return fmt.Errorf("%d rule issue(s) found", len(issues))
}

// checkIdentityExists verifies the rule's identity is defined in config.
func checkIdentityExists(cfg *config.Config, rule rules.Rule, issues []ruleIssue) []ruleIssue {
	if _, err := cfg.GetIdentity(rule.Identity); err != nil {
		suggestion := closestIdentityName(rule.Identity, cfg.Identities)
		fix := "gitch rule remove " + fmt.Sprintf("%q", rule.Pattern)
		if suggestion != "" {
			fix = fmt.Sprintf("Did you mean %q? Remove and re-add, or rename the identity", suggestion)
		}
		issues = append(issues, ruleIssue{
			Rule:    rule,
			Problem: fmt.Sprintf("references unknown identity %q", rule.Identity),
			Fix:     fix,
		})
	}
	return issues
}

// checkDirectoryExists verifies the base directory of a directory rule pattern exists.
func checkDirectoryExists(rule rules.Rule, issues []ruleIssue) []ruleIssue {
	basePath := directoryRuleBasePath(rule.Pattern)
	if basePath == "" {
		return issues
	}

	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		issues = append(issues, ruleIssue{
			Rule:    rule,
			Problem: fmt.Sprintf("base directory %q does not exist", basePath),
			Fix:     fmt.Sprintf("gitch rule remove %q", rule.Pattern),
		})
	}
	return issues
}

// directoryRuleBasePath extracts the concrete base directory from a glob pattern.
// Returns "" if the pattern has no static prefix (e.g. "**/foo").
func directoryRuleBasePath(pattern string) string {
	expanded := expandPatternTilde(pattern)
	cleaned := filepath.Clean(expanded)

	// Walk path segments until we hit a glob metacharacter.
	parts := strings.Split(cleaned, string(filepath.Separator))
	var staticParts []string
	for _, part := range parts {
		if strings.ContainsAny(part, "*?[") {
			break
		}
		staticParts = append(staticParts, part)
	}

	if len(staticParts) == 0 {
		return ""
	}

	base := strings.Join(staticParts, string(filepath.Separator))
	// Restore leading separator for absolute paths.
	if filepath.IsAbs(cleaned) && !filepath.IsAbs(base) {
		base = string(filepath.Separator) + base
	}
	return base
}

// expandPatternTilde expands ~ to the home directory for validation purposes.
func expandPatternTilde(pattern string) string {
	if !strings.HasPrefix(pattern, "~/") && pattern != "~" {
		return pattern
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return pattern
	}
	if pattern == "~" {
		return home
	}
	return filepath.Join(home, pattern[2:])
}

// checkConflicts finds overlapping rules that assign different identities.
func checkConflicts(allRules []rules.Rule, issues []ruleIssue) []ruleIssue {
	// Track reported pairs to avoid duplicates.
	type pair struct{ a, b int }
	reported := make(map[pair]bool)

	for i, r1 := range allRules {
		for j := i + 1; j < len(allRules); j++ {
			r2 := allRules[j]
			if r1.Type != r2.Type {
				continue
			}
			if strings.EqualFold(r1.Identity, r2.Identity) {
				continue // same identity — no conflict
			}
			if reported[pair{i, j}] {
				continue
			}

			overlaps := false
			switch r1.Type {
			case rules.DirectoryRule:
				overlaps = directoryPatternsOverlap(r1.Pattern, r2.Pattern)
			case rules.RemoteRule:
				overlaps = remotePatternsOverlap(r1.Pattern, r2.Pattern)
			}

			if overlaps {
				reported[pair{i, j}] = true
				issues = append(issues, ruleIssue{
					Rule: r1,
					Problem: fmt.Sprintf("overlaps with %q (identity %q vs %q) — the more specific rule wins at runtime",
						r2.Pattern, r1.Identity, r2.Identity),
					Fix: "Ensure the overlap is intentional, or remove one rule",
				})
			}
		}
	}
	return issues
}

// directoryPatternsOverlap checks if two directory patterns could match the same path.
func directoryPatternsOverlap(p1, p2 string) bool {
	b1 := directoryRuleBasePath(p1)
	b2 := directoryRuleBasePath(p2)
	if b1 == "" || b2 == "" {
		return false
	}
	return strings.HasPrefix(b1, b2) || strings.HasPrefix(b2, b1)
}

// remotePatternsOverlap checks if two remote patterns could match the same remote.
func remotePatternsOverlap(p1, p2 string) bool {
	host1, path1, hasPath1 := strings.Cut(p1, "/")
	host2, path2, hasPath2 := strings.Cut(p2, "/")

	if !strings.EqualFold(host1, host2) {
		return false
	}
	if !hasPath1 || !hasPath2 {
		return true
	}

	path1 = strings.TrimSuffix(path1, "/*")
	path2 = strings.TrimSuffix(path2, "/*")

	return strings.HasPrefix(path1, path2) || strings.HasPrefix(path2, path1)
}
