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

# hack/go/mod-upgrade.sh - Upgrade all direct dependencies in all go.mod files
#
# Usage:
#   ./hack/go/mod-upgrade.sh [--dry-run] [directory]
#
# Examples:
#   ./hack/go/mod-upgrade.sh               # Upgrade all go.mod files
#   ./hack/go/mod-upgrade.sh --dry-run     # Report available upgrades without applying
#   ./hack/go/mod-upgrade.sh internal/lsp  # Upgrade specific directory

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Directory to search for go.mod files.
TARGET_DIR=""

# Whether to perform a dry run (report only, no changes).
DRY_RUN="false"

# Newline-separated list of go.mod file paths.
MODULES=""

# Total number of modules found.
MODULE_COUNT=0

# Array of failed module paths.
FAILED=()

# validate_args parses flags and the optional directory argument.
# Globals:
#   TARGET_DIR - Set
#   DRY_RUN - Set
# Arguments:
#   $@ - Flags and optional directory path
validate_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help)
                piko::log::info "Usage: $0 [--dry-run] [directory]"
                piko::log::blank
                piko::log::info "Upgrade all direct dependencies in all go.mod files."
                piko::log::blank
                piko::log::info "Flags:"
                piko::log::detail "--dry-run  Report available upgrades without applying them"
                piko::log::blank
                piko::log::info "Examples:"
                piko::log::detail "$0                  # Upgrade all go.mod files"
                piko::log::detail "$0 --dry-run        # Report available upgrades"
                piko::log::detail "$0 internal/lsp     # Upgrade specific directory"
                exit 0
                ;;
            --dry-run)
                DRY_RUN="true"
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

# upgrade_modules runs go get -u on each module and reports changes.
# Globals:
#   MODULES - Read
#   MODULE_COUNT - Read
#   DRY_RUN - Read
#   FAILED - Modified with failed module paths
upgrade_modules() {
    local current=0

    while IFS= read -r mod_file; do
        current=$((current + 1))
        local mod_dir
        mod_dir=$(dirname "$mod_file")

        piko::log::step "$current" "$MODULE_COUNT" "$(piko::util::relative_path "$mod_dir")"

        if piko::go::upgrade_module "$mod_dir" "$DRY_RUN"; then
            if [[ "$DRY_RUN" == "true" ]]; then
                piko::log::success "Checked: $(piko::util::relative_path "$mod_dir")"
            else
                piko::log::success "Upgraded: $(piko::util::relative_path "$mod_dir")"
            fi
        else
            piko::log::error "Failed: $(piko::util::relative_path "$mod_dir")"
            FAILED+=("$mod_dir")
        fi
    done <<<"$MODULES"
}

# sync_workspaces runs go work sync on all go.work files in the target directory.
# Globals:
#   TARGET_DIR - Read
#   DRY_RUN - Read
sync_workspaces() {
    local workspaces
    workspaces=$(piko::util::find_go_workspaces "$TARGET_DIR")

    if [[ -z "$workspaces" ]]; then
        return 0
    fi

    local ws_count
    ws_count=$(echo "$workspaces" | wc -l)

    piko::log::blank
    piko::log::header "Syncing Go workspaces"
    piko::log::info "Found $ws_count go.work file(s)"
    piko::log::blank

    local current=0
    while IFS= read -r work_file; do
        current=$((current + 1))
        local work_dir
        work_dir=$(dirname "$work_file")

        piko::log::step "$current" "$ws_count" "$(piko::util::relative_path "$work_dir")"

        if (cd "$work_dir" && go work sync 2>/dev/null); then
            piko::log::success "Synced: $(piko::util::relative_path "$work_dir")"
        else
            piko::log::error "Failed to sync: $(piko::util::relative_path "$work_dir")"
            FAILED+=("$work_dir")
        fi
    done <<<"$workspaces"
}

# tidy_modules runs go mod tidy on all modules after workspace sync.
# Globals:
#   MODULES - Read
#   MODULE_COUNT - Read
tidy_modules() {
    piko::log::blank
    piko::log::header "Tidying all modules"
    piko::log::blank

    local current=0

    while IFS= read -r mod_file; do
        current=$((current + 1))
        local mod_dir
        mod_dir=$(dirname "$mod_file")

        piko::log::step "$current" "$MODULE_COUNT" "$(piko::util::relative_path "$mod_dir")"

        if (cd "$mod_dir" && go mod tidy 2>/dev/null); then
            piko::log::success "Tidied: $(piko::util::relative_path "$mod_dir")"
        else
            piko::log::warn "Tidy failed: $(piko::util::relative_path "$mod_dir")"
        fi
    done <<<"$MODULES"
}

# print_summary displays upgrade results.
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

    if [[ "$DRY_RUN" == "true" ]]; then
        piko::log::success "Dry run complete. No files were modified."
    else
        piko::log::success "All direct dependencies upgraded successfully!"
    fi
}

# main upgrades all direct dependencies in all go.mod files.
# Arguments:
#   $@ - Optional directory path
main() {
    validate_args "$@"

    if [[ "$DRY_RUN" == "true" ]]; then
        piko::log::header "Checking available dependency upgrades (dry run)"
    else
        piko::log::header "Upgrading direct dependencies"
    fi
    piko::log::info "Target: $TARGET_DIR"
    piko::log::footer

    find_modules
    piko::log::blank
    upgrade_modules

    if [[ "$DRY_RUN" != "true" ]]; then
        sync_workspaces
        tidy_modules
    fi

    print_summary
}

main "$@"
