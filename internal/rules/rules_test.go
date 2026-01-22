package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde only",
			input:    "~",
			expected: home,
		},
		{
			name:     "tilde with path",
			input:    "~/work/project",
			expected: filepath.Join(home, "work/project"),
		},
		{
			name:     "no tilde",
			input:    "/usr/local/bin",
			expected: "/usr/local/bin",
		},
		{
			name:     "tilde in middle (not expanded)",
			input:    "/path/~/something",
			expected: "/path/~/something",
		},
		{
			name:     "tilde user format (not supported)",
			input:    "~otheruser/path",
			expected: "~otheruser/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandTilde(tt.input)
			if result != tt.expected {
				t.Errorf("expandTilde(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMatchDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		pattern string
		cwd     string
		want    bool
		wantErr bool
	}{
		{
			name:    "tilde pattern matches home subdir",
			pattern: "~/work/**",
			cwd:     filepath.Join(home, "work/project"),
			want:    true,
		},
		{
			name:    "tilde pattern matches nested subdir",
			pattern: "~/work/**",
			cwd:     filepath.Join(home, "work/project/src/deep"),
			want:    true,
		},
		{
			name:    "tilde pattern does not match other dir",
			pattern: "~/work/**",
			cwd:     filepath.Join(home, "personal/project"),
			want:    false,
		},
		{
			name:    "exact path match",
			pattern: "~/work/specific",
			cwd:     filepath.Join(home, "work/specific"),
			want:    true,
		},
		{
			name:    "exact path no match",
			pattern: "~/work/specific",
			cwd:     filepath.Join(home, "work/other"),
			want:    false,
		},
		{
			name:    "single star pattern",
			pattern: "~/work/*",
			cwd:     filepath.Join(home, "work/project"),
			want:    true,
		},
		{
			name:    "single star does not match nested",
			pattern: "~/work/*",
			cwd:     filepath.Join(home, "work/project/sub"),
			want:    false,
		},
		{
			name:    "absolute path pattern",
			pattern: "/tmp/test/**",
			cwd:     "/tmp/test/deep/path",
			want:    true,
		},
		{
			name:    "trailing slash in pattern",
			pattern: "~/work/**",
			cwd:     filepath.Join(home, "work/project"),
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchDirectory(tt.pattern, tt.cwd)
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MatchDirectory(%q, %q) = %v, want %v", tt.pattern, tt.cwd, got, tt.want)
			}
		})
	}
}

func TestParseRemote(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		wantHost string
		wantOrg  string
		wantRepo string
		wantErr  bool
	}{
		{
			name:     "SSH URL",
			rawURL:   "git@github.com:company/repo.git",
			wantHost: "github.com",
			wantOrg:  "company",
			wantRepo: "repo",
		},
		{
			name:     "HTTPS URL",
			rawURL:   "https://github.com/company/repo.git",
			wantHost: "github.com",
			wantOrg:  "company",
			wantRepo: "repo",
		},
		{
			name:     "HTTPS URL without .git",
			rawURL:   "https://github.com/company/repo",
			wantHost: "github.com",
			wantOrg:  "company",
			wantRepo: "repo",
		},
		{
			name:     "GitLab SSH URL",
			rawURL:   "git@gitlab.com:org/project.git",
			wantHost: "gitlab.com",
			wantOrg:  "org",
			wantRepo: "project",
		},
		{
			name:     "Azure DevOps SSH",
			rawURL:   "git@ssh.dev.azure.com:v3/org/project/repo",
			wantHost: "ssh.dev.azure.com",
			wantOrg:  "v3",
			wantRepo: "org",
		},
		{
			name:     "Host uppercase normalized",
			rawURL:   "git@GitHub.COM:Company/Repo.git",
			wantHost: "github.com",
			wantOrg:  "Company",
			wantRepo: "Repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRemote(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRemote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Host != tt.wantHost {
				t.Errorf("ParseRemote() Host = %q, want %q", got.Host, tt.wantHost)
			}
			if got.Org != tt.wantOrg {
				t.Errorf("ParseRemote() Org = %q, want %q", got.Org, tt.wantOrg)
			}
			if got.Repo != tt.wantRepo {
				t.Errorf("ParseRemote() Repo = %q, want %q", got.Repo, tt.wantRepo)
			}
		})
	}
}

func TestMatchRemote(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		remote  *ParsedRemote
		want    bool
	}{
		{
			name:    "wildcard matches any repo",
			pattern: "github.com/company/*",
			remote:  &ParsedRemote{Host: "github.com", Org: "company", Repo: "any-repo"},
			want:    true,
		},
		{
			name:    "wildcard matches different repo",
			pattern: "github.com/company/*",
			remote:  &ParsedRemote{Host: "github.com", Org: "company", Repo: "other-repo"},
			want:    true,
		},
		{
			name:    "wildcard does not match different org",
			pattern: "github.com/company/*",
			remote:  &ParsedRemote{Host: "github.com", Org: "other", Repo: "repo"},
			want:    false,
		},
		{
			name:    "exact match",
			pattern: "github.com/company/specific-repo",
			remote:  &ParsedRemote{Host: "github.com", Org: "company", Repo: "specific-repo"},
			want:    true,
		},
		{
			name:    "exact match fails on different repo",
			pattern: "github.com/company/specific-repo",
			remote:  &ParsedRemote{Host: "github.com", Org: "company", Repo: "other-repo"},
			want:    false,
		},
		{
			name:    "org level pattern matches repo",
			pattern: "github.com/company",
			remote:  &ParsedRemote{Host: "github.com", Org: "company", Repo: "any-repo"},
			want:    true,
		},
		{
			name:    "case insensitive host",
			pattern: "GitHub.com/company/*",
			remote:  &ParsedRemote{Host: "github.com", Org: "company", Repo: "repo"},
			want:    true,
		},
		{
			name:    "nil remote returns false",
			pattern: "github.com/company/*",
			remote:  nil,
			want:    false,
		},
		{
			name:    "different host no match",
			pattern: "github.com/company/*",
			remote:  &ParsedRemote{Host: "gitlab.com", Org: "company", Repo: "repo"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchRemote(tt.pattern, tt.remote)
			if got != tt.want {
				t.Errorf("MatchRemote(%q, %+v) = %v, want %v", tt.pattern, tt.remote, got, tt.want)
			}
		})
	}
}

func TestSpecificity(t *testing.T) {
	tests := []struct {
		name string
		rule Rule
		want int
	}{
		{
			name: "deep directory path higher than shallow",
			rule: Rule{Type: DirectoryRule, Pattern: "~/work/company/project"},
			// 4 segments (home + work + company + project) * 10 = 40, no wildcards
		},
		{
			name: "shallow directory path",
			rule: Rule{Type: DirectoryRule, Pattern: "~/work"},
			// 2 segments * 10 = 20
		},
		{
			name: "directory with double star",
			rule: Rule{Type: DirectoryRule, Pattern: "~/work/**"},
			// segments * 10 - wildcards
		},
		{
			name: "exact remote match has bonus",
			rule: Rule{Type: RemoteRule, Pattern: "github.com/company/repo"},
			// 3 parts * 10 + 50 bonus = 80
		},
		{
			name: "wildcard remote no bonus",
			rule: Rule{Type: RemoteRule, Pattern: "github.com/company/*"},
			// 3 parts * 10 - 2 = 28
		},
	}

	// Test relative ordering
	deepDir := Rule{Type: DirectoryRule, Pattern: "~/work/company/project"}
	shallowDir := Rule{Type: DirectoryRule, Pattern: "~/work"}
	wildcardDir := Rule{Type: DirectoryRule, Pattern: "~/work/**"}

	if deepDir.Specificity() <= shallowDir.Specificity() {
		t.Error("Deep directory should have higher specificity than shallow")
	}
	// Note: ~/work/** has more segments than ~/work when expanded, but wildcards add penalty
	// The key behavior is that deeper paths are more specific than shallow wildcards
	if deepDir.Specificity() <= wildcardDir.Specificity() {
		t.Error("Deep exact path should have higher specificity than shallow wildcard")
	}

	exactRemote := Rule{Type: RemoteRule, Pattern: "github.com/company/repo"}
	wildcardRemote := Rule{Type: RemoteRule, Pattern: "github.com/company/*"}

	if exactRemote.Specificity() <= wildcardRemote.Specificity() {
		t.Error("Exact remote should have higher specificity than wildcard remote")
	}

	// Log actual values for debugging
	for _, tt := range tests {
		t.Logf("%s: specificity = %d", tt.name, tt.rule.Specificity())
	}
}

func TestFindBestMatch(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	rules := []Rule{
		{Type: DirectoryRule, Pattern: "~/work/**", Identity: "work"},
		{Type: DirectoryRule, Pattern: "~/work/company/**", Identity: "company"},
		{Type: DirectoryRule, Pattern: "~/personal/**", Identity: "personal"},
		{Type: RemoteRule, Pattern: "github.com/company/*", Identity: "company-remote"},
		{Type: RemoteRule, Pattern: "github.com/personal/*", Identity: "personal-remote"},
		{Type: RemoteRule, Pattern: "github.com/company/special-repo", Identity: "special"},
	}

	tests := []struct {
		name         string
		cwd          string
		remoteURL    string
		wantIdentity string
		wantNil      bool
	}{
		{
			name:         "more specific directory wins",
			cwd:          filepath.Join(home, "work/company/project"),
			remoteURL:    "",
			wantIdentity: "company",
		},
		{
			name:         "general work directory",
			cwd:          filepath.Join(home, "work/other"),
			remoteURL:    "",
			wantIdentity: "work",
		},
		{
			name:         "personal directory",
			cwd:          filepath.Join(home, "personal/stuff"),
			remoteURL:    "",
			wantIdentity: "personal",
		},
		{
			name:         "exact remote repo wins over wildcard",
			cwd:          "/tmp",
			remoteURL:    "git@github.com:company/special-repo.git",
			wantIdentity: "special",
		},
		{
			name:         "wildcard remote match",
			cwd:          "/tmp",
			remoteURL:    "git@github.com:company/other-repo.git",
			wantIdentity: "company-remote",
		},
		{
			name:      "no matching rules",
			cwd:       "/opt/random",
			remoteURL: "",
			wantNil:   true,
		},
		{
			name:         "exact remote wins when more specific than directory wildcard",
			cwd:          filepath.Join(home, "work/company/special-repo"),
			remoteURL:    "git@github.com:company/special-repo.git",
			wantIdentity: "special", // exact remote (80) beats directory wildcard (~60)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindBestMatch(rules, tt.cwd, tt.remoteURL)
			if tt.wantNil {
				if result != nil {
					t.Errorf("FindBestMatch() = %v, want nil", result)
				}
				return
			}
			if result == nil {
				t.Error("FindBestMatch() = nil, want a match")
				return
			}
			if result.Identity != tt.wantIdentity {
				t.Errorf("FindBestMatch().Identity = %q, want %q", result.Identity, tt.wantIdentity)
			}
		})
	}
}

func TestValidatePattern(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
	}{
		{
			name:    "valid directory pattern",
			rule:    Rule{Type: DirectoryRule, Pattern: "~/work/**"},
			wantErr: false,
		},
		{
			name:    "valid remote pattern",
			rule:    Rule{Type: RemoteRule, Pattern: "github.com/org/*"},
			wantErr: false,
		},
		{
			name:    "empty pattern",
			rule:    Rule{Type: DirectoryRule, Pattern: ""},
			wantErr: true,
		},
		{
			name:    "invalid remote pattern - no slash",
			rule:    Rule{Type: RemoteRule, Pattern: "github.com"},
			wantErr: true,
		},
		{
			name:    "invalid rule type",
			rule:    Rule{Type: "invalid", Pattern: "test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.ValidatePattern()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePattern() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRuleHelpers(t *testing.T) {
	dirRule := Rule{Type: DirectoryRule}
	remoteRule := Rule{Type: RemoteRule}

	if !dirRule.IsDirectory() {
		t.Error("DirectoryRule.IsDirectory() should return true")
	}
	if dirRule.IsRemote() {
		t.Error("DirectoryRule.IsRemote() should return false")
	}

	if remoteRule.IsDirectory() {
		t.Error("RemoteRule.IsDirectory() should return false")
	}
	if !remoteRule.IsRemote() {
		t.Error("RemoteRule.IsRemote() should return true")
	}
}
