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

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# generate_interp_piko_symbols extracts piko runtime symbols for the bytecode interpreter.
# Globals:
#   PIKO_ROOT - Read
generate_interp_piko_symbols() {
    cd "$PIKO_ROOT" || piko::log::fatal "Failed to cd to $PIKO_ROOT"

    go run ./cmd/piko extract generate \
        --manifest "${PIKO_ROOT}/piko-symbols-runtime.yaml" \
        --output "${PIKO_ROOT}/internal/interp/interp_adapters/driven_piko_symbols"
}

# main generates piko runtime symbol tables.
main() {
    piko::log::header "Generating bytecode interpreter piko runtime symbol tables"

    generate_interp_piko_symbols

    piko::log::footer
    piko::log::success "Piko runtime symbol generation complete!"
}

main "$@"
