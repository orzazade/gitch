package prompt

import "fmt"

// BashInit returns shell integration code for bash prompt.
// The output should be evaled: eval "$(gitch init bash)"
func BashInit() string {
	cachePath, _ := cachePath()
	return fmt.Sprintf(`# gitch shell integration for bash
# Add to ~/.bashrc: eval "$(gitch init bash)"

# Function to get current gitch identity
_gitch_prompt() {
  local identity
  identity=$(cat "%s" 2>/dev/null)
  if [[ -n "$identity" ]]; then
    printf '\[\e[36m\][%%s]\[\e[0m\] ' "$identity"
  fi
}

# Save original PS1 if not already saved
[[ -z "$_GITCH_ORIGINAL_PS1" ]] && _GITCH_ORIGINAL_PS1="$PS1"

_gitch_autoswitch() {
  if [[ "$PWD" != "$_GITCH_LAST_PWD" ]]; then
    _GITCH_LAST_PWD="$PWD"
    gitch autoswitch --quiet >/dev/null 2>&1
  fi
}

# Update PS1 with gitch identity
_gitch_update_ps1() {
  _gitch_autoswitch
  PS1="$(_gitch_prompt)${_GITCH_ORIGINAL_PS1}"
}

# Run on each prompt
PROMPT_COMMAND="_gitch_update_ps1${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
`, cachePath)
}
