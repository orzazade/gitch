package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", dir)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %s: %s", dir, err, out)
	}
}

func setGitIdentity(t *testing.T, dir, name, email string) {
	t.Helper()
	for _, kv := range [][2]string{
		{"user.name", name},
		{"user.email", email},
	} {
		cmd := exec.Command("git", "config", "--local", kv[0], kv[1])
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git config %s %s: %s: %s", kv[0], kv[1], err, out)
		}
	}
}

func TestFindGitRepos(t *testing.T) {
	root := t.TempDir()

	// Create some git repos
	repoA := filepath.Join(root, "project-a")
	repoB := filepath.Join(root, "org", "project-b")
	notRepo := filepath.Join(root, "notes")

	for _, dir := range []string{repoA, repoB, notRepo} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	initGitRepo(t, repoA)
	initGitRepo(t, repoB)

	repos := findGitRepos([]string{root}, 3)

	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d: %v", len(repos), repos)
	}

	// Both repos should be found
	found := make(map[string]bool)
	for _, r := range repos {
		found[r] = true
	}
	if !found[repoA] {
		t.Errorf("expected to find %s", repoA)
	}
	if !found[repoB] {
		t.Errorf("expected to find %s", repoB)
	}
}

func TestFindGitRepos_DepthLimit(t *testing.T) {
	root := t.TempDir()

	shallow := filepath.Join(root, "shallow")
	deep := filepath.Join(root, "a", "b", "c", "deep")

	for _, dir := range []string{shallow, deep} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	initGitRepo(t, shallow)
	initGitRepo(t, deep)

	// depth=2 should find shallow but not deep (which is at depth 4)
	repos := findGitRepos([]string{root}, 2)

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo at depth<=2, got %d: %v", len(repos), repos)
	}
	if repos[0] != shallow {
		t.Errorf("expected %s, got %s", shallow, repos[0])
	}
}

func TestFindGitRepos_SkipsHiddenDirs(t *testing.T) {
	root := t.TempDir()

	visible := filepath.Join(root, "visible")
	hidden := filepath.Join(root, ".hidden", "repo")

	for _, dir := range []string{visible, hidden} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	initGitRepo(t, visible)
	initGitRepo(t, hidden)

	repos := findGitRepos([]string{root}, 3)

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo (hidden skipped), got %d: %v", len(repos), repos)
	}
}

func TestFindGitRepos_NoDuplicates(t *testing.T) {
	root := t.TempDir()

	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repo)

	// Pass the same root twice
	repos := findGitRepos([]string{root, root}, 3)

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo (deduped), got %d: %v", len(repos), repos)
	}
}

func TestScanSingleRepo_ManagedIdentity(t *testing.T) {
	repo := t.TempDir()
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Work User", "work@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}

	result := scanSingleRepo(repo, cfg, false)

	if !result.Managed {
		t.Error("expected identity to be managed")
	}
	if result.Identity != "work" {
		t.Errorf("expected identity 'work', got %q", result.Identity)
	}
	if result.Email != "work@example.com" {
		t.Errorf("expected email 'work@example.com', got %q", result.Email)
	}
}

func TestScanSingleRepo_UnmanagedIdentity(t *testing.T) {
	repo := t.TempDir()
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Random", "random@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}

	result := scanSingleRepo(repo, cfg, false)

	if result.Managed {
		t.Error("expected identity to be unmanaged")
	}
	if result.hasIssue() != true {
		t.Error("expected unmanaged repo to have an issue")
	}
}

func TestScanSingleRepo_RuleMismatch(t *testing.T) {
	repo := t.TempDir()
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Personal", "personal@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "personal", Email: "personal@example.com"},
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: repo + "/**", Identity: "work"},
		},
	}

	result := scanSingleRepo(repo, cfg, false)

	if !result.RuleMismatch {
		t.Error("expected rule mismatch")
	}
	if result.RuleMatch != "work" {
		t.Errorf("expected rule match 'work', got %q", result.RuleMatch)
	}
}

func TestScanSingleRepo_NoIdentityWithRule(t *testing.T) {
	repo := t.TempDir()
	initGitRepo(t, repo)
	// No identity set — email is empty

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: repo + "/**", Identity: "work"},
		},
	}

	result := scanSingleRepo(repo, cfg, false)

	if !result.RuleMismatch {
		t.Error("expected rule mismatch when no identity is configured but a rule matches")
	}
	if result.RuleMatch != "work" {
		t.Errorf("expected rule match 'work', got %q", result.RuleMatch)
	}
}

func TestScanSingleRepo_GlobalHookCovered(t *testing.T) {
	repo := t.TempDir()
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Work", "work@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}

	// Simulate global hook being installed
	result := scanSingleRepo(repo, cfg, true)

	if result.hasIssue() {
		t.Error("expected no issues when globally hooked and managed")
	}
	if !result.HookGlobal {
		t.Error("expected HookGlobal to be true")
	}
}

func TestRepoResult_HasIssue(t *testing.T) {
	tests := []struct {
		name     string
		result   repoResult
		expected bool
	}{
		{
			name:     "all good with global hook",
			result:   repoResult{Managed: true, HookGlobal: true},
			expected: false,
		},
		{
			name:     "all good with local hook",
			result:   repoResult{Managed: true, HookLocal: true},
			expected: false,
		},
		{
			name:     "not managed",
			result:   repoResult{Managed: false, HookGlobal: true},
			expected: true,
		},
		{
			name:     "no hook",
			result:   repoResult{Managed: true, HookGlobal: false, HookLocal: false},
			expected: true,
		},
		{
			name:     "rule mismatch",
			result:   repoResult{Managed: true, HookGlobal: true, RuleMismatch: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.hasIssue(); got != tt.expected {
				t.Errorf("hasIssue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFixScanMismatches(t *testing.T) {
	// Create a repo with a mismatch: has personal email but rule says work
	repo := t.TempDir()
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Personal User", "personal@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
			{Name: "personal", Email: "personal@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: repo + "/**", Identity: "work"},
		},
	}

	// Verify mismatch is detected
	result := scanSingleRepo(repo, cfg, false)
	if !result.RuleMismatch {
		t.Fatal("expected rule mismatch before fix")
	}

	// Fix the mismatch
	if err := fixScanMismatches([]repoResult{result}, cfg); err != nil {
		t.Fatalf("fixScanMismatches: %v", err)
	}

	// Re-scan: the mismatch should be resolved
	after := scanSingleRepo(repo, cfg, false)
	if after.RuleMismatch {
		t.Errorf("repo still has a mismatch after fix (email=%s)", after.Email)
	}
	if after.Email != "work@example.com" {
		t.Errorf("expected email work@example.com after fix, got %s", after.Email)
	}
}

func TestFixScanMismatches_NoMismatches(t *testing.T) {
	results := []repoResult{
		{Path: "/tmp/fake", Managed: true, HookGlobal: true, RuleMismatch: false},
	}

	cfg := &config.Config{}

	// Should succeed silently with no mismatches to fix
	if err := fixScanMismatches(results, cfg); err != nil {
		t.Fatalf("fixScanMismatches: %v", err)
	}
}

func TestPrintScanJSON(t *testing.T) {
	results := []repoResult{
		{
			Path:       "/home/user/work/project-a",
			Identity:   "work",
			Email:      "work@company.com",
			Managed:    true,
			HookGlobal: true,
			RuleMatch:  "work",
		},
		{
			Path:         "/home/user/oss/project-b",
			Identity:     "Personal",
			Email:        "personal@gmail.com",
			Managed:      false,
			HookGlobal:   false,
			HookLocal:    false,
			RuleMatch:    "oss",
			RuleMismatch: true,
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printScanJSON(results)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("printScanJSON: %v", err)
	}

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	var out scanOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nraw: %s", err, output)
	}

	if out.Total != 2 {
		t.Errorf("total: got %d, want 2", out.Total)
	}
	if out.Issues != 1 {
		t.Errorf("issues: got %d, want 1", out.Issues)
	}
	if len(out.Repos) != 2 {
		t.Fatalf("repos: got %d, want 2", len(out.Repos))
	}

	repoA := out.Repos[0]
	if repoA.Path != "/home/user/work/project-a" {
		t.Errorf("repo[0].path: got %q", repoA.Path)
	}
	if repoA.HasIssue {
		t.Error("repo[0] should not have issue")
	}
	if !repoA.Managed {
		t.Error("repo[0] should be managed")
	}

	repoB := out.Repos[1]
	if !repoB.HasIssue {
		t.Error("repo[1] should have issue")
	}
	if !repoB.RuleMismatch {
		t.Error("repo[1] should have rule mismatch")
	}
	if repoB.RuleMatch != "oss" {
		t.Errorf("repo[1].rule_match: got %q, want %q", repoB.RuleMatch, "oss")
	}
}

func TestIsGitRepoDir(t *testing.T) {
	t.Run("valid repo", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)
		if !isGitRepoDir(dir) {
			t.Error("expected true for git repo")
		}
	})

	t.Run("not a repo", func(t *testing.T) {
		dir := t.TempDir()
		if isGitRepoDir(dir) {
			t.Error("expected false for non-repo")
		}
	})
}
