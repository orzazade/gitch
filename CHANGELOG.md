# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Interactive setup wizard (`gitch setup`)
- Identity selector TUI (`gitch use` without args)
- Shell completions for bash, zsh, and fish
- SSH key generation with Ed25519
- SSH key linking to existing keys
- Automatic ssh-agent integration on identity switch
- macOS Keychain support for SSH keys
- Core identity management (add, list, status, use, delete)
- XDG-compliant configuration storage
- Styled terminal output with Lipgloss

### Coming Soon
- Directory-based auto-switching rules
- Remote-based identity matching
- Pre-commit hook integration
- Shell prompt integration

## [0.1.0] - TBD

Initial release with core functionality:
- Identity CRUD operations
- SSH key management
- Interactive TUI
- Shell completions

---

[Unreleased]: https://github.com/orzazade/gitch/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/orzazade/gitch/releases/tag/v0.1.0
