# ocmgr Usage Guide

Detailed usage reference for **ocmgr**, the OpenCode Profile Manager. This guide covers every command, flag, behavior, and workflow. For installation and project overview, see [README.md](./README.md).

---

## Table of Contents

- [Getting Started](#getting-started)
- [Profiles](#profiles)
- [Command Reference](#command-reference)
  - [`ocmgr init`](#ocmgr-init)
  - [`ocmgr profile list`](#ocmgr-profile-list)
  - [`ocmgr profile show`](#ocmgr-profile-show)
  - [`ocmgr profile create`](#ocmgr-profile-create)
  - [`ocmgr profile delete`](#ocmgr-profile-delete)
  - [`ocmgr profile import`](#ocmgr-profile-import)
  - [`ocmgr profile export`](#ocmgr-profile-export)
  - [`ocmgr snapshot`](#ocmgr-snapshot)
  - [`ocmgr sync push`](#ocmgr-sync-push)
  - [`ocmgr sync pull`](#ocmgr-sync-pull)
  - [`ocmgr sync status`](#ocmgr-sync-status)
  - [`ocmgr config show`](#ocmgr-config-show)
  - [`ocmgr config set`](#ocmgr-config-set)
  - [`ocmgr config init`](#ocmgr-config-init)
- [Workflows](#workflows)
- [File Reference](#file-reference)
- [Troubleshooting](#troubleshooting)

---

## Getting Started

### First-Time Setup

After installing `ocmgr`, run the interactive configuration wizard:

```
$ ocmgr config init
GitHub repository (owner/repo) [acchapm1/opencode-profiles]:
Auth method (gh/env/ssh/token) [gh]:
Default merge strategy (prompt/overwrite/merge/skip) [prompt]:
Editor [nvim]:
Configuration saved to ~/.ocmgr/config.toml
```

Press Enter at each prompt to accept the default value shown in brackets. You can change any setting later with `ocmgr config set`.

### The `~/.ocmgr/` Directory

After setup, your home directory contains:

```
~/.ocmgr/
  config.toml          # Global configuration
  profiles/            # All stored profiles
    base/              # Example profile
      profile.toml
      agents/
      commands/
      skills/
      plugins/
    go/                # Another profile
      profile.toml
      ...
```

- **`config.toml`** -- Global settings (GitHub repo, auth method, defaults, store path).
- **`profiles/`** -- Each subdirectory is a self-contained profile. The store path is configurable via `store.path` in `config.toml`.

If `~/.ocmgr/` does not exist, ocmgr creates it automatically on first use.

---

## Profiles

A **profile** is a reusable bundle of configuration files for OpenCode. Profiles are stored as directories under `~/.ocmgr/profiles/` and contain the agents, commands, skills, and plugins that define how an AI coding assistant behaves in a project.

### Profile Directory Structure

```
~/.ocmgr/profiles/<name>/
  profile.toml         # Metadata (name, description, version, tags, etc.)
  agents/              # AI agent definitions (*.md)
  commands/            # Slash command definitions (*.md)
  skills/              # Knowledge base documents
    <skill-name>/
      SKILL.md
  plugins/             # TypeScript plugins (*.ts)
```

### `profile.toml` Format

Every profile has a `profile.toml` file with a `[profile]` table:

```toml
[profile]
name = "go"
description = "Go development profile"
version = "1.0.0"
author = "acchapm1"
tags = ["go", "golang", "backend"]
extends = "base"
```

| Field         | Required | Description                                        |
|---------------|----------|----------------------------------------------------|
| `name`        | Yes      | Short identifier for the profile                   |
| `description` | No       | Human-readable summary                             |
| `version`     | No       | Semver-style version string                        |
| `author`      | No       | Profile creator's identifier                       |
| `tags`        | No       | List of keywords for discovery and categorization  |
| `extends`     | No       | Name of another profile this one inherits from     |

Only `name` is required. All other fields are optional and omitted from display when empty.

### Profile Contents

#### `agents/*.md` -- Agent Definitions

Markdown files with YAML frontmatter that define AI agent personas, capabilities, and instructions. Each `.md` file in the `agents/` directory represents one agent.

#### `commands/*.md` -- Slash Commands

Markdown files that define custom slash commands available in OpenCode. Each `.md` file in the `commands/` directory represents one command.

#### `skills/<name>/SKILL.md` -- Skills

Knowledge base documents organized into subdirectories. Each skill lives in its own directory under `skills/` and must contain a `SKILL.md` file. Skills provide domain-specific instructions and workflows that agents can reference.

#### `plugins/*.ts` -- Plugins

TypeScript files that extend OpenCode functionality using the `@opencode-ai/plugin` SDK. When plugins are present, ocmgr detects them and offers to install dependencies via `bun install`.

### Profile Naming Rules

Profile names must:

- Start with an alphanumeric character (`a-z`, `A-Z`, `0-9`)
- Contain only alphanumeric characters, hyphens (`-`), underscores (`_`), and dots (`.`)
- Not contain path separators (`/`, `\`) or double dots (`..`)

Valid: `base`, `go-backend`, `react_v2`, `my.profile.1`

Invalid: `.hidden`, `-dash-first`, `../escape`, `has/slash`, `has spaces`

The regex used for validation is: `^[a-zA-Z0-9][a-zA-Z0-9._-]*$`

---

## Command Reference

### `ocmgr init`

Initialize a `.opencode/` directory by copying one or more profile contents into a target directory.

#### Syntax

```
ocmgr init [target-dir] [flags]
```

#### Flags

| Flag                   | Short | Type     | Default | Description                                   |
|------------------------|-------|----------|---------|-----------------------------------------------|
| `--profile <name>`     | `-p`  | strings  | (none)  | Profile name(s) to apply (required, repeatable) |
| `--force`              | `-f`  | bool     | false   | Overwrite existing files without prompting     |
| `--merge`              | `-m`  | bool     | false   | Only copy new files, skip existing ones        |
| `--dry-run`            | `-d`  | bool     | false   | Preview changes without writing to disk        |

- `--profile` is **required** and can be specified multiple times to layer profiles.
- `--force` and `--merge` are **mutually exclusive**. Using both produces an error.
- If `target-dir` is omitted, the current working directory (`.`) is used.

#### Behavior

1. **Validates flags** -- Errors immediately if both `--force` and `--merge` are set.
2. **Resolves target** -- Converts `target-dir` to an absolute path and appends `.opencode/`.
3. **Loads profiles** -- All requested profiles are loaded up-front. If any profile is not found, the command fails before copying anything.
4. **Copies files** -- For each profile (in order), walks `agents/`, `commands/`, `skills/`, and `plugins/` and copies files into the target `.opencode/` directory. The `profile.toml` file is never copied.
5. **Reports results** -- Prints a summary of copied, skipped, and errored files per profile.
6. **Detects plugin dependencies** -- If any `.ts` files exist under `.opencode/plugins/`, prompts to run `bun install`.

#### Conflict Resolution

When a file in the profile already exists in the target directory:

| Mode | Behavior |
|------|----------|
| Default (no flags) | Prompts per-file with interactive choices |
| `--force` | Overwrites every conflicting file silently |
| `--merge` | Skips every conflicting file silently |
| `--dry-run` | Reports what would happen without writing anything |

**Interactive prompt (default mode):**

```
Conflict: agents/code-reviewer.md
  [o]verwrite  [s]kip  [c]ompare  [a]bort
Choice:
```

| Choice | Action |
|--------|--------|
| `o` | Overwrite the existing file with the profile version |
| `s` | Keep the existing file, skip the profile version |
| `c` | Show a colored diff between the two files, then re-prompt |
| `a` | Abort the entire init operation immediately |

The compare option (`c`) runs `diff --color=always` between the source and destination files, displays the output, then presents the same prompt again so you can make a final decision.

#### Multi-Profile Layering

When multiple profiles are specified, they are applied left-to-right. Later profiles overlay earlier ones. If two profiles contain the same file (e.g., both have `agents/reviewer.md`), the conflict resolution strategy applies to the second profile's copy attempt against whatever is already in the target directory.

```
$ ocmgr init -p base -p go -p custom .
Applying profile "base" ...
✓ Copied 24 files
Applying profile "go" ...
✓ Copied 3 files
→ Skipped 2 files
Applying profile "custom" ...
✓ Copied 1 files
```

#### Plugin Dependency Detection

After all profiles are applied, if any `.ts` files exist under `.opencode/plugins/`, ocmgr prompts:

```
Plugin dependencies detected. Install now? [y/N]
```

- **`y`** -- Runs `bun install` in the `.opencode/` directory.
- **`N`** (default) -- Prints the command to run later: `cd <path>/.opencode && bun install`
- In `--dry-run` mode, prints `[dry run] Would run: bun install in <path>` instead of actually running it.

#### Examples

**Basic initialization:**

```
$ ocmgr init -p base .
Applying profile "base" ...
✓ Copied 31 files
    agents/code-reviewer.md
    agents/architect.md
    commands/review.md
    ...
```

**Apply to a specific directory:**

```
$ ocmgr init -p base ~/projects/myapp
Applying profile "base" ...
✓ Copied 31 files
```

**Layer multiple profiles:**

```
$ ocmgr init -p base -p go .
Applying profile "base" ...
✓ Copied 31 files
Applying profile "go" ...
✓ Copied 5 files
→ Skipped 2 files
```

**Force overwrite all conflicts:**

```
$ ocmgr init -p base -f .
Applying profile "base" ...
✓ Copied 31 files
```

**Merge (additive only, skip existing):**

```
$ ocmgr init -p go -m .
Applying profile "go" ...
✓ Copied 3 files
→ Skipped 8 files
```

**Dry run to preview changes:**

```
$ ocmgr init -p base -d .
[dry run] Applying profile "base" ...
[dry run] ✓ Copied 31 files
    agents/code-reviewer.md
    agents/architect.md
    ...
```

**Error: mutually exclusive flags:**

```
$ ocmgr init -p base -f -m .
Error: --force and --merge are mutually exclusive
```

**Error: profile not found:**

```
$ ocmgr init -p nonexistent .
Error: profile "nonexistent": profile "nonexistent" not found
```

---

### `ocmgr profile list`

List all profiles in the local store.

#### Syntax

```
ocmgr profile list
```

#### Flags

None.

#### Behavior

Scans every subdirectory under the profiles store path (`~/.ocmgr/profiles/` by default). Directories without a valid `profile.toml` are silently skipped. Results are sorted alphabetically by name.

#### Output

Displays a formatted table with four columns:

| Column      | Description                                    |
|-------------|------------------------------------------------|
| NAME        | Profile name                                   |
| VERSION     | Version string from `profile.toml`             |
| DESCRIPTION | Description, truncated to 42 characters + `...` |
| TAGS        | Comma-separated list of tags                   |

#### Examples

**Profiles exist:**

```
$ ocmgr profile list
NAME    VERSION  DESCRIPTION                                   TAGS
base    1.0.0    Base orchestrator profile with multi-agent...  base, orchestrator, multi-agent, general
go      1.0.0    Go development profile                        go, golang, backend
react   0.2.0    React frontend profile                        react, frontend, typescript
```

**No profiles:**

```
$ ocmgr profile list
No profiles found. Create one with: ocmgr profile create <name>
```

---

### `ocmgr profile show`

Display detailed information about a single profile.

#### Syntax

```
ocmgr profile show <name>
```

#### Arguments

| Argument | Required | Description                |
|----------|----------|----------------------------|
| `name`   | Yes      | Name of the profile to show |

#### Flags

None.

#### Behavior

Loads the profile's `profile.toml` and scans its content directories. Only non-empty metadata fields are displayed. The contents tree shows each directory with a file count and lists every file.

#### Output

```
$ ocmgr profile show base
Profile: base
Description: Base orchestrator profile with multi-agent setup
Version: 1.0.0
Author: acchapm1
Tags: base, orchestrator, multi-agent, general

Contents:
  agents/ (7 files)
    architect.md
    code-reviewer.md
    documentation-writer.md
    orchestrator.md
    planner.md
    researcher.md
    security-auditor.md
  commands/ (12 files)
    architect.md
    deep-research.md
    plan.md
    review.md
    ...
  skills/ (7 skills)
    analyzing-projects/SKILL.md
    designing-apis/SKILL.md
    designing-architecture/SKILL.md
    designing-tests/SKILL.md
    managing-git/SKILL.md
    optimizing-performance/SKILL.md
    parallel-execution/SKILL.md
  plugins/ (5 files)
    context-plugin.ts
    git-plugin.ts
    ...
```

Empty content directories are omitted from the output. If a profile has no `extends` field, that line is not shown.

**Error: profile not found:**

```
$ ocmgr profile show nonexistent
Error: profile "nonexistent" not found
```

---

### `ocmgr profile create`

Create a new empty profile scaffold.

#### Syntax

```
ocmgr profile create <name>
```

#### Arguments

| Argument | Required | Description                     |
|----------|----------|---------------------------------|
| `name`   | Yes      | Name for the new profile        |

#### Flags

None.

#### Behavior

1. Validates the profile name against the naming rules (see [Profile Naming Rules](#profile-naming-rules)).
2. Creates the directory `~/.ocmgr/profiles/<name>/`.
3. Creates four empty subdirectories: `agents/`, `commands/`, `skills/`, `plugins/`.
4. Writes a minimal `profile.toml` containing only the name.

#### Examples

**Create a profile:**

```
$ ocmgr profile create my-project
Created profile 'my-project' at /home/user/.ocmgr/profiles/my-project
Add files to agents/, commands/, skills/, plugins/ directories.
```

**Error: invalid name:**

```
$ ocmgr profile create .bad-name
Error: creating profile: invalid profile name ".bad-name": must be a simple directory name
```

```
$ ocmgr profile create "has spaces"
Error: creating profile: invalid profile name "has spaces": must start with alphanumeric and contain only alphanumeric, hyphens, underscores, or dots
```

---

### `ocmgr profile delete`

Delete a profile from the local store.

#### Syntax

```
ocmgr profile delete <name> [flags]
```

#### Arguments

| Argument | Required | Description                     |
|----------|----------|---------------------------------|
| `name`   | Yes      | Name of the profile to delete   |

#### Flags

| Flag      | Short | Type | Default | Description                  |
|-----------|-------|------|---------|------------------------------|
| `--force` | `-f`  | bool | false   | Skip confirmation prompt     |

#### Behavior

Without `--force`, prompts for confirmation:

```
Delete profile 'my-project'? This cannot be undone. [y/N]
```

- Only `y` or `Y` confirms. Any other input (including Enter) aborts.
- With `--force`, deletes immediately without prompting.
- The entire profile directory is removed recursively.

#### Examples

**Delete with confirmation:**

```
$ ocmgr profile delete my-project
Delete profile 'my-project'? This cannot be undone. [y/N] y
Deleted profile 'my-project'
```

**Abort deletion:**

```
$ ocmgr profile delete my-project
Delete profile 'my-project'? This cannot be undone. [y/N] n
Aborted.
```

**Force delete:**

```
$ ocmgr profile delete my-project -f
Deleted profile 'my-project'
```

**Error: profile not found:**

```
$ ocmgr profile delete nonexistent -f
Error: profile "nonexistent" not found
```

---

### `ocmgr profile import`

Import a profile from a local directory or GitHub URL.

#### Syntax

```
ocmgr profile import <source>
```

#### Arguments

| Argument  | Required | Description                                          |
|-----------|----------|------------------------------------------------------|
| `source`  | Yes      | Local directory path or GitHub URL to import from    |

#### Flags

None.

#### Behavior

1. **Detects source type** — Checks if the source is a GitHub URL or local path.
2. **GitHub URL handling** — If a GitHub URL is provided:
   - Parses the URL to extract owner, repo, branch, and profile path
   - Clones the repository to a temporary directory (shallow clone, depth 1)
   - Extracts the profile from the specified path
3. **Local path handling** — Resolves the path to an absolute path.
4. **Validates profile** — Ensures the source contains a valid `profile.toml`.
5. **Checks for conflicts** — Fails if a profile with the same name already exists.
6. **Copies profile** — Recursively copies all files to the local store.

#### GitHub URL Format

```
https://github.com/<owner>/<repo>/tree/<branch>/<path-to-profile>
```

For example:
```
https://github.com/user/opencode-profiles/tree/main/profiles/go
```

#### Examples

**Import from a local directory:**

```
$ ocmgr profile import /path/to/my-profile
✓ Imported profile "my-profile" to /home/user/.ocmgr/profiles/my-profile
```

**Import from GitHub:**

```
$ ocmgr profile import https://github.com/acchapm1/opencode-profiles/tree/main/profiles/go
✓ Imported profile "go" to /home/user/.ocmgr/profiles/go
```

**Error: invalid GitHub URL:**

```
$ ocmgr profile import https://github.com/user/repo
Error: cannot parse GitHub URL; expected format: https://github.com/<owner>/<repo>/tree/<branch>/<path>
```

**Error: profile already exists:**

```
$ ocmgr profile import /path/to/base
Error: profile "base" already exists; delete it first with 'ocmgr profile delete base'
```

**Error: invalid profile directory:**

```
$ ocmgr profile import /empty/directory
Error: profile.toml not found in /empty/directory
```

---

### `ocmgr profile export`

Export a profile to a local directory.

#### Syntax

```
ocmgr profile export <name> <target-dir>
```

#### Arguments

| Argument     | Required | Description                              |
|--------------|----------|------------------------------------------|
| `name`       | Yes      | Name of the profile to export            |
| `target-dir` | Yes      | Directory to export the profile to       |

#### Flags

None.

#### Behavior

1. Loads the profile from the local store.
2. Resolves the target directory to an absolute path.
3. Creates a subdirectory with the profile name inside the target.
4. Recursively copies all profile files.

#### Examples

**Export to a directory:**

```
$ ocmgr profile export go /tmp/backup
✓ Exported profile "go" to /tmp/backup/go
```

**Export for sharing:**

```
$ ocmgr profile export my-team-profile ~/shared/profiles
✓ Exported profile "my-team-profile" to /home/user/shared/profiles/my-team-profile
```

**Error: profile not found:**

```
$ ocmgr profile export nonexistent /tmp
Error: profile "nonexistent" not found
```

---

### `ocmgr snapshot`

Capture an existing `.opencode/` directory as a new profile.

#### Syntax

```
ocmgr snapshot <name> [source-dir]
```

#### Arguments

| Argument     | Required | Default | Description                                  |
|--------------|----------|---------|----------------------------------------------|
| `name`       | Yes      | --      | Name for the new profile                     |
| `source-dir` | No       | `.`     | Directory containing the `.opencode/` to capture |

#### Flags

None.

#### Behavior

1. Resolves `source-dir` to an absolute path.
2. Verifies that `.opencode/` exists in the source directory.
3. Validates the profile name.
4. Checks that no profile with this name already exists.
5. Creates a new profile scaffold.
6. Walks `agents/`, `commands/`, `skills/`, and `plugins/` inside `.opencode/`, copying files into the new profile.
7. Prompts for a description and tags.
8. Saves the profile metadata.

**Skipped infrastructure files:** The following files and directories are excluded from the snapshot:

- `node_modules/` (entire directory)
- `package.json`
- `bun.lock`
- `.gitignore`

**Cleanup on failure:** If the snapshot fails partway through, the partially created profile directory is automatically removed. No orphaned profiles are left behind.

#### Examples

**Snapshot current directory:**

```
$ ocmgr snapshot my-setup
Description []: My custom AI coding setup
Tags (comma-separated) []: custom, fullstack, react
Snapshot 'my-setup' created with 7 agents, 12 commands, 7 skills, 5 plugins
```

**Snapshot a specific directory:**

```
$ ocmgr snapshot work-config ~/projects/work-app
Description []: Work project configuration
Tags (comma-separated) []:
Snapshot 'work-config' created with 3 agents, 5 commands, 2 skills, 0 plugins
```

**Leave description and tags empty:**

```
$ ocmgr snapshot minimal .
Description []:
Tags (comma-separated) []:
Snapshot 'minimal' created with 2 agents, 4 commands, 0 skills, 0 plugins
```

**Error: no .opencode directory:**

```
$ ocmgr snapshot my-setup ~/empty-dir
Error: no .opencode directory found in /home/user/empty-dir
```

**Error: profile already exists:**

```
$ ocmgr snapshot base .
Error: profile "base" already exists; delete it first with 'ocmgr profile delete base' or choose a different name
```

---

### `ocmgr sync push`

Push a local profile to a GitHub repository.

#### Syntax

```
ocmgr sync push <name>
```

#### Arguments

| Argument | Required | Description                      |
|----------|----------|----------------------------------|
| `name`   | Yes      | Name of the profile to push      |

#### Flags

None.

#### Behavior

1. Loads the profile from the local store.
2. Reads the GitHub repository and auth method from `~/.ocmgr/config.toml`.
3. Clones the remote repository to a temporary directory.
4. Copies the profile into the `profiles/` directory in the repo.
5. Commits and pushes the changes.

#### Prerequisites

- Git must be installed and configured.
- You must have write access to the configured GitHub repository.
- The auth method must be properly configured (see [Configuration](#configuration)).

#### Examples

**Push a profile:**

```
$ ocmgr sync push go
Pushing profile "go" to acchapm1/opencode-profiles …
✓ Pushed profile "go"
```

**Error: profile not found:**

```
$ ocmgr sync push nonexistent
Error: profile "nonexistent" not found
```

**Error: no GitHub configuration:**

```
$ ocmgr sync push go
Error: loading config: github.repo is not configured
```

---

### `ocmgr sync pull`

Pull a profile from a GitHub repository.

#### Syntax

```
ocmgr sync pull [name]
```

#### Arguments

| Argument | Required | Description                              |
|----------|----------|------------------------------------------|
| `name`   | No*      | Name of the profile to pull              |

*Required unless `--all` is specified.

#### Flags

| Flag   | Type | Default | Description                           |
|--------|------|---------|---------------------------------------|
| `--all`| bool | false   | Pull all profiles from the remote     |

#### Behavior

1. Reads the GitHub repository and auth method from `~/.ocmgr/config.toml`.
2. Clones the remote repository to a temporary directory.
3. **Single profile:** Copies the specified profile from `profiles/<name>/` to the local store.
4. **All profiles:** Copies every profile from `profiles/` to the local store.

#### Prerequisites

- Git must be installed.
- You must have read access to the configured GitHub repository.

#### Examples

**Pull a single profile:**

```
$ ocmgr sync pull go
Pulling profile "go" from acchapm1/opencode-profiles …
✓ Pulled profile "go"
```

**Pull all profiles:**

```
$ ocmgr sync pull --all
Pulling all profiles from acchapm1/opencode-profiles …
✓ Pulled 3 profiles:
    base
    go
    python
```

**No profiles found:**

```
$ ocmgr sync pull --all
Pulling all profiles from acchapm1/opencode-profiles …
No profiles found in remote repository.
```

**Error: missing argument:**

```
$ ocmgr sync pull
Error: provide a profile name or use --all
```

---

### `ocmgr sync status`

Show the sync status between local and remote profiles.

#### Syntax

```
ocmgr sync status
```

#### Flags

None.

#### Behavior

1. Reads the GitHub repository and auth method from `~/.ocmgr/config.toml`.
2. Lists all local profiles in the store.
3. Clones the remote repository to a temporary directory.
4. Lists all remote profiles in `profiles/`.
5. Compares local and remote profiles to determine status.

#### Status Indicators

| Status      | Description                                    |
|-------------|------------------------------------------------|
| ✓ in sync   | Profile exists locally and remotely, identical |
| ~ modified  | Profile differs between local and remote       |
| ● local only| Profile exists locally but not remotely        |
| ○ remote only| Profile exists remotely but not locally       |

#### Examples

**Mixed status:**

```
$ ocmgr sync status
Comparing local profiles with acchapm1/opencode-profiles …

PROFILE    STATUS
base       ✓ in sync
go         ~ modified (push or pull to sync)
my-custom  ● local only (push to sync)
python     ○ remote only (pull to sync)
```

**All in sync:**

```
$ ocmgr sync status
Comparing local profiles with acchapm1/opencode-profiles …

PROFILE  STATUS
base     ✓ in sync
go       ✓ in sync
```

**No profiles:**

```
$ ocmgr sync status
Comparing local profiles with acchapm1/opencode-profiles …

No profiles found locally or remotely.
```

---

### `ocmgr config show`

Display all current configuration values.

#### Syntax

```
ocmgr config show
```

#### Flags

None.

#### Output

```
$ ocmgr config show
Configuration (~/.ocmgr/config.toml):

[github]
  repo             = acchapm1/opencode-profiles
  auth             = gh

[defaults]
  merge_strategy   = prompt
  editor           = nvim

[store]
  path             = ~/.ocmgr/profiles
```

If no `config.toml` exists, default values are displayed.

---

### `ocmgr config set`

Set a single configuration value.

#### Syntax

```
ocmgr config set <key> <value>
```

#### Arguments

| Argument | Required | Description              |
|----------|----------|--------------------------|
| `key`    | Yes      | Dot-separated config key |
| `value`  | Yes      | New value to set         |

#### Valid Keys

| Key                       | Valid Values                          | Description                          |
|---------------------------|---------------------------------------|--------------------------------------|
| `github.repo`             | Any string (e.g., `owner/repo`)       | GitHub repository for remote profiles |
| `github.auth`             | `gh`, `env`, `ssh`, `token`           | Authentication method                |
| `defaults.merge_strategy` | `prompt`, `overwrite`, `merge`, `skip` | Default conflict resolution strategy |
| `defaults.editor`         | Any string (e.g., `nvim`, `code`)     | Editor command for file editing      |
| `store.path`              | Any path (`~` is expanded)            | Profile store directory              |

#### Examples

**Set the GitHub repository:**

```
$ ocmgr config set github.repo myorg/my-profiles
Set github.repo = myorg/my-profiles
```

**Change the default merge strategy:**

```
$ ocmgr config set defaults.merge_strategy overwrite
Set defaults.merge_strategy = overwrite
```

**Change the store path:**

```
$ ocmgr config set store.path ~/dotfiles/ocmgr-profiles
Set store.path = ~/dotfiles/ocmgr-profiles
```

**Error: invalid auth method:**

```
$ ocmgr config set github.auth password
Error: invalid auth method "password"; must be one of: gh, env, ssh, token
```

**Error: invalid merge strategy:**

```
$ ocmgr config set defaults.merge_strategy yolo
Error: invalid merge strategy "yolo"; must be one of: prompt, overwrite, merge, skip
```

**Error: unrecognized key:**

```
$ ocmgr config set foo.bar baz
Error: unrecognized key "foo.bar"
Valid keys: github.repo, github.auth, defaults.merge_strategy, defaults.editor, store.path
```

---

### `ocmgr config init`

Interactive first-run configuration setup.

#### Syntax

```
ocmgr config init
```

#### Flags

None.

#### Behavior

Prompts for each configuration value with sensible defaults shown in brackets. Press Enter to accept the default. The store path is always set to `~/.ocmgr/profiles` (not prompted).

```
$ ocmgr config init
GitHub repository (owner/repo) [acchapm1/opencode-profiles]:
Auth method (gh/env/ssh/token) [gh]:
Default merge strategy (prompt/overwrite/merge/skip) [prompt]:
Editor [nvim]:
Configuration saved to ~/.ocmgr/config.toml
```

> Note: `config init` does not validate the auth method or merge strategy values during interactive input. Use valid values to avoid unexpected behavior. You can always correct them later with `ocmgr config set`.

If `~/.ocmgr/config.toml` already exists, it is overwritten with the new values.

---

## Workflows

### Setting Up a New Project

Start from an empty project directory and apply a profile:

```
$ mkdir ~/projects/my-api && cd ~/projects/my-api
$ git init

# Apply the base profile
$ ocmgr init -p base .
Applying profile "base" ...
✓ Copied 31 files

# Verify the result
$ ls .opencode/
agents/  commands/  skills/  plugins/

# Start coding with OpenCode
$ opencode
```

### Capturing Your Current Setup

If you've already customized a `.opencode/` directory and want to reuse it:

```
$ cd ~/projects/existing-project

# Snapshot the current .opencode/ as a profile
$ ocmgr snapshot my-setup .
Description []: My team's standard AI setup
Tags (comma-separated) []: team, fullstack, node
Snapshot 'my-setup' created with 5 agents, 8 commands, 3 skills, 2 plugins

# Verify it was saved
$ ocmgr profile list
NAME       VERSION  DESCRIPTION                  TAGS
my-setup            My team's standard AI setup  team, fullstack, node

# Apply it to another project
$ cd ~/projects/new-project
$ ocmgr init -p my-setup .
```

### Layering Profiles

Use a base profile for common setup and layer language-specific profiles on top:

```
# Apply base first, then Go-specific additions
$ ocmgr init -p base -p go .
Applying profile "base" ...
✓ Copied 31 files
Applying profile "go" ...
✓ Copied 5 files
→ Skipped 2 files
```

Profiles are applied left-to-right. The `go` profile's files are copied on top of `base`. If both profiles contain the same file, the conflict resolution strategy determines what happens (prompt by default, or use `--force`/`--merge`).

**Common layering patterns:**

```
# Base + language
$ ocmgr init -p base -p python .

# Base + language + team customizations
$ ocmgr init -p base -p go -p team-standards .

# Force overwrite for a clean reset
$ ocmgr init -p base -p go -f .
```

### Creating a Custom Profile from Scratch

```
# 1. Create the scaffold
$ ocmgr profile create my-custom
Created profile 'my-custom' at /home/user/.ocmgr/profiles/my-custom
Add files to agents/, commands/, skills/, plugins/ directories.

# 2. Add agent definitions
$ cp my-agent.md ~/.ocmgr/profiles/my-custom/agents/

# 3. Add commands
$ cp my-command.md ~/.ocmgr/profiles/my-custom/commands/

# 4. Add a skill
$ mkdir ~/.ocmgr/profiles/my-custom/skills/my-skill
$ cp SKILL.md ~/.ocmgr/profiles/my-custom/skills/my-skill/

# 5. Edit the profile metadata
$ $EDITOR ~/.ocmgr/profiles/my-custom/profile.toml

# 6. Verify
$ ocmgr profile show my-custom

# 7. Use it
$ ocmgr init -p my-custom .
```

### Sharing Profiles via GitHub Sync

ocmgr can sync profiles with a GitHub repository, making it easy to share configurations across machines and with team members.

#### Setup

1. Create a GitHub repository to store your profiles (e.g., `opencode-profiles`).
2. Configure ocmgr to use it:

```
$ ocmgr config set github.repo your-username/opencode-profiles
$ ocmgr config set github.auth gh
```

The repository should have a `profiles/` directory containing your profiles:

```
github.com/<user>/opencode-profiles/
├── profiles/
│   ├── base/
│   │   ├── profile.toml
│   │   ├── agents/
│   │   └── ...
│   ├── go/
│   └── python/
└── README.md
```

#### Push a Profile

Upload a local profile to the remote repository:

```
$ ocmgr sync push my-custom
Pushing profile "my-custom" to your-username/opencode-profiles …
✓ Pushed profile "my-custom"
```

#### Pull a Profile

Download a profile from the remote repository:

```
$ ocmgr sync pull go
Pulling profile "go" from your-username/opencode-profiles …
✓ Pulled profile "go"
```

Pull all remote profiles at once:

```
$ ocmgr sync pull --all
Pulling all profiles from your-username/opencode-profiles …
✓ Pulled 3 profiles:
    base
    go
    python
```

#### Check Sync Status

See which profiles are in sync, modified, or only exist locally/remote:

```
$ ocmgr sync status
Comparing local profiles with your-username/opencode-profiles …

PROFILE    STATUS
base       ✓ in sync
go         ~ modified (push or pull to sync)
my-custom  ● local only (push to sync)
python     ○ remote only (pull to sync)
```

#### Authentication Methods

| Method | Description |
|--------|-------------|
| `gh` | Uses the GitHub CLI token (default, recommended) |
| `env` | Reads `OCMGR_GITHUB_TOKEN` or `GITHUB_TOKEN` environment variable |
| `ssh` | Uses SSH key authentication |
| `token` | Reads from `~/.ocmgr/.token` file |

---

## File Reference

### `~/.ocmgr/config.toml`

Global configuration file. Created by `ocmgr config init` or automatically with defaults on first use.

```toml
# GitHub repository for remote profile sync.
# Format: "owner/repo"
[github]
  repo = "acchapm1/opencode-profiles"

  # Authentication method for GitHub access.
  # Options: "gh" (GitHub CLI), "env" (environment variable),
  #          "ssh" (SSH key), "token" (personal access token)
  auth = "gh"

# Default behaviors for ocmgr commands.
[defaults]
  # How file conflicts are resolved during `ocmgr init`.
  # Options: "prompt" (ask per-file), "overwrite" (replace all),
  #          "merge" (skip existing), "skip" (same as merge)
  merge_strategy = "prompt"

  # Editor command used when opening files.
  editor = "nvim"

# Local profile store settings.
[store]
  # Directory where profiles are stored.
  # The "~" prefix is expanded to your home directory.
  path = "~/.ocmgr/profiles"
```

If the file does not exist, ocmgr uses these defaults internally without creating the file.

### `~/.ocmgr/profiles/<name>/profile.toml`

Profile metadata file. Every profile directory must contain this file.

```toml
[profile]
  # Required. Short identifier used in commands (e.g., `ocmgr init -p go`).
  name = "go"

  # Optional. Human-readable summary shown in `ocmgr profile list`.
  description = "Go development profile with linting and testing agents"

  # Optional. Semver-style version string.
  version = "1.0.0"

  # Optional. Profile creator's identifier.
  author = "acchapm1"

  # Optional. Keywords for discovery and categorization.
  tags = ["go", "golang", "backend", "testing"]

  # Optional. Name of another profile this one builds upon.
  # Used for documentation purposes; does not auto-inherit files.
  extends = "base"
```

### `.opencode/` Directory Structure

When you run `ocmgr init`, the following structure is created in your project:

```
your-project/
  .opencode/
    agents/                    # AI agent definitions
      orchestrator.md
      code-reviewer.md
      ...
    commands/                  # Slash commands
      review.md
      plan.md
      ...
    skills/                    # Knowledge base
      analyzing-projects/
        SKILL.md
      designing-tests/
        SKILL.md
      ...
    plugins/                   # TypeScript plugins
      context-plugin.ts
      git-plugin.ts
      ...
    node_modules/              # Created by `bun install` (if plugins exist)
    package.json               # Created by `bun install` (if plugins exist)
    bun.lock                   # Created by `bun install` (if plugins exist)
```

Only the `agents/`, `commands/`, `skills/`, and `plugins/` directories come from profiles. Infrastructure files like `node_modules/`, `package.json`, `bun.lock`, and `.gitignore` are not part of profiles and are excluded from snapshots.

---

## Troubleshooting

### "profile not found"

```
Error: profile "myprofile": profile "myprofile" not found
```

**Cause:** The profile name doesn't match any directory in the store.

**Fix:**
1. Check available profiles: `ocmgr profile list`
2. Verify the exact name (names are case-sensitive)
3. Check your store path: `ocmgr config show` -- look at `store.path`

---

### "already exists" on snapshot

```
Error: profile "base" already exists; delete it first with 'ocmgr profile delete base' or choose a different name
```

**Cause:** A profile with that name is already in the store.

**Fix:**
- Delete the existing profile first: `ocmgr profile delete base`
- Or choose a different name: `ocmgr snapshot base-v2 .`

---

### "--force and --merge are mutually exclusive"

```
Error: --force and --merge are mutually exclusive
```

**Cause:** Both `--force` and `--merge` were passed to `ocmgr init`.

**Fix:** Use only one:
- `--force` to overwrite all conflicting files
- `--merge` to skip all conflicting files

---

### Invalid profile name

```
Error: creating profile: invalid profile name ".bad": must be a simple directory name
```

or

```
Error: creating profile: invalid profile name "my profile": must start with alphanumeric and contain only alphanumeric, hyphens, underscores, or dots
```

**Cause:** The profile name contains invalid characters.

**Fix:** Use a name that starts with a letter or number and contains only letters, numbers, hyphens, underscores, and dots. See [Profile Naming Rules](#profile-naming-rules).

---

### Plugin dependencies prompt

```
Plugin dependencies detected. Install now? [y/N]
```

**If bun is installed:** Press `y` to install dependencies automatically.

**If bun is not installed:** Press `N` (or Enter), then install bun first:
```
$ curl -fsSL https://bun.sh/install | bash
```
Then install dependencies manually:
```
$ cd .opencode && bun install
```

---

### Configuration not taking effect

**Fix:** Verify current values with:
```
$ ocmgr config show
```

Check that you're setting the right key:
```
$ ocmgr config set defaults.merge_strategy overwrite
```

> Note: The `defaults.merge_strategy` config value is reserved for future use. Currently, the merge strategy is determined solely by the `--force` and `--merge` flags passed to `ocmgr init`. The default behavior without flags is always interactive prompting.

---

### "no .opencode directory found"

```
Error: no .opencode directory found in /home/user/myproject
```

**Cause:** The source directory for `ocmgr snapshot` does not contain a `.opencode/` subdirectory.

**Fix:**
- Verify you're in the right directory: `ls -la .opencode/`
- Specify the correct source: `ocmgr snapshot my-setup ~/correct/project`

---

### Version

Check your installed version:

```
$ ocmgr --version
```

If the version shows `dev`, you're running a development build without version injection via ldflags.
