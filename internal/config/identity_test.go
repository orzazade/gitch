package config

import (
	"strings"
	"testing"
)

func TestValidateName_Valid(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"work"},
		{"personal"},
		{"my-work"},
		{"work-2024"},
		{"a"},
		{"A"},
		{"1"},
		{"a1"},
		{"A1B2"},
		{"my-long-identity-name"},
		{"work-email-identity"},
		{"my--work"}, // double hyphen is valid (alphanumeric + hyphens, no leading/trailing)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateName(tt.name); err != nil {
				t.Errorf("ValidateName(%q) returned error: %v, expected nil", tt.name, err)
			}
		})
	}
}

func TestValidateName_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		wantContain string
	}{
		{"", "cannot be empty"},
		{"-invalid", "cannot start or end with a hyphen"},
		{"invalid-", "cannot start or end with a hyphen"},
		{"-", "alphanumeric"}, // single hyphen is not alphanumeric
		{"work@home", "alphanumeric"},
		{"work home", "alphanumeric"},
		{"work.personal", "alphanumeric"},
		{strings.Repeat("a", 51), "cannot exceed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.name)
			if err == nil {
				t.Errorf("ValidateName(%q) returned nil, expected error containing %q", tt.name, tt.wantContain)
				return
			}
			if !strings.Contains(err.Error(), tt.wantContain) {
				t.Errorf("ValidateName(%q) error = %v, want error containing %q", tt.name, err, tt.wantContain)
			}
		})
	}
}

func TestValidateEmail_Valid(t *testing.T) {
	tests := []struct {
		email string
	}{
		{"user@example.com"},
		{"user.name@example.com"},
		{"user+tag@example.com"},
		{"user@subdomain.example.com"},
		{"first.last@example.co.uk"},
		{"a@b.com"},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			if err := ValidateEmail(tt.email); err != nil {
				t.Errorf("ValidateEmail(%q) returned error: %v, expected nil", tt.email, err)
			}
		})
	}
}

func TestValidateEmail_Invalid(t *testing.T) {
	tests := []struct {
		email       string
		wantContain string
	}{
		{"", "cannot be empty"},
		{"invalid", "invalid email"},
		{"@example.com", "invalid email"},
		{"user@", "invalid email"},
		{"user@.com", "invalid email"},
		{"user name@example.com", "invalid email"},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if err == nil {
				t.Errorf("ValidateEmail(%q) returned nil, expected error containing %q", tt.email, tt.wantContain)
				return
			}
			if !strings.Contains(err.Error(), tt.wantContain) {
				t.Errorf("ValidateEmail(%q) error = %v, want error containing %q", tt.email, err, tt.wantContain)
			}
		})
	}
}

func TestIdentity_Validate(t *testing.T) {
	tests := []struct {
		name      string
		identity  Identity
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "valid identity",
			identity: Identity{Name: "work", Email: "user@example.com"},
			wantErr:  false,
		},
		{
			name:      "invalid name",
			identity:  Identity{Name: "-invalid", Email: "user@example.com"},
			wantErr:   true,
			errSubstr: "hyphen",
		},
		{
			name:      "invalid email",
			identity:  Identity{Name: "work", Email: "invalid"},
			wantErr:   true,
			errSubstr: "invalid email",
		},
		{
			name:      "empty name",
			identity:  Identity{Name: "", Email: "user@example.com"},
			wantErr:   true,
			errSubstr: "empty",
		},
		{
			name:      "empty email",
			identity:  Identity{Name: "work", Email: ""},
			wantErr:   true,
			errSubstr: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.identity.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Identity.Validate() returned nil, expected error containing %q", tt.errSubstr)
					return
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("Identity.Validate() error = %v, want error containing %q", err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("Identity.Validate() returned error: %v, expected nil", err)
				}
			}
		})
	}
}
