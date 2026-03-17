package cmd

import (
	"fmt"
	"os"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
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

	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	identities := cfg.ListIdentities()
	completions := make([]string, 0, len(identities))
	for _, id := range identities {
		// Format: "name\temail" - tab separates name from description
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
			return fmt.Errorf("identity '%s' not found. Use 'gitch list' to see available identities", name)
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

	// Print success
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Switched to '%s' (%s)", identity.Name, identity.Email)))
	fmt.Printf("Git author: %s\n", identity.GitAuthorName())
	if scope == git.ScopeLocal {
		fmt.Println("Scope: local repository")
	} else {
		fmt.Println("Scope: global")
	}

	return nil
}
