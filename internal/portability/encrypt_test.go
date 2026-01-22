package portability

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	plaintext := []byte("-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key-content\n-----END OPENSSH PRIVATE KEY-----")
	passphrase := []byte("test-passphrase-123")

	encrypted, err := EncryptWithPassphrase(plaintext, passphrase)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Verify it's armored (starts with age header)
	if !bytes.Contains(encrypted, []byte("-----BEGIN AGE ENCRYPTED FILE-----")) {
		t.Error("encrypted output is not ASCII armored")
	}

	decrypted, err := DecryptWithPassphrase(encrypted, passphrase)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecrypt_WrongPassphrase(t *testing.T) {
	plaintext := []byte("secret data")
	passphrase := []byte("correct-passphrase")
	wrongPassphrase := []byte("wrong-passphrase")

	encrypted, err := EncryptWithPassphrase(plaintext, passphrase)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	_, err = DecryptWithPassphrase(encrypted, wrongPassphrase)
	if err == nil {
		t.Error("expected error for wrong passphrase")
	}
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("expected ErrDecryptionFailed, got: %v", err)
	}
}

func TestEncrypt_EmptyPassphrase(t *testing.T) {
	_, err := EncryptWithPassphrase([]byte("data"), []byte{})
	if !errors.Is(err, ErrEmptyPassphrase) {
		t.Errorf("expected ErrEmptyPassphrase, got: %v", err)
	}
}

func TestDecrypt_EmptyPassphrase(t *testing.T) {
	_, err := DecryptWithPassphrase([]byte("data"), []byte{})
	if !errors.Is(err, ErrEmptyPassphrase) {
		t.Errorf("expected ErrEmptyPassphrase, got: %v", err)
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	plaintext := []byte{}
	passphrase := []byte("passphrase")

	encrypted, err := EncryptWithPassphrase(plaintext, passphrase)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := DecryptWithPassphrase(encrypted, passphrase)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("expected empty plaintext, got %q", decrypted)
	}
}

func TestEncryptDecrypt_LargeData(t *testing.T) {
	// Simulate a typical SSH key (4KB)
	plaintext := make([]byte, 4096)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}
	passphrase := []byte("test-passphrase")

	encrypted, err := EncryptWithPassphrase(plaintext, passphrase)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := DecryptWithPassphrase(encrypted, passphrase)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("roundtrip failed for large data")
	}
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	_, err := DecryptWithPassphrase([]byte("not-valid-age-data"), []byte("passphrase"))
	if err == nil {
		t.Error("expected error for invalid ciphertext")
	}
}
