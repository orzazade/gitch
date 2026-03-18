package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/hooks"
	"github.com/orzazade/gitch/internal/profile"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var cloneHook bool

var cloneCmd = &cobra.Command{
	Use:   "clone <repository> [directory]",
	Short: "Clone a repository and automatically apply the matching identity",
	Long: `Clone a git repository and automatically apply the correct identity based
on your configured rules. This prevents the common mistake of cloning a work
repository while your personal identity is active.

After cloning, gitch checks your remote rules against the clone URL and your
directory rules against the clone destination. If a match is found, the
matching identity is applied to the new repository's local git config.

Use --hook to also install the gitch pre-commit hook in the cloned repository.

Any extra flags after -- are passed directly to git clone.

Examples:
  gitch clone git@github.com:company/project.git
  gitch clone https://github.com/personal/dotfiles.git ~/repos/dotfiles
  gitch clone --hook git@github.com:company/project.git
  gitch clone git@github.com:company/project.git -- --depth 1`,
	Args:               cobra.MinimumNArgs(1),
	RunE:               runClone,
	DisableFlagParsing: false,
}

func init() {
	rootCmd.AddCommand(cloneCmd)
	cloneCmd.Flags().BoolVar(&cloneHook, "hook", false, "Install pre-commit hook in the cloned repository")
}

func runClone(cmd *cobra.Command, args []string) error {
	repoURL := args[0]
	var targetDir string
	var extraArgs []string

	// Separate our args from git clone args (after --)
	if cmd.ArgsLenAtDash() >= 0 {
		dashIdx := cmd.ArgsLenAtDash()
		if dashIdx < len(args) {
			if len(args[:dashIdx]) > 1 {
				targetDir = args[1]
			}
			extraArgs = args[dashIdx:]
		}
	} else if len(args) > 1 {
		targetDir = args[1]
	}

	// Load config to check rules before cloning
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Run git clone
	gitArgs := []string{"clone"}
	gitArgs = append(gitArgs, extraArgs...)
	gitArgs = append(gitArgs, repoURL)
	if targetDir != "" {
		gitArgs = append(gitArgs, targetDir)
	}

	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	gitCmd.Stdin = os.Stdin
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Determine the cloned directory
	cloneDir, err := resolveCloneDir(repoURL, targetDir)
	if err != nil {
		return fmt.Errorf("failed to determine clone directory: %w", err)
	}

	// Change into the cloned directory to apply identity
	origDir, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(cloneDir); err != nil {
		return fmt.Errorf("failed to enter cloned repository: %w", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Find matching rule (check both remote and directory rules)
	matchedRule := rules.FindBestMatch(cfg.Rules, cloneDir, repoURL)
	if matchedRule == nil {
		fmt.Println(ui.DimStyle.Render("No matching identity rule for this repository."))
		fmt.Println(ui.DimStyle.Render("Use 'gitch use <identity>' to set one, or add a rule with 'gitch rule add'."))
		return maybeInstallHook()
	}

	// Apply the matched identity
	identity, err := cfg.GetIdentity(matchedRule.Identity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: matched rule references unknown identity %q\n", matchedRule.Identity)
		return maybeInstallHook()
	}

	result, err := profile.Apply(identity, gitpkg.ScopeLocal)
	if err != nil {
		return fmt.Errorf("failed to apply identity: %w", err)
	}
	printProfileWarnings(result.Warnings)

	printSwitchSuccess(identity)
	fmt.Printf("Matched rule: %s %q\n", matchedRule.Type, matchedRule.Pattern)

	return maybeInstallHook()
}

// maybeInstallHook installs the pre-commit hook if --hook was passed.
// Operates on the current working directory.
func maybeInstallHook() error {
	if !cloneHook {
		return nil
	}

	if err := hooks.InstallLocal(); err != nil {
		return fmt.Errorf("failed to install pre-commit hook: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render("Pre-commit hook installed."))
	return nil
}

// resolveCloneDir determines the directory that git clone created.
// If targetDir is specified, use that. Otherwise, derive from the repo URL
// (same logic git uses: last path component, minus .git suffix).
func resolveCloneDir(repoURL, targetDir string) (string, error) {
	if targetDir != "" {
		return filepath.Abs(targetDir)
	}

	// Extract repo name from URL the same way git does
	name := repoURL

	// Handle SCP-style URLs (git@host:org/repo.git)
	if idx := strings.LastIndex(name, ":"); idx > 0 && !strings.Contains(name[:idx], "/") {
		name = name[idx+1:]
	}

	// Take the last path component
	name = filepath.Base(name)

	// Remove .git suffix
	name = strings.TrimSuffix(name, ".git")

	if name == "" || name == "." {
		return "", fmt.Errorf("could not determine repository name from URL: %s", repoURL)
	}

	return filepath.Abs(name)
}
