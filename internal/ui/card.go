package ui

import (
	"strings"

	"github.com/orkhanrz/gitch/internal/config"
)

// RenderIdentityCard renders a single identity as a styled card.
// If active: uses ActiveCardStyle with checkmark prefix.
// If not active: uses CardStyle with space prefix.
// If isDefault: appends "(default)" after the name.
func RenderIdentityCard(name, email string, isActive, isDefault bool) string {
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
	content := nameLine + "\n" + emailLine

	return style.Render(content)
}

// RenderIdentityList renders all identities as cards.
// The active identity is determined by matching email to the activeEmail parameter.
// The default identity is determined by matching name to the defaultName parameter.
func RenderIdentityList(identities []config.Identity, activeEmail, defaultName string) string {
	if len(identities) == 0 {
		return ""
	}

	var cards []string
	for _, identity := range identities {
		isActive := strings.EqualFold(identity.Email, activeEmail)
		isDefault := strings.EqualFold(identity.Name, defaultName)
		card := RenderIdentityCard(identity.Name, identity.Email, isActive, isDefault)
		cards = append(cards, card)
	}

	return strings.Join(cards, "\n")
}
