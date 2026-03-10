# Project Direction

`gitch` exists to make multi-profile Git work predictable for developers who switch between repositories such as `appxite` and `scifi`.

## Product Goal

When a repository or folder matches a rule, `gitch` should automatically select the correct profile and keep these settings in sync:

- `git user.name`
- `git user.email`
- GPG signing configuration
- SSH key selection for Git network operations
- Pre-commit validation and prompt display

## Current Architecture Decisions

- A profile has a stable profile id (`name`) and a separate Git author name (`git_name`).
- Rule-based switching defaults to repo-local Git config when running inside a repository.
- SSH selection is deterministic through `core.sshCommand`, not just best-effort `ssh-agent` loading.
- Shell integrations should trigger `gitch autoswitch --quiet` on directory changes.
- Hook installation should default to repo-local hooks and must not overwrite unrelated hook setups.

## Quality Bar

Before calling the tool complete, keep these guarantees true:

- `status` is read-only unless the user explicitly asks to auto-switch.
- Manual switch, auto-switch, editor integration, and hooks all use the same profile application path.
- Tests cover repo-local switching, SSH/GPG application, shell autoswitch generation, and hook installation safety.

## Follow-Up Work

- Improve multi-root VS Code behavior so status and switching can target the active repository instead of the primary workspace fallback.
- Consider an explicit profile scope setting per rule if users want global-only rules outside repositories.
- Evaluate whether SSH host alias generation still adds enough value once `core.sshCommand` is the primary mechanism.
