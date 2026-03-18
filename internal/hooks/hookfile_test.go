package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsManagedHook_ManagedFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "pre-commit")
	if err := os.WriteFile(path, []byte(preCommitScript), 0755); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	managed, err := isManagedHook(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !managed {
		t.Error("expected managed=true for gitch hook script")
	}
}

func TestIsManagedHook_NonManagedFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "pre-commit")
	if err := os.WriteFile(path, []byte("#!/bin/bash\necho custom\n"), 0755); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	managed, err := isManagedHook(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if managed {
		t.Error("expected managed=false for non-gitch hook")
	}
}

func TestIsManagedHook_NonExistent(t *testing.T) {
	managed, err := isManagedHook(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if managed {
		t.Error("expected managed=false for nonexistent file")
	}
}

func TestWriteManagedHook_CreatesExecutableFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "pre-commit")

	if err := writeManagedHook(path); err != nil {
		t.Fatalf("writeManagedHook failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written hook: %v", err)
	}
	if string(data) != preCommitScript {
		t.Error("written content does not match preCommitScript")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat hook: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Error("hook file should be executable")
	}
}

func TestRemoveManagedHook_RemovesManagedFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "pre-commit")

	if err := writeManagedHook(path); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if err := removeManagedHook(path); err != nil {
		t.Fatalf("removeManagedHook failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected hook file to be removed")
	}
}

func TestRemoveManagedHook_RefusesNonManaged(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "pre-commit")
	if err := os.WriteFile(path, []byte("#!/bin/bash\necho custom\n"), 0755); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if err := removeManagedHook(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should still exist — not managed, so not removed
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("non-managed hook should not be removed")
	}
}

func TestRemoveManagedHook_NonExistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent")
	if err := removeManagedHook(path); err != nil {
		t.Fatalf("removeManagedHook on nonexistent file should not error: %v", err)
	}
}

func TestEnsureHookPathWritable_NoExistingHook(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "hooks", "pre-commit")

	if err := ensureHookPathWritable(path); err != nil {
		t.Fatalf("ensureHookPathWritable failed: %v", err)
	}

	// Parent directory should have been created
	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		t.Error("expected parent directory to be created")
	}
}

func TestEnsureHookPathWritable_ExistingManagedHook(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "pre-commit")

	// Write a managed hook first
	if err := writeManagedHook(path); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Should succeed — managed hook can be overwritten
	if err := ensureHookPathWritable(path); err != nil {
		t.Fatalf("ensureHookPathWritable should allow overwriting managed hook: %v", err)
	}
}

func TestEnsureHookPathWritable_ExistingNonManagedHook(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "pre-commit")
	if err := os.WriteFile(path, []byte("#!/bin/bash\necho custom\n"), 0755); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	err := ensureHookPathWritable(path)
	if err == nil {
		t.Fatal("expected error for existing non-managed hook")
	}
}
