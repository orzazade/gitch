package cmd

import (
	"errors"
	"fmt"

	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/hooks"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var hookGlobal bool
var hookPrePush bool

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Manage git hooks for identity validation",
	Long: `Install and manage hooks that validate identity before commits and pushes.

The pre-commit hook detects identity mismatches and prompts you to switch, continue, or abort.
The pre-push hook runs 'gitch verify' to catch wrong-identity commits before they leave your machine.
Use GITCH_BYPASS=1 environment variable to skip hooks.

Examples:
	  gitch hook install
	  gitch hook install --global
	  gitch hook install --pre-push
	  gitch hook uninstall
	  gitch hook uninstall --global`,
}

var hookInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install gitch hooks",
	Long: `Install gitch hooks to validate identity before commits and pushes.

By default, installs the pre-commit hook in the current repository.
Use --global to install via core.hooksPath (includes both pre-commit and pre-push).
Use --pre-push to install the pre-push hook locally.

The pre-commit hook runs 'gitch hook validate' and prompts on mismatch.
The pre-push hook runs 'gitch verify' to catch wrong-identity commits before push.

Examples:
  gitch hook install              # pre-commit hook (local)
  gitch hook install --pre-push   # pre-push hook (local)
  gitch hook install --global     # both hooks (global)`,
	RunE: runHookInstall,
}

var hookUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall gitch hooks",
	Long: `Remove gitch hooks.

By default, removes the pre-commit hook from the current repository.
Use --pre-push to remove the pre-push hook.
Use --global to remove the gitch-managed global hooks.

Examples:
  gitch hook uninstall
  gitch hook uninstall --pre-push
  gitch hook uninstall --global`,
	RunE: runHookUninstall,
}

// hookValidateCmd is called by the pre-commit script
var hookValidateCmd = &cobra.Command{
	Use:           "validate",
	Short:         "Validate current identity (used by pre-commit hook)",
	Hidden:        true,
	RunE:          runHookValidate,
	SilenceErrors: true,
	SilenceUsage:  true,
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
	hookInstallCmd.Flags().BoolVar(&hookPrePush, "pre-push", false, "Install the pre-push hook (local only)")
	hookInstallCmd.MarkFlagsMutuallyExclusive("global", "pre-push")
	hookUninstallCmd.Flags().BoolVar(&hookGlobal, "global", false, "Uninstall the gitch-managed global hooks")
	hookUninstallCmd.Flags().BoolVar(&hookPrePush, "pre-push", false, "Uninstall the pre-push hook (local only)")
}

func runHookInstall(cmd *cobra.Command, args []string) error {
	switch {
	case hookGlobal:
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
		fmt.Println(ui.DimStyle.Render("Git will validate identity before each commit and push."))

	case hookPrePush:
		installed, err := hooks.IsInstalledLocalPrePush()
		if err != nil {
			return fmt.Errorf("failed to check local pre-push hook status: %w", err)
		}
		if installed {
			fmt.Println("Gitch pre-push hook is already installed in this repository.")
			return nil
		}
		if err := hooks.InstallLocalPrePush(); err != nil {
			return fmt.Errorf("failed to install local pre-push hook: %w", err)
		}
		fmt.Println(ui.SuccessStyle.Render("Local pre-push hook installed"))
		fmt.Println(ui.DimStyle.Render("Git will verify commit identities before each push."))

	default:
		installed, err := hooks.IsInstalledLocal()
		if err != nil {
			return fmt.Errorf("failed to check local hook status: %w", err)
		}
		if installed {
			fmt.Println("Gitch pre-commit hook is already installed in this repository.")
			return nil
		}
		if err := hooks.InstallLocal(); err != nil {
			return fmt.Errorf("failed to install local hook: %w", err)
		}
		fmt.Println(ui.SuccessStyle.Render("Local pre-commit hook installed"))
		fmt.Println(ui.DimStyle.Render("Git will validate identity before each commit."))
	}

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

	if hookPrePush {
		installed, err := hooks.IsInstalledLocalPrePush()
		if err != nil {
			return fmt.Errorf("failed to check local pre-push hook status: %w", err)
		}
		if !installed {
			fmt.Println("Gitch pre-push hook is not installed in this repository.")
			return nil
		}
		if err := hooks.UninstallLocalPrePush(); err != nil {
			return fmt.Errorf("failed to uninstall local pre-push hook: %w", err)
		}
		fmt.Println(ui.SuccessStyle.Render("Local pre-push hook removed"))
		return nil
	}

	installed, err := hooks.IsInstalledLocal()
	if err != nil {
		return fmt.Errorf("failed to check local hook status: %w", err)
	}
	if !installed {
		fmt.Println("Gitch pre-commit hook is not installed in this repository.")
		return nil
	}
	if err := hooks.UninstallLocal(); err != nil {
		return fmt.Errorf("failed to uninstall local hook: %w", err)
	}
	fmt.Println(ui.SuccessStyle.Render("Local pre-commit hook removed"))
	return nil
}

func runHookValidate(cmd *cobra.Command, args []string) error {
	result, err := hooks.Validate()
	if err != nil {
		return err
	}

	if !result.Match {
		fmt.Println(result.FormatMismatch())
		return errors.New("identity mismatch")
	}
	return nil
}

func runHookSwitch(cmd *cobra.Command, args []string) error {
	// Get the expected identity from validation
	result, err := hooks.Validate()
	if err != nil {
		return err
	}

	if result.ExpectedIdentity == nil {
		return errors.New("no expected identity found")
	}

	identity := result.ExpectedIdentity

	// Apply the full profile to git config, signing config, ssh-agent, and prompt cache.
	if err := applyConfiguredIdentity(identity, git.ScopeLocal); err != nil {
		return fmt.Errorf("failed to switch identity: %w", err)
	}

	printSwitchSuccess(identity)
	return nil
}

func runHookMode(cmd *cobra.Command, args []string) error {
	result, err := hooks.Validate()
	if err != nil {
		fmt.Print("warn")
		return nil
	}

	if result.ExpectedIdentity == nil {
		fmt.Print("warn")
		return nil
	}

	fmt.Print(result.ExpectedIdentity.GetHookMode())
	return nil
}
