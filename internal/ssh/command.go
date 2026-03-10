package ssh

import (
	"fmt"
	"strings"
)

// GitSSHCommand builds a deterministic ssh command for git operations.
// It pins git to a single identity file and disables ssh-agent key guessing.
func GitSSHCommand(keyPath string) (string, error) {
	expandedPath, err := ExpandPath(keyPath)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", shellQuote(expandedPath)), nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
