package git

import (
	"testing"
)

func TestIsAzureDevOpsRemote(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Azure DevOps HTTPS URLs (modern)
		{
			name:     "Azure DevOps HTTPS",
			url:      "https://dev.azure.com/org/project/_git/repo",
			expected: true,
		},
		{
			name:     "Azure DevOps HTTPS with username",
			url:      "https://user@dev.azure.com/org/project/_git/repo",
			expected: true,
		},

		// Azure DevOps SSH URLs (modern)
		{
			name:     "Azure DevOps SSH v3",
			url:      "git@ssh.dev.azure.com:v3/org/project/repo",
			expected: true,
		},
		{
			name:     "Azure DevOps SSH URL format",
			url:      "ssh://git@ssh.dev.azure.com/v3/org/project/repo",
			expected: true,
		},

		// Azure DevOps legacy (visualstudio.com)
		{
			name:     "Azure DevOps legacy HTTPS",
			url:      "https://org.visualstudio.com/project/_git/repo",
			expected: true,
		},
		{
			name:     "Azure DevOps legacy HTTPS DefaultCollection",
			url:      "https://org.visualstudio.com/DefaultCollection/project/_git/repo",
			expected: true,
		},
		{
			name:     "Azure DevOps legacy SSH",
			url:      "git@vs-ssh.visualstudio.com:v3/org/project/repo",
			expected: true,
		},
		{
			name:     "Azure DevOps legacy SSH URL format",
			url:      "ssh://git@vs-ssh.visualstudio.com/v3/org/project/repo",
			expected: true,
		},

		// GitHub (should return false)
		{
			name:     "GitHub HTTPS",
			url:      "https://github.com/user/repo.git",
			expected: false,
		},
		{
			name:     "GitHub SSH",
			url:      "git@github.com:user/repo.git",
			expected: false,
		},

		// GitLab (should return false)
		{
			name:     "GitLab HTTPS",
			url:      "https://gitlab.com/user/repo.git",
			expected: false,
		},
		{
			name:     "GitLab SSH",
			url:      "git@gitlab.com:user/repo.git",
			expected: false,
		},
		{
			name:     "GitLab self-hosted",
			url:      "https://gitlab.company.com/user/repo.git",
			expected: false,
		},

		// Bitbucket (should return false)
		{
			name:     "Bitbucket HTTPS",
			url:      "https://bitbucket.org/user/repo.git",
			expected: false,
		},
		{
			name:     "Bitbucket SSH",
			url:      "git@bitbucket.org:user/repo.git",
			expected: false,
		},

		// Edge cases
		{
			name:     "empty string",
			url:      "",
			expected: false,
		},
		{
			name:     "invalid URL",
			url:      "not-a-url",
			expected: false,
		},
		{
			name:     "local path",
			url:      "/path/to/repo",
			expected: false,
		},
		{
			name:     "file URL",
			url:      "file:///path/to/repo",
			expected: false,
		},

		// Case insensitivity
		{
			name:     "Azure DevOps uppercase",
			url:      "https://DEV.AZURE.COM/org/project/_git/repo",
			expected: true,
		},
		{
			name:     "Azure DevOps mixed case",
			url:      "https://Dev.Azure.Com/org/project/_git/repo",
			expected: true,
		},
		{
			name:     "VisualStudio uppercase",
			url:      "https://ORG.VISUALSTUDIO.COM/project/_git/repo",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAzureDevOpsRemote(tt.url)
			if result != tt.expected {
				t.Errorf("IsAzureDevOpsRemote(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}
