package ssh

import (
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// validateSSHKey validates that the given PEM data is a supported SSH private key.
// Supported types: Ed25519 and RSA.
// Returns nil if the key is valid (encrypted or not).
// Returns an error if the key is not a supported type or cannot be parsed.
func validateSSHKey(pemData []byte) error {
	_, err := getKeyType(pemData)
	return err
}

// getKeyType parses the PEM data and returns the key type.
// Supports both encrypted and unencrypted keys.
// Returns an error if the key cannot be parsed or is not a supported type.
func getKeyType(pemData []byte) (KeyType, error) {
	// Try to parse the private key
	key, err := ssh.ParseRawPrivateKey(pemData)
	if err != nil {
		// Check if it's a passphrase-protected key
		var passErr *ssh.PassphraseMissingError
		if errors.As(err, &passErr) {
			// Key is encrypted - determine type via the public key
			kt := passErr.PublicKey.Type()
			switch kt {
			case ssh.KeyAlgoED25519:
				return KeyTypeEd25519, nil
			case ssh.KeyAlgoRSA:
				return KeyTypeRSA, nil
			default:
				return "", fmt.Errorf("unsupported key type: %s (supported: ed25519, rsa)", kt)
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
		return "", fmt.Errorf("unsupported key type: %T (supported: ed25519, rsa)", key)
	}
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
		if errors.Is(err, os.ErrNotExist) {
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
		return errors.New("path points to a public key (.pub file); provide the private key path instead")
	}

	// Read and validate the key
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	return validateSSHKey(data)
}
