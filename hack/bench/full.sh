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

# hack/bench/full.sh - Full cross-language benchmark sweep for public reporting
#
# Runs each (runner, benchmark) cell in its own subprocess to eliminate
# cross-cell heap pollution. Appends one JSONL row per cell into
# tests/benchmarks/cross_language/results/full_sweep.jsonl. Pass --resume to
# skip cells that completed in a previous run.
#
# Usage:
#   make bench-full                                      Run the full sweep
#   make bench-full ARGS='--resume cpython_20_invert_binary_tree'
#                                                        Resume from a cell
#   make bench-full ARGS='--output /tmp/sweep.jsonl'     Custom output path
#   make bench-full ARGS='--runs 20 --warmup 3'          Override iteration counts
#   make bench-full ARGS='--dry-run'                     Print the plan, do nothing

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Default output JSONL file. Appended, never truncated automatically.
DEFAULT_OUTPUT_FILE="${PIKO_ROOT}/tests/benchmarks/cross_language/results/full_sweep.jsonl"

# Default measured-run count per cell.
DEFAULT_RUN_COUNT=10

# Default warmup-run count per cell. Discarded before any measurement to let
# JITs and lazy initialisers settle.
DEFAULT_WARMUP=2

# Default per-cell timeout.
DEFAULT_CELL_TIMEOUT=900

# GOGC value used by in-process Go runners (piko, scriggo, tengo, go).
GOGC_GO_RUNNER=10000

# GOGC value used by yaegi. Left near the Go default because every
# interpreted value is a reflect.Value boxed on the heap.
GOGC_YAEGI=100

# LANG_BENCH_LIST drives the sweep, one entry per (runner, benchmark) cell in
# the form "<runner>:<benchmark_name>".
LANG_BENCH_LIST=(
    "piko:00_smoke_addition"
    "piko:01_fib_iterative"
    "piko:02_word_frequency_1mb"
    "piko:03_levenshtein_1k_pairs"
    "piko:04_mini_json_parse_100kb"
    "piko:05_expression_eval_10k"
    "piko:06_lru_cache_100k_ops"
    "piko:07_dijkstra_10k_nodes"
    "piko:08_sudoku_solve_100"
    "piko:09_game_of_life_200x200_1k_gens"
    "piko:10_markov_text_gen_10k_words"
    "piko:11_trie_50k_words"
    "piko:12_brainfuck_mandelbrot"
    "piko:13_polymorphic_ast_eval_500k"
    "piko:14_mandelbrot_fp_200x200"
    "piko:15_open_addressing_hashmap_100k"
    "piko:16_parallel_word_count_montecristo"
    "piko:17_closures_pipeline"
    "piko:18_generic_numeric_pipeline"
    "piko:19_type_switches"
    "piko:20_invert_binary_tree"
    "piko:21_dense_layer"
    "piko:22_nbody_simulation"
    "piko:23_native_sha256_per_line"

    "go:00_smoke_addition"
    "go:01_fib_iterative"
    "go:02_word_frequency_1mb"
    "go:03_levenshtein_1k_pairs"
    "go:04_mini_json_parse_100kb"
    "go:05_expression_eval_10k"
    "go:06_lru_cache_100k_ops"
    "go:07_dijkstra_10k_nodes"
    "go:08_sudoku_solve_100"
    "go:09_game_of_life_200x200_1k_gens"
    "go:10_markov_text_gen_10k_words"
    "go:11_trie_50k_words"
    "go:12_brainfuck_mandelbrot"
    "go:13_polymorphic_ast_eval_500k"
    "go:14_mandelbrot_fp_200x200"
    "go:15_open_addressing_hashmap_100k"
    "go:16_parallel_word_count_montecristo"
    "go:17_closures_pipeline"
    "go:18_generic_numeric_pipeline"
    "go:19_type_switches"
    "go:20_invert_binary_tree"
    "go:21_dense_layer"
    "go:22_nbody_simulation"
    "go:23_native_sha256_per_line"

    "cpython:00_smoke_addition"
    "cpython:01_fib_iterative"
    "cpython:02_word_frequency_1mb"
    "cpython:03_levenshtein_1k_pairs"
    "cpython:04_mini_json_parse_100kb"
    "cpython:05_expression_eval_10k"
    "cpython:06_lru_cache_100k_ops"
    "cpython:07_dijkstra_10k_nodes"
    "cpython:08_sudoku_solve_100"
    "cpython:09_game_of_life_200x200_1k_gens"
    "cpython:10_markov_text_gen_10k_words"
    "cpython:11_trie_50k_words"
    "cpython:12_brainfuck_mandelbrot"
    "cpython:13_polymorphic_ast_eval_500k"
    "cpython:14_mandelbrot_fp_200x200"
    "cpython:15_open_addressing_hashmap_100k"
    "cpython:16_parallel_word_count_montecristo"
    "cpython:17_closures_pipeline"
    "cpython:18_generic_numeric_pipeline"
    "cpython:19_type_switches"
    "cpython:20_invert_binary_tree"
    "cpython:21_dense_layer"
    "cpython:22_nbody_simulation"
    "cpython:23_native_sha256_per_line"

    "pypy:00_smoke_addition"
    "pypy:01_fib_iterative"
    "pypy:02_word_frequency_1mb"
    "pypy:03_levenshtein_1k_pairs"
    "pypy:04_mini_json_parse_100kb"
    "pypy:05_expression_eval_10k"
    "pypy:06_lru_cache_100k_ops"
    "pypy:07_dijkstra_10k_nodes"
    "pypy:08_sudoku_solve_100"
    "pypy:09_game_of_life_200x200_1k_gens"
    "pypy:10_markov_text_gen_10k_words"
    "pypy:11_trie_50k_words"
    "pypy:12_brainfuck_mandelbrot"
    "pypy:13_polymorphic_ast_eval_500k"
    "pypy:14_mandelbrot_fp_200x200"
    "pypy:15_open_addressing_hashmap_100k"
    "pypy:16_parallel_word_count_montecristo"
    "pypy:17_closures_pipeline"
    "pypy:18_generic_numeric_pipeline"
    "pypy:19_type_switches"
    "pypy:20_invert_binary_tree"
    "pypy:21_dense_layer"
    "pypy:22_nbody_simulation"
    "pypy:23_native_sha256_per_line"

    "yaegi:00_smoke_addition"
    "yaegi:01_fib_iterative"
    "yaegi:02_word_frequency_1mb"
    "yaegi:03_levenshtein_1k_pairs"
    "yaegi:04_mini_json_parse_100kb"
    "yaegi:05_expression_eval_10k"
    "yaegi:06_lru_cache_100k_ops"
    "yaegi:07_dijkstra_10k_nodes"
    "yaegi:08_sudoku_solve_100"
    "yaegi:09_game_of_life_200x200_1k_gens"
    "yaegi:10_markov_text_gen_10k_words"
    "yaegi:11_trie_50k_words"
    "yaegi:12_brainfuck_mandelbrot"
    "yaegi:13_polymorphic_ast_eval_500k"
    "yaegi:14_mandelbrot_fp_200x200"
    "yaegi:15_open_addressing_hashmap_100k"
    "yaegi:16_parallel_word_count_montecristo"
    "yaegi:17_closures_pipeline"
    "yaegi:18_generic_numeric_pipeline"
    "yaegi:19_type_switches"
    "yaegi:20_invert_binary_tree"
    "yaegi:21_dense_layer"
    "yaegi:22_nbody_simulation"
    "yaegi:23_native_sha256_per_line"

    "scriggo:00_smoke_addition"
    "scriggo:01_fib_iterative"
    "scriggo:02_word_frequency_1mb"
    "scriggo:03_levenshtein_1k_pairs"
    "scriggo:04_mini_json_parse_100kb"
    "scriggo:05_expression_eval_10k"
    "scriggo:06_lru_cache_100k_ops"
    "scriggo:07_dijkstra_10k_nodes"
    "scriggo:08_sudoku_solve_100"
    "scriggo:09_game_of_life_200x200_1k_gens"
    "scriggo:10_markov_text_gen_10k_words"
    "scriggo:11_trie_50k_words"
    "scriggo:12_brainfuck_mandelbrot"
    "scriggo:13_polymorphic_ast_eval_500k"
    "scriggo:14_mandelbrot_fp_200x200"
    "scriggo:15_open_addressing_hashmap_100k"
    "scriggo:16_parallel_word_count_montecristo"
    "scriggo:17_closures_pipeline"
    "scriggo:18_generic_numeric_pipeline"
    "scriggo:19_type_switches"
    "scriggo:20_invert_binary_tree"
    "scriggo:21_dense_layer"
    "scriggo:22_nbody_simulation"
    "scriggo:23_native_sha256_per_line"

    "tengo:00_smoke_addition"
    "tengo:01_fib_iterative"
    "tengo:02_word_frequency_1mb"
    "tengo:03_levenshtein_1k_pairs"
    "tengo:04_mini_json_parse_100kb"
    "tengo:05_expression_eval_10k"
    "tengo:06_lru_cache_100k_ops"
    "tengo:07_dijkstra_10k_nodes"
    "tengo:08_sudoku_solve_100"
    "tengo:09_game_of_life_200x200_1k_gens"
    "tengo:10_markov_text_gen_10k_words"
    "tengo:11_trie_50k_words"
    "tengo:12_brainfuck_mandelbrot"
    "tengo:13_polymorphic_ast_eval_500k"
    "tengo:14_mandelbrot_fp_200x200"
    "tengo:15_open_addressing_hashmap_100k"
    "tengo:16_parallel_word_count_montecristo"
    "tengo:17_closures_pipeline"
    "tengo:18_generic_numeric_pipeline"
    "tengo:19_type_switches"
    "tengo:20_invert_binary_tree"
    "tengo:21_dense_layer"
    "tengo:22_nbody_simulation"
    "tengo:23_native_sha256_per_line"

    "mvm:00_smoke_addition"
    "mvm:01_fib_iterative"
    "mvm:02_word_frequency_1mb"
    "mvm:03_levenshtein_1k_pairs"
    "mvm:04_mini_json_parse_100kb"
    "mvm:05_expression_eval_10k"
    "mvm:06_lru_cache_100k_ops"
    "mvm:07_dijkstra_10k_nodes"
    "mvm:08_sudoku_solve_100"
    "mvm:09_game_of_life_200x200_1k_gens"
    "mvm:10_markov_text_gen_10k_words"
    "mvm:11_trie_50k_words"
    "mvm:12_brainfuck_mandelbrot"
    "mvm:13_polymorphic_ast_eval_500k"
    "mvm:14_mandelbrot_fp_200x200"
    "mvm:15_open_addressing_hashmap_100k"
    "mvm:16_parallel_word_count_montecristo"
    "mvm:17_closures_pipeline"
    "mvm:18_generic_numeric_pipeline"
    "mvm:19_type_switches"
    "mvm:20_invert_binary_tree"
    "mvm:21_dense_layer"
    "mvm:22_nbody_simulation"
    "mvm:23_native_sha256_per_line"
)

# Output JSONL file for this run. Overridable via --output.
OUTPUT_FILE="$DEFAULT_OUTPUT_FILE"

# Measured-run count for this run. Overridable via --runs.
RUN_COUNT="$DEFAULT_RUN_COUNT"

# Warmup-run count for this run. Overridable via --warmup.
WARMUP="$DEFAULT_WARMUP"

# Per-cell go-test timeout in seconds. Overridable via --cell-timeout.
CELL_TIMEOUT="$DEFAULT_CELL_TIMEOUT"

# Resume key matching <runner>_<bench>. Empty means start from the first
# cell. Set via --resume.
RESUME_FROM=""

# Dry-run flag. When 1, the planned sequence is printed and no cells are
# executed. Set via --dry-run.
DRY_RUN=0

# gogc_for picks the GOGC value for a runner family.
# Arguments:
#   $1 - runner name (piko, go, cpython, yaegi, scriggo, tengo)
# Outputs:
#   Writes the GOGC value to stdout
gogc_for() {
    local runner="$1"
    case "$runner" in
        piko|scriggo|tengo|go) echo "$GOGC_GO_RUNNER" ;;
        yaegi)                 echo "$GOGC_YAEGI" ;;
        *)                     echo "$GOGC_YAEGI" ;;
    esac
}

# resume_key_for builds the resume key for a (runner, bench) cell.
# Arguments:
#   $1 - runner
#   $2 - bench
# Outputs:
#   Writes the underscore-joined key to stdout
resume_key_for() {
    echo "$1_$2"
}

# run_cell executes a single (runner, bench) cell in a fresh subprocess
# and appends one JSONL row per (mode) result to OUTPUT_FILE.
#
# Globals:
#   PIKO_ROOT, OUTPUT_FILE, RUN_COUNT, WARMUP, CELL_TIMEOUT - Read
# Arguments:
#   $1 - runner
#   $2 - bench
# Returns:
#   0 if the cell completed. Individual mode failures are still written as
#   rows. Non-zero only if the subprocess crashed before producing
#   latest.json.
run_cell() {
    local runner="$1"
    local bench="$2"
    local gogc
    gogc=$(gogc_for "$runner")

    piko::log::info "Running ${runner} / ${bench} (GOGC=${gogc}, runs=${RUN_COUNT}, warmup=${WARMUP})..."

    local start_epoch
    start_epoch=$(date +%s)

    local exit_code=0
    GOGC="$gogc" \
        RUN_CROSS_LANG_BENCH=1 \
        CROSS_LANG_RUNNERS="$runner" \
        CROSS_LANG_FILTER="$bench" \
        CROSS_LANG_RUNS="$RUN_COUNT" \
        CROSS_LANG_WARMUP="$WARMUP" \
        go test -tags=crosslang -timeout "${CELL_TIMEOUT}s" \
        "${PIKO_ROOT}/tests/benchmarks/cross_language/" >/dev/null 2>&1 || exit_code=$?

    local end_epoch
    end_epoch=$(date +%s)
    local elapsed=$((end_epoch - start_epoch))

    local latest_json="${PIKO_ROOT}/tests/benchmarks/cross_language/results/latest.json"

    if [[ ! -f "$latest_json" ]]; then
        piko::log::error "  No latest.json produced (exit=$exit_code, elapsed=${elapsed}s)"
        emit_failed_row "$runner" "$bench" "subprocess-crashed"
        return 0
    fi

    append_cell_rows "$runner" "$bench" "$latest_json" "$exit_code" "$elapsed"
}

# append_cell_rows extracts the aggregates for this (runner, bench) from
# latest.json and appends one JSONL row per (mode) to OUTPUT_FILE. Also
# writes a status=failed row for any combo that wasn't aggregated (which
# happens when the harness recorded only failed/skipped runs).
#
# Globals:
#   OUTPUT_FILE - Read
# Arguments:
#   $1 - runner
#   $2 - bench
#   $3 - path to latest.json
#   $4 - subprocess exit code (for diagnostics)
#   $5 - elapsed seconds
append_cell_rows() {
    local runner="$1"
    local bench="$2"
    local latest_json="$3"
    local sub_exit="$4"
    local elapsed="$5"

    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    local host_os host_arch cpu_count go_version
    host_os=$(jq -r '.host.os' "$latest_json")
    host_arch=$(jq -r '.host.arch' "$latest_json")
    cpu_count=$(jq -r '.host.cpu_count_logical' "$latest_json")
    go_version=$(jq -r '.host.go_version // ""' "$latest_json")

    local row_count
    row_count=$(jq --arg r "$runner" --arg b "$bench" \
        '[.aggregates[] | select(.runner == $r and .benchmark == $b)] | length' \
        "$latest_json")

    if [[ "$row_count" == "0" ]]; then
        piko::log::warn "  No aggregates produced for ${runner}/${bench} (likely skipped or all-failed)"
        emit_failed_row "$runner" "$bench" "no-aggregates"
        return
    fi

    jq -c --arg runner "$runner" --arg bench "$bench" \
          --arg ts "$timestamp" \
          --arg os "$host_os" --arg arch "$host_arch" \
          --arg cpu "$cpu_count" --arg go "$go_version" \
          --arg elapsed "$elapsed" --arg sub_exit "$sub_exit" '
        .aggregates[] |
        select(.runner == $runner and .benchmark == $bench) |
        {
            lang: .runner,
            bench: .benchmark,
            mode: .mode,
            status: "ok",
            runs: .runs,
            compile_ns: (.median_compile_nanos // 0),
            runtime_ns: .median_nanos,
            cold_start_ns: (.cold_start_nanos // 0),
            mean_ns: .mean_nanos,
            stddev_ns: .stddev_nanos,
            min_ns: .min_nanos,
            p95_ns: .p95_nanos,
            peak_rss_kb: .peak_rss_kb_median,
            timestamp: $ts,
            host_os: $os,
            host_arch: $arch,
            cpu_count_logical: ($cpu | tonumber),
            go_version: $go,
            cell_elapsed_seconds: ($elapsed | tonumber),
            sub_exit_code: ($sub_exit | tonumber)
        }
    ' "$latest_json" >> "$OUTPUT_FILE"

    piko::log::success "  Recorded ${row_count} row(s) (${elapsed}s wall)"
}

# emit_failed_row appends a placeholder JSONL row for cells that produced
# no aggregates (e.g., subprocess crashed, all runs failed, runner skipped).
#
# Globals:
#   OUTPUT_FILE - Read
# Arguments:
#   $1 - runner
#   $2 - bench
#   $3 - reason string
emit_failed_row() {
    local runner="$1"
    local bench="$2"
    local reason="$3"

    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    jq -nc --arg runner "$runner" --arg bench "$bench" \
           --arg reason "$reason" --arg ts "$timestamp" '
        {
            lang: $runner,
            bench: $bench,
            mode: "innerloop",
            status: "failed",
            reason: $reason,
            timestamp: $ts
        }
    ' >> "$OUTPUT_FILE"
}

# parse_args handles command-line options. Unknown args are warned about and
# ignored so this script remains forward-compatible with future Makefile
# additions.
#
# Globals:
#   OUTPUT_FILE, RUN_COUNT, WARMUP, CELL_TIMEOUT, RESUME_FROM, DRY_RUN - Set
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --resume)
                RESUME_FROM="$2"
                shift 2
                ;;
            --output)
                OUTPUT_FILE="$2"
                shift 2
                ;;
            --runs)
                RUN_COUNT="$2"
                shift 2
                ;;
            --warmup)
                WARMUP="$2"
                shift 2
                ;;
            --cell-timeout)
                CELL_TIMEOUT="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=1
                shift
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            *)
                piko::log::warn "Unknown argument: $1 (ignoring)"
                shift
                ;;
        esac
    done
}

# usage prints the script's command-line interface.
usage() {
    cat <<'EOF'
Usage: full.sh [OPTIONS]

Run the full cross-language benchmark sweep, one process per cell.

Options:
  --resume <key>       Resume from cell <runner>_<bench>. Earlier cells skipped.
  --output <path>      Override output JSONL path
                       (default: tests/benchmarks/cross_language/results/full_sweep.jsonl).
  --runs <n>           Override per-cell run count (default: 10).
  --warmup <n>         Override per-cell warmup count (default: 2).
  --cell-timeout <s>   Override per-cell go-test timeout in seconds (default: 900).
  --dry-run            Print the planned sequence without running anything.
  --help, -h           Show this help.

Examples:
  ./full.sh --resume cpython_20_invert_binary_tree   Resume from cpython on bench 20
  ./full.sh --runs 3 --warmup 1 --output /tmp/sweep.jsonl   Quick exploratory run
EOF
}

# main orchestrates the sweep.
# Arguments:
#   $@ - Command-line arguments
main() {
    parse_args "$@"

    piko::log::header "Cross-language benchmark sweep"
    piko::log::info "Output:        $OUTPUT_FILE"
    piko::log::info "Runs/cell:     $RUN_COUNT (+$WARMUP warmup)"
    piko::log::info "Cell timeout:  ${CELL_TIMEOUT}s"
    piko::log::info "Cell count:    ${#LANG_BENCH_LIST[@]}"
    if [[ -n "$RESUME_FROM" ]]; then
        piko::log::info "Resume from:   $RESUME_FROM"
    fi
    if [[ "$DRY_RUN" -eq 1 ]]; then
        piko::log::info "Dry run:       enabled (no cells will execute)"
    fi
    piko::log::blank

    piko::util::ensure_dir "$(dirname "$OUTPUT_FILE")"

    local found_resume=1
    if [[ -n "$RESUME_FROM" ]]; then
        found_resume=0
    fi

    local cell_index=0
    local cell_total=${#LANG_BENCH_LIST[@]}
    local cell_done=0
    local cell_skipped=0

    local sweep_start
    sweep_start=$(date +%s)

    for cell in "${LANG_BENCH_LIST[@]}"; do
        cell_index=$((cell_index + 1))

        local runner="${cell%%:*}"
        local bench="${cell##*:}"
        local cell_key
        cell_key=$(resume_key_for "$runner" "$bench")

        if [[ $found_resume -eq 0 ]]; then
            if [[ "$cell_key" == "$RESUME_FROM" ]]; then
                found_resume=1
            else
                cell_skipped=$((cell_skipped + 1))
                continue
            fi
        fi

        piko::log::info "[$cell_index/$cell_total] $cell_key"

        if [[ "$DRY_RUN" -eq 1 ]]; then
            cell_done=$((cell_done + 1))
            continue
        fi

        run_cell "$runner" "$bench"
        cell_done=$((cell_done + 1))
    done

    local sweep_end
    sweep_end=$(date +%s)
    local sweep_elapsed=$((sweep_end - sweep_start))

    piko::log::blank
    piko::log::footer
    if [[ -n "$RESUME_FROM" ]] && [[ $found_resume -eq 0 ]]; then
        piko::log::error "Resume key '$RESUME_FROM' not found in LANG_BENCH_LIST"
        exit 1
    fi
    piko::log::success "Sweep complete: ran $cell_done cells, skipped $cell_skipped (elapsed ${sweep_elapsed}s)"
    piko::log::info "Output: $OUTPUT_FILE"
}

main "$@"
