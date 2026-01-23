package ssh

import (
	"fmt"
	"strings"
)

// KeyType represents the type of SSH key algorithm.
type KeyType string

const (
	// KeyTypeEd25519 represents Ed25519 keys (default, modern, fast).
	KeyTypeEd25519 KeyType = "ed25519"
	// KeyTypeRSA represents RSA keys (required for Azure DevOps compatibility).
	KeyTypeRSA KeyType = "rsa"
)

// String returns the string representation of the key type.
func (kt KeyType) String() string {
	return string(kt)
}

// ParseKeyType parses a string into a KeyType.
// Accepts "ed25519", "rsa" (case-insensitive).
// Returns an error for invalid types.
func ParseKeyType(s string) (KeyType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ed25519":
		return KeyTypeEd25519, nil
	case "rsa":
		return KeyTypeRSA, nil
	default:
		return "", fmt.Errorf("invalid key type %q: must be one of %v", s, ValidKeyTypes())
	}
}

// ValidKeyTypes returns a slice of valid key types for help text and validation.
func ValidKeyTypes() []string {
	return []string{string(KeyTypeEd25519), string(KeyTypeRSA)}
}
