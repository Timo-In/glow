#!/usr/bin/env bash

# ==============================================================================
# Glow Build Script (dev & release)
# ==============================================================================

set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Git metadata
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")

print_info() {
    echo -e "${BLUE}==>${NC} ${CYAN}$1${NC}"
}

print_success() {
    echo -e "${GREEN}==> $1${NC}"
}

print_warn() {
    echo -e "${YELLOW}==> $1${NC}"
}

print_error() {
    echo -e "${RED}==> $1${NC}"
}

# ------------------------------------------------------------------------------
# Dev Build
# ------------------------------------------------------------------------------
build_dev() {
    local target_bin="./glow"
    if [[ "$(go env GOOS)" == "windows" ]]; then
        target_bin="./glow.exe"
    fi

    print_info "Building local dev binary for $(go env GOOS)/$(go env GOARCH)..."
    print_info "Version: ${VERSION} (${COMMIT}) built at ${DATE}"

    local ldflags="-X main.Version=${VERSION} -X main.CommitSHA=${COMMIT}"

    CGO_ENABLED=0 go build -ldflags="${ldflags}" -o "${target_bin}" .

    print_success "Dev build finished: ${target_bin}"
    ls -lh "${target_bin}"
}

# ------------------------------------------------------------------------------
# Release Build
# ------------------------------------------------------------------------------
build_release() {
    local release_dir="./release"
    rm -rf "${release_dir}"
    mkdir -p "${release_dir}"

    print_info "Starting multi-platform release builds & packaging..."
    print_info "Version: ${VERSION} (${COMMIT}) Date: ${DATE}"
    print_info "Output directory: ${release_dir}"
    echo ""

    local ldflags="-s -w -X main.Version=${VERSION} -X main.CommitSHA=${COMMIT}"

    # Format: "GOOS/GOARCH/ARCH_NAME/EXT"
    local platforms=(
        "darwin/arm64/darwin_arm64/"
        "darwin/amd64/darwin_x86_64/"
        "linux/amd64/linux_x86_64/"
        "linux/arm64/linux_arm64/"
        "linux/arm/linux_armv7/"
        "linux/386/linux_386/"
        "windows/amd64/windows_x86_64/.exe"
        "windows/arm64/windows_arm64/.exe"
        "freebsd/amd64/freebsd_x86_64/"
    )

    local tmp_dir="${release_dir}/.tmp"

    for item in "${platforms[@]}"; do
        IFS="/" read -r os arch arch_name ext <<< "${item}"
        local bin_name="glow${ext}"
        local archive_base="glow_${VERSION}_${arch_name}"
        local package_dir="${tmp_dir}/${archive_base}"

        rm -rf "${package_dir}"
        mkdir -p "${package_dir}"

        printf "  • Compiling & packaging for %-7s %-7s -> " "${os}" "${arch}"

        GOOS="${os}" GOARCH="${arch}" CGO_ENABLED=0 go build \
            -ldflags="${ldflags}" \
            -trimpath \
            -o "${package_dir}/${bin_name}" .

        # Include LICENSE & README if present
        [[ -f LICENSE ]] && cp LICENSE "${package_dir}/"
        [[ -f README.md ]] && cp README.md "${package_dir}/"

        # Create tar.gz archive
        (
            cd "${package_dir}"
            tar -czf "../../${archive_base}.tar.gz" *
        )

        # For Windows, also create a .zip archive if zip is available
        if [[ "${os}" == "windows" ]] && command -v zip >/dev/null 2>&1; then
            (
                cd "${package_dir}"
                zip -q -r "../../${archive_base}.zip" *
            )
        fi

        rm -rf "${package_dir}"
        echo -e "${archive_base}.tar.gz ${GREEN}[OK]${NC}"
    done

    rm -rf "${tmp_dir}"

    echo ""
    print_info "Generating SHA256 checksums..."
    (
        cd "${release_dir}"
        if command -v shasum >/dev/null 2>&1; then
            shasum -a 256 glow_* > checksums.txt
        elif command -v sha256sum >/dev/null 2>&1; then
            sha256sum glow_* > checksums.txt
        fi
    )

    print_success "All release archives successfully built in ${release_dir}:"
    ls -lh "${release_dir}"
}

# ------------------------------------------------------------------------------
# Clean
# ------------------------------------------------------------------------------
clean() {
    print_info "Cleaning build artifacts..."
    rm -rf ./glow ./glow.exe ./release
    print_success "Clean finished."
}

# ------------------------------------------------------------------------------
# Install
# ------------------------------------------------------------------------------
install_bin() {
    build_dev
    local dest_dir="${HOME}/.local/bin"
    if [[ ! -d "${dest_dir}" ]]; then
        dest_dir="$(go env GOPATH)/bin"
    fi
    mkdir -p "${dest_dir}"
    cp ./glow "${dest_dir}/glow"
    print_success "Installed glow to ${dest_dir}/glow"
}

# ------------------------------------------------------------------------------
# Main Dispatcher
# ------------------------------------------------------------------------------
cmd="${1:-dev}"

case "${cmd}" in
    dev)
        build_dev
        ;;
    install)
        install_bin
        ;;
    release)
        build_release
        ;;
    clean)
        clean
        ;;
    help|--help|-h)
        echo "Usage: $0 [dev|install|release|clean|help]"
        echo ""
        echo "Commands:"
        echo "  dev       Build single local binary in current directory (./glow)"
        echo "  install   Build and install to ~/.local/bin/glow"
        echo "  release   Cross-compile release binaries for all platforms into ./release"
        echo "  clean     Remove built binaries and release folder"
        echo "  help      Show this help message"
        ;;
    *)
        print_error "Unknown command: ${cmd}"
        echo "Run '$0 help' for available commands."
        exit 1
        ;;
esac
