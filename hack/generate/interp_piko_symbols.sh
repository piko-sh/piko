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

# hack/generate/interp_piko_symbols.sh - Generate bytecode interpreter piko runtime symbol tables
#
# Extracts Go symbols from piko framework packages for the bytecode
# interpreter so that interpreted code can call runtime functions at
# native speed via reflect.Value.Call().
#
# Usage:
#   ./hack/generate/interp_piko_symbols.sh
#   ./hack/generate/interp_piko_symbols.sh --validate

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

MANIFEST="${PIKO_ROOT}/piko-symbols-runtime.yaml"
OUTPUT_DIR="${PIKO_ROOT}/wdk/interp/interp_piko_symbols"

# generate_symbols runs the extractor against the manifest, writing into the directory
# given as the first argument.
# Arguments:
#   $1 - Output directory
# Globals:
#   PIKO_ROOT, MANIFEST - Read
generate_symbols() {
    local output_dir="$1"

    cd "$PIKO_ROOT" || piko::log::fatal "Failed to cd to $PIKO_ROOT"

    go run ./cmd/piko extract generate \
        --manifest "$MANIFEST" \
        --output "$output_dir"
}

# list_generated_files prints the generated files (gen_* and the export blobs) below a
# directory, relative to it and sorted, so two output trees can be compared.
# Arguments:
#   $1 - Directory to list
list_generated_files() {
    (cd "$1" && find . -path './gen_*' -type f | sort)
}

# validate_symbols regenerates into a scratch directory inside the repository (the
# extractor's sandbox refuses paths outside it) and fails when any generated file differs
# from the committed output, including the embedded types export blobs. The scratch
# directory is removed on every exit path: the EXIT trap embeds its already-quoted path,
# because the local variable is out of scope by the time the trap runs.
# Globals:
#   PIKO_ROOT, OUTPUT_DIR - Read
validate_symbols() {
    local scratch scratch_quoted expected actual stale=0 relative

    scratch="$(mktemp -d "${PIKO_ROOT}/.interp_piko_symbols-validate.XXXXXX")"
    printf -v scratch_quoted '%q' "$scratch"
    # shellcheck disable=SC2064
    trap "rm -rf -- ${scratch_quoted}" EXIT

    generate_symbols "$scratch" > /dev/null

    expected="$(list_generated_files "$scratch")"
    actual="$(list_generated_files "$OUTPUT_DIR")"
    if [[ "$expected" != "$actual" ]]; then
        diff <(echo "$actual") <(echo "$expected") || true
        stale=1
    fi

    while IFS= read -r relative; do
        [[ -n "$relative" ]] || continue
        if [[ -f "${OUTPUT_DIR}/${relative}" ]] && ! cmp -s "${OUTPUT_DIR}/${relative}" "${scratch}/${relative}"; then
            piko::log::error "Stale generated file: wdk/interp/interp_piko_symbols/${relative#./}"
            stale=1
        fi
    done <<< "$expected"

    rm -rf -- "$scratch"
    trap - EXIT

    if [[ $stale -ne 0 ]]; then
        piko::log::fatal "bytecode interpreter piko runtime symbol tables are stale; run make generate-interp-piko-symbols"
    fi
    piko::log::success "bytecode interpreter piko runtime symbol tables are up to date"
}

# main generates or validates the symbol tables.
main() {
    if [[ "${1:-}" == "--validate" ]]; then
        piko::log::header "Validating bytecode interpreter piko runtime symbol tables"
        validate_symbols
        piko::log::footer
        return 0
    fi

    piko::log::header "Generating bytecode interpreter piko runtime symbol tables"

    generate_symbols "$OUTPUT_DIR"

    piko::log::footer
    piko::log::success "bytecode interpreter piko runtime symbol tables generation complete!"
}

main "$@"
