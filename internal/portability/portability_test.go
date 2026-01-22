package portability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

// ============================================================================
// Export Tests
// ============================================================================

func TestBuildExportConfig(t *testing.T) {
	cfg := &config.Config{
		Default: "work",
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", SSHKeyPath: "~/.ssh/work"},
			{Name: "personal", Email: "personal@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "work"},
		},
	}

	export := BuildExportConfig(cfg)

	if export.Version != CurrentExportVersion {
		t.Errorf("expected version %d, got %d", CurrentExportVersion, export.Version)
	}
	if export.Default != "work" {
		t.Errorf("expected default 'work', got %q", export.Default)
	}
	if len(export.Identities) != 2 {
		t.Errorf("expected 2 identities, got %d", len(export.Identities))
	}
	if len(export.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(export.Rules))
	}
	if export.ExportedAt.IsZero() {
		t.Error("expected ExportedAt to be set")
	}
}

func TestExportToFile(t *testing.T) {
	cfg := &config.Config{
		Default: "work",
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", SSHKeyPath: "~/.ssh/work"},
			{Name: "personal", Email: "personal@example.com", GPGKeyID: "ABCD1234"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "work"},
		},
	}

	// Create temp file
	tmpDir := t.TempDir()
	exportPath := filepath.Join(tmpDir, "gitch-export.yaml")

	// Export
	err := ExportToFile(cfg, exportPath)
	if err != nil {
		t.Fatalf("ExportToFile failed: %v", err)
	}

	// Read file and verify structure
	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}

	content := string(data)

	// Verify header comment
	if !strings.HasPrefix(content, "# gitch configuration export") {
		t.Error("expected header comment at start of file")
	}
	if !strings.Contains(content, "# Exported:") {
		t.Error("expected exported timestamp in header")
	}
	if !strings.Contains(content, "# Version: 1") {
		t.Error("expected version in header")
	}

	// Verify YAML content
	if !strings.Contains(content, "version: 1") {
		t.Error("expected 'version: 1' in YAML")
	}
	if !strings.Contains(content, "default: work") {
		t.Error("expected 'default: work' in YAML")
	}
	if !strings.Contains(content, "work@example.com") {
		t.Error("expected work email in YAML")
	}
	if !strings.Contains(content, "gpg_key_id: ABCD1234") {
		t.Error("expected GPG key ID in YAML")
	}
}

func TestExportToFile_EmptyConfig(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{},
	}

	tmpDir := t.TempDir()
	exportPath := filepath.Join(tmpDir, "gitch-export.yaml")

	err := ExportToFile(cfg, exportPath)
	if err != ErrNoIdentities {
		t.Errorf("expected ErrNoIdentities, got %v", err)
	}

	// File should not exist
	if _, err := os.Stat(exportPath); !os.IsNotExist(err) {
		t.Error("expected file not to be created")
	}
}

// ============================================================================
// Import Tests
// ============================================================================

func TestImportFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	importPath := filepath.Join(tmpDir, "gitch-export.yaml")

	// Write a valid export file
	content := `version: 1
exported_at: 2024-01-15T10:30:00Z
default: work
identities:
  - name: work
    email: work@example.com
    ssh_key_path: ~/.ssh/work
  - name: personal
    email: personal@example.com
rules:
  - type: directory
    pattern: ~/work/**
    identity: work
`
	if err := os.WriteFile(importPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Import
	export, err := ImportFromFile(importPath)
	if err != nil {
		t.Fatalf("ImportFromFile failed: %v", err)
	}

	if export.Version != 1 {
		t.Errorf("expected version 1, got %d", export.Version)
	}
	if export.Default != "work" {
		t.Errorf("expected default 'work', got %q", export.Default)
	}
	if len(export.Identities) != 2 {
		t.Errorf("expected 2 identities, got %d", len(export.Identities))
	}
	if len(export.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(export.Rules))
	}
	if export.Identities[0].Name != "work" {
		t.Errorf("expected first identity name 'work', got %q", export.Identities[0].Name)
	}
}

func TestImportFromFile_NotFound(t *testing.T) {
	_, err := ImportFromFile("/nonexistent/path/to/file.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' error, got: %v", err)
	}
}

func TestImportFromFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	importPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	content := `version: 1
identities:
  - this is not valid yaml syntax [[[
`
	if err := os.WriteFile(importPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := ImportFromFile(importPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "invalid YAML") {
		t.Errorf("expected 'invalid YAML' error, got: %v", err)
	}
}

func TestImportFromFile_NewerVersion(t *testing.T) {
	tmpDir := t.TempDir()
	importPath := filepath.Join(tmpDir, "future.yaml")

	// Write a file with a version higher than current
	content := `version: 999
exported_at: 2024-01-15T10:30:00Z
identities: []
`
	if err := os.WriteFile(importPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := ImportFromFile(importPath)
	if err == nil {
		t.Error("expected error for newer version")
	}
	if !strings.Contains(err.Error(), "newer than supported") {
		t.Errorf("expected 'newer than supported' error, got: %v", err)
	}
}

// ============================================================================
// Conflict Detection Tests
// ============================================================================

func TestDetectConflicts_NoConflicts(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "existing", Email: "existing@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/existing/**", Identity: "existing"},
		},
	}

	export := &ExportConfig{
		Identities: []config.Identity{
			{Name: "new", Email: "new@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/new/**", Identity: "new"},
		},
	}

	conflicts := DetectConflicts(cfg, export)
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(conflicts))
	}
}

func TestDetectConflicts_IdentityNameCollision(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "old@example.com"},
		},
		Rules: []rules.Rule{},
	}

	export := &ExportConfig{
		Identities: []config.Identity{
			{Name: "work", Email: "new@example.com"}, // Same name, different email
		},
		Rules: []rules.Rule{},
	}

	conflicts := DetectConflicts(cfg, export)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Type != IdentityConflict {
		t.Errorf("expected IdentityConflict, got %v", conflicts[0].Type)
	}
	if conflicts[0].Key != "work" {
		t.Errorf("expected key 'work', got %q", conflicts[0].Key)
	}
}

func TestDetectConflicts_IdentityIdentical(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", SSHKeyPath: "~/.ssh/work"},
		},
		Rules: []rules.Rule{},
	}

	export := &ExportConfig{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", SSHKeyPath: "~/.ssh/work"}, // Identical
		},
		Rules: []rules.Rule{},
	}

	conflicts := DetectConflicts(cfg, export)
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts for identical identity, got %d", len(conflicts))
	}
}

func TestDetectConflicts_RulePatternCollision(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "old-identity"},
		},
	}

	export := &ExportConfig{
		Identities: []config.Identity{},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "new-identity"}, // Same pattern, different identity
		},
	}

	conflicts := DetectConflicts(cfg, export)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Type != RuleConflict {
		t.Errorf("expected RuleConflict, got %v", conflicts[0].Type)
	}
	if conflicts[0].Key != "~/work/**" {
		t.Errorf("expected key '~/work/**', got %q", conflicts[0].Key)
	}
}

func TestDetectConflicts_CaseInsensitive(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "Work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{},
	}

	export := &ExportConfig{
		Identities: []config.Identity{
			{Name: "work", Email: "different@example.com"}, // Different case, different email
		},
		Rules: []rules.Rule{},
	}

	conflicts := DetectConflicts(cfg, export)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict (case-insensitive match), got %d", len(conflicts))
	}
	if conflicts[0].Key != "work" {
		t.Errorf("expected key 'work', got %q", conflicts[0].Key)
	}
}

// ============================================================================
// Merge Tests
// ============================================================================

func TestMergeConfig_AllNew(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{},
		Rules:      []rules.Rule{},
	}

	export := &ExportConfig{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
			{Name: "personal", Email: "personal@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "work"},
		},
	}

	result, err := MergeConfig(cfg, export, nil)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	if len(result.AddedIdentities) != 2 {
		t.Errorf("expected 2 added identities, got %d", len(result.AddedIdentities))
	}
	if len(result.AddedRules) != 1 {
		t.Errorf("expected 1 added rule, got %d", len(result.AddedRules))
	}
	if len(result.UpdatedIdentities) != 0 {
		t.Errorf("expected 0 updated identities, got %d", len(result.UpdatedIdentities))
	}
	if len(result.Skipped) != 0 {
		t.Errorf("expected 0 skipped, got %d", len(result.Skipped))
	}

	// Verify config was updated
	if len(cfg.Identities) != 2 {
		t.Errorf("expected 2 identities in config, got %d", len(cfg.Identities))
	}
	if len(cfg.Rules) != 1 {
		t.Errorf("expected 1 rule in config, got %d", len(cfg.Rules))
	}
}

func TestMergeConfig_OverwriteIdentity(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "old@example.com"},
		},
		Rules: []rules.Rule{},
	}

	export := &ExportConfig{
		Identities: []config.Identity{
			{Name: "work", Email: "new@example.com"},
		},
		Rules: []rules.Rule{},
	}

	overwrite := map[string]bool{"work": true}
	result, err := MergeConfig(cfg, export, overwrite)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	if len(result.UpdatedIdentities) != 1 {
		t.Errorf("expected 1 updated identity, got %d", len(result.UpdatedIdentities))
	}
	if len(result.Skipped) != 0 {
		t.Errorf("expected 0 skipped, got %d", len(result.Skipped))
	}

	// Verify identity was updated
	identity, err := cfg.GetIdentity("work")
	if err != nil {
		t.Fatalf("failed to get identity: %v", err)
	}
	if identity.Email != "new@example.com" {
		t.Errorf("expected email 'new@example.com', got %q", identity.Email)
	}
}

func TestMergeConfig_SkipIdentity(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "old@example.com"},
		},
		Rules: []rules.Rule{},
	}

	export := &ExportConfig{
		Identities: []config.Identity{
			{Name: "work", Email: "new@example.com"},
		},
		Rules: []rules.Rule{},
	}

	overwrite := map[string]bool{"work": false}
	result, err := MergeConfig(cfg, export, overwrite)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	if len(result.UpdatedIdentities) != 0 {
		t.Errorf("expected 0 updated identities, got %d", len(result.UpdatedIdentities))
	}
	if len(result.Skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d", len(result.Skipped))
	}

	// Verify identity was NOT updated
	identity, err := cfg.GetIdentity("work")
	if err != nil {
		t.Fatalf("failed to get identity: %v", err)
	}
	if identity.Email != "old@example.com" {
		t.Errorf("expected email 'old@example.com' (unchanged), got %q", identity.Email)
	}
}

func TestMergeConfig_OverwriteRule(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "old-identity", Email: "old@example.com"},
			{Name: "new-identity", Email: "new@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "old-identity"},
		},
	}

	export := &ExportConfig{
		Identities: []config.Identity{},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "new-identity"},
		},
	}

	overwrite := map[string]bool{"~/work/**": true}
	result, err := MergeConfig(cfg, export, overwrite)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	if len(result.UpdatedRules) != 1 {
		t.Errorf("expected 1 updated rule, got %d", len(result.UpdatedRules))
	}

	// Verify rule was updated
	if cfg.Rules[0].Identity != "new-identity" {
		t.Errorf("expected rule identity 'new-identity', got %q", cfg.Rules[0].Identity)
	}
}

func TestMergeConfig_MixedDecisions(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work-old@example.com"},
			{Name: "personal", Email: "personal-old@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "work"},
		},
	}

	export := &ExportConfig{
		Identities: []config.Identity{
			{Name: "work", Email: "work-new@example.com"},        // Conflict - will overwrite
			{Name: "personal", Email: "personal-new@example.com"}, // Conflict - will skip
			{Name: "opensource", Email: "oss@example.com"},        // New - will add
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "work-new"},   // Conflict - will skip
			{Type: rules.DirectoryRule, Pattern: "~/projects/**", Identity: "personal"}, // New - will add
		},
	}

	overwrite := map[string]bool{
		"work":       true,  // Overwrite this identity
		"personal":   false, // Skip this identity
		"~/work/**":  false, // Skip this rule
	}

	result, err := MergeConfig(cfg, export, overwrite)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	// Check results
	if len(result.AddedIdentities) != 1 || result.AddedIdentities[0] != "opensource" {
		t.Errorf("expected 'opensource' added, got %v", result.AddedIdentities)
	}
	if len(result.UpdatedIdentities) != 1 || result.UpdatedIdentities[0] != "work" {
		t.Errorf("expected 'work' updated, got %v", result.UpdatedIdentities)
	}
	if len(result.AddedRules) != 1 || result.AddedRules[0] != "~/projects/**" {
		t.Errorf("expected '~/projects/**' added, got %v", result.AddedRules)
	}
	if len(result.Skipped) != 2 {
		t.Errorf("expected 2 skipped, got %d: %v", len(result.Skipped), result.Skipped)
	}

	// Verify final state
	if len(cfg.Identities) != 3 {
		t.Errorf("expected 3 identities, got %d", len(cfg.Identities))
	}
	if len(cfg.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(cfg.Rules))
	}

	// Verify work was updated
	work, _ := cfg.GetIdentity("work")
	if work.Email != "work-new@example.com" {
		t.Errorf("expected work email updated, got %q", work.Email)
	}

	// Verify personal was NOT updated
	personal, _ := cfg.GetIdentity("personal")
	if personal.Email != "personal-old@example.com" {
		t.Errorf("expected personal email unchanged, got %q", personal.Email)
	}
}

// ============================================================================
// Round-trip Tests
// ============================================================================

func TestExportImportRoundTrip(t *testing.T) {
	// Create original config
	original := &config.Config{
		Default: "work",
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", SSHKeyPath: "~/.ssh/work", GPGKeyID: "ABC123"},
			{Name: "personal", Email: "personal@example.com", HookMode: "block"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "work"},
			{Type: rules.RemoteRule, Pattern: "github.com/company/*", Identity: "work"},
		},
	}

	// Export
	tmpDir := t.TempDir()
	exportPath := filepath.Join(tmpDir, "export.yaml")
	if err := ExportToFile(original, exportPath); err != nil {
		t.Fatalf("ExportToFile failed: %v", err)
	}

	// Import
	imported, err := ImportFromFile(exportPath)
	if err != nil {
		t.Fatalf("ImportFromFile failed: %v", err)
	}

	// Verify round-trip
	if imported.Default != original.Default {
		t.Errorf("default mismatch: expected %q, got %q", original.Default, imported.Default)
	}
	if len(imported.Identities) != len(original.Identities) {
		t.Errorf("identity count mismatch: expected %d, got %d", len(original.Identities), len(imported.Identities))
	}
	if len(imported.Rules) != len(original.Rules) {
		t.Errorf("rule count mismatch: expected %d, got %d", len(original.Rules), len(imported.Rules))
	}

	// Verify identity details
	for i, orig := range original.Identities {
		imp := imported.Identities[i]
		if orig.Name != imp.Name || orig.Email != imp.Email ||
			orig.SSHKeyPath != imp.SSHKeyPath || orig.GPGKeyID != imp.GPGKeyID ||
			orig.HookMode != imp.HookMode {
			t.Errorf("identity %d mismatch: original=%+v, imported=%+v", i, orig, imp)
		}
	}

	// Verify rule details
	for i, orig := range original.Rules {
		imp := imported.Rules[i]
		if orig.Type != imp.Type || orig.Pattern != imp.Pattern || orig.Identity != imp.Identity {
			t.Errorf("rule %d mismatch: original=%+v, imported=%+v", i, orig, imp)
		}
	}
}

// ============================================================================
// Helper function tests
// ============================================================================

func TestIdentitiesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a, b     *config.Identity
		expected bool
	}{
		{
			name:     "identical",
			a:        &config.Identity{Name: "work", Email: "work@example.com", SSHKeyPath: "~/.ssh/work"},
			b:        &config.Identity{Name: "work", Email: "work@example.com", SSHKeyPath: "~/.ssh/work"},
			expected: true,
		},
		{
			name:     "different email",
			a:        &config.Identity{Name: "work", Email: "old@example.com"},
			b:        &config.Identity{Name: "work", Email: "new@example.com"},
			expected: false,
		},
		{
			name:     "different ssh key",
			a:        &config.Identity{Name: "work", Email: "work@example.com", SSHKeyPath: "~/.ssh/old"},
			b:        &config.Identity{Name: "work", Email: "work@example.com", SSHKeyPath: "~/.ssh/new"},
			expected: false,
		},
		{
			name:     "different gpg key",
			a:        &config.Identity{Name: "work", Email: "work@example.com", GPGKeyID: "ABC"},
			b:        &config.Identity{Name: "work", Email: "work@example.com", GPGKeyID: "XYZ"},
			expected: false,
		},
		{
			name:     "email case insensitive",
			a:        &config.Identity{Name: "work", Email: "Work@Example.com"},
			b:        &config.Identity{Name: "work", Email: "work@example.com"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := identitiesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("identitiesEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestRulesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a, b     *rules.Rule
		expected bool
	}{
		{
			name:     "identical",
			a:        &rules.Rule{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "work"},
			b:        &rules.Rule{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "work"},
			expected: true,
		},
		{
			name:     "different identity",
			a:        &rules.Rule{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "work"},
			b:        &rules.Rule{Type: rules.DirectoryRule, Pattern: "~/work/**", Identity: "personal"},
			expected: false,
		},
		{
			name:     "different type",
			a:        &rules.Rule{Type: rules.DirectoryRule, Pattern: "pattern", Identity: "work"},
			b:        &rules.Rule{Type: rules.RemoteRule, Pattern: "pattern", Identity: "work"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rulesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("rulesEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Ensure ExportedAt is populated with a reasonable time
func TestExportedAtTimestamp(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "test", Email: "test@example.com"},
		},
	}

	before := time.Now().UTC().Add(-time.Second)
	export := BuildExportConfig(cfg)
	after := time.Now().UTC().Add(time.Second)

	if export.ExportedAt.Before(before) || export.ExportedAt.After(after) {
		t.Errorf("ExportedAt %v not within expected range [%v, %v]", export.ExportedAt, before, after)
	}
}
