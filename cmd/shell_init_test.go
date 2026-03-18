package cmd

import (
	"strings"
	"testing"
)

func TestBashHook(t *testing.T) {
	script := bashHook("/usr/bin/gitch")

	if !strings.Contains(script, "_gitch_hook") {
		t.Error("bash hook should define _gitch_hook function")
	}
	if !strings.Contains(script, "PROMPT_COMMAND") {
		t.Error("bash hook should register via PROMPT_COMMAND")
	}
	if !strings.Contains(script, "/usr/bin/gitch autoswitch --quiet") {
		t.Error("bash hook should call gitch autoswitch --quiet")
	}
	if !strings.Contains(script, "GITCH_DISABLE") {
		t.Error("bash hook should check GITCH_DISABLE env var")
	}
	if !strings.Contains(script, "_GITCH_PREV_DIR") {
		t.Error("bash hook should skip when directory hasn't changed")
	}
}

func TestZshHook(t *testing.T) {
	script := zshHook("/usr/bin/gitch")

	if !strings.Contains(script, "_gitch_hook") {
		t.Error("zsh hook should define _gitch_hook function")
	}
	if !strings.Contains(script, "chpwd_functions") {
		t.Error("zsh hook should register via chpwd_functions")
	}
	if !strings.Contains(script, "/usr/bin/gitch autoswitch --quiet") {
		t.Error("zsh hook should call gitch autoswitch --quiet")
	}
	if !strings.Contains(script, "GITCH_DISABLE") {
		t.Error("zsh hook should check GITCH_DISABLE env var")
	}
}

func TestFishHook(t *testing.T) {
	script := fishHook("/usr/bin/gitch")

	if !strings.Contains(script, "_gitch_hook") {
		t.Error("fish hook should define _gitch_hook function")
	}
	if !strings.Contains(script, "--on-variable PWD") {
		t.Error("fish hook should trigger on PWD change")
	}
	if !strings.Contains(script, "/usr/bin/gitch autoswitch --quiet") {
		t.Error("fish hook should call gitch autoswitch --quiet")
	}
	if !strings.Contains(script, "GITCH_DISABLE") {
		t.Error("fish hook should check GITCH_DISABLE env var")
	}
}

func TestHooksUseProvidedPath(t *testing.T) {
	for _, tt := range []struct {
		name string
		fn   func(string) string
	}{
		{"bash", bashHook},
		{"zsh", zshHook},
		{"fish", fishHook},
	} {
		t.Run(tt.name, func(t *testing.T) {
			script := tt.fn("/custom/path/gitch")
			if !strings.Contains(script, "/custom/path/gitch") {
				t.Error("hook should use the provided executable path")
			}
		})
	}
}

func TestBashHookIdempotent(t *testing.T) {
	script := bashHook("gitch")
	// The bash hook checks if _gitch_hook is already in PROMPT_COMMAND
	if !strings.Contains(script, `"_gitch_hook"`) {
		t.Error("bash hook should guard against duplicate registration")
	}
}

func TestZshHookIdempotent(t *testing.T) {
	script := zshHook("gitch")
	// The zsh hook checks if _gitch_hook is already in chpwd_functions
	if !strings.Contains(script, `chpwd_functions[(r)_gitch_hook]`) {
		t.Error("zsh hook should guard against duplicate registration")
	}
}
