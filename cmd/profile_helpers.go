package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/profile"
	"github.com/orzazade/gitch/internal/ui"
)

// printJSON marshals v as indented JSON and prints it to stdout.
func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

type currentProfileState struct {
	CurrentName  string
	CurrentEmail string
	ExactMatch   *config.Identity
	EmailMatch   *config.Identity
}

func applyConfiguredIdentity(identity *config.Identity, scope gitpkg.Scope) error {
	result, err := profile.Apply(identity, scope)
	if err != nil {
		return err
	}

	printProfileWarnings(result.Warnings)
	return nil
}

func printSwitchSuccess(identity *config.Identity) {
	fmt.Println(ui.SuccessStyle.Render("Switched to '" + identity.Name + "' (" + identity.Email + ")"))
	fmt.Printf("Git author: %s\n", identity.GitAuthorName())
}

func printScopeInfo(scope gitpkg.Scope) {
	if scope == gitpkg.ScopeLocal {
		fmt.Println("Scope: local repository")
	} else {
		fmt.Println("Scope: global")
	}
}

func printProfileWarnings(warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
	}
}

func resolveGitAuthorName(explicitGitName, profileName string) (string, string) {
	if gitName := strings.TrimSpace(explicitGitName); gitName != "" {
		return gitName, ""
	}

	currentName, _, err := gitpkg.GetCurrentIdentity()
	if err == nil && strings.TrimSpace(currentName) != "" {
		return strings.TrimSpace(currentName), ""
	}

	return profileName, "No git author name found in existing git config; using profile name as git user.name"
}

func defaultApplyScope() gitpkg.Scope {
	if gitpkg.IsGitRepository() {
		return gitpkg.ScopeLocal
	}
	return gitpkg.ScopeGlobal
}

func resolveApplyScope(forceLocal, forceGlobal bool) (gitpkg.Scope, error) {
	if forceLocal && forceGlobal {
		return "", errors.New("cannot use both --local and --global")
	}
	if forceLocal {
		return gitpkg.ScopeLocal, nil
	}
	if forceGlobal {
		return gitpkg.ScopeGlobal, nil
	}
	return defaultApplyScope(), nil
}

// shortHash returns the first 8 characters of a commit hash, or the full hash if shorter.
func shortHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

// truncateSubject shortens a commit subject to maxLen runes, appending "..." if truncated.
func truncateSubject(subject string, maxLen int) string {
	runes := []rune(subject)
	if len(runes) <= maxLen {
		return subject
	}
	return string(runes[:maxLen-3]) + "..."
}

func resolveCurrentProfileState(cfg *config.Config) (*currentProfileState, error) {
	currentName, currentEmail, err := gitpkg.GetCurrentIdentity()
	if err != nil {
		return nil, err
	}

	state := &currentProfileState{
		CurrentName:  currentName,
		CurrentEmail: currentEmail,
	}

	if currentEmail == "" {
		return state, nil
	}

	for i := range cfg.Identities {
		identity := &cfg.Identities[i]
		if !strings.EqualFold(identity.Email, currentEmail) {
			continue
		}

		if state.EmailMatch == nil {
			state.EmailMatch = identity
		}

		matches, err := profile.Matches(identity)
		if err != nil {
			return nil, err
		}
		if matches {
			state.ExactMatch = identity
			break
		}
	}

	return state, nil
}
