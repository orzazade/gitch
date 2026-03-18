package profile

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/prompt"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
)

// ApplyResult captures non-fatal warnings from applying a profile.
type ApplyResult struct {
	Warnings []string
	Scope    git.Scope
}

// Matches reports whether the current git configuration matches the identity.
// This includes the author name, author email, SSH command, and GPG signing configuration.
func Matches(identity *config.Identity) (bool, error) {
	return MatchesAtScope(identity, git.ScopeEffective)
}

// MatchesAtScope reports whether the git configuration in the requested scope matches the identity.
func MatchesAtScope(identity *config.Identity, scope git.Scope) (bool, error) {
	currentName, currentEmail, err := git.GetCurrentIdentityScoped(scope)
	if err != nil {
		return false, err
	}

	if !strings.EqualFold(strings.TrimSpace(currentEmail), strings.TrimSpace(identity.Email)) {
		return false, nil
	}

	if strings.TrimSpace(currentName) != strings.TrimSpace(identity.GitAuthorName()) {
		return false, nil
	}

	signingMatches, err := signingMatches(identity, scope)
	if err != nil {
		return false, err
	}
	if !signingMatches {
		return false, nil
	}

	return sshMatches(identity, scope)
}

// Apply writes the identity into git config, updates signing and SSH config,
// best-effort loads the SSH key into the agent, and refreshes the prompt cache.
func Apply(identity *config.Identity, scope git.Scope) (*ApplyResult, error) {
	if err := git.ApplyIdentityScoped(identity.GitAuthorName(), identity.Email, scope); err != nil {
		return nil, fmt.Errorf("failed to apply identity: %w", err)
	}

	if identity.GPGKeyID != "" {
		if err := git.ApplySigningConfigScoped(identity.GPGKeyID, scope); err != nil {
			return nil, fmt.Errorf("failed to configure GPG signing: %w", err)
		}
	} else {
		if err := git.ClearSigningConfigScoped(scope); err != nil {
			return nil, fmt.Errorf("failed to clear GPG signing config: %w", err)
		}
	}

	result := &ApplyResult{Scope: scope}

	if identity.SSHKeyPath != "" {
		keyPath, err := sshpkg.ExpandPath(identity.SSHKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve SSH key path: %w", err)
		}
		sshCmd, err := sshpkg.GitSSHCommand(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build SSH command: %w", err)
		}
		if err := git.SetConfigScoped("core.sshCommand", sshCmd, scope); err != nil {
			return nil, fmt.Errorf("failed to configure SSH command: %w", err)
		}
		if _, err := os.Stat(keyPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				result.Warnings = append(result.Warnings, "SSH key not found: "+keyPath)
			} else {
				result.Warnings = append(result.Warnings, fmt.Sprintf("failed to access SSH key %s: %v", keyPath, err))
			}
		} else if err := sshpkg.AddKeyToAgent(keyPath); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to add SSH key to agent: %v", err))
		}
	} else {
		if err := git.UnsetConfigScoped("core.sshCommand", scope); err != nil {
			return nil, fmt.Errorf("failed to clear SSH command: %w", err)
		}
	}

	if err := prompt.UpdateCache(identity.Name); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("failed to update prompt cache: %v", err))
	}

	return result, nil
}

func signingMatches(identity *config.Identity, scope git.Scope) (bool, error) {
	signingKey, err := git.GetConfigScoped("user.signingkey", scope)
	if err != nil {
		return false, fmt.Errorf("failed to get signing key: %w", err)
	}

	gpgSign, err := git.GetConfigScoped("commit.gpgsign", scope)
	if err != nil {
		return false, fmt.Errorf("failed to get commit signing flag: %w", err)
	}

	if identity.GPGKeyID == "" {
		return signingKey == "" && !isTruthy(gpgSign), nil
	}

	return strings.TrimSpace(signingKey) == strings.TrimSpace(identity.GPGKeyID) && isTruthy(gpgSign), nil
}

func sshMatches(identity *config.Identity, scope git.Scope) (bool, error) {
	sshCommand, err := git.GetConfigScoped("core.sshCommand", scope)
	if err != nil {
		return false, fmt.Errorf("failed to get SSH command: %w", err)
	}

	if identity.SSHKeyPath == "" {
		return sshCommand == "", nil
	}

	expected, err := sshpkg.GitSSHCommand(identity.SSHKeyPath)
	if err != nil {
		return false, fmt.Errorf("failed to build SSH command: %w", err)
	}

	return strings.TrimSpace(sshCommand) == strings.TrimSpace(expected), nil
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
