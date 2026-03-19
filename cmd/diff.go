package cmd

import (
	"fmt"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var diffJSON bool

var diffCmd = &cobra.Command{
	Use:   "diff <identity-name>",
	Short: "Preview what would change when switching to an identity",
	Long: `Show the differences between the current git configuration and a
target identity without making any changes.

Compares user.name, user.email, SSH key, and GPG signing configuration
so you can preview the impact before running 'gitch use'.

Examples:
  gitch diff work
  gitch diff personal --json`,
	Args:              cobra.ExactArgs(1),
	RunE:              runDiff,
	SilenceErrors:     true,
	SilenceUsage:      true,
	ValidArgsFunction: identityCompletionFunc,
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().BoolVar(&diffJSON, "json", false, "Output in JSON format")
}

type diffField struct {
	Field   string `json:"field"`
	Current string `json:"current"`
	Target  string `json:"target"`
	Changed bool   `json:"changed"`
}

type diffOutput struct {
	Identity string      `json:"identity"`
	Fields   []diffField `json:"fields"`
	Match    bool        `json:"match"`
}

func runDiff(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	name := args[0]
	identity, err := cfg.GetIdentity(name)
	if err != nil {
		return identityNotFoundError(name, cfg.ListIdentities(), "gitch diff "+name)
	}

	fields, err := buildDiffFields(identity)
	if err != nil {
		return err
	}

	allMatch := true
	for _, f := range fields {
		if f.Changed {
			allMatch = false
			break
		}
	}

	output := diffOutput{
		Identity: identity.Name,
		Fields:   fields,
		Match:    allMatch,
	}

	if diffJSON {
		return printJSON(output)
	}

	printDiffHuman(output)
	return nil
}

func buildDiffFields(identity *config.Identity) ([]diffField, error) {
	currentName, currentEmail, err := gitpkg.GetCurrentIdentity()
	if err != nil {
		return nil, fmt.Errorf("failed to read current git identity: %w", err)
	}

	targetName := identity.GitAuthorName()
	targetEmail := identity.Email

	fields := []diffField{
		{
			Field:   "user.name",
			Current: currentName,
			Target:  targetName,
			Changed: strings.TrimSpace(currentName) != strings.TrimSpace(targetName),
		},
		{
			Field:   "user.email",
			Current: currentEmail,
			Target:  targetEmail,
			Changed: !strings.EqualFold(strings.TrimSpace(currentEmail), strings.TrimSpace(targetEmail)),
		},
	}

	// SSH command
	currentSSH, err := gitpkg.GetConfigScoped("core.sshCommand", gitpkg.ScopeEffective)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH config: %w", err)
	}
	targetSSH := ""
	if identity.SSHKeyPath != "" {
		keyPath, err := sshpkg.ExpandPath(identity.SSHKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to expand SSH key path: %w", err)
		}
		targetSSH, err = sshpkg.GitSSHCommand(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build SSH command: %w", err)
		}
	}
	fields = append(fields, diffField{
		Field:   "core.sshCommand",
		Current: currentSSH,
		Target:  targetSSH,
		Changed: strings.TrimSpace(currentSSH) != strings.TrimSpace(targetSSH),
	})

	// GPG signing key
	currentSigningKey, err := gitpkg.GetConfigScoped("user.signingkey", gitpkg.ScopeEffective)
	if err != nil {
		return nil, fmt.Errorf("failed to read signing key: %w", err)
	}
	fields = append(fields, diffField{
		Field:   "user.signingkey",
		Current: currentSigningKey,
		Target:  identity.GPGKeyID,
		Changed: strings.TrimSpace(currentSigningKey) != strings.TrimSpace(identity.GPGKeyID),
	})

	// GPG sign enabled
	currentGPGSign, err := gitpkg.GetConfigScoped("commit.gpgsign", gitpkg.ScopeEffective)
	if err != nil {
		return nil, fmt.Errorf("failed to read GPG sign flag: %w", err)
	}
	targetGPGSign := ""
	if identity.GPGKeyID != "" {
		targetGPGSign = "true"
	}
	fields = append(fields, diffField{
		Field:   "commit.gpgsign",
		Current: currentGPGSign,
		Target:  targetGPGSign,
		Changed: strings.TrimSpace(currentGPGSign) != strings.TrimSpace(targetGPGSign),
	})

	return fields, nil
}

func printDiffHuman(output diffOutput) {
	if output.Match {
		fmt.Println(ui.SuccessStyle.Render("Already using '" + output.Identity + "' — no changes needed"))
		return
	}

	fmt.Println(ui.NameStyle.Render("Diff: current → " + output.Identity))
	fmt.Println()

	for _, f := range output.Fields {
		if !f.Changed {
			continue
		}
		fmt.Println(ui.DimStyle.Render("  " + f.Field))
		if f.Current == "" {
			fmt.Println(ui.SuccessStyle.Render("    + " + f.Target))
		} else if f.Target == "" {
			fmt.Println(ui.ErrorStyle.Render("    - " + f.Current))
		} else {
			fmt.Println(ui.ErrorStyle.Render("    - " + f.Current))
			fmt.Println(ui.SuccessStyle.Render("    + " + f.Target))
		}
	}

	// Show unchanged fields dimmed
	hasUnchanged := false
	for _, f := range output.Fields {
		if f.Changed {
			continue
		}
		if !hasUnchanged {
			fmt.Println()
			fmt.Println(ui.DimStyle.Render("  unchanged"))
			hasUnchanged = true
		}
		val := f.Current
		if val == "" {
			val = "(not set)"
		}
		fmt.Println(ui.DimStyle.Render("    " + f.Field + " = " + val))
	}
}
