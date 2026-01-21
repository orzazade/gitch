// Package wizard provides an interactive setup wizard for creating identities.
package wizard

// Step constants for the wizard flow
const (
	stepName        = 0
	stepEmail       = 1
	stepSSH         = 2
	stepPassphrase  = 3
	stepConfirmPass = 4
)

// sshOptions are the choices for SSH key handling
var sshOptions = []string{
	"Generate new SSH key",
	"Skip SSH setup",
}

// sshChoiceGenerate is the index for generating a new SSH key
const sshChoiceGenerate = 0

// sshChoiceSkip is the index for skipping SSH setup
const sshChoiceSkip = 1

// getTotalSteps returns the total number of steps based on SSH choice.
// If generating SSH key, includes passphrase and optionally confirm steps.
func getTotalSteps(sshChoice int, passphraseEmpty bool) int {
	if sshChoice == sshChoiceSkip {
		return 3 // name, email, ssh choice
	}
	if passphraseEmpty {
		return 4 // name, email, ssh choice, passphrase
	}
	return 5 // name, email, ssh choice, passphrase, confirm
}

// getStepTitle returns the title/prompt for each step
func getStepTitle(step int) string {
	switch step {
	case stepName:
		return "What name would you like to use for this identity?"
	case stepEmail:
		return "What's your email address for this identity?"
	case stepSSH:
		return "Would you like to set up an SSH key?"
	case stepPassphrase:
		return "Enter a passphrase for your SSH key (optional, press Enter to skip)"
	case stepConfirmPass:
		return "Confirm your passphrase"
	default:
		return ""
	}
}

// getStepHint returns the hint text shown below the input for each step
func getStepHint(step int) string {
	switch step {
	case stepName:
		return "Alphanumeric and hyphens only (e.g., work, personal, github)"
	case stepEmail:
		return "This will be used as your git user.email"
	case stepSSH:
		return ""
	case stepPassphrase:
		return "Leave empty for no passphrase"
	case stepConfirmPass:
		return "Type your passphrase again to confirm"
	default:
		return ""
	}
}
