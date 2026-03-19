package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

func TestRunRepos_FiltersByIdentity(t *testing.T) {
	root := t.TempDir()

	// Create two repos: one with work identity, one with personal
	workRepo := filepath.Join(root, "work-project")
	personalRepo := filepath.Join(root, "personal-project")

	for _, dir := range []string{workRepo, personalRepo} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	initGitRepo(t, workRepo)
	initGitRepo(t, personalRepo)
	setGitIdentity(t, workRepo, "Work User", "work@example.com")
	setGitIdentity(t, personalRepo, "Personal User", "personal@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
			{Name: "personal", Email: "personal@example.com"},
		},
	}

	allRepos := findGitRepos([]string{root}, 3)
	results, err := scanRepos(allRepos, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Filter for "work" identity
	workIdentity, _ := cfg.GetIdentity("work")
	var matched []reposRepoEntry
	for _, r := range results {
		if r.Managed && r.Identity == "work" {
			matched = append(matched, reposRepoEntry{
				Path:  r.Path,
				Match: "active",
			})
		}
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 repo for work identity, got %d", len(matched))
	}
	if matched[0].Path != workRepo {
		t.Errorf("expected %s, got %s", workRepo, matched[0].Path)
	}
	_ = workIdentity // used for context
}

func TestRunRepos_MatchesRuleOnly(t *testing.T) {
	root := t.TempDir()

	// Create a repo with personal email but a rule that says work
	repo := filepath.Join(root, "project")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Personal", "personal@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
			{Name: "personal", Email: "personal@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: repo + "/**", Identity: "work"},
		},
	}

	allRepos := findGitRepos([]string{root}, 3)
	results, err := scanRepos(allRepos, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Filter for "work" — should match via rule even though active email is personal
	var matched []reposRepoEntry
	for _, r := range results {
		activeMatch := r.Managed && r.Identity == "work"
		ruleMatch := r.RuleMatch == "work"
		if activeMatch || ruleMatch {
			matchType := "rule"
			if activeMatch && ruleMatch {
				matchType = "both"
			} else if activeMatch {
				matchType = "active"
			}
			matched = append(matched, reposRepoEntry{
				Path:  r.Path,
				Match: matchType,
			})
		}
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 repo matched by rule, got %d", len(matched))
	}
	if matched[0].Match != "rule" {
		t.Errorf("expected match type 'rule', got %q", matched[0].Match)
	}
}

func TestRunRepos_BothActiveAndRule(t *testing.T) {
	root := t.TempDir()

	repo := filepath.Join(root, "project")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Work User", "work@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: repo + "/**", Identity: "work"},
		},
	}

	allRepos := findGitRepos([]string{root}, 3)
	results, err := scanRepos(allRepos, cfg)
	if err != nil {
		t.Fatal(err)
	}

	var matched []reposRepoEntry
	for _, r := range results {
		activeMatch := r.Managed && r.Identity == "work"
		ruleMatch := r.RuleMatch == "work"
		if activeMatch || ruleMatch {
			matchType := "active"
			if activeMatch && ruleMatch {
				matchType = "both"
			} else if ruleMatch {
				matchType = "rule"
			}
			matched = append(matched, reposRepoEntry{
				Path:  r.Path,
				Match: matchType,
			})
		}
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(matched))
	}
	if matched[0].Match != "both" {
		t.Errorf("expected match type 'both', got %q", matched[0].Match)
	}
}

func TestRunRepos_NoMatches(t *testing.T) {
	root := t.TempDir()

	repo := filepath.Join(root, "project")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Personal", "personal@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
			{Name: "personal", Email: "personal@example.com"},
		},
	}

	allRepos := findGitRepos([]string{root}, 3)
	results, err := scanRepos(allRepos, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Filter for "work" — should find nothing
	var matched []reposRepoEntry
	for _, r := range results {
		if r.Managed && r.Identity == "work" {
			matched = append(matched, reposRepoEntry{Path: r.Path})
		}
	}

	if len(matched) != 0 {
		t.Errorf("expected 0 repos for work identity, got %d", len(matched))
	}
}

func TestRunRepos_JSONOutput(t *testing.T) {
	root := t.TempDir()

	repo := filepath.Join(root, "project")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Work User", "work@example.com")

	identity := &config.Identity{Name: "work", Email: "work@example.com"}
	entries := []reposRepoEntry{
		{Path: repo, Match: "active", HookLocal: false},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printJSON(reposOutput{
		Identity: identity.Name,
		Email:    identity.Email,
		Repos:    entries,
		Total:    len(entries),
	})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("printJSON: %v", err)
	}

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	var out reposOutput
	if err := json.Unmarshal([]byte(output), &out); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, output)
	}

	if out.Identity != "work" {
		t.Errorf("identity: got %q, want %q", out.Identity, "work")
	}
	if out.Email != "work@example.com" {
		t.Errorf("email: got %q, want %q", out.Email, "work@example.com")
	}
	if out.Total != 1 {
		t.Errorf("total: got %d, want 1", out.Total)
	}
	if len(out.Repos) != 1 {
		t.Fatalf("repos: got %d, want 1", len(out.Repos))
	}
	if out.Repos[0].Match != "active" {
		t.Errorf("match: got %q, want %q", out.Repos[0].Match, "active")
	}
}

func TestRunRepos_CaseInsensitiveMatch(t *testing.T) {
	root := t.TempDir()

	repo := filepath.Join(root, "project")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Work User", "Work@Example.COM")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}

	allRepos := findGitRepos([]string{root}, 3)
	results, err := scanRepos(allRepos, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// scanSingleRepo uses EqualFold for managed check, so it should match
	var matched int
	for _, r := range results {
		if r.Managed && r.Identity == "work" {
			matched++
		}
	}

	if matched != 1 {
		t.Errorf("expected 1 case-insensitive match, got %d", matched)
	}
}
