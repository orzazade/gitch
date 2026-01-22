package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUpdateCache verifies that UpdateCache writes to file and can be read back
func TestUpdateCache(t *testing.T) {
	// Use temp directory for test isolation
	tmpDir := t.TempDir()

	// Override XDG cache path for testing
	origXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", origXDG)

	// Test writing identity name
	identityName := "work"
	if err := UpdateCache(identityName); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}

	// Verify file exists
	cachePath, _ := CachePath()
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file was not created")
	}

	// Read back and verify content
	content, err := ReadCache()
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

	origXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", origXDG)

	// Create cache file first
	if err := UpdateCache("test-identity"); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Clear it
	if err := ClearCache(); err != nil {
		t.Fatalf("ClearCache failed: %v", err)
	}

	// Verify file is gone
	cachePath, _ := CachePath()
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("Cache file should not exist after ClearCache")
	}
}

// TestReadCacheMissing verifies ReadCache returns empty string for non-existent file
func TestReadCacheMissing(t *testing.T) {
	tmpDir := t.TempDir()

	origXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", origXDG)

	// Don't create any file - just read
	content, err := ReadCache()
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

	origXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", origXDG)

	// Clear without creating - should not error
	if err := ClearCache(); err != nil {
		t.Fatalf("ClearCache should not error for missing file: %v", err)
	}
}

// TestAtomicWrite verifies the temp file is cleaned up after successful write
func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()

	origXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", origXDG)

	// Write to cache
	if err := UpdateCache("atomic-test"); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}

	// Verify .tmp file doesn't exist
	cachePath, _ := CachePath()
	tmpPath := cachePath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error(".tmp file should be cleaned up after successful write")
	}

	// Verify main file exists with correct content
	content, err := ReadCache()
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

	origXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", origXDG)

	// Create with content first
	if err := UpdateCache("some-identity"); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Update with empty string
	if err := UpdateCache(""); err != nil {
		t.Fatalf("UpdateCache with empty string failed: %v", err)
	}

	// Read should return empty
	content, err := ReadCache()
	if err != nil {
		t.Fatalf("ReadCache failed: %v", err)
	}
	if content != "" {
		t.Errorf("Expected empty string, got %q", content)
	}
}

// TestReadCacheTrimsWhitespace verifies whitespace is trimmed from content
func TestReadCacheTrimsWhitespace(t *testing.T) {
	tmpDir := t.TempDir()

	origXDG := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Setenv("XDG_CACHE_HOME", origXDG)

	// Write with whitespace directly to file (simulating external modification)
	cachePath, _ := CachePath()
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if err := os.WriteFile(cachePath, []byte("  work  \n"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Read should return trimmed content
	content, err := ReadCache()
	if err != nil {
		t.Fatalf("ReadCache failed: %v", err)
	}
	if content != "work" {
		t.Errorf("Expected 'work', got %q", content)
	}
}
