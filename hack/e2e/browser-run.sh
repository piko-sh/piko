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

# hack/e2e/browser-run.sh - Run E2E browser tests interactively
#
# Usage:
#   ./hack/e2e/browser-run.sh                    # Run all tests
#   ./hack/e2e/browser-run.sh 001_basic           # Run specific test
#   ./hack/e2e/browser-run.sh 003_pkc -headed     # Headed mode (no TUI)

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the E2E browser test directory.
E2E_DIR="${PIKO_ROOT}/tests/integration/e2e_browser"

# Test pattern to run.
TEST_PATTERN=""

# build_test_binary compiles the E2E test binary.
# Globals:
#   E2E_DIR - Read
build_test_binary() {
    cd "$E2E_DIR" || piko::log::fatal "E2E directory not found: $E2E_DIR"

    piko::log::info "Building test binary..."
    go test -c -tags "cgo,integration" -o e2e.test .
}

# run_tests executes the browser tests interactively.
# Globals:
#   TEST_PATTERN - Set
# Arguments:
#   $@ - Optional test pattern and flags (e.g., 01_basic -headed)
run_tests() {
    TEST_PATTERN="TestE2EBrowser_Integration"
    if [[ -n "$1" ]] && [[ "$1" != -* ]]; then
        TEST_PATTERN="TestE2EBrowser_Integration/$1"
        shift
    fi

    piko::log::info "Running: $TEST_PATTERN"
    ./e2e.test -test.v -test.run "$TEST_PATTERN" -interactive "$@"
}

# main builds and runs the E2E browser tests.
# Arguments:
#   $@ - Optional test pattern and flags
main() {
    build_test_binary
    run_tests "$@"
}

main "$@"
