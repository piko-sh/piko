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

# hack/test/coverage.sh - Generate LLM-friendly annotated coverage report for a package
#
# Usage:
#   ./hack/test/coverage.sh <package-path>
#
# Example:
#   ./hack/test/coverage.sh internal/generator/generator_adapters/driven_code_emitter_go_literal

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the Go package to analyse.
PACKAGE_PATH=""

# validate_args parses the package path argument.
# Globals:
#   PACKAGE_PATH - Set to the package path
# Arguments:
#   $1 - Package path (required)
validate_args() {
    if [[ $# -eq 0 || "$1" == "-h" || "$1" == "--help" ]]; then
        piko::log::info "Usage: $0 <package-path>"
        piko::log::detail "Example: $0 internal/generator/generator_adapters/driven_code_emitter_go_literal"
        exit 1
    fi

    PACKAGE_PATH="$1"

    if [[ ! -d "$PACKAGE_PATH" ]]; then
        piko::log::fatal "Package path '$PACKAGE_PATH' does not exist or is not a directory."
    fi
}

# run_coverage generates annotated coverage report for the package.
# Globals:
#   PACKAGE_PATH - Read
#   PIKO_ROOT - Read
run_coverage() {
    local coverage_file="/tmp/coverage.out"
    trap 'rm -f "$coverage_file"' RETURN

    piko::log::info "Generating coverage for: $PACKAGE_PATH"

    (
        cd "$PACKAGE_PATH" || exit 1
        go test -coverprofile="$coverage_file"
    )

    go run "${PIKO_ROOT}/cmd/cover/cover-annotate.go" \
        --uncovered-only \
        --show-code \
        --threshold=90 \
        --skip-files "otel.go,*_mock.go,*_test.go" \
        "$coverage_file"
}

# main generates coverage for the specified package.
# Arguments:
#   $@ - Package path argument
main() {
    validate_args "$@"
    run_coverage
}

main "$@"
