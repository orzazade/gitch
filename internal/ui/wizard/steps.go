// Package wizard provides an interactive setup wizard for creating identities.
package wizard

// Step constants for the wizard flow
const (
	stepName           = 0
	stepEmail          = 1
	stepSSH            = 2
	stepSSHPassphrase  = 3
	stepSSHConfirmPass = 4
	stepGPG            = 5
	stepGPGPassphrase  = 6
	stepGPGConfirmPass = 7
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

// gpgOptions are the choices for GPG key handling
var gpgOptions = []string{
	"Generate new GPG key for commit signing",
	"Skip GPG setup (can add later)",
}

// gpgChoiceGenerate is the index for generating a new GPG key
const gpgChoiceGenerate = 0

// gpgChoiceSkip is the index for skipping GPG setup
const gpgChoiceSkip = 1

// getTotalSteps returns the total number of steps based on SSH and GPG choices.
func getTotalSteps(sshChoice, gpgChoice int, sshPassphraseEmpty, gpgPassphraseEmpty bool) int {
	total := 3 // name, email, ssh choice

	// Add SSH steps if generating
	if sshChoice == sshChoiceGenerate {
		if sshPassphraseEmpty {
			total++ // just passphrase step
		} else {
			total += 2 // passphrase + confirm
		}
	}

	// Always add GPG choice step
	total++

	// Add GPG steps if generating
	if gpgChoice == gpgChoiceGenerate {
		if gpgPassphraseEmpty {
			total++ // just passphrase step
		} else {
			total += 2 // passphrase + confirm
		}
	}

	return total
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
	case stepSSHPassphrase:
		return "Enter a passphrase for your SSH key (optional, press Enter to skip)"
	case stepSSHConfirmPass:
		return "Confirm your SSH passphrase"
	case stepGPG:
		return "Would you like to set up a GPG key for commit signing?"
	case stepGPGPassphrase:
		return "Enter a passphrase for your GPG key (optional, press Enter to skip)"
	case stepGPGConfirmPass:
		return "Confirm your GPG passphrase"
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
	case stepSSHPassphrase:
		return "Leave empty for no passphrase"
	case stepSSHConfirmPass:
		return "Type your passphrase again to confirm"
	case stepGPG:
		return "GPG keys enable verified commit signing on GitHub/GitLab"
	case stepGPGPassphrase:
		return "Leave empty for no passphrase"
	case stepGPGConfirmPass:
		return "Type your passphrase again to confirm"
	default:
		return ""
	}
}
