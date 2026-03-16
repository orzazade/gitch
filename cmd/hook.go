package cmd

import (
	"fmt"
	"os"

	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/hooks"
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
	  gitch hook install
	  gitch hook install --global
	  gitch hook uninstall
	  gitch hook uninstall --global`,
}

var hookInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the gitch pre-commit hook",
	Long: `Install the gitch pre-commit hook to validate identity before commits.

By default, gitch installs the hook in the current repository's .git/hooks directory.
Use --global to install via core.hooksPath instead. Global install refuses to overwrite
an existing non-gitch core.hooksPath.

The hook runs 'gitch hook validate' before each commit.

If the current identity doesn't match the expected identity for the repository,
the hook will prompt you to [S]witch, [C]ontinue, or [A]bort.

Examples:
  gitch hook install
  gitch hook install --global`,
	RunE: runHookInstall,
}

var hookUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the gitch pre-commit hook",
	Long: `Remove the gitch pre-commit hook.

By default, removes the hook from the current repository.
Use --global to remove the gitch-managed global hook.

Examples:
  gitch hook uninstall
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
	hookInstallCmd.Flags().BoolVar(&hookGlobal, "global", false, "Install hooks globally via core.hooksPath")
	hookUninstallCmd.Flags().BoolVar(&hookGlobal, "global", false, "Uninstall the gitch-managed global hook")
}

func runHookInstall(cmd *cobra.Command, args []string) error {
	if hookGlobal {
		installed, err := hooks.IsInstalled()
		if err != nil {
			return fmt.Errorf("failed to check hook status: %w", err)
		}
		if installed {
			fmt.Println("Gitch hooks are already installed globally.")
			return nil
		}
		if err := hooks.InstallGlobal(); err != nil {
			return fmt.Errorf("failed to install global hooks: %w", err)
		}
		hooksDir, _ := hooks.HooksDir()
		fmt.Println(ui.SuccessStyle.Render("Global hooks installed at " + hooksDir))
	} else {
		installed, err := hooks.IsInstalledLocal()
		if err != nil {
			return fmt.Errorf("failed to check local hook status: %w", err)
		}
		if installed {
			fmt.Println("Gitch hook is already installed in this repository.")
			return nil
		}
		if err := hooks.InstallLocal(); err != nil {
			return fmt.Errorf("failed to install local hook: %w", err)
		}
		fmt.Println(ui.SuccessStyle.Render("Local repository hook installed"))
	}

	fmt.Println(ui.DimStyle.Render("Git will now validate identity before each commit."))
	fmt.Println(ui.DimStyle.Render("Use GITCH_BYPASS=1 to skip validation."))
	return nil
}

func runHookUninstall(cmd *cobra.Command, args []string) error {
	if hookGlobal {
		installed, err := hooks.IsInstalled()
		if err != nil {
			return fmt.Errorf("failed to check hook status: %w", err)
		}
		if !installed {
			fmt.Println("Gitch global hooks are not installed.")
			return nil
		}
		if err := hooks.UninstallGlobal(); err != nil {
			return fmt.Errorf("failed to uninstall global hooks: %w", err)
		}
		fmt.Println(ui.SuccessStyle.Render("Global hooks removed"))
		return nil
	}

	installed, err := hooks.IsInstalledLocal()
	if err != nil {
		return fmt.Errorf("failed to check local hook status: %w", err)
	}
	if !installed {
		fmt.Println("Gitch hook is not installed in this repository.")
		return nil
	}
	if err := hooks.UninstallLocal(); err != nil {
		return fmt.Errorf("failed to uninstall local hook: %w", err)
	}
	fmt.Println(ui.SuccessStyle.Render("Local repository hook removed"))
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

	// Apply the full profile to git config, signing config, ssh-agent, and prompt cache.
	if err := applyConfiguredIdentity(identity, git.ScopeLocal); err != nil {
		return fmt.Errorf("failed to switch identity: %w", err)
	}

	// Print success
	msg := fmt.Sprintf("Switched to '%s' (%s)", identity.Name, identity.Email)
	fmt.Println(ui.SuccessStyle.Render(msg))
	fmt.Printf("Git author: %s\n", identity.GitAuthorName())

	return nil
}

func runHookMode(cmd *cobra.Command, args []string) error {
	// Get validation result to find expected identity
	result, err := hooks.Validate()
	if err != nil {
		// Default to warn on error
		fmt.Print("warn")
		return nil
	}

	// If no expected identity (no rule matched), default to warn
	if result.ExpectedIdentity == nil {
		fmt.Print("warn")
		return nil
	}

	// Get the hook mode for this identity
	mode := result.ExpectedIdentity.GetHookMode()
	fmt.Print(mode)
	return nil
}

