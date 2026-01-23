# gitch VS Code Extension

VS Code extension for [gitch](https://github.com/orzazade/gitch) - Git Identity Manager.

## Features

- Automatic gitch CLI binary management
- Downloads correct binary for your platform from GitHub Releases
- Binary cascade: PATH -> cached -> auto-download

## Requirements

- VS Code 1.96.0 or later
- Git repository in workspace (extension activates on `workspaceContains:.git`)

## Extension Settings

- `gitch.binaryPath`: Custom path to gitch binary (optional)
- `gitch.autoDownload`: Auto-download binary if not found (default: true)

## Development

### Prerequisites

- Node.js 20+
- npm

### Setup

```bash
cd vscode-gitch
npm install
```

### Build

```bash
npm run compile    # Single build
npm run watch      # Watch mode
```

### Test in VS Code

1. Open `vscode-gitch/` folder in VS Code
2. Press F5 to launch Extension Development Host
3. In new window, open a folder containing `.git`
4. Observe extension activation in Output > Extension Host

### Package

```bash
npm run package
```

Creates `.vsix` file for distribution.

## Binary Cascade Logic

1. Check `gitch.binaryPath` setting - if set and file exists, use it
2. Check system PATH for `gitch` binary
3. Check extension cache (`globalStorageUri/bin/gitch`)
4. If not found and `gitch.autoDownload` enabled, download from GitHub Releases

## Architecture

```
src/
  extension.ts      - Entry point, calls ensureBinary on activate
  cli/
    platform.ts     - Platform/arch detection for GoReleaser assets
    binary.ts       - Binary cascade and download logic
    runner.ts       - CLI execution helper
```

## License

MIT
