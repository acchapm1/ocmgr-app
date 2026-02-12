# ocmgr — OpenCode Profile Manager

## Vision

A CLI (and eventually TUI) tool written in Go that manages `.opencode` directory contents across projects. Profiles bundle curated sets of agents, commands, skills, and plugins that can be initialized into any project with a single command. Running `ocmgr` with no arguments launches an interactive TUI.

---

## Phase 1: Project Bootstrap & Core (MVP)

> Goal: Working `ocmgr init --profile <name> .` command that copies a profile into `.opencode/`.

### 1.0 — Bootstrap
- [x] Create `install.sh` — detects Go, offers to install or prints instructions and exits
- [x] Initialize Go module (`github.com/acchapm1/ocmgr-app`)
- [x] Set up project directory structure
- [x] Add `.gitignore`, `Makefile`
- [ ] Initialize git repo

### 1.1 — Data Model & Local Store
- [x] Define profile struct with metadata
  - `name`, `description`, `version`, `author`, `tags`
  - `extends` — name of parent profile (for composition, resolved in Phase 2)
- [x] Implement `~/.ocmgr/` local store layout
  ```
  ~/.ocmgr/
  ├── config.toml
  └── profiles/
      └── <name>/
          ├── profile.toml
          ├── agents/          # *.md with YAML frontmatter
          ├── commands/        # *.md with YAML frontmatter
          ├── skills/          # <skill-name>/SKILL.md
          └── plugins/         # *.ts + package.json
  ```
- [x] Profile read/write to local filesystem
- [x] Profile validation (directory structure, required metadata, name sanitization)

### 1.2 — `ocmgr init`
- [x] `ocmgr init --profile <name> [target-dir]` — copy profile into `.opencode/`
- [x] When `.opencode/` exists: prompt user to **overwrite**, **compare**, **merge**, or **cancel**
- [x] Flags: `--force` (overwrite), `--merge`, `--compare`, `--dry-run` (mutually exclusive)
- [x] Support multiple profiles: `ocmgr init --profile base --profile go .`
  - Apply in order, later profiles overlay earlier ones
  - Prompt on file conflicts between profiles
- [x] Plugin dependency handling:
  - Detect `package.json` in profile plugins
  - Prompt: "Install plugin dependencies now? (bun install)" or print install commands
  - Always copy plugin files regardless of answer

### 1.3 — `ocmgr profile`
- [x] `ocmgr profile list` — list all locally available profiles with metadata
- [x] `ocmgr profile show <name>` — display profile contents tree + metadata
- [x] `ocmgr profile create <name>` — scaffold a new empty profile
- [x] `ocmgr profile delete <name>` — remove a local profile (with confirmation)

### 1.4 — `ocmgr snapshot`
- [x] `ocmgr snapshot <name> [source-dir]` — capture `.opencode/` as a new profile
- [x] Auto-categorize files into agents/commands/skills/plugins
- [x] Prompt for metadata (description, tags, etc.)
- [ ] Record `extends` if the source was initialized from a known profile

### 1.5 — `ocmgr config`
- [x] `ocmgr config show` — display current configuration
- [x] `ocmgr config set <key> <value>` — set config values (with validation)
- [x] `ocmgr config init` — first-run setup (GitHub repo, defaults)

### 1.6 — Distribution
- [x] `install.sh` — curl-friendly installer
  - Detect OS/arch
  - Detect Go; offer to install or print instructions
  - Build from source or download pre-built binary
  - Install to `~/.local/bin` or `/usr/local/bin`
- [ ] GitHub Releases with pre-built binaries (via `goreleaser` or Makefile)
- [x] Usage: `curl -sSL https://raw.githubusercontent.com/acchapm1/ocmgr-app/main/install.sh | bash`

---

## Phase 2: GitHub Sync, Composition & Polish

> Goal: Profiles sync to/from GitHub. Profile inheritance works. Selective init. Distribution via Brew/AUR.

### 2.1 — GitHub Sync
- [ ] Single repo layout for all profiles:
  ```
  github.com/<user>/opencode-profiles/
  ├── profiles/
  │   ├── go/
  │   ├── python/
  │   └── ...
  └── README.md
  ```
- [ ] `ocmgr sync push <name>` — push local profile to GitHub repo
- [ ] `ocmgr sync pull <name>` — pull profile from GitHub to local
- [ ] `ocmgr sync pull --all` — pull all remote profiles
- [ ] `ocmgr sync status` — show local vs remote diff
- [ ] Support both public and private repos
- [ ] Multiple auth methods:
  - `gh` CLI token (auto-detect)
  - `GITHUB_TOKEN` / `OCMGR_GITHUB_TOKEN` env var
  - SSH key
  - Interactive token prompt on first use
- [ ] Conflict resolution: prompt on diverged files

### 2.2 — Profile Composition & Layering
- [ ] "base" profiles that others extend via `extends` in `profile.toml`
- [ ] `ocmgr init` resolves dependency chain (e.g., `go` → `base`)
- [ ] Merge strategies: overlay (default), skip-existing, prompt-per-file
- [ ] Circular dependency detection

### 2.3 — Selective Init
- [ ] `ocmgr init --profile go --only agents,skills .`
- [ ] `ocmgr init --profile go --exclude plugins .`

### 2.4 — Additional Distribution
- [ ] Homebrew tap
- [ ] AUR package (yay)
- [ ] `goreleaser` config for cross-platform builds

### 2.5 — Profile Import/Export
- [ ] `ocmgr profile import <path|url>` — import from directory or GitHub URL
- [ ] `ocmgr profile export <name> <path>` — export to directory

---

## Phase 3: TUI (Charmbracelet)

> Goal: `ocmgr` with no arguments launches a full interactive TUI.

### Dependencies
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/huh` — Form/prompt components
- `github.com/charmbracelet/lipgloss` — Styling/layout
- `github.com/charmbracelet/bubbles` — Common UI components (list, table, viewport, etc.)

### 3.1 — TUI Shell
- [ ] `ocmgr` (no args) launches TUI
- [ ] Main menu: Init, Profiles, Sync, Snapshot, Config
- [ ] Styled with lipgloss theme (consistent color palette)
- [ ] Keyboard navigation + help bar

### 3.2 — Profile Browser
- [ ] Searchable/filterable profile list
- [ ] Profile detail view with file tree preview
- [ ] Side-by-side profile comparison

### 3.3 — Init Wizard
- [ ] Select profile(s) from list
- [ ] Select target directory (default: current)
- [ ] Preview changes (diff view)
- [ ] Conflict resolution UI (overwrite/merge/compare/cancel per file)
- [ ] Progress indicator during copy

### 3.4 — Profile Editor
- [ ] Browse profile contents (agents, commands, skills, plugins)
- [ ] Open files in `nvim` for editing
- [ ] Add/remove files from a profile
- [ ] Edit profile metadata via `huh` forms

### 3.5 — Sync UI
- [ ] Visual sync status (local vs remote)
- [ ] Push/pull with progress
- [ ] Diff viewer for conflicts

### 3.6 — Snapshot Wizard
- [ ] Select source directory
- [ ] Preview detected files by category
- [ ] Fill metadata via `huh` form
- [ ] Confirm and save

---

## Phase 4: Advanced Features (Future)

- [ ] Profile versioning (semver in `profile.toml`)
- [ ] Profile registry/discovery website — search and share community profiles
- [ ] Template variables in profiles (e.g., `{{.ProjectName}}`, `{{.Author}}`)
- [ ] Pre/post init hooks (run scripts after profile application)
- [ ] `ocmgr diff <profile> [dir]` — compare profile to current `.opencode/`
- [ ] `ocmgr rollback [dir]` — undo last init (stash previous state)
- [ ] Auto-detect project type and suggest profiles
- [ ] Shell completions (bash, zsh, fish)
- [ ] `ocmgr doctor` — validate current `.opencode/` setup
- [ ] Plugin marketplace / community sharing

---

## Architecture

### Project Layout
```
ocmgr/
├── cmd/
│   └── ocmgr/
│       └── main.go                 # Entry point: CLI or TUI based on args
├── internal/
│   ├── cli/                        # Cobra command definitions
│   │   ├── root.go                 # Root cmd — no args → TUI, with args → CLI
│   │   ├── init.go
│   │   ├── profile.go
│   │   ├── snapshot.go
│   │   ├── sync.go
│   │   └── config.go
│   ├── config/                     # Config loading/saving (~/.ocmgr/config.toml)
│   │   └── config.go
│   ├── profile/                    # Profile data model & operations
│   │   ├── profile.go              # Struct definitions
│   │   ├── loader.go               # Read profiles from disk
│   │   ├── writer.go               # Write profiles to disk
│   │   └── validator.go            # Validate profile structure
│   ├── store/                      # Local store (~/.ocmgr) management
│   │   └── store.go
│   ├── copier/                     # File copy, merge, compare logic
│   │   └── copier.go
│   ├── github/                     # GitHub sync (Phase 2)
│   │   └── sync.go
│   └── tui/                        # Bubble Tea TUI (Phase 3)
│       ├── app.go                  # Main TUI model
│       ├── theme.go                # Lipgloss theme
│       ├── views/
│       │   ├── home.go
│       │   ├── profiles.go
│       │   ├── init_wizard.go
│       │   ├── snapshot.go
│       │   └── sync.go
│       └── components/
│           ├── filelist.go
│           ├── preview.go
│           └── confirm.go
├── install.sh
├── go.mod
├── go.sum
├── Makefile
├── TODO.md
└── .gitignore
```

### Key Libraries
| Library | Phase | Purpose |
|---------|-------|---------|
| `github.com/spf13/cobra` | 1 | CLI framework |
| `github.com/BurntSushi/toml` | 1 | TOML parsing (config.toml, profile.toml) |
| `github.com/charmbracelet/lipgloss` | 1 | Styled CLI output (used early for pretty printing) |
| `github.com/charmbracelet/huh` | 1 | Interactive prompts in CLI (conflict resolution, config init) |
| `github.com/charmbracelet/bubbletea` | 3 | Full TUI framework |
| `github.com/charmbracelet/bubbles` | 3 | TUI components (list, viewport, etc.) |
| `github.com/google/go-github/v60` | 2 | GitHub API client |
| `github.com/go-git/go-git/v5` | 2 | Git operations |

### Config File (`~/.ocmgr/config.toml`)
```toml
[github]
repo = "username/opencode-profiles"    # Single repo for all profiles
auth = "gh"                            # "gh", "env", "ssh", "token"

[defaults]
merge_strategy = "prompt"              # "prompt", "overwrite", "merge", "skip"
editor = "nvim"

[store]
path = "~/.ocmgr/profiles"
```

### Profile Metadata (`profile.toml`)
```toml
[profile]
name = "go"
description = "Go development profile with Go-specific agents, commands, and tooling"
version = "1.0.0"
author = "username"
tags = ["go", "golang", "backend"]
extends = "base"                       # Optional: parent profile name
```

---

## Design Decisions

1. **Conflict handling** — Default is interactive prompt (overwrite/compare/merge/cancel). CLI flags (`--force`, `--merge`, `--dry-run`) for scripting.
2. **Multi-profile layering** — Profiles applied in order; later profiles overlay earlier. File conflicts prompt by default.
3. **Plugin deps** — Always copy plugins. Prompt to install deps; if declined, print the commands needed.
4. **Profile inheritance** — `extends` field in `profile.toml`. Resolved at init time. Full composition in Phase 2.
5. **GitHub layout** — Single repo with all profiles under `profiles/` directory.
6. **Auth** — Support `gh` CLI, env var, SSH, interactive token. Auto-detect best available.
7. **TUI as default** — `ocmgr` (no args) → TUI. `ocmgr <command>` → CLI. Both share the same core logic.
8. **Editor** — `nvim` for editing profile files from TUI.
9. **Distribution** — Phase 1: `install.sh` via curl + GitHub Releases. Phase 2: Homebrew, AUR.
10. **Charmbracelet early** — Use `lipgloss` and `huh` from Phase 1 for styled output and interactive prompts in CLI mode.
