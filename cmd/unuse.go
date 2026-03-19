package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/prompt"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var unuseCmd = &cobra.Command{
	Use:   "unuse",
	Short: "Clear manually-set identity from this repository",
	Long: `Clear the locally-set git identity so automatic rules take over.

When you run 'gitch use <name>' inside a repository, it writes user.name,
user.email, and optionally signing/SSH config to the local git config.
Running 'gitch unuse' removes those local overrides, letting global config
or auto-switch rules determine the active identity.

Must be run inside a git repository.

Examples:
  gitch unuse          # Clear local identity, let rules decide
  gitch use work       # Manually set identity
  gitch unuse          # Undo it — rules take over again`,
	Args: cobra.NoArgs,
	RunE: runUnuse,
}

func init() {
	rootCmd.AddCommand(unuseCmd)
}

func runUnuse(cmd *cobra.Command, args []string) error {
	if !gitpkg.IsGitRepository() {
		return errNotGitRepo
	}

	// Read current local identity before clearing, so we can report what was removed.
	localName, localEmail, _ := gitpkg.GetCurrentIdentityScoped(gitpkg.ScopeLocal)
	if localName == "" && localEmail == "" {
		fmt.Println("No local identity override is set in this repository.")
		return nil
	}

	// Clear all identity-related local config keys.
	keysToUnset := []string{
		"user.name",
		"user.email",
		"user.signingkey",
		"commit.gpgsign",
		"core.sshCommand",
	}
	for _, key := range keysToUnset {
		if err := gitpkg.UnsetConfigScoped(key, gitpkg.ScopeLocal); err != nil {
			return fmt.Errorf("failed to unset %s: %w", key, err)
		}
	}

	// Clear prompt cache since the manually-set identity is gone.
	_ = prompt.UpdateCache("")

	// Report what was cleared.
	cleared := localName
	if localEmail != "" {
		cleared += " (" + localEmail + ")"
	}
	fmt.Println(ui.SuccessStyle.Render("Cleared local identity override: " + strings.TrimSpace(cleared)))

	// Show what identity will now take effect.
	printEffectiveIdentity()

	return nil
}

// printEffectiveIdentity shows the user what identity is now active after clearing local config.
func printEffectiveIdentity() {
	// Check if a rule matches first.
	cwd, _ := os.Getwd()
	remoteURL, _ := rules.GetGitRemoteURL()

	cfg, err := config.Load()
	if err == nil && len(cfg.Rules) > 0 {
		if matched := rules.FindBestMatch(cfg.Rules, cwd, remoteURL); matched != nil {
			fmt.Println("Rule match: identity '" + matched.Identity + "' will apply on next auto-switch or commit hook.")
			return
		}
	}

	// Fall back to global config.
	globalName, globalEmail, err := gitpkg.GetCurrentIdentityScoped(gitpkg.ScopeGlobal)
	if err == nil && globalEmail != "" {
		fmt.Println("Falling back to global identity: " + globalName + " (" + globalEmail + ")")
		return
	}

	fmt.Println(ui.WarningStyle.Render("No rule matches this directory and no global identity is set. Run 'gitch use --global <name>' or add a rule."))
}
