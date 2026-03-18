package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/hooks"
	"github.com/orzazade/gitch/internal/rules"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var scanDepth int

var scanCmd = &cobra.Command{
	Use:   "scan [paths...]",
	Short: "Scan directories for git repos and report identity coverage",
	Long: `Recursively scan directories for git repositories and show the identity
status of each one: which identity is active, whether a rule matches,
and whether a pre-commit hook is installed.

This gives you a birds-eye view across all your repos so you can spot
misconfigurations before they become wrong commits.

By default, scans the current directory up to 3 levels deep.

Examples:
  gitch scan
  gitch scan ~/Projects ~/Work
  gitch scan --depth 5 ~/code`,
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().IntVar(&scanDepth, "depth", 3, "Maximum directory depth to search")
}

// repoResult holds the scan result for a single git repository.
type repoResult struct {
	Path         string
	Identity     string
	Email        string
	Managed      bool
	HookGlobal   bool
	HookLocal    bool
	RuleMatch    string // identity name from matched rule, or ""
	RuleMismatch bool   // true if rule expects a different identity
}

func (r *repoResult) hasIssue() bool {
	return !r.Managed || r.RuleMismatch || (!r.HookGlobal && !r.HookLocal)
}

func runScan(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	roots := args
	if len(roots) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		roots = []string{cwd}
	}

	// Expand tildes and env vars in paths
	for i, root := range roots {
		if expanded, err := sshpkg.ExpandPath(root); err == nil {
			roots[i] = expanded
		}
	}

	repos := findGitRepos(roots, scanDepth)
	if len(repos) == 0 {
		fmt.Println("No git repositories found.")
		return nil
	}

	results := scanRepos(repos, cfg)
	printScanResults(results)
	return nil
}

// findGitRepos walks the given root directories up to maxDepth levels
// and returns paths that contain a .git directory.
func findGitRepos(roots []string, maxDepth int) []string {
	var repos []string
	seen := make(map[string]bool)

	for _, root := range roots {
		root, err := filepath.Abs(root)
		if err != nil {
			continue
		}

		// Check if root itself is a repo
		if isGitRepoDir(root) {
			if !seen[root] {
				seen[root] = true
				repos = append(repos, root)
			}
		}

		rootDepth := strings.Count(root, string(os.PathSeparator))

		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return fs.SkipDir
			}
			if !d.IsDir() {
				return nil
			}

			// Skip hidden directories (except .git check at root)
			if d.Name() != "." && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}

			depth := strings.Count(path, string(os.PathSeparator)) - rootDepth
			if depth > maxDepth {
				return fs.SkipDir
			}

			if isGitRepoDir(path) && !seen[path] {
				seen[path] = true
				repos = append(repos, path)
				return fs.SkipDir // don't recurse into nested repos
			}

			return nil
		})
	}

	return repos
}

// isGitRepoDir checks if the given directory contains a .git directory or file.
func isGitRepoDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil {
		return false
	}
	// .git can be a directory (normal repo) or a file (worktree)
	return info.IsDir() || info.Mode().IsRegular()
}

// scanRepos inspects each repo and builds results.
func scanRepos(repoPaths []string, cfg *config.Config) []repoResult {
	// Check global hook status once (same for all repos)
	globalHook, _ := hooks.IsInstalled()

	originalDir, err := os.Getwd()
	if err != nil {
		return nil
	}
	defer func() { _ = os.Chdir(originalDir) }()

	results := make([]repoResult, 0, len(repoPaths))

	for _, repoPath := range repoPaths {
		result := scanSingleRepo(repoPath, cfg, globalHook)
		results = append(results, result)
	}

	return results
}

func scanSingleRepo(repoPath string, cfg *config.Config, globalHook bool) repoResult {
	result := repoResult{
		Path:       repoPath,
		HookGlobal: globalHook,
	}

	// We need to chdir to read repo-specific git config and hooks
	if err := os.Chdir(repoPath); err != nil {
		return result
	}

	// Get current identity from effective git config
	name, email, err := gitpkg.GetCurrentIdentity()
	if err == nil {
		result.Identity = name
		result.Email = email
	}

	// Check if identity is managed
	if email != "" {
		if idx := slices.IndexFunc(cfg.Identities, func(id config.Identity) bool {
			return strings.EqualFold(id.Email, email)
		}); idx != -1 {
			result.Managed = true
			result.Identity = cfg.Identities[idx].Name
		}
	}

	// Check local hook
	result.HookLocal, _ = hooks.IsInstalledLocal()

	// Check rule match
	remoteURL, _ := rules.GetGitRemoteURL()
	if matched := rules.FindBestMatch(cfg.Rules, repoPath, remoteURL); matched != nil {
		result.RuleMatch = matched.Identity
		if email != "" {
			expected, err := cfg.GetIdentity(matched.Identity)
			if err == nil && !strings.EqualFold(expected.Email, email) {
				result.RuleMismatch = true
			}
		}
	}

	return result
}

func printScanResults(results []repoResult) {
	// Find shortest unique path prefix to shorten display
	home, _ := os.UserHomeDir()

	issues := 0
	for i := range results {
		if results[i].hasIssue() {
			issues++
		}
	}

	fmt.Println(ui.DimStyle.Render(fmt.Sprintf("gitch scan — %d %s found", len(results), nounPlural(len(results), "repo", "repos"))))
	fmt.Println()

	for _, r := range results {
		displayPath := r.Path
		if home != "" && strings.HasPrefix(displayPath, home) {
			displayPath = "~" + displayPath[len(home):]
		}

		// Status indicator
		if r.hasIssue() {
			fmt.Printf("  %s  %s\n", ui.WarningStyle.Render("!"), displayPath)
		} else {
			fmt.Printf("  %s  %s\n", ui.SuccessStyle.Render("ok"), displayPath)
		}

		// Identity line
		if r.Identity != "" {
			if r.Managed {
				fmt.Printf("       identity: %s (%s)\n", r.Identity, r.Email)
			} else {
				fmt.Printf("       identity: %s (%s) %s\n", r.Identity, r.Email, ui.WarningStyle.Render("[not managed]"))
			}
		} else {
			fmt.Printf("       identity: %s\n", ui.WarningStyle.Render("none"))
		}

		// Hook line
		switch {
		case r.HookGlobal:
			fmt.Printf("       hook: global\n")
		case r.HookLocal:
			fmt.Printf("       hook: local\n")
		default:
			fmt.Printf("       hook: %s\n", ui.WarningStyle.Render("not installed"))
		}

		// Rule line
		if r.RuleMatch != "" {
			if r.RuleMismatch {
				fmt.Printf("       rule: expects %s %s\n", r.RuleMatch, ui.WarningStyle.Render("[MISMATCH]"))
			} else {
				fmt.Printf("       rule: %s\n", r.RuleMatch)
			}
		}

		fmt.Println()
	}

	// Summary
	if issues == 0 {
		fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("All %d %s properly configured.", len(results), nounPlural(len(results), "repo", "repos"))))
	} else {
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("%d of %d %s %s issues.",
			issues, len(results),
			nounPlural(len(results), "repo", "repos"),
			nounPlural(issues, "has", "have"),
		)))
		fmt.Println(ui.DimStyle.Render("Run 'gitch doctor' inside a repo for details, or 'gitch use <name>' to fix."))
	}
}
