// Package wizard provides an interactive setup wizard for creating identities.
package wizard

// Step constants for the wizard flow
const (
	stepName           = 0
	stepEmail          = 1
	stepSSH            = 2
	stepSSHKeyPath     = 3  // New: enter existing SSH key path
	stepSSHKeyType     = 4  // Moved: was 3
	stepSSHPassphrase  = 5  // Moved: was 4
	stepSSHConfirmPass = 6  // Moved: was 5
	stepGPG            = 7  // Moved: was 6
	stepGPGKeyID       = 8  // New: enter existing GPG key ID
	stepGPGPassphrase  = 9  // Moved: was 7
	stepGPGConfirmPass = 10 // Moved: was 8
)

// sshOptions are the choices for SSH key handling
var sshOptions = []string{
	"Generate new SSH key",
	"Use existing SSH key",
	"Skip SSH setup",
}

// sshChoiceGenerate is the index for generating a new SSH key
const sshChoiceGenerate = 0

// sshChoiceUseExisting is the index for using an existing SSH key
const sshChoiceUseExisting = 1

// sshChoiceSkip is the index for skipping SSH setup
const sshChoiceSkip = 2

// gpgOptions are the choices for GPG key handling
var gpgOptions = []string{
	"Generate new GPG key for commit signing",
	"Use existing GPG key",
	"Skip GPG setup (can add later)",
}

// gpgChoiceGenerate is the index for generating a new GPG key
const gpgChoiceGenerate = 0

// gpgChoiceUseExisting is the index for using an existing GPG key
const gpgChoiceUseExisting = 1

// gpgChoiceSkip is the index for skipping GPG setup
const gpgChoiceSkip = 2

// sshKeyTypeOptions are the choices for SSH key type
var sshKeyTypeOptions = []string{
	"Ed25519 (recommended, modern)",
	"RSA 4096-bit (Azure DevOps compatible)",
}

// sshKeyTypeEd25519 is the index for Ed25519 key type
const sshKeyTypeEd25519 = 0

// sshKeyTypeRSA is the index for RSA key type
const sshKeyTypeRSA = 1

// getTotalSteps returns the total number of steps based on SSH and GPG choices.
func getTotalSteps(sshChoice, gpgChoice int, sshPassphraseEmpty, gpgPassphraseEmpty bool) int {
	total := 3 // name, email, ssh choice

	// Add SSH steps based on choice
	switch sshChoice {
	case sshChoiceGenerate:
		total++ // key type step
		if sshPassphraseEmpty {
			total++ // just passphrase step
		} else {
			total += 2 // passphrase + confirm
		}
	case sshChoiceUseExisting:
		total++ // key path step
	}

	// Always add GPG choice step
	total++

	// Add GPG steps based on choice
	switch gpgChoice {
	case gpgChoiceGenerate:
		if gpgPassphraseEmpty {
			total++ // just passphrase step
		} else {
			total += 2 // passphrase + confirm
		}
	case gpgChoiceUseExisting:
		total++ // key ID step
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
	case stepSSHKeyPath:
		return "Enter the path to your existing SSH private key"
	case stepSSHKeyType:
		return "Which SSH key type would you like?"
	case stepSSHPassphrase:
		return "Enter a passphrase for your SSH key (optional, press Enter to skip)"
	case stepSSHConfirmPass:
		return "Confirm your SSH passphrase"
	case stepGPG:
		return "Would you like to set up a GPG key for commit signing?"
	case stepGPGKeyID:
		return "Enter your existing GPG key ID"
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
	case stepSSHKeyPath:
		return "e.g., ~/.ssh/id_ed25519 or ~/.ssh/id_rsa"
	case stepSSHKeyType:
		return "Ed25519 is recommended. Use RSA if you need Azure DevOps compatibility."
	case stepSSHPassphrase:
		return "Leave empty for no passphrase"
	case stepSSHConfirmPass:
		return "Type your passphrase again to confirm"
	case stepGPG:
		return "GPG keys enable verified commit signing on GitHub/GitLab"
	case stepGPGKeyID:
		return "Run 'gpg --list-secret-keys' to find your key ID"
	case stepGPGPassphrase:
		return "Leave empty for no passphrase"
	case stepGPGConfirmPass:
		return "Type your passphrase again to confirm"
	default:
		return ""
	}
}
