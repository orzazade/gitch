package cmd

import (
	"fmt"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure gitch settings",
	Long: `Configure gitch settings.

Subcommands allow you to configure various aspects of gitch behavior.

Examples:
  gitch config hook-mode work block`,
}

var configHookModeCmd = &cobra.Command{
	Use:   "hook-mode <identity> <mode>",
	Short: "Set hook behavior for an identity",
	Long: `Set how the pre-commit hook behaves for a specific identity.

Modes:
  allow - Always allow commits (no warning)
  warn  - Show warning but allow commit (default)
  block - Block commits until identity matches

Example:
  gitch config hook-mode work block
  gitch config hook-mode personal allow`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: configHookModeCompletionFunc,
	RunE:              runConfigHookMode,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configHookModeCmd)
}

// configHookModeCompletionFunc provides tab completion for config hook-mode command
func configHookModeCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		// First arg: identity names
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
	case 1:
		// Second arg: mode values
		return []string{
			"allow\tAlways allow commits",
			"warn\tShow warning but allow",
			"block\tBlock commits until identity matches",
		}, cobra.ShellCompDirectiveNoFileComp
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func runConfigHookMode(cmd *cobra.Command, args []string) error {
	identityName := args[0]
	mode := args[1]

	// Validate the mode
	if err := config.ValidateHookMode(mode); err != nil {
		return err
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Find the identity
	identity, err := cfg.GetIdentity(identityName)
	if err != nil {
		return fmt.Errorf("identity '%s' not found. Use 'gitch list' to see available identities", identityName)
	}

	// Update the hook mode
	identity.HookMode = mode

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	msg := fmt.Sprintf("Hook mode for '%s' set to '%s'", identity.Name, mode)
	fmt.Println(ui.SuccessStyle.Render(msg))

	return nil
}
