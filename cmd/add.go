package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	gpgpkg "github.com/orzazade/gitch/internal/gpg"
	"github.com/orzazade/gitch/internal/prompt"
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
	addKeyType     string
	addGenerateGPG bool
	addGPGKey      string
	addForce       bool
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new git identity",
	Long: `Add a new git identity with a name and email.

The name is used to reference the identity in other commands.
The email is the git user.email that will be used when this identity is active.

SSH Key Options:
  --generate-ssh (-s)  Generate a new SSH keypair for this identity
  --key-type           SSH key type: ed25519 (default) or rsa
  --ssh-key            Link an existing SSH private key to this identity
  --force              Overwrite existing SSH key if it exists

Key Type Auto-Detection:
  When --key-type is not specified, gitch automatically detects Azure DevOps
  remotes and defaults to RSA (which is required for Azure DevOps compatibility).
  For all other remotes, Ed25519 is used by default.

GPG Key Options:
  --generate-gpg       Generate a new Ed25519 GPG key for commit signing
  --gpg-key            Link an existing GPG key ID for commit signing

Examples:
  gitch add --name work --email work@company.com
  gitch add -n personal -e me@example.com --default
  gitch add --name github --email me@github.com --generate-ssh
  gitch add --name azuredev --email work@company.com --generate-ssh --key-type rsa
  gitch add --name work --email work@co.com --ssh-key ~/.ssh/id_ed25519
  gitch add --name work --email work@co.com --generate-gpg
  gitch add --name work --email work@co.com --gpg-key ABCD1234EFGH5678`,
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&addName, "name", "n", "", "Identity name (required)")
	addCmd.Flags().StringVarP(&addEmail, "email", "e", "", "Email address (required)")
	addCmd.Flags().BoolVarP(&addDefault, "default", "d", false, "Set as default identity")
	addCmd.Flags().BoolVarP(&addGenerateSSH, "generate-ssh", "s", false, "Generate new SSH keypair")
	addCmd.Flags().StringVar(&addSSHKey, "ssh-key", "", "Path to existing SSH private key")
	addCmd.Flags().StringVar(&addKeyType, "key-type", "", "SSH key type: ed25519 (default) or rsa")
	addCmd.Flags().BoolVar(&addGenerateGPG, "generate-gpg", false, "Generate new GPG key for signing")
	addCmd.Flags().StringVar(&addGPGKey, "gpg-key", "", "GPG key ID to use for signing")
	addCmd.Flags().BoolVar(&addForce, "force", false, "Overwrite existing SSH key if it exists")

	_ = addCmd.MarkFlagRequired("name")
	_ = addCmd.MarkFlagRequired("email")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Validate SSH flags are mutually exclusive
	if addGenerateSSH && addSSHKey != "" {
		return errors.New("cannot use both --generate-ssh and --ssh-key")
	}

	// Validate GPG flags are mutually exclusive
	if addGenerateGPG && addGPGKey != "" {
		return errors.New("cannot use both --generate-gpg and --gpg-key")
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

		// Determine key type
		var keyType sshpkg.KeyType
		isAzureDevOps, _ := gitpkg.GetCurrentRemoteType()

		if addKeyType != "" {
			// User explicitly specified key type
			var err error
			keyType, err = sshpkg.ParseKeyType(addKeyType)
			if err != nil {
				return fmt.Errorf("invalid --key-type: %w", err)
			}

			// Warn if using Ed25519 with Azure DevOps
			if keyType == sshpkg.KeyTypeEd25519 && isAzureDevOps {
				fmt.Println(ui.WarningStyle.Render("Warning: Ed25519 keys may not work with Azure DevOps. Consider using --key-type rsa"))
				fmt.Println()
			}
		} else {
			// Auto-detect based on remote
			if isAzureDevOps {
				keyType = sshpkg.KeyTypeRSA
				fmt.Println(ui.DimStyle.Render("Using RSA key (Azure DevOps detected)"))
				fmt.Println()
			} else {
				keyType = sshpkg.KeyTypeEd25519
			}
		}

		// Prompt for passphrase
		passphrase, err := ui.ReadPassphraseWithConfirm()
		if err != nil {
			return fmt.Errorf("failed to read passphrase: %w", err)
		}

		// Generate keypair with specified type
		privateKey, publicKey, err := sshpkg.GenerateKeyPairWithType(keyType, addEmail, passphrase)
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

		// Print key generation success info with key type
		keyTypeLabel := "Ed25519"
		if keyType == sshpkg.KeyTypeRSA {
			keyTypeLabel = "RSA 4096-bit"
		}
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Generated %s SSH key:", keyTypeLabel)))
		fmt.Printf("  Path: %s\n", keyPath)
		fmt.Printf("  Fingerprint: %s\n", fingerprint)
		fmt.Println()
		fmt.Println("Public key (add to GitHub/GitLab):")
		fmt.Print(strings.TrimSuffix(string(publicKey), "\n"))
		fmt.Println()
		fmt.Println()
	}

	// Handle GPG key linking (existing key)
	if addGPGKey != "" {
		if err := gpgpkg.ValidateKeyID(addGPGKey); err != nil {
			return fmt.Errorf("GPG key validation failed: %w", err)
		}
		identity.GPGKeyID = addGPGKey
	}

	// Handle GPG key generation
	if addGenerateGPG {
		// Check if gpg is available
		if !gpgpkg.IsGPGAvailable() {
			return errors.New("gpg command not found - install GPG to use signing features")
		}

		// Prompt for passphrase (same as SSH)
		passphrase, err := ui.ReadPassphraseWithConfirm()
		if err != nil {
			return fmt.Errorf("failed to read passphrase: %w", err)
		}

		// Generate GPG key
		keyInfo, err := gpgpkg.GenerateKey(addName, addEmail, passphrase)
		if err != nil {
			return fmt.Errorf("failed to generate GPG key: %w", err)
		}

		identity.GPGKeyID = keyInfo.ID

		// Export public key for display
		publicKey, err := gpgpkg.ExportPublicKey(keyInfo.ID)
		if err != nil {
			// Key was generated but export failed - warn but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to export public key: %v\n", err)
		}

		// Print key generation success info
		fmt.Println(ui.SuccessStyle.Render("Generated GPG key:"))
		fmt.Printf("  Key ID: %s\n", keyInfo.ID)
		fmt.Printf("  Fingerprint: %s\n", keyInfo.Fingerprint)
		fmt.Println()
		if publicKey != "" {
			fmt.Println("Public key (add to GitHub/GitLab):")
			fmt.Print(strings.TrimSuffix(publicKey, "\n"))
			fmt.Println()
			fmt.Println()
		}
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

	// If this is the first identity, update prompt cache (it becomes implicitly active)
	if len(cfg.Identities) == 1 {
		_ = prompt.UpdateCache(identity.Name) // Best effort
	}

	// Print success
	msg := fmt.Sprintf("Added identity '%s' (%s)", addName, addEmail)
	fmt.Println(ui.SuccessStyle.Render(msg))

	if addDefault {
		fmt.Println("Set as default identity")
	}

	return nil
}
