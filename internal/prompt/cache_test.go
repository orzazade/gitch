package prompt

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readCache is a test helper that reads the current identity from the cache file.
func readCache() (string, error) {
	p, err := cachePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// TestUpdateCache verifies that UpdateCache writes to file and can be read back
func TestUpdateCache(t *testing.T) {
	// Use temp directory for test isolation
	tmpDir := t.TempDir()

	// Override XDG cache path for testing
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// Test writing identity name
	identityName := "work"
	if err := UpdateCache(identityName); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}

	// Verify file exists
	cachePath, _ := cachePath()
	if _, err := os.Stat(cachePath); errors.Is(err, os.ErrNotExist) {
		t.Fatal("Cache file was not created")
	}

	// Read back and verify content
	content, err := readCache()
	if err != nil {
		t.Fatalf("ReadCache failed: %v", err)
	}
	if content != identityName {
		t.Errorf("Expected %q, got %q", identityName, content)
	}
}

// TestClearCache verifies that ClearCache removes the cache file
func TestClearCache(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// Create cache file first
	if err := UpdateCache("test-identity"); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Clear it
	if err := ClearCache(); err != nil {
		t.Fatalf("ClearCache failed: %v", err)
	}

	// Verify file is gone
	cachePath, _ := cachePath()
	if _, err := os.Stat(cachePath); !errors.Is(err, os.ErrNotExist) {
		t.Error("Cache file should not exist after ClearCache")
	}
}

// Test_readCacheMissing verifies ReadCache returns empty string for non-existent file
func Test_readCacheMissing(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// Don't create any file - just read
	content, err := readCache()
	if err != nil {
		t.Fatalf("ReadCache should not error for missing file: %v", err)
	}
	if content != "" {
		t.Errorf("Expected empty string for missing file, got %q", content)
	}
}

// TestClearCacheMissing verifies ClearCache doesn't error for non-existent file
func TestClearCacheMissing(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// Clear without creating - should not error
	if err := ClearCache(); err != nil {
		t.Fatalf("ClearCache should not error for missing file: %v", err)
	}
}

// TestAtomicWrite verifies the temp file is cleaned up after successful write
func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// Write to cache
	if err := UpdateCache("atomic-test"); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}

	// Verify .tmp file doesn't exist
	cachePath, _ := cachePath()
	tmpPath := cachePath + ".tmp"
	if _, err := os.Stat(tmpPath); !errors.Is(err, os.ErrNotExist) {
		t.Error(".tmp file should be cleaned up after successful write")
	}

	// Verify main file exists with correct content
	content, err := readCache()
	if err != nil {
		t.Fatalf("ReadCache failed: %v", err)
	}
	if content != "atomic-test" {
		t.Errorf("Expected 'atomic-test', got %q", content)
	}
}

// TestUpdateCacheEmpty verifies empty string clears cache content
func TestUpdateCacheEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// Create with content first
	if err := UpdateCache("some-identity"); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Update with empty string
	if err := UpdateCache(""); err != nil {
		t.Fatalf("UpdateCache with empty string failed: %v", err)
	}

	// Read should return empty
	content, err := readCache()
	if err != nil {
		t.Fatalf("ReadCache failed: %v", err)
	}
	if content != "" {
		t.Errorf("Expected empty string, got %q", content)
	}
}

// Test_readCacheTrimsWhitespace verifies whitespace is trimmed from content
func Test_readCacheTrimsWhitespace(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// Write with whitespace directly to file (simulating external modification)
	cachePath, _ := cachePath()
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if err := os.WriteFile(cachePath, []byte("  work  \n"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Read should return trimmed content
	content, err := readCache()
	if err != nil {
		t.Fatalf("ReadCache failed: %v", err)
	}
	if content != "work" {
		t.Errorf("Expected 'work', got %q", content)
	}
}
