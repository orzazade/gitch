package portability

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ssh"
	"gopkg.in/yaml.v3"
)

// ConflictType indicates whether a conflict is for an identity or rule.
type ConflictType string

const (
	// IdentityConflict indicates a conflict with an existing identity.
	IdentityConflict ConflictType = "identity"
	// RuleConflict indicates a conflict with an existing rule.
	RuleConflict ConflictType = "rule"
)

// Conflict represents a detected conflict between existing and imported configuration.
type Conflict struct {
	Type     ConflictType
	Key      string      // identity name or rule pattern
	Existing interface{} // existing config.Identity or rules.Rule
	Incoming interface{} // incoming config.Identity or rules.Rule
}

// ImportResult tracks what was added, updated, and skipped during merge.
type ImportResult struct {
	AddedIdentities   []string
	AddedRules        []string
	UpdatedIdentities []string
	UpdatedRules      []string
	Skipped           []string
}

// ErrVersionTooNew is returned when the export file version is newer than supported.
var ErrVersionTooNew = errors.New("export file version is newer than supported")

// ImportFromFile reads and parses a YAML export file.
// The path supports ~ expansion for home directory.
func ImportFromFile(path string) (*ExportConfig, error) {
	// Expand path (handle ~)
	expandedPath, err := ssh.ExpandPath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var export ExportConfig
	if err := yaml.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	// Validate version
	if export.Version > CurrentExportVersion {
		return nil, fmt.Errorf("%w: file version %d, supported up to %d",
			ErrVersionTooNew, export.Version, CurrentExportVersion)
	}

	// Ensure slices are not nil
	if export.Identities == nil {
		export.Identities = []config.Identity{}
	}
	if export.Rules == nil {
		export.Rules = []rules.Rule{}
	}

	return &export, nil
}

// DetectConflicts finds conflicts between existing config and imported export.
// Uses case-insensitive comparison for identity names.
// For rules, matches by exact pattern.
func DetectConflicts(cfg *config.Config, export *ExportConfig) []Conflict {
	var conflicts []Conflict

	// Check identity conflicts
	for _, incoming := range export.Identities {
		existing, err := cfg.GetIdentity(incoming.Name)
		if err != nil {
			// Identity doesn't exist, no conflict
			continue
		}

		// Identity exists, check if it's different
		if !identitiesEqual(existing, &incoming) {
			conflicts = append(conflicts, Conflict{
				Type:     IdentityConflict,
				Key:      incoming.Name,
				Existing: *existing,
				Incoming: incoming,
			})
		}
	}

	// Check rule conflicts
	for _, incoming := range export.Rules {
		for _, existing := range cfg.Rules {
			if existing.Pattern == incoming.Pattern {
				// Same pattern, check if it's different
				if !rulesEqual(&existing, &incoming) {
					conflicts = append(conflicts, Conflict{
						Type:     RuleConflict,
						Key:      incoming.Pattern,
						Existing: existing,
						Incoming: incoming,
					})
				}
				break
			}
		}
	}

	return conflicts
}

// identitiesEqual checks if two identities are functionally equal.
// Compares email, ssh_key_path, and gpg_key_id (case-insensitive for email).
func identitiesEqual(a, b *config.Identity) bool {
	if !strings.EqualFold(a.Email, b.Email) {
		return false
	}
	if a.SSHKeyPath != b.SSHKeyPath {
		return false
	}
	if a.GPGKeyID != b.GPGKeyID {
		return false
	}
	if a.HookMode != b.HookMode {
		return false
	}
	return true
}

// rulesEqual checks if two rules are functionally equal.
func rulesEqual(a, b *rules.Rule) bool {
	return a.Type == b.Type && a.Pattern == b.Pattern && a.Identity == b.Identity
}

// MergeConfig merges imported configuration into existing config.
// The overwrite map specifies which conflicts to overwrite (key is identity name or rule pattern).
// If a conflict exists and the key is not in overwrite map or overwrite[key] is false, the item is skipped.
func MergeConfig(cfg *config.Config, export *ExportConfig, overwrite map[string]bool) (*ImportResult, error) {
	result := &ImportResult{
		AddedIdentities:   []string{},
		AddedRules:        []string{},
		UpdatedIdentities: []string{},
		UpdatedRules:      []string{},
		Skipped:           []string{},
	}

	// Ensure overwrite map is not nil
	if overwrite == nil {
		overwrite = make(map[string]bool)
	}

	// Process identities
	for _, incoming := range export.Identities {
		existing, err := cfg.GetIdentity(incoming.Name)
		if err != nil {
			// Identity doesn't exist, add it
			if err := cfg.AddIdentity(incoming); err != nil {
				return nil, fmt.Errorf("failed to add identity %q: %w", incoming.Name, err)
			}
			result.AddedIdentities = append(result.AddedIdentities, incoming.Name)
			continue
		}

		// Identity exists
		if identitiesEqual(existing, &incoming) {
			// Identical, skip silently
			continue
		}

		// Check if we should overwrite
		if shouldOverwrite, ok := overwrite[incoming.Name]; ok && shouldOverwrite {
			// Update the identity
			if err := updateIdentity(cfg, incoming); err != nil {
				return nil, fmt.Errorf("failed to update identity %q: %w", incoming.Name, err)
			}
			result.UpdatedIdentities = append(result.UpdatedIdentities, incoming.Name)
		} else {
			// Skip this identity
			result.Skipped = append(result.Skipped, fmt.Sprintf("identity:%s", incoming.Name))
		}
	}

	// Process rules
	for _, incoming := range export.Rules {
		existingIdx := -1
		for i, existing := range cfg.Rules {
			if existing.Pattern == incoming.Pattern {
				existingIdx = i
				break
			}
		}

		if existingIdx == -1 {
			// Rule doesn't exist, add it
			if err := cfg.AddRule(incoming); err != nil {
				return nil, fmt.Errorf("failed to add rule %q: %w", incoming.Pattern, err)
			}
			result.AddedRules = append(result.AddedRules, incoming.Pattern)
			continue
		}

		// Rule exists
		if rulesEqual(&cfg.Rules[existingIdx], &incoming) {
			// Identical, skip silently
			continue
		}

		// Check if we should overwrite
		if shouldOverwrite, ok := overwrite[incoming.Pattern]; ok && shouldOverwrite {
			// Update the rule
			cfg.Rules[existingIdx] = incoming
			result.UpdatedRules = append(result.UpdatedRules, incoming.Pattern)
		} else {
			// Skip this rule
			result.Skipped = append(result.Skipped, fmt.Sprintf("rule:%s", incoming.Pattern))
		}
	}

	return result, nil
}

// updateIdentity updates an existing identity with new values.
func updateIdentity(cfg *config.Config, updated config.Identity) error {
	for i, id := range cfg.Identities {
		if strings.EqualFold(id.Name, updated.Name) {
			// Preserve the original name case
			updated.Name = id.Name
			cfg.Identities[i] = updated
			return nil
		}
	}
	return fmt.Errorf("identity %q not found", updated.Name)
}
