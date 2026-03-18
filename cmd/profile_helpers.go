package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/profile"
	"github.com/orzazade/gitch/internal/ui"
)

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
