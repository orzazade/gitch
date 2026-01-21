package ui

import (
	"strings"

	"github.com/orzazade/gitch/internal/config"
)

// RenderIdentityCard renders a single identity as a styled card.
// If active: uses ActiveCardStyle with checkmark prefix.
// If not active: uses CardStyle with space prefix.
// If isDefault: appends "(default)" after the name.
// If hasSSHKey: shows SSH indicator on a third line.
func RenderIdentityCard(name, email string, isActive, isDefault, hasSSHKey bool) string {
	var prefix string
	var style = CardStyle

	if isActive {
		prefix = CheckmarkStyle.Render("✓ ")
		style = ActiveCardStyle
	} else {
		prefix = "  "
	}

	// Build name line with optional default marker
	nameLine := prefix + NameStyle.Render(name)
	if isDefault {
		nameLine += " (default)"
	}

	// Build email line with same indentation
	emailLine := "  " + EmailStyle.Render(email)

	// Combine into card content
	var content strings.Builder
	content.WriteString(nameLine)
	content.WriteString("\n")
	content.WriteString(emailLine)
	if hasSSHKey {
		content.WriteString("\n")
		content.WriteString("  ")
		content.WriteString(DimStyle.Render("SSH configured"))
	}

	return style.Render(content.String())
}

// RenderIdentityList renders all identities as cards.
// The active identity is determined by matching email to the activeEmail parameter.
// The default identity is determined by matching name to the defaultName parameter.
// SSH key status is determined by checking if identity.SSHKeyPath is set.
func RenderIdentityList(identities []config.Identity, activeEmail, defaultName string) string {
	if len(identities) == 0 {
		return ""
	}

	var cards []string
	for _, identity := range identities {
		isActive := strings.EqualFold(identity.Email, activeEmail)
		isDefault := strings.EqualFold(identity.Name, defaultName)
		hasSSHKey := identity.SSHKeyPath != ""
		card := RenderIdentityCard(identity.Name, identity.Email, isActive, isDefault, hasSSHKey)
		cards = append(cards, card)
	}

	return strings.Join(cards, "\n")
}
