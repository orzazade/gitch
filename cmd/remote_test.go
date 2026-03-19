package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func captureRemoteOutput(t *testing.T, json bool) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	remoteJSON = json
	cmd := remoteCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func TestRemote_NoRemotes(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRemoteOutput(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "No remotes") {
		t.Errorf("expected no-remotes message, got: %s", output)
	}
}

func TestRemote_SingleRemoteMatches(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "remote", "add", "origin", "git@github.com:workorg/project.git")
	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
		Rules: []rules.Rule{
			{Type: rules.RemoteRule, Pattern: "github.com/workorg/*", Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRemoteOutput(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "origin") {
		t.Errorf("expected remote name in output, got: %s", output)
	}
	if !strings.Contains(output, "ok") {
		t.Errorf("expected ok marker in output, got: %s", output)
	}
}

func TestRemote_MultipleRemotesConflict(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "remote", "add", "origin", "git@github.com:personal/project.git")
	env.Run(t, "git", "remote", "add", "upstream", "git@github.com:workorg/project.git")
	env.Run(t, "git", "config", "user.email", "personal@gmail.com")
	env.Run(t, "git", "config", "user.name", "Personal")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "personal", Email: "personal@gmail.com", GitName: "Personal"},
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
		Rules: []rules.Rule{
			{Type: rules.RemoteRule, Pattern: "github.com/personal/*", Identity: "personal"},
			{Type: rules.RemoteRule, Pattern: "github.com/workorg/*", Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRemoteOutput(t, false)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if err != errRemoteConflict {
		t.Errorf("expected errRemoteConflict, got: %v", err)
	}
	if !strings.Contains(output, "origin") {
		t.Errorf("expected origin in output, got: %s", output)
	}
	if !strings.Contains(output, "upstream") {
		t.Errorf("expected upstream in output, got: %s", output)
	}
	if !strings.Contains(output, "personal") {
		t.Errorf("expected personal identity in output, got: %s", output)
	}
	if !strings.Contains(output, "work") {
		t.Errorf("expected work identity in output, got: %s", output)
	}
}

func TestRemote_NoMatchingRules(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "remote", "add", "origin", "git@github.com:someorg/project.git")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.RemoteRule, Pattern: "github.com/different/*", Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRemoteOutput(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "no matching rule") {
		t.Errorf("expected no-matching-rule message, got: %s", output)
	}
}

func TestRemote_JSON_Conflict(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "remote", "add", "origin", "git@github.com:personal/project.git")
	env.Run(t, "git", "remote", "add", "upstream", "git@github.com:workorg/project.git")
	env.Run(t, "git", "config", "user.email", "personal@gmail.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "personal", Email: "personal@gmail.com"},
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.RemoteRule, Pattern: "github.com/personal/*", Identity: "personal"},
			{Type: rules.RemoteRule, Pattern: "github.com/workorg/*", Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRemoteOutput(t, true)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if !strings.Contains(output, `"has_conflict": true`) {
		t.Errorf("expected has_conflict:true in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"identity": "personal"`) {
		t.Errorf("expected personal identity in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"identity": "work"`) {
		t.Errorf("expected work identity in JSON, got: %s", output)
	}
}

func TestRemote_JSON_NoConflict(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "remote", "add", "origin", "git@github.com:workorg/project.git")
	env.Run(t, "git", "remote", "add", "upstream", "git@github.com:workorg/other.git")
	env.Run(t, "git", "config", "user.email", "work@example.com")

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.RemoteRule, Pattern: "github.com/workorg/*", Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRemoteOutput(t, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, `"has_conflict": false`) {
		t.Errorf("expected has_conflict:false in JSON, got: %s", output)
	}
	if !strings.Contains(output, `"matched": true`) {
		t.Errorf("expected matched:true in JSON, got: %s", output)
	}
}

func TestRemote_DirectoryRuleAlsoMatches(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "remote", "add", "origin", "git@github.com:someorg/project.git")
	env.Run(t, "git", "config", "user.email", "work@example.com")
	env.Run(t, "git", "config", "user.name", "Worker")

	// Directory rule matches but no remote rule
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com", GitName: "Worker"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRemoteOutput(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Directory rule should match for the remote since FindBestMatch checks both
	if !strings.Contains(output, "work") {
		t.Errorf("expected work identity from directory rule, got: %s", output)
	}
}

func TestRemote_JSON_NoRemotes(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureRemoteOutput(t, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, `"has_conflict": false`) {
		t.Errorf("expected empty JSON output, got: %s", output)
	}
}
