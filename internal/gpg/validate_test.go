package gpg

import (
	"strings"
	"testing"
)

func TestIsGPGAvailable(t *testing.T) {
	// Just verify it doesn't panic and returns a valid bool
	result := IsGPGAvailable()
	t.Logf("IsGPGAvailable() = %v", result)
}

func TestErrKeyNotFound_ContainsKeyID(t *testing.T) {
	err := errKeyNotFound("ABCD1234")
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "ABCD1234") {
		t.Errorf("expected error to contain key ID, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to mention 'not found', got: %v", err)
	}
}

func TestErrGPGNotFound_Message(t *testing.T) {
	if !strings.Contains(ErrGPGNotFound.Error(), "gpg") {
		t.Errorf("ErrGPGNotFound should mention gpg: %v", ErrGPGNotFound)
	}
}
