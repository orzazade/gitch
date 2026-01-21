package cmd

import (
	"fmt"

	"github.com/orkhanrz/gitch/internal/config"
	"github.com/orkhanrz/gitch/internal/git"
	"github.com/orkhanrz/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <identity-name>",
	Short: "Switch to a git identity",
	Long: `Switch to a git identity by name.

Updates the global git config (user.name and user.email) to use
the specified identity.

Examples:
  gitch use work
  gitch use personal`,
	Args: cobra.ExactArgs(1),
	RunE: runUse,
}

func init() {
	rootCmd.AddCommand(useCmd)
}

func runUse(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get identity by name (case-insensitive)
	identity, err := cfg.GetIdentity(name)
	if err != nil {
		return fmt.Errorf("identity '%s' not found. Use 'gitch list' to see available identities", name)
	}

	// Apply identity to git config
	if err := git.ApplyIdentity(identity.Name, identity.Email); err != nil {
		return fmt.Errorf("failed to switch identity: %w", err)
	}

	// Print success
	msg := fmt.Sprintf("Switched to '%s' (%s)", identity.Name, identity.Email)
	fmt.Println(ui.SuccessStyle.Render(msg))

	return nil
}
