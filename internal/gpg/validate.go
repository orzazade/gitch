package gpg

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ValidateKeyID validates that a GPG key with the given ID exists in the keyring.
// The keyID should be in long format (16 hex characters), but short IDs and
// fingerprints are also accepted.
// Returns nil if the key is found, or an error if not found.
func ValidateKeyID(keyID string) error {
	cmd := exec.Command("gpg", "--list-secret-keys", "--keyid-format", "LONG", keyID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if gpg is not installed
		if isCommandNotFound(err) {
			return fmt.Errorf("gpg command not found - install GPG to use signing features")
		}
		// Key not found or other error
		return fmt.Errorf("GPG key not found: %s", keyID)
	}

	// Verify output contains key information
	if !strings.Contains(string(output), "sec") {
		return fmt.Errorf("GPG key not found: %s", keyID)
	}

	return nil
}

// FindKeyByEmail searches for GPG secret keys associated with the given email address.
// Returns a slice of KeyInfo for all matching keys (may be empty if none found).
// This enables auto-detection of existing GPG keys for an identity.
func FindKeyByEmail(email string) ([]KeyInfo, error) {
	// Check if gpg is available first
	if !IsGPGAvailable() {
		return nil, fmt.Errorf("gpg command not found - install GPG to use signing features")
	}

	cmd := exec.Command("gpg", "--list-secret-keys", "--keyid-format", "LONG", "--with-colons", email)
	output, err := cmd.Output()
	if err != nil {
		// No keys found is not an error - return empty slice
		if exitErr, ok := err.(*exec.ExitError); ok {
			// gpg returns non-zero when no keys match
			_ = exitErr
			return []KeyInfo{}, nil
		}
		return nil, fmt.Errorf("failed to search for GPG keys: %w", err)
	}

	return parseMultipleKeys(string(output))
}

// IsGPGAvailable checks if the gpg command is installed and accessible.
// Returns true if gpg is available, false otherwise.
func IsGPGAvailable() bool {
	cmd := exec.Command("gpg", "--version")
	err := cmd.Run()
	return err == nil
}

// parseMultipleKeys parses gpg --with-colons output that may contain multiple keys.
func parseMultipleKeys(output string) ([]KeyInfo, error) {
	var keys []KeyInfo
	var currentKey *KeyInfo
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) < 10 {
			continue
		}

		recordType := fields[0]

		switch recordType {
		case "sec":
			// New secret key - save previous if exists
			if currentKey != nil && currentKey.ID != "" {
				keys = append(keys, *currentKey)
			}
			currentKey = &KeyInfo{}

			currentKey.ID = fields[4]
			currentKey.Algorithm = parseAlgorithm(fields[3])

			if fields[5] != "" {
				if ts, err := parseUnixTimestamp(fields[5]); err == nil {
					currentKey.Created = ts
				}
			}

			if fields[6] != "" {
				if ts, err := parseUnixTimestamp(fields[6]); err == nil {
					currentKey.Expires = &ts
				}
			}

		case "fpr":
			if currentKey != nil && currentKey.Fingerprint == "" {
				if len(fields) > 9 && fields[9] != "" {
					currentKey.Fingerprint = fields[9]
				}
			}

		case "uid":
			if currentKey != nil && currentKey.Email == "" {
				if len(fields) > 9 {
					name, email := parseUID(fields[9])
					currentKey.Name = name
					currentKey.Email = email
				}
			}
		}
	}

	// Don't forget the last key
	if currentKey != nil && currentKey.ID != "" {
		keys = append(keys, *currentKey)
	}

	return keys, nil
}

// parseUnixTimestamp parses a Unix timestamp string to time.Time.
func parseUnixTimestamp(s string) (t time.Time, err error) {
	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(ts, 0), nil
}

// isCommandNotFound checks if the error indicates the command was not found.
func isCommandNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "executable file not found") ||
		strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "no such file")
}
