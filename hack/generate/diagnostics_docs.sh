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

# hack/generate/diagnostics_docs.sh - Generate the diagnostics reference
#
# Writes docs/reference/diagnostics.md and docs/reference/diagnostics.json from the
# catalogue in internal/diagnostic_docs, which also backs `piko explain` and the
# documentation links in build output and the language server.
#
# Usage:
#   ./hack/generate/diagnostics_docs.sh
#   ./hack/generate/diagnostics_docs.sh --validate

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

OUTPUT_DIR="${PIKO_ROOT}/docs/reference"
GENERATED_FILES=(diagnostics.md diagnostics.json)

# generate_docs renders the reference into the directory given as the first argument.
# Arguments:
#   $1 - Output directory
# Globals:
#   PIKO_ROOT - Read
generate_docs() {
    local output_dir="$1"

    cd "$PIKO_ROOT" || piko::log::fatal "Failed to cd to $PIKO_ROOT"

    go run ./internal/diagnostic_docs/gendocs -out "$output_dir"
}

# validate_docs compares freshly generated references with the committed files and
# removes its scratch directory on exit.
# Globals:
#   PIKO_ROOT, OUTPUT_DIR, GENERATED_FILES - Read
validate_docs() {
    local scratch scratch_quoted name stale=0

    scratch="$(mktemp -d "${PIKO_ROOT}/.diagnostics_docs-validate.XXXXXX")"
    printf -v scratch_quoted '%q' "$scratch"
    # shellcheck disable=SC2064
    trap "rm -rf -- ${scratch_quoted}" EXIT

    generate_docs "$scratch"

    for name in "${GENERATED_FILES[@]}"; do
        if ! cmp -s "${OUTPUT_DIR}/${name}" "${scratch}/${name}"; then
            piko::log::error "Stale generated file: docs/reference/${name}"
            stale=1
        fi
    done

    rm -rf -- "$scratch"
    trap - EXIT

    if [[ $stale -ne 0 ]]; then
        piko::log::fatal "diagnostics reference is stale; run make generate-diagnostics-docs"
    fi
    piko::log::success "diagnostics reference is up to date"
}

# main generates or validates the diagnostics reference.
main() {
    if [[ "${1:-}" == "--validate" ]]; then
        piko::log::header "Validating the diagnostics reference"
        validate_docs
        piko::log::footer
        return 0
    fi

    piko::log::header "Generating the diagnostics reference"

    generate_docs "$OUTPUT_DIR"

    piko::log::footer
    piko::log::success "diagnostics reference generation complete!"
}

main "$@"
