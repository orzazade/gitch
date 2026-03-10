# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.3.0] - 2026-03-11

### Added
- Repo-local profile application by default when `gitch` runs inside a Git repository
- Deterministic SSH profile switching through `core.sshCommand`
- Quiet shell auto-switching on directory change for Bash, Zsh, and Fish
- Repo-local hook installation with managed-hook safety checks
- Repo-aware VS Code extension tests and CI coverage
- Project direction document for product scope and architecture guardrails

### Changed
- Profiles now separate profile id (`name`) from Git author name (`git_name`)
- `gitch status` is read-only by default and only switches when explicitly requested
- Manual switch, auto-switch, hook switch, and editor integration now share one profile application path
- Active-profile detection now requires a full profile match instead of email-only matching
- VS Code integration now resolves status and switching against the current workspace repository

### Fixed
- Inconsistent GPG and SSH behavior between manual and automatic switching paths
- macOS directory rule mismatches caused by symlinked path aliases
- Duplicate-email profile ambiguity in active-profile resolution
- Unsafe global hook takeover of unrelated existing hook setups
- Extension publish pipeline shipping without extension test coverage

## [0.1.0] - TBD

Initial release with core functionality:
- Identity CRUD operations
- SSH key management
- Interactive TUI
- Shell completions

---

[Unreleased]: https://github.com/orzazade/gitch/compare/v2.3.0...HEAD
[2.3.0]: https://github.com/orzazade/gitch/compare/v2.2.0...v2.3.0
[0.1.0]: https://github.com/orzazade/gitch/releases/tag/v0.1.0
