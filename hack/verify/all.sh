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

# hack/verify/all.sh - Run all verification checks
#
# This script runs all verification checks for the Piko project.
# It is designed to be run before submitting code or in CI.
#
# Usage:
#   ./hack/verify/all.sh

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the hack directory.
HACK_DIR="${PIKO_ROOT}/hack"

# Array of failed check names.
FAILED=()

# run_check executes a verification script and tracks results.
# Globals:
#   FAILED - Modified with failed check names
# Arguments:
#   $1 - Display name for the check
#   $2 - Path to the script to run
#   $@ - Additional arguments for the script
run_check() {
    local name="$1"
    local script="$2"
    shift 2

    piko::log::header "$name"

    if [[ ! -x "$script" ]]; then
        piko::log::error "Script not found or not executable: $script"
        FAILED+=("$name")
        return
    fi

    if "$script" "$@"; then
        piko::log::success "$name passed"
    else
        piko::log::error "$name failed"
        FAILED+=("$name")
    fi

    piko::log::footer
}

# run_all_checks runs all configured verification checks.
# Globals:
#   HACK_DIR - Read
run_all_checks() {
    run_check "Go Linting (golangci-lint)" "${HACK_DIR}/lint/go.sh"
    run_check "Script Linting (shellcheck)" "${HACK_DIR}/lint/scripts.sh"
    run_check "Licence Compatibility" "${HACK_DIR}/lint/licences.sh"
    run_check "Go Module Verification" "${HACK_DIR}/go/mod-verify.sh"
}

# print_summary displays verification results.
# Globals:
#   FAILED - Read
print_summary() {
    local total=4
    local passed=$((total - ${#FAILED[@]}))

    piko::log::blank
    piko::log::header "Summary"

    piko::log::info "Total checks: $total"
    piko::log::info "Passed: $passed"
    piko::log::info "Failed: ${#FAILED[@]}"

    if [[ ${#FAILED[@]} -gt 0 ]]; then
        piko::log::blank
        piko::log::error "The following checks failed:"
        for check in "${FAILED[@]}"; do
            piko::log::detail "- $check"
        done
        piko::log::blank
        piko::log::info "Fix the issues and run ./hack/verify/all.sh again."
        exit 1
    fi

    piko::log::success "All verification checks passed!"
}

# main runs all verification checks.
# Arguments:
#   $@ - Unused
main() {
    piko::log::header "Piko Verification Suite"
    piko::log::info "Running all verification checks..."
    piko::log::footer

    run_all_checks
    print_summary
}

main "$@"
