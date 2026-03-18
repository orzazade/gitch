package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var whyCmd = &cobra.Command{
	Use:   "why",
	Short: "Explain why the current identity is active",
	Long: `Show a detailed explanation of why the current git identity is active.

Evaluates all configured rules against the current directory and remote,
showing which matched, which didn't, and why the active identity was selected.

Useful for debugging auto-switching rules.

Examples:
  gitch why`,
	Args: cobra.NoArgs,
	RunE: runWhy,
}

func init() {
	rootCmd.AddCommand(whyCmd)
}

func runWhy(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	state, err := resolveCurrentProfileState(cfg)
	if err != nil {
		return fmt.Errorf("failed to read git identity: %w", err)
	}
	currentName := state.CurrentName
	currentEmail := state.CurrentEmail

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	remoteURL, _ := rules.GetGitRemoteURL()

	// Section 1: Current identity
	fmt.Println(ui.NameStyle.Render("Current Identity"))
	if currentName == "" && currentEmail == "" {
		fmt.Println("  No git identity configured.")
		fmt.Println()
		fmt.Println("  Fix: run 'gitch use <name>' to set an identity.")
		return nil
	}

	matchedIdentity := state.ExactMatch
	managed := matchedIdentity != nil
	if managed {
		fmt.Printf("  %s (%s)\n", matchedIdentity.GitAuthorName(), matchedIdentity.Email)
		fmt.Printf("  Profile: %s\n", ui.SuccessStyle.Render(matchedIdentity.Name))
	} else {
		if currentName != "" {
			fmt.Printf("  %s (%s)\n", currentName, currentEmail)
		} else {
			fmt.Printf("  %s\n", currentEmail)
		}
		fmt.Printf("  Profile: %s\n", ui.WarningStyle.Render("not managed by gitch"))
	}

	// Section 2: Context
	fmt.Println()
	fmt.Println(ui.NameStyle.Render("Context"))
	fmt.Printf("  Directory: %s\n", cwd)
	if remoteURL != "" {
		fmt.Printf("  Remote:    %s\n", remoteURL)
	} else {
		fmt.Printf("  Remote:    %s\n", ui.DimStyle.Render("none"))
	}

	// Section 3: Rule evaluation
	fmt.Println()
	fmt.Println(ui.NameStyle.Render("Rule Evaluation"))

	if len(cfg.Rules) == 0 {
		fmt.Println("  No rules configured.")
		fmt.Println()
		printWhyReason(managed, matchedIdentity, nil)
		return nil
	}

	bestMatch := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  \tTYPE\tPATTERN\tIDENTITY\tSCORE\n")

	for _, rule := range cfg.Rules {
		matched := rule.Matches(cwd, remoteURL)
		score := rule.Specificity()

		var marker string
		if matched && bestMatch != nil && rule.Pattern == bestMatch.Pattern && rule.Type == bestMatch.Type {
			marker = ui.SuccessStyle.Render(">>")
		} else if matched {
			marker = ui.WarningStyle.Render(" ~")
		} else {
			marker = ui.DimStyle.Render(" -")
		}

		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%d\n", marker, rule.Type, rule.Pattern, rule.Identity, score)
	}
	w.Flush()

	fmt.Println()
	fmt.Printf("  %s = selected (highest score)  %s = matched  %s = no match\n",
		ui.SuccessStyle.Render(">>"),
		ui.WarningStyle.Render("~"),
		ui.DimStyle.Render("-"))

	// Section 4: Reason
	fmt.Println()
	printWhyReason(managed, matchedIdentity, bestMatch)
	return nil
}

func printWhyReason(managed bool, matchedIdentity *config.Identity, bestMatch *rules.Rule) {
	fmt.Println(ui.NameStyle.Render("Reason"))

	if bestMatch != nil {
		if managed && matchedIdentity != nil && matchedIdentity.Name == bestMatch.Identity {
			fmt.Printf("  Identity '%s' is active because %s rule '%s' matched.\n",
				matchedIdentity.Name, bestMatch.Type, bestMatch.Pattern)
		} else {
			fmt.Printf("  Rule '%s' expects identity '%s'", bestMatch.Pattern, bestMatch.Identity)
			if managed {
				fmt.Printf(", but '%s' is active instead.\n", matchedIdentity.Name)
			} else {
				fmt.Println(", but current identity is not managed by gitch.")
			}
			fmt.Printf("  Fix: run 'gitch use %s' or 'gitch autoswitch'.\n", bestMatch.Identity)
		}
	} else {
		if managed {
			fmt.Printf("  No rule matches this location. Identity '%s' was set manually.\n", matchedIdentity.Name)
		} else {
			fmt.Println("  No rule matches this location. Identity was set directly in git config.")
			fmt.Println("  Fix: run 'gitch use <name>' to switch to a managed identity.")
		}
	}
}
