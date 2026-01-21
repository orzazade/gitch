<div align="center">

# 🔀 gitch

### Never commit with the wrong git identity again.

[![CI](https://github.com/orzazade/gitch/actions/workflows/ci.yml/badge.svg)](https://github.com/orzazade/gitch/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/orzazade/gitch?color=success)](https://github.com/orzazade/gitch/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/orzazade/gitch?color=00ADD8)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/orzazade/gitch)](https://goreportcard.com/report/github.com/orzazade/gitch)
[![License](https://img.shields.io/github/license/orzazade/gitch?color=blue)](LICENSE)

**A beautiful CLI for managing multiple git identities with SSH keys, interactive TUI, and auto-switching.**

[Installation](#-installation) · [Quick Start](#-quick-start) · [Commands](#-commands) · [Contributing](#-contributing)

---

</div>

## The Problem

You work on multiple projects:
- **Work** — commits should use `you@company.com`
- **Personal** — commits should use `you@gmail.com`
- **Open Source** — commits should use `you@users.noreply.github.com`

You accidentally commit with the wrong email. Your work repo now has personal commits. Your contribution graph is broken. Sound familiar?

## The Solution

**gitch** manages your git identities so you don't have to think about it.

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│   $ gitch use                                               │
│                                                             │
│   ┌─────────────────────────────────────────────────────┐   │
│   │  Select an identity                                 │   │
│   │                                                     │   │
│   │  > 🏢 work                                          │   │
│   │      you@company.com                                │   │
│   │                                                     │   │
│   │    🏠 personal                                      │   │
│   │      you@gmail.com                                  │   │
│   │                                                     │   │
│   │    🌐 opensource                                    │   │
│   │      you@users.noreply.github.com                   │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                             │
│   ✓ Switched to "work" identity                             │
│   ✓ SSH key loaded into agent                               │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

<br/>

## ✨ Features

<table>
<tr>
<td width="50%">

### 🎭 Identity Management
Create, switch, and manage multiple git identities. Each identity stores name, email, and optional SSH key.

### 🔐 SSH Key Integration
Generate new SSH keys per identity or link existing ones. Keys auto-load into ssh-agent on switch.

### 🎨 Beautiful TUI
Interactive setup wizard and identity selector built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Terminal UI that sparks joy.

</td>
<td width="50%">

### ⚡ Instant Switching
Switch identities in milliseconds. Git config updates globally, SSH key loads automatically.

### 🐚 Shell Completions
First-class tab completion for Bash, Zsh, and Fish. Never type a full command again.

### 🔒 Secure by Default
SSH keys stored in `~/.ssh/` with proper permissions. No credentials in plain text.

</td>
</tr>
</table>

<br/>

## 📦 Installation

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

### Homebrew <sup><sub>coming soon</sub></sup>

```bash
brew install orzazade/tap/gitch
```

<br/>

## 🚀 Quick Start

### Option 1: Interactive Setup (Recommended)

```bash
gitch setup
```

This launches a beautiful wizard that guides you through creating your first identity.

### Option 2: Manual Setup

```bash
# Create your first identity with a new SSH key
gitch add --name "work" --email "you@company.com" --generate-ssh

# Create another with an existing SSH key
gitch add --name "personal" --email "you@gmail.com" --ssh-key ~/.ssh/id_personal

# Switch between them
gitch use work

# Or use the interactive selector
gitch use
```

<br/>

## 📖 Commands

| Command | Description |
|:--------|:------------|
| `gitch setup` | 🧙 Interactive setup wizard |
| `gitch add` | ➕ Create a new identity |
| `gitch list` | 📋 List all identities |
| `gitch status` | 👁️ Show current active identity |
| `gitch use [name]` | 🔀 Switch to an identity (interactive if no name) |
| `gitch delete <name>` | 🗑️ Delete an identity |
| `gitch completion <shell>` | 🐚 Generate shell completions |

<br/>

## 🐚 Shell Completions

Enable tab completion for your shell:

<details>
<summary><b>Bash</b></summary>

```bash
# Add to ~/.bashrc
source <(gitch completion bash)
```
</details>

<details>
<summary><b>Zsh</b></summary>

```bash
# Add to ~/.zshrc (before compinit)
source <(gitch completion zsh)
```
</details>

<details>
<summary><b>Fish</b></summary>

```bash
gitch completion fish > ~/.config/fish/completions/gitch.fish
```
</details>

<br/>

## ⚙️ Configuration

gitch stores configuration in the XDG config directory:

| Platform | Location |
|:---------|:---------|
| **Linux/macOS** | `~/.config/gitch/config.yaml` |
| **Windows** | `%APPDATA%\gitch\config.yaml` |

SSH keys are stored in `~/.ssh/` with the naming convention `gitch_<identity-name>`.

<br/>

## 🗺️ Roadmap

| Phase | Status | Features |
|:------|:------:|:---------|
| **Phase 1** | ✅ | Core identity management (add, list, status, use, delete) |
| **Phase 2** | ✅ | SSH key generation and ssh-agent integration |
| **Phase 3** | ✅ | Interactive TUI (setup wizard, identity selector, completions) |
| **Phase 4** | 🚧 | Auto-switching rules and pre-commit hooks |
| **Phase 5** | 📋 | Shell prompt integration |

<br/>

## 🤝 Contributing

Contributions are welcome! Whether it's bug reports, feature requests, or pull requests.

- 🐛 [Report bugs](https://github.com/orzazade/gitch/issues)
- 💡 [Request features](https://github.com/orzazade/gitch/issues)
- 🔧 [Submit PRs](https://github.com/orzazade/gitch/pulls)

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

<br/>

## 🏗️ Built With

<p>
<a href="https://github.com/spf13/cobra"><img src="https://img.shields.io/badge/Cobra-CLI_Framework-blue?style=flat-square" alt="Cobra"/></a>
<a href="https://github.com/spf13/viper"><img src="https://img.shields.io/badge/Viper-Configuration-green?style=flat-square" alt="Viper"/></a>
<a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/Bubble_Tea-TUI-ff69b4?style=flat-square" alt="Bubble Tea"/></a>
<a href="https://github.com/charmbracelet/lipgloss"><img src="https://img.shields.io/badge/Lipgloss-Styling-purple?style=flat-square" alt="Lipgloss"/></a>
</p>

<br/>

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

---

<div align="center">

**Stop context-switching. Start committing with confidence.**

<br/>

Made with ❤️ by [Orkhan Rzazade](https://github.com/orzazade)

<br/>

<a href="https://github.com/orzazade/gitch/stargazers">⭐ Star this repo</a> ·
<a href="https://github.com/orzazade/gitch/issues">🐛 Report Bug</a> ·
<a href="https://github.com/orzazade/gitch/issues">💡 Request Feature</a>

</div>
