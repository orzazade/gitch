package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestNounPlural(t *testing.T) {
	cases := []struct {
		n        int
		singular string
		plural   string
		want     string
	}{
		{1, "identity", "identities", "identity"},
		{0, "identity", "identities", "identities"},
		{2, "identity", "identities", "identities"},
		{1, "issue", "issues", "issue"},
		{3, "issue", "issues", "issues"},
	}
	for _, tc := range cases {
		got := nounPlural(tc.n, tc.singular, tc.plural)
		if got != tc.want {
			t.Errorf("nounPlural(%d, %q, %q) = %q, want %q", tc.n, tc.singular, tc.plural, got, tc.want)
		}
	}
}

func TestDoctorCmd_NoIdentities(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	cfg := &config.Config{Identities: []config.Identity{}}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var checks []doctorCheck
	checks = append(checks, doctorCheck{false, "no identities configured", "run: gitch add"})

	warnings := 0
	for _, c := range checks {
		if !c.ok {
			warnings++
		}
	}
	if warnings != 1 {
		t.Errorf("expected 1 warning for no identities, got %d", warnings)
	}
}

func TestPrintDoctorResults_ReturnsWarningCount(t *testing.T) {
	cases := []struct {
		name   string
		checks []doctorCheck
		want   int
	}{
		{"all ok", []doctorCheck{{true, "a", ""}, {true, "b", ""}}, 0},
		{"one warning", []doctorCheck{{true, "a", ""}, {false, "b", "fix b"}}, 1},
		{"all warnings", []doctorCheck{{false, "a", ""}, {false, "b", ""}}, 2},
		{"empty", []doctorCheck{}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := printDoctorResults(tc.checks)
			if got != tc.want {
				t.Errorf("printDoctorResults() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDoctorCmd_ActiveIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	if err := git.ApplyIdentityScoped("Jane Doe", "jane@example.com", git.ScopeLocal); err != nil {
		t.Fatalf("failed to apply identity: %v", err)
	}

	state, err := resolveCurrentProfileState(cfg)
	if err != nil {
		t.Fatalf("resolveCurrentProfileState failed: %v", err)
	}
	if state.ExactMatch == nil {
		t.Fatal("expected exact match, got nil")
	}
	if state.ExactMatch.Name != "work" {
		t.Errorf("expected active identity 'work', got %q", state.ExactMatch.Name)
	}
}
