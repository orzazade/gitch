package config

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

// MaxNameLength is the maximum allowed length for an identity name
const MaxNameLength = 50

// HookMode constants define how the pre-commit hook behaves for an identity
const (
	HookModeAllow = "allow" // Always allow commits (no warning)
	HookModeWarn  = "warn"  // Warn but allow commit (default)
	HookModeBlock = "block" // Block commits until identity matches
)

// Identity represents a git identity with name and email
type Identity struct {
	Name       string `mapstructure:"name" yaml:"name"`
	Email      string `mapstructure:"email" yaml:"email"`
	SSHKeyPath string `mapstructure:"ssh_key_path" yaml:"ssh_key_path,omitempty"`
	GPGKeyID   string `mapstructure:"gpg_key_id" yaml:"gpg_key_id,omitempty"`
	HookMode   string `mapstructure:"hook_mode" yaml:"hook_mode,omitempty"`
}

// ValidateHookMode validates that the hook mode is a valid value
func ValidateHookMode(mode string) error {
	switch mode {
	case HookModeAllow, HookModeWarn, HookModeBlock:
		return nil
	case "":
		return nil // Empty is valid, defaults to warn
	default:
		return fmt.Errorf("invalid hook mode %q: must be one of: allow, warn, block", mode)
	}
}

// GetHookMode returns the hook mode for the identity, defaulting to warn if not set
func (i *Identity) GetHookMode() string {
	if i.HookMode == "" {
		return HookModeWarn
	}
	return i.HookMode
}

// nameRegex validates identity names: alphanumeric + hyphens, no leading/trailing hyphens
var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// singleCharRegex validates single character names
var singleCharRegex = regexp.MustCompile(`^[a-zA-Z0-9]$`)

// ValidateName validates an identity name according to the naming rules:
// - Alphanumeric characters and hyphens only
// - No leading or trailing hyphens
// - Maximum 50 characters
// - Single character names allowed
func ValidateName(name string) error {
	if name == "" {
		return errors.New("identity name cannot be empty")
	}

	if len(name) > MaxNameLength {
		return fmt.Errorf("identity name cannot exceed %d characters", MaxNameLength)
	}

	// Single character names are valid
	if len(name) == 1 {
		if !singleCharRegex.MatchString(name) {
			return errors.New("identity name must contain only alphanumeric characters and hyphens")
		}
		return nil
	}

	// Multi-character names must match the pattern
	if !nameRegex.MatchString(name) {
		if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
			return errors.New("identity name cannot start or end with a hyphen")
		}
		return errors.New("identity name must contain only alphanumeric characters and hyphens")
	}

	return nil
}

// ValidateEmail validates an email address using RFC 5322 parsing
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email cannot be empty")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}

	return nil
}

// Validate validates both name and email of the identity
func (i *Identity) Validate() error {
	if err := ValidateName(i.Name); err != nil {
		return err
	}

	if err := ValidateEmail(i.Email); err != nil {
		return err
	}

	return nil
}
