package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var ruleFromGitconfigApply bool

var ruleFromGitconfigCmd = &cobra.Command{
	Use:   "from-gitconfig",
	Short: "Import rules from git's includeIf conditional includes",
	Long: `Parse your global git config for [includeIf "gitdir:..."] conditional
includes and convert them into gitch rules.

Many developers already have conditional includes in ~/.gitconfig to switch
identities by directory. This command detects those and imports them as gitch
rules, so gitch can manage the switching instead.

By default, shows what would be imported without making changes. Use --apply
to create the rules.

Examples:
  gitch rule from-gitconfig
  gitch rule from-gitconfig --apply`,
	Args: cobra.NoArgs,
	RunE: runRuleFromGitconfig,
}

func init() {
	ruleCmd.AddCommand(ruleFromGitconfigCmd)
	ruleFromGitconfigCmd.Flags().BoolVar(&ruleFromGitconfigApply, "apply", false,
		"Actually create the rules (default: dry-run)")
}

// gitconfigInclude represents a parsed includeIf entry from git config.
type gitconfigInclude struct {
	GitdirPattern string
	ConfigPath    string
	UserName      string
	UserEmail     string
}

// gitconfigToGitchPattern converts a git includeIf gitdir pattern to a gitch
// directory rule pattern. Git uses trailing "/" to mean "this directory and
// everything below it"; gitch uses "/**" for the same.
func gitconfigToGitchPattern(gitdirPattern string) string {
	p := strings.TrimRight(gitdirPattern, "/")
	if !strings.HasSuffix(p, "/**") && !strings.HasSuffix(p, "/*") {
		p += "/**"
	}
	return p
}

// parseGitconfigIncludes reads global git config for includeIf gitdir entries.
func parseGitconfigIncludes() ([]gitconfigInclude, error) {
	cmd := exec.Command("git", "config", "--global", "--get-regexp",
		`^includeif\.gitdir.*\.path$`)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, nil // no matches found
		}
		return nil, fmt.Errorf("failed to read git config: %w", err)
	}

	var includes []gitconfigInclude
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		include, ok := parseIncludeIfLine(line)
		if !ok {
			continue
		}
		include.UserName = readGitConfigValue(include.ConfigPath, "user.name")
		include.UserEmail = readGitConfigValue(include.ConfigPath, "user.email")
		includes = append(includes, include)
	}

	return includes, nil
}

// parseIncludeIfLine parses a line from "git config --get-regexp" output.
// Format: "includeif.gitdir:PATTERN.path VALUE"
func parseIncludeIfLine(line string) (gitconfigInclude, bool) {
	key, value, ok := strings.Cut(line, " ")
	if !ok {
		return gitconfigInclude{}, false
	}

	const prefix = "includeif."
	const suffix = ".path"
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
		return gitconfigInclude{}, false
	}
	condition := key[len(prefix) : len(key)-len(suffix)]

	var pattern string
	switch {
	case strings.HasPrefix(condition, "gitdir:"):
		pattern = condition[len("gitdir:"):]
	case strings.HasPrefix(condition, "gitdir/i:"):
		pattern = condition[len("gitdir/i:"):]
	default:
		return gitconfigInclude{}, false
	}

	configPath := value
	if expanded, err := sshpkg.ExpandPath(configPath); err == nil {
		configPath = expanded
	}

	return gitconfigInclude{
		GitdirPattern: pattern,
		ConfigPath:    configPath,
	}, true
}

// readGitConfigValue reads a single value from a git config file.
func readGitConfigValue(path, key string) string {
	out, err := exec.Command("git", "config", "--file", path, key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runRuleFromGitconfig(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	includes, err := parseGitconfigIncludes()
	if err != nil {
		return err
	}

	if len(includes) == 0 {
		fmt.Println("No [includeIf \"gitdir:...\"] entries found in global git config.")
		fmt.Println(ui.DimStyle.Render("This command imports rules from git's conditional includes."))
		return nil
	}

	fmt.Println(ui.DimStyle.Render("Found " + strconv.Itoa(len(includes)) + " conditional " +
		nounPlural(len(includes), "include", "includes") + " in git config:"))
	fmt.Println()

	type importCandidate struct {
		pattern  string
		identity string
	}
	var candidates []importCandidate

	existingRules := cfg.ListRules()

	for _, inc := range includes {
		fmt.Printf("  gitdir: %s\n", inc.GitdirPattern)
		fmt.Printf("  config: %s\n", inc.ConfigPath)

		if inc.UserEmail == "" {
			fmt.Printf("  %s\n\n", ui.WarningStyle.Render("no user.email in included config — skipping"))
			continue
		}

		fmt.Printf("  identity: %s <%s>\n", inc.UserName, inc.UserEmail)

		// Match to gitch identity by email
		idx := slices.IndexFunc(cfg.Identities, func(id config.Identity) bool {
			return strings.EqualFold(id.Email, inc.UserEmail)
		})
		if idx < 0 {
			fmt.Printf("  %s\n\n", ui.WarningStyle.Render(
				"no matching gitch identity — add one first: gitch add --email "+inc.UserEmail))
			continue
		}
		identityName := cfg.Identities[idx].Name

		gitchPattern := gitconfigToGitchPattern(inc.GitdirPattern)

		// Check if rule already exists
		alreadyExists := slices.ContainsFunc(existingRules, func(r rules.Rule) bool {
			return r.Type == rules.DirectoryRule && r.Pattern == gitchPattern
		})
		if alreadyExists {
			fmt.Printf("  %s\n\n", ui.DimStyle.Render("rule already exists: "+gitchPattern))
			continue
		}

		fmt.Printf("  %s  %s -> %s\n\n",
			ui.SuccessStyle.Render("new rule:"), gitchPattern, identityName)
		candidates = append(candidates, importCandidate{gitchPattern, identityName})
	}

	if len(candidates) == 0 {
		fmt.Println("No new rules to import.")
		return nil
	}

	if !ruleFromGitconfigApply {
		fmt.Println(ui.DimStyle.Render(strconv.Itoa(len(candidates)) + " " +
			nounPlural(len(candidates), "rule", "rules") + " ready to import. " +
			"Run with --apply to create " + nounPlural(len(candidates), "it", "them") + "."))
		return nil
	}

	// Apply the rules
	created := 0
	for _, c := range candidates {
		rule := rules.Rule{
			Type:     rules.DirectoryRule,
			Pattern:  c.pattern,
			Identity: c.identity,
		}
		if err := cfg.AddRule(rule); err != nil {
			fmt.Printf("  %s  failed to add rule %s: %v\n",
				ui.WarningStyle.Render("warn"), c.pattern, err)
			continue
		}
		created++
	}

	if created > 0 {
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Println(ui.SuccessStyle.Render(strconv.Itoa(created) + " " +
			nounPlural(created, "rule", "rules") + " imported successfully."))
	}

	return nil
}
