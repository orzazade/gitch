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

func TestUninstallLocal_RemovesManagedHook(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	// Install first
	if err := InstallLocal(); err != nil {
		t.Fatalf("InstallLocal failed: %v", err)
	}

	// Uninstall
	if err := UninstallLocal(); err != nil {
		t.Fatalf("UninstallLocal failed: %v", err)
	}

	// Verify hook is gone
	hookPath, err := LocalHookPath()
	if err != nil {
		t.Fatalf("failed to get local hook path: %v", err)
	}
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("expected hook file to be removed after UninstallLocal")
	}
}

func TestIsInstalledLocal_NotInstalled(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	installed, err := IsInstalledLocal()
	if err != nil {
		t.Fatalf("IsInstalledLocal failed: %v", err)
	}
	if installed {
		t.Error("expected IsInstalledLocal=false when no hook installed")
	}
}

func TestIsInstalledLocal_Installed(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	if err := InstallLocal(); err != nil {
		t.Fatalf("InstallLocal failed: %v", err)
	}

	installed, err := IsInstalledLocal()
	if err != nil {
		t.Fatalf("IsInstalledLocal failed: %v", err)
	}
	if !installed {
		t.Error("expected IsInstalledLocal=true after install")
	}
}

func TestInstallGlobal_Success(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	if err := InstallGlobal(); err != nil {
		t.Fatalf("InstallGlobal failed: %v", err)
	}

	// Verify core.hooksPath was set
	hooksPath, err := git.GetConfigScoped("core.hooksPath", git.ScopeGlobal)
	if err != nil {
		t.Fatalf("failed to get core.hooksPath: %v", err)
	}
	if hooksPath == "" {
		t.Fatal("core.hooksPath should be set after InstallGlobal")
	}

	// Verify hook file exists
	hookFile := filepath.Join(hooksPath, "pre-commit")
	data, err := os.ReadFile(hookFile)
	if err != nil {
		t.Fatalf("failed to read hook file: %v", err)
	}
	if !strings.Contains(string(data), managedHookMarker) {
		t.Error("global hook should contain managed marker")
	}
}

func TestInstallGlobal_Idempotent(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	// Install twice — second call should succeed (our own hooks path)
	if err := InstallGlobal(); err != nil {
		t.Fatalf("first InstallGlobal failed: %v", err)
	}
	if err := InstallGlobal(); err != nil {
		t.Fatalf("second InstallGlobal should be idempotent: %v", err)
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

func TestUninstallGlobal_AfterInstall(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	// Install first
	if err := InstallGlobal(); err != nil {
		t.Fatalf("InstallGlobal failed: %v", err)
	}

	// Uninstall
	if err := UninstallGlobal(); err != nil {
		t.Fatalf("UninstallGlobal failed: %v", err)
	}

	// Verify core.hooksPath is cleared
	hooksPath, err := git.GetConfigScoped("core.hooksPath", git.ScopeGlobal)
	if err != nil {
		t.Fatalf("failed to get core.hooksPath: %v", err)
	}
	if hooksPath != "" {
		t.Errorf("core.hooksPath should be empty after uninstall, got %q", hooksPath)
	}
}

func TestUninstallGlobal_NotInstalled(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	// Uninstall when nothing is installed — should not error
	if err := UninstallGlobal(); err != nil {
		t.Fatalf("UninstallGlobal should be safe when not installed: %v", err)
	}
}

func TestIsInstalled_NotInstalled(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	installed, err := IsInstalled()
	if err != nil {
		t.Fatalf("IsInstalled failed: %v", err)
	}
	if installed {
		t.Error("expected IsInstalled=false when not installed")
	}
}

func TestIsInstalled_AfterInstall(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	if err := InstallGlobal(); err != nil {
		t.Fatalf("InstallGlobal failed: %v", err)
	}

	installed, err := IsInstalled()
	if err != nil {
		t.Fatalf("IsInstalled failed: %v", err)
	}
	if !installed {
		t.Error("expected IsInstalled=true after InstallGlobal")
	}
}
