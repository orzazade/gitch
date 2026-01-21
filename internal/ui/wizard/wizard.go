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
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
)

// WizardResult holds the collected data from the wizard
type WizardResult struct {
	Name        string
	Email       string
	SSHKeyPath  string
	GenerateSSH bool
}

// Model is the Bubble Tea model for the setup wizard
type Model struct {
	step            int
	nameInput       textinput.Model
	emailInput      textinput.Model
	sshChoice       int
	passphraseInput textinput.Model
	confirmInput    textinput.Model
	spinner         spinner.Model
	progress        progress.Model
	loading         bool
	err             error
	warning         string // non-fatal warning message
	done            bool
	Cancelled       bool
	result          *WizardResult
	passphrase      []byte
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

	// Passphrase input (hidden)
	passphraseInput := textinput.New()
	passphraseInput.Placeholder = ""
	passphraseInput.EchoMode = textinput.EchoPassword
	passphraseInput.EchoCharacter = '*'
	passphraseInput.CharLimit = 100
	passphraseInput.Width = 40

	// Confirm passphrase input (hidden)
	confirmInput := textinput.New()
	confirmInput.Placeholder = ""
	confirmInput.EchoMode = textinput.EchoPassword
	confirmInput.EchoCharacter = '*'
	confirmInput.CharLimit = 100
	confirmInput.Width = 40

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

	return Model{
		step:            stepName,
		nameInput:       nameInput,
		emailInput:      emailInput,
		sshChoice:       sshChoiceGenerate,
		passphraseInput: passphraseInput,
		confirmInput:    confirmInput,
		spinner:         s,
		progress:        p,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle SSH key generation result
	switch msg := msg.(type) {
	case sshKeyGenerated:
		m.loading = false
		m.result.SSHKeyPath = msg.keyPath
		m.done = true
		return m, tea.Quit

	case sshKeyError:
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
			// If going back from confirm passphrase, go back to passphrase
			// If going back from passphrase, go back to SSH choice
			m.step--
			// Reset confirm input when going back
			if m.step == stepPassphrase {
				m.confirmInput.Reset()
			}
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

		case "down", "j":
			if m.step == stepSSH {
				if m.sshChoice < len(sshOptions)-1 {
					m.sshChoice++
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
	case stepPassphrase:
		m.passphraseInput, cmd = m.passphraseInput.Update(msg)
	case stepConfirmPass:
		m.confirmInput, cmd = m.confirmInput.Update(msg)
	}

	return m, cmd
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
		m.step++
		return m, m.emailInput.Focus()

	case stepEmail:
		email := strings.TrimSpace(m.emailInput.Value())
		if err := config.ValidateEmail(email); err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		m.step++
		return m, nil

	case stepSSH:
		m.err = nil
		m.warning = ""
		if m.sshChoice == sshChoiceSkip {
			// Skip SSH, complete the wizard
			m.result = &WizardResult{
				Name:        strings.TrimSpace(m.nameInput.Value()),
				Email:       strings.TrimSpace(m.emailInput.Value()),
				GenerateSSH: false,
			}
			m.done = true
			return m, tea.Quit
		}
		// Check if key already exists and warn
		keyPath := sshpkg.DefaultSSHKeyPath(strings.TrimSpace(m.nameInput.Value()))
		if _, err := os.Stat(keyPath); err == nil {
			m.warning = fmt.Sprintf("SSH key already exists at %s (will be overwritten)", keyPath)
		}
		// Continue to passphrase step
		m.step++
		return m, m.passphraseInput.Focus()

	case stepPassphrase:
		passphrase := m.passphraseInput.Value()
		m.passphrase = []byte(passphrase)
		m.err = nil

		// If passphrase is empty, skip confirmation (per Phase 2 decision)
		if passphrase == "" {
			return m.startSSHKeyGeneration()
		}

		// Continue to confirmation step
		m.step++
		return m, m.confirmInput.Focus()

	case stepConfirmPass:
		confirm := m.confirmInput.Value()
		if confirm != m.passphraseInput.Value() {
			m.err = fmt.Errorf("passphrases don't match")
			m.confirmInput.Reset()
			return m, m.confirmInput.Focus()
		}
		m.err = nil
		return m.startSSHKeyGeneration()
	}

	return m, nil
}

// startSSHKeyGeneration initiates SSH key generation
func (m Model) startSSHKeyGeneration() (tea.Model, tea.Cmd) {
	m.loading = true
	m.result = &WizardResult{
		Name:        strings.TrimSpace(m.nameInput.Value()),
		Email:       strings.TrimSpace(m.emailInput.Value()),
		GenerateSSH: true,
	}

	return m, tea.Batch(
		m.spinner.Tick,
		generateSSHKeyCmd(m.result.Name, m.result.Email, m.passphrase),
	)
}

// generateSSHKeyCmd returns a command that generates an SSH keypair
func generateSSHKeyCmd(name, email string, passphrase []byte) tea.Cmd {
	return func() tea.Msg {
		keyPath := sshpkg.DefaultSSHKeyPath(name)
		if keyPath == "" {
			return sshKeyError{fmt.Errorf("failed to determine SSH key path")}
		}

		privateKey, publicKey, err := sshpkg.GenerateKeyPair(email, passphrase)
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

// focusCurrentInput returns a command to focus the current step's input
func (m Model) focusCurrentInput() tea.Cmd {
	switch m.step {
	case stepName:
		return m.nameInput.Focus()
	case stepEmail:
		return m.emailInput.Focus()
	case stepPassphrase:
		return m.passphraseInput.Focus()
	case stepConfirmPass:
		return m.confirmInput.Focus()
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
		b.WriteString(" Generating SSH key...\n")
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

	case stepPassphrase:
		b.WriteString("  > ")
		b.WriteString(m.passphraseInput.View())
		b.WriteString("\n")

	case stepConfirmPass:
		b.WriteString("  > ")
		b.WriteString(m.confirmInput.View())
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
	passphraseEmpty := m.passphraseInput.Value() == ""
	total := getTotalSteps(m.sshChoice, passphraseEmpty)

	// Adjust displayed step for progress calculation
	displayStep := m.step + 1
	if displayStep > total {
		displayStep = total
	}

	percent := float64(displayStep) / float64(total)
	progressBar := m.progress.ViewAs(percent)

	stepText := ui.DimStyle.Render(fmt.Sprintf(" Step %d of %d", displayStep, total))

	return progressBar + stepText
}

// renderHints renders the keyboard hints based on current step
func (m Model) renderHints() string {
	var hints string
	switch m.step {
	case stepName:
		hints = "Enter Continue  Esc Quit"
	case stepSSH:
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
