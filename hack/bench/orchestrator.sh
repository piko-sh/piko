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

# hack/bench/orchestrator.sh - Benchmark the orchestrator and compare revisions
#
# Runs the orchestrator's layered benchmarks (micro, meso and macro), compares two
# revisions with benchstat, profiles a benchmark, or reports heap escapes in the hot path.
#
# Usage:
#   ./hack/bench/orchestrator.sh baseline [output-file] [ref]
#   ./hack/bench/orchestrator.sh compare <base-ref> [head-ref]
#   ./hack/bench/orchestrator.sh profile <benchmark-regex> [package]
#   ./hack/bench/orchestrator.sh escape
#   ./hack/bench/orchestrator.sh gate <benchstat.csv>
#
# A ref of "." means the working tree. compare runs base and head in alternating rounds
# so drift on the machine affects both equally. Both sides run the working tree's
# benchmark harness, so they are measured the same way; set BENCH_OVERLAY=missing to keep
# a revision's own harness where it has one, or none to never overlay.
#
# gate checks benchstat.csv for significant allocation or timing regressions.
# BENCH_GATE_FAIL_PCT and BENCH_GATE_P set the timing failure threshold.
# BENCH_GATE_WARN_PCT sets the timing warning threshold; increased B/op also warns.
#
# Environment:
#   BENCH_COUNT    - rounds per benchmark (default: 6; benchstat needs at least 6)
#   BENCH_FILTER   - regex selecting benchmarks (default: every orchestrator benchmark)
#   BENCH_TIME     - -benchtime value (default: 1s)
#   BENCH_TAGS     - build tags, e.g. "bench" for the no-op logger (default: none)
#   BENCH_OUT_DIR  - where results are written (default: .bench/orchestrator)
#   BENCH_OVERLAY  - always, missing or none: when to copy the harness into a revision
#                    (default: always)
#   BENCH_EXTRA_OVERLAY - extra space-separated paths copied into a revision with the
#                    harness, for fixes a measurement depends on (default: none)
#   PIKO_BENCH_WAL_DIR - where write-ahead logs and databases are created; keep it on a
#                    disk-backed filesystem (default: .bench/wal)
#   BENCH_GATE_WARN_PCT - sec/op rise that warns (default: 10)
#   BENCH_GATE_FAIL_PCT - sec/op rise that fails when significant (default: 20)
#   BENCH_GATE_P   - p-value below which a sec/op rise counts (default: 0.01)

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Pinned benchstat; bump deliberately so comparisons stay reproducible.
BENCHSTAT="golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e"

# Packages holding orchestrator benchmarks, relative to the module that owns them.
ROOT_PACKAGES=(
    ./internal/orchestrator/orchestrator_test_bench/
    ./internal/orchestrator/orchestrator_adapters/
    ./internal/orchestrator/orchestrator_dal/otter/
    ./wdk/events/events_provider_gochannel/
)

# Module directory and packages for the SQL benchmarks, which live in their own module.
SQL_MODULE="tests/benchmarks/orchestrator"

# Paths of the benchmark harness, overlaid onto revisions that predate it.
HARNESS_PATHS=(
    internal/orchestrator/orchestrator_test_bench
    internal/orchestrator/orchestrator_dal/storebench
    internal/testutil/allocbudget
    internal/orchestrator/orchestrator_adapters/hotpath_bench_test.go
    internal/orchestrator/orchestrator_dal/otter/index_bench_test.go
    wdk/events/events_provider_gochannel/bounded_bench_test.go
    tests/benchmarks/orchestrator/service_test.go
    tests/benchmarks/orchestrator/artefact_compiler_test.go
)

# Default regex selecting every orchestrator benchmark.
DEFAULT_FILTER='^Benchmark(Lifecycle|ServiceDispatchWait|Throughput|Store|HotPathLog|Envelope|BusPublishDeliver|BoundedPublishDeliver|TimeIndex|TagIndexChurn|Service_|CompilerExecutor_)'

BENCH_COUNT="${BENCH_COUNT:-6}"
BENCH_FILTER="${BENCH_FILTER:-$DEFAULT_FILTER}"
BENCH_TIME="${BENCH_TIME:-1s}"
BENCH_TAGS="${BENCH_TAGS:-}"
BENCH_OUT_DIR="${BENCH_OUT_DIR:-${PIKO_ROOT}/.bench/orchestrator}"
BENCH_OVERLAY="${BENCH_OVERLAY:-always}"
BENCH_EXTRA_OVERLAY="${BENCH_EXTRA_OVERLAY:-}"
export PIKO_BENCH_WAL_DIR="${PIKO_BENCH_WAL_DIR:-${PIKO_ROOT}/.bench/wal}"

# run_benchmarks runs one round of every selected benchmark in a source tree.
# Globals:
#   ROOT_PACKAGES, SQL_MODULE, BENCH_FILTER, BENCH_TIME, BENCH_TAGS - Read
# Arguments:
#   $1 - Root of the source tree
#   $2 - Number of rounds (-count)
# Outputs:
#   Benchmark results on stdout
run_benchmarks() {
    local tree="$1"
    local count="$2"
    local tag_args=()
    if [[ -n "$BENCH_TAGS" ]]; then
        tag_args=(-tags "$BENCH_TAGS")
    fi

    (cd "$tree" && go test -run '^$' -bench "$BENCH_FILTER" -benchtime "$BENCH_TIME" \
        -count "$count" -timeout 60m "${tag_args[@]}" "${ROOT_PACKAGES[@]}")
    # The SQL benchmarks log through the default logger; keep it quiet so log lines do not
    # split benchmark result lines.
    (cd "${tree}/${SQL_MODULE}" && PIKO_LOG_LEVEL=error go test -run '^$' -bench "$BENCH_FILTER" \
        -benchtime "$BENCH_TIME" -count "$count" -timeout 60m "${tag_args[@]}" ./...)
}

# describe_machine prints the context benchstat needs to interpret a result file.
# Arguments:
#   $1 - Label of the benchmarked revision
# Outputs:
#   Key-value lines on stdout
describe_machine() {
    echo "revision: $1"
    echo "date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "go: $(go env GOVERSION)"
    echo "walfs: $(findmnt -no FSTYPE -T "$PIKO_BENCH_WAL_DIR" 2>/dev/null || echo unknown)"
    echo "tags: ${BENCH_TAGS:-none}"
}

# cmd_baseline records every selected benchmark for one revision.
# Arguments:
#   $1 - Optional output file
#   $2 - Ref to benchmark (default: the working tree)
cmd_baseline() {
    mkdir -p "$BENCH_OUT_DIR" "$PIKO_BENCH_WAL_DIR"
    local ref="${2:-.}"
    local label
    if [[ "$ref" == "." ]]; then
        label="$(git -C "$PIKO_ROOT" rev-parse --short HEAD)$(git -C "$PIKO_ROOT" diff --quiet || echo '-dirty')"
    else
        label="$(git -C "$PIKO_ROOT" rev-parse --short "$ref")"
    fi
    local out="${1:-${BENCH_OUT_DIR}/baseline-${label}.txt}"

    local scratch tree
    scratch="$(mktemp -d "${BENCH_OUT_DIR}/baseline-XXXXXX")"
    tree="$(prepare_tree "$ref" "${scratch}/tree")"
    if [[ "$ref" != "." ]]; then
        # shellcheck disable=SC2064 # expand now so the trap removes this worktree
        trap "git -C '$PIKO_ROOT' worktree remove --force '${scratch}/tree' 2>/dev/null; rm -rf '${scratch}'" EXIT
    fi

    piko::log::header "Orchestrator baseline for ${label}"
    piko::log::info "Writing ${out}"
    {
        describe_machine "$label"
        run_benchmarks "$tree" "$BENCH_COUNT"
    } | tee "$out"
    piko::log::footer
    piko::log::success "Baseline written to ${out}"
}

# prepare_tree checks a ref out into a temporary worktree and overlays the benchmark
# harness from the working tree as BENCH_OVERLAY selects.
# Globals:
#   HARNESS_PATHS, BENCH_OVERLAY, BENCH_EXTRA_OVERLAY - Read
# Arguments:
#   $1 - Ref, or "." for the working tree
#   $2 - Directory for the worktree
# Outputs:
#   The tree's root on stdout
prepare_tree() {
    local ref="$1"
    local dir="$2"
    if [[ "$ref" == "." ]]; then
        echo "$PIKO_ROOT"
        return
    fi

    git -C "$PIKO_ROOT" worktree add --detach --quiet "$dir" "$ref" >&2
    local path
    local overlay_paths=("${HARNESS_PATHS[@]}")
    if [[ -n "$BENCH_EXTRA_OVERLAY" ]]; then
        local extra
        read -r -a extra <<< "$BENCH_EXTRA_OVERLAY"
        overlay_paths+=("${extra[@]}")
    fi
    for path in "${overlay_paths[@]}"; do
        [[ -e "${PIKO_ROOT}/${path}" ]] || continue
        case "$BENCH_OVERLAY" in
            none) continue ;;
            missing) [[ -e "${dir}/${path}" ]] && continue ;;
        esac
        rm -rf "${dir:?}/${path}"
        mkdir -p "$(dirname "${dir}/${path}")"
        cp -R "${PIKO_ROOT}/${path}" "${dir}/${path}"
        piko::log::detail "Overlaid ${path} onto ${ref}" >&2
    done
    echo "$dir"
}

# cmd_compare benchmarks two refs in alternating rounds and compares them.
# Arguments:
#   $1 - Base ref
#   $2 - Head ref (default: the working tree)
cmd_compare() {
    local base_ref="${1:?compare needs a base ref}"
    local head_ref="${2:-.}"
    mkdir -p "$BENCH_OUT_DIR" "$PIKO_BENCH_WAL_DIR"

    local scratch
    scratch="$(mktemp -d "${BENCH_OUT_DIR}/compare-XXXXXX")"
    local base_tree head_tree
    base_tree="$(prepare_tree "$base_ref" "${scratch}/base")"
    head_tree="$(prepare_tree "$head_ref" "${scratch}/head")"
    # shellcheck disable=SC2064 # expand now so the trap removes these worktrees
    trap "git -C '$PIKO_ROOT' worktree remove --force '${scratch}/base' 2>/dev/null; git -C '$PIKO_ROOT' worktree remove --force '${scratch}/head' 2>/dev/null" EXIT

    local base_out="${scratch}/base.txt"
    local head_out="${scratch}/head.txt"

    piko::log::header "Comparing ${base_ref} (base) with ${head_ref} (head)"
    local round
    for ((round = 1; round <= BENCH_COUNT; round++)); do
        piko::log::step "$round" "$BENCH_COUNT" "Round ${round}"
        run_benchmarks "$base_tree" 1 >>"$base_out"
        run_benchmarks "$head_tree" 1 >>"$head_out"
    done

    piko::log::footer
    go run "$BENCHSTAT" -table .name "base=${base_out}" "head=${head_out}" | tee "${scratch}/benchstat.txt"
    go run "$BENCHSTAT" -format csv "base=${base_out}" "head=${head_out}" >"${scratch}/benchstat.csv" 2>/dev/null
    piko::log::success "Results in ${scratch}"
    echo "${scratch}/benchstat.csv" >"${BENCH_OUT_DIR}/last-compare-csv"
}

# cmd_gate fails on a regression in a benchstat comparison.
# Globals:
#   BENCH_GATE_WARN_PCT, BENCH_GATE_FAIL_PCT, BENCH_GATE_P - Read
# Arguments:
#   $1 - benchstat CSV (default: the last compare's)
# Outputs:
#   One line per regression on stdout; GitHub annotations when run in Actions
# Returns:
#   1 when a regression fails the gate
cmd_gate() {
    local csv="${1:-$(cat "${BENCH_OUT_DIR}/last-compare-csv" 2>/dev/null)}"
    [[ -f "$csv" ]] || piko::log::fatal "gate needs a benchstat CSV; run compare first"

    awk -F, -v warn_pct="${BENCH_GATE_WARN_PCT:-10}" -v fail_pct="${BENCH_GATE_FAIL_PCT:-20}"         -v max_p="${BENCH_GATE_P:-0.01}" -v gha="${GITHUB_ACTIONS:-}" '
        function report(level, message) {
            print level ": " message
            if (gha != "") print "::" (level == "FAIL" ? "error" : "warning") "::" message
        }
        $1 == "" && $6 == "vs base" { unit = $2; next }
        $1 == "" || $1 == "geomean" || $6 == "" || $6 == "~" { next }
        {
            delta = $6; sub(/%$/, "", delta); delta += 0
            p = $7; sub(/^p=/, "", p); sub(/ .*/, "", p); p += 0
            if (delta <= 0) next
            if (unit == "allocs/op") {
                report("FAIL", $1 " allocs/op rose " $6 " (" $2 " -> " $4 ")"); failed = 1
            } else if (unit == "B/op") {
                report("WARN", $1 " B/op rose " $6 " (" $2 " -> " $4 ")")
            } else if (unit == "sec/op" && delta > fail_pct && p < max_p) {
                report("FAIL", $1 " sec/op rose " $6 " (" $7 ")"); failed = 1
            } else if (unit == "sec/op" && delta > warn_pct) {
                report("WARN", $1 " sec/op rose " $6 " (" $7 ")")
            }
        }
        END { exit failed }
    ' "$csv"
}

# cmd_profile records CPU and allocation profiles for one benchmark.
# Arguments:
#   $1 - Benchmark regex
#   $2 - Package (default: the orchestrator rig)
cmd_profile() {
    local bench="${1:?profile needs a benchmark regex}"
    local pkg="${2:-./internal/orchestrator/orchestrator_test_bench/}"
    mkdir -p "$BENCH_OUT_DIR" "$PIKO_BENCH_WAL_DIR"

    local report
    report="${BENCH_OUT_DIR}/profile-$(date -u +%Y%m%dT%H%M%SZ).txt"
    piko::log::header "Profiling ${bench}"
    piko::go::run_profile "$pkg" "$bench" cpu "${BENCH_OUT_DIR}/cpu.pprof" "$report" 40 "$BENCH_TAGS"
    piko::go::run_profile "$pkg" "$bench" mem "${BENCH_OUT_DIR}/mem.pprof" "$report" 40 "$BENCH_TAGS" 1
    piko::log::footer
    piko::log::success "Profile report written to ${report}"
}

# cmd_escape reports values the compiler moves to the heap in the orchestrator's hot path.
cmd_escape() {
    piko::log::header "Heap escapes in the orchestrator hot path"
    go build -gcflags='-m=1' \
        ./internal/orchestrator/orchestrator_domain/ \
        ./internal/orchestrator/orchestrator_adapters/ \
        ./internal/orchestrator/orchestrator_dal/otter/ 2>&1 |
        grep -E 'escapes to heap|moved to heap' |
        grep -E 'task_processing_core|execution_claim|driven_taskdispatcher_watermill|driven_eventbus_watermill|otter\.go|delayed_publisher' |
        sort | uniq -c | sort -rn
    piko::log::footer
}

# main dispatches the subcommand.
main() {
    local command="${1:-}"
    shift || true
    case "$command" in
        baseline) cmd_baseline "$@" ;;
        compare) cmd_compare "$@" ;;
        profile) cmd_profile "$@" ;;
        escape) cmd_escape ;;
        gate) cmd_gate "$@" ;;
        *)
            piko::log::info "Usage: $0 baseline|compare|profile|escape|gate [args]"
            exit 1
            ;;
    esac
}

main "$@"
