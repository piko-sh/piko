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

# hack/lsp/build.sh - Build LSP binaries
#
# This script is called by Makefile targets:
#   make build-lsp      Build for current platform
#   make build-lsp-all  Build for all platforms

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# show_usage displays help information.
show_usage() {
    piko::log::info "Usage: ./hack/lsp/build.sh [current|all] [output_dir]"
    piko::log::blank
    piko::log::info "Modes:"
    piko::log::detail "current  Build for current platform only"
    piko::log::detail "all      Build for all supported platforms (default)"
    piko::log::blank
    piko::log::info "Makefile targets:"
    piko::log::detail "make build-lsp      # Build for current platform"
    piko::log::detail "make build-lsp-all  # Build for all platforms"
}

# build_all_binaries builds LSP binaries for all platforms and shows next steps.
# Arguments:
#   $1 - Output directory
build_all_binaries() {
    local output_dir="$1"

    piko::lsp::build_all "$output_dir"

    piko::log::blank
    piko::log::info "Next steps:"
    piko::log::detail "make plugin-vscode-build  # Build VSCode extension"
    piko::log::detail "make plugin-idea-build    # Build IntelliJ plugin"
}

# main handles the build process.
# Arguments:
#   $1 - Mode: current or all (default)
#   $2 - Output directory (default: bin/lsp/)
main() {
    if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
        show_usage
        exit 0
    fi

    local mode="${1:-all}"

    case "$mode" in
        current)
            local output_dir="${2:-${PIKO_ROOT}/bin/lsp}"
            piko::lsp::build_current "$output_dir"
            ;;
        all)
            local output_dir="${2:-${PIKO_ROOT}/bin/lsp}"
            build_all_binaries "$output_dir"
            ;;
        *)
            piko::log::fatal "Unknown mode: $mode (expected: current, all)"
            ;;
    esac
}

main "$@"
