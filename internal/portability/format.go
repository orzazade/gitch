// Package portability provides export/import functionality for gitch configuration.
package portability

import (
	"time"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

// currentExportVersion is the current version of the export format.
// Increment this when making breaking changes to the export format.
const currentExportVersion = 2

// encryptionInfo describes the encryption method used for SSH keys.
type encryptionInfo struct {
	Method  string `yaml:"method"`  // "age-scrypt"
	Armored bool   `yaml:"armored"` // true if ASCII armored
}

// encryptedIdentity extends Identity with optional encrypted SSH key content.
// When exporting with --encrypt, SSHKeyEncrypted contains the age-encrypted private key.
type encryptedIdentity struct {
	Name            string `yaml:"name"`
	GitName         string `yaml:"git_name,omitempty"`
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
	Encryption *encryptionInfo   `yaml:"encryption,omitempty"`
	Default    string            `yaml:"default,omitempty"`
	Identities []config.Identity `yaml:"identities,omitempty"`
	// EncryptedIdentities is used when exporting with --encrypt flag
	EncryptedIdentities []encryptedIdentity `yaml:"encrypted_identities,omitempty"`
	Rules               []rules.Rule        `yaml:"rules,omitempty"`
}

// toencryptedIdentity converts a config.Identity to encryptedIdentity.
func toencryptedIdentity(id config.Identity) encryptedIdentity {
	return encryptedIdentity{
		Name:       id.Name,
		GitName:    id.GitName,
		Email:      id.Email,
		SSHKeyPath: id.SSHKeyPath,
		GPGKeyID:   id.GPGKeyID,
		HookMode:   id.HookMode,
	}
}

// ToIdentity converts an encryptedIdentity back to config.Identity.
func (e encryptedIdentity) ToIdentity() config.Identity {
	return config.Identity{
		Name:       e.Name,
		GitName:    e.GitName,
		Email:      e.Email,
		SSHKeyPath: e.SSHKeyPath,
		GPGKeyID:   e.GPGKeyID,
		HookMode:   e.HookMode,
	}
}
