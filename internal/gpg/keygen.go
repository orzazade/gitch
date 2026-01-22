package gpg

import (
	"bytes"
	"crypto"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

// GenerateKey generates a new Ed25519 GPG key and imports it into the system gpg keyring.
// The key is created with the given name and email, using go-crypto for pure Go generation.
// If passphrase is provided, the key will be encrypted.
// Returns KeyInfo for the newly created key.
func GenerateKey(name, email string, passphrase []byte) (*KeyInfo, error) {
	// Create entity config for Ed25519
	config := &packet.Config{
		Algorithm:              packet.PubKeyAlgoEdDSA,
		DefaultHash:            crypto.SHA256,
		DefaultCipher:          packet.CipherAES256,
		DefaultCompressionAlgo: packet.CompressionZLIB,
		Time:                   func() time.Time { return time.Now() },
	}

	// Create comment for the key
	comment := fmt.Sprintf("gitch identity: %s", name)

	// Generate new entity (keypair)
	entity, err := openpgp.NewEntity(name, comment, email, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate GPG key: %w", err)
	}

	// If passphrase provided, encrypt the private key
	if len(passphrase) > 0 {
		err = entity.PrivateKey.Encrypt(passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
		// Also encrypt subkeys
		for _, subkey := range entity.Subkeys {
			if subkey.PrivateKey != nil {
				err = subkey.PrivateKey.Encrypt(passphrase)
				if err != nil {
					return nil, fmt.Errorf("failed to encrypt subkey: %w", err)
				}
			}
		}
	}

	// Serialize the private key to armored format
	var privateKeyBuf bytes.Buffer
	privateArmor, err := armor.Encode(&privateKeyBuf, openpgp.PrivateKeyType, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create armor encoder: %w", err)
	}

	err = entity.SerializePrivate(privateArmor, config)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize private key: %w", err)
	}
	privateArmor.Close()

	// Import the key into system gpg
	armoredKey := privateKeyBuf.Bytes()
	err = importKeyToGPG(armoredKey)
	if err != nil {
		return nil, fmt.Errorf("failed to import key to gpg: %w", err)
	}

	// Get the key info from gpg (which now has the imported key)
	// Use the email to find the key we just created
	keys, err := FindKeyByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve imported key info: %w", err)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("key was imported but could not be found in keyring")
	}

	// Return the most recently created key (should be the one we just made)
	return &keys[len(keys)-1], nil
}

// importKeyToGPG imports an armored private key into the system gpg keyring.
func importKeyToGPG(armoredKey []byte) error {
	cmd := exec.Command("gpg", "--import", "--batch")
	cmd.Stdin = bytes.NewReader(armoredKey)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gpg import failed: %s - %w", string(output), err)
	}

	return nil
}

// DefaultKeyPath returns the default path for a GPG key file for a gitch identity.
// Format: ~/.gnupg/gitch-{identityName}.asc
// Note: This is for exported key backup; the key is stored in gpg keyring.
func DefaultKeyPath(identityName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gnupg", fmt.Sprintf("gitch-%s.asc", identityName))
}

// ExportPublicKey exports the public key for the given key ID in armored ASCII format.
// This is useful for displaying to the user to add to GitHub/GitLab.
func ExportPublicKey(keyID string) (string, error) {
	cmd := exec.Command("gpg", "--armor", "--export", keyID)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("failed to export public key: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to export public key: %w", err)
	}

	if len(output) == 0 {
		return "", fmt.Errorf("GPG key not found: %s", keyID)
	}

	return string(output), nil
}

// ExportPrivateKey exports the private key for the given key ID in armored ASCII format.
// Warning: This exports the secret key material. Use with caution.
func ExportPrivateKey(keyID string) (string, error) {
	cmd := exec.Command("gpg", "--armor", "--export-secret-keys", keyID)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("failed to export private key: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to export private key: %w", err)
	}

	if len(output) == 0 {
		return "", fmt.Errorf("GPG key not found: %s", keyID)
	}

	return string(output), nil
}

// WriteKeyBackup writes the exported public and private keys to files.
// This creates a backup of the key outside the gpg keyring.
func WriteKeyBackup(keyID, basePath string) error {
	// Export and write public key
	pubKey, err := ExportPublicKey(keyID)
	if err != nil {
		return fmt.Errorf("failed to export public key for backup: %w", err)
	}

	pubPath := basePath + ".pub.asc"
	if err := os.WriteFile(pubPath, []byte(pubKey), 0644); err != nil {
		return fmt.Errorf("failed to write public key backup: %w", err)
	}

	// Export and write private key with restricted permissions
	privKey, err := ExportPrivateKey(keyID)
	if err != nil {
		os.Remove(pubPath) // Clean up
		return fmt.Errorf("failed to export private key for backup: %w", err)
	}

	privPath := basePath + ".asc"
	if err := os.WriteFile(privPath, []byte(privKey), 0600); err != nil {
		os.Remove(pubPath) // Clean up
		return fmt.Errorf("failed to write private key backup: %w", err)
	}

	return nil
}
