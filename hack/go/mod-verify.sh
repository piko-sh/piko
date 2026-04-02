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

# hack/go/mod-verify.sh - Verify all go.mod files are tidy (read-only check)
#
# This script checks if `go mod tidy` would make any changes without
# actually modifying the files. Useful for CI checks.
#
# Usage:
#   ./hack/go/mod-verify.sh [directory]
#
# Exit codes:
#   0 - All go.mod files are tidy
#   1 - Some go.mod files need tidying

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

# Array of dirty module paths.
DIRTY=()

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
        piko::log::info "Verify all go.mod files are tidy (read-only check)."
        piko::log::info "By default, testdata directories are excluded."
        piko::log::info "Returns exit code 1 if any files need tidying."
        piko::log::blank
        piko::log::info "Flags:"
        piko::log::detail "  --all  Include testdata directories"
        piko::log::blank
        piko::log::info "To fix issues, run: ./hack/go/mod-tidy.sh"
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
#   MODULES - Set
#   MODULE_COUNT - Set
find_modules() {
    MODULES=$(piko::util::find_go_modules "$TARGET_DIR" "$INCLUDE_ALL")

    if [[ -z "$MODULES" ]]; then
        piko::log::warn "No go.mod files found in $TARGET_DIR"
        exit 0
    fi

    MODULE_COUNT=$(echo "$MODULES" | wc -l)
    piko::log::info "Checking $MODULE_COUNT go.mod file(s)"
}

# verify_modules checks each module is tidy without modifying files.
# Globals:
#   MODULES - Read
#   MODULE_COUNT - Read
#   DIRTY - Modified with dirty module paths
verify_modules() {
    local current=0

    while IFS= read -r mod_file; do
        current=$((current + 1))
        local mod_dir
        mod_dir=$(dirname "$mod_file")

        piko::log::step "$current" "$MODULE_COUNT" "$(piko::util::relative_path "$mod_dir")"

        if piko::go::tidy_module "$mod_dir" "true"; then
            piko::log::success "Clean: $(piko::util::relative_path "$mod_dir")"
        else
            piko::log::error "Dirty: $(piko::util::relative_path "$mod_dir")"
            DIRTY+=("$mod_dir")
        fi
    done <<<"$MODULES"
}

# print_summary displays verification results.
# Globals:
#   MODULE_COUNT - Read
#   DIRTY - Read
print_summary() {
    piko::log::blank
    piko::log::header "Summary"
    piko::log::info "Total modules: $MODULE_COUNT"
    piko::log::info "Clean: $((MODULE_COUNT - ${#DIRTY[@]}))"
    piko::log::info "Dirty: ${#DIRTY[@]}"

    if [[ ${#DIRTY[@]} -gt 0 ]]; then
        piko::log::blank
        piko::log::error "The following modules need tidying:"
        for mod in "${DIRTY[@]}"; do
            piko::log::detail "- $(piko::util::relative_path "$mod")"
        done
        piko::log::blank
        piko::log::info "Run './hack/go/mod-tidy.sh' to fix."
        exit 1
    fi

    piko::log::success "All go.mod files are tidy!"
}

# main verifies all go.mod files are tidy.
# Arguments:
#   $@ - Optional directory path
main() {
    validate_args "$@"

    piko::log::header "Verifying go.mod files are tidy"
    piko::log::info "Target: $TARGET_DIR"
    piko::log::footer

    find_modules
    piko::log::blank
    verify_modules
    print_summary
}

main "$@"
