// Package portability provides export/import functionality for gitch configuration.
package portability

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"filippo.io/age"
	"filippo.io/age/armor"
)

var (
	// ErrEmptyPassphrase is returned when an empty passphrase is provided.
	ErrEmptyPassphrase = errors.New("passphrase cannot be empty")
	// ErrDecryptionFailed is returned when decryption fails due to wrong passphrase or corrupted data.
	ErrDecryptionFailed = errors.New("decryption failed: wrong passphrase or corrupted data")
)

// EncryptWithPassphrase encrypts plaintext using age with a passphrase.
// Returns ASCII-armored ciphertext suitable for embedding in YAML.
func EncryptWithPassphrase(plaintext, passphrase []byte) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, ErrEmptyPassphrase
	}

	recipient, err := age.NewScryptRecipient(string(passphrase))
	if err != nil {
		return nil, fmt.Errorf("failed to create recipient: %w", err)
	}

	var buf bytes.Buffer
	armorWriter := armor.NewWriter(&buf)

	w, err := age.Encrypt(armorWriter, recipient)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	if _, err := w.Write(plaintext); err != nil {
		return nil, fmt.Errorf("failed to write plaintext: %w", err)
	}

	// IMPORTANT: Close age writer first, then armor writer
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close encryptor: %w", err)
	}

	if err := armorWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close armor writer: %w", err)
	}

	return buf.Bytes(), nil
}

// DecryptWithPassphrase decrypts age-encrypted ciphertext using a passphrase.
// The ciphertext should be ASCII-armored (from EncryptWithPassphrase).
func DecryptWithPassphrase(ciphertext, passphrase []byte) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, ErrEmptyPassphrase
	}

	identity, err := age.NewScryptIdentity(string(passphrase))
	if err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	// IMPORTANT: Set max work factor to prevent DoS from malicious high values
	identity.SetMaxWorkFactor(22)

	armorReader := armor.NewReader(bytes.NewReader(ciphertext))

	r, err := age.Decrypt(armorReader, identity)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	plaintext, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read decrypted data: %w", err)
	}

	return plaintext, nil
}
