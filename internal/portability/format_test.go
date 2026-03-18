package portability

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
)

func Test_toEncryptedIdentity_AllFields(t *testing.T) {
	id := config.Identity{
		Name:       "work",
		GitName:    "Jane Doe",
		Email:      "jane@work.com",
		SSHKeyPath: "~/.ssh/work_key",
		GPGKeyID:   "ABCD1234",
		HookMode:   "block",
	}

	enc := toEncryptedIdentity(id)

	if enc.Name != id.Name {
		t.Errorf("Name: got %q, want %q", enc.Name, id.Name)
	}
	if enc.GitName != id.GitName {
		t.Errorf("GitName: got %q, want %q", enc.GitName, id.GitName)
	}
	if enc.Email != id.Email {
		t.Errorf("Email: got %q, want %q", enc.Email, id.Email)
	}
	if enc.SSHKeyPath != id.SSHKeyPath {
		t.Errorf("SSHKeyPath: got %q, want %q", enc.SSHKeyPath, id.SSHKeyPath)
	}
	if enc.GPGKeyID != id.GPGKeyID {
		t.Errorf("GPGKeyID: got %q, want %q", enc.GPGKeyID, id.GPGKeyID)
	}
	if enc.HookMode != id.HookMode {
		t.Errorf("HookMode: got %q, want %q", enc.HookMode, id.HookMode)
	}
	if enc.SSHKeyEncrypted != "" {
		t.Error("SSHKeyEncrypted should be empty for fresh conversion")
	}
}

func TestToIdentity_RoundTrip(t *testing.T) {
	original := config.Identity{
		Name:       "personal",
		GitName:    "John",
		Email:      "john@personal.com",
		SSHKeyPath: "~/.ssh/personal",
		GPGKeyID:   "EFGH5678",
		HookMode:   "warn",
	}

	enc := toEncryptedIdentity(original)
	back := enc.ToIdentity()

	if back.Name != original.Name {
		t.Errorf("Name: got %q, want %q", back.Name, original.Name)
	}
	if back.GitName != original.GitName {
		t.Errorf("GitName: got %q, want %q", back.GitName, original.GitName)
	}
	if back.Email != original.Email {
		t.Errorf("Email: got %q, want %q", back.Email, original.Email)
	}
	if back.SSHKeyPath != original.SSHKeyPath {
		t.Errorf("SSHKeyPath: got %q, want %q", back.SSHKeyPath, original.SSHKeyPath)
	}
	if back.GPGKeyID != original.GPGKeyID {
		t.Errorf("GPGKeyID: got %q, want %q", back.GPGKeyID, original.GPGKeyID)
	}
	if back.HookMode != original.HookMode {
		t.Errorf("HookMode: got %q, want %q", back.HookMode, original.HookMode)
	}
}

func Test_toEncryptedIdentity_MinimalFields(t *testing.T) {
	id := config.Identity{
		Name:  "minimal",
		Email: "min@test.com",
	}

	enc := toEncryptedIdentity(id)
	if enc.Name != "minimal" || enc.Email != "min@test.com" {
		t.Error("basic fields should be preserved")
	}
	if enc.GitName != "" || enc.SSHKeyPath != "" || enc.GPGKeyID != "" || enc.HookMode != "" {
		t.Error("unset fields should remain empty")
	}
}

func TestHasEncryptedKeys_NoEncryption(t *testing.T) {
	export := &ExportConfig{}
	if HasEncryptedKeys(export) {
		t.Error("expected false when Encryption is nil")
	}
}

func TestHasEncryptedKeys_WithEncryptedKey(t *testing.T) {
	export := &ExportConfig{
		Encryption: &encryptionInfo{Method: "age-scrypt"},
		EncryptedIdentities: []encryptedIdentity{
			{Name: "work", SSHKeyEncrypted: "encrypted-data"},
		},
	}
	if !HasEncryptedKeys(export) {
		t.Error("expected true when encrypted key data exists")
	}
}

func TestHasEncryptedKeys_EncryptionButNoKeys(t *testing.T) {
	export := &ExportConfig{
		Encryption: &encryptionInfo{Method: "age-scrypt"},
		EncryptedIdentities: []encryptedIdentity{
			{Name: "work", SSHKeyEncrypted: ""},
		},
	}
	if HasEncryptedKeys(export) {
		t.Error("expected false when no encrypted key data")
	}
}

func TestGetEncryptedKeyPaths_Empty(t *testing.T) {
	export := &ExportConfig{}
	paths := GetEncryptedKeyPaths(export)
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}

func TestGetEncryptedKeyPaths_WithPaths(t *testing.T) {
	export := &ExportConfig{
		EncryptedIdentities: []encryptedIdentity{
			{Name: "work", SSHKeyPath: "/tmp/test_key", SSHKeyEncrypted: "data"},
			{Name: "personal", SSHKeyPath: "", SSHKeyEncrypted: "data"},
			{Name: "oss", SSHKeyPath: "/tmp/oss_key", SSHKeyEncrypted: ""},
		},
	}
	paths := GetEncryptedKeyPaths(export)
	// Only the first entry has both SSHKeyPath and SSHKeyEncrypted
	if len(paths) != 1 {
		t.Errorf("expected 1 path, got %d: %v", len(paths), paths)
	}
}
