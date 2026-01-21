package cmd

import (
	"fmt"

	"github.com/orkhanrz/gitch/internal/config"
	"github.com/orkhanrz/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var (
	addName    string
	addEmail   string
	addDefault bool
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new git identity",
	Long: `Add a new git identity with a name and email.

The name is used to reference the identity in other commands.
The email is the git user.email that will be used when this identity is active.

Examples:
  gitch add --name work --email work@company.com
  gitch add -n personal -e me@example.com --default`,
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&addName, "name", "n", "", "Identity name (required)")
	addCmd.Flags().StringVarP(&addEmail, "email", "e", "", "Email address (required)")
	addCmd.Flags().BoolVarP(&addDefault, "default", "d", false, "Set as default identity")

	_ = addCmd.MarkFlagRequired("name")
	_ = addCmd.MarkFlagRequired("email")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create identity
	identity := config.Identity{
		Name:  addName,
		Email: addEmail,
	}

	// Add identity (handles validation and duplicate checks)
	if err := cfg.AddIdentity(identity); err != nil {
		return err
	}

	// Set as default if requested
	if addDefault {
		if err := cfg.SetDefault(addName); err != nil {
			return fmt.Errorf("failed to set default: %w", err)
		}
	}

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Print success
	msg := fmt.Sprintf("Added identity '%s' (%s)", addName, addEmail)
	fmt.Println(ui.SuccessStyle.Render(msg))

	if addDefault {
		fmt.Println("Set as default identity")
	}

	return nil
}
