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

# hack/lint/go.sh - Lint all Go modules in the workspace with golangci-lint
#
# golangci-lint does not support go.work natively, so this script parses the
# workspace file and runs the linter against each module individually.
#
# Usage:
#   ./hack/lint/go.sh              # CI mode: skip build-constrained modules
#   ./hack/lint/go.sh --all        # Local mode: include all build tags

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Build tags to pass to golangci-lint via --build-tags.
# When --all is used, this includes all custom build tags so that modules
# gated behind constraints (integration, vips, ffmpeg, etc.) are also linted.
BUILD_TAGS=""

# All custom build tags used in the project. Platform tags (linux, darwin,
# windows) and Go version tags (go1.25) are excluded since they are handled
# automatically by the toolchain.
readonly ALL_BUILD_TAGS="integration,vips,ffmpeg,bench,fuzz"

# Array of workspace module paths.
MODULES=()

# Array of module paths that require GOOS/GOARCH cross-compilation for linting.
# These are handled separately from normal modules because golangci-lint's
# --build-tags flag cannot set GOOS/GOARCH.
WASM_MODULES=()

# Array of module paths that had lint failures.
FAILED=()

# Count of modules skipped due to build constraints.
SKIPPED=0

# parse_args processes command-line arguments.
# Globals:
#   BUILD_TAGS - Set when --all is passed
# Arguments:
#   $@ - Command-line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --all)
                BUILD_TAGS="$ALL_BUILD_TAGS"
                shift
                ;;
            --tags)
                BUILD_TAGS="$2"
                shift 2
                ;;
            -h|--help)
                piko::log::info "Usage: $0 [--all] [--tags <tags>]"
                piko::log::detail "--all         Include all custom build tags (integration,vips,ffmpeg,...)"
                piko::log::detail "--tags <tags>  Specify custom build tags (comma-separated)"
                exit 0
                ;;
            *)
                piko::log::fatal "Unknown argument: $1 (use --help for usage)"
                ;;
        esac
    done
}

# verify_tools checks that required tools are installed.
# Returns:
#   Exits with code 1 if any tool is missing
verify_tools() {
    if ! piko::util::verify_binary "golangci-lint" "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; then
        exit 1
    fi
    if ! piko::util::verify_binary "jq" "brew install jq OR apt install jq"; then
        exit 1
    fi
}

# is_wasm_module checks if a module requires GOOS=js GOARCH=wasm to lint.
# Arguments:
#   $1 - Module path relative to PIKO_ROOT
# Returns:
#   0 if the module contains only js/wasm build-constrained files, 1 otherwise
is_wasm_module() {
    local mod_path="$1"
    local abs_path="${PIKO_ROOT}/${mod_path}"

    # Check if all .go files in the module have //go:build js && wasm
    local total_files
    total_files=$(find "$abs_path" -maxdepth 1 -name '*.go' | wc -l)
    if [[ "$total_files" -eq 0 ]]; then
        return 1
    fi

    local wasm_files
    wasm_files=$(grep -rl '//go:build js && wasm' "$abs_path"/*.go 2>/dev/null | wc -l)

    [[ "$total_files" -eq "$wasm_files" ]]
}

# discover_modules reads workspace module paths from go.work.
# Modules that require GOOS=js GOARCH=wasm are separated into WASM_MODULES
# so they can be linted with the correct cross-compilation environment.
# Globals:
#   PIKO_ROOT - Read
#   BUILD_TAGS - Read
#   MODULES - Set to array of module paths
#   WASM_MODULES - Set to array of wasm module paths (only when --all)
discover_modules() {
    local gowork="${PIKO_ROOT}/go.work"

    if [[ ! -f "$gowork" ]]; then
        piko::log::fatal "No go.work file found at ${gowork}"
    fi

    local all_modules=()
    while IFS= read -r mod_path; do
        all_modules+=("$mod_path")
    done < <(go work edit -json | jq -r '.Use[].DiskPath')

    if [[ ${#all_modules[@]} -eq 0 ]]; then
        piko::log::fatal "No modules found in go.work"
    fi

    # When --all is used, separate wasm modules for cross-compilation linting.
    for mod_path in "${all_modules[@]}"; do
        if [[ -n "$BUILD_TAGS" ]] && is_wasm_module "$mod_path"; then
            WASM_MODULES+=("$mod_path")
        else
            MODULES+=("$mod_path")
        fi
    done

    piko::log::info "Found ${#MODULES[@]} modules in go.work"
    if [[ ${#WASM_MODULES[@]} -gt 0 ]]; then
        piko::log::info "Found ${#WASM_MODULES[@]} wasm modules (will lint with GOOS=js GOARCH=wasm)"
    fi
}

# lint_module runs golangci-lint against a single module path.
# Modules whose files are all excluded by build constraints (e.g. //go:build
# integration, wasm, vips, ffmpeg) produce a "no go files to analyze" error
# from golangci-lint. These are skipped rather than counted as failures.
# Globals:
#   BUILD_TAGS - Read
#   FAILED - Modified with failed module paths
#   SKIPPED - Incremented for build-constrained modules
# Arguments:
#   $1 - Module path relative to PIKO_ROOT
lint_module() {
    local mod_path="$1"
    local output
    local rc=0
    local -a args=("run")

    if [[ -n "$BUILD_TAGS" ]]; then
        args+=("--build-tags" "$BUILD_TAGS")
    fi
    args+=("${mod_path}/...")

    output=$(golangci-lint "${args[@]}" 2>&1) || rc=$?

    if [[ $rc -eq 0 ]]; then
        piko::log::success "$mod_path"
        return
    fi

    if echo "$output" | grep -q "no go files to analyze"; then
        piko::log::warn "$mod_path (skipped: no files match build constraints)"
        SKIPPED=$((SKIPPED + 1))
        return
    fi

    echo "$output"
    piko::log::error "$mod_path"
    FAILED+=("$mod_path")
}

# lint_wasm_module runs golangci-lint against a wasm module using GOOS=js
# GOARCH=wasm cross-compilation environment.
# Globals:
#   BUILD_TAGS - Read
#   FAILED - Modified with failed module paths
# Arguments:
#   $1 - Module path relative to PIKO_ROOT
lint_wasm_module() {
    local mod_path="$1"
    local output
    local rc=0
    local -a args=("run")

    if [[ -n "$BUILD_TAGS" ]]; then
        args+=("--build-tags" "$BUILD_TAGS")
    fi
    args+=("${mod_path}/...")

    output=$(GOOS=js GOARCH=wasm golangci-lint "${args[@]}" 2>&1) || rc=$?

    if [[ $rc -eq 0 ]]; then
        piko::log::success "$mod_path (GOOS=js GOARCH=wasm)"
        return
    fi

    echo "$output"
    piko::log::error "$mod_path (GOOS=js GOARCH=wasm)"
    FAILED+=("$mod_path")
}

# lint_modules runs golangci-lint against each workspace module.
# Wasm modules are linted separately with cross-compilation environment.
# Globals:
#   MODULES - Read
#   WASM_MODULES - Read
#   FAILED - Modified with failed module paths
#   SKIPPED - Modified with skipped module count
lint_modules() {
    local i=0
    local total=$(( ${#MODULES[@]} + ${#WASM_MODULES[@]} ))

    for mod_path in "${MODULES[@]}"; do
        i=$((i + 1))
        piko::log::step "$i" "$total" "$mod_path"
        lint_module "$mod_path"
    done

    for mod_path in "${WASM_MODULES[@]}"; do
        i=$((i + 1))
        piko::log::step "$i" "$total" "$mod_path (wasm)"
        lint_wasm_module "$mod_path"
    done
}

# print_summary displays lint results across all modules.
# Globals:
#   MODULES - Read
#   FAILED - Read
#   SKIPPED - Read
print_summary() {
    local total=$(( ${#MODULES[@]} + ${#WASM_MODULES[@]} ))
    local failed=${#FAILED[@]}
    local passed=$((total - failed - SKIPPED))

    piko::log::blank
    piko::log::header "Summary"
    piko::log::info "Total modules: $total"
    piko::log::info "Passed: $passed"
    piko::log::info "Skipped: $SKIPPED (build constraints)"
    piko::log::info "Failed: $failed"

    if [[ $failed -gt 0 ]]; then
        piko::log::blank
        piko::log::error "The following modules have lint issues:"
        for mod in "${FAILED[@]}"; do
            piko::log::detail "- $mod"
        done
        piko::log::blank
        piko::log::info "Run golangci-lint directly on failing modules for details:"
        piko::log::detail "golangci-lint run <module>/..."
        exit 1
    fi

    piko::log::success "All modules passed golangci-lint!"
}

# main lints all Go workspace modules.
# Arguments:
#   $@ - Optional flags (--all, --tags)
main() {
    parse_args "$@"
    verify_tools

    piko::log::header "Linting Go workspace modules with golangci-lint"
    if [[ -n "$BUILD_TAGS" ]]; then
        piko::log::info "Build tags: $BUILD_TAGS"
    fi

    discover_modules
    piko::log::blank
    lint_modules
    print_summary
}

main "$@"
