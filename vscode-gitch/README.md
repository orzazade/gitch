# gitch - Git Identity Manager for VS Code

Never commit with the wrong git identity again.

## Features

- **Status Bar Identity** - See your current git identity at a glance
- **Quick Switch** - Click the status bar to switch between identities
- **Auto-Switch** - Automatically switch identity based on workspace rules
- **Mismatch Warnings** - Get notified when using an unmanaged identity

## Requirements

The extension automatically downloads the `gitch` CLI on first activation. Alternatively, install via:

```bash
# macOS
brew install orzazade/tap/gitch

# Linux (Debian/Ubuntu)
curl -fsSL https://orzazade.github.io/gitch/apt/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/gitch.gpg
echo "deb [signed-by=/usr/share/keyrings/gitch.gpg] https://orzazade.github.io/gitch/apt stable main" | sudo tee /etc/apt/sources.list.d/gitch.list
sudo apt update && sudo apt install gitch

# Windows
scoop bucket add gitch https://github.com/orzazade/scoop-bucket
scoop install gitch
```

## Usage

1. Open a Git repository in VS Code
2. Look at the status bar (bottom left) - you'll see your current identity
3. Click the status bar to switch identities
4. Set up auto-switch rules with `gitch rule add` in the terminal

## Extension Settings

- `gitch.binaryPath`: Custom path to gitch binary
- `gitch.autoDownload`: Automatically download gitch if not found (default: true)

## Links

- [GitHub Repository](https://github.com/orzazade/gitch)
- [CLI Documentation](https://github.com/orzazade/gitch#readme)
- [Report Issues](https://github.com/orzazade/gitch/issues)

## License

MIT
