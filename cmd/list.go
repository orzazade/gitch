package cmd

import (
	"fmt"

	"github.com/orkhanrz/gitch/internal/config"
	"github.com/orkhanrz/gitch/internal/git"
	"github.com/orkhanrz/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all configured identities",
	Long: `List all configured git identities.

The currently active identity is highlighted with a checkmark and green border.
The default identity is marked with "(default)".

Examples:
  gitch list
  gitch ls`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get all identities
	identities := cfg.ListIdentities()
	if len(identities) == 0 {
		fmt.Println("No identities configured. Use 'gitch add' to create one.")
		return nil
	}

	// Get current git identity to determine which is active
	_, activeEmail, err := git.GetCurrentIdentity()
	if err != nil {
		// Non-fatal: just means no identity will be marked as active
		activeEmail = ""
	}

	// Render and print identity list
	output := ui.RenderIdentityList(identities, activeEmail, cfg.Default)
	fmt.Println(output)

	return nil
}
