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

# hack/cli/build.sh - Build CLI binaries
#
# This script is called by Makefile targets:
#   make build-cli      Build for current platform
#   make build-cli-all  Build for all platforms

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# show_usage displays help information.
show_usage() {
    piko::log::info "Usage: ./hack/cli/build.sh [current|all] [output]"
    piko::log::blank
    piko::log::info "Modes:"
    piko::log::detail "current  Build for current platform only (default)"
    piko::log::detail "all      Build for all supported platforms"
    piko::log::blank
    piko::log::info "Makefile targets:"
    piko::log::detail "make build-cli      # Build for current platform"
    piko::log::detail "make build-cli-all  # Build for all platforms"
}

# main handles the build process.
# Arguments:
#   $1 - Mode: current (default) or all
#   $2 - Output path or directory
main() {
    if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
        show_usage
        exit 0
    fi

    local mode="${1:-current}"

    case "$mode" in
        current)
            local output="${2:-${PIKO_ROOT}/bin/piko}"
            piko::cli::build_current "$output"
            ;;
        all)
            local output="${2:-${PIKO_ROOT}/bin/cli}"
            piko::cli::build_all "$output"
            ;;
        *)
            piko::log::fatal "Unknown mode: $mode (expected: current, all)"
            ;;
    esac
}

main "$@"
