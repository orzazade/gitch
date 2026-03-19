package cmd

import (
	"errors"
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
  gitch config default work
  gitch config hook-mode work block
  gitch config path`,
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

var configDefaultCmd = &cobra.Command{
	Use:   "default [identity]",
	Short: "Get or set the default identity",
	Long: `Get or set the default identity used when no rule matches.

Without arguments, shows the current default identity.
With an argument, sets the default identity to the given name.

Examples:
  gitch config default          # Show current default
  gitch config default work     # Set default to "work"`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: configDefaultCompletionFunc,
	RunE:              runConfigDefault,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the configuration file path",
	Long: `Show the path to the gitch configuration file.

Useful for debugging or manually editing the config.

Examples:
  gitch config path`,
	Args: cobra.NoArgs,
	RunE: runConfigPath,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configDefaultCmd)
	configCmd.AddCommand(configHookModeCmd)
	configCmd.AddCommand(configPathCmd)
}

// configHookModeCompletionFunc provides tab completion for config hook-mode command
func configHookModeCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeIdentityNames()
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

func configDefaultCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeIdentityNames()
}

func runConfigDefault(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// No argument: show current default
	if len(args) == 0 {
		if cfg.Default == "" {
			fmt.Println("No default identity set. Use 'gitch config default <name>' to set one.")
			return nil
		}
		fmt.Println(cfg.Default)
		return nil
	}

	// Set default
	name := args[0]
	if err := cfg.SetDefault(name); err != nil {
		msg := "identity '" + name + "' not found"
		if suggestion := closestIdentityName(name, cfg.ListIdentities()); suggestion != "" {
			msg += "\n\nDid you mean '" + suggestion + "'?  Run: gitch config default " + suggestion
		}
		return errors.New(msg)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render("Default identity set to '" + cfg.Default + "'"))
	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	path, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to determine config path: %w", err)
	}
	fmt.Println(path)
	return nil
}

func runConfigHookMode(cmd *cobra.Command, args []string) error {
	identityName := args[0]
	mode := args[1]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check identity exists first for a helpful error message
	if _, err := cfg.GetIdentity(identityName); err != nil {
		msg := "identity '" + identityName + "' not found"
		if suggestion := closestIdentityName(identityName, cfg.ListIdentities()); suggestion != "" {
			msg += "\n\nDid you mean '" + suggestion + "'?  Run: gitch config hook-mode " + suggestion + " " + mode
		}
		return errors.New(msg)
	}

	// Update identity via UpdateIdentity (validates mode and finds identity)
	updated, err := cfg.UpdateIdentity(identityName, config.IdentityUpdates{HookMode: &mode})
	if err != nil {
		return err
	}

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render("Hook mode for '" + updated.Name + "' set to '" + mode + "'"))

	return nil
}
