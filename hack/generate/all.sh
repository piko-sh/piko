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

# hack/generate/all.sh - Run all code generators
#
# Runs all code generation tools: dal, flatc, protoc, quicktemplate,
# interp symbols, and asmgen.
#
# Usage:
#   ./hack/generate/all.sh

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the generate scripts.
GENERATE_DIR="${PIKO_ROOT}/hack/generate"

# generate_all runs all code generation scripts.
# Globals:
#   GENERATE_DIR - Read
generate_all() {
    piko::log::header "Running all code generators"

    piko::log::info "Running dal..."
    "${GENERATE_DIR}/dal.sh"

    piko::log::blank
    piko::log::info "Running flatc..."
    "${GENERATE_DIR}/flatc.sh"

    piko::log::blank
    piko::log::info "Running protoc..."
    "${GENERATE_DIR}/protoc.sh"

    piko::log::blank
    piko::log::info "Running qtc..."
    "${GENERATE_DIR}/qt.sh"

    piko::log::blank
    piko::log::info "Running interp symbol extraction..."
    "${GENERATE_DIR}/interp_symbols.sh"

    piko::log::blank
    piko::log::info "Running asmgen..."
    "${GENERATE_DIR}/asmgen.sh"

    piko::log::footer
    piko::log::success "All code generation complete!"
}

# main runs all generators.
main() {
    generate_all
}

main "$@"
