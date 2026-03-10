# gitch - Git Identity Manager for VS Code

Never commit with the wrong git identity again.

## Features

- **Status Bar Identity** - See the active profile for the current repository or workspace
- **Quick Switch** - Click the status bar to switch between identities
- **Auto-Switch** - Automatically switch identity based on workspace rules
- **Mismatch Warnings** - Get notified when a profile is unmanaged or only partially applied

## Requirements

The extension automatically downloads the `gitch` CLI on first activation. Alternatively, install via:

```bash
# macOS
brew install orzazade/tap/gitch

# Linux / Windows
# Download the appropriate package or archive from GitHub Releases
```

## Usage

1. Open a Git repository in VS Code
2. Look at the status bar (bottom left) - you'll see your current identity
3. Click the status bar to switch identities
4. Set up auto-switch rules with `gitch rule add` in the terminal

In multi-root workspaces, the extension prefers the active editor's repository and falls back to the first workspace folder when no editor is focused.

## Extension Settings

- `gitch.binaryPath`: Custom path to gitch binary
- `gitch.autoDownload`: Automatically download gitch if not found (default: true)

## Links

- [GitHub Repository](https://github.com/orzazade/gitch)
- [CLI Documentation](https://github.com/orzazade/gitch#readme)
- [Report Issues](https://github.com/orzazade/gitch/issues)

## License

MIT
