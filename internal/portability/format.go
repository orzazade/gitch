// Package portability provides export/import functionality for gitch configuration.
package portability

import (
	"time"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

// CurrentExportVersion is the current version of the export format.
// Increment this when making breaking changes to the export format.
const CurrentExportVersion = 1

// ExportConfig is the root structure for exported configuration.
// It contains all identities and rules that can be backed up and restored.
type ExportConfig struct {
	Version    int               `yaml:"version"`
	ExportedAt time.Time         `yaml:"exported_at"`
	Default    string            `yaml:"default,omitempty"`
	Identities []config.Identity `yaml:"identities"`
	Rules      []rules.Rule      `yaml:"rules,omitempty"`
}
