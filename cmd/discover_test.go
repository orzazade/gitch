package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
)

func TestDiscoverIdentities_GroupsByEmail(t *testing.T) {
	root := t.TempDir()

	repoA := root + "/project-a"
	repoB := root + "/project-b"

	for _, dir := range []string{repoA, repoB} {
		initGitRepo(t, dir)
		setGitIdentity(t, dir, "Work User", "work@example.com")
	}

	cfg := &config.Config{Identities: []config.Identity{}}

	identities := discoverIdentities([]string{repoA, repoB}, cfg)

	if len(identities) != 1 {
		t.Fatalf("expected 1 unique identity, got %d", len(identities))
	}
	if identities[0].Email != "work@example.com" {
		t.Errorf("expected email work@example.com, got %q", identities[0].Email)
	}
	if len(identities[0].Repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(identities[0].Repos))
	}
}

func TestDiscoverIdentities_DistinguishesDifferentEmails(t *testing.T) {
	root := t.TempDir()

	repoWork := root + "/work-project"
	repoPersonal := root + "/personal-project"

	initGitRepo(t, repoWork)
	setGitIdentity(t, repoWork, "Work User", "work@company.com")

	initGitRepo(t, repoPersonal)
	setGitIdentity(t, repoPersonal, "Personal User", "me@personal.com")

	cfg := &config.Config{Identities: []config.Identity{}}

	identities := discoverIdentities([]string{repoWork, repoPersonal}, cfg)

	if len(identities) != 2 {
		t.Fatalf("expected 2 unique identities, got %d", len(identities))
	}
}

func TestDiscoverIdentities_MarksManagedIdentities(t *testing.T) {
	root := t.TempDir()

	repo := root + "/project"
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Work User", "work@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}

	identities := discoverIdentities([]string{repo}, cfg)

	if len(identities) != 1 {
		t.Fatalf("expected 1 identity, got %d", len(identities))
	}
	if identities[0].Managed != "work" {
		t.Errorf("expected managed='work', got %q", identities[0].Managed)
	}
}

func TestDiscoverIdentities_UnmanagedIdentity(t *testing.T) {
	root := t.TempDir()

	repo := root + "/project"
	initGitRepo(t, repo)
	setGitIdentity(t, repo, "Unknown", "unknown@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}

	identities := discoverIdentities([]string{repo}, cfg)

	if len(identities) != 1 {
		t.Fatalf("expected 1 identity, got %d", len(identities))
	}
	if identities[0].Managed != "" {
		t.Errorf("expected unmanaged (empty), got %q", identities[0].Managed)
	}
}

func TestDiscoverIdentities_SortedByRepoCount(t *testing.T) {
	root := t.TempDir()

	// Create 3 repos with "common" identity, 1 repo with "rare" identity
	for _, name := range []string{"a", "b", "c"} {
		dir := root + "/" + name
		initGitRepo(t, dir)
		setGitIdentity(t, dir, "Common", "common@example.com")
	}
	rareDir := root + "/rare"
	initGitRepo(t, rareDir)
	setGitIdentity(t, rareDir, "Rare", "rare@example.com")

	cfg := &config.Config{Identities: []config.Identity{}}
	repos := []string{root + "/a", root + "/b", root + "/c", rareDir}

	identities := discoverIdentities(repos, cfg)

	if len(identities) != 2 {
		t.Fatalf("expected 2 identities, got %d", len(identities))
	}
	// Most-used identity should be first
	if identities[0].Email != "common@example.com" {
		t.Errorf("expected most-used identity first, got %q", identities[0].Email)
	}
}

func TestDiscoverIdentities_SkipsReposWithNoEmail(t *testing.T) {
	// Isolate from system git config so no fallback email is found
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	root := t.TempDir()

	repo := root + "/project"
	initGitRepo(t, repo)
	// Don't set any identity — email will be empty

	cfg := &config.Config{Identities: []config.Identity{}}

	identities := discoverIdentities([]string{repo}, cfg)

	if len(identities) != 0 {
		t.Fatalf("expected 0 identities for repo with no email, got %d", len(identities))
	}
}

func TestDiscoverIdentities_CaseInsensitiveEmailGrouping(t *testing.T) {
	root := t.TempDir()

	repoA := root + "/project-a"
	repoB := root + "/project-b"

	initGitRepo(t, repoA)
	setGitIdentity(t, repoA, "User", "Work@Example.COM")

	initGitRepo(t, repoB)
	setGitIdentity(t, repoB, "User", "work@example.com")

	cfg := &config.Config{Identities: []config.Identity{}}

	identities := discoverIdentities([]string{repoA, repoB}, cfg)

	if len(identities) != 1 {
		t.Fatalf("expected 1 identity (case-insensitive grouping), got %d", len(identities))
	}
	if len(identities[0].Repos) != 2 {
		t.Errorf("expected 2 repos grouped together, got %d", len(identities[0].Repos))
	}
}
