package ssh

import (
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// ValidateSSHKey validates that the given PEM data is a supported SSH private key.
// Supported types: Ed25519 and RSA.
// Returns nil if the key is valid (encrypted or not).
// Returns an error if the key is not a supported type or cannot be parsed.
func ValidateSSHKey(pemData []byte) error {
	// Try to parse the private key
	key, err := ssh.ParseRawPrivateKey(pemData)
	if err != nil {
		// Check if it's a passphrase-protected key
		passErr, ok := err.(*ssh.PassphraseMissingError)
		if ok {
			// Key is encrypted - check if it's a supported type via the public key
			keyType := passErr.PublicKey.Type()
			if keyType == ssh.KeyAlgoED25519 || keyType == ssh.KeyAlgoRSA {
				return nil // Valid encrypted key of supported type
			}
			return fmt.Errorf("unsupported key type: %s (supported: ed25519, rsa)", keyType)
		}
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Key parsed successfully - verify it's a supported type
	switch key.(type) {
	case ed25519.PrivateKey, *ed25519.PrivateKey:
		return nil
	case *rsa.PrivateKey:
		return nil
	default:
		return fmt.Errorf("unsupported key type: %T (supported: ed25519, rsa)", key)
	}
}

// ValidateEd25519Key validates that the given PEM data is an Ed25519 private key.
// Returns nil if the key is a valid Ed25519 key (encrypted or not).
// Returns an error if the key is not Ed25519 or cannot be parsed.
// Deprecated: Use ValidateSSHKey for broader key type support.
func ValidateEd25519Key(pemData []byte) error {
	// Try to parse the private key
	key, err := ssh.ParseRawPrivateKey(pemData)
	if err != nil {
		// Check if it's a passphrase-protected key
		passErr, ok := err.(*ssh.PassphraseMissingError)
		if ok {
			// Key is encrypted - check if it's Ed25519 via the public key
			if passErr.PublicKey.Type() == ssh.KeyAlgoED25519 {
				return nil // Valid encrypted Ed25519 key
			}
			return fmt.Errorf("key is not Ed25519: found %s", passErr.PublicKey.Type())
		}
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Key parsed successfully - verify it's Ed25519
	switch key.(type) {
	case ed25519.PrivateKey:
		return nil
	case *ed25519.PrivateKey:
		return nil
	default:
		return fmt.Errorf("key is not Ed25519: found %T", key)
	}
}

// GetKeyType parses the PEM data and returns the key type.
// Supports both encrypted and unencrypted keys.
// Returns an error if the key cannot be parsed or is not a supported type.
func GetKeyType(pemData []byte) (KeyType, error) {
	// Try to parse the private key
	key, err := ssh.ParseRawPrivateKey(pemData)
	if err != nil {
		// Check if it's a passphrase-protected key
		passErr, ok := err.(*ssh.PassphraseMissingError)
		if ok {
			// Key is encrypted - determine type via the public key
			switch passErr.PublicKey.Type() {
			case ssh.KeyAlgoED25519:
				return KeyTypeEd25519, nil
			case ssh.KeyAlgoRSA:
				return KeyTypeRSA, nil
			default:
				return "", fmt.Errorf("unsupported key type: %s", passErr.PublicKey.Type())
			}
		}
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// Key parsed successfully - determine type
	switch key.(type) {
	case ed25519.PrivateKey, *ed25519.PrivateKey:
		return KeyTypeEd25519, nil
	case *rsa.PrivateKey:
		return KeyTypeRSA, nil
	default:
		return "", fmt.Errorf("unsupported key type: %T", key)
	}
}

// IsEncrypted checks if the given PEM data represents an encrypted private key.
func IsEncrypted(pemData []byte) bool {
	_, err := ssh.ParseRawPrivateKey(pemData)
	if err == nil {
		return false // Key parsed without passphrase, not encrypted
	}

	// Check if the error is because passphrase is required
	_, ok := err.(*ssh.PassphraseMissingError)
	return ok
}

// ValidateKeyPath validates an SSH key file at the given path.
// Expands the path, checks the file exists, validates it's not a .pub file,
// and verifies it's a supported key type (Ed25519 or RSA).
func ValidateKeyPath(path string) error {
	// Expand path (~ and env vars)
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	// Check file exists
	info, err := os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("key file not found: %s", expandedPath)
		}
		return fmt.Errorf("cannot access key file: %w", err)
	}

	// Check it's a file, not a directory
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a key file: %s", expandedPath)
	}

	// Check it's not a .pub file (common mistake)
	if strings.HasSuffix(filepath.Base(expandedPath), ".pub") {
		return fmt.Errorf("path points to a public key (.pub file); provide the private key path instead")
	}

	// Read and validate the key
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	return ValidateSSHKey(data)
}
