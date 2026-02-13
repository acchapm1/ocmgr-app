# MCP Server Selection Feature - Implementation Plan

## Overview

Add interactive MCP server selection during `ocmgr init`. Users will be prompted to select MCP servers from a curated list in `~/.ocmgr/mcps/`, and the selected MCPs will be written to the generated `opencode.json` alongside any selected plugins.

---

## Prerequisites

- Phase 1 (Remove opencode.json copy) must be complete
- Phase 2 (Plugin selection) should be complete for best UX (combined prompt)

---

## MCP Configuration Structure

### opencode.json MCP Format (from docs)

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "MCP_SERVER_NAME": {
      "type": "local",
      "command": ["npx", "-y", "my-mcp-command"],
      "enabled": true,
      "environment": {
        "MY_ENV_VAR": "value"
      }
    },
    "REMOTE_MCP": {
      "type": "remote",
      "url": "https://mcp.example.com/mcp",
      "enabled": true,
      "headers": {
        "Authorization": "Bearer {env:MY_API_KEY}"
      }
    }
  }
}
```

### MCP Types

| Type | Required Fields | Optional Fields |
|------|-----------------|-----------------|
| `local` | `type`, `command` | `enabled`, `environment`, `timeout` |
| `remote` | `type`, `url` | `enabled`, `headers`, `oauth`, `timeout` |

---

## Phase 3: Implement MCP Selection Prompt

### 3.1 Define MCP List File Format

**Recommended: JSON format with one file per MCP** (`~/.ocmgr/mcps/<name>.json`)

**Why one file per MCP:**
- MCP configs can be complex (multiple fields, nested objects)
- Easier to manage individual MCPs (add/remove without editing large file)
- JSON format matches opencode.json output format
- Simple structure: name, description, and config

**File structure:**

```
~/.ocmgr/mcps/
├── sequentialthinking.json    # npx-based MCP
├── context7.json              # npx-based MCP
├── docker.json                # Docker-based MCP
├── sentry.json                # Remote MCP
└── filesystem.json            # npx with args
```

**Example: npx-based MCP (sequentialthinking.json)**

```json
{
  "name": "sequential-thinking",
  "description": "Step-by-step reasoning for complex problems",
  "config": {
    "type": "local",
    "command": ["npx", "-y", "@modelcontextprotocol/server-sequential-thinking"],
    "enabled": true
  }
}
```

**Example: Docker-based MCP (docker.json)**

```json
{
  "name": "MCP_DOCKER",
  "description": "Docker container management via MCP gateway",
  "config": {
    "type": "local",
    "command": ["docker", "mcp", "gateway", "run"],
    "enabled": true
  }
}
```

**Example: Remote MCP (sentry.json)**

```json
{
  "name": "sentry",
  "description": "Query Sentry issues and projects",
  "config": {
    "type": "remote",
    "url": "https://mcp.sentry.dev/mcp",
    "enabled": true
  }
}
```

**Example: npx with arguments (filesystem.json)**

```json
{
  "name": "filesystem",
  "description": "File system operations with allowed paths",
  "config": {
    "type": "local",
    "command": ["npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
    "enabled": true
  }
}
```

### 3.2 Extract MCP Name

**Strategy:** Use the `name` field from the JSON file

- This is the name that appears in opencode.json as the MCP key
- Also used for display in the selection prompt
- Filename can be descriptive (e.g., `sequentialthinking.json`) while `name` is the actual config key

**How it works:**
1. Read all `*.json` files from `~/.ocmgr/mcps/`
2. Parse each file to extract `name`, `description`, and `config`
3. The `config` object is directly inserted into opencode.json under the `name` key

### 3.3 Create MCP Package

**New directory:** `internal/mcps/`

```
internal/mcps/
├── mcps.go         # MCP list loading and parsing
└── mcps_test.go    # Unit tests
```

**File:** `internal/mcps/mcps.go`

```go
// Package mcps handles loading and managing the MCP server registry.
package mcps

// Definition represents an MCP server definition file.
type Definition struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Config      map[string]interface{} `json:"config"`
}

// Registry holds all available MCP servers.
type Registry struct {
    Servers []Definition
}

// Load reads all MCP definitions from ~/.ocmgr/mcps/*.json
func Load() (*Registry, error)

// List returns all available MCP servers.
func (r *Registry) List() []Definition

// GetByName finds an MCP by its name field.
func (r *Registry) GetByName(name string) *Definition
```

### 3.4 Update Init Command

**File:** `internal/cli/init.go`

Add MCP prompt after plugin prompt:

```go
// After plugin selection, prompt for MCPs
if !dryRun {
    if err := promptForMCPs(targetOpencode); err != nil {
        return fmt.Errorf("MCP selection: %w", err)
    }
}
```

**New function:** `promptForMCPs(targetDir string) error`

1. Check if `~/.ocmgr/mcps/` directory exists
2. If not, skip silently
3. Load all `*.toml` files from the directory
4. Display interactive selection prompt:
   ```
   Would you like to add MCP servers to this project? [y/N]
   ```
5. If yes, show categorized list:
   ```
   Available MCP servers:
   
   [devops]
     1. docker          - Docker container management
     2. filesystem      - File system operations
   
   [documentation]
     3. context7        - Search library documentation
     4. gh_grep         - Search GitHub code snippets
   
   [monitoring]
     5. sentry          - Query Sentry issues
   
   Select MCP servers (comma-separated numbers, or 'all'): 
   ```
6. Parse selection and update `opencode.json`

### 3.5 Update opencode.json Generation

**File:** `internal/plugins/plugins.go` (or new `internal/configgen/configgen.go`)

```go
// Config represents the opencode.json structure.
type Config struct {
    Schema string            `json:"$schema,omitempty"`
    Plugin []string          `json:"plugin,omitempty"`
    MCP    map[string]MCPEntry `json:"mcp,omitempty"`
}

// MCPEntry represents an MCP server entry in opencode.json.
type MCPEntry struct {
    Type        string            `json:"type"`
    Command     []string          `json:"command,omitempty"`
    URL         string            `json:"url,omitempty"`
    Enabled     bool              `json:"enabled,omitempty"`
    Environment map[string]string `json:"environment,omitempty"`
    Headers     map[string]string `json:"headers,omitempty"`
    OAuth       interface{}       `json:"oauth,omitempty"`
    Timeout     int               `json:"timeout,omitempty"`
}

// WriteConfig generates or updates opencode.json with plugins and MCPs.
func WriteConfig(targetDir string, plugins []string, mcps map[string]MCPEntry) error
```

**Generated file example:**

```json
{
  "$schema": "https://opencode.ai/config.json",
  "plugin": [
    "@plannotator/opencode@latest"
  ],
  "mcp": {
    "docker": {
      "type": "local",
      "command": ["docker", "mcp", "gateway", "run"],
      "enabled": true
    },
    "context7": {
      "type": "remote",
      "url": "https://mcp.context7.com/mcp",
      "enabled": true
    }
  }
}
```

### 3.6 Handle Existing opencode.json

Same logic as plugins:
- [ ] Read existing config
- [ ] Merge new MCPs with existing (deduplicate by name)
- [ ] Preserve any existing plugins
- [ ] Write merged config back

### 3.7 Update Documentation

**Files to update:**
- [ ] `README.md` - Add MCP selection to init command description
- [ ] `USAGE.md` - Add detailed MCP selection workflow

---

## Testing & Edge Cases

### Unit Tests

- [ ] Test TOML parsing of MCP definition files
- [ ] Test opencode.json generation with MCPs
- [ ] Test merging with existing opencode.json
- [ ] Test handling of missing mcps directory
- [ ] Test invalid TOML files (skip with warning)

### Integration Tests

- [ ] Test `ocmgr init` with MCP selection only
- [ ] Test `ocmgr init` with both plugins and MCPs
- [ ] Test `ocmgr init --dry-run` shows MCP prompt preview
- [ ] Test MCP selection with various flags

### Edge Cases

- [ ] Empty mcps directory → skip prompt
- [ ] Invalid TOML file → skip that file, log warning
- [ ] MCP with missing required fields → skip with error message
- [ ] User enters invalid selection → re-prompt
- [ ] User selects no MCPs → only include plugins in opencode.json
- [ ] Non-interactive terminal → skip prompt

---

## File Structure Summary

```
~/.ocmgr/
├── mcps/
│   ├── sequentialthinking.json  # MCP definition (NEW)
│   ├── context7.json            # MCP definition (NEW)
│   ├── docker.json              # MCP definition (NEW)
│   ├── sentry.json              # MCP definition (NEW)
│   └── filesystem.json          # MCP definition (NEW)
├── plugins/
│   └── plugins.toml             # Plugin registry
├── profiles/
│   └── base/
│       └── ...
└── config.toml

ocmgr/
├── internal/
│   ├── cli/
│   │   └── init.go              # Add MCP prompt logic
│   ├── mcps/
│   │   ├── mcps.go              # NEW: MCP loading/generation
│   │   └── mcps_test.go         # NEW: Unit tests
│   └── plugins/
│       └── plugins.go           # Plugin loading/generation
```

---

## Sample MCP Definition Files

### sequentialthinking.json (npx-based)

```json
{
  "name": "sequential-thinking",
  "description": "Step-by-step reasoning for complex problems",
  "config": {
    "type": "local",
    "command": ["npx", "-y", "@modelcontextprotocol/server-sequential-thinking"],
    "enabled": true
  }
}
```

### context7.json (npx-based)

```json
{
  "name": "context7",
  "description": "Search library documentation",
  "config": {
    "type": "local",
    "command": ["npx", "-y", "@upstash/context7-mcp"],
    "enabled": true
  }
}
```

### docker.json (Docker-based)

```json
{
  "name": "MCP_DOCKER",
  "description": "Docker container management via MCP gateway",
  "config": {
    "type": "local",
    "command": ["docker", "mcp", "gateway", "run"],
    "enabled": true
  }
}
```

### sentry.json (Remote with OAuth)

```json
{
  "name": "sentry",
  "description": "Query Sentry issues and projects",
  "config": {
    "type": "remote",
    "url": "https://mcp.sentry.dev/mcp",
    "enabled": true
  }
}
```

### filesystem.json (npx with arguments)

```json
{
  "name": "filesystem",
  "description": "File system operations with allowed paths",
  "config": {
    "type": "local",
    "command": ["npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
    "enabled": true
  }
}
```

---

## Future Enhancements (Out of Scope)

- MCP server health checks
- MCP server documentation links
- MCP server dependency resolution
- MCP server authentication setup prompts
- MCP server templates for custom servers
