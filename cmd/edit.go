package cmd

import (
	"errors"
	"fmt"

	"github.com/orzazade/gitch/internal/config"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var (
	editGitName  string
	editEmail    string
	editSSHKey   string
	editGPGKey   string
	editHookMode string
)

var editCmd = &cobra.Command{
	Use:   "edit <identity-name>",
	Short: "Modify an existing identity",
	Long: `Update properties of an existing identity without re-creating it.

Only the flags you provide are changed; everything else stays the same.
Auto-switch rules that reference this identity are preserved.

Use --ssh-key="" or --gpg-key="" to clear a key association.

Examples:
  gitch edit work --email new@company.com
  gitch edit work --git-name "New Name"
  gitch edit work --ssh-key ~/.ssh/id_new
  gitch edit work --gpg-key ABCD1234EFGH5678
  gitch edit work --hook-mode block
  gitch edit work --ssh-key ""                # remove SSH key`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: identityCompletionFunc,
	RunE:              runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().StringVar(&editGitName, "git-name", "", "Git author name for commits")
	editCmd.Flags().StringVar(&editEmail, "email", "", "Email address")
	editCmd.Flags().StringVar(&editSSHKey, "ssh-key", "", "Path to SSH private key (use empty string to clear)")
	editCmd.Flags().StringVar(&editGPGKey, "gpg-key", "", "GPG key ID (use empty string to clear)")
	editCmd.Flags().StringVar(&editHookMode, "hook-mode", "", "Pre-commit hook mode: allow, warn, or block")
}

func runEdit(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Build updates from flags that were explicitly set
	var updates config.IdentityUpdates
	changed := false

	if cmd.Flags().Changed("git-name") {
		updates.GitName = &editGitName
		changed = true
	}
	if cmd.Flags().Changed("email") {
		updates.Email = &editEmail
		changed = true
	}
	if cmd.Flags().Changed("ssh-key") {
		// Validate non-empty SSH key path
		if editSSHKey != "" {
			expanded, err := sshpkg.ExpandPath(editSSHKey)
			if err != nil {
				return fmt.Errorf("invalid SSH key path: %w", err)
			}
			if err := sshpkg.ValidateKeyPath(expanded); err != nil {
				return fmt.Errorf("SSH key validation failed: %w", err)
			}
			editSSHKey = expanded
		}
		updates.SSHKeyPath = &editSSHKey
		changed = true
	}
	if cmd.Flags().Changed("gpg-key") {
		updates.GPGKeyID = &editGPGKey
		changed = true
	}
	if cmd.Flags().Changed("hook-mode") {
		updates.HookMode = &editHookMode
		changed = true
	}

	if !changed {
		return errors.New("no changes specified; use flags like --email, --git-name, --ssh-key, --gpg-key, or --hook-mode")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check identity exists first for a helpful error message
	if _, err := cfg.GetIdentity(name); err != nil {
		return identityNotFoundError(name, cfg.ListIdentities(), "gitch edit "+name)
	}

	identity, err := cfg.UpdateIdentity(name, updates)
	if err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render("Updated identity '" + identity.Name + "'"))
	fmt.Printf("  Email:     %s\n", identity.Email)
	fmt.Printf("  Git name:  %s\n", identity.GitAuthorName())
	if identity.SSHKeyPath != "" {
		fmt.Printf("  SSH key:   %s\n", identity.SSHKeyPath)
	}
	if identity.GPGKeyID != "" {
		fmt.Printf("  GPG key:   %s\n", identity.GPGKeyID)
	}
	if identity.HookMode != "" {
		fmt.Printf("  Hook mode: %s\n", identity.HookMode)
	}

	return nil
}
