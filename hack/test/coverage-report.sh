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

# hack/test/coverage-report.sh - Generate markdown test coverage report for all packages
#
# Usage:
#   ./hack/test/coverage-report.sh [output_file]
#
# Output file defaults to COVERAGE_REPORT.md

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the output markdown report file.
OUTPUT_FILE=""

# Temporary file for coverage data.
TEMP_FILE=""

# run_coverage_analysis runs go test with coverage on all internal packages.
# Globals:
#   TEMP_FILE - Read (output written here)
run_coverage_analysis() {
    piko::log::info "Running test coverage analysis..."
    go test -short -cover ./internal/... 2>&1 | tee "$TEMP_FILE"
}

# generate_report creates a markdown report from coverage data.
# Globals:
#   TEMP_FILE - Read
#   OUTPUT_FILE - Read (report written here)
generate_report() {
    piko::log::blank
    piko::log::info "Generating report..."

    grep -E '(coverage:|no test files)' "$TEMP_FILE" | awk '
{
    if (/\[no test files\]/) {
        pkg = $2
        gsub(/github.com\/politepixels\/piko\//, "", pkg)
        print "0.0|" pkg "|no test files"
        next
    }

    if (/\[no statements\]/) {
        pkg = $2
        gsub(/github.com\/politepixels\/piko\//, "", pkg)
        print "0.0|" pkg "|no statements (test helper)"
        next
    }

    if (/coverage:/) {
        for (i = 1; i <= NF; i++) {
            if ($i ~ /github.com/) {
                pkg = $i
                gsub(/github.com\/politepixels\/piko\//, "", pkg)
            }
            if ($i ~ /[0-9]+\.[0-9]+%/) {
                pct = $i
                gsub(/%/, "", pct)
                print pct "|" pkg "|" pct "%"
            }
        }
    }
}' | sort -t'|' -k1 -n | awk -F'|' '
BEGIN {
    print "# Test Coverage Report for internal/"
    print ""
    print "Generated: " strftime("%Y-%m-%d %H:%M:%S")
    print ""
    print "## Summary Statistics"
    print ""
}
{
    packages[NR] = $2
    coverage[NR] = $3
    numeric[NR] = $1
    total += $1
    count++

    if ($1 == 0) zero_count++
    else if ($1 < 25) low_count++
    else if ($1 < 50) medium_low_count++
    else if ($1 < 75) medium_count++
    else if ($1 < 90) good_count++
    else excellent_count++
}
END {
    avg = (count > 0) ? total / count : 0

    print "- **Total packages analysed:** " count
    print "- **Average coverage:** " sprintf("%.1f%%", avg)
    print "- **0% coverage:** " zero_count " packages"
    print "- **1-24% coverage:** " low_count " packages"
    print "- **25-49% coverage:** " medium_low_count " packages"
    print "- **50-74% coverage:** " medium_count " packages"
    print "- **75-89% coverage:** " good_count " packages"
    print "- **90-100% coverage:** " excellent_count " packages"
    print ""
    print "---"
    print ""
    print "## Coverage by Package (Sorted: Lowest to Highest)"
    print ""
    print "| Coverage | Package |"
    print "|----------|---------|"

    for (i = 1; i <= count; i++) {
        printf "| %s | `%s` |\n", coverage[i], packages[i]
    }

    print ""
    print "---"
    print ""
    print "## Critical Packages Needing Attention (0% Coverage)"
    print ""
    print "These packages have no test coverage and should be prioritised:"
    print ""

    for (i = 1; i <= count; i++) {
        if (numeric[i] == 0 && coverage[i] !~ /no statements|no test files/) {
            print "- `" packages[i] "`"
        }
    }

    print ""
    print "## Packages Without Test Files"
    print ""

    for (i = 1; i <= count; i++) {
        if (coverage[i] ~ /no test files/) {
            print "- `" packages[i] "`"
        }
    }

    print ""
    print "## Test Helper Packages (No Statements to Cover)"
    print ""

    for (i = 1; i <= count; i++) {
        if (coverage[i] ~ /no statements/) {
            print "- `" packages[i] "`"
        }
    }
}
' >"$OUTPUT_FILE"

    piko::log::blank
    piko::log::success "Coverage report written to: $OUTPUT_FILE"
    piko::log::blank

    head -20 "$OUTPUT_FILE"
}

# main generates a coverage report for all internal packages.
# Arguments:
#   $1 - Optional output file path (default: COVERAGE_REPORT.md)
main() {
    OUTPUT_FILE="${1:-COVERAGE_REPORT.md}"
    TEMP_FILE=$(mktemp)
    trap 'rm -f "$TEMP_FILE"' EXIT

    run_coverage_analysis
    generate_report
}

main "$@"
