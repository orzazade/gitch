package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure
type Config struct {
	Default    string     `mapstructure:"default" yaml:"default"`
	Identities []Identity `mapstructure:"identities" yaml:"identities"`
}

// ConfigPath returns the XDG config file path for gitch
func ConfigPath() (string, error) {
	return xdg.ConfigFile("gitch/config.yaml")
}

// Load reads the config from the XDG config file
// Returns an empty Config with nil error if the file doesn't exist
func Load() (*Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine config path: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - return empty config (not an error condition)
			return &Config{
				Identities: []Identity{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Ensure Identities is not nil
	if cfg.Identities == nil {
		cfg.Identities = []Identity{}
	}

	return &cfg, nil
}

// Save writes the config to the XDG config file
func (c *Config) Save() error {
	configPath, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to determine config path: %w", err)
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// findIdentityIndex finds the index of an identity by name (case-insensitive)
// Returns -1 if not found
func (c *Config) findIdentityIndex(name string) int {
	nameLower := strings.ToLower(name)
	for i, identity := range c.Identities {
		if strings.ToLower(identity.Name) == nameLower {
			return i
		}
	}
	return -1
}

// GetIdentity returns an identity by name (case-insensitive)
// Returns an error if the identity is not found
func (c *Config) GetIdentity(name string) (*Identity, error) {
	idx := c.findIdentityIndex(name)
	if idx == -1 {
		return nil, fmt.Errorf("identity %q not found", name)
	}
	return &c.Identities[idx], nil
}

// AddIdentity adds a new identity to the config
// Validates the identity, checks for duplicate names, and warns on duplicate emails
func (c *Config) AddIdentity(identity Identity) error {
	// Validate the identity
	if err := identity.Validate(); err != nil {
		return err
	}

	// Check for duplicate name (case-insensitive)
	if c.findIdentityIndex(identity.Name) != -1 {
		return fmt.Errorf("identity with name %q already exists", identity.Name)
	}

	// Check for duplicate email (warn but allow)
	for _, existing := range c.Identities {
		if strings.EqualFold(existing.Email, identity.Email) {
			fmt.Fprintf(os.Stderr, "Warning: email %q is already used by identity %q\n", identity.Email, existing.Name)
			break
		}
	}

	c.Identities = append(c.Identities, identity)
	return nil
}

// DeleteIdentity removes an identity by name (case-insensitive)
// Returns an error if the identity is not found
// Clears the default if the deleted identity was the default
func (c *Config) DeleteIdentity(name string) error {
	idx := c.findIdentityIndex(name)
	if idx == -1 {
		return fmt.Errorf("identity %q not found", name)
	}

	// Check if this is the default identity
	if strings.EqualFold(c.Default, c.Identities[idx].Name) {
		c.Default = ""
	}

	// Remove the identity
	c.Identities = append(c.Identities[:idx], c.Identities[idx+1:]...)
	return nil
}

// ListIdentities returns all identities
// Returns an empty slice if there are no identities
func (c *Config) ListIdentities() []Identity {
	return c.Identities
}

// SetDefault sets the default identity
// Returns an error if the identity doesn't exist
func (c *Config) SetDefault(name string) error {
	idx := c.findIdentityIndex(name)
	if idx == -1 {
		return fmt.Errorf("identity %q not found", name)
	}

	// Use the actual stored name (preserves original case)
	c.Default = c.Identities[idx].Name
	return nil
}

// ErrIdentityNotFound is returned when an identity is not found
var ErrIdentityNotFound = errors.New("identity not found")
