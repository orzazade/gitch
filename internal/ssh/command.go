package ssh

import "strings"

// GitSSHCommand builds a deterministic ssh command for git operations.
// It pins git to a single identity file and disables ssh-agent key guessing.
func GitSSHCommand(keyPath string) (string, error) {
	expandedPath, err := ExpandPath(keyPath)
	if err != nil {
		return "", err
	}

	return "ssh -i " + shellQuote(expandedPath) + " -o IdentitiesOnly=yes", nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
