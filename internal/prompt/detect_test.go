package prompt

import (
	"testing"
)

func TestDetectedFramework_String(t *testing.T) {
	tests := []struct {
		framework DetectedFramework
		expected  string
	}{
		{FrameworkNone, ""},
		{FrameworkStarship, "Starship"},
		{FrameworkOhMyZsh, "Oh My Zsh"},
		{FrameworkPowerlevel, "Powerlevel10k"},
	}

	for _, tt := range tests {
		t.Run(string(tt.framework), func(t *testing.T) {
			got := tt.framework.String()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestDetectedFramework_StringUnknownValue(t *testing.T) {
	f := DetectedFramework("unknown")
	if f.String() != "" {
		t.Errorf("expected empty string for unknown framework, got %q", f.String())
	}
}

func TestDetectPromptFramework_NoPowerlevelEnv(t *testing.T) {
	// DetectPromptFramework checks for POWERLEVEL9K_* env vars.
	// In a clean test environment without starship.toml or .oh-my-zsh,
	// and without POWERLEVEL9K_* env vars, it should return FrameworkNone
	// (unless the test runner's home has those files).
	// We just verify it doesn't panic and returns a valid value.
	result := DetectPromptFramework()
	switch result {
	case FrameworkNone, FrameworkStarship, FrameworkOhMyZsh, FrameworkPowerlevel:
		// valid
	default:
		t.Errorf("unexpected framework: %q", result)
	}
}

func TestFrameworkConstants(t *testing.T) {
	// Ensure constants have expected underlying string values
	if FrameworkNone != "" {
		t.Errorf("FrameworkNone should be empty string, got %q", FrameworkNone)
	}
	if FrameworkStarship != "starship" {
		t.Errorf("FrameworkStarship should be 'starship', got %q", FrameworkStarship)
	}
	if FrameworkOhMyZsh != "oh-my-zsh" {
		t.Errorf("FrameworkOhMyZsh should be 'oh-my-zsh', got %q", FrameworkOhMyZsh)
	}
	if FrameworkPowerlevel != "powerlevel10k" {
		t.Errorf("FrameworkPowerlevel should be 'powerlevel10k', got %q", FrameworkPowerlevel)
	}
}
