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

# hack/test/profile.sh - Run profiling benchmarks for a Go package
#
# Runs CPU, memory, mutex, and block profiling on benchmarks and
# generates a comprehensive report.
#
# Usage:
#   ./hack/test/profile.sh <package_path> [benchmark_pattern] [test_subdir]
#
# Examples:
#   ./hack/test/profile.sh internal/ast
#   ./hack/test/profile.sh internal/ast "BenchmarkFullPipeline.*"
#   ./hack/test/profile.sh internal/ast "Benchmark.*" ./ast_test
#
# Arguments:
#   package_path      - Path to the package (e.g., internal/ast)
#   benchmark_pattern - Optional regex pattern for benchmarks (default: Benchmark.*)
#   test_subdir       - Optional subdirectory containing tests (default: .)

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the Go package to profile.
PACKAGE_PATH=""

# Regex pattern for benchmarks to run.
BENCHMARK_PATTERN=""

# Subdirectory containing tests.
TEST_SUBDIR=""

# Full path to the test package.
TEST_PKG_PATH=""

# Directory for output files.
OUTPUT_DIR=""

# Path to the performance report file.
REPORT_FILE=""

# Number of top entries to show in profiles.
TOP_N=40

# List of discovered benchmarks.
BENCHMARKS=""

# Types of profiles to run.
PROFILE_TYPES=("cpu" "mem" "mutex" "block")

# validate_args parses command-line arguments for profiling.
# Globals:
#   PACKAGE_PATH - Set
#   BENCHMARK_PATTERN - Set (default: Benchmark.*)
#   TEST_SUBDIR - Set (default: .)
#   TEST_PKG_PATH - Set
# Arguments:
#   $1 - Package path
#   $2 - Optional benchmark pattern
#   $3 - Optional test subdirectory
validate_args() {
    if [[ $# -eq 0 || "$1" == "-h" || "$1" == "--help" ]]; then
        piko::log::info "Usage: $0 <package_path> [benchmark_pattern] [test_subdir]"
        piko::log::blank
        piko::log::info "Arguments:"
        piko::log::detail "package_path       - Path to the package (e.g., internal/ast)"
        piko::log::detail "benchmark_pattern  - Optional regex pattern for benchmarks (default: Benchmark.*)"
        piko::log::detail "test_subdir        - Optional subdirectory containing tests (default: .)"
        piko::log::blank
        piko::log::info "Examples:"
        piko::log::detail "$0 internal/ast"
        piko::log::detail "$0 internal/ast \"BenchmarkFullPipeline.*\""
        piko::log::detail "$0 internal/ast \"Benchmark.*\" ./ast_test"
        exit 1
    fi

    PACKAGE_PATH="$1"
    BENCHMARK_PATTERN="${2:-Benchmark.*}"
    TEST_SUBDIR="${3:-.}"

    if [[ ! -d "$PACKAGE_PATH" ]]; then
        piko::log::fatal "Package path '$PACKAGE_PATH' does not exist or is not a directory."
    fi

    if [[ "$TEST_SUBDIR" == "." ]]; then
        TEST_PKG_PATH="$PACKAGE_PATH"
    else
        TEST_PKG_PATH="${PACKAGE_PATH}/${TEST_SUBDIR}"
    fi

    if [[ ! -d "$TEST_PKG_PATH" ]]; then
        piko::log::fatal "Test package path '$TEST_PKG_PATH' does not exist."
    fi
}

# setup_output creates the output directory and report file.
# Globals:
#   PACKAGE_PATH - Read
#   OUTPUT_DIR - Set
#   REPORT_FILE - Set
setup_output() {
    OUTPUT_DIR="${PACKAGE_PATH}/tmp"
    REPORT_FILE="${OUTPUT_DIR}/performance_report.txt"

    piko::util::ensure_dir "$OUTPUT_DIR"

    piko::log::header "Profiling: $PACKAGE_PATH"
    piko::log::info "Benchmark pattern: $BENCHMARK_PATTERN"
    piko::log::info "Test directory: $TEST_PKG_PATH"
    piko::log::info "Output: $REPORT_FILE"
    piko::log::footer

    : >"$REPORT_FILE"
}

# discover_benchmarks finds all benchmarks in the test package.
# Globals:
#   BENCHMARK_PATTERN - Read
#   TEST_PKG_PATH - Read
#   BENCHMARKS - Set
discover_benchmarks() {
    if [[ "$BENCHMARK_PATTERN" == "Benchmark.*" ]]; then
        piko::log::info "Discovering benchmarks..."
        BENCHMARKS=$(piko::go::list_benchmarks "$TEST_PKG_PATH")
        if [[ -z "$BENCHMARKS" ]]; then
            piko::log::warn "No benchmarks found in $TEST_PKG_PATH"
            exit 0
        fi
        piko::log::info "Found benchmarks:"
        while IFS= read -r bench; do
            piko::log::detail "- $bench"
        done <<< "$BENCHMARKS"
        piko::log::blank
    else
        BENCHMARKS="$BENCHMARK_PATTERN"
    fi
}

# run_profiling runs CPU, memory, mutex and block profiles.
# Globals:
#   BENCHMARKS - Read
#   OUTPUT_DIR - Read
#   REPORT_FILE - Read
#   PROFILE_TYPES - Read
run_profiling() {
    for bench in $BENCHMARKS; do
        if [[ ! "$bench" =~ ^Benchmark ]]; then
            continue
        fi

        piko::log::header "Benchmark: $bench"

        for profile_type in "${PROFILE_TYPES[@]}"; do
            local profile_out="${OUTPUT_DIR}/${profile_type}.out"

            if piko::go::run_profile "$TEST_PKG_PATH" "$bench" "$profile_type" "$profile_out" "$REPORT_FILE" "$TOP_N"; then
                piko::log::success "$profile_type profile complete"
            else
                piko::log::warn "$profile_type profile failed"
            fi
        done

        piko::log::footer
    done
}

# print_summary displays profiling completion message.
# Globals:
#   REPORT_FILE - Read
#   PACKAGE_PATH - Read
print_summary() {
    {
        piko::log::info "========================================================================"
        piko::log::info "Performance report generated at: $(date)"
        piko::log::info "Package: $PACKAGE_PATH"
        piko::log::info "========================================================================"
    } >>"$REPORT_FILE"

    piko::log::success "Profiling complete!"
    piko::log::info "Full report: $REPORT_FILE"
    piko::log::blank
    piko::log::info "To view the report:"
    piko::log::detail "cat $REPORT_FILE"
    piko::log::blank
    piko::log::info "To re-run:"
    piko::log::detail "./hack/test/profile.sh $PACKAGE_PATH \"$BENCHMARK_PATTERN\""
}

# main runs profiling on the specified package.
# Arguments:
#   $1 - Package path
#   $2 - Optional benchmark pattern
#   $3 - Optional test subdirectory
main() {
    validate_args "$@"
    setup_output
    discover_benchmarks
    run_profiling
    print_summary
}

main "$@"
