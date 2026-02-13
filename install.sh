#!/usr/bin/env bash
#
# ocmgr installer
# Usage: curl -sSL https://raw.githubusercontent.com/acchapm1/ocmgr/main/install.sh | bash
#
set -euo pipefail

REPO="acchapm1/ocmgr"
BINARY="ocmgr"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Colors — use $'...' (ANSI-C quoting) so escape bytes are embedded at
# assignment time.  This makes every echo work regardless of whether the
# shell's built-in echo supports -e.
RED=$'\033[0;31m'
GREEN=$'\033[0;32m'
YELLOW=$'\033[1;33m'
BLUE=$'\033[0;34m'
BOLD=$'\033[1m'
NC=$'\033[0m' # No Color

info()  { echo "${BLUE}ℹ${NC}  $*"; }
ok()    { echo "${GREEN}✓${NC}  $*"; }
warn()  { echo "${YELLOW}⚠${NC}  $*"; }
error() { echo "${RED}✗${NC}  $*" >&2; }

echo "${BOLD}ocmgr installer${NC}"
echo ""

# ─── Detect OS and Architecture ───────────────────────────────────────────────

detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux*)  os="linux" ;;
        Darwin*) os="darwin" ;;
        *)       error "Unsupported OS: $(uname -s)"; exit 1 ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)             error "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac

    echo "${os}_${arch}"
}

PLATFORM=$(detect_platform)
info "Detected platform: ${PLATFORM}"

# ─── Check for Go ─────────────────────────────────────────────────────────────

check_go() {
    if command -v go &>/dev/null; then
        local go_version
        go_version=$(go version | awk '{print $3}')
        ok "Go found: ${go_version}"
        return 0
    fi
    return 1
}

install_go_prompt() {
    warn "Go is not installed."
    echo ""
    echo "  ocmgr can be installed by:"
    echo "    1) Installing Go first, then building from source"
    echo "    2) Downloading a pre-built binary from GitHub Releases"
    echo ""

    # Check if we're running interactively
    if [ -t 0 ]; then
        echo -n "Would you like to install Go now? [y/N] "
        read -r answer
        if [[ "${answer,,}" == "y" ]]; then
            install_go
            return 0
        fi
    fi

    echo ""
    echo "  To install Go manually:"
    echo ""
    echo "    ${BOLD}Linux (apt):${NC}"
    echo "      sudo apt update && sudo apt install -y golang-go"
    echo ""
    echo "    ${BOLD}Linux (snap):${NC}"
    echo "      sudo snap install go --classic"
    echo ""
    echo "    ${BOLD}macOS (Homebrew):${NC}"
    echo "      brew install go"
    echo ""
    echo "    ${BOLD}Any platform:${NC}"
    echo "      https://go.dev/dl/"
    echo ""
    echo "  After installing Go, re-run this script."
    return 1
}

install_go() {
    info "Detecting latest Go version..."
    local go_version
    go_version=$(curl -sSL "https://go.dev/VERSION?m=text" | head -1 | sed 's/^go//')
    if [ -z "${go_version}" ]; then
        error "Could not detect latest Go version from go.dev"
        exit 1
    fi
    info "Latest Go: ${go_version}"

    local os arch
    os=$(echo "$PLATFORM" | cut -d_ -f1)
    arch=$(echo "$PLATFORM" | cut -d_ -f2)
    local tarball="go${go_version}.${os}-${arch}.tar.gz"
    local url="https://go.dev/dl/${tarball}"

    info "Downloading ${url}..."
    curl -sSL -o "/tmp/${tarball}" "${url}"

    info "Extracting to /usr/local/go..."
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "/tmp/${tarball}"
    rm -f "/tmp/${tarball}"

    export PATH="/usr/local/go/bin:$PATH"

    if command -v go &>/dev/null; then
        ok "Go installed: $(go version | awk '{print $3}')"
        warn "Add to your shell profile: export PATH=/usr/local/go/bin:\$PATH"
    else
        error "Go installation failed"
        exit 1
    fi
}

# ─── Try pre-built binary first, fall back to building from source ────────────

install_from_release() {
    info "Checking for pre-built release..."

    local latest_tag
    latest_tag=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
        | grep '"tag_name"' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "${latest_tag}" ]; then
        warn "No releases found on GitHub. Will build from source."
        return 1
    fi

    local os arch
    os=$(echo "$PLATFORM" | cut -d_ -f1)
    arch=$(echo "$PLATFORM" | cut -d_ -f2)

    # Try common release naming patterns
    local asset_url=""
    for pattern in \
        "${BINARY}_${latest_tag}_${os}_${arch}.tar.gz" \
        "${BINARY}_${os}_${arch}.tar.gz" \
        "${BINARY}-${latest_tag}-${os}-${arch}.tar.gz"; do

        local try_url="https://github.com/${REPO}/releases/download/${latest_tag}/${pattern}"
        if curl -sSL --head "${try_url}" 2>/dev/null | grep -q "200"; then
            asset_url="${try_url}"
            break
        fi
    done

    if [ -z "${asset_url}" ]; then
        warn "No pre-built binary found for ${PLATFORM}. Will build from source."
        return 1
    fi

    info "Downloading ${asset_url}..."
    local tmpdir
    tmpdir=$(mktemp -d)
    curl -sSL -o "${tmpdir}/ocmgr.tar.gz" "${asset_url}"
    tar -xzf "${tmpdir}/ocmgr.tar.gz" -C "${tmpdir}"

    mkdir -p "${INSTALL_DIR}"
    cp "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    chmod +x "${INSTALL_DIR}/${BINARY}"
    rm -rf "${tmpdir}"

    ok "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
    return 0
}

install_from_source() {
    if ! check_go; then
        if ! install_go_prompt; then
            exit 1
        fi
    fi

    info "Building from source..."

    local tmpdir
    tmpdir=$(mktemp -d)

    info "Cloning ${REPO}..."
    git clone --depth 1 "https://github.com/${REPO}.git" "${tmpdir}/${BINARY}" 2>/dev/null

    info "Building..."
    (
        cd "${tmpdir}/${BINARY}"
        go build -ldflags "-s -w" -o "${BINARY}" ./cmd/ocmgr
    )

    mkdir -p "${INSTALL_DIR}"
    cp "${tmpdir}/${BINARY}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    chmod +x "${INSTALL_DIR}/${BINARY}"
    rm -rf "${tmpdir}"

    ok "Built and installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
}

# ─── Main ─────────────────────────────────────────────────────────────────────

if ! install_from_release; then
    install_from_source
fi

# Verify installation
if command -v "${BINARY}" &>/dev/null; then
    ok "ocmgr is ready! Run 'ocmgr --help' to get started."
elif [ -x "${INSTALL_DIR}/${BINARY}" ]; then
    ok "ocmgr installed to ${INSTALL_DIR}/${BINARY}"
    echo ""
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        warn "${INSTALL_DIR} is not in your PATH."

        # Detect the user's shell and suggest the right config file.
        shell_name=$(basename "${SHELL:-/bin/bash}")
        case "${shell_name}" in
            zsh)  shell_rc="~/.zshrc" ;;
            fish) shell_rc="~/.config/fish/config.fish" ;;
            *)    shell_rc="~/.bashrc" ;;
        esac

        echo "  Add this to ${shell_rc}:"
        echo ""
        if [ "${shell_name}" = "fish" ]; then
            echo "    fish_add_path ${INSTALL_DIR}"
        else
            echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
        fi
        echo ""
    fi
    echo "  Run '${INSTALL_DIR}/${BINARY} --help' to get started."
else
    error "Installation may have failed. Check the output above."
    exit 1
fi

# ─── Set up ~/.ocmgr if needed ───────────────────────────────────────────────

if [ ! -d "$HOME/.ocmgr" ]; then
    echo ""
    info "Setting up ~/.ocmgr..."
    mkdir -p "$HOME/.ocmgr/profiles"
    ok "Created ~/.ocmgr/profiles"
    echo "  Run 'ocmgr config init' to configure GitHub sync and defaults."
fi

echo ""
ok "Done!"
