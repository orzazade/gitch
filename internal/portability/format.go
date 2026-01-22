// Package portability provides export/import functionality for gitch configuration.
package portability

import (
	"time"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

// CurrentExportVersion is the current version of the export format.
// Increment this when making breaking changes to the export format.
const CurrentExportVersion = 2

// EncryptionInfo describes the encryption method used for SSH keys.
type EncryptionInfo struct {
	Method  string `yaml:"method"`  // "age-scrypt"
	Armored bool   `yaml:"armored"` // true if ASCII armored
}

// EncryptedIdentity extends Identity with optional encrypted SSH key content.
// When exporting with --encrypt, SSHKeyEncrypted contains the age-encrypted private key.
type EncryptedIdentity struct {
	Name            string `yaml:"name"`
	Email           string `yaml:"email"`
	SSHKeyPath      string `yaml:"ssh_key_path,omitempty"`
	SSHKeyEncrypted string `yaml:"ssh_key_encrypted,omitempty"`
	GPGKeyID        string `yaml:"gpg_key_id,omitempty"`
	HookMode        string `yaml:"hook_mode,omitempty"`
}

// ExportConfig is the root structure for exported configuration.
// It contains all identities and rules that can be backed up and restored.
type ExportConfig struct {
	Version    int               `yaml:"version"`
	ExportedAt time.Time         `yaml:"exported_at"`
	Encryption *EncryptionInfo   `yaml:"encryption,omitempty"`
	Default    string            `yaml:"default,omitempty"`
	Identities []config.Identity `yaml:"identities,omitempty"`
	// EncryptedIdentities is used when exporting with --encrypt flag
	EncryptedIdentities []EncryptedIdentity `yaml:"encrypted_identities,omitempty"`
	Rules               []rules.Rule        `yaml:"rules,omitempty"`
}

// ToEncryptedIdentity converts a config.Identity to EncryptedIdentity.
func ToEncryptedIdentity(id config.Identity) EncryptedIdentity {
	return EncryptedIdentity{
		Name:       id.Name,
		Email:      id.Email,
		SSHKeyPath: id.SSHKeyPath,
		GPGKeyID:   id.GPGKeyID,
		HookMode:   id.HookMode,
	}
}

// ToIdentity converts an EncryptedIdentity back to config.Identity.
func (e EncryptedIdentity) ToIdentity() config.Identity {
	return config.Identity{
		Name:       e.Name,
		Email:      e.Email,
		SSHKeyPath: e.SSHKeyPath,
		GPGKeyID:   e.GPGKeyID,
		HookMode:   e.HookMode,
	}
}
