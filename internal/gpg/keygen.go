package gpg

import (
	"bytes"
	"crypto"
	"errors"
	"fmt"
	"os/exec"
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

	// Generate new entity (keypair)
	entity, err := openpgp.NewEntity(name, fmt.Sprintf("gitch identity: %s", name), email, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate GPG key: %w", err)
	}

	// If passphrase provided, encrypt the private key
	if len(passphrase) > 0 {
		if err := entity.PrivateKey.Encrypt(passphrase); err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
		// Also encrypt subkeys
		for _, subkey := range entity.Subkeys {
			if subkey.PrivateKey != nil {
				if err := subkey.PrivateKey.Encrypt(passphrase); err != nil {
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

	if err := entity.SerializePrivate(privateArmor, config); err != nil {
		return nil, fmt.Errorf("failed to serialize private key: %w", err)
	}
	if err := privateArmor.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize armored key: %w", err)
	}

	// Import the key into system gpg
	if err := importKeyToGPG(privateKeyBuf.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to import key to gpg: %w", err)
	}

	// Get the key info from gpg (which now has the imported key)
	// Use the email to find the key we just created
	keys, err := findKeyByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve imported key info: %w", err)
	}

	if len(keys) == 0 {
		return nil, errors.New("key was imported but could not be found in keyring")
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

// ExportPublicKey exports the public key for the given key ID in armored ASCII format.
// This is useful for displaying to the user to add to GitHub/GitLab.
func ExportPublicKey(keyID string) (string, error) {
	cmd := exec.Command("gpg", "--armor", "--export", keyID)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("failed to export public key: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to export public key: %w", err)
	}

	if len(output) == 0 {
		return "", errKeyNotFound(keyID)
	}

	return string(output), nil
}
