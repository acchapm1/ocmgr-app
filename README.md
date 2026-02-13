# ocmgr

> Manage reusable `.opencode` profiles across all your projects.

**ocmgr** (OpenCode Profile Manager) is a CLI tool that bundles the agents, commands, skills, and plugins from an [OpenCode](https://opencode.ai) `.opencode/` directory into portable profiles. Store them locally, apply them to any project with a single command, and keep every repo configured the way you want.

## Quick Start

```bash
# Install
curl -sSL https://raw.githubusercontent.com/acchapm1/ocmgr/main/install.sh | bash

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

If a profile has `extends = "base"` in its `profile.toml`, the parent is automatically included — no need to list it explicitly.

### Sync profiles with GitHub

```bash
# Push a profile to your remote repo
ocmgr sync push go

# Pull a profile from the remote
ocmgr sync pull python

# Pull everything
ocmgr sync pull --all

# See what's in sync and what's not
ocmgr sync status
```

### Import and export profiles

```bash
# Import from a local directory
ocmgr profile import /path/to/my-profile

# Import from a GitHub URL
ocmgr profile import https://github.com/user/profiles/tree/main/profiles/go

# Export to a directory
ocmgr profile export go /tmp/backup
```

## Installation

### curl installer

Tries a pre-built binary first, falls back to building from source. Detects OS/arch and checks for Go automatically:

```bash
curl -sSL https://raw.githubusercontent.com/acchapm1/ocmgr/main/install.sh | bash
```

The binary installs to `~/.local/bin` by default. Override with `INSTALL_DIR`:

```bash
INSTALL_DIR=/usr/local/bin curl -sSL https://raw.githubusercontent.com/acchapm1/ocmgr/main/install.sh | bash
```

### Homebrew

```bash
brew tap acchapm1/ocmgr https://github.com/acchapm1/ocmgr
brew install ocmgr
```

### Build from source

Requires Go 1.22+:

```bash
git clone https://github.com/acchapm1/ocmgr.git
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
ocmgr                              Show help (TUI coming soon)
ocmgr init [target-dir]            Initialize .opencode/ from profile(s)
ocmgr profile list                 List all local profiles
ocmgr profile show <name>          Show profile details and file tree
ocmgr profile create <name>        Scaffold an empty profile
ocmgr profile delete <name>        Delete a profile (with confirmation)
ocmgr profile import <source>      Import a profile from dir or GitHub URL
ocmgr profile export <name> <dir>  Export a profile to a directory
ocmgr snapshot <name> [dir]        Capture .opencode/ as a new profile
ocmgr sync push <name>             Push a profile to GitHub
ocmgr sync pull <name>             Pull a profile from GitHub
ocmgr sync pull --all              Pull all remote profiles
ocmgr sync status                  Show local vs remote sync status
ocmgr config show                  Show current configuration
ocmgr config set <key> <value>     Set a config value
ocmgr config init                  Interactive first-run setup
```

### `ocmgr init`

Copies one or more profiles into a project's `.opencode/` directory.

| Flag | Short | Description |
|------|-------|-------------|
| `--profile` | `-p` | Profile name to apply (required, repeatable) |
| `--force` | `-f` | Overwrite all existing files without prompting |
| `--merge` | `-m` | Only copy new files, skip existing ones |
| `--dry-run` | `-d` | Preview what would be copied without writing |
| `--only` | `-o` | Content dirs to include (comma-separated: agents,commands,skills,plugins) |
| `--exclude` | `-e` | Content dirs to exclude (comma-separated: agents,commands,skills,plugins) |

`--force` and `--merge` are mutually exclusive. `--only` and `--exclude` are mutually exclusive. When neither force nor merge is set, ocmgr prompts per-file on conflicts:

```
Conflict: agents/code-reviewer.md
  [o]verwrite  [s]kip  [c]ompare  [a]bort
Choice:
```

Choosing `c` shows a colored diff, then re-prompts for a decision.

**Profile composition:** If a profile's `profile.toml` has `extends = "base"`, ocmgr automatically resolves the dependency chain and applies parent profiles first. Circular dependencies are detected and reported as errors.

**Selective init:** Use `--only` to copy only specific content directories, or `--exclude` to skip them:

```bash
# Only copy agents and skills
ocmgr init --profile go --only agents,skills .

# Copy everything except plugins
ocmgr init --profile go --exclude plugins .
```

If the profile contains plugins (`.ts` files), ocmgr detects them after copying and offers to run `bun install`.

### `ocmgr sync`

Synchronizes profiles between the local store and a remote GitHub repository. The repository and auth method are read from `~/.ocmgr/config.toml`.

The remote repo uses a simple layout:

```
github.com/<user>/opencode-profiles/
├── profiles/
│   ├── base/
│   ├── go/
│   └── python/
└── README.md
```

Auth methods:
- **gh** — uses the `gh` CLI token (default, recommended)
- **env** — reads `OCMGR_GITHUB_TOKEN` or `GITHUB_TOKEN`
- **ssh** — uses SSH key authentication
- **token** — reads from `~/.ocmgr/.token`

## Configuration

Config lives at `~/.ocmgr/config.toml`. Run `ocmgr config init` for interactive setup, or edit directly:

```toml
[github]
repo = "acchapm1/opencode-profiles"    # GitHub owner/repo for sync
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
extends = "base"    # Optional: parent profile resolved at init time
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
│   │   ├── init.go             # ocmgr init (with extends + selective)
│   │   ├── profile.go          # ocmgr profile {list,show,create,delete,import,export}
│   │   ├── snapshot.go         # ocmgr snapshot
│   │   ├── sync.go             # ocmgr sync {push,pull,status}
│   │   └── config.go           # ocmgr config {show,set,init}
│   ├── config/                 # Config loading/saving (~/.ocmgr/config.toml)
│   ├── profile/                # Profile data model, validation, scaffolding
│   ├── store/                  # Local store (~/.ocmgr/profiles) management
│   ├── copier/                 # File copy, merge, conflict resolution
│   ├── resolver/               # Profile extends dependency chain resolution
│   └── github/                 # GitHub sync (auth, clone, push, pull, status)
├── Formula/ocmgr.rb            # Homebrew formula
├── install.sh                  # curl-friendly installer
├── Makefile                    # build, install, test, lint, clean, dist
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

**Phase 2 -- GitHub Sync & Composition (complete)**
`ocmgr sync push/pull/status` to sync profiles with a GitHub repo. Profile inheritance via `extends`. Selective init (`--only`, `--exclude`). Profile import/export. Homebrew formula.

**Phase 3 -- Interactive TUI (planned)**
`ocmgr` with no arguments launches a full TUI built with [Charmbracelet](https://charm.sh) (Bubble Tea, Huh, Lip Gloss). Profile browser, init wizard, diff viewer, and snapshot wizard.

**Phase 4 -- Advanced (future)**
Profile registry, template variables, pre/post init hooks, auto-detect project type, shell completions, `ocmgr doctor`.
