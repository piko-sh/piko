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

# hack/generate/stdlib-bundle.sh - Generate the embedded stdlib type bundle
#
# Builds the stdlib type bundle used by the wasm playground and checks that it
# holds no machine specific paths.
#
# Usage:
#   ./hack/generate/stdlib-bundle.sh

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Output path of the embedded stdlib type bundle, relative to PIKO_ROOT.
readonly STDLIB_BIN="internal/wasm/wasm_data/stdlib.bin"

# generate_bundle builds the stdlib type bundle.
# Globals:
#   STDLIB_BIN - Read
# Returns:
#   Exits with code 1 if generation fails
generate_bundle() {
    piko::log::info "Generating embedded stdlib type bundle..."

    if ! go run ./cmd/stdlib-generator -output "${STDLIB_BIN}"; then
        piko::log::error "stdlib bundle generation failed"
        exit 1
    fi
}

# verify_no_machine_paths checks that the bundle holds no machine specific paths.
# Globals:
#   STDLIB_BIN - Read
#   HOME - Read
# Returns:
#   Exits with code 1 if a machine specific path is found
verify_no_machine_paths() {
    if strings "${STDLIB_BIN}" | grep -qE "${HOME}|/Users/|/home/"; then
        piko::log::error "stdlib bundle contains machine specific paths; the path sanitiser in"
        piko::log::error "internal/inspector/inspector_domain/stdlib_paths.go is not covering a field"
        exit 1
    fi
}

# main generates and verifies the stdlib type bundle.
# Globals:
#   STDLIB_BIN - Read
# Arguments:
#   $@ - Unused
main() {
    generate_bundle
    verify_no_machine_paths

    piko::log::success "Generated: ${STDLIB_BIN} ($(wc -c < "${STDLIB_BIN}") bytes)"
}

main "$@"
