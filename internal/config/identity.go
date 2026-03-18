package config

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

// maxNameLength is the maximum allowed length for an identity name
const maxNameLength = 50

// hookMode constants define how the pre-commit hook behaves for an identity.
const (
	hookModeAllow = "allow" // Always allow commits (no warning)
	hookModeWarn  = "warn"  // Warn but allow commit (default)
	hookModeBlock = "block" // Block commits until identity matches
)

// Identity represents a git identity with name and email
type Identity struct {
	Name       string `yaml:"name"`
	GitName    string `yaml:"git_name,omitempty"`
	Email      string `yaml:"email"`
	SSHKeyPath string `yaml:"ssh_key_path,omitempty"`
	GPGKeyID   string `yaml:"gpg_key_id,omitempty"`
	HookMode   string `yaml:"hook_mode,omitempty"`
}

// validateHookMode validates that the hook mode is a valid value
func validateHookMode(mode string) error {
	switch mode {
	case hookModeAllow, hookModeWarn, hookModeBlock, "":
		return nil
	default:
		return fmt.Errorf("invalid hook mode %q: must be one of: allow, warn, block", mode)
	}
}

// GetHookMode returns the hook mode for the identity, defaulting to warn if not set
func (i *Identity) GetHookMode() string {
	if i.HookMode == "" {
		return hookModeWarn
	}
	return i.HookMode
}

// nameRegex validates identity names: alphanumeric + hyphens, no leading/trailing hyphens
var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// ValidateName validates an identity name according to the naming rules:
// - Alphanumeric characters and hyphens only
// - No leading or trailing hyphens
// - Maximum 50 characters
// - Single character names allowed
func ValidateName(name string) error {
	if name == "" {
		return errors.New("identity name cannot be empty")
	}

	if len(name) > maxNameLength {
		return fmt.Errorf("identity name cannot exceed %d characters", maxNameLength)
	}

	if !nameRegex.MatchString(name) {
		hasAlpha := strings.ContainsFunc(name, func(r rune) bool {
			return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		})
		if hasAlpha && (strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-")) {
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

	if i.GitName != "" && strings.TrimSpace(i.GitName) == "" {
		return errors.New("git author name cannot be empty")
	}

	return ValidateEmail(i.Email)
}

// GitAuthorName returns the git user.name value for the identity.
// Older configs may not have git_name set, so fall back to the profile name.
func (i Identity) GitAuthorName() string {
	gitName := strings.TrimSpace(i.GitName)
	if gitName != "" {
		return gitName
	}
	return i.Name
}
