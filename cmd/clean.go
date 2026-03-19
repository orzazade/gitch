package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cleanJSON bool
	cleanFix  bool
)

var errCleanIssuesFound = errors.New("issues found in gitch configuration")

type cleanIssue struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
	Fixed  bool   `json:"fixed,omitempty"`
}

type cleanOutput struct {
	OrphanedRules    []cleanIssue `json:"orphaned_rules"`
	UnusedIdentities []cleanIssue `json:"unused_identities"`
	OrphanedRuleRefs []cleanIssue `json:"orphaned_rule_refs"`
	TotalIssues      int          `json:"total_issues"`
	FixedCount       int          `json:"fixed_count,omitempty"`
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Find orphaned rules and unused identities",
	Long: `Scan gitch configuration for stale or orphaned entries:

  - Orphaned directory rules: patterns pointing to directories that no longer exist
  - Orphaned rule references: rules that reference identities that don't exist
  - Unused identities: identities not referenced by any rule, not set as default,
    and not currently active in git config

Use --fix to automatically remove orphaned rules. Unused identities are only
reported (use 'gitch delete' to remove them manually).

Exit code 0 means no issues. Exit code 1 means issues were found.

Examples:
  gitch clean
  gitch clean --fix
  gitch clean --json`,
	Args:          cobra.NoArgs,
	RunE:          runClean,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVar(&cleanJSON, "json", false, "Output in JSON format")
	cleanCmd.Flags().BoolVar(&cleanFix, "fix", false, "Remove orphaned rules")
}

func runClean(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	out := findCleanIssues(cfg)

	if cleanFix && len(out.OrphanedRules)+len(out.OrphanedRuleRefs) > 0 {
		fixCleanIssues(cfg, &out)
	}

	if cleanJSON {
		if err := printJSON(out); err != nil {
			return err
		}
		if out.TotalIssues > 0 && (!cleanFix || out.TotalIssues > out.FixedCount) {
			return errCleanIssuesFound
		}
		return nil
	}

	return printCleanHuman(out)
}

func findCleanIssues(cfg *config.Config) cleanOutput {
	var out cleanOutput

	identityNames := make(map[string]bool)
	for _, id := range cfg.ListIdentities() {
		identityNames[strings.ToLower(id.Name)] = true
	}

	// Check for orphaned directory rules and orphaned rule references
	for _, r := range cfg.ListRules() {
		if !identityNames[strings.ToLower(r.Identity)] {
			out.OrphanedRuleRefs = append(out.OrphanedRuleRefs, cleanIssue{
				Type:   "orphaned_rule_ref",
				Name:   r.Pattern,
				Detail: "references non-existent identity " + r.Identity,
			})
			continue
		}

		if r.Type != rules.DirectoryRule {
			continue
		}

		expanded := expandDirPattern(r.Pattern)
		if expanded == "" {
			continue
		}

		// Skip glob patterns — we can only validate concrete paths
		if strings.ContainsAny(expanded, "*?[") {
			continue
		}

		if _, err := os.Stat(expanded); errors.Is(err, os.ErrNotExist) {
			out.OrphanedRules = append(out.OrphanedRules, cleanIssue{
				Type:   "orphaned_rule",
				Name:   r.Pattern,
				Detail: "directory does not exist",
			})
		}
	}

	// Check for unused identities
	referencedIdentities := make(map[string]bool)
	for _, r := range cfg.ListRules() {
		referencedIdentities[strings.ToLower(r.Identity)] = true
	}
	if cfg.Default != "" {
		referencedIdentities[strings.ToLower(cfg.Default)] = true
	}

	// Check currently active identity
	activeIdentity := resolveActiveIdentityName(cfg)
	if activeIdentity != "" {
		referencedIdentities[strings.ToLower(activeIdentity)] = true
	}

	for _, id := range cfg.ListIdentities() {
		if !referencedIdentities[strings.ToLower(id.Name)] {
			out.UnusedIdentities = append(out.UnusedIdentities, cleanIssue{
				Type:   "unused_identity",
				Name:   id.Name,
				Detail: id.Email,
			})
		}
	}

	out.TotalIssues = len(out.OrphanedRules) + len(out.UnusedIdentities) + len(out.OrphanedRuleRefs)
	return out
}

func fixCleanIssues(cfg *config.Config, out *cleanOutput) {
	var fixed int
	for i := range out.OrphanedRules {
		if err := cfg.RemoveRule(out.OrphanedRules[i].Name); err == nil {
			out.OrphanedRules[i].Fixed = true
			fixed++
		}
	}
	for i := range out.OrphanedRuleRefs {
		if err := cfg.RemoveRule(out.OrphanedRuleRefs[i].Name); err == nil {
			out.OrphanedRuleRefs[i].Fixed = true
			fixed++
		}
	}

	if fixed > 0 {
		_ = cfg.Save()
	}
	out.FixedCount = fixed
}

func expandDirPattern(pattern string) string {
	if strings.HasPrefix(pattern, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, pattern[2:])
	}
	if pattern == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return home
	}
	return pattern
}

func resolveActiveIdentityName(cfg *config.Config) string {
	state, err := resolveCurrentProfileState(cfg)
	if err != nil {
		return ""
	}
	if state.ExactMatch != nil {
		return state.ExactMatch.Name
	}
	if state.EmailMatch != nil {
		return state.EmailMatch.Name
	}
	return ""
}

func printCleanHuman(out cleanOutput) error {
	home, _ := os.UserHomeDir()
	fmt.Println(ui.DimStyle.Render("gitch clean"))
	fmt.Println()

	if out.TotalIssues == 0 {
		fmt.Println(ui.SuccessStyle.Render("Configuration is clean — no issues found."))
		return nil
	}

	if len(out.OrphanedRules) > 0 {
		fmt.Println("  Orphaned directory rules (directory does not exist):")
		fmt.Println()
		for _, issue := range out.OrphanedRules {
			displayName := shortenHome(issue.Name, home)
			if issue.Fixed {
				fmt.Printf("    %s  %s\n", ui.SuccessStyle.Render("removed"), displayName)
			} else {
				fmt.Printf("    %s  %s\n", ui.WarningStyle.Render("orphaned"), displayName)
			}
		}
		fmt.Println()
	}

	if len(out.OrphanedRuleRefs) > 0 {
		fmt.Println("  Rules referencing non-existent identities:")
		fmt.Println()
		for _, issue := range out.OrphanedRuleRefs {
			if issue.Fixed {
				fmt.Printf("    %s  %s  %s\n", ui.SuccessStyle.Render("removed"), issue.Name, ui.DimStyle.Render("("+issue.Detail+")"))
			} else {
				fmt.Printf("    %s  %s  %s\n", ui.WarningStyle.Render("orphaned"), issue.Name, ui.DimStyle.Render("("+issue.Detail+")"))
			}
		}
		fmt.Println()
	}

	if len(out.UnusedIdentities) > 0 {
		fmt.Println("  Unused identities (no rules, not default, not active):")
		fmt.Println()
		for _, issue := range out.UnusedIdentities {
			fmt.Printf("    %s  %s  %s\n", ui.DimStyle.Render("unused"), issue.Name, ui.DimStyle.Render("("+issue.Detail+")"))
		}
		fmt.Println()
	}

	// Summary
	remaining := out.TotalIssues - out.FixedCount
	if out.FixedCount > 0 {
		fmt.Printf("Fixed %d of %d %s.", out.FixedCount, out.TotalIssues, nounPlural(out.TotalIssues, "issue", "issues"))
		if remaining > 0 {
			fmt.Printf(" %d remaining.", remaining)
		}
		fmt.Println()
	} else {
		fmt.Printf("%d %s found.", out.TotalIssues, nounPlural(out.TotalIssues, "issue", "issues"))
		if len(out.OrphanedRules)+len(out.OrphanedRuleRefs) > 0 {
			fmt.Print(" Run 'gitch clean --fix' to remove orphaned rules.")
		}
		fmt.Println()
	}

	if remaining > 0 {
		return errCleanIssuesFound
	}
	return nil
}
