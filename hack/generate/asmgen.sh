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

# hack/generate/asmgen.sh - Generate Plan 9 assembly files
#
# Generates architecture-specific assembly (.s) and header (.h) files
# for the bytecode interpreter dispatch loop and vectormaths SIMD
# functions.
#
# Usage:
#   ./hack/generate/asmgen.sh
#   ./hack/generate/asmgen.sh --validate

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# generate_asm runs the asmgen tool to produce .s and .h files.
# Globals:
#   PIKO_ROOT - Read
generate_asm() {
    cd "$PIKO_ROOT" || piko::log::fatal "Failed to cd to $PIKO_ROOT"

    go run ./cmd/asmgen "$@"
}

# main generates assembly files.
main() {
    piko::log::header "Generating Plan 9 assembly files"

    generate_asm "$@"

    piko::log::footer
    piko::log::success "Assembly generation complete!"
}

main "$@"
