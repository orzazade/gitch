package cmd

import (
	"errors"
	"fmt"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/portability"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var exportEncrypt bool

var exportCmd = &cobra.Command{
	Use:   "export <file>",
	Short: "Export identities and rules to a YAML file",
	Long: `Export all gitch identities and rules to a YAML file for backup or migration.

The exported file includes:
- All identity names, emails, SSH key paths, GPG key IDs
- All auto-switch rules (directory and remote patterns)
- Export metadata (timestamp, version)

Note: By default, only SSH key paths are exported, not the keys themselves.
Use --encrypt to include encrypted SSH private keys in the export.

Examples:
  gitch export backup.yaml
  gitch export ~/gitch-backup.yaml
  gitch export --encrypt backup.yaml  # Include encrypted SSH keys`,
	Args: cobra.ExactArgs(1),
	RunE: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().BoolVarP(&exportEncrypt, "encrypt", "e", false, "Include encrypted SSH private keys in export")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	outputPath := args[0]

	if len(cfg.Identities) == 0 {
		return errors.New("no identities configured — add one with: gitch add")
	}

	var keysEncrypted int
	if exportEncrypt {
		// Prompt for passphrase with confirmation
		passphrase, err := ui.ReadPassphraseWithConfirm()
		if err != nil {
			return fmt.Errorf("failed to read passphrase: %w", err)
		}
		if len(passphrase) == 0 {
			return errors.New("passphrase required for encrypted export")
		}

		// Count identities with SSH keys
		for _, id := range cfg.Identities {
			if id.SSHKeyPath != "" {
				keysEncrypted++
			}
		}

		if keysEncrypted == 0 {
			fmt.Println(ui.WarningStyle.Render("Warning: No SSH keys to encrypt"))
		}

		if err := portability.ExportToFileEncrypted(cfg, outputPath, passphrase); err != nil {
			return fmt.Errorf("failed to export: %w", err)
		}
		fmt.Println(ui.SuccessStyle.Render("Encrypted export complete!"))
	} else {
		if err := portability.ExportToFile(cfg, outputPath); err != nil {
			return fmt.Errorf("failed to export: %w", err)
		}
		fmt.Println(ui.SuccessStyle.Render("Export complete!"))
	}

	fmt.Printf("  File: %s\n", outputPath)
	fmt.Printf("  Identities: %d\n", len(cfg.Identities))
	if keysEncrypted > 0 {
		fmt.Printf("  SSH keys encrypted: %d\n", keysEncrypted)
	}
	if len(cfg.Rules) > 0 {
		fmt.Printf("  Rules: %d\n", len(cfg.Rules))
	}

	return nil
}
