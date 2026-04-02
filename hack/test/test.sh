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

# hack/test/test.sh - Unified test runner for the Piko project
#
# This script is called by Makefile targets:
#   make test             Quick Go unit tests (quiet output)
#   make test-go          All Go tests including integration
#   make test-frontend    Frontend core tests
#   make test-vscode      VSCode plugin tests
#   make test-idea        IntelliJ plugin tests
#   make test-all         Run ALL tests (the "is everything OK" check)

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

VSCODE_DIR="${PIKO_ROOT}/plugins/vscode"
IDEA_DIR="${PIKO_ROOT}/plugins/idea"

FAILED=()

# run_test executes a test suite and tracks results.
# Globals:
#   FAILED - Modified with failed test names
# Arguments:
#   $1 - Display name for the test suite
#   $2 - Function to execute
run_test() {
    local name="$1"
    local func="$2"

    piko::log::header "$name"

    if "$func"; then
        piko::log::success "$name passed"
    else
        piko::log::error "$name failed"
        FAILED+=("$name")
    fi

    piko::log::footer
}

# test_go_quick runs Go unit tests with -short flag and quiet output.
test_go_quick() {
    piko::log::header "Go Unit Tests (quick)"

    local output
    local exit_code=0

    output=$(go test -short -race -count=1 piko.sh/piko/... 2>&1) || exit_code=$?

    if [[ $exit_code -ne 0 ]]; then
        echo "$output" | grep -E '^(FAIL|---|\s+Error|panic:)' || echo "$output"
        return 1
    fi

    piko::log::success "All Go unit tests passed"
    return 0
}

# test_go_sum runs Go unit tests using gotestsum for better output.
test_go_sum() {
    if ! command -v gotestsum &> /dev/null; then
        piko::log::error "gotestsum not found"
        piko::log::blank
        piko::log::info "Install with: go install gotest.tools/gotestsum@latest"
        return 1
    fi

    local packages
    packages=$(go list piko.sh/piko/... | grep -v esbuild)

    # shellcheck disable=SC2086
    gotestsum --format short -- -short -cover $packages
}

# test_go runs all Go tests including integration tests.
test_go() {
    go test -tags=vips,ffmpeg -p 2 -count=1 -timeout=40m piko.sh/piko/...
}

# check_qemu_binfmt warns if QEMU binfmt is not registered for cross-arch tests.
# The cross-arch integration tests run arm64 binaries via Docker+QEMU, which
# requires binfmt_misc handlers to be registered in the host kernel.
#
# Only relevant on Linux - macOS and Windows use Docker Desktop which runs a
# Linux VM with binfmt already registered.
check_qemu_binfmt() {
    if [[ "$(uname -s)" != "Linux" ]]; then
        return
    fi

    if [[ "$(uname -m)" != "x86_64" ]]; then
        return
    fi

    if [[ ! -d /proc/sys/fs/binfmt_misc ]]; then
        piko::log::warn "QEMU binfmt not available - cross-arch tests (TestCrossArch/arm64) will fail"
        piko::log::detail "  Run: docker run --rm --privileged multiarch/qemu-user-static --reset -p yes"
        piko::log::blank
        return
    fi

    if ! grep -ql aarch64 /proc/sys/fs/binfmt_misc/* 2>/dev/null; then
        piko::log::warn "QEMU binfmt not registered for arm64 - cross-arch tests (TestCrossArch/arm64) will fail"
        piko::log::detail "  Run: docker run --rm --privileged multiarch/qemu-user-static --reset -p yes"
        piko::log::blank
    fi
}

# test_go_full runs all Go tests including integration tests.
test_go_full() {
    check_qemu_binfmt
    go test -tags=integration,vips,ffmpeg -p 2 -count=1 -timeout=60m piko.sh/piko/...
}

# test_frontend runs vitest tests across frontend packages that have spec files.
# Globals:
#   PIKO_ROOT - Read
#   FRONTEND_TEST_DIRS - Read
test_frontend() {
    local failed=0

    for dir in "${FRONTEND_TEST_DIRS[@]}"; do
        local abs_dir="${PIKO_ROOT}/${dir}"
        if [[ ! -d "$abs_dir" ]]; then
            piko::log::warn "Frontend directory not found: ${dir}"
            continue
        fi

        local spec_count
        spec_count=$(find "$abs_dir/src" -name "*.spec.ts" 2>/dev/null | head -1 | wc -l)
        if [[ "$spec_count" -eq 0 ]]; then
            continue
        fi

        piko::log::info "Testing ${dir}..."
        cd "$abs_dir" || continue

        if [[ ! -x "node_modules/.bin/vitest" ]]; then
            piko::log::info "Installing dependencies..."
            npm install >/dev/null 2>&1
        fi

        if ! npm test; then
            failed=1
        fi
    done

    return "$failed"
}

# test_vscode runs VSCode plugin vitest tests.
# Globals:
#   VSCODE_DIR - Read
test_vscode() {
    if [[ ! -d "$VSCODE_DIR" ]]; then
        piko::log::error "VSCode plugin directory not found: $VSCODE_DIR"
        return 1
    fi

    cd "$VSCODE_DIR" || return 1

    if [[ ! -d "node_modules" ]]; then
        piko::log::info "Installing dependencies..."
        npm install >/dev/null 2>&1
    fi

    npm test
}

# test_idea runs IntelliJ plugin gradle tests.
# Globals:
#   IDEA_DIR - Read
test_idea() {
    if [[ ! -d "$IDEA_DIR" ]]; then
        piko::log::error "IntelliJ plugin directory not found: $IDEA_DIR"
        return 1
    fi

    cd "$IDEA_DIR" || return 1

    ./gradlew test
}

# Packages to exclude from coverage (non-Go files, vendored, generated).
COVERAGE_EXCLUDE_PATTERN='(render_templates|esbuild|/schema$|_schema$|_schema_gen|/gen$|/gen/|/db$|_db$|/symbols$|_test/|/testutil|_mock$|_mock/|^piko\.sh/piko/tests/|/clitest$|/apitest$|/pikotest|_dal/mock)'

# deduplicate_coverage_profile merges duplicate blocks in a coverage profile.
#
# When -coverpkg lists N packages and M test binaries exist, every block appears
# up to M times in the profile (once per binary). Most entries have count=0
# because that binary's tests did not exercise the block. go tool cover -func
# does not correctly merge these duplicates, so the inflated zero-count entries
# dilute the total and produce an artificially low percentage.
#
# This function keeps only the entry with the highest count for each unique block,
# producing a clean profile suitable for accurate coverage calculation.
#
# Arguments:
#   $1 - Input coverage profile path
#   $2 - Output deduplicated profile path
deduplicate_coverage_profile() {
    local input="$1"
    local output="$2"

    awk 'NR==1 { print; next }
    {
        block = $1
        stmts = $2
        count = $3 + 0
        if (count > max_count[block] + 0) {
            max_count[block] = count
        }
        stmt_count[block] = stmts
    }
    END {
        for (b in stmt_count) {
            printf "%s %s %d\n", b, stmt_count[b], max_count[b]
        }
    }' "$input" > "$output"
}

# README file path for badge updates.
README_FILE="${PIKO_ROOT}/README.md"

# test_coverage_total calculates whole-project coverage percentage (quick mode).
test_coverage_total() {
    run_coverage_total "quick" "-short"
}

# test_coverage_total_full calculates whole-project coverage including integration tests.
test_coverage_total_full() {
    run_coverage_total "full (with integration)" "-tags=integration"
}

# test_coverage_internal_total calculates coverage for internal/ packages only (quick mode).
test_coverage_internal_total() {
    run_coverage_total "internal only" "-short" "./internal/..."
}

# run_coverage_total runs coverage with specified flags.
# Globals:
#   COVERAGE_EXCLUDE_PATTERN - Read
# Arguments:
#   $1 - Mode description (e.g., "quick", "full")
#   $2 - Additional go test flags
#   $3 - Package scope (optional, defaults to "piko.sh/piko/...")
run_coverage_total() {
    local mode="$1"
    local flags="$2"
    local scope="${3:-piko.sh/piko/...}"
    local coverpkgs

    local coverage_file="/tmp/piko-coverage-total.out"
    local deduped_file="/tmp/piko-coverage-total.deduped.out"
    local tests_passed=true

    rm -f "$coverage_file" "$deduped_file"
    go clean -cache -testcache
    trap 'rm -f /tmp/piko-coverage-total.out /tmp/piko-coverage-total.deduped.out' RETURN

    piko::log::header "Project Coverage ($mode)"
    piko::log::info "Calculating coverage for $scope (this may take a while)..."

    coverpkgs=$(go list "$scope" | grep -v -E "$COVERAGE_EXCLUDE_PATTERN" | paste -sd,)

    # shellcheck disable=SC2086
    if ! go test -coverprofile="$coverage_file" -coverpkg="$coverpkgs" -covermode=atomic $flags "$scope" > /dev/null 2>&1; then
        tests_passed=false
    fi

    if [[ ! -f "$coverage_file" ]]; then
        piko::log::error "Coverage profile was not generated"
        return 1
    fi

    deduplicate_coverage_profile "$coverage_file" "$deduped_file"

    local total
    total=$(go tool cover -func="$deduped_file" | grep "^total:" | awk '{print $NF}')

    piko::log::blank
    if [[ "$tests_passed" == "true" ]]; then
        piko::log::success "Total coverage: $total"
    else
        piko::log::warn "Total coverage: $total (some tests failed)"
        return 1
    fi
}

# test_all runs all test suites and reports results.
# Globals:
#   FAILED - Modified
test_all() {
    FAILED=()

    piko::log::header "Piko Test Suite"
    piko::log::info "Running all tests..."
    piko::log::footer

    run_test "Go Tests (with integration)" test_go_full
    run_test "Frontend Core Tests" test_frontend
    run_test "VSCode Plugin Tests" test_vscode
    run_test "IntelliJ Plugin Tests" test_idea

    print_summary
}

# print_summary displays test results.
# Globals:
#   FAILED - Read
print_summary() {
    local total=4
    local passed=$((total - ${#FAILED[@]}))

    piko::log::blank
    piko::log::header "Test Summary"

    piko::log::info "Total suites: $total"
    piko::log::info "Passed: $passed"
    piko::log::info "Failed: ${#FAILED[@]}"

    if [[ ${#FAILED[@]} -gt 0 ]]; then
        piko::log::blank
        piko::log::error "The following test suites failed:"
        for suite in "${FAILED[@]}"; do
            piko::log::detail "- $suite"
        done
        piko::log::blank
        piko::log::info "Run individual suites with: make test-go, make test-frontend, etc."
        exit 1
    fi

    piko::log::blank
    piko::log::success "All tests passed!"
}

# get_coverage_color returns a shields.io color based on coverage percentage.
# Arguments:
#   $1 - Coverage percentage (numeric, e.g., 50.2)
# Outputs:
#   Writes color name to stdout
get_coverage_color() {
    local pct="$1"
    local int_pct="${pct%.*}"

    if [[ "$int_pct" -lt 50 ]]; then
        echo "red"
    elif [[ "$int_pct" -lt 70 ]]; then
        echo "yellow"
    elif [[ "$int_pct" -lt 80 ]]; then
        echo "yellowgreen"
    elif [[ "$int_pct" -lt 90 ]]; then
        echo "green"
    else
        echo "brightgreen"
    fi
}

# update_readme_badge updates a coverage badge in the README.
# Globals:
#   README_FILE - Read
# Arguments:
#   $1 - Badge name (e.g., "Go_Coverage", "Frontend_Coverage")
#   $2 - Coverage value (e.g., "50%", "-")
#   $3 - Color (e.g., "yellow", "lightgrey")
update_readme_badge() {
    local badge_name="$1"
    local value="$2"
    local color="$3"

    local escaped_value
    escaped_value=$(echo "$value" | sed 's/%/%25/g')

    sed -i "s|${badge_name}-[^?]*?|${badge_name}-${escaped_value}-${color}?|g" "$README_FILE"
}

# get_go_coverage runs Go coverage and returns the percentage.
# Globals:
#   COVERAGE_EXCLUDE_PATTERN - Read
# Outputs:
#   Writes percentage (e.g., "50.2") to stdout, or empty on failure
get_go_coverage() {
    local coverpkgs
    local coverage_file="/tmp/piko-coverage-total.out"

    rm -f "$coverage_file"
    go clean -cache -testcache

    coverpkgs=$(go list piko.sh/piko/... | grep -v -E "$COVERAGE_EXCLUDE_PATTERN" | paste -sd,)

    if ! go test -coverprofile="$coverage_file" -coverpkg="$coverpkgs" -covermode=atomic -short piko.sh/piko/... > /dev/null 2>&1; then
        return 1
    fi

    local deduped_file="${coverage_file%.out}.deduped.out"
    deduplicate_coverage_profile "$coverage_file" "$deduped_file"

    go tool cover -func="$deduped_file" 2>/dev/null | grep "^total:" | awk '{print $NF}' | tr -d '%'
}

# FRONTEND_TEST_DIRS lists frontend packages included in coverage reporting.
# Each entry is a path relative to PIKO_ROOT.
FRONTEND_TEST_DIRS=(
    "frontend/core"
    "frontend/extensions/analytics"
    "frontend/extensions/animation"
    "frontend/extensions/components"
    "frontend/extensions/dev"
    "frontend/extensions/modals"
    "frontend/extensions/toasts"
)

# get_frontend_coverage runs tests across all frontend packages and returns
# combined coverage percentage. Uses json-summary reporter to extract
# statement counts, then computes a weighted average.
# Globals:
#   PIKO_ROOT - Read
#   FRONTEND_TEST_DIRS - Read
# Outputs:
#   Writes percentage to stdout, or empty on failure
get_frontend_coverage() {
    local total_covered=0
    local total_stmts=0

    for dir in "${FRONTEND_TEST_DIRS[@]}"; do
        local abs_dir="${PIKO_ROOT}/${dir}"
        if [[ ! -d "$abs_dir" ]]; then
            continue
        fi

        local spec_count
        spec_count=$(find "$abs_dir/src" -name "*.spec.ts" 2>/dev/null | head -1 | wc -l)
        if [[ "$spec_count" -eq 0 ]]; then
            continue
        fi

        cd "$abs_dir" || continue

        if [[ ! -x "node_modules/.bin/vitest" ]]; then
            npm install >/dev/null 2>&1
        fi

        ./node_modules/.bin/vitest run --coverage --coverage.reportOnFailure=true \
            --coverage.reporter=json-summary >/dev/null 2>&1

        local summary="${abs_dir}/coverage/coverage-summary.json"
        if [[ ! -f "$summary" ]]; then
            continue
        fi

        local covered total
        covered=$(python3 -c "import json; print(json.load(open('${summary}'))['total']['statements']['covered'])" 2>/dev/null)
        total=$(python3 -c "import json; print(json.load(open('${summary}'))['total']['statements']['total'])" 2>/dev/null)

        if [[ -n "$covered" ]] && [[ -n "$total" ]] && [[ "$total" -gt 0 ]]; then
            total_covered=$((total_covered + covered))
            total_stmts=$((total_stmts + total))
        fi
    done

    if [[ "$total_stmts" -gt 0 ]]; then
        python3 -c "print(round(${total_covered} / ${total_stmts} * 100, 1))"
    fi
}

# get_vscode_coverage runs VSCode plugin tests and returns coverage percentage.
# Globals:
#   VSCODE_DIR - Read
# Outputs:
#   Writes percentage to stdout, or empty on failure
get_vscode_coverage() {
    if [[ ! -d "$VSCODE_DIR" ]]; then
        return 1
    fi

    cd "$VSCODE_DIR" || return 1

    if [[ ! -d "node_modules" ]]; then
        npm install >/dev/null 2>&1
    fi

    local output
    output=$(npx vitest run --coverage --coverage.reportOnFailure=true --coverage.reporter=text 2>&1)

    echo "$output" | grep "^All files" | awk -F'|' '{gsub(/[[:space:]]/, "", $2); print $2}'
}

# get_idea_coverage runs IntelliJ plugin tests and returns coverage percentage.
# Globals:
#   IDEA_DIR - Read
# Outputs:
#   Writes percentage to stdout, or empty on failure
# Note: Requires JDK 21. Will fail silently if not found.
get_idea_coverage() {
    if [[ ! -d "$IDEA_DIR" ]]; then
        return 1
    fi

    local java_home
    java_home=$(piko::java::find_21)
    if [[ -z "$java_home" ]]; then
        return 1
    fi

    cd "$IDEA_DIR" || return 1

    if ! JAVA_HOME="$java_home" ./gradlew test koverXmlReport --quiet >/dev/null 2>&1; then
        return 1
    fi

    local report_file="${IDEA_DIR}/build/reports/kover/report.xml"
    if [[ -f "$report_file" ]]; then
        local line_counter
        line_counter=$(grep 'type="LINE"' "$report_file" | tail -1)
        if [[ -n "$line_counter" ]]; then
            local missed covered
            missed=$(echo "$line_counter" | grep -o 'missed="[0-9]*"' | grep -o '[0-9]*')
            covered=$(echo "$line_counter" | grep -o 'covered="[0-9]*"' | grep -o '[0-9]*')
            if [[ -n "$missed" ]] && [[ -n "$covered" ]]; then
                echo "$missed $covered" | awk '{total=$1+$2; if(total>0) printf "%.2f", ($2/total)*100; else print "0"}'
            fi
        fi
    fi
}

# update_badges updates all coverage badges in the README.
update_badges() {
    piko::log::header "Updating Coverage Badges"

    piko::log::info "Calculating Go coverage..."
    local go_pct
    go_pct=$(get_go_coverage)

    if [[ -n "$go_pct" ]]; then
        local go_color
        go_color=$(get_coverage_color "$go_pct")
        local go_display="${go_pct%.*}%"  # Round to integer for display
        update_readme_badge "Go_Coverage" "$go_display" "$go_color"
        piko::log::success "Go: ${go_display} (${go_color})"
    else
        piko::log::warn "Go: Failed to calculate coverage"
    fi

    piko::log::info "Calculating Frontend coverage..."
    local fe_pct
    fe_pct=$(get_frontend_coverage)

    if [[ -n "$fe_pct" ]] && [[ "$fe_pct" != "0" ]]; then
        local fe_color
        fe_color=$(get_coverage_color "$fe_pct")
        local fe_display="${fe_pct%.*}%"
        update_readme_badge "Frontend_Coverage" "$fe_display" "$fe_color"
        piko::log::success "Frontend: ${fe_display} (${fe_color})"
    else
        piko::log::warn "Frontend: Coverage not available"
        update_readme_badge "Frontend_Coverage" "-" "lightgrey"
    fi

    piko::log::info "Calculating VSCode plugin coverage..."
    local vsc_pct
    vsc_pct=$(get_vscode_coverage)

    if [[ -n "$vsc_pct" ]] && [[ "$vsc_pct" != "0" ]]; then
        local vsc_color
        vsc_color=$(get_coverage_color "$vsc_pct")
        local vsc_display="${vsc_pct%.*}%"
        update_readme_badge "VSCode_Plugin" "$vsc_display" "$vsc_color"
        piko::log::success "VSCode: ${vsc_display} (${vsc_color})"
    else
        piko::log::warn "VSCode: Coverage not available"
        update_readme_badge "VSCode_Plugin" "-" "lightgrey"
    fi

    piko::log::info "Calculating IntelliJ plugin coverage..."
    local idea_pct
    idea_pct=$(get_idea_coverage)

    if [[ -n "$idea_pct" ]] && [[ "$idea_pct" != "0" ]]; then
        local idea_color
        idea_color=$(get_coverage_color "$idea_pct")
        local idea_display="${idea_pct%.*}%"
        update_readme_badge "IntelliJ_Plugin" "$idea_display" "$idea_color"
        piko::log::success "IntelliJ: ${idea_display} (${idea_color})"
    else
        piko::log::warn "IntelliJ: Coverage not available"
        update_readme_badge "IntelliJ_Plugin" "-" "lightgrey"
    fi

    piko::log::blank
    piko::log::success "README badges updated!"
    piko::log::info "Review changes with: git diff README.md"
}

# print_usage displays help information.
print_usage() {
    cat <<EOF
Usage: $(basename "$0") <command>

Commands:
    quick                Run Go unit tests with quiet output (default)
    sum                  Run Go unit tests with gotestsum (better output)
    go                   Run all Go tests including integration
    frontend             Run frontend/core vitest tests
    vscode               Run VSCode plugin vitest tests
    idea                 Run IntelliJ plugin gradle tests
    all                  Run ALL test suites
    coverage-total           Show whole-project coverage percentage (quick)
    coverage-total-full      Show coverage including integration tests (slow)
    coverage-internal-total  Show internal/ packages coverage percentage (quick)
    update-badges            Update README.md coverage badges

Examples:
    $(basename "$0") quick                # Quick Go tests, only show failures
    $(basename "$0") all                  # Full test run, tells you if project is OK
    $(basename "$0") coverage-total       # Show "Project has X% coverage"
    $(basename "$0") coverage-total-full  # Comprehensive coverage (takes longer)
EOF
}

# main handles command dispatch.
# Arguments:
#   $1 - Command to execute
main() {
    local command="${1:-quick}"

    case "$command" in
        quick)
            test_go_quick
            ;;

        sum)
            test_go_sum
            ;;

        go)
            piko::log::header "Go Tests"
            test_go
            piko::log::success "All Go tests passed"
            ;;

        integration)
            piko::log::header "Go Tests (with integration)"
            test_go_full
            piko::log::success "All Go tests passed"
            ;;

        frontend)
            piko::log::header "Frontend Core Tests"
            test_frontend
            piko::log::success "Frontend tests passed"
            ;;

        vscode)
            piko::log::header "VSCode Plugin Tests"
            test_vscode
            piko::log::success "VSCode plugin tests passed"
            ;;

        idea)
            piko::log::header "IntelliJ Plugin Tests"
            test_idea
            piko::log::success "IntelliJ plugin tests passed"
            ;;

        coverage-total)
            test_coverage_total
            ;;

        coverage-total-full)
            test_coverage_total_full
            ;;

        coverage-internal-total)
            test_coverage_internal_total
            ;;

        update-badges)
            update_badges
            ;;

        all)
            test_all
            ;;

        help|--help|-h)
            print_usage
            ;;

        *)
            piko::log::error "Unknown command: $command"
            piko::log::blank
            print_usage
            exit 1
            ;;
    esac
}

main "$@"
