// Package wizard provides an interactive setup wizard for creating identities.
package wizard

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	gpgpkg "github.com/orzazade/gitch/internal/gpg"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
)

// WizardResult holds the collected data from the wizard
type WizardResult struct {
	Name           string
	Email          string
	SSHKeyPath     string
	SSHKeyType     string // "ed25519" or "rsa"
	GenerateSSH    bool
	UseExistingSSH bool
	GPGKeyID       string
	GenerateGPG    bool
	UseExistingGPG bool
}

// Model is the Bubble Tea model for the setup wizard
type Model struct {
	step                 int
	nameInput            textinput.Model
	emailInput           textinput.Model
	sshChoice            int
	sshKeyPathInput      textinput.Model // for existing SSH key path
	sshKeyTypeChoice     int             // 0 = Ed25519, 1 = RSA
	isAzureDevOps        bool            // auto-detected Azure DevOps remote
	sshPassphraseInput   textinput.Model
	sshConfirmInput      textinput.Model
	gpgChoice            int
	gpgKeyIDInput        textinput.Model // for existing GPG key ID
	gpgPassphraseInput   textinput.Model
	gpgConfirmInput      textinput.Model
	spinner              spinner.Model
	progress             progress.Model
	loading              bool
	loadingMessage       string
	err                  error
	warning              string // non-fatal warning message
	done                 bool
	Cancelled            bool
	result               *WizardResult
	sshPassphrase        []byte
	gpgPassphrase        []byte
	generatedSSHKeyPath  string // track SSH result for later
	generatedGPGKeyID    string // track GPG result for later
	existingSSHKeyPath   string // track existing SSH key path
	existingGPGKeyID     string // track existing GPG key ID
}

// titleStyle is the style for the wizard header
var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ui.ActiveColor).
	MarginBottom(1)

// sshKeyGenerated is a message sent when SSH key generation completes
type sshKeyGenerated struct {
	keyPath     string
	fingerprint string
}

// sshKeyError is a message sent when SSH key generation fails
type sshKeyError struct {
	err error
}

// gpgKeyGenerated is a message sent when GPG key generation completes
type gpgKeyGenerated struct {
	keyID string
}

// gpgKeyError is a message sent when GPG key generation fails
type gpgKeyError struct {
	err error
}

// New creates a new wizard Model
func New() Model {
	// Name input
	nameInput := textinput.New()
	nameInput.Placeholder = "work"
	nameInput.Focus()
	nameInput.CharLimit = 50
	nameInput.Width = 40

	// Email input
	emailInput := textinput.New()
	emailInput.Placeholder = "you@example.com"
	emailInput.CharLimit = 100
	emailInput.Width = 40

	// SSH Key Path input (for existing key)
	sshKeyPathInput := textinput.New()
	sshKeyPathInput.Placeholder = "~/.ssh/id_ed25519"
	sshKeyPathInput.CharLimit = 200
	sshKeyPathInput.Width = 40

	// SSH Passphrase input (hidden)
	sshPassphraseInput := textinput.New()
	sshPassphraseInput.Placeholder = ""
	sshPassphraseInput.EchoMode = textinput.EchoPassword
	sshPassphraseInput.EchoCharacter = '*'
	sshPassphraseInput.CharLimit = 100
	sshPassphraseInput.Width = 40

	// SSH Confirm passphrase input (hidden)
	sshConfirmInput := textinput.New()
	sshConfirmInput.Placeholder = ""
	sshConfirmInput.EchoMode = textinput.EchoPassword
	sshConfirmInput.EchoCharacter = '*'
	sshConfirmInput.CharLimit = 100
	sshConfirmInput.Width = 40

	// GPG Key ID input (for existing key)
	gpgKeyIDInput := textinput.New()
	gpgKeyIDInput.Placeholder = "ABCD1234EFGH5678"
	gpgKeyIDInput.CharLimit = 50
	gpgKeyIDInput.Width = 40

	// GPG Passphrase input (hidden)
	gpgPassphraseInput := textinput.New()
	gpgPassphraseInput.Placeholder = ""
	gpgPassphraseInput.EchoMode = textinput.EchoPassword
	gpgPassphraseInput.EchoCharacter = '*'
	gpgPassphraseInput.CharLimit = 100
	gpgPassphraseInput.Width = 40

	// GPG Confirm passphrase input (hidden)
	gpgConfirmInput := textinput.New()
	gpgConfirmInput.Placeholder = ""
	gpgConfirmInput.EchoMode = textinput.EchoPassword
	gpgConfirmInput.EchoCharacter = '*'
	gpgConfirmInput.CharLimit = 100
	gpgConfirmInput.Width = 40

	// Spinner for loading state
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.SuccessStyle

	// Progress bar
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	// Detect Azure DevOps remote for key type default
	isAzureDevOps, _ := gitpkg.GetCurrentRemoteType()
	sshKeyTypeDefault := sshKeyTypeEd25519
	if isAzureDevOps {
		sshKeyTypeDefault = sshKeyTypeRSA
	}

	return Model{
		step:               stepName,
		nameInput:          nameInput,
		emailInput:         emailInput,
		sshChoice:          sshChoiceGenerate,
		sshKeyPathInput:    sshKeyPathInput,
		sshKeyTypeChoice:   sshKeyTypeDefault,
		isAzureDevOps:      isAzureDevOps,
		sshPassphraseInput: sshPassphraseInput,
		sshConfirmInput:    sshConfirmInput,
		gpgChoice:          gpgChoiceGenerate,
		gpgKeyIDInput:      gpgKeyIDInput,
		gpgPassphraseInput: gpgPassphraseInput,
		gpgConfirmInput:    gpgConfirmInput,
		spinner:            s,
		progress:           p,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle key generation results
	switch msg := msg.(type) {
	case sshKeyGenerated:
		m.loading = false
		m.generatedSSHKeyPath = msg.keyPath
		// After SSH generation, move to GPG step
		m.step = stepGPG
		return m, nil

	case sshKeyError:
		m.loading = false
		m.err = msg.err
		return m, nil

	case gpgKeyGenerated:
		m.loading = false
		m.generatedGPGKeyID = msg.keyID
		// GPG generation complete, finish wizard
		m.result = m.buildResult(true, false)
		m.done = true
		return m, tea.Quit

	case gpgKeyError:
		m.loading = false
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Handle global keys
		switch msg.String() {
		case "ctrl+c":
			m.Cancelled = true
			return m, tea.Quit

		case "esc":
			if m.step == stepName {
				m.Cancelled = true
				return m, tea.Quit
			}
			// Go back to previous step
			m.err = nil
			m.warning = ""
			m.step = m.getPreviousStep()
			return m, m.focusCurrentInput()

		case "enter":
			return m.handleEnter()

		case "up", "k":
			if m.step == stepSSH {
				if m.sshChoice > 0 {
					m.sshChoice--
				}
				return m, nil
			}
			if m.step == stepSSHKeyType {
				if m.sshKeyTypeChoice > 0 {
					m.sshKeyTypeChoice--
				}
				return m, nil
			}
			if m.step == stepGPG {
				if m.gpgChoice > 0 {
					m.gpgChoice--
				}
				return m, nil
			}

		case "down", "j":
			if m.step == stepSSH {
				if m.sshChoice < len(sshOptions)-1 {
					m.sshChoice++
				}
				return m, nil
			}
			if m.step == stepSSHKeyType {
				if m.sshKeyTypeChoice < len(sshKeyTypeOptions)-1 {
					m.sshKeyTypeChoice++
				}
				return m, nil
			}
			if m.step == stepGPG {
				if m.gpgChoice < len(gpgOptions)-1 {
					m.gpgChoice++
				}
				return m, nil
			}
		}
	}

	// Update the current input
	switch m.step {
	case stepName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case stepEmail:
		m.emailInput, cmd = m.emailInput.Update(msg)
	case stepSSHKeyPath:
		m.sshKeyPathInput, cmd = m.sshKeyPathInput.Update(msg)
	case stepSSHPassphrase:
		m.sshPassphraseInput, cmd = m.sshPassphraseInput.Update(msg)
	case stepSSHConfirmPass:
		m.sshConfirmInput, cmd = m.sshConfirmInput.Update(msg)
	case stepGPGKeyID:
		m.gpgKeyIDInput, cmd = m.gpgKeyIDInput.Update(msg)
	case stepGPGPassphrase:
		m.gpgPassphraseInput, cmd = m.gpgPassphraseInput.Update(msg)
	case stepGPGConfirmPass:
		m.gpgConfirmInput, cmd = m.gpgConfirmInput.Update(msg)
	}

	return m, cmd
}

// getPreviousStep returns the step to go back to
func (m Model) getPreviousStep() int {
	switch m.step {
	case stepEmail:
		return stepName
	case stepSSH:
		return stepEmail
	case stepSSHKeyPath:
		return stepSSH
	case stepSSHKeyType:
		return stepSSH
	case stepSSHPassphrase:
		return stepSSHKeyType
	case stepSSHConfirmPass:
		m.sshConfirmInput.Reset()
		return stepSSHPassphrase
	case stepGPG:
		// Go back based on SSH choice
		switch m.sshChoice {
		case sshChoiceGenerate:
			if m.sshPassphraseInput.Value() == "" {
				return stepSSHPassphrase
			}
			return stepSSHConfirmPass
		case sshChoiceUseExisting:
			return stepSSHKeyPath
		default:
			return stepSSH
		}
	case stepGPGKeyID:
		return stepGPG
	case stepGPGPassphrase:
		return stepGPG
	case stepGPGConfirmPass:
		m.gpgConfirmInput.Reset()
		return stepGPGPassphrase
	default:
		return m.step - 1
	}
}

// handleEnter processes the Enter key for the current step
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case stepName:
		name := strings.TrimSpace(m.nameInput.Value())
		if err := config.ValidateName(name); err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		m.step = stepEmail
		return m, m.emailInput.Focus()

	case stepEmail:
		email := strings.TrimSpace(m.emailInput.Value())
		if err := config.ValidateEmail(email); err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		m.step = stepSSH
		return m, nil

	case stepSSH:
		m.err = nil
		m.warning = ""
		switch m.sshChoice {
		case sshChoiceSkip:
			// Skip SSH, continue to GPG step
			m.step = stepGPG
			return m, nil
		case sshChoiceUseExisting:
			// Use existing key, go to key path input
			m.step = stepSSHKeyPath
			return m, m.sshKeyPathInput.Focus()
		default:
			// Generate new key
			// Check if key already exists and warn
			keyPath := sshpkg.DefaultSSHKeyPath(strings.TrimSpace(m.nameInput.Value()))
			if _, err := os.Stat(keyPath); err == nil {
				m.warning = fmt.Sprintf("SSH key already exists at %s (will be overwritten)", keyPath)
			}
			// Continue to SSH key type step
			m.step = stepSSHKeyType
			return m, nil
		}

	case stepSSHKeyPath:
		keyPath := strings.TrimSpace(m.sshKeyPathInput.Value())
		if keyPath == "" {
			m.err = fmt.Errorf("please enter a key path")
			return m, nil
		}
		// Validate the key exists and is valid
		if err := sshpkg.ValidateKeyPath(keyPath); err != nil {
			m.err = err
			return m, nil
		}
		// Store the path and continue to GPG
		expandedPath, _ := sshpkg.ExpandPath(keyPath)
		m.existingSSHKeyPath = expandedPath
		m.err = nil
		m.step = stepGPG
		return m, nil

	case stepSSHKeyType:
		m.err = nil
		// Show warning if Ed25519 selected with Azure DevOps
		if m.sshKeyTypeChoice == sshKeyTypeEd25519 && m.isAzureDevOps {
			m.warning = "Ed25519 keys may not work with Azure DevOps"
		} else {
			m.warning = ""
		}
		// Continue to SSH passphrase step
		m.step = stepSSHPassphrase
		return m, m.sshPassphraseInput.Focus()

	case stepSSHPassphrase:
		passphrase := m.sshPassphraseInput.Value()
		m.sshPassphrase = []byte(passphrase)
		m.err = nil

		// If passphrase is empty, skip confirmation
		if passphrase == "" {
			return m.startSSHKeyGeneration()
		}

		// Continue to confirmation step
		m.step = stepSSHConfirmPass
		return m, m.sshConfirmInput.Focus()

	case stepSSHConfirmPass:
		confirm := m.sshConfirmInput.Value()
		if confirm != m.sshPassphraseInput.Value() {
			m.err = fmt.Errorf("passphrases don't match")
			m.sshConfirmInput.Reset()
			return m, m.sshConfirmInput.Focus()
		}
		m.err = nil
		return m.startSSHKeyGeneration()

	case stepGPG:
		m.err = nil
		m.warning = ""
		switch m.gpgChoice {
		case gpgChoiceSkip:
			// Skip GPG, complete the wizard
			m.result = m.buildResult(false, false)
			m.done = true
			return m, tea.Quit
		case gpgChoiceUseExisting:
			// Use existing GPG key, go to key ID input
			if !gpgpkg.IsGPGAvailable() {
				m.err = fmt.Errorf("gpg command not found - install GPG to use GPG features")
				return m, nil
			}
			m.step = stepGPGKeyID
			return m, m.gpgKeyIDInput.Focus()
		default:
			// Generate new key
			// Check if GPG is available
			if !gpgpkg.IsGPGAvailable() {
				m.err = fmt.Errorf("gpg command not found - install GPG to generate keys")
				return m, nil
			}
			// Continue to GPG passphrase step
			m.step = stepGPGPassphrase
			return m, m.gpgPassphraseInput.Focus()
		}

	case stepGPGKeyID:
		keyID := strings.TrimSpace(m.gpgKeyIDInput.Value())
		if keyID == "" {
			m.err = fmt.Errorf("please enter a GPG key ID")
			return m, nil
		}
		// Validate the key exists
		if err := gpgpkg.ValidateKeyID(keyID); err != nil {
			m.err = err
			return m, nil
		}
		// Store the key ID and complete wizard
		m.existingGPGKeyID = keyID
		m.err = nil
		m.result = m.buildResult(false, true)
		m.done = true
		return m, tea.Quit

	case stepGPGPassphrase:
		passphrase := m.gpgPassphraseInput.Value()
		m.gpgPassphrase = []byte(passphrase)
		m.err = nil

		// If passphrase is empty, skip confirmation
		if passphrase == "" {
			return m.startGPGKeyGeneration()
		}

		// Continue to confirmation step
		m.step = stepGPGConfirmPass
		return m, m.gpgConfirmInput.Focus()

	case stepGPGConfirmPass:
		confirm := m.gpgConfirmInput.Value()
		if confirm != m.gpgPassphraseInput.Value() {
			m.err = fmt.Errorf("passphrases don't match")
			m.gpgConfirmInput.Reset()
			return m, m.gpgConfirmInput.Focus()
		}
		m.err = nil
		return m.startGPGKeyGeneration()
	}

	return m, nil
}

// startSSHKeyGeneration initiates SSH key generation
func (m Model) startSSHKeyGeneration() (tea.Model, tea.Cmd) {
	m.loading = true
	keyTypeLabel := "Ed25519"
	if m.sshKeyTypeChoice == sshKeyTypeRSA {
		keyTypeLabel = "RSA 4096-bit"
	}
	m.loadingMessage = fmt.Sprintf("Generating %s SSH key...", keyTypeLabel)

	return m, tea.Batch(
		m.spinner.Tick,
		generateSSHKeyCmd(
			strings.TrimSpace(m.nameInput.Value()),
			strings.TrimSpace(m.emailInput.Value()),
			m.sshPassphrase,
			m.sshKeyTypeChoice,
		),
	)
}

// startGPGKeyGeneration initiates GPG key generation
func (m Model) startGPGKeyGeneration() (tea.Model, tea.Cmd) {
	m.loading = true
	m.loadingMessage = "Generating GPG key..."

	return m, tea.Batch(
		m.spinner.Tick,
		generateGPGKeyCmd(
			strings.TrimSpace(m.nameInput.Value()),
			strings.TrimSpace(m.emailInput.Value()),
			m.gpgPassphrase,
		),
	)
}

// generateSSHKeyCmd returns a command that generates an SSH keypair
func generateSSHKeyCmd(name, email string, passphrase []byte, keyTypeChoice int) tea.Cmd {
	return func() tea.Msg {
		keyPath := sshpkg.DefaultSSHKeyPath(name)
		if keyPath == "" {
			return sshKeyError{fmt.Errorf("failed to determine SSH key path")}
		}

		// Convert choice to KeyType
		keyType := sshpkg.KeyTypeEd25519
		if keyTypeChoice == sshKeyTypeRSA {
			keyType = sshpkg.KeyTypeRSA
		}

		privateKey, publicKey, err := sshpkg.GenerateKeyPairWithType(keyType, email, passphrase)
		if err != nil {
			return sshKeyError{err}
		}

		if err := sshpkg.WriteKeyFiles(keyPath, privateKey, publicKey); err != nil {
			return sshKeyError{err}
		}

		fingerprint, _ := sshpkg.GetFingerprint(publicKey)
		return sshKeyGenerated{keyPath: keyPath, fingerprint: fingerprint}
	}
}

// generateGPGKeyCmd returns a command that generates a GPG keypair
func generateGPGKeyCmd(name, email string, passphrase []byte) tea.Cmd {
	return func() tea.Msg {
		keyInfo, err := gpgpkg.GenerateKey(name, email, passphrase)
		if err != nil {
			return gpgKeyError{err}
		}
		return gpgKeyGenerated{keyID: keyInfo.ID}
	}
}

// focusCurrentInput returns a command to focus the current step's input
func (m Model) focusCurrentInput() tea.Cmd {
	switch m.step {
	case stepName:
		return m.nameInput.Focus()
	case stepEmail:
		return m.emailInput.Focus()
	case stepSSHKeyPath:
		return m.sshKeyPathInput.Focus()
	case stepSSHPassphrase:
		return m.sshPassphraseInput.Focus()
	case stepSSHConfirmPass:
		return m.sshConfirmInput.Focus()
	case stepGPGKeyID:
		return m.gpgKeyIDInput.Focus()
	case stepGPGPassphrase:
		return m.gpgPassphraseInput.Focus()
	case stepGPGConfirmPass:
		return m.gpgConfirmInput.Focus()
	}
	return nil
}

// View renders the wizard UI
func (m Model) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	// Wizard header
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  gitch setup"))
	b.WriteString("\n\n")

	// Progress bar
	b.WriteString("  ")
	b.WriteString(m.renderProgress())
	b.WriteString("\n\n")

	// Show spinner during key generation
	if m.loading {
		b.WriteString("  ")
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(m.loadingMessage)
		b.WriteString("\n")
		return b.String()
	}

	// Step title
	title := getStepTitle(m.step)
	b.WriteString("  ")
	b.WriteString(ui.NameStyle.Render(title))
	b.WriteString("\n\n")

	// Hint text
	hint := getStepHint(m.step)
	if hint != "" {
		b.WriteString("  ")
		b.WriteString(ui.DimStyle.Render(hint))
		b.WriteString("\n\n")
	}

	// Warning message (non-fatal)
	if m.warning != "" {
		b.WriteString("  ")
		b.WriteString(ui.WarningStyle.Render("Warning: " + m.warning))
		b.WriteString("\n\n")
	}

	// Input/Selection based on step
	switch m.step {
	case stepName:
		b.WriteString("  > ")
		b.WriteString(m.nameInput.View())
		b.WriteString("\n")

	case stepEmail:
		b.WriteString("  > ")
		b.WriteString(m.emailInput.View())
		b.WriteString("\n")

	case stepSSH:
		for i, option := range sshOptions {
			if i == m.sshChoice {
				b.WriteString("  ")
				b.WriteString(ui.SuccessStyle.Render("> " + option))
			} else {
				b.WriteString("    ")
				b.WriteString(ui.DimStyle.Render(option))
			}
			b.WriteString("\n")
		}

	case stepSSHKeyPath:
		b.WriteString("  > ")
		b.WriteString(m.sshKeyPathInput.View())
		b.WriteString("\n")

	case stepSSHKeyType:
		for i, option := range sshKeyTypeOptions {
			if i == m.sshKeyTypeChoice {
				b.WriteString("  ")
				b.WriteString(ui.SuccessStyle.Render("> " + option))
			} else {
				b.WriteString("    ")
				b.WriteString(ui.DimStyle.Render(option))
			}
			b.WriteString("\n")
		}
		// Show Azure DevOps detection info
		if m.isAzureDevOps {
			b.WriteString("\n  ")
			b.WriteString(ui.DimStyle.Render("(Azure DevOps detected - RSA recommended)"))
			b.WriteString("\n")
		}

	case stepSSHPassphrase:
		b.WriteString("  > ")
		b.WriteString(m.sshPassphraseInput.View())
		b.WriteString("\n")

	case stepSSHConfirmPass:
		b.WriteString("  > ")
		b.WriteString(m.sshConfirmInput.View())
		b.WriteString("\n")

	case stepGPG:
		for i, option := range gpgOptions {
			if i == m.gpgChoice {
				b.WriteString("  ")
				b.WriteString(ui.SuccessStyle.Render("> " + option))
			} else {
				b.WriteString("    ")
				b.WriteString(ui.DimStyle.Render(option))
			}
			b.WriteString("\n")
		}

	case stepGPGKeyID:
		b.WriteString("  > ")
		b.WriteString(m.gpgKeyIDInput.View())
		b.WriteString("\n")

	case stepGPGPassphrase:
		b.WriteString("  > ")
		b.WriteString(m.gpgPassphraseInput.View())
		b.WriteString("\n")

	case stepGPGConfirmPass:
		b.WriteString("  > ")
		b.WriteString(m.gpgConfirmInput.View())
		b.WriteString("\n")
	}

	// Error message
	if m.err != nil {
		b.WriteString("\n  ")
		b.WriteString(ui.ErrorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n")
	}

	// Keyboard hints
	b.WriteString("\n")
	b.WriteString(m.renderHints())
	b.WriteString("\n")

	return b.String()
}

// renderProgress renders the progress bar with step indicator
func (m Model) renderProgress() string {
	sshPassEmpty := m.sshPassphraseInput.Value() == ""
	gpgPassEmpty := m.gpgPassphraseInput.Value() == ""
	total := getTotalSteps(m.sshChoice, m.gpgChoice, sshPassEmpty, gpgPassEmpty)

	// Calculate current step number for display
	displayStep := m.getDisplayStep()
	if displayStep > total {
		displayStep = total
	}

	percent := float64(displayStep) / float64(total)
	progressBar := m.progress.ViewAs(percent)

	stepText := ui.DimStyle.Render(fmt.Sprintf(" Step %d of %d", displayStep, total))

	return progressBar + stepText
}

// getDisplayStep returns the current step number for progress display
func (m Model) getDisplayStep() int {
	switch m.step {
	case stepName:
		return 1
	case stepEmail:
		return 2
	case stepSSH:
		return 3
	case stepSSHKeyPath:
		return 4 // use existing path
	case stepSSHKeyType:
		return 4 // generate: key type selection
	case stepSSHPassphrase:
		return 5
	case stepSSHConfirmPass:
		return 6
	case stepGPG:
		return m.getGPGBaseStep()
	case stepGPGKeyID:
		return m.getGPGBaseStep() + 1
	case stepGPGPassphrase:
		return m.getGPGBaseStep() + 1
	case stepGPGConfirmPass:
		return m.getGPGBaseStep() + 2
	default:
		return m.step + 1
	}
}

// getGPGBaseStep returns the step number for the GPG choice step
func (m Model) getGPGBaseStep() int {
	base := 3 // name, email, ssh choice
	switch m.sshChoice {
	case sshChoiceSkip:
		base++ // just ssh choice
	case sshChoiceUseExisting:
		base += 2 // ssh choice + key path
	case sshChoiceGenerate:
		base++ // key type step
		if m.sshPassphraseInput.Value() == "" {
			base++ // passphrase only
		} else {
			base += 2 // passphrase + confirm
		}
	}
	return base
}

// getSSHKeyTypeString returns the key type as a string for the result
func (m Model) getSSHKeyTypeString() string {
	if m.sshChoice == sshChoiceSkip || m.sshChoice == sshChoiceUseExisting {
		return ""
	}
	if m.sshKeyTypeChoice == sshKeyTypeRSA {
		return "rsa"
	}
	return "ed25519"
}

// buildResult constructs the WizardResult based on current state
func (m Model) buildResult(gpgGenerated, gpgExisting bool) *WizardResult {
	// Determine SSH key path
	sshKeyPath := ""
	if m.sshChoice == sshChoiceGenerate {
		sshKeyPath = m.generatedSSHKeyPath
	} else if m.sshChoice == sshChoiceUseExisting {
		sshKeyPath = m.existingSSHKeyPath
	}

	// Determine GPG key ID
	gpgKeyID := ""
	if gpgGenerated {
		gpgKeyID = m.generatedGPGKeyID
	} else if gpgExisting {
		gpgKeyID = m.existingGPGKeyID
	}

	return &WizardResult{
		Name:           strings.TrimSpace(m.nameInput.Value()),
		Email:          strings.TrimSpace(m.emailInput.Value()),
		SSHKeyPath:     sshKeyPath,
		SSHKeyType:     m.getSSHKeyTypeString(),
		GenerateSSH:    m.sshChoice == sshChoiceGenerate,
		UseExistingSSH: m.sshChoice == sshChoiceUseExisting,
		GPGKeyID:       gpgKeyID,
		GenerateGPG:    gpgGenerated,
		UseExistingGPG: gpgExisting,
	}
}

// renderHints renders the keyboard hints based on current step
func (m Model) renderHints() string {
	var hints string
	switch m.step {
	case stepName:
		hints = "Enter Continue  Esc Quit"
	case stepSSH, stepSSHKeyType, stepGPG:
		hints = "Up/Down Select  Enter Confirm  Esc Back"
	default:
		hints = "Enter Continue  Esc Back"
	}
	return "  " + ui.DimStyle.Render(hints)
}

// Result returns the wizard result if completed successfully
func (m Model) Result() *WizardResult {
	return m.result
}

// Run launches the wizard and returns the result or error.
// This is a convenience wrapper around tea.NewProgram.
func Run() (*WizardResult, error) {
	m := New()
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(Model)
	if result.Cancelled {
		return nil, nil
	}

	return result.Result(), nil
}
