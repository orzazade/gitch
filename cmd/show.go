package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/gpg"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var showJSON bool

var showCmd = &cobra.Command{
	Use:   "show <identity-name>",
	Short: "Show detailed information about an identity",
	Long: `Display all details of a single identity, including SSH key status,
GPG key status, associated rules, and hook mode.

Useful for verifying an identity is correctly configured before use.

Examples:
  gitch show work
  gitch show personal --json`,
	Args:              cobra.ExactArgs(1),
	RunE:              runShow,
	SilenceErrors:     true,
	SilenceUsage:      true,
	ValidArgsFunction: identityCompletionFunc,
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output in JSON format")
}

type showOutput struct {
	Name      string      `json:"name"`
	GitName   string      `json:"git_name"`
	Email     string      `json:"email"`
	IsDefault bool        `json:"is_default"`
	IsActive  bool        `json:"is_active"`
	HookMode  string      `json:"hook_mode"`
	SSHKey    *showSSHKey `json:"ssh_key,omitempty"`
	GPGKey    *showGPGKey `json:"gpg_key,omitempty"`
	Rules     []showRule  `json:"rules"`
}

type showSSHKey struct {
	Path        string `json:"path"`
	Exists      bool   `json:"exists"`
	Fingerprint string `json:"fingerprint,omitempty"`
	AgentLoaded bool   `json:"agent_loaded"`
}

type showGPGKey struct {
	KeyID string `json:"key_id"`
	Valid bool   `json:"valid"`
}

type showRule struct {
	Type    string `json:"type"`
	Pattern string `json:"pattern"`
}

func runShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	name := args[0]
	identity, err := cfg.GetIdentity(name)
	if err != nil {
		msg := "identity '" + name + "' not found"
		if suggestion := closestIdentityName(name, cfg.ListIdentities()); suggestion != "" {
			msg += "\n\nDid you mean '" + suggestion + "'?  Run: gitch show " + suggestion
		}
		return errors.New(msg)
	}

	// Check if this is the active identity
	state, _ := resolveCurrentProfileState(cfg)
	isActive := state != nil && state.ExactMatch != nil &&
		strings.EqualFold(state.ExactMatch.Name, identity.Name)

	isDefault := strings.EqualFold(cfg.Default, identity.Name)

	// Gather SSH key info
	var sshInfo *showSSHKey
	if identity.SSHKeyPath != "" {
		sshInfo = gatherSSHInfo(identity.SSHKeyPath)
	}

	// Gather GPG key info
	var gpgInfo *showGPGKey
	if identity.GPGKeyID != "" {
		gpgInfo = gatherGPGInfo(identity.GPGKeyID)
	}

	// Find associated rules
	associatedRules := []showRule{}
	for _, r := range cfg.Rules {
		if strings.EqualFold(r.Identity, identity.Name) {
			associatedRules = append(associatedRules, showRule{
				Type:    string(r.Type),
				Pattern: r.Pattern,
			})
		}
	}

	hookMode := identity.GetHookMode()

	output := showOutput{
		Name:      identity.Name,
		GitName:   identity.GitAuthorName(),
		Email:     identity.Email,
		IsDefault: isDefault,
		IsActive:  isActive,
		HookMode:  hookMode,
		SSHKey:    sshInfo,
		GPGKey:    gpgInfo,
		Rules:     associatedRules,
	}

	if showJSON {
		return printJSON(output)
	}
	printShowHuman(output)
	return nil
}

func gatherSSHInfo(keyPath string) *showSSHKey {
	info := &showSSHKey{Path: keyPath}

	expanded, err := sshpkg.ExpandPath(keyPath)
	if err != nil {
		return info
	}

	if _, err := os.Stat(expanded); err != nil {
		return info
	}
	info.Exists = true

	// Read public key for fingerprint
	pubKeyPath := expanded + ".pub"
	pubKeyData, err := os.ReadFile(pubKeyPath)
	if err == nil {
		fp, err := sshpkg.GetFingerprint(pubKeyData)
		if err == nil {
			info.Fingerprint = fp
		}
	}

	info.AgentLoaded = sshpkg.IsKeyLoadedInAgent(expanded)
	return info
}

func gatherGPGInfo(keyID string) *showGPGKey {
	info := &showGPGKey{KeyID: keyID}
	if gpg.IsGPGAvailable() {
		info.Valid = gpg.ValidateKeyID(keyID) == nil
	}
	return info
}

func printShowHuman(output showOutput) {
	// Header
	var badges []string
	if output.IsActive {
		badges = append(badges, ui.SuccessStyle.Render("active"))
	}
	if output.IsDefault {
		badges = append(badges, ui.DimStyle.Render("default"))
	}

	header := ui.NameStyle.Render(output.Name)
	if len(badges) > 0 {
		header += "  " + strings.Join(badges, " ")
	}
	fmt.Println(header)
	fmt.Println()

	// Identity
	fmt.Println(ui.DimStyle.Render("  Identity"))
	fmt.Printf("    Name:      %s\n", output.GitName)
	fmt.Printf("    Email:     %s\n", output.Email)
	fmt.Printf("    Hook mode: %s\n", output.HookMode)
	fmt.Println()

	// SSH Key
	if output.SSHKey != nil {
		fmt.Println(ui.DimStyle.Render("  SSH Key"))
		fmt.Printf("    Path:        %s\n", output.SSHKey.Path)
		if !output.SSHKey.Exists {
			fmt.Printf("    Status:      %s\n", ui.ErrorStyle.Render("file not found"))
		} else {
			fmt.Printf("    Status:      %s\n", ui.SuccessStyle.Render("exists"))
			if output.SSHKey.Fingerprint != "" {
				fmt.Printf("    Fingerprint: %s\n", output.SSHKey.Fingerprint)
			}
			if output.SSHKey.AgentLoaded {
				fmt.Printf("    Agent:       %s\n", ui.SuccessStyle.Render("loaded"))
			} else {
				fmt.Printf("    Agent:       %s\n", ui.WarningStyle.Render("not loaded"))
			}
		}
		fmt.Println()
	}

	// GPG Key
	if output.GPGKey != nil {
		fmt.Println(ui.DimStyle.Render("  GPG Key"))
		fmt.Printf("    Key ID: %s\n", output.GPGKey.KeyID)
		if output.GPGKey.Valid {
			fmt.Printf("    Status: %s\n", ui.SuccessStyle.Render("valid"))
		} else {
			fmt.Printf("    Status: %s\n", ui.ErrorStyle.Render("not found in keyring"))
		}
		fmt.Println()
	}

	// Rules
	fmt.Println(ui.DimStyle.Render("  Rules"))
	if len(output.Rules) > 0 {
		for _, r := range output.Rules {
			fmt.Printf("    %s  %s\n", ui.DimStyle.Render(r.Type), r.Pattern)
		}
	} else {
		fmt.Println("    " + ui.DimStyle.Render("none"))
	}
}
