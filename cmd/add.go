package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var (
	addName        string
	addEmail       string
	addDefault     bool
	addGenerateSSH bool
	addSSHKey      string
	addForce       bool
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new git identity",
	Long: `Add a new git identity with a name and email.

The name is used to reference the identity in other commands.
The email is the git user.email that will be used when this identity is active.

SSH Key Options:
  --generate-ssh (-s)  Generate a new Ed25519 SSH keypair for this identity
  --ssh-key            Link an existing SSH private key to this identity
  --force              Overwrite existing SSH key if it exists

Examples:
  gitch add --name work --email work@company.com
  gitch add -n personal -e me@example.com --default
  gitch add --name github --email me@github.com --generate-ssh
  gitch add --name work --email work@co.com --ssh-key ~/.ssh/id_ed25519`,
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&addName, "name", "n", "", "Identity name (required)")
	addCmd.Flags().StringVarP(&addEmail, "email", "e", "", "Email address (required)")
	addCmd.Flags().BoolVarP(&addDefault, "default", "d", false, "Set as default identity")
	addCmd.Flags().BoolVarP(&addGenerateSSH, "generate-ssh", "s", false, "Generate new SSH keypair")
	addCmd.Flags().StringVar(&addSSHKey, "ssh-key", "", "Path to existing SSH private key")
	addCmd.Flags().BoolVar(&addForce, "force", false, "Overwrite existing SSH key if it exists")

	_ = addCmd.MarkFlagRequired("name")
	_ = addCmd.MarkFlagRequired("email")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Validate SSH flags are mutually exclusive
	if addGenerateSSH && addSSHKey != "" {
		return errors.New("cannot use both --generate-ssh and --ssh-key")
	}

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

	// Handle SSH key linking
	if addSSHKey != "" {
		expandedPath, err := sshpkg.ExpandPath(addSSHKey)
		if err != nil {
			return fmt.Errorf("invalid SSH key path: %w", err)
		}

		if err := sshpkg.ValidateKeyPath(expandedPath); err != nil {
			return fmt.Errorf("SSH key validation failed: %w", err)
		}

		identity.SSHKeyPath = expandedPath
	}

	// Handle SSH key generation
	if addGenerateSSH {
		keyPath := sshpkg.DefaultSSHKeyPath(addName)
		if keyPath == "" {
			return errors.New("failed to determine SSH key path")
		}

		// Check if key already exists
		if _, err := os.Stat(keyPath); err == nil {
			if !addForce {
				return fmt.Errorf("SSH key already exists at %s; use --force to overwrite", keyPath)
			}
		}

		// Prompt for passphrase
		passphrase, err := ui.ReadPassphraseWithConfirm()
		if err != nil {
			return fmt.Errorf("failed to read passphrase: %w", err)
		}

		// Generate keypair
		privateKey, publicKey, err := sshpkg.GenerateKeyPair(addEmail, passphrase)
		if err != nil {
			return fmt.Errorf("failed to generate SSH keypair: %w", err)
		}

		// Write key files
		if err := sshpkg.WriteKeyFiles(keyPath, privateKey, publicKey); err != nil {
			return fmt.Errorf("failed to write SSH key files: %w", err)
		}

		// Get fingerprint for display
		fingerprint, err := sshpkg.GetFingerprint(publicKey)
		if err != nil {
			return fmt.Errorf("failed to get key fingerprint: %w", err)
		}

		identity.SSHKeyPath = keyPath

		// Print key generation success info
		fmt.Println(ui.SuccessStyle.Render("Generated SSH key:"))
		fmt.Printf("  Path: %s\n", keyPath)
		fmt.Printf("  Fingerprint: %s\n", fingerprint)
		fmt.Println()
		fmt.Println("Public key (add to GitHub/GitLab):")
		fmt.Print(strings.TrimSuffix(string(publicKey), "\n"))
		fmt.Println()
		fmt.Println()
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
