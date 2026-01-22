package cmd

import (
	"fmt"
	"os"

	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/hooks"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var hookGlobal bool

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Manage git pre-commit hooks",
	Long: `Install and manage pre-commit hooks that validate identity before commits.

The hook will detect identity mismatches and prompt you to switch, continue, or abort.
Use GITCH_BYPASS=1 environment variable to skip the hook.

Examples:
  gitch hook install --global
  gitch hook uninstall --global`,
}

var hookInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the gitch pre-commit hook",
	Long: `Install the gitch pre-commit hook to validate identity before commits.

Currently only global installation via core.hooksPath is supported.
This sets up a pre-commit hook that runs 'gitch hook validate' before each commit.

If the current identity doesn't match the expected identity for the repository,
the hook will prompt you to [S]witch, [C]ontinue, or [A]bort.

Examples:
  gitch hook install --global`,
	RunE: runHookInstall,
}

var hookUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the gitch pre-commit hook",
	Long: `Remove the gitch pre-commit hook.

This removes the core.hooksPath configuration and deletes the hooks directory.

Examples:
  gitch hook uninstall --global`,
	RunE: runHookUninstall,
}

// hookValidateCmd is called by the pre-commit script
var hookValidateCmd = &cobra.Command{
	Use:    "validate",
	Short:  "Validate current identity (used by pre-commit hook)",
	Hidden: true,
	RunE:   runHookValidate,
}

// hookSwitchCmd is called by the pre-commit script
var hookSwitchCmd = &cobra.Command{
	Use:    "switch",
	Short:  "Switch to expected identity (used by pre-commit hook)",
	Hidden: true,
	RunE:   runHookSwitch,
}

// hookModeCmd is called by the pre-commit script for non-interactive mode
var hookModeCmd = &cobra.Command{
	Use:    "mode",
	Short:  "Get hook mode for current context (used by pre-commit hook)",
	Hidden: true,
	RunE:   runHookMode,
}

func init() {
	rootCmd.AddCommand(hookCmd)
	hookCmd.AddCommand(hookInstallCmd)
	hookCmd.AddCommand(hookUninstallCmd)
	hookCmd.AddCommand(hookValidateCmd)
	hookCmd.AddCommand(hookSwitchCmd)
	hookCmd.AddCommand(hookModeCmd)

	// Flags
	hookInstallCmd.Flags().BoolVar(&hookGlobal, "global", false, "Install hooks globally (required)")
	_ = hookInstallCmd.MarkFlagRequired("global")

	hookUninstallCmd.Flags().BoolVar(&hookGlobal, "global", false, "Uninstall global hooks (required)")
	_ = hookUninstallCmd.MarkFlagRequired("global")
}

func runHookInstall(cmd *cobra.Command, args []string) error {
	if !hookGlobal {
		return fmt.Errorf("only --global installation is currently supported")
	}

	// Check if already installed
	installed, err := hooks.IsInstalled()
	if err != nil {
		return fmt.Errorf("failed to check hook status: %w", err)
	}

	if installed {
		fmt.Println("Gitch hooks are already installed.")
		return nil
	}

	// Install hooks
	if err := hooks.InstallGlobal(); err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	// Get hooks dir for display
	hooksDir, _ := hooks.HooksDir()

	fmt.Println(ui.SuccessStyle.Render("Global hooks installed at " + hooksDir))
	fmt.Println(ui.DimStyle.Render("Git will now validate identity before each commit."))
	fmt.Println(ui.DimStyle.Render("Use GITCH_BYPASS=1 to skip validation."))

	return nil
}

func runHookUninstall(cmd *cobra.Command, args []string) error {
	if !hookGlobal {
		return fmt.Errorf("only --global uninstallation is currently supported")
	}

	// Check if installed
	installed, err := hooks.IsInstalled()
	if err != nil {
		return fmt.Errorf("failed to check hook status: %w", err)
	}

	if !installed {
		fmt.Println("Gitch hooks are not installed.")
		return nil
	}

	// Uninstall hooks
	if err := hooks.UninstallGlobal(); err != nil {
		return fmt.Errorf("failed to uninstall hooks: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render("Global hooks removed"))

	return nil
}

func runHookValidate(cmd *cobra.Command, args []string) error {
	result, err := hooks.Validate()
	if err != nil {
		return err
	}

	if result.Match {
		// Identity matches or no rule applies - exit silently
		return nil
	}

	// Identity mismatch - print message and exit with error
	fmt.Println(result.FormatMismatch())
	os.Exit(1)
	return nil
}

func runHookSwitch(cmd *cobra.Command, args []string) error {
	// Get the expected identity from validation
	result, err := hooks.Validate()
	if err != nil {
		return err
	}

	if result.ExpectedIdentity == nil {
		return fmt.Errorf("no expected identity found")
	}

	identity := result.ExpectedIdentity

	// Apply identity to git config
	if err := git.ApplyIdentity(identity.Name, identity.Email); err != nil {
		return fmt.Errorf("failed to switch identity: %w", err)
	}

	// Add SSH key to agent if configured
	if identity.SSHKeyPath != "" {
		if err := addSSHKeyForHook(identity.SSHKeyPath); err != nil {
			// Print warning but don't fail the switch
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	// Print success
	msg := fmt.Sprintf("Switched to '%s' (%s)", identity.Name, identity.Email)
	fmt.Println(ui.SuccessStyle.Render(msg))

	return nil
}

func runHookMode(cmd *cobra.Command, args []string) error {
	// For now, always return "warn" as the default mode
	// PREV-02 will add per-identity hook mode configuration
	fmt.Print("warn")
	return nil
}

// addSSHKeyForHook adds an SSH key to the ssh-agent
// Duplicated from use.go to avoid circular dependencies in cmd package
func addSSHKeyForHook(keyPath string) error {
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
