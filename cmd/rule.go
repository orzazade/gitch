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

var (
	ruleUse    string
	ruleRemote string
)

var ruleCmd = &cobra.Command{
	Use:   "rule",
	Short: "Manage identity rules",
	Long: `Create, list, and remove rules that automatically match identities to directories or git remotes.

Rules allow gitch to automatically determine which identity to use based on:
- Directory patterns: Match the current working directory
- Remote patterns: Match the git remote URL

Examples:
  gitch rule add ~/work/** --use work
  gitch rule add --remote "github.com/company/*" --use work
  gitch rule list
  gitch rule remove "~/work/**"`,
}

var ruleAddCmd = &cobra.Command{
	Use:   "add [directory-pattern]",
	Short: "Add a new identity rule",
	Long: `Add a new rule that maps a directory or remote pattern to an identity.

For directory rules, provide the pattern as a positional argument:
  gitch rule add ~/work/** --use work
  gitch rule add ~/projects/personal/** --use personal

For remote rules, use the --remote flag:
  gitch rule add --remote "github.com/company/*" --use work
  gitch rule add --remote "github.com/personal/*" --use personal

Patterns support glob syntax:
  * matches any single path segment
  ** matches any number of path segments

Examples:
  gitch rule add ~/work/** --use work
  gitch rule add --remote "github.com/myorg/*" --use work`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRuleAdd,
}

var ruleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured rules",
	Long: `Display all configured identity rules in a table format.

Shows the rule type (directory or remote), the pattern, and the associated identity.`,
	Args: cobra.NoArgs,
	RunE: runRuleList,
}

var ruleRemoveCmd = &cobra.Command{
	Use:   "remove <pattern>",
	Short: "Remove an identity rule",
	Long: `Remove a rule by its exact pattern.

The pattern must match exactly as it was specified when the rule was created.

Examples:
  gitch rule remove "~/work/**"
  gitch rule remove "github.com/company/*"`,
	Args: cobra.ExactArgs(1),
	RunE: runRuleRemove,
}

func init() {
	rootCmd.AddCommand(ruleCmd)
	ruleCmd.AddCommand(ruleAddCmd)
	ruleCmd.AddCommand(ruleListCmd)
	ruleCmd.AddCommand(ruleRemoveCmd)

	// Flags for ruleAddCmd
	ruleAddCmd.Flags().StringVar(&ruleUse, "use", "", "Identity to use when rule matches (required)")
	ruleAddCmd.Flags().StringVar(&ruleRemote, "remote", "", "Remote pattern (mutually exclusive with positional arg)")
	_ = ruleAddCmd.MarkFlagRequired("use")
}

func runRuleAdd(cmd *cobra.Command, args []string) error {
	// Validate that exactly one of positional arg or --remote is provided
	hasPositional := len(args) > 0
	hasRemote := ruleRemote != ""

	if hasPositional && hasRemote {
		return fmt.Errorf("cannot specify both a directory pattern and --remote; use one or the other")
	}

	if !hasPositional && !hasRemote {
		return fmt.Errorf("must specify either a directory pattern or --remote")
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate identity exists
	if _, err := cfg.GetIdentity(ruleUse); err != nil {
		return fmt.Errorf("identity %q not found; use 'gitch list' to see available identities", ruleUse)
	}

	// Build the rule
	var rule rules.Rule
	if hasRemote {
		rule = rules.Rule{
			Type:     rules.RemoteRule,
			Pattern:  ruleRemote,
			Identity: ruleUse,
		}
	} else {
		rule = rules.Rule{
			Type:     rules.DirectoryRule,
			Pattern:  args[0],
			Identity: ruleUse,
		}
	}

	// Validate pattern
	if err := rule.ValidatePattern(); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	// Check for overlapping rules and warn
	overlapping := cfg.FindOverlappingRules(rule)
	if len(overlapping) > 0 {
		fmt.Println(ui.WarningStyle.Render("Warning: This rule may overlap with existing rules:"))
		for _, overlap := range overlapping {
			fmt.Printf("  %s: %s -> %s\n", overlap.Type, overlap.Pattern, overlap.Identity)
		}
		fmt.Println()
	}

	// Add the rule
	if err := cfg.AddRule(rule); err != nil {
		return err
	}

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Print success
	msg := fmt.Sprintf("Rule added: %s -> %s", rule.Pattern, rule.Identity)
	fmt.Println(ui.SuccessStyle.Render(msg))

	return nil
}

func runRuleList(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	rules := cfg.ListRules()
	if len(rules) == 0 {
		fmt.Println("No rules configured. Use 'gitch rule add' to create one.")
		return nil
	}

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TYPE\tPATTERN\tIDENTITY")

	for _, rule := range rules {
		fmt.Fprintf(w, "%s\t%s\t%s\n", rule.Type, rule.Pattern, rule.Identity)
	}

	w.Flush()
	return nil
}

func runRuleRemove(cmd *cobra.Command, args []string) error {
	pattern := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Remove the rule
	if err := cfg.RemoveRule(pattern); err != nil {
		return fmt.Errorf("rule with pattern %q not found", pattern)
	}

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Print success
	msg := fmt.Sprintf("Rule removed: %s", pattern)
	fmt.Println(ui.SuccessStyle.Render(msg))

	return nil
}
