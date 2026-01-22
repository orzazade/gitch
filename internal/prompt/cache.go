package prompt

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
)

// CachePath returns the XDG cache file path for the current identity
// The cache file stores the name of the active identity for shell prompt display
func CachePath() (string, error) {
	return xdg.CacheFile("gitch/current-identity")
}

// UpdateCache writes the current identity name to the cache file
// Uses atomic write (temp file + rename) to prevent corruption
// Empty string clears the cache (writes empty file)
func UpdateCache(identityName string) error {
	cachePath, err := CachePath()
	if err != nil {
		return err
	}

	// Create directory if needed
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	// Write to temp file first for atomic operation
	tmpPath := cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(identityName), 0644); err != nil {
		return err
	}

	// Atomic rename
	if err := os.Rename(tmpPath, cachePath); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}

// ClearCache removes the cache file
// Silently succeeds if the file doesn't exist
func ClearCache() error {
	cachePath, err := CachePath()
	if err != nil {
		return err
	}

	err = os.Remove(cachePath)
	if err != nil && os.IsNotExist(err) {
		// File doesn't exist - that's fine
		return nil
	}
	return err
}

// ReadCache reads the current identity from the cache file
// Returns empty string (no error) if file doesn't exist
func ReadCache() (string, error) {
	cachePath, err := CachePath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - return empty string, not error
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}
