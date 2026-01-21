// Package ui provides styled terminal output using lipgloss.
package ui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	ActiveColor   = lipgloss.Color("10") // Green
	InactiveColor = lipgloss.Color("8")  // Gray
	NameColor     = lipgloss.Color("15") // White
	EmailColor    = lipgloss.Color("7")  // Light gray
	WarningColor  = lipgloss.Color("11") // Yellow
	ErrorColor    = lipgloss.Color("9")  // Red
	SuccessColor  = lipgloss.Color("10") // Green
)

// Card styles
var (
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(InactiveColor).
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
	NameStyle      = lipgloss.NewStyle().Foreground(NameColor).Bold(true)
	EmailStyle     = lipgloss.NewStyle().Foreground(EmailColor)
	CheckmarkStyle = lipgloss.NewStyle().Foreground(ActiveColor).Bold(true)
	WarningStyle   = lipgloss.NewStyle().Foreground(WarningColor)
	ErrorStyle     = lipgloss.NewStyle().Foreground(ErrorColor)
	SuccessStyle   = lipgloss.NewStyle().Foreground(SuccessColor)
	DimStyle       = lipgloss.NewStyle().Foreground(InactiveColor)
)
