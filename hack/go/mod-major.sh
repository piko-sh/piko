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

# hack/go/mod-major.sh - List available major version upgrades for all go.mod files
#
# Usage:
#   ./hack/go/mod-major.sh [directory]
#
# Examples:
#   ./hack/go/mod-major.sh               # Check all go.mod files
#   ./hack/go/mod-major.sh internal/lsp  # Check specific directory
#
# Requires: gomajor (go install github.com/icholy/gomajor@latest)

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Directory to search for go.mod files.
TARGET_DIR=""

# Newline-separated list of go.mod file paths.
MODULES=""

# Total number of modules found.
MODULE_COUNT=0

# Total number of available major version upgrades found.
TOTAL_UPGRADES=0

# validate_args parses the optional directory argument.
# Globals:
#   TARGET_DIR - Set
# Arguments:
#   $1 - Optional directory path (default: PIKO_ROOT)
validate_args() {
    if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
        piko::log::info "Usage: $0 [directory]"
        piko::log::blank
        piko::log::info "List available major version upgrades for all go.mod files."
        piko::log::blank
        piko::log::info "Examples:"
        piko::log::detail "$0               # Check all go.mod files"
        piko::log::detail "$0 internal/lsp  # Check specific directory"
        piko::log::blank
        piko::log::info "Requires:"
        piko::log::detail "gomajor  (go install github.com/icholy/gomajor@latest)"
        exit 0
    fi

    TARGET_DIR="${1:-${PIKO_ROOT}}"

    if [[ ! -d "$TARGET_DIR" ]]; then
        piko::log::fatal "Directory '$TARGET_DIR' does not exist."
    fi
}

# find_modules locates all go.mod files in the target directory.
# Globals:
#   TARGET_DIR - Read
#   MODULES - Set to newline-separated list of paths
#   MODULE_COUNT - Set
find_modules() {
    MODULES=$(piko::util::find_go_modules "$TARGET_DIR")

    if [[ -z "$MODULES" ]]; then
        piko::log::warn "No go.mod files found in $TARGET_DIR"
        exit 0
    fi

    MODULE_COUNT=$(echo "$MODULES" | wc -l)
    piko::log::info "Found $MODULE_COUNT go.mod file(s)"
}

# check_modules runs gomajor list on each module and reports results.
# Globals:
#   MODULES - Read
#   MODULE_COUNT - Read
#   TOTAL_UPGRADES - Modified
check_modules() {
    local current=0

    while IFS= read -r mod_file; do
        current=$((current + 1))
        local mod_dir
        mod_dir=$(dirname "$mod_file")

        local output
        output=$(cd "$mod_dir" && gomajor list 2>/dev/null) || true

        if [[ -n "$output" ]]; then
            piko::log::step "$current" "$MODULE_COUNT" "$(piko::util::relative_path "$mod_dir")"
            while IFS= read -r line; do
                piko::log::detail "  $line"
                TOTAL_UPGRADES=$((TOTAL_UPGRADES + 1))
            done <<<"$output"
        fi
    done <<<"$MODULES"
}

# print_summary displays results.
# Globals:
#   MODULE_COUNT - Read
#   TOTAL_UPGRADES - Read
print_summary() {
    piko::log::blank
    piko::log::header "Summary"
    piko::log::info "Modules checked: $MODULE_COUNT"

    if [[ "$TOTAL_UPGRADES" -gt 0 ]]; then
        piko::log::warn "$TOTAL_UPGRADES major version upgrade(s) available"
        piko::log::blank
        piko::log::info "To upgrade a specific dependency:"
        piko::log::detail "gomajor get github.com/example/module"
    else
        piko::log::success "All dependencies are on the latest major version."
    fi
}

# main checks all modules for available major version upgrades.
# Arguments:
#   $@ - Optional directory path
main() {
    validate_args "$@"

    piko::util::verify_binary "gomajor" "go install github.com/icholy/gomajor@latest" || exit 1

    piko::log::header "Checking for major version upgrades"
    piko::log::info "Target: $TARGET_DIR"
    piko::log::footer

    find_modules
    piko::log::blank
    check_modules
    print_summary
}

main "$@"
