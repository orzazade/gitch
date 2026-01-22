package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
)

// AutoSwitchResult contains the result of an auto-switch attempt
type AutoSwitchResult struct {
	Switched      bool
	FromIdentity  string
	ToIdentity    string
	MatchedRule   *rules.Rule
	SkippedReason string
}

// TryAutoSwitch checks if identity should switch based on rules and performs the switch
// Returns result indicating what happened (switched, already correct, no rule, etc.)
func TryAutoSwitch(cfg *config.Config) (*AutoSwitchResult, error) {
	// 1. Get current working directory and remote
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	remoteURL, _ := rules.GetGitRemoteURL()

	// 2. Find best matching rule
	matchedRule := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)
	if matchedRule == nil {
		return &AutoSwitchResult{
			Switched:      false,
			SkippedReason: "no matching rule",
		}, nil
	}

	// 3. Get expected identity from rule
	expectedIdentity, err := cfg.GetIdentity(matchedRule.Identity)
	if err != nil {
		return &AutoSwitchResult{
			Switched:      false,
			MatchedRule:   matchedRule,
			SkippedReason: fmt.Sprintf("identity '%s' not found", matchedRule.Identity),
		}, nil
	}

	// 4. Get current git identity
	_, currentEmail, err := git.GetCurrentIdentity()
	if err != nil {
		return nil, err
	}

	// 5. Check if already using correct identity
	if strings.EqualFold(currentEmail, expectedIdentity.Email) {
		return &AutoSwitchResult{
			Switched:      false,
			ToIdentity:    expectedIdentity.Name,
			MatchedRule:   matchedRule,
			SkippedReason: "already using correct identity",
		}, nil
	}

	// 6. Perform the switch
	// Set git config
	if err := git.SetConfig("user.name", expectedIdentity.Name, true); err != nil {
		return nil, err
	}
	if err := git.SetConfig("user.email", expectedIdentity.Email, true); err != nil {
		return nil, err
	}

	// Load SSH key if present (silently ignore errors)
	if expectedIdentity.SSHKeyPath != "" {
		_ = sshpkg.AddKeyToAgent(expectedIdentity.SSHKeyPath)
	}

	// Update default in config
	cfg.Default = expectedIdentity.Name
	_ = cfg.Save()

	return &AutoSwitchResult{
		Switched:     true,
		FromIdentity: currentEmail,
		ToIdentity:   expectedIdentity.Name,
		MatchedRule:  matchedRule,
	}, nil
}
