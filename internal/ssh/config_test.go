package ssh

import (
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/config"
)

func TestHostConfig_String(t *testing.T) {
	host := HostConfig{
		Alias:        "github-work",
		HostName:     "github.com",
		User:         "git",
		IdentityFile: "/home/user/.ssh/key",
	}

	result := host.String()

	// Check all expected components
	if !strings.Contains(result, "Host github-work") {
		t.Errorf("Expected output to contain 'Host github-work', got:\n%s", result)
	}
	if !strings.Contains(result, "HostName github.com") {
		t.Errorf("Expected output to contain 'HostName github.com', got:\n%s", result)
	}
	if !strings.Contains(result, "User git") {
		t.Errorf("Expected output to contain 'User git', got:\n%s", result)
	}
	if !strings.Contains(result, "IdentityFile /home/user/.ssh/key") {
		t.Errorf("Expected output to contain 'IdentityFile /home/user/.ssh/key', got:\n%s", result)
	}
	if !strings.Contains(result, "IdentitiesOnly yes") {
		t.Errorf("Expected output to contain 'IdentitiesOnly yes', got:\n%s", result)
	}

	// Check indentation (4 spaces)
	if !strings.Contains(result, "    HostName") {
		t.Errorf("Expected 4-space indentation for HostName")
	}
}

func TestGenerateConfigBlock_Empty(t *testing.T) {
	result := GenerateConfigBlock([]HostConfig{})

	if result != "" {
		t.Errorf("Expected empty string for empty hosts, got: %s", result)
	}
}

func TestGenerateConfigBlock_NilSlice(t *testing.T) {
	result := GenerateConfigBlock(nil)

	if result != "" {
		t.Errorf("Expected empty string for nil hosts, got: %s", result)
	}
}

func TestGenerateConfigBlock_SingleHost(t *testing.T) {
	hosts := []HostConfig{
		{
			Alias:        "github-personal",
			HostName:     "github.com",
			User:         "git",
			IdentityFile: "/home/user/.ssh/personal",
		},
	}

	result := GenerateConfigBlock(hosts)

	// Check markers
	if !strings.HasPrefix(result, MarkerStart) {
		t.Errorf("Expected output to start with MarkerStart, got:\n%s", result)
	}
	if !strings.Contains(result, MarkerEnd) {
		t.Errorf("Expected output to contain MarkerEnd, got:\n%s", result)
	}

	// Check host block is included
	if !strings.Contains(result, "Host github-personal") {
		t.Errorf("Expected output to contain host block, got:\n%s", result)
	}
}

func TestGenerateConfigBlock_MultipleHosts(t *testing.T) {
	hosts := []HostConfig{
		{
			Alias:        "github-work",
			HostName:     "github.com",
			User:         "git",
			IdentityFile: "/home/user/.ssh/work",
		},
		{
			Alias:        "gitlab-work",
			HostName:     "gitlab.com",
			User:         "git",
			IdentityFile: "/home/user/.ssh/work",
		},
	}

	result := GenerateConfigBlock(hosts)

	// Both hosts should be present
	if !strings.Contains(result, "Host github-work") {
		t.Errorf("Expected github-work host in output")
	}
	if !strings.Contains(result, "Host gitlab-work") {
		t.Errorf("Expected gitlab-work host in output")
	}
}

func TestRemoveManagedBlock_NoMarkers(t *testing.T) {
	content := `Host personal
    HostName github.com
    User git
`

	result := removeManagedBlock(content)

	if result != content {
		t.Errorf("Expected content unchanged when no markers, got:\n%s", result)
	}
}

func TestRemoveManagedBlock_WithMarkers(t *testing.T) {
	content := `Host personal
    HostName github.com
    User git

# gitch:start - MANAGED BY GITCH, DO NOT EDIT
Host github-work
    HostName github.com
    User git
    IdentityFile /home/user/.ssh/work
    IdentitiesOnly yes

# gitch:end

Host other
    HostName other.com
`

	result := removeManagedBlock(content)

	// Should not contain gitch markers
	if strings.Contains(result, "gitch:start") {
		t.Errorf("Expected gitch:start to be removed")
	}
	if strings.Contains(result, "gitch:end") {
		t.Errorf("Expected gitch:end to be removed")
	}
	if strings.Contains(result, "github-work") {
		t.Errorf("Expected github-work block to be removed")
	}

	// Should preserve surrounding content
	if !strings.Contains(result, "Host personal") {
		t.Errorf("Expected 'Host personal' to be preserved")
	}
	if !strings.Contains(result, "Host other") {
		t.Errorf("Expected 'Host other' to be preserved")
	}
}

func TestRemoveManagedBlock_MalformedOnlyStart(t *testing.T) {
	content := `Host personal
    HostName github.com

# gitch:start - MANAGED BY GITCH, DO NOT EDIT
Host github-work
    HostName github.com
`

	result := removeManagedBlock(content)

	// Should return unchanged for safety when malformed
	if result != content {
		t.Errorf("Expected content unchanged when only start marker present, got:\n%s", result)
	}
}

func TestRemoveManagedBlock_BlockAtEnd(t *testing.T) {
	content := `Host personal
    HostName github.com

# gitch:start - MANAGED BY GITCH, DO NOT EDIT
Host github-work
    HostName github.com
    User git
    IdentityFile /path/to/key
    IdentitiesOnly yes

# gitch:end
`

	result := removeManagedBlock(content)

	if strings.Contains(result, "gitch:start") {
		t.Errorf("Expected gitch block to be removed")
	}
	if !strings.Contains(result, "Host personal") {
		t.Errorf("Expected 'Host personal' to be preserved")
	}
}

func TestIdentityToHosts_NoSSHKey(t *testing.T) {
	identity := config.Identity{
		Name:       "work",
		Email:      "work@example.com",
		SSHKeyPath: "",
	}

	result := IdentityToHosts(identity)

	if result != nil {
		t.Errorf("Expected nil for identity without SSH key, got: %v", result)
	}
}

func TestIdentityToHosts_WithSSHKey(t *testing.T) {
	identity := config.Identity{
		Name:       "work",
		Email:      "work@example.com",
		SSHKeyPath: "/home/user/.ssh/work_key",
	}

	result := IdentityToHosts(identity)

	if len(result) != 2 {
		t.Fatalf("Expected 2 hosts, got %d", len(result))
	}

	// Check GitHub host
	githubHost := result[0]
	if githubHost.Alias != "github-work" {
		t.Errorf("Expected github alias 'github-work', got %s", githubHost.Alias)
	}
	if githubHost.HostName != "github.com" {
		t.Errorf("Expected hostname 'github.com', got %s", githubHost.HostName)
	}
	if githubHost.User != "git" {
		t.Errorf("Expected user 'git', got %s", githubHost.User)
	}
	if githubHost.IdentityFile != "/home/user/.ssh/work_key" {
		t.Errorf("Expected identity file '/home/user/.ssh/work_key', got %s", githubHost.IdentityFile)
	}

	// Check GitLab host
	gitlabHost := result[1]
	if gitlabHost.Alias != "gitlab-work" {
		t.Errorf("Expected gitlab alias 'gitlab-work', got %s", gitlabHost.Alias)
	}
	if gitlabHost.HostName != "gitlab.com" {
		t.Errorf("Expected hostname 'gitlab.com', got %s", gitlabHost.HostName)
	}
}

func TestIdentityToHosts_TildeExpansion(t *testing.T) {
	identity := config.Identity{
		Name:       "personal",
		Email:      "personal@example.com",
		SSHKeyPath: "~/.ssh/personal_key",
	}

	result := IdentityToHosts(identity)

	if len(result) != 2 {
		t.Fatalf("Expected 2 hosts, got %d", len(result))
	}

	// IdentityFile should be expanded (not start with ~)
	if strings.HasPrefix(result[0].IdentityFile, "~") {
		t.Errorf("Expected tilde to be expanded, got: %s", result[0].IdentityFile)
	}
}

func TestSSHConfigPath(t *testing.T) {
	path, err := SSHConfigPath()
	if err != nil {
		t.Fatalf("SSHConfigPath returned error: %v", err)
	}

	if !strings.HasSuffix(path, ".ssh/config") {
		t.Errorf("Expected path to end with '.ssh/config', got: %s", path)
	}
}
