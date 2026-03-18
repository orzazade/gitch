package ssh

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/pem"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestValidateKeyPath_ValidKey(t *testing.T) {
	tmpDir := t.TempDir()

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
	tmpDir := t.TempDir()

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
	tmpDir := t.TempDir()

	// Try to validate the directory itself
	err := ValidateKeyPath(tmpDir)
	if err == nil {
		t.Error("ValidateKeyPath should reject directories")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("Error should mention 'directory', got: %v", err)
	}
}

func TestValidateKeyPath_AcceptsRSAKey(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate an RSA key using our function
	privKey, pubKey, err := GenerateKeyPairWithType(KeyTypeRSA, "test@gitch", nil)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "rsa_key")
	err = WriteKeyFiles(keyPath, privKey, pubKey)
	if err != nil {
		t.Fatalf("Failed to write RSA key: %v", err)
	}

	// ValidateKeyPath should now accept RSA keys
	err = ValidateKeyPath(keyPath)
	if err != nil {
		t.Errorf("ValidateKeyPath should accept RSA key: %v", err)
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

// Tests for ValidateSSHKey

func TestValidateSSHKey_AcceptsEd25519(t *testing.T) {
	privKey, _, err := GenerateKeyPairWithType(KeyTypeEd25519, "test@gitch", nil)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}

	err = ValidateSSHKey(privKey)
	if err != nil {
		t.Errorf("ValidateSSHKey should accept Ed25519 key: %v", err)
	}
}

func TestValidateSSHKey_AcceptsRSA(t *testing.T) {
	privKey, _, err := GenerateKeyPairWithType(KeyTypeRSA, "test@gitch", nil)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	err = ValidateSSHKey(privKey)
	if err != nil {
		t.Errorf("ValidateSSHKey should accept RSA key: %v", err)
	}
}

func TestValidateSSHKey_AcceptsEncryptedEd25519(t *testing.T) {
	privKey, _, err := GenerateKeyPairWithType(KeyTypeEd25519, "test@gitch", []byte("passphrase"))
	if err != nil {
		t.Fatalf("Failed to generate encrypted Ed25519 key: %v", err)
	}

	err = ValidateSSHKey(privKey)
	if err != nil {
		t.Errorf("ValidateSSHKey should accept encrypted Ed25519 key: %v", err)
	}
}

func TestValidateSSHKey_AcceptsEncryptedRSA(t *testing.T) {
	privKey, _, err := GenerateKeyPairWithType(KeyTypeRSA, "test@gitch", []byte("passphrase"))
	if err != nil {
		t.Fatalf("Failed to generate encrypted RSA key: %v", err)
	}

	err = ValidateSSHKey(privKey)
	if err != nil {
		t.Errorf("ValidateSSHKey should accept encrypted RSA key: %v", err)
	}
}

func TestValidateSSHKey_RejectsECDSA(t *testing.T) {
	// Generate an ECDSA key (unsupported)
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}

	pemBlock, err := ssh.MarshalPrivateKey(ecdsaKey, "test")
	if err != nil {
		t.Fatalf("Failed to marshal ECDSA key: %v", err)
	}
	pemData := pem.EncodeToMemory(pemBlock)

	err = ValidateSSHKey(pemData)
	if err == nil {
		t.Error("ValidateSSHKey should reject ECDSA key")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("Error should mention 'unsupported', got: %v", err)
	}
}

func TestValidateSSHKey_RejectsInvalidData(t *testing.T) {
	err := ValidateSSHKey([]byte("not a valid key"))
	if err == nil {
		t.Error("ValidateSSHKey should reject invalid data")
	}
}

// Tests for GetKeyType

func TestGetKeyType_IdentifiesEd25519(t *testing.T) {
	privKey, _, err := GenerateKeyPairWithType(KeyTypeEd25519, "test@gitch", nil)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key: %v", err)
	}

	kt, err := GetKeyType(privKey)
	if err != nil {
		t.Fatalf("GetKeyType failed: %v", err)
	}
	if kt != KeyTypeEd25519 {
		t.Errorf("GetKeyType = %v, want %v", kt, KeyTypeEd25519)
	}
}

func TestGetKeyType_IdentifiesRSA(t *testing.T) {
	privKey, _, err := GenerateKeyPairWithType(KeyTypeRSA, "test@gitch", nil)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	kt, err := GetKeyType(privKey)
	if err != nil {
		t.Fatalf("GetKeyType failed: %v", err)
	}
	if kt != KeyTypeRSA {
		t.Errorf("GetKeyType = %v, want %v", kt, KeyTypeRSA)
	}
}

func TestGetKeyType_IdentifiesEncryptedEd25519(t *testing.T) {
	privKey, _, err := GenerateKeyPairWithType(KeyTypeEd25519, "test@gitch", []byte("passphrase"))
	if err != nil {
		t.Fatalf("Failed to generate encrypted Ed25519 key: %v", err)
	}

	kt, err := GetKeyType(privKey)
	if err != nil {
		t.Fatalf("GetKeyType failed for encrypted key: %v", err)
	}
	if kt != KeyTypeEd25519 {
		t.Errorf("GetKeyType = %v, want %v", kt, KeyTypeEd25519)
	}
}

func TestGetKeyType_IdentifiesEncryptedRSA(t *testing.T) {
	privKey, _, err := GenerateKeyPairWithType(KeyTypeRSA, "test@gitch", []byte("passphrase"))
	if err != nil {
		t.Fatalf("Failed to generate encrypted RSA key: %v", err)
	}

	kt, err := GetKeyType(privKey)
	if err != nil {
		t.Fatalf("GetKeyType failed for encrypted key: %v", err)
	}
	if kt != KeyTypeRSA {
		t.Errorf("GetKeyType = %v, want %v", kt, KeyTypeRSA)
	}
}

func TestGetKeyType_RejectsUnsupportedType(t *testing.T) {
	// Generate an ECDSA key (unsupported)
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}

	pemBlock, err := ssh.MarshalPrivateKey(ecdsaKey, "test")
	if err != nil {
		t.Fatalf("Failed to marshal ECDSA key: %v", err)
	}
	pemData := pem.EncodeToMemory(pemBlock)

	_, err = GetKeyType(pemData)
	if err == nil {
		t.Error("GetKeyType should reject unsupported key type")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("Error should mention 'unsupported', got: %v", err)
	}
}

func TestGetKeyType_InvalidData(t *testing.T) {
	_, err := GetKeyType([]byte("not a valid key"))
	if err == nil {
		t.Error("GetKeyType should reject invalid data")
	}
}

func TestValidateKeyPath_AcceptsEncryptedRSAKey(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate an encrypted RSA key
	privKey, pubKey, err := GenerateKeyPairWithType(KeyTypeRSA, "test@gitch", []byte("passphrase"))
	if err != nil {
		t.Fatalf("Failed to generate encrypted RSA key: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "encrypted_rsa_key")
	err = WriteKeyFiles(keyPath, privKey, pubKey)
	if err != nil {
		t.Fatalf("Failed to write RSA key: %v", err)
	}

	// ValidateKeyPath should accept encrypted RSA keys
	err = ValidateKeyPath(keyPath)
	if err != nil {
		t.Errorf("ValidateKeyPath should accept encrypted RSA key: %v", err)
	}
}
