package hooks

// PreCommitScript is the bash script installed as pre-commit hook
const PreCommitScript = `#!/bin/bash
# gitch pre-commit hook - validates identity before commit

# Check for bypass
if [ "$GITCH_BYPASS" = "1" ]; then
    exit 0
fi

# Run gitch validation
result=$(gitch hook validate 2>&1)
exit_code=$?

if [ $exit_code -eq 0 ]; then
    exit 0
fi

# Identity mismatch detected
# Check if we have an interactive terminal
if [ -t 0 ]; then
    # Redirect stdin from tty for interactive prompt
    exec < /dev/tty

    echo "$result"
    echo ""
    while true; do
        read -p "[S]witch identity, [C]ontinue anyway, [A]bort? " choice
        case $choice in
            [Ss]) gitch hook switch && exit 0 || exit 1 ;;
            [Cc]) exit 0 ;;
            [Aa]) exit 1 ;;
            *) echo "Please answer S, C, or A" ;;
        esac
    done
else
    # Non-interactive mode - check hook mode from gitch
    mode=$(gitch hook mode 2>/dev/null || echo "warn")
    case $mode in
        allow) exit 0 ;;
        warn)  echo "$result"; exit 0 ;;
        block) echo "$result"; exit 1 ;;
    esac
fi
`
