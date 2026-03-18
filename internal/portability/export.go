package portability

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/ssh"
	"gopkg.in/yaml.v3"
)

// ErrNoIdentities is returned when trying to export a config with no identities.
var ErrNoIdentities = errors.New("no identities to export")

// BuildExportConfig builds an ExportConfig from the given config.
// Returns the export structure with all identities and rules.
func BuildExportConfig(cfg *config.Config) *ExportConfig {
	return &ExportConfig{
		Version:    CurrentExportVersion,
		ExportedAt: time.Now().UTC(),
		Default:    cfg.Default,
		Identities: cfg.Identities,
		Rules:      cfg.Rules,
	}
}

// writeExportFile creates the file at expandedPath, writes header, then encodes export as YAML.
func writeExportFile(expandedPath, header string, export *ExportConfig) (retErr error) {
	file, err := os.Create(expandedPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()

	if _, err := file.WriteString(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(export); err != nil {
		return fmt.Errorf("failed to write YAML: %w", err)
	}

	return encoder.Close()
}

// prepareExportPath expands ~ in path and ensures the parent directory exists.
func prepareExportPath(path string) (string, error) {
	expandedPath, err := ssh.ExpandPath(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(expandedPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	return expandedPath, nil
}

// ExportToFile exports the configuration to a YAML file at the specified path.
// The path supports ~ expansion for home directory.
// Returns ErrNoIdentities if there are no identities to export.
func ExportToFile(cfg *config.Config, path string) error {
	if len(cfg.Identities) == 0 {
		return ErrNoIdentities
	}

	expandedPath, err := prepareExportPath(path)
	if err != nil {
		return err
	}

	export := BuildExportConfig(cfg)
	header := fmt.Sprintf("# gitch configuration export\n# Exported: %s\n# Version: %d\n\n",
		export.ExportedAt.Format(time.RFC3339),
		export.Version,
	)
	return writeExportFile(expandedPath, header, export)
}

// ExportToFileEncrypted exports configuration with encrypted SSH private keys.
// Reads SSH private key files, encrypts them with the passphrase, and embeds in YAML.
// Returns ErrNoIdentities if there are no identities to export.
func ExportToFileEncrypted(cfg *config.Config, path string, passphrase []byte) error {
	if len(cfg.Identities) == 0 {
		return ErrNoIdentities
	}

	expandedPath, err := prepareExportPath(path)
	if err != nil {
		return err
	}

	// Build encrypted export config
	export := &ExportConfig{
		Version:    CurrentExportVersion,
		ExportedAt: time.Now().UTC(),
		Encryption: &EncryptionInfo{
			Method:  "age-scrypt",
			Armored: true,
		},
		Default:             cfg.Default,
		EncryptedIdentities: make([]EncryptedIdentity, 0, len(cfg.Identities)),
		Rules:               cfg.Rules,
	}

	// Process each identity
	for _, id := range cfg.Identities {
		encId := ToEncryptedIdentity(id)

		// If identity has SSH key, read and encrypt it
		if id.SSHKeyPath != "" {
			keyPath, err := ssh.ExpandPath(id.SSHKeyPath)
			if err != nil {
				return fmt.Errorf("invalid SSH key path for %q: %w", id.Name, err)
			}

			keyData, err := os.ReadFile(keyPath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					// Key file doesn't exist, skip encryption but keep path
					export.EncryptedIdentities = append(export.EncryptedIdentities, encId)
					continue
				}
				return fmt.Errorf("failed to read SSH key for %q: %w", id.Name, err)
			}

			// Encrypt the key
			encrypted, err := EncryptWithPassphrase(keyData, passphrase)
			if err != nil {
				return fmt.Errorf("failed to encrypt SSH key for %q: %w", id.Name, err)
			}

			encId.SSHKeyEncrypted = string(encrypted)
		}

		export.EncryptedIdentities = append(export.EncryptedIdentities, encId)
	}

	header := fmt.Sprintf("# gitch encrypted configuration export\n# Exported: %s\n# Version: %d\n# Encryption: %s\n\n",
		export.ExportedAt.Format(time.RFC3339),
		export.Version,
		export.Encryption.Method,
	)
	return writeExportFile(expandedPath, header, export)
}
