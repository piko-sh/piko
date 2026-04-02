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

# hack/go/mod-tidy.sh - Tidy all go.mod files in the project
#
# Usage:
#   ./hack/go/mod-tidy.sh [directory]
#
# Examples:
#   ./hack/go/mod-tidy.sh               # Tidy all go.mod files
#   ./hack/go/mod-tidy.sh internal/lsp  # Tidy specific directory

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Directory to search for go.mod files.
TARGET_DIR=""

# Whether to include testdata directories.
INCLUDE_ALL=""

# Newline-separated list of go.mod file paths.
MODULES=""

# Total number of modules found.
MODULE_COUNT=0

# Array of failed module paths.
FAILED=()

# validate_args parses the optional directory argument.
# Globals:
#   TARGET_DIR - Set
#   INCLUDE_ALL - Set
# Arguments:
#   $@ - Optional flags and directory path
validate_args() {
    if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
        piko::log::info "Usage: $0 [--all] [directory]"
        piko::log::blank
        piko::log::info "Tidy all go.mod files in the project or a specific directory."
        piko::log::info "By default, testdata directories are excluded."
        piko::log::blank
        piko::log::info "Flags:"
        piko::log::detail "  --all  Include testdata directories"
        piko::log::blank
        piko::log::info "Examples:"
        piko::log::detail "$0               # Tidy all go.mod files"
        piko::log::detail "$0 --all         # Tidy all go.mod files including testdata"
        piko::log::detail "$0 internal/lsp  # Tidy specific directory"
        exit 0
    fi

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --all)
                INCLUDE_ALL="--all"
                shift
                ;;
            *)
                TARGET_DIR="$1"
                shift
                ;;
        esac
    done

    TARGET_DIR="${TARGET_DIR:-${PIKO_ROOT}}"

    if [[ ! -d "$TARGET_DIR" ]]; then
        piko::log::fatal "Directory '$TARGET_DIR' does not exist."
    fi
}

# find_modules locates all go.mod files in the target directory.
# Globals:
#   TARGET_DIR - Read
#   INCLUDE_ALL - Read
#   MODULES - Set to newline-separated list of paths
#   MODULE_COUNT - Set
find_modules() {
    MODULES=$(piko::util::find_go_modules "$TARGET_DIR" "$INCLUDE_ALL")

    if [[ -z "$MODULES" ]]; then
        piko::log::warn "No go.mod files found in $TARGET_DIR"
        exit 0
    fi

    MODULE_COUNT=$(echo "$MODULES" | wc -l)
    piko::log::info "Found $MODULE_COUNT go.mod file(s)"
}

# tidy_modules runs go mod tidy on each module.
# Globals:
#   MODULES - Read
#   MODULE_COUNT - Read
#   FAILED - Modified with failed module paths
tidy_modules() {
    local current=0

    while IFS= read -r mod_file; do
        current=$((current + 1))
        local mod_dir
        mod_dir=$(dirname "$mod_file")

        piko::log::step "$current" "$MODULE_COUNT" "$(piko::util::relative_path "$mod_dir")"

        if piko::go::tidy_module "$mod_dir"; then
            piko::log::success "Tidied: $(piko::util::relative_path "$mod_dir")"
        else
            piko::log::error "Failed: $(piko::util::relative_path "$mod_dir")"
            FAILED+=("$mod_dir")
        fi
    done <<<"$MODULES"
}

# print_summary displays tidying results.
# Globals:
#   MODULE_COUNT - Read
#   FAILED - Read
print_summary() {
    piko::log::blank
    piko::log::header "Summary"
    piko::log::info "Total modules: $MODULE_COUNT"
    piko::log::info "Successful: $((MODULE_COUNT - ${#FAILED[@]}))"
    piko::log::info "Failed: ${#FAILED[@]}"

    if [[ ${#FAILED[@]} -gt 0 ]]; then
        piko::log::blank
        piko::log::error "Failed modules:"
        for mod in "${FAILED[@]}"; do
            piko::log::detail "- $(piko::util::relative_path "$mod")"
        done
        exit 1
    fi

    piko::log::success "All go.mod files tidied successfully!"
}

# main tidies all go.mod files in the target directory.
# Arguments:
#   $@ - Optional directory path
main() {
    validate_args "$@"

    piko::log::header "Tidying go.mod files"
    piko::log::info "Target: $TARGET_DIR"
    piko::log::footer

    find_modules
    piko::log::blank
    tidy_modules
    print_summary
}

main "$@"
