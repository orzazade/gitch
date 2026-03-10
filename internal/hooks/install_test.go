package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestInstallLocal_WritesManagedHook(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	if err := InstallLocal(); err != nil {
		t.Fatalf("InstallLocal failed: %v", err)
	}

	hookPath, err := LocalHookPath()
	if err != nil {
		t.Fatalf("failed to get local hook path: %v", err)
	}

	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read hook file: %v", err)
	}
	if !strings.Contains(string(data), managedHookMarker) {
		t.Fatalf("expected hook to contain managed marker, got:\n%s", string(data))
	}
}

func TestInstallLocal_RefusesOverwriteNonManagedHook(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	hookPath, err := LocalHookPath()
	if err != nil {
		t.Fatalf("failed to get local hook path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(hookPath), 0755); err != nil {
		t.Fatalf("failed to create hook dir: %v", err)
	}
	if err := os.WriteFile(hookPath, []byte("#!/bin/bash\necho custom\n"), 0755); err != nil {
		t.Fatalf("failed to seed custom hook: %v", err)
	}

	if err := InstallLocal(); err == nil {
		t.Fatal("expected InstallLocal to refuse overwriting a non-managed hook")
	}
}

func TestInstallGlobal_RefusesOverwriteExistingHooksPath(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	customHooksPath := filepath.Join(env.Dir, "custom-hooks")
	if err := os.MkdirAll(customHooksPath, 0755); err != nil {
		t.Fatalf("failed to create custom hooks dir: %v", err)
	}
	if err := git.SetConfigScoped("core.hooksPath", customHooksPath, git.ScopeGlobal); err != nil {
		t.Fatalf("failed to seed global hooksPath: %v", err)
	}

	if err := InstallGlobal(); err == nil {
		t.Fatal("expected InstallGlobal to refuse overwriting custom core.hooksPath")
	}
}
