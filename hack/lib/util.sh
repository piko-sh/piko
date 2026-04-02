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

# hack/lib/util.sh - Common utility functions for Piko scripts
#
# This file should be sourced, not executed directly.
# All functions are namespaced with piko::util::

# Prevent double-sourcing
if [[ -n "${_PIKO_UTIL_LOADED:-}" ]]; then
    return 0
fi
readonly _PIKO_UTIL_LOADED=1

# piko::util::find_project_root returns the root directory of the Piko project
# This is set by init.sh and exported as PIKO_ROOT
piko::util::find_project_root() {
    echo "${PIKO_ROOT}"
}

# piko::util::verify_binary checks if a binary is available in PATH
# Arguments:
#   $1 - Binary name
#   $2 - Optional install command hint
# Returns:
#   0 if binary exists, 1 otherwise
piko::util::verify_binary() {
    local binary="$1"
    local install_cmd="${2:-}"

    if ! command -v "$binary" &>/dev/null; then
        piko::log::error "Required binary not found: $binary"
        if [[ -n "$install_cmd" ]]; then
            piko::log::info "Install with: $install_cmd"
        fi
        return 1
    fi
    return 0
}

# piko::util::verify_binaries checks if all required binaries are available
# Arguments:
#   $@ - Array of "binary:install_cmd" strings
# Returns:
#   0 if all binaries exist, 1 otherwise (and prints missing ones)
piko::util::verify_binaries() {
    local -a required=("$@")
    local -a missing=()

    for entry in "${required[@]}"; do
        local binary="${entry%%:*}"
        if ! command -v "$binary" &>/dev/null; then
            missing+=("$entry")
        fi
    done

    if [[ ${#missing[@]} -gt 0 ]]; then
        piko::log::error "The following required binaries are not installed:"
        echo
        for entry in "${missing[@]}"; do
            local binary="${entry%%:*}"
            local install_path="${entry##*:}"
            echo "  - $binary"
            echo "    Install with: go install $install_path"
            echo
        done
        return 1
    fi
    return 0
}

# piko::util::host_os returns the current operating system
# Returns: linux, darwin, or windows
piko::util::host_os() {
    local os
    os="$(uname -s)"
    case "$os" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)       piko::log::fatal "Unsupported OS: $os" ;;
    esac
}

# piko::util::host_arch returns the current architecture
# Returns: amd64, arm64, etc.
piko::util::host_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64)  echo "amd64" ;;
        aarch64) echo "arm64" ;;
        arm64)   echo "arm64" ;;
        *)       piko::log::fatal "Unsupported architecture: $arch" ;;
    esac
}

# piko::util::ensure_dir creates a directory if it doesn't exist
# Arguments:
#   $1 - Directory path
piko::util::ensure_dir() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        mkdir -p "$dir"
    fi
}

# piko::util::is_git_clean checks if the git working directory is clean
# Returns:
#   0 if clean, 1 if dirty
piko::util::is_git_clean() {
    if [[ -n "$(git status --porcelain 2>/dev/null)" ]]; then
        return 1
    fi
    return 0
}

# piko::util::git_root returns the root of the git repository
piko::util::git_root() {
    git rev-parse --show-toplevel 2>/dev/null
}

# piko::util::relative_path converts an absolute path to a path relative to PIKO_ROOT
# Arguments:
#   $1 - Absolute path
piko::util::relative_path() {
    local path="$1"
    echo "${path#"${PIKO_ROOT}/"}"
}

# piko::util::find_go_packages finds all Go packages under a directory
# Arguments:
#   $1 - Directory to search (defaults to PIKO_ROOT)
piko::util::find_go_packages() {
    local dir="${1:-${PIKO_ROOT}}"
    find "$dir" -name "*.go" -type f -not -path "*/vendor/*" -not -path "*/.git/*" -print0 \
        | xargs -0 -I{} dirname {} \
        | sort -u
}

# piko::util::find_go_modules finds all go.mod files under a directory
# Arguments:
#   $1 - Directory to search (defaults to PIKO_ROOT)
piko::util::find_go_modules() {
    local dir="${1:-${PIKO_ROOT}}"
    find "$dir" -name "go.mod" -type f -not -path "*/vendor/*" -not -path "*/.git/*" -not -path "*/node_modules/*"
}

# piko::util::find_go_workspaces finds all go.work files under a directory
# Arguments:
#   $1 - Directory to search (defaults to PIKO_ROOT)
piko::util::find_go_workspaces() {
    local dir="${1:-${PIKO_ROOT}}"
    find "$dir" -name "go.work" -type f -not -path "*/vendor/*" -not -path "*/.git/*" -not -path "*/node_modules/*"
}
