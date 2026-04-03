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

# hack/lib/go.sh - Go-specific utilities for Piko scripts
#
# This file should be sourced, not executed directly.
# All functions are namespaced with piko::go::

# Prevent double-sourcing
if [[ -n "${_PIKO_GO_LOADED:-}" ]]; then
    return 0
fi
readonly _PIKO_GO_LOADED=1

# piko::go::module_name returns the Go module name from go.mod
piko::go::module_name() {
    local dir="${1:-${PIKO_ROOT}}"
    if [[ -f "${dir}/go.mod" ]]; then
        head -n1 "${dir}/go.mod" | awk '{print $2}'
    else
        piko::log::fatal "No go.mod found in ${dir}"
    fi
}

# piko::go::list_benchmarks lists all benchmarks in a package
# Arguments:
#   $1 - Package path
piko::go::list_benchmarks() {
    local pkg_path="$1"
    go test -list 'Benchmark.*' "$pkg_path" 2>/dev/null | grep '^Benchmark' || true
}

# piko::go::run_benchmarks runs benchmarks with standard settings
# Arguments:
#   $1 - Package path
#   $2 - Benchmark pattern (default: Benchmark.*)
#   $3 - Bench time (default: 3s)
#   $4 - Timeout (default: 5m)
piko::go::run_benchmarks() {
    local pkg_path="$1"
    local bench_pattern="${2:-Benchmark.*}"
    local benchtime="${3:-3s}"
    local timeout="${4:-5m}"

    go test -run='^$' -bench="${bench_pattern}" \
        -benchtime="${benchtime}" -timeout="${timeout}" \
        "$pkg_path"
}

# piko::go::run_profile runs a single profiling operation
# Arguments:
#   $1 - Package path
#   $2 - Benchmark name
#   $3 - Profile type (cpu, mem, mutex, block)
#   $4 - Output file for profile data
#   $5 - Report file to append results
#   $6 - Top N entries to show (default: 40)
piko::go::run_profile() {
    local pkg_path="$1"
    local bench_name="$2"
    local profile_type="$3"
    local profile_out="$4"
    local report_file="$5"
    local top_n="${6:-40}"

    piko::log::info "Running ${profile_type} profile for ${bench_name}..."

    local test_output
    local exit_code=0
    test_output=$(go test -run='^$' -bench="${bench_name}" \
        -benchtime=3s -timeout=5m \
        "-${profile_type}profile=${profile_out}" \
        "$pkg_path" 2>&1) || exit_code=$?

    if [[ $exit_code -ne 0 ]] || [[ "$test_output" == *"no benchmarks to run"* ]]; then
        piko::log::error "Failed to run benchmark '${bench_name}'"
        echo "$test_output"
        return 1
    fi

    if [[ -f "$profile_out" ]]; then
        {
            echo "========================================================================"
            echo "BENCHMARK: ${bench_name}"
            echo "PROFILE:   ${profile_type}"
            echo "========================================================================"
            echo ""
            if [[ "$profile_type" == "mem" ]]; then
                go tool pprof -top -inuse_space -nodecount="${top_n}" "$profile_out"
            else
                go tool pprof -top -nodecount="${top_n}" "$profile_out"
            fi
            echo ""
            echo "--- End of ${profile_type} profile ---"
            echo ""
        } >>"$report_file"
        rm "$profile_out"
    fi
}

# piko::go::tidy_module runs go mod tidy in a directory
# Arguments:
#   $1 - Directory containing go.mod
#   $2 - verify_only (true/false, default: false)
# Returns:
#   0 on success, 1 if verification fails
piko::go::tidy_module() {
    local dir="$1"
    local verify_only="${2:-false}"

    if [[ ! -f "${dir}/go.mod" ]]; then
        return 0
    fi

    piko::log::info "Tidying module in: $(piko::util::relative_path "$dir")"

    if [[ "$verify_only" == "true" ]]; then
        local original_mod original_sum
        original_mod=$(cat "${dir}/go.mod")
        original_sum=$(cat "${dir}/go.sum" 2>/dev/null || echo "")

        (cd "$dir" && go mod tidy)

        local after_mod after_sum
        after_mod=$(cat "${dir}/go.mod")
        after_sum=$(cat "${dir}/go.sum" 2>/dev/null || echo "")

        if [[ "$original_mod" != "$after_mod" ]] || [[ "$original_sum" != "$after_sum" ]]; then
            piko::log::error "go mod tidy would make changes in $(piko::util::relative_path "$dir")"
            echo "$original_mod" >"${dir}/go.mod"
            if [[ -n "$original_sum" ]]; then
                echo "$original_sum" >"${dir}/go.sum"
            fi
            return 1
        fi
    else
        (cd "$dir" && rm -f go.sum && go mod tidy)
    fi

    return 0
}

# piko::go::upgrade_module upgrades all direct dependencies in a module
# and reports what changed.
# Arguments:
#   $1 - Directory containing go.mod
#   $2 - dry_run (true/false, default: false)
# Outputs:
#   Prints upgraded dependencies to stderr
# Returns:
#   0 on success, 1 on failure
piko::go::upgrade_module() {
    local dir="$1"
    local dry_run="${2:-false}"

    if [[ ! -f "${dir}/go.mod" ]]; then
        return 0
    fi

    local original_mod="" original_sum="" had_sum="false"
    local original_work_sum="" had_work_sum="false"
    if [[ "$dry_run" == "true" ]]; then
        original_mod=$(cat "${dir}/go.mod")
        if [[ -f "${dir}/go.sum" ]]; then
            had_sum="true"
            original_sum=$(cat "${dir}/go.sum")
        fi
        if [[ -f "${dir}/go.work.sum" ]]; then
            had_work_sum="true"
            original_work_sum=$(cat "${dir}/go.work.sum")
        fi
    fi

    local has_internal_deps="false"
    if grep -q 'piko\.sh/piko' "${dir}/go.mod" 2>/dev/null; then
        has_internal_deps="true"
    fi

    local before after
    before=$(cd "$dir" && go list -m -f '{{if not .Indirect}}{{.Path}}@{{.Version}}{{end}}' all 2>/dev/null | sort)

    if [[ "$has_internal_deps" == "true" ]]; then
        local external_deps
        external_deps=$(cd "$dir" && go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all 2>/dev/null | grep -v '^piko\.sh/piko')

        if [[ -n "$external_deps" ]]; then
            (cd "$dir" && echo "$external_deps" | xargs go get -u 2>/dev/null) || true
        fi
    else
        if ! (cd "$dir" && go get -u ./... 2>/dev/null); then
            if [[ "$dry_run" == "true" ]]; then
                piko::go::_restore_mod_files "$dir" "$original_mod" "$original_sum" "$had_sum" "$original_work_sum" "$had_work_sum"
            fi
            return 1
        fi

        if ! (cd "$dir" && go mod tidy 2>/dev/null); then
            if [[ "$dry_run" == "true" ]]; then
                piko::go::_restore_mod_files "$dir" "$original_mod" "$original_sum" "$had_sum" "$original_work_sum" "$had_work_sum"
            fi
            return 1
        fi
    fi

    after=$(cd "$dir" && go list -m -f '{{if not .Indirect}}{{.Path}}@{{.Version}}{{end}}' all 2>/dev/null | sort)

    if [[ "$dry_run" == "true" ]]; then
        piko::go::_restore_mod_files "$dir" "$original_mod" "$original_sum" "$had_sum" "$original_work_sum" "$had_work_sum"
    fi

    piko::go::_report_module_changes "$before" "$after"
}

# piko::go::_restore_mod_files restores go.mod, go.sum, and go.work.sum from saved content.
# Arguments:
#   $1 - Directory containing go.mod
#   $2 - Original go.mod content
#   $3 - Original go.sum content
#   $4 - Whether go.sum existed before (true/false)
#   $5 - Original go.work.sum content
#   $6 - Whether go.work.sum existed before (true/false)
piko::go::_restore_mod_files() {
    local dir="$1"
    local mod_content="$2"
    local sum_content="$3"
    local had_sum="${4:-false}"
    local work_sum_content="${5:-}"
    local had_work_sum="${6:-false}"

    echo "$mod_content" >"${dir}/go.mod"

    if [[ "$had_sum" == "true" ]]; then
        echo "$sum_content" >"${dir}/go.sum"
    else
        rm -f "${dir}/go.sum"
    fi

    if [[ "$had_work_sum" == "true" ]]; then
        echo "$work_sum_content" >"${dir}/go.work.sum"
    else
        rm -f "${dir}/go.work.sum"
    fi
}

# piko::go::_report_module_changes prints a diff of before/after dependency lists.
# Arguments:
#   $1 - Before dependency list (path@version, sorted)
#   $2 - After dependency list (path@version, sorted)
piko::go::_report_module_changes() {
    local before="$1"
    local after="$2"

    local before_paths after_paths
    before_paths=$(echo "$before" | sed 's/@.*//' | sort)
    after_paths=$(echo "$after" | sed 's/@.*//' | sort)

    local common_paths
    common_paths=$(comm -12 <(echo "$before_paths") <(echo "$after_paths"))

    local has_changes=false

    while IFS= read -r path; do
        [[ -z "$path" ]] && continue
        local old_ver new_ver
        old_ver=$(echo "$before" | grep "^${path}@" | head -1 | sed 's/.*@//')
        new_ver=$(echo "$after" | grep "^${path}@" | head -1 | sed 's/.*@//')
        if [[ "$old_ver" != "$new_ver" ]]; then
            piko::log::detail "  ${path}: ${old_ver} -> ${new_ver}"
            has_changes=true
        fi
    done <<<"$common_paths"

    local removed
    removed=$(comm -23 <(echo "$before_paths") <(echo "$after_paths"))
    if [[ -n "$removed" ]]; then
        while IFS= read -r path; do
            [[ -z "$path" ]] && continue
            local old_ver
            old_ver=$(echo "$before" | grep "^${path}@" | head -1 | sed 's/.*@//')
            piko::log::detail "  removed: ${path}@${old_ver}"
            has_changes=true
        done <<<"$removed"
    fi

    local added
    added=$(comm -13 <(echo "$before_paths") <(echo "$after_paths"))
    if [[ -n "$added" ]]; then
        while IFS= read -r path; do
            [[ -z "$path" ]] && continue
            local new_ver
            new_ver=$(echo "$after" | grep "^${path}@" | head -1 | sed 's/.*@//')
            piko::log::detail "  added: ${path}@${new_ver}"
            has_changes=true
        done <<<"$added"
    fi

    if [[ "$has_changes" != "true" ]]; then
        piko::log::detail "  (no changes)"
    fi

    return 0
}

# piko::go::build builds a Go binary
# Arguments:
#   $1 - Output path
#   $2 - Package path
#   $3 - GOOS (default: current)
#   $4 - GOARCH (default: current)
#   $5 - Additional ldflags (default: "-s -w")
piko::go::build() {
    local output="$1"
    local pkg="$2"
    local goos="${3:-$(piko::util::host_os)}"
    local goarch="${4:-$(piko::util::host_arch)}"
    local ldflags="${5:--s -w}"

    piko::log::info "Building ${pkg} for ${goos}/${goarch}..."

    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
        go build -trimpath -ldflags="$ldflags" -o "$output" "$pkg"

    local size
    size=$(du -h "$output" | cut -f1)
    piko::log::success "Built $(basename "$output") (${size})"
}

# piko::go::test runs Go tests
# Arguments:
#   $1 - Package path (default: ./...)
#   $2 - Short mode (true/false, default: true)
#   $@ - Additional arguments
piko::go::test() {
    local pkg="${1:-./...}"
    local short="${2:-true}"
    shift 2 || true

    local args=()
    if [[ "$short" == "true" ]]; then
        args+=("-short")
    fi

    go test "${args[@]}" "$@" "$pkg"
}
