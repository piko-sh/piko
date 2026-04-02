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

# hack/generate/dal.sh - Generate Go code from SQL using generate_dal
#
# Generates type-safe Go code from SQL migrations and queries for all
# SQLite DAL directories in the project.
#
# Usage:
#   ./hack/generate/dal.sh

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Array of DAL directories (relative to PIKO_ROOT) and their package names.
DAL_TARGETS=(
    "internal/orchestrator/orchestrator_dal/querier_sqlite:db"
    "internal/registry/registry_dal/querier_sqlite:db"
)

# generate_dal runs the generate_dal tool for all DAL targets.
# Globals:
#   DAL_TARGETS - Read
#   PIKO_ROOT - Read
generate_dal() {
    local target_count=${#DAL_TARGETS[@]}
    local current=0

    for target in "${DAL_TARGETS[@]}"; do
        current=$((current + 1))
        local base_dir="${target%%:*}"
        local package_name="${target##*:}"
        local full_path="${PIKO_ROOT}/${base_dir}"

        piko::log::step "$current" "$target_count" "Generating: ${base_dir}"

        if [[ ! -d "$full_path" ]]; then
            piko::log::warn "Directory not found: $full_path"
            continue
        fi

        cd "$PIKO_ROOT" || piko::log::fatal "Failed to cd to $PIKO_ROOT"

        if go run ./cmd/generate_dal "$full_path" "$package_name"; then
            piko::log::success "Generated: ${base_dir}"
        else
            piko::log::error "Failed: ${base_dir}"
        fi
    done
}

# main generates DAL code.
main() {
    piko::log::header "Generating DAL code"

    generate_dal

    piko::log::footer
    piko::log::success "DAL generation complete!"
}

main "$@"
