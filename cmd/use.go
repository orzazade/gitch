package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/orzazade/gitch/internal/ui/selector"
	"github.com/spf13/cobra"
)

var (
	useLocalScope  bool
	useGlobalScope bool
)

var useCmd = &cobra.Command{
	Use:   "use [identity-name]",
	Short: "Switch to a git identity",
	Long: `Switch to a git identity by name.

When called without arguments, launches an interactive selector.
When called with an identity name, switches directly.

Applies the selected identity to git config, GPG signing, SSH selection,
and prompt state.

By default, gitch writes repo-local config when run inside a git repository
and global config everywhere else. Use --local or --global to override.

Examples:
  gitch use          # Interactive selector
  gitch use work     # Direct switch
  gitch use personal`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: identityCompletionFunc,
	RunE:              runUse,
}

// identityCompletionFunc returns completions for identity names.
// This provides tab completion for commands that take an identity name argument.
func identityCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete first argument
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeIdentityNames()
}

// completeIdentityNames returns all identity names formatted for shell completion.
func completeIdentityNames() ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	identities := cfg.ListIdentities()
	completions := make([]string, 0, len(identities))
	for _, id := range identities {
		completions = append(completions, id.Name+"\t"+id.Email)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(useCmd)
	useCmd.Flags().BoolVar(&useLocalScope, "local", false, "Apply identity to the current repository only")
	useCmd.Flags().BoolVar(&useGlobalScope, "global", false, "Apply identity to global git config")
}

func runUse(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var identity *config.Identity

	if len(args) == 0 {
		// Interactive mode
		identities := cfg.ListIdentities()
		if len(identities) == 0 {
			fmt.Println("No identities configured.")
			fmt.Println(ui.DimStyle.Render("Run 'gitch setup' to create one."))
			return nil
		}

		// Resolve the exact active profile for highlighting.
		activeProfileName := ""
		if state, stateErr := resolveCurrentProfileState(cfg); stateErr == nil && state.ExactMatch != nil {
			activeProfileName = state.ExactMatch.Name
		}

		// Check if a rule matches - use rule's identity as default selection
		defaultName := cfg.Default
		cwd, _ := os.Getwd()
		remoteURL, _ := rules.GetGitRemoteURL()
		if matchedRule := rules.FindBestMatch(cfg.Rules, cwd, remoteURL); matchedRule != nil {
			defaultName = matchedRule.Identity
		}

		selected, err := selector.Run(identities, activeProfileName, defaultName)
		if err != nil {
			return fmt.Errorf("selector error: %w", err)
		}

		if selected == nil {
			// User cancelled
			return nil
		}

		identity = selected
	} else {
		// Direct mode (existing logic)
		name := args[0]
		identity, err = cfg.GetIdentity(name)
		if err != nil {
			msg := fmt.Sprintf("identity '%s' not found", name)
			if suggestion := closestIdentityName(name, cfg.ListIdentities()); suggestion != "" {
				msg += fmt.Sprintf("\n\nDid you mean '%s'?  Run: gitch use %s", suggestion, suggestion)
			} else {
				msg += ". Use 'gitch list' to see available identities"
			}
			return errors.New(msg)
		}
	}

	scope, err := resolveApplyScope(useLocalScope, useGlobalScope)
	if err != nil {
		return err
	}

	// Apply the full profile to git config, signing config, ssh-agent, and prompt cache.
	if err := applyConfiguredIdentity(identity, scope); err != nil {
		return fmt.Errorf("failed to switch identity: %w", err)
	}

	printSwitchSuccess(identity)
	printScopeInfo(scope)
	if msg := ruleMismatchWarning(identity, cfg); msg != "" {
		fmt.Println(ui.WarningStyle.Render(msg))
	}
	return nil
}

// ruleMismatchWarning returns a warning string if the active identity conflicts
// with the best-matching auto-switch rule for the current directory, or "" if all is fine.
func ruleMismatchWarning(identity *config.Identity, cfg *config.Config) string {
	if len(cfg.Rules) == 0 {
		return ""
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	remoteURL, _ := rules.GetGitRemoteURL()
	matched := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)
	if matched == nil || strings.EqualFold(matched.Identity, identity.Name) {
		return ""
	}
	return fmt.Sprintf(
		"Note: a rule for this directory expects '%s'. The pre-commit hook will catch this mismatch.",
		matched.Identity,
	)
}

// closestIdentityName returns the best-matching identity name for a mistyped input,
// or "" if no candidate is close enough (within 2 edits).
func closestIdentityName(input string, identities []config.Identity) string {
	input = strings.ToLower(input)
	best, bestDist := "", 3 // threshold: only suggest if within 2 edits
	for _, id := range identities {
		d := levenshtein(input, strings.ToLower(id.Name))
		if d < bestDist {
			bestDist, best = d, id.Name
		}
	}
	return best
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	row := make([]int, lb+1)
	for j := range row {
		row[j] = j
	}
	for i := 1; i <= la; i++ {
		prev := i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			val := min(row[j]+1, prev+1, row[j-1]+cost)
			row[j-1] = prev
			prev = val
		}
		row[lb] = prev
	}
	return row[lb]
}
