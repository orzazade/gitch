package ssh

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateKeyPair_NoPassphrase(t *testing.T) {
	privKey, pubKey, err := GenerateKeyPair("test@gitch", nil)
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Verify private key format
	if !bytes.HasPrefix(privKey, []byte("-----BEGIN OPENSSH PRIVATE KEY-----")) {
		t.Errorf("Private key does not start with expected header, got: %s", string(privKey[:50]))
	}

	// Verify public key format
	if !bytes.HasPrefix(pubKey, []byte("ssh-ed25519 ")) {
		t.Errorf("Public key does not start with 'ssh-ed25519', got: %s", string(pubKey[:30]))
	}

	// Verify comment is in public key
	if !bytes.Contains(pubKey, []byte("test@gitch")) {
		t.Errorf("Public key does not contain comment 'test@gitch'")
	}

	// Verify private key can be parsed back
	_, err = ssh.ParseRawPrivateKey(privKey)
	if err != nil {
		t.Errorf("Failed to parse generated private key: %v", err)
	}
}

func TestGenerateKeyPair_WithPassphrase(t *testing.T) {
	passphrase := []byte("test-passphrase-123")
	privKey, pubKey, err := GenerateKeyPair("encrypted@gitch", passphrase)
	if err != nil {
		t.Fatalf("GenerateKeyPair with passphrase failed: %v", err)
	}

	// Verify private key format
	if !bytes.HasPrefix(privKey, []byte("-----BEGIN OPENSSH PRIVATE KEY-----")) {
		t.Errorf("Encrypted private key does not start with expected header")
	}

	// Verify public key format
	if !bytes.HasPrefix(pubKey, []byte("ssh-ed25519 ")) {
		t.Errorf("Public key does not start with 'ssh-ed25519'")
	}

	// Verify parsing without passphrase fails (key is encrypted)
	_, err = ssh.ParseRawPrivateKey(privKey)
	if err == nil {
		t.Error("Expected error when parsing encrypted key without passphrase")
	}

	// Verify it's a passphrase missing error
	if _, ok := err.(*ssh.PassphraseMissingError); !ok {
		t.Errorf("Expected PassphraseMissingError, got: %T", err)
	}

	// Verify parsing with correct passphrase succeeds
	_, err = ssh.ParseRawPrivateKeyWithPassphrase(privKey, passphrase)
	if err != nil {
		t.Errorf("Failed to parse encrypted key with passphrase: %v", err)
	}
}

func TestGenerateKeyPair_EmptyComment(t *testing.T) {
	privKey, pubKey, err := GenerateKeyPair("", nil)
	if err != nil {
		t.Fatalf("GenerateKeyPair with empty comment failed: %v", err)
	}

	// Verify keys are still valid
	if !bytes.HasPrefix(privKey, []byte("-----BEGIN OPENSSH PRIVATE KEY-----")) {
		t.Errorf("Private key has unexpected format")
	}

	if !bytes.HasPrefix(pubKey, []byte("ssh-ed25519 ")) {
		t.Errorf("Public key has unexpected format")
	}
}

func TestWriteKeyFiles(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitch-keygen-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test key
	privKey, pubKey, err := GenerateKeyPair("test@gitch", nil)
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Write key files
	keyPath := filepath.Join(tmpDir, "test_key")
	err = WriteKeyFiles(keyPath, privKey, pubKey)
	if err != nil {
		t.Fatalf("WriteKeyFiles failed: %v", err)
	}

	// Verify private key file exists and has correct permissions
	privInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Private key file not found: %v", err)
	}
	if privInfo.Mode().Perm() != 0600 {
		t.Errorf("Private key has wrong permissions: %o, expected 0600", privInfo.Mode().Perm())
	}

	// Verify public key file exists and has correct permissions
	pubKeyPath := keyPath + ".pub"
	pubInfo, err := os.Stat(pubKeyPath)
	if err != nil {
		t.Fatalf("Public key file not found: %v", err)
	}
	if pubInfo.Mode().Perm() != 0644 {
		t.Errorf("Public key has wrong permissions: %o, expected 0644", pubInfo.Mode().Perm())
	}

	// Verify content matches
	readPriv, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to read private key: %v", err)
	}
	if !bytes.Equal(readPriv, privKey) {
		t.Error("Private key content mismatch")
	}

	readPub, err := os.ReadFile(pubKeyPath)
	if err != nil {
		t.Fatalf("Failed to read public key: %v", err)
	}
	if !bytes.Equal(readPub, pubKey) {
		t.Error("Public key content mismatch")
	}
}

func TestWriteKeyFiles_CreatesDirectory(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitch-keygen-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test key
	privKey, pubKey, err := GenerateKeyPair("test@gitch", nil)
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Write to nested directory that doesn't exist
	keyPath := filepath.Join(tmpDir, "nested", "dir", "test_key")
	err = WriteKeyFiles(keyPath, privKey, pubKey)
	if err != nil {
		t.Fatalf("WriteKeyFiles failed to create nested directories: %v", err)
	}

	// Verify directory was created with secure permissions
	dirInfo, err := os.Stat(filepath.Dir(keyPath))
	if err != nil {
		t.Fatalf("Directory was not created: %v", err)
	}
	if dirInfo.Mode().Perm() != 0700 {
		t.Errorf("Directory has wrong permissions: %o, expected 0700", dirInfo.Mode().Perm())
	}
}

func TestGetFingerprint(t *testing.T) {
	// Generate test key
	_, pubKey, err := GenerateKeyPair("test@gitch", nil)
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Get fingerprint
	fp, err := GetFingerprint(pubKey)
	if err != nil {
		t.Fatalf("GetFingerprint failed: %v", err)
	}

	// Verify fingerprint format (SHA256:base64)
	if !strings.HasPrefix(fp, "SHA256:") {
		t.Errorf("Fingerprint does not have SHA256 prefix: %s", fp)
	}

	// Fingerprint should be deterministic for the same key
	fp2, err := GetFingerprint(pubKey)
	if err != nil {
		t.Fatalf("GetFingerprint second call failed: %v", err)
	}
	if fp != fp2 {
		t.Errorf("Fingerprint is not deterministic: %s vs %s", fp, fp2)
	}
}

func TestGetFingerprint_InvalidKey(t *testing.T) {
	invalidKey := []byte("not a valid ssh key")
	_, err := GetFingerprint(invalidKey)
	if err == nil {
		t.Error("Expected error for invalid key")
	}
}
