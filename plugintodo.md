# Plugin Selection Feature - Implementation Plan

## Overview

Add interactive plugin selection during `ocmgr init`. Users will be prompted to select plugins from a curated list in `~/.ocmgr/plugins/`, and the selected plugins will be written to the generated `opencode.json` in the target project.

---

## Phase 1: Remove opencode.json from Copy Process

### 1.1 Update Copier

**File:** `internal/copier/copier.go`

- [ ] Remove `"opencode.json": true` from the `profileFiles` map
- [ ] Update the doc comment on `profileFiles` to reflect it's now empty (or remove the map entirely if no other files need it)

### 1.2 Clean Up Template Files

- [ ] Delete `~/.ocmgr/profiles/base/opencode.json` (template no longer needed)
- [ ] Delete `~/.ocmgr/profiles/macos-base/opencode.json` (template no longer needed)

### 1.3 Verification

- [ ] Run `ocmgr init -p base .` and verify `opencode.json` is NOT copied
- [ ] Verify other files (agents, commands, skills, plugins dirs) still copy correctly

---

## Phase 2: Implement Plugin Selection Prompt

### 2.1 Define Plugin List File Format

**Recommended: TOML format** (`~/.ocmgr/plugins/plugins.toml`)

```toml
# ~/.ocmgr/plugins/plugins.toml
# Plugin registry for ocmgr

[[plugin]]
name = "@plannotator/opencode@latest"
description = "AI-powered code annotation and documentation"

[[plugin]]
name = "@franlol/opencode-md-table-formatter@0.0.3"
description = "Format markdown tables automatically"

[[plugin]]
name = "@zenobius/opencode-skillful"
description = "Enhanced skill management for OpenCode"

[[plugin]]
name = "@knikolov/opencode-plugin-simple-memory"
description = "Simple memory/context persistence across sessions"
```

**Why TOML over plain text:**
- Human-readable and easy to edit
- Supports metadata (descriptions) for better UX during selection
- Can be extended with additional fields (category, tags, etc.)
- Standard format with good Go library support (already used for config.toml)

**Alternative considered:** JSON
- More verbose
- Harder to edit manually
- No significant advantages for this use case

### 2.2 Create Plugin Package

**New directory:** `internal/plugins/`

```
internal/plugins/
├── plugins.go        # Plugin list loading and parsing
└── plugins_test.go   # Unit tests
```

**File:** `internal/plugins/plugins.go`

```go
// Package plugins handles loading and managing the plugin registry.
package plugins

// Plugin represents an available plugin from the registry.
type Plugin struct {
    Name        string `toml:"name"`
    Description string `toml:"description"`
}

// Registry holds all available plugins.
type Registry struct {
    Plugins []Plugin `toml:"plugin"`
}

// Load reads the plugin registry from ~/.ocmgr/plugins/plugins.toml
func Load() (*Registry, error)

// List returns all available plugins.
func (r *Registry) List() []Plugin

// GetByName finds a plugin by its npm package name.
func (r *Registry) GetByName(name string) *Plugin
```

### 2.3 Update Init Command

**File:** `internal/cli/init.go`

Add after profile copy completes:

```go
// After all profiles are applied, prompt for plugins
if !dryRun {
    if err := promptForPlugins(targetOpencode); err != nil {
        return fmt.Errorf("plugin selection: %w", err)
    }
}
```

**New function:** `promptForPlugins(targetDir string) error`

1. Check if `~/.ocmgr/plugins/plugins.toml` exists
2. If not, skip silently (no plugins configured)
3. Load the plugin registry
4. Display interactive selection prompt:
   ```
   Would you like to add plugins to this project? [y/N]
   ```
5. If yes, show numbered list:
   ```
   Available plugins:
     1. @plannotator/opencode@latest
            AI-powered code annotation and documentation
     2. @franlol/opencode-md-table-formatter@0.0.3
            Format markdown tables automatically
     3. @zenobius/opencode-skillful
            Enhanced skill management for OpenCode
     4. @knikolov/opencode-plugin-simple-memory
            Simple memory/context persistence across sessions
   
   Select plugins (comma-separated numbers, or 'all'): 
   ```
6. Parse selection and generate/update `opencode.json`

### 2.4 Generate opencode.json

**File:** `internal/plugins/plugins.go`

```go
// Config represents the opencode.json structure.
type Config struct {
    Schema string   `json:"$schema,omitempty"`
    Plugin []string `json:"plugin,omitempty"`
    MCP    map[string]MCPConfig `json:"mcp,omitempty"`
}

// WriteConfig generates opencode.json with selected plugins.
func WriteConfig(targetDir string, plugins []string) error
```

**Generated file format:**

```json
{
  "$schema": "https://opencode.ai/config.json",
  "plugin": [
    "@plannotator/opencode@latest",
    "@zenobius/opencode-skillful"
  ]
}
```

### 2.5 Handle Existing opencode.json

If `opencode.json` already exists in target:
- [ ] Read existing config
- [ ] Merge new plugins with existing (deduplicate)
- [ ] Preserve any existing MCP configurations
- [ ] Write merged config back

### 2.6 Update Documentation

**Files to update:**
- [ ] `README.md` - Add plugin selection to init command description
- [ ] `USAGE.md` - Add detailed plugin selection workflow

---

## Phase 3: Testing & Edge Cases

### 3.1 Unit Tests

- [ ] Test TOML parsing of plugins file
- [ ] Test opencode.json generation
- [ ] Test merging with existing opencode.json
- [ ] Test handling of missing plugins file

### 3.2 Integration Tests

- [ ] Test `ocmgr init` with plugin selection
- [ ] Test `ocmgr init --dry-run` shows plugin prompt preview
- [ ] Test plugin selection with `--force` flag (auto-select all?)
- [ ] Test plugin selection with `--merge` flag

### 3.3 Edge Cases

- [ ] Empty plugins file → skip prompt
- [ ] Invalid TOML → show error, continue without plugins
- [ ] User enters invalid selection → re-prompt
- [ ] User selects no plugins → create minimal opencode.json with just `$schema`
- [ ] Non-interactive terminal → skip prompt or use default selection

---

## File Structure Summary

```
~/.ocmgr/
├── plugins/
│   └── plugins.toml          # Plugin registry (NEW)
├── profiles/
│   └── base/
│       ├── agents/
│       ├── commands/
│       ├── skills/
│       └── plugins/          # Plugin source files (unchanged)
└── config.toml

ocmgr/
├── internal/
│   ├── cli/
│   │   └── init.go           # Add plugin prompt logic
│   └── plugins/
│       ├── plugins.go        # NEW: Plugin loading/generation
│       └── plugins_test.go   # NEW: Unit tests
```

---

## Migration Path

1. Create `~/.ocmgr/plugins/plugins.toml` from existing `plugins` text file
2. Run `ocmgr init` to test plugin selection
3. Delete old `plugins` text file after verification

---

## Future Enhancements (Out of Scope)

- Plugin categories/grouping
- Plugin search/filtering
- Plugin version constraints
- Plugin dependency resolution
- Plugin installation verification
