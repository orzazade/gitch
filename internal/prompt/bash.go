package prompt

// BashInit returns shell integration code for bash prompt.
// The output should be evaled: eval "$(gitch init bash)"
func BashInit() string {
	return `# gitch shell integration for bash
# Add to ~/.bashrc: eval "$(gitch init bash)"

# Function to get current gitch identity
_gitch_prompt() {
  local identity
  identity=$(cat "${XDG_CACHE_HOME:-$HOME/.cache}/gitch/current-identity" 2>/dev/null)
  if [[ -n "$identity" ]]; then
    printf '\[\e[36m\][%s]\[\e[0m\] ' "$identity"
  fi
}

# Save original PS1 if not already saved
[[ -z "$_GITCH_ORIGINAL_PS1" ]] && _GITCH_ORIGINAL_PS1="$PS1"

# Update PS1 with gitch identity
_gitch_update_ps1() {
  PS1="$(_gitch_prompt)${_GITCH_ORIGINAL_PS1}"
}

# Run on each prompt
PROMPT_COMMAND="_gitch_update_ps1${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
`
}
