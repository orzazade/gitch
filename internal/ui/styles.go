// Package ui provides styled terminal output using lipgloss.
package ui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	ActiveColor   = lipgloss.Color("10") // Green (also used for success)
	inactiveColor = lipgloss.Color("8")  // Gray
	nameColor     = lipgloss.Color("15") // White
	emailColor    = lipgloss.Color("7")  // Light gray
	warningColor  = lipgloss.Color("11") // Yellow
	errorColor    = lipgloss.Color("9")  // Red
)

// Card styles
var (
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(inactiveColor).
			Padding(0, 1).
			MarginBottom(1)

	ActiveCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ActiveColor).
			Padding(0, 1).
			MarginBottom(1)
)

// Text styles
var (
	NameStyle      = lipgloss.NewStyle().Foreground(nameColor).Bold(true)
	EmailStyle     = lipgloss.NewStyle().Foreground(emailColor)
	CheckmarkStyle = lipgloss.NewStyle().Foreground(ActiveColor).Bold(true)
	WarningStyle   = lipgloss.NewStyle().Foreground(warningColor)
	ErrorStyle     = lipgloss.NewStyle().Foreground(errorColor)
	SuccessStyle   = lipgloss.NewStyle().Foreground(ActiveColor)
	DimStyle       = lipgloss.NewStyle().Foreground(inactiveColor)
)
