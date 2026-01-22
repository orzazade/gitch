package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/orzazade/gitch/internal/rules"
	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure
type Config struct {
	Default    string       `mapstructure:"default" yaml:"default"`
	Identities []Identity   `mapstructure:"identities" yaml:"identities"`
	Rules      []rules.Rule `mapstructure:"rules" yaml:"rules,omitempty"`
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

	// Ensure Rules is not nil
	if cfg.Rules == nil {
		cfg.Rules = []rules.Rule{}
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

// ErrRuleNotFound is returned when a rule is not found
var ErrRuleNotFound = errors.New("rule not found")

// AddRule adds a new rule to the config
// Validates the rule and checks for exact duplicates
func (c *Config) AddRule(rule rules.Rule) error {
	// Validate the rule pattern
	if err := rule.ValidatePattern(); err != nil {
		return err
	}

	// Check for exact duplicate (same pattern)
	for _, existing := range c.Rules {
		if existing.Pattern == rule.Pattern {
			return fmt.Errorf("rule with pattern %q already exists", rule.Pattern)
		}
	}

	c.Rules = append(c.Rules, rule)
	return nil
}

// RemoveRule removes a rule by pattern (exact match)
// Returns an error if the rule is not found
func (c *Config) RemoveRule(pattern string) error {
	for i, rule := range c.Rules {
		if rule.Pattern == pattern {
			c.Rules = append(c.Rules[:i], c.Rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("rule with pattern %q not found", pattern)
}

// ListRules returns all rules
func (c *Config) ListRules() []rules.Rule {
	return c.Rules
}

// FindOverlappingRules returns rules that might conflict with the new rule
// For directory rules: checks if patterns share a common prefix or one is a subset of another
// For remote rules: checks if patterns share the same host and overlapping org/repo paths
func (c *Config) FindOverlappingRules(newRule rules.Rule) []rules.Rule {
	var overlapping []rules.Rule

	for _, existing := range c.Rules {
		// Only compare rules of the same type
		if existing.Type != newRule.Type {
			continue
		}

		// Skip exact duplicates (handled separately)
		if existing.Pattern == newRule.Pattern {
			continue
		}

		if newRule.Type == rules.DirectoryRule {
			// For directory rules, check for prefix overlap
			if isDirectoryOverlap(existing.Pattern, newRule.Pattern) {
				overlapping = append(overlapping, existing)
			}
		} else if newRule.Type == rules.RemoteRule {
			// For remote rules, check for host/org overlap
			if isRemoteOverlap(existing.Pattern, newRule.Pattern) {
				overlapping = append(overlapping, existing)
			}
		}
	}

	return overlapping
}

// isDirectoryOverlap checks if two directory patterns might overlap
func isDirectoryOverlap(pattern1, pattern2 string) bool {
	// Normalize patterns by removing trailing wildcards for prefix comparison
	p1 := strings.TrimSuffix(strings.TrimSuffix(pattern1, "/**"), "/*")
	p2 := strings.TrimSuffix(strings.TrimSuffix(pattern2, "/**"), "/*")

	// Check if one is a prefix of the other
	return strings.HasPrefix(p1, p2) || strings.HasPrefix(p2, p1)
}

// isRemoteOverlap checks if two remote patterns might overlap
func isRemoteOverlap(pattern1, pattern2 string) bool {
	// Split patterns into host and path
	parts1 := strings.SplitN(pattern1, "/", 2)
	parts2 := strings.SplitN(pattern2, "/", 2)

	// If different hosts, no overlap
	if parts1[0] != parts2[0] {
		return false
	}

	// Same host - check path overlap
	if len(parts1) < 2 || len(parts2) < 2 {
		return true // One pattern is just the host, overlaps with all on that host
	}

	path1 := strings.TrimSuffix(parts1[1], "/*")
	path2 := strings.TrimSuffix(parts2[1], "/*")

	// Check if one path is a prefix of the other
	return strings.HasPrefix(path1, path2) || strings.HasPrefix(path2, path1)
}
