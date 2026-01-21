package cmd

import (
	"fmt"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current active git identity",
	Long: `Show the currently active git identity.

Displays the name and email from git config and indicates if it's
managed by gitch.

Examples:
  gitch status`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Get current git identity
	name, email, err := git.GetCurrentIdentity()
	if err != nil {
		return fmt.Errorf("failed to get current identity: %w", err)
	}

	// Check if git has any identity configured
	if name == "" && email == "" {
		fmt.Println("No active identity. Use 'gitch use <name>' to set one.")
		return nil
	}

	// Load config to check if this identity is managed by gitch
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Try to find matching identity by email
	var managed bool
	var managedName string
	for _, identity := range cfg.ListIdentities() {
		if strings.EqualFold(identity.Email, email) {
			managed = true
			managedName = identity.Name
			break
		}
	}

	// Format output
	if managed {
		msg := fmt.Sprintf("Active: %s (%s)", managedName, email)
		fmt.Println(ui.SuccessStyle.Render(msg))
	} else {
		if name != "" {
			fmt.Printf("Active: %s (%s) ", name, email)
		} else {
			fmt.Printf("Active: (%s) ", email)
		}
		fmt.Println(ui.WarningStyle.Render("[not managed by gitch]"))
	}

	return nil
}
