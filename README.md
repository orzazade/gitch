<div align="center">

<img src="https://raw.githubusercontent.com/orzazade/gitch/main/.github/assets/logo.png" alt="gitch logo" width="120" />

# gitch

### Never commit with the wrong git identity again.

[![CI](https://github.com/orzazade/gitch/actions/workflows/ci.yml/badge.svg)](https://github.com/orzazade/gitch/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/orzazade/gitch?color=success)](https://github.com/orzazade/gitch/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/orzazade/gitch?color=00ADD8)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/orzazade/gitch)](https://goreportcard.com/report/github.com/orzazade/gitch)
[![License](https://img.shields.io/github/license/orzazade/gitch?color=blue)](LICENSE)

**A beautiful CLI for managing multiple git identities with SSH keys, GPG signing, auto-switching rules, and shell prompt integration.**

[Installation](#-installation) · [Quick Start](#-quick-start) · [Features](#-features) · [Commands](#-commands) · [Contributing](#-contributing)

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
│   ✓ GPG signing configured                                  │
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
Generate new SSH keys per identity or link existing ones. Keys auto-load into ssh-agent on switch. Choose key type (Ed25519 or RSA) with smart defaults for Azure DevOps.

### ✍️ GPG Commit Signing
Generate GPG keys or link existing ones. Git commit signing auto-configures on identity switch.

### 🎨 Beautiful TUI
Interactive setup wizard and identity selector built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Terminal UI that sparks joy.

</td>
<td width="50%">

### ⚡ Auto-Switching Rules
Define directory or remote-based rules. Enter `~/work/**` → automatically switch to work identity.

### 🛡️ Pre-Commit Protection
Install hooks that prevent wrong-identity commits. Configure per-identity: warn, block, or allow.

### 🐚 Shell Prompt Integration
See your current identity in your prompt. Ultra-fast (<5ms) cache-based updates for Bash, Zsh, and Fish.

</td>
</tr>
</table>

<table>
<tr>
<td width="50%">

### 🐚 Shell Completions
First-class tab completion for Bash, Zsh, and Fish. Never type a full command again.

</td>
<td width="50%">

### 🔒 Secure by Default
SSH keys stored in `~/.ssh/` with proper permissions. GPG keys in system keyring. No credentials in plain text.

</td>
</tr>
</table>

<table>
<tr>
<td width="50%">

### 🔍 Commit History Audit
Scan your repo for commits made with the wrong identity. See which commits are local-only vs already pushed.

</td>
<td width="50%">

### 🔧 History Rewriting
Fix mismatched commits with `gitch audit --fix`. Automatic backup, GPG warnings, typed confirmation, and remote removal for safety.

</td>
</tr>
</table>

<br/>

## 📦 Installation

### macOS (Homebrew)

```bash
brew install orzazade/tap/gitch
```

### Windows

```powershell
# Scoop
scoop bucket add gitch https://github.com/orzazade/scoop-bucket
scoop install gitch

# Chocolatey (coming soon)
# choco install gitch

# winget (coming soon)
# winget install orzazade.gitch

# Or download directly from GitHub Releases
```

### Linux

```bash
# Debian/Ubuntu (APT)
curl -fsSL https://orzazade.github.io/apt-repo/pubkey.gpg | sudo gpg --dearmor -o /etc/apt/keyrings/gitch.gpg
echo "deb [signed-by=/etc/apt/keyrings/gitch.gpg] https://orzazade.github.io/apt-repo stable main" | sudo tee /etc/apt/sources.list.d/gitch.list
sudo apt update && sudo apt install gitch

# Or download .deb/.rpm directly from GitHub Releases
```

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

# Create identity with GPG signing enabled
gitch add --name "opensource" --email "you@github.com" --generate-gpg

# Or use an existing GPG key
gitch add --name "secure" --email "you@secure.com" --gpg-key ABCD1234EFGH5678

# For Azure DevOps, use RSA key type (auto-detected in repos)
gitch add --name "azure" --email "you@company.com" --generate-ssh --key-type rsa

# Switch between them
gitch use work

# Or use the interactive selector
gitch use
```

<br/>

## 📖 Commands

### Core Commands

| Command | Description |
|:--------|:------------|
| `gitch setup` | 🧙 Interactive setup wizard |
| `gitch add` | ➕ Create a new identity (with `--generate-ssh`, `--generate-gpg` options) |
| `gitch list` | 📋 List all identities |
| `gitch status` | 👁️ Show current active identity (`-v` for rule details) |
| `gitch use [name]` | 🔀 Switch to an identity (interactive if no name) |
| `gitch delete <name>` | 🗑️ Delete an identity |

### Auto-Switching & Hooks

| Command | Description |
|:--------|:------------|
| `gitch rule add <pattern> --use <identity>` | 📍 Add directory rule (e.g., `~/work/**`) |
| `gitch rule add --remote <pattern> --use <identity>` | 🌐 Add remote rule (e.g., `github.com/company/*`) |
| `gitch rule list` | 📋 List all switching rules |
| `gitch rule remove <pattern>` | 🗑️ Remove a rule |
| `gitch hook install` | 🛡️ Install pre-commit hook globally |
| `gitch hook uninstall` | ❌ Remove pre-commit hook |
| `gitch config hook-mode <identity> <mode>` | ⚙️ Set hook behavior (warn/block/allow) |

### Audit & History

| Command | Description |
|:--------|:------------|
| `gitch audit` | 🔍 Scan repo for commits with wrong identity |
| `gitch audit --fix` | 🔧 Rewrite mismatched commits (with backup + confirmation) |

### Shell Integration

| Command | Description |
|:--------|:------------|
| `gitch init <shell>` | 🐚 Output shell prompt integration code (bash/zsh/fish) |
| `gitch completion <shell>` | 📝 Generate shell completions |

<br/>

## 📍 Auto-Switching Rules

Set up rules to automatically switch identities based on directory or remote:

```bash
# Switch to "work" when in any subdirectory of ~/work
gitch rule add ~/work/** --use work

# Switch to "opensource" for any github.com/orzazade/* repo
gitch rule add --remote "github.com/orzazade/*" --use opensource

# View all rules
gitch rule list

# Remove a rule
gitch rule remove ~/work/**
```

<br/>

## 🛡️ Pre-Commit Hooks

Prevent accidental commits with the wrong identity:

```bash
# Install the pre-commit hook globally
gitch hook install

# When you commit with wrong identity, you'll see:
#   ⚠ Identity mismatch: expected "work", but current is "personal"
#   [S]witch to work / [C]ontinue anyway / [A]bort

# Configure per-identity behavior
gitch config hook-mode work block    # Always block wrong identity
gitch config hook-mode personal warn # Just warn (default)
gitch config hook-mode oss allow     # No checks for this identity

# Bypass when needed
GITCH_BYPASS=1 git commit -m "emergency fix"
```

<br/>

## 🐚 Shell Prompt Integration

See your current git identity right in your prompt:

```bash
# Add to your shell config:
eval "$(gitch init zsh)"   # For Zsh (~/.zshrc)
eval "$(gitch init bash)"  # For Bash (~/.bashrc)
source (gitch init fish)   # For Fish (~/.config/fish/config.fish)

# Your prompt will show:
# [work] ~/projects/company $
```

<br/>

## 📝 Shell Completions

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

GPG keys are generated and imported into your system GPG keyring (`~/.gnupg/`).

<br/>

## 🗺️ Roadmap

### v1.x - Core Features (Complete)

| Phase | Features |
|:------|:---------|
| ✅ **Foundation** | Core identity management (add, list, status, use, delete) |
| ✅ **SSH Integration** | SSH key generation and ssh-agent integration |
| ✅ **TUI Experience** | Interactive setup wizard, identity selector, shell completions |
| ✅ **Auto-Switching** | Directory/remote rules, pre-commit hooks, bypass support |
| ✅ **Shell Prompt** | Fast prompt integration for Bash, Zsh, and Fish |
| ✅ **Distribution** | Homebrew tap for easy macOS installation |
| ✅ **GPG Signing** | GPG key generation, linking, and auto-configure on switch |

### v2.0 - Security & Distribution (Complete)

| Phase | Features |
|:------|:---------|
| ✅ **Import/Export** | YAML export/import for identity backup and migration |
| ✅ **SSH Config** | Auto-generate SSH config Host aliases per identity |
| ✅ **Encrypted Backup** | Age-encrypted SSH key export/import |
| ✅ **Cross-Platform** | Windows (Scoop, Chocolatey, winget) and Linux (APT) |

### v2.1 - SSH Flexibility & Audit (Complete)

| Phase | Features |
|:------|:---------|
| ✅ **SSH Key Types** | Choose Ed25519 or RSA, auto-detect Azure DevOps |
| ✅ **Commit Audit** | Scan history for wrong-identity commits |
| ✅ **History Rewrite** | Fix mismatched commits with safety guardrails |

### Future

| Phase | Features |
|:------|:---------|
| 🔮 **VS Code Extension** | Show identity in status bar, quick switcher |
| 🔮 **Statistics** | Commits per identity, usage patterns |
| 🔮 **Identity Templates** | Preset patterns for common setups |

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
