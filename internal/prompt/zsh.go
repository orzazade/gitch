package prompt

// ZshInit returns shell integration code for zsh prompt.
// The output should be evaled: eval "$(gitch init zsh)"
func ZshInit() string {
	return `# gitch shell integration for zsh
# Add to ~/.zshrc: eval "$(gitch init zsh)"

# Function to get current gitch identity
_gitch_prompt() {
  local identity
  identity=$(cat "${XDG_CACHE_HOME:-$HOME/.cache}/gitch/current-identity" 2>/dev/null)
  if [[ -n "$identity" ]]; then
    echo -n "%F{cyan}[${identity}]%f "
  fi
}

# Prepend gitch identity to prompt
setopt PROMPT_SUBST
PROMPT='$(_gitch_prompt)'"${PROMPT}"
`
}
