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
# Generates type-safe Go code from SQL migrations and queries for every DAL
# directory in the project, across the SQLite and PostgreSQL dialects.
#
# Usage:
#   ./hack/generate/dal.sh
#   ./hack/generate/dal.sh --validate

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Array of DAL directories (relative to PIKO_ROOT) and their package names.
DAL_TARGETS=(
    "internal/orchestrator/orchestrator_dal/querier_sqlite:db:sqlite"
    "internal/registry/registry_dal/querier_sqlite:db:sqlite"
    "internal/orchestrator/orchestrator_dal/querier_postgres:db:postgres"
    "internal/registry/registry_dal/querier_postgres:db:postgres"
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
        local base_dir package_name dialect
        IFS=':' read -r base_dir package_name dialect <<< "$target"
        local full_path="${PIKO_ROOT}/${base_dir}"

        piko::log::step "$current" "$target_count" "Generating: ${base_dir} (${dialect})"

        if [[ ! -d "$full_path" ]]; then
            piko::log::warn "Directory not found: $full_path"
            continue
        fi

        cd "$PIKO_ROOT" || piko::log::fatal "Failed to cd to $PIKO_ROOT"

        if go run ./cmd/generate_dal "$full_path" "$package_name" "$dialect"; then
            piko::log::success "Generated: ${base_dir}"
        else
            piko::log::error "Failed: ${base_dir}"
        fi
    done
}

# validate_dal regenerates the DAL and fails when the result differs from what is checked in.
# Globals:
#   DAL_TARGETS - Read
#   PIKO_ROOT - Read
validate_dal() {
    local generated_paths=()
    local target base_dir package_name
    for target in "${DAL_TARGETS[@]}"; do
        IFS=':' read -r base_dir package_name _ <<< "$target"
        generated_paths+=("${base_dir}/${package_name}")
    done

    local dirty
    dirty=$(cd "$PIKO_ROOT" && git status --porcelain -- "${generated_paths[@]}")
    if [[ -n "$dirty" ]]; then
        piko::log::error "Generated DAL directories are already modified; validate needs a clean tree"
        printf '%s\n' "$dirty"
        return 1
    fi

    generate_dal

    dirty=$(cd "$PIKO_ROOT" && git status --porcelain -- "${generated_paths[@]}")
    if [[ -n "$dirty" ]]; then
        piko::log::error "Generated DAL code is out of date; run 'make generate-dal' and commit the result"
        printf '%s\n' "$dirty"
        return 1
    fi

    piko::log::success "Generated DAL code is up to date"
}

# main generates DAL code, or validates that what is checked in matches the generator.
main() {
    if [[ "${1:-}" == "--validate" ]]; then
        piko::log::header "Validating generated DAL code"
        validate_dal || exit 1
        piko::log::footer

        return
    fi

    piko::log::header "Generating DAL code"

    generate_dal

    piko::log::footer
    piko::log::success "DAL generation complete!"
}

main "$@"
