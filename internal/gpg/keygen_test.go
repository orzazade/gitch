package gpg

import (
	"os"
	"strings"
	"testing"
)

func Test_defaultKeyPath_Format(t *testing.T) {
	path := defaultKeyPath("work")

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
	if !strings.Contains(path, ".gnupg") {
		t.Errorf("expected path to contain .gnupg, got %q", path)
	}
	if !strings.HasSuffix(path, "gitch-work.asc") {
		t.Errorf("expected path to end with gitch-work.asc, got %q", path)
	}
}

func Test_defaultKeyPath_DifferentNames(t *testing.T) {
	path1 := defaultKeyPath("personal")
	path2 := defaultKeyPath("work")

	if path1 == "" || path2 == "" {
		t.Skip("could not determine home directory")
	}

	if path1 == path2 {
		t.Error("different identity names should produce different paths")
	}
	if !strings.HasSuffix(path1, "gitch-personal.asc") {
		t.Errorf("expected path to end with gitch-personal.asc, got %q", path1)
	}
}
