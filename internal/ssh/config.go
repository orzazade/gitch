package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orzazade/gitch/internal/config"
)

// SSH config marker constants for identifying gitch-managed blocks
const (
	MarkerStart = "# gitch:start - MANAGED BY GITCH, DO NOT EDIT"
	MarkerEnd   = "# gitch:end"
)

// HostConfig represents an SSH Host block configuration
type HostConfig struct {
	Alias        string
	HostName     string
	User         string
	IdentityFile string
}

// String generates an SSH config Host block from the HostConfig
func (h HostConfig) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Host %s\n", h.Alias))
	sb.WriteString(fmt.Sprintf("    HostName %s\n", h.HostName))
	sb.WriteString(fmt.Sprintf("    User %s\n", h.User))
	sb.WriteString(fmt.Sprintf("    IdentityFile %s\n", h.IdentityFile))
	sb.WriteString("    IdentitiesOnly yes\n")
	return sb.String()
}

// GenerateConfigBlock wraps host configurations in gitch markers
// Returns empty string if hosts slice is empty
func GenerateConfigBlock(hosts []HostConfig) string {
	if len(hosts) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(MarkerStart)
	sb.WriteString("\n")

	for _, host := range hosts {
		sb.WriteString(host.String())
		sb.WriteString("\n")
	}

	sb.WriteString(MarkerEnd)
	sb.WriteString("\n")

	return sb.String()
}

// IdentityToHosts converts a config.Identity to SSH HostConfigs
// Returns nil if the identity has no SSH key configured
// Generates hosts for both github.com and gitlab.com
func IdentityToHosts(identity config.Identity) []HostConfig {
	if identity.SSHKeyPath == "" {
		return nil
	}

	// Expand the SSH key path
	expandedPath, err := ExpandPath(identity.SSHKeyPath)
	if err != nil {
		// If expansion fails, use the original path
		expandedPath = identity.SSHKeyPath
	}

	return []HostConfig{
		{
			Alias:        fmt.Sprintf("github-%s", identity.Name),
			HostName:     "github.com",
			User:         "git",
			IdentityFile: expandedPath,
		},
		{
			Alias:        fmt.Sprintf("gitlab-%s", identity.Name),
			HostName:     "gitlab.com",
			User:         "git",
			IdentityFile: expandedPath,
		},
	}
}

// removeManagedBlock removes the gitch-managed block from SSH config content
// Returns content unchanged if markers are not found or malformed
func removeManagedBlock(content string) string {
	startIdx := strings.Index(content, MarkerStart)
	if startIdx == -1 {
		return content
	}

	endIdx := strings.Index(content, MarkerEnd)
	if endIdx == -1 {
		// Malformed - only start marker, no end marker
		// Return unchanged for safety
		return content
	}

	// Remove from start marker to end of end marker
	endOfBlock := endIdx + len(MarkerEnd)

	// Remove trailing newlines after the block (up to 2)
	newlinesRemoved := 0
	for endOfBlock < len(content) && content[endOfBlock] == '\n' && newlinesRemoved < 2 {
		endOfBlock++
		newlinesRemoved++
	}

	return content[:startIdx] + content[endOfBlock:]
}

// UpdateSSHConfig updates the user's SSH config with the new gitch block
// Creates backup before modification and writes atomically
func UpdateSSHConfig(newBlock string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	sshDir := filepath.Join(home, ".ssh")
	configPath := filepath.Join(sshDir, "config")

	// Ensure .ssh directory exists with proper permissions
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Read existing content
	existingContent := ""
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read SSH config: %w", err)
		}
		// File doesn't exist - that's ok
	} else {
		existingContent = string(data)

		// Create backup if file has content
		if len(existingContent) > 0 {
			backupPath := configPath + ".gitch.backup"
			if err := os.WriteFile(backupPath, data, 0600); err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}
		}
	}

	// Remove old managed block
	cleanedContent := removeManagedBlock(existingContent)

	// Trim trailing whitespace/newlines from existing content
	cleanedContent = strings.TrimRight(cleanedContent, "\n\t ")

	// Build new content
	var finalContent string
	if cleanedContent == "" {
		finalContent = newBlock
	} else {
		finalContent = cleanedContent + "\n\n" + newBlock
	}

	// Write to temp file first (atomic write)
	tempPath := configPath + ".tmp"
	if err := os.WriteFile(tempPath, []byte(finalContent), 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename temp to actual config (atomic operation)
	if err := os.Rename(tempPath, configPath); err != nil {
		// Clean up temp file on rename failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to update SSH config: %w", err)
	}

	return nil
}

// SSHConfigPath returns the default SSH config path
func SSHConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "config"), nil
}
