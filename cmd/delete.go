package cmd

import (
	"fmt"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/prompt"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var (
	deleteYes bool
)

var deleteCmd = &cobra.Command{
	Use:     "delete <identity-name>",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a git identity",
	Long: `Delete a git identity by name.

Prompts for confirmation unless --yes is specified.
If the deleted identity is the default, the default is cleared.

Examples:
  gitch delete work
  gitch rm personal --yes`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: identityCompletionFunc,
	RunE:              runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "Skip confirmation prompt")
}

func runDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Verify identity exists
	identity, err := cfg.GetIdentity(name)
	if err != nil {
		return fmt.Errorf("identity '%s' not found. Use 'gitch list' to see available identities", name)
	}

	// Check if this is the currently active identity
	_, activeEmail, _ := git.GetCurrentIdentity()
	isActive := strings.EqualFold(identity.Email, activeEmail)

	// Confirm deletion
	message := fmt.Sprintf("Delete identity '%s'? This cannot be undone.", identity.Name)
	confirmed, err := ui.ConfirmPrompt(message, deleteYes)
	if err != nil {
		return err
	}

	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	// Delete identity
	if err := cfg.DeleteIdentity(name); err != nil {
		return fmt.Errorf("failed to delete identity: %w", err)
	}

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Handle prompt cache if we deleted the active identity
	if isActive {
		// If no identities left, clear cache
		if len(cfg.Identities) == 0 {
			_ = prompt.ClearCache() // Best effort
		}
		// If other identities exist, leave cache as-is (user will switch)
	}

	// Print success
	msg := fmt.Sprintf("Deleted identity '%s'", identity.Name)
	fmt.Println(ui.SuccessStyle.Render(msg))

	// Warn if this was the active identity
	if isActive {
		fmt.Println(ui.WarningStyle.Render("Note: This was the active identity. Git config is unchanged."))
	}

	return nil
}
