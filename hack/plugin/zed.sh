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

# hack/plugin/zed.sh - Build and validate the Zed extension
#
# This script is called by Makefile targets:
#   make plugin-zed-build        Generate the grammar, test it, build the WASM
#   make plugin-zed-install      Build, then print the dev-install instructions
#   make plugin-zed-sync-commit  Pin extension.toml's grammar commit to HEAD
#   make plugin-zed-clean        Clean build artefacts

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the Zed extension directory.
ZED_DIR="${PIKO_ROOT}/plugins/zed"

# Path to the in-repo tree-sitter grammar package.
GRAMMAR_DIR="${ZED_DIR}/tree-sitter-piko"

# wasm_target_available reports whether a wasm target is installed for the
# active Rust toolchain, working for both rustup and a distro system toolchain
# (the std lives under the rustc sysroot in either case).
# Arguments:
#   $1 - the wasm target triple to probe
wasm_target_available() {
    local target="$1" sysroot
    sysroot="$(rustc --print sysroot 2>/dev/null)"
    if [[ -n "$sysroot" && -d "${sysroot}/lib/rustlib/${target}" ]]; then
        return 0
    fi
    rustup target list --installed 2>/dev/null | grep -qx "$target"
}

# detect_wasm_target echoes the wasm target Zed extensions compile to. Modern
# Zed (zed_extension_api >= 0.1) builds a component for wasm32-wasip2; older
# releases used wasm32-wasip1. Prefer an installed wasip2, fall back to an
# installed wasip1, and otherwise default to wasip2 so the install hint names
# the target current Zed actually needs.
detect_wasm_target() {
    if wasm_target_available "wasm32-wasip2"; then
        echo "wasm32-wasip2"
    elif wasm_target_available "wasm32-wasip1"; then
        echo "wasm32-wasip1"
    else
        echo "wasm32-wasip2"
    fi
}

# The wasm target Zed extensions compile to (auto-detected; see above).
WASM_TARGET="$(detect_wasm_target)"

# tree_sitter_cmd echoes the tree-sitter invocation to use, preferring a local
# binary and falling back to npx.
tree_sitter_cmd() {
    if command -v tree-sitter >/dev/null 2>&1; then
        echo "tree-sitter"
    else
        echo "npx --yes tree-sitter-cli@0.24.7"
    fi
}

# generate_grammar regenerates src/parser.c from grammar.js.
# Globals:
#   GRAMMAR_DIR - Read
generate_grammar() {
    piko::log::header "Generating Tree-sitter Grammar"

    cd "$GRAMMAR_DIR" || exit 1

    local ts
    ts="$(tree_sitter_cmd)"
    piko::log::info "Running ${ts} generate..."
    if $ts generate; then
        piko::log::success "Grammar generated"
    else
        piko::log::fatal "Grammar generation failed"
    fi
}

# test_grammar runs the corpus tests.
# Globals:
#   GRAMMAR_DIR - Read
test_grammar() {
    piko::log::header "Testing Tree-sitter Grammar"

    cd "$GRAMMAR_DIR" || exit 1

    local ts
    ts="$(tree_sitter_cmd)"
    piko::log::info "Running ${ts} test..."
    if $ts test; then
        piko::log::success "Grammar tests passed"
    else
        piko::log::fatal "Grammar tests failed"
    fi
}

# wasm_target_installed reports whether the active Rust toolchain can compile to
# the auto-detected wasm target.
# Globals:
#   WASM_TARGET - Read
wasm_target_installed() {
    wasm_target_available "$WASM_TARGET"
}

# build_extension compiles the extension to WebAssembly.
# Globals:
#   ZED_DIR - Read
#   WASM_TARGET - Read
build_extension() {
    piko::log::header "Building Extension (WASM)"

    cd "$ZED_DIR" || exit 1

    if ! wasm_target_installed; then
        piko::log::warn "Rust target ${WASM_TARGET} is not installed."
        piko::log::detail "rustup:  rustup target add ${WASM_TARGET}"
        piko::log::detail "Fedora:  sudo dnf install rust-std-static-${WASM_TARGET}"
        piko::log::detail "Debian:  sudo apt install rust-std-${WASM_TARGET}  # or rustup"
        piko::log::fatal "Cannot build the extension without the wasm target"
    fi

    piko::log::info "Running cargo build --release --target ${WASM_TARGET}..."
    if cargo build --release --target "$WASM_TARGET"; then
        piko::log::success "Extension built"
    else
        piko::log::fatal "Extension build failed"
    fi
}

# lint_rust runs clippy with the project's complexity thresholds.
# Globals:
#   ZED_DIR - Read
#   WASM_TARGET - Read
lint_rust() {
    piko::log::header "Linting Rust Extension (clippy)"

    cd "$ZED_DIR" || exit 1

    if ! wasm_target_installed; then
        piko::log::warn "Rust target ${WASM_TARGET} is not installed; skipping clippy."
        piko::log::detail "rustup target add ${WASM_TARGET}"
        return 0
    fi

    piko::log::info "Running cargo clippy (warnings as errors)..."
    if cargo clippy --release --target "$WASM_TARGET" --all-targets -- -D warnings; then
        piko::log::success "clippy passed"
    else
        piko::log::fatal "clippy reported issues"
    fi
}

# lint_c runs clang-tidy on the hand-written scanner with complexity thresholds.
# Globals:
#   GRAMMAR_DIR - Read
lint_c() {
    piko::log::header "Linting Scanner (clang-tidy)"

    cd "$GRAMMAR_DIR" || exit 1

    if ! command -v clang-tidy >/dev/null 2>&1; then
        piko::log::warn "clang-tidy is not installed; skipping C lint."
        return 0
    fi

    piko::log::info "Running clang-tidy on src/scanner.c..."
    if clang-tidy --warnings-as-errors='*' src/scanner.c -- -I src -std=c11; then
        piko::log::success "clang-tidy passed"
    else
        piko::log::fatal "clang-tidy reported issues"
    fi
}

# sync_commit pins the grammar commit in extension.toml to the current HEAD.
# The grammar lives in this repo, so the [grammars.piko] commit must advance
# whenever the grammar changes.
# Globals:
#   ZED_DIR - Read
sync_commit() {
    piko::log::header "Pinning Grammar Commit"

    local head_sha
    head_sha="$(git -C "$PIKO_ROOT" rev-parse HEAD)"

    local manifest="${ZED_DIR}/extension.toml"
    piko::log::info "Setting [grammars.piko].commit to ${head_sha}..."
    sed -i.bak -E "s/^commit = \".*\"/commit = \"${head_sha}\"/" "$manifest"
    rm -f "${manifest}.bak"

    piko::log::success "Pinned grammar commit to ${head_sha}"
    piko::log::warn "Commit extension.toml so the pinned grammar is reachable."
}

# validate_release_manifest guards the publish path against shipping a
# local-development grammar source. During development [grammars.piko].repository
# is pinned to a file:// URL so the grammar builds from a local checkout; that
# must never reach the marketplace, where it is unreachable for everyone else.
# Globals:
#   ZED_DIR - Read
validate_release_manifest() {
    piko::log::header "Validating Extension Manifest"

    local manifest="${ZED_DIR}/extension.toml"
    piko::log::info "Checking the grammar repository URL in extension.toml..."
    if grep -qE '^[[:space:]]*repository[[:space:]]*=[[:space:]]*"file://' "$manifest"; then
        piko::log::fatal "extension.toml grammar repository is a local file:// URL; set it to https://github.com/piko-sh/piko before publishing"
    fi

    piko::log::success "Manifest grammar repository is publishable"
}

# print_install_instructions explains the Zed dev-install flow (Zed has no CLI
# install for dev extensions).
# Globals:
#   ZED_DIR - Read
print_install_instructions() {
    piko::log::header "Installing in Zed"

    piko::log::info "Zed installs dev extensions from the editor:"
    piko::log::detail "1. Open Zed"
    piko::log::detail "2. Command palette -> 'zed: install dev extension'"
    piko::log::detail "3. Select: ${ZED_DIR}"
    piko::log::blank
    piko::log::info "Ensure pikopls is on PATH (the binary must be named pikopls):"
    piko::log::detail "make build-lsp && cp ${PIKO_ROOT}/bin/lsp/pikopls /usr/local/bin/"
    piko::log::detail "# go install names the binary 'lsp'; rename it to pikopls if you use that route"
}

# clean_build removes build artefacts.
# Globals:
#   ZED_DIR - Read
clean_build() {
    piko::log::header "Cleaning Build Artefacts"

    piko::log::info "Removing Rust target directory..."
    rm -rf "${ZED_DIR:?}/target"

    piko::log::success "Build artefacts cleaned"
}

# main handles build, install, lint, sync-commit, publish, or clean commands.
# Arguments:
#   $1 - Command to execute: build (default), install, lint, sync-commit,
#        publish, or clean
main() {
    local command="${1:-build}"

    case "$command" in
        build)
            generate_grammar
            test_grammar
            build_extension
            piko::log::blank
            piko::log::success "Build complete!"
            piko::log::blank
            piko::log::info "To install: make plugin-zed-install"
            ;;

        install)
            generate_grammar
            test_grammar
            build_extension
            piko::log::blank
            print_install_instructions
            ;;

        lint)
            lint_rust
            lint_c
            piko::log::blank
            piko::log::success "Lint complete!"
            ;;

        sync-commit)
            sync_commit
            ;;

        publish)
            validate_release_manifest
            generate_grammar
            test_grammar
            build_extension
            piko::log::blank
            piko::log::success "Publish checks passed; extension is ready to publish."
            ;;

        clean)
            clean_build
            ;;

        *)
            piko::log::fatal "Unknown command: $command (expected: build, install, lint, sync-commit, publish, clean)"
            ;;
    esac
}

main "$@"
