# Releasing ocmgr

## Quick Release (Automated via GitHub Actions)

```
You tag a commit  →  Pipeline triggers  →  Binaries built  →  Release published
    (v1.0.0)         (GitHub Actions)      (6 platforms)      (download page)
```

A **release** is a GitHub page tied to a **git tag** where you can attach downloadable files (your binaries). Users see it at `github.com/acchapm1/ocmgr/releases`.

### Automated Release (Recommended)

```bash
# 1. Tag the current commit
git tag v0.1.0

# 2. Push the tag to GitHub
git push origin v0.1.0
```

The GitHub Actions workflow will automatically build binaries for all platforms and create the release.

---

## Manual Release (Step-by-Step Guide)

Use this method to learn the release process or when GitHub Actions is unavailable.

### Phase 1: Prepare the Codebase

#### Step 1.1: Update CHANGELOG.md

Move items from `[Unreleased]` to a new `[0.1.0]` section with today's date.

#### Step 1.2: Commit All Changes

```bash
git status
git add -A
git commit -m "chore: prepare for v0.1.0 release"
git push origin main
```

### Phase 2: Create and Push the Git Tag

```bash
# Create annotated tag
git tag -a v0.1.0 -m "Release v0.1.0

Features:
- Profile management (init, list, show, create, delete, import, export, snapshot)
- GitHub sync (push, pull, status)
- Plugin selection during init
- MCP server selection during init
- Self-update command
- Profile inheritance
- Conflict resolution (prompt, force, merge)
- Dry-run mode"

# Push the tag
git push origin v0.1.0
```

### Phase 3: Build Binaries for All Platforms

#### Step 3.1: Create Build Directory

```bash
mkdir -p release/v0.1.0
cd release/v0.1.0
```

#### Step 3.2: Build for All Platforms

**Linux AMD64:**
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/acchapm1/ocmgr/internal/cli.Version=v0.1.0" -o ocmgr ../../cmd/ocmgr
tar -czvf ocmgr_v0.1.0_linux_amd64.tar.gz ocmgr
rm ocmgr
```

**Linux ARM64:**
```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/acchapm1/ocmgr/internal/cli.Version=v0.1.0" -o ocmgr ../../cmd/ocmgr
tar -czvf ocmgr_v0.1.0_linux_arm64.tar.gz ocmgr
rm ocmgr
```

**macOS AMD64 (Intel):**
```bash
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/acchapm1/ocmgr/internal/cli.Version=v0.1.0" -o ocmgr ../../cmd/ocmgr
tar -czvf ocmgr_v0.1.0_darwin_amd64.tar.gz ocmgr
rm ocmgr
```

**macOS ARM64 (Apple Silicon):**
```bash
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/acchapm1/ocmgr/internal/cli.Version=v0.1.0" -o ocmgr ../../cmd/ocmgr
tar -czvf ocmgr_v0.1.0_darwin_arm64.tar.gz ocmgr
rm ocmgr
```

**Windows AMD64:**
```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/acchapm1/ocmgr/internal/cli.Version=v0.1.0" -o ocmgr.exe ../../cmd/ocmgr
tar -czvf ocmgr_v0.1.0_windows_amd64.tar.gz ocmgr.exe
rm ocmgr.exe
```

**Windows ARM64:**
```bash
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/acchapm1/ocmgr/internal/cli.Version=v0.1.0" -o ocmgr.exe ../../cmd/ocmgr
tar -czvf ocmgr_v0.1.0_windows_arm64.tar.gz ocmgr.exe
rm ocmgr.exe
```

#### Step 3.3: Verify Builds

```bash
ls -la *.tar.gz
```

Expected output:
```
ocmgr_v0.1.0_darwin_amd64.tar.gz
ocmgr_v0.1.0_darwin_arm64.tar.gz
ocmgr_v0.1.0_linux_amd64.tar.gz
ocmgr_v0.1.0_linux_arm64.tar.gz
ocmgr_v0.1.0_windows_amd64.tar.gz
ocmgr_v0.1.0_windows_arm64.tar.gz
```

#### Step 3.4: Generate Checksums

```bash
shasum -a 256 *.tar.gz > checksums.txt
cat checksums.txt
```

### Phase 4: Create GitHub Release

#### Step 4.1: Create Release with GitHub CLI

```bash
cd ../..  # Back to repo root

gh release create v0.1.0 \
  --title "v0.1.0" \
  --notes "## ocmgr v0.1.0 - Initial Release

### Installation

**macOS/Linux (curl):**
\`\`\`bash
curl -sSL https://raw.githubusercontent.com/acchapm1/ocmgr/main/install.sh | bash
\`\`\`

**From Source:**
\`\`\`bash
go install github.com/acchapm1/ocmgr/cmd/ocmgr@v0.1.0
\`\`\`

### Features

- **Profile Management**: Create, list, show, delete, import, export, and snapshot profiles
- **GitHub Sync**: Push and pull profiles to/from GitHub repositories
- **Plugin Selection**: Interactive plugin selection during \`ocmgr init\`
- **MCP Server Selection**: Interactive MCP server selection during \`ocmgr init\`
- **Self-Update**: Update ocmgr with \`ocmgr update\`
- **Profile Inheritance**: Extend profiles with parent profiles
- **Conflict Resolution**: Interactive, force, or merge modes
- **Dry-Run Mode**: Preview changes before applying

### Supported Platforms

| OS      | Architecture | Download                              |
|---------|--------------|---------------------------------------|
| Linux   | AMD64        | ocmgr_v0.1.0_linux_amd64.tar.gz       |
| Linux   | ARM64        | ocmgr_v0.1.0_linux_arm64.tar.gz       |
| macOS   | AMD64        | ocmgr_v0.1.0_darwin_amd64.tar.gz      |
| macOS   | ARM64        | ocmgr_v0.1.0_darwin_arm64.tar.gz      |
| Windows | AMD64        | ocmgr_v0.1.0_windows_amd64.tar.gz     |
| Windows | ARM64        | ocmgr_v0.1.0_windows_arm64.tar.gz     |

### What's Changed

See [CHANGELOG.md](https://github.com/acchapm1/ocmgr/blob/main/CHANGELOG.md) for details." \
  release/v0.1.0/*.tar.gz release/v0.1.0/checksums.txt
```

#### Step 4.2: Verify Release

Visit: https://github.com/acchapm1/ocmgr/releases/tag/v0.1.0

### Phase 5: Test the Release

```bash
# Test install script
curl -sSL https://raw.githubusercontent.com/acchapm1/ocmgr/main/install.sh | bash

# Verify version
ocmgr --version
# Should show: ocmgr version v0.1.0

# Test update command
ocmgr update
# Should show: ocmgr is already up to date
```

---

## Pipeline Details

### GitHub Actions — `.github/workflows/release.yml`

Triggers automatically when you push a tag starting with `v`. It:

- Builds binaries for **6 platforms**: darwin_arm64, darwin_amd64, linux_arm64, linux_amd64, windows_arm64, windows_amd64
- Packages each as a `.tar.gz` with SHA256 checksums
- Creates a GitHub Release with auto-generated release notes
- Tags containing `-rc`, `-beta`, or `-alpha` are marked as pre-releases

### Makefile — `make dist`

Cross-compiles all platforms locally (useful for testing before tagging):

```bash
make dist
```

Output:
```
dist/ocmgr_v0.1.0_darwin_arm64.tar.gz
dist/ocmgr_v0.1.0_darwin_amd64.tar.gz
dist/ocmgr_v0.1.0_linux_arm64.tar.gz
dist/ocmgr_v0.1.0_linux_amd64.tar.gz
dist/ocmgr_v0.1.0_windows_arm64.tar.gz
dist/ocmgr_v0.1.0_windows_amd64.tar.gz
```

---

## Pre-releases (for testing)

```bash
git tag v0.1.0-beta.1
git push origin v0.1.0-beta.1
```

---

## Install Script Integration

Once a release exists, `curl ... | bash` will automatically download the pre-built binary instead of building from source — the asset naming pattern (`ocmgr_v0.1.0_darwin_arm64.tar.gz`) matches what `install.sh` looks for.

---

## Troubleshooting

### "Tag already exists"
```bash
git tag -d v0.1.0
git push origin :refs/tags/v0.1.0
# Then recreate the tag
```

### "Release already exists"
```bash
gh release delete v0.1.0 --yes
# Then recreate the release
```

### Build fails for a specific platform
- Ensure you have Go 1.21+
- Use `CGO_ENABLED=0` for cross-compilation
- Check for platform-specific code

---

## Setup Checklist (One-time)

- [ ] Commit the workflow file (`.github/workflows/release.yml`) and push to `main`
- [ ] Verify GitHub Actions is enabled for the repo
  - Go to **Settings → Actions → General** in the repo
  - Ensure "Allow all actions and reusable workflows" is selected
- [ ] Verify workflow permissions
  - Go to **Settings → Actions → General → Workflow permissions**
  - Select "Read and write permissions"
  - This allows the pipeline to create releases and upload assets
- [ ] Test with a pre-release tag
  - `git tag v0.1.0-beta.1 && git push origin v0.1.0-beta.1`
  - Check the **Actions** tab in GitHub to watch the pipeline run
  - Verify the release appears at `github.com/acchapm1/ocmgr/releases`
