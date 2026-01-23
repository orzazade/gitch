package ui

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/mattn/go-isatty"
	"golang.org/x/term"
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

// ReadPassphrase reads a passphrase from stdin without echoing.
// Returns the passphrase bytes, or error if reading fails.
func ReadPassphrase(prompt string) ([]byte, error) {
	// Check if stdin is a TTY
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return nil, ErrNotInteractive
	}

	// Print prompt
	fmt.Print(prompt)

	// Read password without echo
	passphrase, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, fmt.Errorf("failed to read passphrase: %w", err)
	}

	// Print newline after hidden input
	fmt.Println()

	return passphrase, nil
}

// ReadPassphraseWithConfirm prompts for a passphrase with confirmation.
// If the user enters an empty passphrase, returns nil (no passphrase).
// Returns error if passphrases don't match.
func ReadPassphraseWithConfirm() ([]byte, error) {
	// First passphrase
	passphrase, err := ReadPassphrase("Enter passphrase (empty for no passphrase): ")
	if err != nil {
		return nil, err
	}

	// Empty passphrase - no confirmation needed
	if len(passphrase) == 0 {
		return nil, nil
	}

	// Confirm passphrase
	confirm, err := ReadPassphrase("Confirm passphrase: ")
	if err != nil {
		return nil, err
	}

	// Check match
	if !bytes.Equal(passphrase, confirm) {
		return nil, errors.New("passphrases do not match")
	}

	return passphrase, nil
}

// TypedConfirm requires the user to type an exact phrase to confirm.
// Used for destructive operations where accidental confirmation must be prevented.
// Returns true only if the user types the exact phrase (case-sensitive).
// Returns ErrNotInteractive if stdin is not a TTY.
func TypedConfirm(message, phrase string) (bool, error) {
	// Check if stdin is a TTY
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return false, ErrNotInteractive
	}

	// Print message (caller styles the message)
	fmt.Println(message)

	// Prompt for typed confirmation
	fmt.Printf("\nType '%s' to proceed: ", phrase)

	// Read response
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	// Check exact match (case-sensitive)
	return strings.TrimSpace(response) == phrase, nil
}
