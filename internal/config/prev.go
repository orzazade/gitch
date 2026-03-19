package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// PrevIdentityPath returns the path to the previous-identity state file,
// stored alongside the main config file.
func PrevIdentityPath() (string, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(configPath), "prev-identity"), nil
}

// SavePreviousIdentity writes the given identity name to the state file.
func SavePreviousIdentity(name string) error {
	path, err := PrevIdentityPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(name+"\n"), 0644)
}

// LoadPreviousIdentity reads the previous identity name from the state file.
// Returns empty string with nil error if the file does not exist.
func LoadPreviousIdentity() (string, error) {
	path, err := PrevIdentityPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
