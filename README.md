# ocmgr

> Manage reusable `.opencode` profiles across all your projects.

**ocmgr** (OpenCode Profile Manager) is a CLI tool that bundles the agents, commands, skills, and plugins from an [OpenCode](https://opencode.ai) `.opencode/` directory into portable profiles. Store them locally, apply them to any project with a single command, and keep every repo configured the way you want.

## Quick Start

```bash
# Install
curl -sSL https://raw.githubusercontent.com/acchapm1/ocmgr-app/main/install.sh | bash

# First-time setup
ocmgr config init

# See available profiles
ocmgr profile list

# Apply a profile to the current project
ocmgr init --profile base .
```

### Snapshot an existing project

Already have a `.opencode/` directory you like? Capture it as a reusable profile:

```bash
ocmgr snapshot my-setup .
```

### Create a new profile from scratch

```bash
ocmgr profile create python-web
# Add files to ~/.ocmgr/profiles/python-web/agents/, commands/, etc.
```

### Layer multiple profiles

Profiles are applied in order. Later profiles overlay earlier ones:

```bash
ocmgr init --profile base --profile go --profile my-overrides .
```

## Installation

### curl installer

Tries a pre-built binary first, falls back to building from source. Detects OS/arch and checks for Go automatically:

```bash
curl -sSL https://raw.githubusercontent.com/acchapm1/ocmgr-app/main/install.sh | bash
```

The binary installs to `~/.local/bin` by default. Override with `INSTALL_DIR`:

```bash
INSTALL_DIR=/usr/local/bin curl -sSL https://raw.githubusercontent.com/acchapm1/ocmgr-app/main/install.sh | bash
```

### Build from source

Requires Go 1.25+:

```bash
git clone https://github.com/acchapm1/ocmgr-app.git
cd ocmgr
make build
# Binary at ./bin/ocmgr
```

Install to your Go bin or `/usr/local/bin`:

```bash
make install
```

## Commands

```
ocmgr                          Show help (TUI coming soon)
ocmgr init [target-dir]        Initialize .opencode/ from profile(s)
ocmgr profile list             List all local profiles
ocmgr profile show <name>      Show profile details and file tree
ocmgr profile create <name>    Scaffold an empty profile
ocmgr profile delete <name>    Delete a profile (with confirmation)
ocmgr snapshot <name> [dir]    Capture .opencode/ as a new profile
ocmgr config show              Show current configuration
ocmgr config set <key> <value> Set a config value
ocmgr config init              Interactive first-run setup
```

### `ocmgr init`

Copies one or more profiles into a project's `.opencode/` directory.

| Flag | Short | Description |
|------|-------|-------------|
| `--profile` | `-p` | Profile name to apply (required, repeatable) |
| `--force` | `-f` | Overwrite all existing files without prompting |
| `--merge` | `-m` | Only copy new files, skip existing ones |
| `--dry-run` | `-d` | Preview what would be copied without writing |

`--force` and `--merge` are mutually exclusive. When neither is set, ocmgr prompts per-file on conflicts:

```
Conflict: agents/code-reviewer.md
  [o]verwrite  [s]kip  [c]ompare  [a]bort
Choice:
```

Choosing `c` shows a colored diff, then re-prompts for a decision.

If the profile contains plugins (`.ts` files), ocmgr detects them after copying and offers to run `bun install`.

## Configuration

Config lives at `~/.ocmgr/config.toml`. Run `ocmgr config init` for interactive setup, or edit directly:

```toml
[github]
repo = "acchapm1/opencode-profiles"    # GitHub owner/repo for sync (Phase 2)
auth = "gh"                            # gh, env, ssh, token

[defaults]
merge_strategy = "prompt"              # prompt, overwrite, merge, skip
editor = "nvim"                        # editor for TUI editing (Phase 3)

[store]
path = "~/.ocmgr/profiles"            # local profile storage directory
```

Set individual values:

```bash
ocmgr config set defaults.merge_strategy overwrite
ocmgr config set github.repo myuser/my-profiles
```

Valid keys: `github.repo`, `github.auth`, `defaults.merge_strategy`, `defaults.editor`, `store.path`.

## Profile Structure

Profiles are stored at `~/.ocmgr/profiles/<name>/`. Each profile is a directory containing:

```
~/.ocmgr/profiles/go/
├── profile.toml          # Metadata
├── agents/               # Markdown files with YAML frontmatter
│   ├── code-reviewer.md
│   └── go-expert.md
├── commands/             # Markdown files defining slash commands
│   └── test.md
├── skills/               # Subdirectories, each with a SKILL.md
│   └── go-patterns/
│       └── SKILL.md
└── plugins/              # TypeScript files using @opencode-ai/plugin
    └── linter.ts
```

### profile.toml

```toml
[profile]
name = "go"
description = "Go development profile with Go-specific agents and tooling"
version = "1.0.0"
author = "acchapm1"
tags = ["go", "golang", "backend"]
extends = "base"    # Optional: parent profile (composition in Phase 2)
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Profile identifier (alphanumeric, hyphens, underscores, dots) |
| `description` | No | Human-readable summary |
| `version` | No | Semver version string |
| `author` | No | Profile creator |
| `tags` | No | Keywords for discovery |
| `extends` | No | Parent profile name (resolved at init time) |

## Project Structure

```
ocmgr/
├── cmd/ocmgr/main.go          # Entry point
├── internal/
│   ├── cli/                    # Cobra command definitions
│   │   ├── root.go             # Root command, subcommand registration
│   │   ├── init.go             # ocmgr init
│   │   ├── profile.go          # ocmgr profile {list,show,create,delete}
│   │   ├── snapshot.go         # ocmgr snapshot
│   │   └── config.go           # ocmgr config {show,set,init}
│   ├── config/                 # Config loading/saving (~/.ocmgr/config.toml)
│   ├── profile/                # Profile data model, validation, scaffolding
│   ├── store/                  # Local store (~/.ocmgr/profiles) management
│   └── copier/                 # File copy, merge, conflict resolution
├── install.sh                  # curl-friendly installer
├── Makefile                    # build, install, test, lint, clean
└── go.mod
```

### Dependencies

| Library | Purpose |
|---------|---------|
| [spf13/cobra](https://github.com/spf13/cobra) | CLI framework |
| [BurntSushi/toml](https://github.com/BurntSushi/toml) | TOML parsing for config and profile metadata |

## Roadmap

**Phase 1 -- CLI (complete)**
Core profile management: init, profile CRUD, snapshot, config, installer.

**Phase 2 -- GitHub Sync (planned)**
`ocmgr sync push/pull` to sync profiles with a GitHub repo. Profile inheritance via `extends`. Selective init (`--only agents`, `--exclude plugins`). Homebrew tap and AUR package.

**Phase 3 -- Interactive TUI (planned)**
`ocmgr` with no arguments launches a full TUI built with [Charmbracelet](https://charm.sh) (Bubble Tea, Huh, Lip Gloss). Profile browser, init wizard, diff viewer, and snapshot wizard.

**Phase 4 -- Advanced (future)**
Profile registry, template variables, pre/post init hooks, auto-detect project type, shell completions, `ocmgr doctor`.
