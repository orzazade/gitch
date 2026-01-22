package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/prompt"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/orzazade/gitch/internal/ui/wizard"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for creating identities",
	Long: `Launch an interactive wizard to create a new git identity.

The wizard guides you through:
  1. Choosing an identity name
  2. Setting the email address
  3. Optionally generating an SSH key
  4. Optionally generating a GPG key for commit signing

Examples:
  gitch setup`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	m := wizard.New()
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard error: %w", err)
	}

	result := finalModel.(wizard.Model)

	// User cancelled
	if result.Cancelled {
		fmt.Println("Setup cancelled.")
		return nil
	}

	// Get wizard result
	data := result.Result()
	if data == nil {
		return nil
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create identity
	identity := config.Identity{
		Name:       data.Name,
		Email:      data.Email,
		SSHKeyPath: data.SSHKeyPath,
		GPGKeyID:   data.GPGKeyID,
	}

	// Add identity
	if err := cfg.AddIdentity(identity); err != nil {
		return err
	}

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Update prompt cache (wizard creates active identity)
	_ = prompt.UpdateCache(data.Name) // Best effort

	// Print success
	fmt.Println()
	msg := fmt.Sprintf("Created identity '%s' (%s)", data.Name, data.Email)
	fmt.Println(ui.SuccessStyle.Render(msg))

	if data.SSHKeyPath != "" {
		fmt.Printf("SSH key: %s\n", data.SSHKeyPath)
	}
	if data.GPGKeyID != "" {
		fmt.Printf("GPG key: %s\n", data.GPGKeyID)
	}

	// Suggest next steps
	fmt.Println()
	fmt.Println(ui.DimStyle.Render("Run 'gitch setup' again to add more identities"))
	fmt.Println(ui.DimStyle.Render("Run 'gitch use " + data.Name + "' to activate this identity"))

	return nil
}
