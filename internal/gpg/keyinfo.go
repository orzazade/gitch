// Package gpg provides GPG key generation, validation, and management utilities.
package gpg

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// KeyInfo contains metadata about a GPG key.
type KeyInfo struct {
	// ID is the long (16-character) key ID, e.g., "ABCD1234EFGH5678"
	ID string

	// Email is the email address associated with the key
	Email string

	// Fingerprint is the full 40-character fingerprint
	Fingerprint string

	// Created is when the key was created
	Created time.Time

	// Expires is when the key expires, nil if no expiry
	Expires *time.Time

	// Algorithm is the key algorithm, e.g., "ed25519", "rsa4096"
	Algorithm string

	// Name is the user name associated with the key
	Name string
}

// parseKeyInfo parses gpg --with-colons output to extract key information.
// Delegates to parseMultipleKeys and returns the first key found.
func parseKeyInfo(output string) (*KeyInfo, error) {
	keys := parseMultipleKeys(output)
	if len(keys) == 0 {
		return nil, errors.New("failed to parse GPG key information")
	}
	return &keys[0], nil
}

// parseAlgorithm converts gpg algorithm number to human-readable string.
func parseAlgorithm(algoNum string) string {
	switch algoNum {
	case "1":
		return "rsa"
	case "16":
		return "elgamal"
	case "17":
		return "dsa"
	case "18":
		return "ecdh"
	case "19":
		return "ecdsa"
	case "22":
		return "ed25519"
	default:
		// The field might contain the algorithm name directly (e.g. "rsa4096", "ed25519")
		lower := strings.ToLower(algoNum)
		if strings.Contains(lower, "ed25519") {
			return "ed25519"
		}
		if strings.Contains(lower, "rsa") {
			return "rsa"
		}
		return algoNum
	}
}

// parseUID extracts name and email from a UID string like "Name <email@example.com>".
func parseUID(uid string) (name, email string) {
	// Decode percent-encoded characters (gpg encodes special chars)
	uid = decodeUID(uid)

	// Extract email from angle brackets
	if before, after, ok := strings.Cut(uid, "<"); ok {
		if emailPart, _, ok := strings.Cut(after, ">"); ok {
			email = emailPart
			name = strings.TrimSpace(before)
		}
	}

	if name == "" && email == "" {
		// If no angle brackets, treat entire string as name
		name = uid
	}

	return name, email
}

// decodeUID decodes percent-encoded characters in GPG UID strings.
func decodeUID(s string) string {
	// GPG encodes special characters as %XX
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '%' && i+2 < len(s) {
			if hex, err := strconv.ParseInt(s[i+1:i+3], 16, 32); err == nil {
				result.WriteByte(byte(hex))
				i += 3
				continue
			}
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}
