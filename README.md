# gitch

**Never commit with the wrong git identity again.**

gitch is a git identity manager CLI that helps developers manage multiple git identities (work, personal, open source) without confusion. It provides automatic identity switching, SSH key management, and pre-commit hooks to prevent accidental commits with the wrong identity.

[![CI](https://github.com/orzazade/gitch/actions/workflows/ci.yml/badge.svg)](https://github.com/orzazade/gitch/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/orzazade/gitch)](https://github.com/orzazade/gitch/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/orzazade/gitch)](https://go.dev/)
[![License](https://img.shields.io/github/license/orzazade/gitch)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/orzazade/gitch)](https://goreportcard.com/report/github.com/orzazade/gitch)

## Features

- **Identity Management** - Create, list, switch, and delete git identities
- **SSH Key Integration** - Generate or link SSH keys per identity, auto-load on switch
- **Interactive TUI** - Setup wizard and identity selector with beautiful terminal UI
- **Shell Completions** - Tab completion for bash, zsh, and fish
- **Prevention** (coming soon) - Pre-commit hooks to catch wrong-identity commits
- **Auto-Switching** (coming soon) - Directory and remote-based identity rules

## Installation

### Using Go

```bash
go install github.com/orzazade/gitch@latest
```

### From Source

```bash
git clone https://github.com/orzazade/gitch.git
cd gitch
make build
```

### Homebrew (coming soon)

```bash
brew install orzazade/tap/gitch
```

## Quick Start

### Interactive Setup

The easiest way to get started:

```bash
gitch setup
```

This launches an interactive wizard that guides you through creating your first identity.

### Manual Setup

```bash
# Create identities
gitch add --name "work" --email "you@company.com" --generate-ssh
gitch add --name "personal" --email "you@gmail.com" --ssh-key ~/.ssh/id_personal

# List identities
gitch list

# Switch identity
gitch use work          # by name
gitch use               # interactive selector

# Check current identity
gitch status
```

## Commands

| Command | Description |
|---------|-------------|
| `gitch setup` | Interactive setup wizard |
| `gitch add` | Create a new identity |
| `gitch list` | List all identities |
| `gitch status` | Show current active identity |
| `gitch use [name]` | Switch to an identity |
| `gitch delete <name>` | Delete an identity |
| `gitch completion <shell>` | Generate shell completions |

### Shell Completions

Enable tab completion for your shell:

**Bash:**
```bash
# Add to ~/.bashrc
source <(gitch completion bash)
```

**Zsh:**
```bash
# Add to ~/.zshrc (before compinit)
source <(gitch completion zsh)
```

**Fish:**
```bash
gitch completion fish > ~/.config/fish/completions/gitch.fish
```

## Configuration

gitch stores configuration in the XDG config directory:

- **Linux/macOS:** `~/.config/gitch/config.yaml`
- **Windows:** `%APPDATA%\gitch\config.yaml`

SSH keys are stored in `~/.ssh/` with the naming convention `gitch_<identity-name>`.

## Roadmap

- [x] **Phase 1:** Core identity management (add, list, status, use, delete)
- [x] **Phase 2:** SSH key generation and ssh-agent integration
- [x] **Phase 3:** Interactive TUI (setup wizard, identity selector, completions)
- [ ] **Phase 4:** Auto-switching rules and pre-commit hooks
- [ ] **Phase 5:** Shell prompt integration

See [ROADMAP.md](.planning/ROADMAP.md) for detailed plans.

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

- Report bugs via [GitHub Issues](https://github.com/orzazade/gitch/issues)
- Submit features via [Pull Requests](https://github.com/orzazade/gitch/pulls)
- Join discussions in [GitHub Discussions](https://github.com/orzazade/gitch/discussions)

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

Built with these excellent libraries:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling

---

**Made with love by [Orkhan Rzazade](https://github.com/orzazade)**
