package prompt

import "fmt"

// ZshInit returns shell integration code for zsh prompt.
// The output should be evaled: eval "$(gitch init zsh)"
func ZshInit() string {
	cachePath, _ := cachePath()
	return fmt.Sprintf(`# gitch shell integration for zsh
# Add to ~/.zshrc: eval "$(gitch init zsh)"

autoload -Uz add-zsh-hook

# Function to get current gitch identity
_gitch_prompt() {
  local identity
  identity=$(cat "%s" 2>/dev/null)
  if [[ -n "$identity" ]]; then
    echo -n "%%F{cyan}[${identity}]%%f "
  fi
}

_gitch_autoswitch() {
  if [[ "$PWD" != "$_GITCH_LAST_PWD" ]]; then
    _GITCH_LAST_PWD="$PWD"
    gitch autoswitch --quiet >/dev/null 2>&1
  fi
}

add-zsh-hook chpwd _gitch_autoswitch
_gitch_autoswitch

# Prepend gitch identity to prompt
setopt PROMPT_SUBST
PROMPT='$(_gitch_prompt)'"${PROMPT}"
`, cachePath)
}
