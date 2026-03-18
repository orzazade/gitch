package gpg

import (
	"testing"
	"time"
)

func TestParseAlgorithm(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1", "rsa"},
		{"16", "elgamal"},
		{"17", "dsa"},
		{"18", "ecdh"},
		{"19", "ecdsa"},
		{"22", "ed25519"},
		{"rsa4096", "rsa"},
		{"ed25519", "ed25519"},
		{"ED25519", "ed25519"},
		{"42", "42"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseAlgorithm(tt.input)
			if got != tt.want {
				t.Errorf("parseAlgorithm(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseUID(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantEmail string
	}{
		{"standard", "Alice Smith <alice@example.com>", "Alice Smith", "alice@example.com"},
		{"no email", "Alice Smith", "Alice Smith", ""},
		{"email only", "<alice@example.com>", "", "alice@example.com"},
		{"empty", "", "", ""},
		{"percent encoded", "Alice%20Smith <alice@example.com>", "Alice Smith", "alice@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotEmail := parseUID(tt.input)
			if gotName != tt.wantName {
				t.Errorf("parseUID(%q) name = %q, want %q", tt.input, gotName, tt.wantName)
			}
			if gotEmail != tt.wantEmail {
				t.Errorf("parseUID(%q) email = %q, want %q", tt.input, gotEmail, tt.wantEmail)
			}
		})
	}
}

func TestDecodeUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no encoding", "hello", "hello"},
		{"space", "hello%20world", "hello world"},
		{"multiple", "%41%42%43", "ABC"},
		{"invalid hex", "%ZZ", "%ZZ"},
		{"trailing percent", "abc%", "abc%"},
		{"trailing percent+1", "abc%2", "abc%2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeUID(tt.input)
			if got != tt.want {
				t.Errorf("decodeUID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseUnixTimestamp(t *testing.T) {
	ts, err := parseUnixTimestamp("1700000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Unix(1700000000, 0)
	if !ts.Equal(want) {
		t.Errorf("got %v, want %v", ts, want)
	}

	_, err = parseUnixTimestamp("notanumber")
	if err == nil {
		t.Error("expected error for invalid timestamp")
	}

	_, err = parseUnixTimestamp("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestParseKeyInfo(t *testing.T) {
	// Realistic gpg --with-colons output
	output := `sec:u:255:22:ABCD1234EFGH5678:1700000000:1800000000:::::::::
fpr:::::::::AABBCCDDEE1122334455AABBCCDDEE1122334455:
uid:u::::1700000000::HASH::Alice Smith <alice@example.com>::::::::::
ssb:u:255:18:1111222233334444:1700000000::::::::::
`

	info, err := parseKeyInfo(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.ID != "ABCD1234EFGH5678" {
		t.Errorf("ID = %q, want ABCD1234EFGH5678", info.ID)
	}
	if info.Algorithm != "ed25519" {
		t.Errorf("Algorithm = %q, want ed25519", info.Algorithm)
	}
	if info.Fingerprint != "AABBCCDDEE1122334455AABBCCDDEE1122334455" {
		t.Errorf("Fingerprint = %q, want AABBCCDDEE1122334455AABBCCDDEE1122334455", info.Fingerprint)
	}
	if info.Name != "Alice Smith" {
		t.Errorf("Name = %q, want Alice Smith", info.Name)
	}
	if info.Email != "alice@example.com" {
		t.Errorf("Email = %q, want alice@example.com", info.Email)
	}
	if info.Created != time.Unix(1700000000, 0) {
		t.Errorf("Created = %v, want %v", info.Created, time.Unix(1700000000, 0))
	}
	if info.Expires == nil {
		t.Fatal("Expires should not be nil")
	}
	if *info.Expires != time.Unix(1800000000, 0) {
		t.Errorf("Expires = %v, want %v", *info.Expires, time.Unix(1800000000, 0))
	}
}

func TestParseKeyInfoNoExpiry(t *testing.T) {
	output := `sec:u:4096:1:ABCD1234EFGH5678:1700000000:::::::::::
fpr:::::::::AABBCCDDEE1122334455AABBCCDDEE1122334455:
uid:u::::1700000000::HASH::Bob <bob@example.com>::::::::::
`

	info, err := parseKeyInfo(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Algorithm != "rsa" {
		t.Errorf("Algorithm = %q, want rsa", info.Algorithm)
	}
	if info.Expires != nil {
		t.Errorf("Expires should be nil for non-expiring key, got %v", *info.Expires)
	}
}

func TestParseKeyInfoEmpty(t *testing.T) {
	_, err := parseKeyInfo("")
	if err == nil {
		t.Error("expected error for empty output")
	}
}

func TestParseMultipleKeys(t *testing.T) {
	output := `sec:u:255:22:KEY1111111111111:1700000000::::::::::
fpr:::::::::FINGERPRINT1111111111111111111111111111111:
uid:u::::1700000000::HASH::Alice <alice@example.com>::::::::::
sec:u:4096:1:KEY2222222222222:1700100000::::::::::
fpr:::::::::FINGERPRINT2222222222222222222222222222222:
uid:u::::1700100000::HASH::Bob <bob@example.com>::::::::::
`

	keys := parseMultipleKeys(output)

	if len(keys) != 2 {
		t.Fatalf("got %d keys, want 2", len(keys))
	}

	if keys[0].ID != "KEY1111111111111" {
		t.Errorf("keys[0].ID = %q, want KEY1111111111111", keys[0].ID)
	}
	if keys[0].Email != "alice@example.com" {
		t.Errorf("keys[0].Email = %q, want alice@example.com", keys[0].Email)
	}

	if keys[1].ID != "KEY2222222222222" {
		t.Errorf("keys[1].ID = %q, want KEY2222222222222", keys[1].ID)
	}
	if keys[1].Email != "bob@example.com" {
		t.Errorf("keys[1].Email = %q, want bob@example.com", keys[1].Email)
	}
}

func TestParseMultipleKeysEmpty(t *testing.T) {
	keys := parseMultipleKeys("")
	if len(keys) != 0 {
		t.Errorf("got %d keys, want 0", len(keys))
	}
}
