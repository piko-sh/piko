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

# hack/release/sbom.sh - Generate Software Bill of Materials (SBOM)
#
# Generates CycloneDX and SPDX SBOMs for a compiled Go binary using syft.
# Output files are placed alongside the input binary.
#
# This script is called by Makefile targets:
#   make sbom-cli  Generate SBOM for CLI binary
#   make sbom-lsp  Generate SBOM for LSP binary

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# show_usage displays help information.
show_usage() {
    piko::log::info "Usage: ./hack/release/sbom.sh <binary-path>"
    piko::log::blank
    piko::log::info "Arguments:"
    piko::log::detail "binary-path  Path to the compiled Go binary to analyse"
    piko::log::blank
    piko::log::info "Output:"
    piko::log::detail "<binary-path>.sbom.cdx.json   CycloneDX JSON SBOM"
    piko::log::detail "<binary-path>.sbom.spdx.json  SPDX JSON SBOM"
    piko::log::blank
    piko::log::info "Makefile targets:"
    piko::log::detail "make sbom-cli  # Generate SBOM for CLI binary"
    piko::log::detail "make sbom-lsp  # Generate SBOM for LSP binary"
}

# main generates SBOMs for the given binary.
# Arguments:
#   $1 - Path to the compiled Go binary
main() {
    if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
        show_usage
        exit 0
    fi

    local binary="${1:-}"
    if [[ -z "$binary" ]]; then
        piko::log::error "No binary path provided"
        piko::log::blank
        show_usage
        exit 1
    fi

    if [[ ! -f "$binary" ]]; then
        piko::log::fatal "Binary not found: $binary"
    fi

    piko::util::verify_binary "syft" "go install github.com/anchore/syft/cmd/syft@latest" || exit 1

    local cdx_output="${binary}.sbom.cdx.json"
    local spdx_output="${binary}.sbom.spdx.json"

    piko::log::info "Generating CycloneDX SBOM..."
    syft "$binary" -o "cyclonedx-json" --file "$cdx_output"
    piko::log::success "CycloneDX SBOM: $cdx_output"

    piko::log::info "Generating SPDX SBOM..."
    syft "$binary" -o "spdx-json" --file "$spdx_output"
    piko::log::success "SPDX SBOM: $spdx_output"
}

main "$@"
