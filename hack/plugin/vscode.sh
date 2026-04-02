#!/bin/bash
# Copyright 2026 PolitePixels Limited
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# This project stands against fascism, authoritarianism, and all forms of
# oppression. We built this to empower people, not to enable those who would
# strip others of their rights and dignity.

# hack/plugin/vscode.sh - Build and release VSCode extension
#
# This script is called by Makefile targets:
#   make plugin-vscode-build    Build and package
#   make plugin-vscode-install  Build, package, and install
#   make plugin-vscode-clean    Clean build artefacts

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the VSCode extension directory.
VSCODE_DIR="${PIKO_ROOT}/plugins/vscode"

# Central location for built LSP binaries.
LSP_BIN_DIR="${PIKO_ROOT}/bin/lsp"

# Target directory for LSP binaries within the plugin.
PLUGIN_BIN_DIR="${VSCODE_DIR}/bin"

# kill_lsp_processes terminates any running LSP processes.
kill_lsp_processes() {
    piko::log::header "Killing Running LSP Processes"

    if pgrep -f "piko-lsp" >/dev/null 2>&1; then
        piko::log::info "Found running LSP processes, killing them..."
        pkill -f "piko-lsp" || true
        sleep 1
        piko::log::success "LSP processes killed"
    else
        piko::log::info "No running LSP processes found"
    fi
}

# build_lsp_binaries builds LSP binaries and copies to plugin directory.
# Globals:
#   LSP_BIN_DIR - Read
#   PLUGIN_BIN_DIR - Read
build_lsp_binaries() {
    piko::lsp::build_all "$LSP_BIN_DIR"

    piko::log::header "Copying LSP Binaries to Plugin"
    piko::util::ensure_dir "$PLUGIN_BIN_DIR"

    piko::log::info "Copying from $LSP_BIN_DIR to $PLUGIN_BIN_DIR..."
    if cp -r "$LSP_BIN_DIR"/* "$PLUGIN_BIN_DIR/"; then
        piko::log::success "LSP binaries copied to plugin"
    else
        piko::log::fatal "Failed to copy LSP binaries"
    fi
}

# compile_typescript compiles the extension source with esbuild.
# Globals:
#   VSCODE_DIR - Read
compile_typescript() {
    piko::log::header "Compiling TypeScript"

    cd "$VSCODE_DIR" || exit 1

    piko::log::info "Running npm install..."
    npm install >/dev/null 2>&1

    piko::log::info "Compiling with esbuild (production mode)..."
    npm run compile:production

    piko::log::success "TypeScript compiled"
}

# package_extension creates a VSIX package.
# Globals:
#   VSCODE_DIR - Read
package_extension() {
    piko::log::header "Packaging Extension"

    cd "$VSCODE_DIR" || exit 1

    piko::log::info "Creating VSIX package..."
    npm run package

    local vsix_file
    vsix_file=$(find_vsix_file)

    if [[ -n "$vsix_file" ]] && [[ -f "$vsix_file" ]]; then
        local size
        size=$(du -h "$vsix_file" | cut -f1)
        piko::log::success "Extension packaged: $vsix_file ($size)"
        piko::log::detail "Location: ${VSCODE_DIR}/${vsix_file}"
    else
        piko::log::fatal "No VSIX file created"
    fi
}

# find_vsix_file locates the most recent VSIX file.
# Globals:
#   VSCODE_DIR - Read
# Outputs:
#   Writes the VSIX filename to stdout
find_vsix_file() {
    cd "$VSCODE_DIR" || exit 1

    local -a vsix_files
    shopt -s nullglob
    vsix_files=(piko-*.vsix)
    shopt -u nullglob

    if [[ ${#vsix_files[@]} -gt 0 ]]; then
        echo "${vsix_files[-1]}"
    fi
}

# install_extension installs the VSIX into VSCode.
# Globals:
#   VSCODE_DIR - Read
install_extension() {
    piko::log::header "Installing Extension in VSCode"

    cd "$VSCODE_DIR" || exit 1

    local vsix_file
    vsix_file=$(find_vsix_file)

    if [[ -z "$vsix_file" ]] || [[ ! -f "$vsix_file" ]]; then
        piko::log::fatal "No VSIX file found. Run 'make plugin-vscode-build' first."
    fi

    piko::log::info "Uninstalling old version..."
    code --uninstall-extension politepixels.piko >/dev/null 2>&1 || true

    sleep 1

    piko::log::info "Installing $vsix_file..."
    if code --install-extension "$vsix_file" --force; then
        piko::log::success "Extension installed"
        piko::log::blank
        piko::log::warn "Please reload VSCode window to activate the new extension:"
        piko::log::detail "Ctrl+Shift+P -> 'Developer: Reload Window'"
    else
        piko::log::error "Failed to install extension"
        piko::log::info "You may need to manually restart VSCode and run:"
        piko::log::detail "code --install-extension $vsix_file --force"
        exit 1
    fi
}

# clean_build removes all build artefacts.
# Globals:
#   VSCODE_DIR - Read
#   PLUGIN_BIN_DIR - Read
clean_build() {
    piko::log::header "Cleaning Build Artefacts"

    cd "$VSCODE_DIR" || exit 1

    piko::log::info "Removing plugin binaries..."
    rm -rf "${PLUGIN_BIN_DIR:?}"/*

    piko::log::info "Removing compiled TypeScript..."
    rm -rf out/

    piko::log::info "Removing VSIX packages..."
    rm -f piko-*.vsix

    piko::log::success "Build artefacts cleaned"
}

# main handles build, install, or clean commands.
# Arguments:
#   $1 - Command to execute: build (default), install, or clean
main() {
    local command="${1:-build}"

    case "$command" in
        build)
            kill_lsp_processes
            build_lsp_binaries
            compile_typescript
            package_extension
            piko::log::blank
            piko::log::success "Build complete!"
            piko::log::blank
            piko::log::info "To install: make plugin-vscode-install"
            ;;

        install)
            kill_lsp_processes
            build_lsp_binaries
            compile_typescript
            package_extension
            piko::log::blank
            install_extension
            ;;

        clean)
            clean_build
            ;;

        *)
            piko::log::fatal "Unknown command: $command (expected: build, install, clean)"
            ;;
    esac
}

main "$@"
