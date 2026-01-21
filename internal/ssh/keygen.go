package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// GenerateKeyPair generates an Ed25519 SSH keypair.
// Returns the private key in PEM format and the public key in authorized_keys format.
// If passphrase is provided, the private key will be encrypted.
func GenerateKeyPair(comment string, passphrase []byte) (privateKeyPEM, publicKey []byte, err error) {
	// Generate Ed25519 keypair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Marshal private key to OpenSSH format
	var pemBlock *pem.Block
	if len(passphrase) > 0 {
		pemBlock, err = ssh.MarshalPrivateKeyWithPassphrase(privKey, comment, passphrase)
	} else {
		pemBlock, err = ssh.MarshalPrivateKey(privKey, comment)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Encode private key to PEM
	privateKeyPEM = pem.EncodeToMemory(pemBlock)

	// Generate SSH public key
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SSH public key: %w", err)
	}

	// Format public key in authorized_keys format
	publicKey = ssh.MarshalAuthorizedKey(sshPubKey)

	// Append comment to public key (replace trailing newline)
	if comment != "" {
		// MarshalAuthorizedKey adds a trailing newline, so we trim it and add the comment
		publicKey = append(publicKey[:len(publicKey)-1], []byte(" "+comment+"\n")...)
	}

	return privateKeyPEM, publicKey, nil
}

// WriteKeyFiles writes the SSH keypair to disk with appropriate permissions.
// Private key is written with 0600 permissions.
// Public key is written to {path}.pub with 0644 permissions.
func WriteKeyFiles(privateKeyPath string, privateKey, publicKey []byte) error {
	// Ensure parent directory exists with secure permissions
	dir := filepath.Dir(privateKeyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write private key with restricted permissions (0600)
	if err := os.WriteFile(privateKeyPath, privateKey, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Write public key with readable permissions (0644)
	publicKeyPath := privateKeyPath + ".pub"
	if err := os.WriteFile(publicKeyPath, publicKey, 0644); err != nil {
		// Clean up private key if public key write fails
		os.Remove(privateKeyPath)
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// GetFingerprint returns the SHA256 fingerprint of an SSH public key.
// The input should be in authorized_keys format (e.g., "ssh-ed25519 AAAA... comment").
func GetFingerprint(publicKey []byte) (string, error) {
	// Parse the public key from authorized_keys format
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %w", err)
	}

	// Return SHA256 fingerprint
	return ssh.FingerprintSHA256(pubKey), nil
}
