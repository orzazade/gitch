package prompt

import (
	"strings"
	"testing"
)

func TestShellInitScriptsTriggerAutoSwitch(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{name: "bash", script: BashInit()},
		{name: "zsh", script: ZshInit()},
		{name: "fish", script: FishInit()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.script, "gitch autoswitch --quiet") {
				t.Fatalf("expected %s init script to trigger autoswitch, got:\n%s", tt.name, tt.script)
			}
		})
	}
}
