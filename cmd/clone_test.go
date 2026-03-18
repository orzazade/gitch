package cmd

import (
	"testing"
)

func TestResolveCloneDir_ExplicitTarget(t *testing.T) {
	dir, err := resolveCloneDir("git@github.com:org/repo.git", "/tmp/myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != "/tmp/myrepo" {
		t.Fatalf("got %q, want %q", dir, "/tmp/myrepo")
	}
}

func TestResolveCloneDir_HTTPS(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/org/project.git", "project"},
		{"https://github.com/org/project", "project"},
		{"https://github.com/org/my-tool.git", "my-tool"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			dir, err := resolveCloneDir(tt.url, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// dir is absolute, so just check the base name
			if base := baseName(dir); base != tt.want {
				t.Fatalf("got %q, want %q", base, tt.want)
			}
		})
	}
}

func TestResolveCloneDir_SCP(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"git@github.com:org/repo.git", "repo"},
		{"git@gitlab.com:group/subgroup/project.git", "project"},
		{"git@github.com:org/my-app", "my-app"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			dir, err := resolveCloneDir(tt.url, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if base := baseName(dir); base != tt.want {
				t.Fatalf("got %q, want %q", base, tt.want)
			}
		})
	}
}

func TestResolveCloneDir_EmptyName(t *testing.T) {
	_, err := resolveCloneDir("", "")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}
