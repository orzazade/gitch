package ssh

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultSSHKeyPath_Format(t *testing.T) {
	path := DefaultSSHKeyPath("work")

	if path == "" {
		t.Skip("could not determine home directory")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("could not determine home directory")
	}

	if !strings.HasPrefix(path, home) {
		t.Errorf("expected path to start with home dir %q, got %q", home, path)
	}
	if !strings.Contains(path, ".ssh") {
		t.Errorf("expected path to contain .ssh, got %q", path)
	}
	if !strings.HasSuffix(path, "gitch_work_ed25519") {
		t.Errorf("expected path to end with gitch_work_ed25519, got %q", path)
	}
}

func TestDefaultSSHKeyPath_DifferentNames(t *testing.T) {
	path1 := DefaultSSHKeyPath("personal")
	path2 := DefaultSSHKeyPath("work")

	if path1 == "" || path2 == "" {
		t.Skip("could not determine home directory")
	}

	if path1 == path2 {
		t.Error("different identity names should produce different paths")
	}
}

func TestExpandPath_Tilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("could not determine home directory")
	}

	path, err := ExpandPath("~/test")
	if err != nil {
		t.Fatalf("ExpandPath failed: %v", err)
	}
	if !strings.HasPrefix(path, home) {
		t.Errorf("expected tilde expansion to home dir, got %q", path)
	}
}

func TestExpandPath_TildeOnly(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("could not determine home directory")
	}

	path, err := ExpandPath("~")
	if err != nil {
		t.Fatalf("ExpandPath failed: %v", err)
	}
	if path != home {
		t.Errorf("expected %q, got %q", home, path)
	}
}

func TestExpandPath_Empty(t *testing.T) {
	_, err := ExpandPath("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestExpandPath_EnvVar(t *testing.T) {
	t.Setenv("GITCH_TEST_DIR", "/opt/keys")

	path, err := ExpandPath("$GITCH_TEST_DIR/mykey")
	if err != nil {
		t.Fatalf("ExpandPath failed: %v", err)
	}
	if path != "/opt/keys/mykey" {
		t.Errorf("expected /opt/keys/mykey, got %q", path)
	}
}

func TestExpandPath_AbsolutePath(t *testing.T) {
	path, err := ExpandPath("/tmp/test")
	if err != nil {
		t.Fatalf("ExpandPath failed: %v", err)
	}
	if path != "/tmp/test" {
		t.Errorf("expected /tmp/test, got %q", path)
	}
}
