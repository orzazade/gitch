package prompt

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

// cachePath returns the XDG cache file path for the current identity
// The cache file stores the name of the active identity for shell prompt display
func cachePath() (string, error) {
	return xdg.CacheFile("gitch/current-identity")
}

// UpdateCache writes the current identity name to the cache file
// Uses atomic write (temp file + rename) to prevent corruption
// Empty string clears the cache (writes empty file)
func UpdateCache(identityName string) error {
	cachePath, err := cachePath()
	if err != nil {
		return err
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
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
	cachePath, err := cachePath()
	if err != nil {
		return err
	}

	err = os.Remove(cachePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

