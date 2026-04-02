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

# hack/generate/flatc.sh - Generate Go code from FlatBuffers schemas
#
# Generates Go code from all .fbs FlatBuffers schema files in the project.
# Downloads flatc automatically if not present.
#
# Usage:
#   ./hack/generate/flatc.sh

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the flatc binary.
FLATC="${PIKO_ROOT}/.tools/flatc"

# Array of FlatBuffers schema paths.
FBS_SCHEMAS=(
    "internal/ast/ast_schema/template_ast.fbs"
    "internal/collection/collection_schema/collection.fbs"
    "internal/generator/generator_schema/manifest.fbs"
    "internal/i18n/i18n_schema/i18n.fbs"
    "internal/inspector/inspector_schema/type_data.fbs"
    "internal/registry/registry_schema/artefact_meta.fbs"
    "internal/search/search_schema/search_index.fbs"
    "internal/interp/interp_schema/bytecode.fbs"
    "internal/typegen/typegen_schema/action_manifest.fbs"
)

# ensure_flatc downloads flatc if not already present.
ensure_flatc() {
    if [[ ! -x "$FLATC" ]]; then
        piko::log::info "flatc not found, downloading..."
        "${PIKO_ROOT}/hack/tools/flatc.sh"
    fi
}

# generate_flatc runs flatc for all schema files.
# Globals:
#   FLATC - Read
#   FBS_SCHEMAS - Read
#   PIKO_ROOT - Read
generate_flatc() {
    local schema_count=${#FBS_SCHEMAS[@]}
    local current=0

    for schema in "${FBS_SCHEMAS[@]}"; do
        current=$((current + 1))
        local schema_path="${PIKO_ROOT}/${schema}"
        local schema_dir
        schema_dir=$(dirname "$schema_path")

        piko::log::step "$current" "$schema_count" "Generating: $schema"

        if [[ ! -f "$schema_path" ]]; then
            piko::log::warn "Schema not found: $schema_path"
            continue
        fi

        if "$FLATC" --go -o "$schema_dir" "$schema_path"; then
            piko::log::success "Generated: $schema"
        else
            piko::log::error "Failed: $schema"
        fi
    done
}

# main generates FlatBuffers code.
main() {
    piko::log::header "Generating FlatBuffers code"

    ensure_flatc
    generate_flatc

    piko::log::footer
    piko::log::success "FlatBuffers generation complete!"
}

main "$@"
