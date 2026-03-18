package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

func TestParseRemoteForSuggest(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantHost string
		wantOrg  string
		wantRepo string
		wantErr  bool
	}{
		{
			name:     "HTTPS URL",
			url:      "https://github.com/acme/project.git",
			wantHost: "github.com",
			wantOrg:  "acme",
			wantRepo: "project",
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://github.com/acme/project",
			wantHost: "github.com",
			wantOrg:  "acme",
			wantRepo: "project",
		},
		{
			name:     "SSH URL",
			url:      "git@github.com:acme/project.git",
			wantHost: "github.com",
			wantOrg:  "acme",
			wantRepo: "project",
		},
		{
			name:     "SSH URL without .git",
			url:      "git@github.com:acme/project",
			wantHost: "github.com",
			wantOrg:  "acme",
			wantRepo: "project",
		},
		{
			name:     "GitLab SSH",
			url:      "git@gitlab.com:team/service.git",
			wantHost: "gitlab.com",
			wantOrg:  "team",
			wantRepo: "service",
		},
		{
			name:     "Azure DevOps HTTPS",
			url:      "https://dev.azure.com/org/project",
			wantHost: "dev.azure.com",
			wantOrg:  "org",
			wantRepo: "project",
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRemoteForSuggest(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Host != tt.wantHost {
				t.Errorf("host = %q, want %q", result.Host, tt.wantHost)
			}
			if result.Org != tt.wantOrg {
				t.Errorf("org = %q, want %q", result.Org, tt.wantOrg)
			}
			if result.Repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", result.Repo, tt.wantRepo)
			}
		})
	}
}

func TestSuggestFromExistingRules(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "me@company.com"},
			{Name: "personal", Email: "me@gmail.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.RemoteRule, Pattern: "github.com/company/*", Identity: "work"},
			{Type: rules.RemoteRule, Pattern: "github.com/personal-org/*", Identity: "personal"},
		},
	}

	tests := []struct {
		name         string
		remote       *suggestParsedRemote
		wantIdentity string
		wantNil      bool
	}{
		{
			name:         "same org as existing rule",
			remote:       &suggestParsedRemote{Host: "github.com", Org: "company", Repo: "new-project"},
			wantIdentity: "work",
		},
		{
			name:         "same org personal",
			remote:       &suggestParsedRemote{Host: "github.com", Org: "personal-org", Repo: "side-project"},
			wantIdentity: "personal",
		},
		{
			name:         "same host different org",
			remote:       &suggestParsedRemote{Host: "github.com", Org: "unknown-org", Repo: "some-repo"},
			wantIdentity: "work", // first match on host
		},
		{
			name:    "different host no match",
			remote:  &suggestParsedRemote{Host: "gitlab.com", Org: "team", Repo: "project"},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := suggestFromExistingRules(cfg, tt.remote)
			if tt.wantNil {
				if result != nil {
					t.Fatalf("expected nil, got suggestion for %q", result.Identity)
				}
				return
			}
			if result == nil {
				t.Fatal("expected suggestion, got nil")
			}
			if result.Identity != tt.wantIdentity {
				t.Errorf("identity = %q, want %q", result.Identity, tt.wantIdentity)
			}
			if result.RuleType != rules.RemoteRule {
				t.Errorf("rule type = %q, want %q", result.RuleType, rules.RemoteRule)
			}
		})
	}
}

func TestSuggestFromCurrentEmail(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "me@company.com"},
			{Name: "personal", Email: "me@gmail.com"},
		},
	}

	tests := []struct {
		name         string
		email        string
		remote       *suggestParsedRemote
		wantIdentity string
		wantNil      bool
	}{
		{
			name:         "email matches work identity",
			email:        "me@company.com",
			remote:       &suggestParsedRemote{Host: "github.com", Org: "neworg", Repo: "project"},
			wantIdentity: "work",
		},
		{
			name:         "email matches personal identity",
			email:        "me@gmail.com",
			remote:       &suggestParsedRemote{Host: "github.com", Org: "myorg", Repo: "stuff"},
			wantIdentity: "personal",
		},
		{
			name:         "email matches case-insensitive",
			email:        "ME@Company.com",
			remote:       &suggestParsedRemote{Host: "github.com", Org: "team", Repo: "repo"},
			wantIdentity: "work",
		},
		{
			name:    "email does not match any identity",
			email:   "stranger@unknown.com",
			remote:  &suggestParsedRemote{Host: "github.com", Org: "org", Repo: "repo"},
			wantNil: true,
		},
		{
			name:    "empty email",
			email:   "",
			remote:  &suggestParsedRemote{Host: "github.com", Org: "org", Repo: "repo"},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := suggestFromCurrentEmail(cfg, tt.remote, tt.email)
			if tt.wantNil {
				if result != nil {
					t.Fatalf("expected nil, got suggestion for %q", result.Identity)
				}
				return
			}
			if result == nil {
				t.Fatal("expected suggestion, got nil")
			}
			if result.Identity != tt.wantIdentity {
				t.Errorf("identity = %q, want %q", result.Identity, tt.wantIdentity)
			}
		})
	}
}

func TestSuggestRule(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "me@company.com"},
			{Name: "personal", Email: "me@gmail.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.RemoteRule, Pattern: "github.com/company/*", Identity: "work"},
		},
	}

	tests := []struct {
		name         string
		remoteURL    string
		currentEmail string
		wantIdentity string
		wantNil      bool
	}{
		{
			name:         "matches existing rule org",
			remoteURL:    "git@github.com:company/new-service.git",
			currentEmail: "",
			wantIdentity: "work",
		},
		{
			name:         "no rule match but email matches",
			remoteURL:    "https://gitlab.com/team/project.git",
			currentEmail: "me@gmail.com",
			wantIdentity: "personal",
		},
		{
			name:      "no remote URL",
			remoteURL: "",
			wantNil:   true,
		},
		{
			name:         "no match at all",
			remoteURL:    "https://gitlab.com/unknown/project.git",
			currentEmail: "stranger@nowhere.com",
			wantNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := suggestRule(cfg, "/some/path", tt.remoteURL, tt.currentEmail)
			if tt.wantNil {
				if result != nil {
					t.Fatalf("expected nil, got suggestion for %q", result.Identity)
				}
				return
			}
			if result == nil {
				t.Fatal("expected suggestion, got nil")
			}
			if result.Identity != tt.wantIdentity {
				t.Errorf("identity = %q, want %q", result.Identity, tt.wantIdentity)
			}
		})
	}
}
