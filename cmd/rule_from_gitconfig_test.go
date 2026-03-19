package cmd

import (
	"strings"
	"testing"
)

func TestGitconfigToGitchPattern(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"~/work/", "~/work/**"},
		{"~/work", "~/work/**"},
		{"~/work/**", "~/work/**"},
		{"~/work/*", "~/work/*"},
		{"/home/user/projects/", "/home/user/projects/**"},
		{"/home/user/projects", "/home/user/projects/**"},
		{"~/company///", "~/company/**"},
		{"~/.path/", "~/.path/**"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := gitconfigToGitchPattern(tt.input)
			if got != tt.want {
				t.Errorf("gitconfigToGitchPattern(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseIncludeIfLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantOK  bool
		wantPat string
		wantCfg string
	}{
		{
			name:    "standard gitdir with tilde",
			line:    "includeif.gitdir:~/work/.path /tmp/gitconfig-work",
			wantOK:  true,
			wantPat: "~/work/",
			wantCfg: "/tmp/gitconfig-work",
		},
		{
			name:    "case-insensitive gitdir",
			line:    "includeif.gitdir/i:~/Work/.path /tmp/gitconfig-work",
			wantOK:  true,
			wantPat: "~/Work/",
			wantCfg: "/tmp/gitconfig-work",
		},
		{
			name:    "absolute path pattern",
			line:    "includeif.gitdir:/home/user/projects/.path /tmp/gitconfig-projects",
			wantOK:  true,
			wantPat: "/home/user/projects/",
			wantCfg: "/tmp/gitconfig-projects",
		},
		{
			name:    "pattern with glob",
			line:    "includeif.gitdir:~/src/**/.path /tmp/gitconfig-src",
			wantOK:  true,
			wantPat: "~/src/**/",
			wantCfg: "/tmp/gitconfig-src",
		},
		{
			name:   "non-gitdir condition (onbranch)",
			line:   "includeif.onbranch:main.path ~/.gitconfig-main",
			wantOK: false,
		},
		{
			name:   "hasconfig condition",
			line:   "includeif.hasconfig:remote.*.url:https://github.com/**/.path ~/github.inc",
			wantOK: false,
		},
		{
			name:   "malformed line no space",
			line:   "includeif.gitdir:~/work/.path",
			wantOK: false,
		},
		{
			name:   "missing prefix",
			line:   "other.gitdir:~/work/.path /tmp/cfg",
			wantOK: false,
		},
		{
			name:   "missing suffix",
			line:   "includeif.gitdir:~/work/.other /tmp/cfg",
			wantOK: false,
		},
		{
			name:   "empty line",
			line:   "",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseIncludeIfLine(tt.line)
			if ok != tt.wantOK {
				t.Fatalf("parseIncludeIfLine(%q) ok = %v, want %v", tt.line, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.GitdirPattern != tt.wantPat {
				t.Errorf("GitdirPattern = %q, want %q", got.GitdirPattern, tt.wantPat)
			}
			// ConfigPath may be expanded, so check with absolute path
			if tt.wantCfg != "" && got.ConfigPath != tt.wantCfg {
				t.Errorf("ConfigPath = %q, want %q", got.ConfigPath, tt.wantCfg)
			}
		})
	}
}

func TestParseIncludeIfLine_ConfigPathWithSpaces(t *testing.T) {
	line := "includeif.gitdir:~/work/.path /home/user/my configs/work"
	got, ok := parseIncludeIfLine(line)
	if !ok {
		t.Fatal("expected ok=true for path with spaces")
	}
	if got.GitdirPattern != "~/work/" {
		t.Errorf("GitdirPattern = %q, want %q", got.GitdirPattern, "~/work/")
	}
	// The value should include everything after the first space
	if !strings.Contains(got.ConfigPath, "my configs") {
		t.Errorf("ConfigPath should contain full path with spaces, got %q", got.ConfigPath)
	}
}
