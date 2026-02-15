# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Added

- **Plugin selection feature** (Phase 2 - IMPLEMENTED)
  - Interactive plugin selection during `ocmgr init`
  - Plugins loaded from `~/.ocmgr/plugins/plugins.toml`
  - Supports selecting multiple plugins by number, 'all', or 'none'
  - New package: `internal/plugins/` for plugin registry loading

- **MCP server selection feature** (Phase 3 - IMPLEMENTED)
  - Interactive MCP server selection during `ocmgr init`
  - MCPs loaded from individual JSON files in `~/.ocmgr/mcps/`
  - Supports npx-based, Docker-based, and remote MCP servers
  - Supports selecting multiple MCPs by number, 'all', or 'none'
  - New package: `internal/mcps/` for MCP registry loading

- **Config generation package** (`internal/configgen/`)
  - Generates `opencode.json` with `$schema`, plugins, and MCPs
  - Merges with existing `opencode.json` if present
  - Supports all MCP config types (local, remote, with OAuth)

- **Sample MCP definition files** in `~/.ocmgr/mcps/`
  - `sequentialthinking.json` - npx-based sequential thinking MCP
  - `context7.json` - npx-based documentation search MCP
  - `docker.json` - Docker-based container management MCP
  - `sentry.json` - Remote Sentry integration MCP
  - `filesystem.json` - npx-based filesystem operations MCP

- **Plugin registry file** (`~/.ocmgr/plugins/plugins.toml`)
  - TOML format with name and description for each plugin
  - Replaces plain text `plugins` file

- **Planning documents**
  - `plugintodo.md` - Detailed implementation plan for plugin selection feature
  - `mcptodo.md` - Detailed implementation plan for MCP server selection feature

- **Self-update command** (`ocmgr update`)
  - Check for and install latest version from GitHub releases
  - Support specific version: `ocmgr update v0.2.0`
  - Detect installation method (curl, homebrew, go install)
  - Safe binary replacement with backup/restore on failure
  - New package: `internal/updater/` for version checking and binary replacement

- **Interactive TUI** (Phase 3 - IMPLEMENTED)
  - `ocmgr` with no arguments launches full-screen TUI
  - Main menu: Init, Profiles, Sync, Snapshot, Config
  - Styled with lipgloss theme (violet/cyan color palette)
  - Keyboard navigation with contextual help bar
  - New package: `internal/tui/` with Charmbracelet Bubble Tea
  - **Profile Browser**: Searchable/filterable profile list with detail view and file tree
  - **Init Wizard**: Select profile → enter target dir → preview files → confirm and copy
  - **Profile Editor**: Browse profile files, open in `$EDITOR`/nvim
  - **Sync UI**: Visual sync status showing in-sync, modified, local-only, remote-only
  - **Snapshot Wizard**: Name → source dir → metadata (description, tags) → preview → create

### Changed

- **internal/copier/copier.go** - Removed opencode.json from copy process
  - `profileFiles` map is now empty
  - `opencode.json` is no longer copied from profiles
  - Updated doc comments to reflect new behavior

- **internal/cli/init.go** - Added interactive prompts
  - Added plugin selection prompt after profile copy
  - Added MCP server selection prompt after plugin selection
  - Uses shared `bufio.Reader` for all prompts to avoid buffering issues
  - Generates `opencode.json` with selected plugins and MCPs

- **USAGE.md** - Comprehensive documentation updates
  - Added Table of Contents entries for all commands
  - Added `ocmgr profile import` command documentation
  - Added `ocmgr profile export` command documentation
  - Added `ocmgr sync push` command documentation
  - Added `ocmgr sync pull` command documentation
  - Added `ocmgr sync status` command documentation
  - Updated "Sharing Profiles" workflow section with full GitHub sync instructions
  - Removed "(Phase 2)" references from config documentation

### Removed

- Template `opencode.json` files from `~/.ocmgr/profiles/base/` and `~/.ocmgr/profiles/macos-base/`
  - No longer needed since `opencode.json` is now generated dynamically

### Infrastructure

- **Repository renamed** from `ocmgr-app` to `ocmgr`
  - Updated `go.mod`, `Makefile`, `install.sh`
  - Updated all GitHub URLs and references
  - Updated GitHub Actions workflow
  - Updated git remote URL

---

## [0.1.0] - 2025-02-12

### Added

- Initial release of ocmgr
- Core profile management commands:
  - `ocmgr init` - Initialize .opencode/ from profile(s)
  - `ocmgr profile list` - List all local profiles
  - `ocmgr profile show` - Show profile details
  - `ocmgr profile create` - Create new empty profile
  - `ocmgr profile delete` - Delete a profile
  - `ocmgr profile import` - Import from local dir or GitHub URL
  - `ocmgr profile export` - Export to a directory
  - `ocmgr snapshot` - Capture .opencode/ as a new profile
  - `ocmgr config show` - Show current configuration
  - `ocmgr config set` - Set a config value
  - `ocmgr config init` - Interactive first-run setup

- GitHub sync commands:
  - `ocmgr sync push` - Push profile to GitHub
  - `ocmgr sync pull` - Pull profile(s) from GitHub
  - `ocmgr sync status` - Show sync status

- Profile features:
  - Profile inheritance via `extends` in profile.toml
  - Selective init with `--only` and `--exclude` flags
  - Conflict resolution: interactive prompt, `--force`, `--merge`
  - Dry-run mode with `--dry-run`
  - Plugin dependency detection (prompts for `bun install`)

- Installation methods:
  - curl installer (`install.sh`)
  - Homebrew formula (`Formula/ocmgr.rb`)
  - Build from source via Makefile

- GitHub Actions workflow for release binaries (`.github/workflows/release.yml`)

---

## Future Roadmap

### Phase 4 - Interactive TUI
- Full TUI built with Charmbracelet (Bubble Tea, Huh, Lip Gloss)
- Profile browser, init wizard, diff viewer, snapshot wizard

### Phase 5 - Advanced Features
- Profile registry
- Template variables
- Pre/post init hooks
- Auto-detect project type
- Shell completions
- `ocmgr doctor` command
