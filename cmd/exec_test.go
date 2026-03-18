package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
)

func TestBuildIdentityEnv(t *testing.T) {
	identity := &config.Identity{
		Name:    "work",
		GitName: "Alice Smith",
		Email:   "alice@company.com",
	}

	env := buildIdentityEnv(identity)

	expected := map[string]string{
		"GIT_AUTHOR_NAME":     "Alice Smith",
		"GIT_AUTHOR_EMAIL":    "alice@company.com",
		"GIT_COMMITTER_NAME":  "Alice Smith",
		"GIT_COMMITTER_EMAIL": "alice@company.com",
	}

	for key, want := range expected {
		found := false
		prefix := key + "="
		for _, e := range env {
			if len(e) > len(prefix) && e[:len(prefix)] == prefix {
				got := e[len(prefix):]
				if got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s not found in env", key)
		}
	}
}

func TestBuildIdentityEnvFallbackName(t *testing.T) {
	identity := &config.Identity{
		Name:  "personal",
		Email: "me@example.com",
	}

	env := buildIdentityEnv(identity)

	prefix := "GIT_AUTHOR_NAME="
	for _, e := range env {
		if len(e) > len(prefix) && e[:len(prefix)] == prefix {
			got := e[len(prefix):]
			if got != "personal" {
				t.Errorf("GIT_AUTHOR_NAME = %q, want %q (should fall back to identity Name)", got, "personal")
			}
			return
		}
	}
	t.Error("GIT_AUTHOR_NAME not found in env")
}

func TestBuildIdentityEnvWithSSHKey(t *testing.T) {
	identity := &config.Identity{
		Name:       "work",
		GitName:    "Alice",
		Email:      "alice@company.com",
		SSHKeyPath: "/home/alice/.ssh/id_work",
	}

	env := buildIdentityEnv(identity)

	prefix := "GIT_SSH_COMMAND="
	for _, e := range env {
		if len(e) > len(prefix) && e[:len(prefix)] == prefix {
			got := e[len(prefix):]
			if got == "" {
				t.Error("GIT_SSH_COMMAND is empty, expected ssh command")
			}
			return
		}
	}
	t.Error("GIT_SSH_COMMAND not found in env when SSHKeyPath is set")
}

func TestAppendOrReplace(t *testing.T) {
	env := []string{"HOME=/home/user", "PATH=/usr/bin", "GIT_AUTHOR_NAME=old"}

	env = appendOrReplace(env, "GIT_AUTHOR_NAME", "new")
	found := false
	count := 0
	for _, e := range env {
		if len(e) > len("GIT_AUTHOR_NAME=") && e[:len("GIT_AUTHOR_NAME=")] == "GIT_AUTHOR_NAME=" {
			count++
			if e == "GIT_AUTHOR_NAME=new" {
				found = true
			}
		}
	}
	if !found {
		t.Error("GIT_AUTHOR_NAME was not replaced")
	}
	if count != 1 {
		t.Errorf("GIT_AUTHOR_NAME appears %d times, want 1", count)
	}
}

func TestAppendOrReplaceNew(t *testing.T) {
	env := []string{"HOME=/home/user"}

	env = appendOrReplace(env, "GIT_AUTHOR_NAME", "alice")
	found := false
	for _, e := range env {
		if e == "GIT_AUTHOR_NAME=alice" {
			found = true
		}
	}
	if !found {
		t.Error("GIT_AUTHOR_NAME was not appended")
	}
}
