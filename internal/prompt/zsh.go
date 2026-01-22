package prompt

import "fmt"

// ZshInit returns shell integration code for zsh prompt.
// The output should be evaled: eval "$(gitch init zsh)"
func ZshInit() string {
	cachePath, _ := CachePath()
	return fmt.Sprintf(`# gitch shell integration for zsh
# Add to ~/.zshrc: eval "$(gitch init zsh)"

# Function to get current gitch identity
_gitch_prompt() {
  local identity
  identity=$(cat "%s" 2>/dev/null)
  if [[ -n "$identity" ]]; then
    echo -n "%%F{cyan}[${identity}]%%f "
  fi
}

# Prepend gitch identity to prompt
setopt PROMPT_SUBST
PROMPT='$(_gitch_prompt)'"${PROMPT}"
`, cachePath)
}
