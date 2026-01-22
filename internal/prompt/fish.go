package prompt

import "fmt"

// FishInit returns shell integration code for fish prompt.
// The output should be sourced: gitch init fish | source
func FishInit() string {
	cachePath, _ := CachePath()
	return fmt.Sprintf(`# gitch shell integration for fish
# Add to ~/.config/fish/config.fish: gitch init fish | source

# Function to get current gitch identity
function _gitch_prompt
  set -l identity (cat "%s" 2>/dev/null)
  if test -n "$identity"
    set_color cyan
    echo -n "[$identity] "
    set_color normal
  end
end

# Store original prompt function if not already stored
if not functions -q _gitch_original_fish_prompt
  functions -c fish_prompt _gitch_original_fish_prompt
end

# Override fish_prompt to prepend gitch identity
function fish_prompt
  _gitch_prompt
  _gitch_original_fish_prompt
end
`, cachePath)
}
