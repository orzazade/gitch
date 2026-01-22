// Package gpg provides GPG key generation, validation, and management utilities.
package gpg

import (
	"fmt"
	"os/exec"
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

// GetKeyInfo retrieves information about a GPG key by its key ID.
// The keyID can be a short ID, long ID, fingerprint, or email address.
// Returns an error if the key is not found in the gpg keyring.
func GetKeyInfo(keyID string) (*KeyInfo, error) {
	// Run gpg to list secret keys with colon-delimited output
	cmd := exec.Command("gpg", "--list-secret-keys", "--keyid-format", "LONG", "--with-colons", keyID)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// gpg returns non-zero when key not found
			if len(exitErr.Stderr) > 0 {
				return nil, fmt.Errorf("GPG key not found: %s", keyID)
			}
		}
		return nil, fmt.Errorf("GPG key not found: %s", keyID)
	}

	return parseKeyInfo(string(output))
}

// parseKeyInfo parses gpg --with-colons output to extract key information.
// GnuPG colon format documentation: https://www.gnupg.org/documentation/manuals/gnupg/gpg-colon-formats.html
func parseKeyInfo(output string) (*KeyInfo, error) {
	info := &KeyInfo{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) < 10 {
			continue
		}

		recordType := fields[0]

		switch recordType {
		case "sec":
			// Secret key record
			// Field 3: key length
			// Field 4: algorithm (1=RSA, 16=Elgamal, 17=DSA, 18=ECDH, 19=ECDSA, 22=EdDSA)
			// Field 5: key ID (long format)
			// Field 6: creation date (Unix timestamp)
			// Field 7: expiration date (Unix timestamp, empty if no expiry)
			info.ID = fields[4]
			info.Algorithm = parseAlgorithm(fields[3])

			if fields[5] != "" {
				if ts, err := strconv.ParseInt(fields[5], 10, 64); err == nil {
					info.Created = time.Unix(ts, 0)
				}
			}

			if fields[6] != "" {
				if ts, err := strconv.ParseInt(fields[6], 10, 64); err == nil {
					expires := time.Unix(ts, 0)
					info.Expires = &expires
				}
			}

		case "fpr":
			// Fingerprint record
			// Field 10: fingerprint (40 hex chars)
			if len(fields) > 9 && fields[9] != "" {
				info.Fingerprint = fields[9]
			}

		case "uid":
			// User ID record
			// Field 10: user ID string (Name <email>)
			if len(fields) > 9 && info.Email == "" {
				name, email := parseUID(fields[9])
				info.Name = name
				info.Email = email
			}
		}
	}

	if info.ID == "" {
		return nil, fmt.Errorf("failed to parse GPG key information")
	}

	return info, nil
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
		// For algorithms with key length like "rsa4096" or "ed25519"
		// The field might contain the algorithm name directly
		if strings.Contains(strings.ToLower(algoNum), "ed25519") {
			return "ed25519"
		}
		if strings.Contains(strings.ToLower(algoNum), "rsa") {
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
	if start := strings.Index(uid, "<"); start != -1 {
		if end := strings.Index(uid, ">"); end > start {
			email = uid[start+1 : end]
			name = strings.TrimSpace(uid[:start])
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
	result := strings.Builder{}
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
