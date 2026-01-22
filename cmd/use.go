package cmd

import (
	"fmt"
	"os"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/orzazade/gitch/internal/ui/selector"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use [identity-name]",
	Short: "Switch to a git identity",
	Long: `Switch to a git identity by name.

When called without arguments, launches an interactive selector.
When called with an identity name, switches directly.

Updates the global git config (user.name and user.email) to use
the specified identity.

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

		// Get current active email for highlighting
		_, activeEmail, _ := git.GetCurrentIdentity()

		// Check if a rule matches - use rule's identity as default selection
		defaultName := cfg.Default
		cwd, _ := os.Getwd()
		remoteURL, _ := rules.GetGitRemoteURL()
		if matchedRule := rules.FindBestMatch(cfg.Rules, cwd, remoteURL); matchedRule != nil {
			defaultName = matchedRule.Identity
		}

		selected, err := selector.Run(identities, activeEmail, defaultName)
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

	// Apply identity to git config
	if err := git.ApplyIdentity(identity.Name, identity.Email); err != nil {
		return fmt.Errorf("failed to switch identity: %w", err)
	}

	// Add SSH key to agent if configured
	if identity.SSHKeyPath != "" {
		if err := addSSHKeyToAgent(identity.SSHKeyPath); err != nil {
			// Print warning but don't fail the switch
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	// Print success
	msg := fmt.Sprintf("Switched to '%s' (%s)", identity.Name, identity.Email)
	fmt.Println(ui.SuccessStyle.Render(msg))

	return nil
}

// addSSHKeyToAgent adds an SSH key to the ssh-agent.
// Returns an error if the key file doesn't exist or if adding fails.
func addSSHKeyToAgent(keyPath string) error {
	// Check if key file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key not found: %s", keyPath)
	}

	// Add to agent (will prompt for passphrase if needed)
	if err := sshpkg.AddKeyToAgent(keyPath); err != nil {
		return fmt.Errorf("failed to add SSH key to agent: %w", err)
	}

	return nil
}
