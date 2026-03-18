package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/ssh"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <identity> -- <command> [args...]",
	Short: "Run a command as a specific identity",
	Long: `Run an arbitrary command with git environment variables set to a specific identity.

This lets you execute one-off git commands without permanently switching your
active identity. The identity is applied via environment variables only — your
git config is never modified.

Environment variables set:
  GIT_AUTHOR_NAME, GIT_AUTHOR_EMAIL
  GIT_COMMITTER_NAME, GIT_COMMITTER_EMAIL
  GIT_SSH_COMMAND (if the identity has an SSH key)

Examples:
  gitch exec work -- git commit -m "fix bug"
  gitch exec personal -- git push origin main
  gitch exec oss -- git tag -s v1.0.0`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: identityCompletionFunc,
	SilenceUsage:      true,
	SilenceErrors:     true,
	RunE:              runExec,
}

func init() {
	rootCmd.AddCommand(execCmd)
}

func runExec(cmd *cobra.Command, args []string) error {
	identityName := args[0]

	// Find the command args after "--"
	cmdArgs := cmd.ArgsLenAtDash()
	if cmdArgs == -1 || cmdArgs >= len(args) {
		return errors.New("usage: gitch exec <identity> -- <command> [args...]\n\nUse '--' to separate the identity name from the command to run.")
	}

	subArgs := args[cmdArgs:]
	if len(subArgs) == 0 {
		return errors.New("no command specified after '--'")
	}

	// Load config and find identity
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	identity, err := cfg.GetIdentity(identityName)
	if err != nil {
		return fmt.Errorf("identity %q not found. Use 'gitch list' to see available identities", identityName)
	}

	// Build environment
	env := buildIdentityEnv(identity)

	// Resolve and execute the command
	binary, err := exec.LookPath(subArgs[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", subArgs[0])
	}

	child := exec.Command(binary, subArgs[1:]...)
	child.Env = env
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr

	if err := child.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		return err
	}

	return nil
}

// buildIdentityEnv returns the current environment with git identity variables
// injected for the given identity.
func buildIdentityEnv(identity *config.Identity) []string {
	env := os.Environ()
	authorName := identity.GitAuthorName()

	env = appendOrReplace(env, "GIT_AUTHOR_NAME", authorName)
	env = appendOrReplace(env, "GIT_AUTHOR_EMAIL", identity.Email)
	env = appendOrReplace(env, "GIT_COMMITTER_NAME", authorName)
	env = appendOrReplace(env, "GIT_COMMITTER_EMAIL", identity.Email)

	if identity.SSHKeyPath != "" {
		if sshCmd, err := ssh.GitSSHCommand(identity.SSHKeyPath); err == nil {
			env = appendOrReplace(env, "GIT_SSH_COMMAND", sshCmd)
		}
	}

	return env
}

// appendOrReplace sets an environment variable, replacing it if it already exists.
func appendOrReplace(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
