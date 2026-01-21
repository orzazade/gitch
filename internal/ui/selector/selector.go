// Package selector provides an interactive identity selector TUI.
package selector

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/ui"
)

// Model is the Bubble Tea model for the identity selector.
type Model struct {
	identities  []config.Identity
	cursor      int
	activeEmail string
	defaultName string
	Selected    *config.Identity
	Cancelled   bool
}

// New creates a new selector model.
// The cursor starts on the currently active identity (for quick re-confirmation).
func New(identities []config.Identity, activeEmail, defaultName string) Model {
	return Model{
		identities:  identities,
		cursor:      findActiveIndex(identities, activeEmail),
		activeEmail: activeEmail,
		defaultName: defaultName,
	}
}

// findActiveIndex returns the index of the active identity, or 0.
func findActiveIndex(identities []config.Identity, activeEmail string) int {
	for i, id := range identities {
		if strings.EqualFold(id.Email, activeEmail) {
			return i
		}
	}
	return 0
}

// Init is the Bubble Tea init function.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input and updates the model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Store for potential future use (terminal resize)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Cancelled = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.identities)-1 {
				m.cursor++
			}
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = len(m.identities) - 1
		case "enter":
			m.Selected = &m.identities[m.cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the selector UI.
func (m Model) View() string {
	if len(m.identities) == 0 {
		return "No identities configured.\n\n" +
			ui.DimStyle.Render("Run 'gitch setup' to create one.")
	}

	var b strings.Builder

	b.WriteString("Select an identity:\n\n")

	for i, identity := range m.identities {
		isActive := strings.EqualFold(identity.Email, m.activeEmail)
		isDefault := strings.EqualFold(identity.Name, m.defaultName)
		hasSSH := identity.SSHKeyPath != ""
		isCursor := i == m.cursor

		card := renderSelectableCard(identity, isActive, isDefault, hasSSH, isCursor)
		b.WriteString(card)
		b.WriteString("\n")
	}

	b.WriteString(ui.DimStyle.Render("Up/Down Navigate  Enter Select  q Quit"))

	return b.String()
}

// renderSelectableCard renders an identity card with cursor highlighting.
// When cursor is on this card, use ActiveCardStyle (green border).
// When this is the active identity, show checkmark.
// Both can be true (cursor on active identity).
func renderSelectableCard(id config.Identity, active, def, ssh, cursor bool) string {
	// Card style based on cursor position
	style := ui.CardStyle
	if cursor {
		style = ui.ActiveCardStyle
	}

	// Checkmark for active identity (same as card.go)
	var prefix string
	if active {
		prefix = ui.CheckmarkStyle.Render("\u2713 ") // checkmark indicator
	} else {
		prefix = "  "
	}

	// Name line with optional default marker
	nameLine := prefix + ui.NameStyle.Render(id.Name)
	if def {
		nameLine += " (default)"
	}

	// Email line
	emailLine := "  " + ui.EmailStyle.Render(id.Email)

	// Build content
	var content strings.Builder
	content.WriteString(nameLine)
	content.WriteString("\n")
	content.WriteString(emailLine)
	if ssh {
		content.WriteString("\n  ")
		content.WriteString(ui.DimStyle.Render("SSH configured"))
	}

	return style.Render(content.String())
}

// Run launches the selector and returns the selected identity.
// Returns nil if cancelled or no selection made.
func Run(identities []config.Identity, activeEmail, defaultName string) (*config.Identity, error) {
	if len(identities) == 0 {
		return nil, nil
	}

	m := New(identities, activeEmail, defaultName)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(Model)
	if result.Cancelled {
		return nil, nil
	}

	return result.Selected, nil
}
