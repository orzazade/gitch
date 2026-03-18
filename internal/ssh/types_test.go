package ssh

import (
	"testing"
)

func TestKeyType_Label(t *testing.T) {
	tests := []struct {
		kt       KeyType
		expected string
	}{
		{KeyTypeEd25519, "Ed25519"},
		{KeyTypeRSA, "RSA 4096-bit"},
	}

	for _, tt := range tests {
		t.Run(string(tt.kt), func(t *testing.T) {
			if got := tt.kt.Label(); got != tt.expected {
				t.Errorf("Label() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestKeyType_Label_Unknown(t *testing.T) {
	kt := KeyType("unknown")
	if got := kt.Label(); got != "Ed25519" {
		t.Errorf("Label() for unknown = %q, want %q (default)", got, "Ed25519")
	}
}
