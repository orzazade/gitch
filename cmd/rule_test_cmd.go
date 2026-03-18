package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var ruleTestCmd = &cobra.Command{
	Use:   "test [path]",
	Short: "Show which identity rule matches a directory",
	Long: `Test which identity rule would match for a given directory (defaults to
the current working directory). Shows all matching rules ranked by specificity,
highlighting the winning rule.

This is useful for validating your rule setup without actually switching identity.

Examples:
  gitch rule test
  gitch rule test ~/work/project
  gitch rule test /home/user/repos/client-project`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRuleTest,
}

func init() {
	ruleCmd.AddCommand(ruleTestCmd)
}

// RuleTestResult holds the result of testing rules against a path.
type RuleTestResult struct {
	Path      string
	RemoteURL string
	Matches   []RuleTestMatch
	Best      *RuleTestMatch
}

// RuleTestMatch holds a single matching rule and its specificity score.
type RuleTestMatch struct {
	Rule        rules.Rule
	Specificity int
	IsBest      bool
}

// TestRulesForPath finds all matching rules for the given path and remote URL.
func TestRulesForPath(cfg *config.Config, path, remoteURL string) *RuleTestResult {
	result := &RuleTestResult{
		Path:      path,
		RemoteURL: remoteURL,
	}

	allRules := cfg.ListRules()
	bestScore := -1

	for _, rule := range allRules {
		if !rule.Matches(path, remoteURL) {
			continue
		}
		score := rule.Specificity()
		match := RuleTestMatch{
			Rule:        rule,
			Specificity: score,
		}
		if score > bestScore {
			bestScore = score
		}
		result.Matches = append(result.Matches, match)
	}

	// Mark the best match
	for i := range result.Matches {
		if result.Matches[i].Specificity == bestScore {
			result.Matches[i].IsBest = true
			result.Best = &result.Matches[i]
			break // first one with best score wins (same as FindBestMatch)
		}
	}

	return result
}

func runRuleTest(cmd *cobra.Command, args []string) error {
	// Determine target path
	var targetPath string
	if len(args) > 0 {
		targetPath = args[0]
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		targetPath = cwd
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.ListRules()) == 0 {
		fmt.Println("No rules configured. Use 'gitch rule add' to create one.")
		return nil
	}

	// Try to get remote URL (ignore errors — not every directory is a git repo)
	remoteURL, _ := rules.GetGitRemoteURL()

	result := TestRulesForPath(cfg, targetPath, remoteURL)

	// Display results
	fmt.Printf("Testing path: %s\n", targetPath)
	if result.RemoteURL != "" {
		fmt.Printf("Git remote:   %s\n", result.RemoteURL)
	}
	fmt.Println()

	if len(result.Matches) == 0 {
		fmt.Println(ui.DimStyle.Render("No rules match this path."))
		return nil
	}

	// Show all matching rules
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  \tTYPE\tPATTERN\tIDENTITY\tSPECIFICITY")

	for _, match := range result.Matches {
		marker := " "
		if match.IsBest {
			marker = ui.CheckmarkStyle.Render("*")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
			marker, match.Rule.Type, match.Rule.Pattern, match.Rule.Identity, match.Specificity)
	}
	w.Flush()

	fmt.Println()
	if result.Best != nil {
		fmt.Println(ui.SuccessStyle.Render(
			fmt.Sprintf("Winner: %s (via %s rule %q)",
				result.Best.Rule.Identity, result.Best.Rule.Type, result.Best.Rule.Pattern)))
	}

	return nil
}
