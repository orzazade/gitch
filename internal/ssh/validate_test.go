package ssh

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestValidateEd25519Key_Valid(t *testing.T) {
	// Generate a valid Ed25519 key
	privKey, _, err := GenerateKeyPair("test@gitch", nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	err = ValidateEd25519Key(privKey)
	if err != nil {
		t.Errorf("ValidateEd25519Key should accept valid Ed25519 key: %v", err)
	}
}

func TestValidateEd25519Key_ValidEncrypted(t *testing.T) {
	// Generate an encrypted Ed25519 key
	privKey, _, err := GenerateKeyPair("test@gitch", []byte("passphrase"))
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	err = ValidateEd25519Key(privKey)
	if err != nil {
		t.Errorf("ValidateEd25519Key should accept encrypted Ed25519 key: %v", err)
	}
}

func TestValidateEd25519Key_RejectsRSA(t *testing.T) {
	// Generate an RSA key
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Marshal to OpenSSH format
	pemBlock, err := ssh.MarshalPrivateKey(rsaKey, "test")
	if err != nil {
		t.Fatalf("Failed to marshal RSA key: %v", err)
	}
	pemData := pem.EncodeToMemory(pemBlock)

	err = ValidateEd25519Key(pemData)
	if err == nil {
		t.Error("ValidateEd25519Key should reject RSA key")
	}
	if !strings.Contains(err.Error(), "not Ed25519") {
		t.Errorf("Error should mention 'not Ed25519', got: %v", err)
	}
}

func TestValidateEd25519Key_RejectsECDSA(t *testing.T) {
	// Generate an ECDSA key
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}

	// Marshal to OpenSSH format
	pemBlock, err := ssh.MarshalPrivateKey(ecdsaKey, "test")
	if err != nil {
		t.Fatalf("Failed to marshal ECDSA key: %v", err)
	}
	pemData := pem.EncodeToMemory(pemBlock)

	err = ValidateEd25519Key(pemData)
	if err == nil {
		t.Error("ValidateEd25519Key should reject ECDSA key")
	}
	if !strings.Contains(err.Error(), "not Ed25519") {
		t.Errorf("Error should mention 'not Ed25519', got: %v", err)
	}
}

func TestValidateEd25519Key_InvalidData(t *testing.T) {
	err := ValidateEd25519Key([]byte("not a valid key"))
	if err == nil {
		t.Error("ValidateEd25519Key should reject invalid data")
	}
}

func TestIsEncrypted_Encrypted(t *testing.T) {
	// Generate an encrypted key
	privKey, _, err := GenerateKeyPair("test@gitch", []byte("passphrase"))
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	if !IsEncrypted(privKey) {
		t.Error("IsEncrypted should return true for encrypted key")
	}
}

func TestIsEncrypted_NotEncrypted(t *testing.T) {
	// Generate an unencrypted key
	privKey, _, err := GenerateKeyPair("test@gitch", nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	if IsEncrypted(privKey) {
		t.Error("IsEncrypted should return false for unencrypted key")
	}
}

func TestIsEncrypted_InvalidData(t *testing.T) {
	// Invalid data should not be considered encrypted
	if IsEncrypted([]byte("invalid data")) {
		t.Error("IsEncrypted should return false for invalid data")
	}
}

func TestValidateKeyPath_ValidKey(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitch-validate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate and write a valid key
	privKey, pubKey, err := GenerateKeyPair("test@gitch", nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "test_key")
	err = WriteKeyFiles(keyPath, privKey, pubKey)
	if err != nil {
		t.Fatalf("Failed to write key: %v", err)
	}

	// Validate the key path
	err = ValidateKeyPath(keyPath)
	if err != nil {
		t.Errorf("ValidateKeyPath should accept valid key: %v", err)
	}
}

func TestValidateKeyPath_NonExistent(t *testing.T) {
	err := ValidateKeyPath("/nonexistent/path/to/key")
	if err == nil {
		t.Error("ValidateKeyPath should fail for non-existent path")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

func TestValidateKeyPath_RejectsPubFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitch-validate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate and write a key
	privKey, pubKey, err := GenerateKeyPair("test@gitch", nil)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "test_key")
	err = WriteKeyFiles(keyPath, privKey, pubKey)
	if err != nil {
		t.Fatalf("Failed to write key: %v", err)
	}

	// Try to validate the .pub file
	err = ValidateKeyPath(keyPath + ".pub")
	if err == nil {
		t.Error("ValidateKeyPath should reject .pub files")
	}
	if !strings.Contains(err.Error(), ".pub") {
		t.Errorf("Error should mention '.pub', got: %v", err)
	}
}

func TestValidateKeyPath_RejectsDirectory(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitch-validate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Try to validate the directory itself
	err = ValidateKeyPath(tmpDir)
	if err == nil {
		t.Error("ValidateKeyPath should reject directories")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("Error should mention 'directory', got: %v", err)
	}
}

func TestValidateKeyPath_RejectsRSAKey(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "gitch-validate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate an RSA key
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Marshal and write
	pemBlock, err := ssh.MarshalPrivateKey(rsaKey, "test")
	if err != nil {
		t.Fatalf("Failed to marshal RSA key: %v", err)
	}
	pemData := pem.EncodeToMemory(pemBlock)

	keyPath := filepath.Join(tmpDir, "rsa_key")
	err = os.WriteFile(keyPath, pemData, 0600)
	if err != nil {
		t.Fatalf("Failed to write RSA key: %v", err)
	}

	// Try to validate the RSA key
	err = ValidateKeyPath(keyPath)
	if err == nil {
		t.Error("ValidateKeyPath should reject RSA key")
	}
	if !strings.Contains(err.Error(), "not Ed25519") {
		t.Errorf("Error should mention 'not Ed25519', got: %v", err)
	}
}

func TestValidateKeyPath_ExpandsTilde(t *testing.T) {
	// This test verifies tilde expansion works
	// We can't create a file in ~ for testing, but we can verify
	// that a path with ~ that doesn't exist gives "not found" not "expand" error
	err := ValidateKeyPath("~/.ssh/nonexistent_gitch_test_key")
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
	// Should get "not found" error, not path expansion error
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error after tilde expansion, got: %v", err)
	}
}
