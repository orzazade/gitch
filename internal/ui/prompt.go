package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

// ErrNotInteractive is returned when stdin is not a TTY and confirmation is required.
var ErrNotInteractive = errors.New("stdin is not a terminal; use --yes to skip confirmation")

// ConfirmPrompt asks for y/N confirmation.
// Returns true if user confirms, false otherwise.
// If stdin is not a TTY and skipConfirm is false, returns ErrNotInteractive.
// Default is No (N is uppercase in prompt).
func ConfirmPrompt(message string, skipConfirm bool) (bool, error) {
	// If skipping confirmation, return true immediately
	if skipConfirm {
		return true, nil
	}

	// Check if stdin is a TTY
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return false, ErrNotInteractive
	}

	// Prompt the user
	fmt.Printf("%s [y/N] ", message)

	// Read response
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	// Normalize response
	response = strings.TrimSpace(strings.ToLower(response))

	// Check for confirmation
	switch response {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
