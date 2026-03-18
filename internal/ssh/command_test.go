package ssh

import (
	"strings"
	"testing"
)

func TestShellQuote(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", "''"},
		{"/home/user/.ssh/id_ed25519", "'/home/user/.ssh/id_ed25519'"},
		{"/path/with space/key", "'/path/with space/key'"},
		{"it's here", "'it'\"'\"'s here'"},
		{"/normal/path", "'/normal/path'"},
	}
	for _, c := range cases {
		got := shellQuote(c.input)
		if got != c.want {
			t.Errorf("shellQuote(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestGitSSHCommand_ContainsKeyPath(t *testing.T) {
	cmd, err := GitSSHCommand("/home/user/.ssh/id_ed25519")
	if err != nil {
		t.Fatalf("GitSSHCommand failed: %v", err)
	}
	if !strings.HasPrefix(cmd, "ssh -i ") {
		t.Errorf("expected command to start with 'ssh -i ', got: %q", cmd)
	}
	if !strings.Contains(cmd, "IdentitiesOnly=yes") {
		t.Errorf("expected 'IdentitiesOnly=yes' in command, got: %q", cmd)
	}
	if !strings.Contains(cmd, "id_ed25519") {
		t.Errorf("expected key filename in command, got: %q", cmd)
	}
}

func TestGitSSHCommand_QuotesPath(t *testing.T) {
	cmd, err := GitSSHCommand("/home/user/.ssh/my key")
	if err != nil {
		t.Fatalf("GitSSHCommand failed: %v", err)
	}
	// Path with space should be quoted
	if !strings.Contains(cmd, "'") {
		t.Errorf("expected quoted path for path with space, got: %q", cmd)
	}
}
