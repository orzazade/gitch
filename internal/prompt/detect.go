package prompt

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectedFramework represents a detected prompt framework
type DetectedFramework string

const (
	FrameworkNone       DetectedFramework = ""
	FrameworkStarship   DetectedFramework = "starship"
	FrameworkOhMyZsh    DetectedFramework = "oh-my-zsh"
	FrameworkPowerlevel DetectedFramework = "powerlevel10k"
)

// String returns the human-readable name of the framework
func (f DetectedFramework) String() string {
	switch f {
	case FrameworkStarship:
		return "Starship"
	case FrameworkOhMyZsh:
		return "Oh My Zsh"
	case FrameworkPowerlevel:
		return "Powerlevel10k"
	default:
		return ""
	}
}

// DetectPromptFramework checks for common prompt frameworks
// Returns the first detected framework or FrameworkNone
func DetectPromptFramework() DetectedFramework {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return FrameworkNone
	}

	// Check for Starship: ~/.config/starship.toml
	starshipConfig := filepath.Join(homeDir, ".config", "starship.toml")
	if _, err := os.Stat(starshipConfig); err == nil {
		return FrameworkStarship
	}

	// Check for Oh My Zsh: ~/.oh-my-zsh directory
	omzDir := filepath.Join(homeDir, ".oh-my-zsh")
	if info, err := os.Stat(omzDir); err == nil && info.IsDir() {
		return FrameworkOhMyZsh
	}

	// Check for Powerlevel10k: POWERLEVEL9K_* env vars
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "POWERLEVEL9K_") {
			return FrameworkPowerlevel
		}
	}

	return FrameworkNone
}
