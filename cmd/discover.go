package cmd

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var discoverDepth int
var discoverJSON bool

var discoverCmd = &cobra.Command{
	Use:   "discover [paths...]",
	Short: "Discover git identities used across your repositories",
	Long: `Scan directories for git repositories and discover unique author
identities (name + email) configured in each repo's local git config.

This helps with onboarding: instead of manually adding identities,
gitch discovers what you're already using across your repos and shows
which ones aren't yet managed by gitch.

By default, scans the current directory up to 3 levels deep.

Examples:
  gitch discover
  gitch discover ~/Projects ~/Work
  gitch discover --depth 5 ~/code
  gitch discover --json`,
	RunE: runDiscover,
}

func init() {
	rootCmd.AddCommand(discoverCmd)
	discoverCmd.Flags().IntVar(&discoverDepth, "depth", 3, "Maximum directory depth to search")
	discoverCmd.Flags().BoolVar(&discoverJSON, "json", false, "Output in JSON format")
}

// discoveredIdentity represents a unique (name, email) pair found across repos.
type discoveredIdentity struct {
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Repos   []string `json:"repos"`
	Managed string   `json:"managed,omitempty"` // gitch identity name if managed
}

func runDiscover(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	roots, err := resolveRoots(args)
	if err != nil {
		return err
	}

	repos := findGitRepos(roots, discoverDepth)
	if len(repos) == 0 {
		fmt.Println("No git repositories found.")
		return nil
	}

	identities := discoverIdentities(repos, cfg)

	if discoverJSON {
		return printJSON(identities)
	}

	printDiscoverResults(identities, len(repos))
	return nil
}

// discoverIdentities scans each repo for its local git identity and groups
// results by unique (name, email) pairs.
func discoverIdentities(repoPaths []string, cfg *config.Config) []discoveredIdentity {
	type identityKey struct {
		name  string
		email string
	}

	groups := make(map[identityKey]*discoveredIdentity)
	home, _ := os.UserHomeDir()

	originalDir, err := os.Getwd()
	if err != nil {
		return nil
	}
	defer func() { _ = os.Chdir(originalDir) }()

	for _, repoPath := range repoPaths {
		if err := os.Chdir(repoPath); err != nil {
			continue
		}

		// Read effective identity for this repo
		name, email, err := gitpkg.GetCurrentIdentity()
		if err != nil || email == "" {
			continue
		}

		displayPath := shortenHome(repoPath, home)

		key := identityKey{name: name, email: strings.ToLower(email)}
		if group, ok := groups[key]; ok {
			group.Repos = append(group.Repos, displayPath)
		} else {
			groups[key] = &discoveredIdentity{
				Name:  name,
				Email: email,
				Repos: []string{displayPath},
			}
		}
	}

	// Check which discovered identities are already managed by gitch
	for _, group := range groups {
		if idx := slices.IndexFunc(cfg.Identities, func(id config.Identity) bool {
			return strings.EqualFold(id.Email, group.Email)
		}); idx != -1 {
			group.Managed = cfg.Identities[idx].Name
		}
	}

	// Convert map to sorted slice (most repos first, then alphabetical by email)
	result := make([]discoveredIdentity, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	slices.SortFunc(result, func(a, b discoveredIdentity) int {
		if len(a.Repos) != len(b.Repos) {
			return len(b.Repos) - len(a.Repos) // descending by count
		}
		return strings.Compare(strings.ToLower(a.Email), strings.ToLower(b.Email))
	})

	return result
}

func printDiscoverResults(identities []discoveredIdentity, totalRepos int) {
	fmt.Println(ui.DimStyle.Render("gitch discover — " + strconv.Itoa(totalRepos) + " " +
		nounPlural(totalRepos, "repo", "repos") + " scanned, " +
		strconv.Itoa(len(identities)) + " unique " +
		nounPlural(len(identities), "identity", "identities") + " found"))
	fmt.Println()

	unmanaged := 0
	for _, id := range identities {
		// Status indicator
		if id.Managed != "" {
			fmt.Printf("  %s  %s <%s>\n", ui.SuccessStyle.Render("ok"), id.Name, id.Email)
			fmt.Printf("       managed as: %s\n", ui.SuccessStyle.Render(id.Managed))
		} else {
			fmt.Printf("  %s  %s <%s>\n", ui.WarningStyle.Render("??"), id.Name, id.Email)
			fmt.Printf("       %s\n", ui.WarningStyle.Render("not managed by gitch"))
			unmanaged++
		}

		// Show repo count and first few repos
		repoCount := len(id.Repos)
		fmt.Printf("       %s: %s\n", nounPlural(repoCount, "repo", "repos"), strconv.Itoa(repoCount))
		maxShow := 3
		for i, repo := range id.Repos {
			if i >= maxShow {
				fmt.Printf("         ... and %s more\n", strconv.Itoa(repoCount-maxShow))
				break
			}
			fmt.Printf("         %s\n", repo)
		}

		fmt.Println()
	}

	// Summary
	if unmanaged == 0 {
		fmt.Println(ui.SuccessStyle.Render("All discovered identities are managed by gitch."))
	} else {
		fmt.Println(ui.WarningStyle.Render(strconv.Itoa(unmanaged) + " " +
			nounPlural(unmanaged, "identity", "identities") +
			" not yet managed by gitch."))
		fmt.Println(ui.DimStyle.Render("Run 'gitch add --name <name> --email <email>' to add them."))
	}
}
