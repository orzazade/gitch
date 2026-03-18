package cmd

import (
	"fmt"
	"os"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/profile"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var autoSwitchQuiet bool

var autoSwitchCmd = &cobra.Command{
	Use:   "autoswitch",
	Short: "Apply the best matching identity rule for the current directory or repository",
	Long: `Find the best matching directory or remote rule and apply that identity.

This command is useful for shell hooks, editor integration, or manual checks when
you want rule-based switching without using the interactive selector.

Examples:
  gitch autoswitch
  gitch autoswitch --quiet`,
	Args: cobra.NoArgs,
	RunE: runAutoSwitchCommand,
}

// autoSwitchResult contains the result of an auto-switch attempt
type autoSwitchResult struct {
	Switched      bool
	ToIdentity    string
	SkippedReason string
}

func init() {
	rootCmd.AddCommand(autoSwitchCmd)
	autoSwitchCmd.Flags().BoolVar(&autoSwitchQuiet, "quiet", false, "Exit silently when no switch is needed")
}

func runAutoSwitchCommand(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	result, err := tryAutoSwitch(cfg)
	if err != nil {
		return fmt.Errorf("failed to auto-switch: %w", err)
	}

	if !result.Switched {
		if autoSwitchQuiet {
			return nil
		}
		if result.SkippedReason == "no matching rule" {
			fmt.Println(ui.DimStyle.Render("No matching identity rule."))
			return nil
		}
		fmt.Println(ui.DimStyle.Render("No switch performed: " + result.SkippedReason))
		return nil
	}

	identity, err := cfg.GetIdentity(result.ToIdentity)
	if err != nil {
		return fmt.Errorf("switched identity %q is not available in config: %w", result.ToIdentity, err)
	}

	printSwitchSuccess(identity)
	printScopeInfo(defaultApplyScope())
	return nil
}

// tryAutoSwitch checks if identity should switch based on rules and performs the switch
// Returns result indicating what happened (switched, already correct, no rule, etc.)
func tryAutoSwitch(cfg *config.Config) (*autoSwitchResult, error) {
	// 1. Get current working directory and remote
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	remoteURL, _ := rules.GetGitRemoteURL()

	// 2. Find best matching rule
	matchedRule := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)
	if matchedRule == nil {
		return &autoSwitchResult{
			Switched:      false,
			SkippedReason: "no matching rule",
		}, nil
	}

	// 3. Get expected identity from rule
	expectedIdentity, err := cfg.GetIdentity(matchedRule.Identity)
	if err != nil {
		return &autoSwitchResult{
			Switched:      false,
	
			SkippedReason: fmt.Sprintf("identity '%s' not found", matchedRule.Identity),
		}, nil
	}

	scope := defaultApplyScope()

	// 5. Check if already using the full expected profile
	matches, err := profile.MatchesAtScope(expectedIdentity, scope)
	if err != nil {
		return nil, err
	}
	if matches {
		return &autoSwitchResult{
			Switched:      false,
			ToIdentity:    expectedIdentity.Name,
	
			SkippedReason: "already using correct identity",
		}, nil
	}

	// 6. Perform the switch
	if err := applyConfiguredIdentity(expectedIdentity, scope); err != nil {
		return nil, err
	}

	return &autoSwitchResult{
		Switched:    true,
		ToIdentity:  expectedIdentity.Name,

	}, nil
}
