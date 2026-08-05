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

# Newline-separated "module_path<TAB>absolute_dir" for every intra-repository
# module. Built on first use by piko::go::ensure_module_index.
PIKO_LOCAL_MODULE_INDEX=""

# piko::go::ensure_module_index populates PIKO_LOCAL_MODULE_INDEX once per run.
# Globals:
#   PIKO_LOCAL_MODULE_INDEX - Set on first call
piko::go::ensure_module_index() {
    if [[ -n "${PIKO_LOCAL_MODULE_INDEX:-}" ]]; then
        return 0
    fi

    if ! piko::util::verify_binary "jq" "brew install jq OR apt install jq"; then
        piko::log::fatal "jq is required to read go.mod metadata."
    fi

    PIKO_LOCAL_MODULE_INDEX=$(piko::go::local_module_index "${PIKO_ROOT}")

    if [[ -z "$PIKO_LOCAL_MODULE_INDEX" ]]; then
        piko::log::fatal "Could not index any repository modules."
    fi
}

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

    local original_mod original_sum="" had_sum="false"
    local original_work_sum="" had_work_sum="false"
    original_mod=$(cat "${dir}/go.mod")
    if [[ -f "${dir}/go.sum" ]]; then
        had_sum="true"
        original_sum=$(cat "${dir}/go.sum")
    fi
    if [[ -f "${dir}/go.work.sum" ]]; then
        had_work_sum="true"
        original_work_sum=$(cat "${dir}/go.work.sum")
    fi

    local injected
    injected=$(piko::go::_inject_local_replaces "$dir")

    local status=0
    (cd "$dir" && GOWORK=off go mod tidy) || status=1

    piko::go::_drop_local_replaces "$dir" "$injected"
    piko::go::_normalise_local_requires "$dir"

    if [[ "$had_sum" != "true" ]]; then
        rm -f "${dir}/go.sum"
    fi

    if [[ $status -ne 0 ]]; then
        piko::go::_restore_mod_files "$dir" "$original_mod" "$original_sum" "$had_sum" "$original_work_sum" "$had_work_sum"
        return 1
    fi

    if [[ "$verify_only" == "true" ]]; then
        local after_mod after_sum=""
        after_mod=$(cat "${dir}/go.mod")
        if [[ -f "${dir}/go.sum" ]]; then
            after_sum=$(cat "${dir}/go.sum")
        fi

        if [[ "$original_mod" != "$after_mod" ]] || [[ "$original_sum" != "$after_sum" ]]; then
            piko::log::error "go mod tidy would make changes in $(piko::util::relative_path "$dir")"
            piko::go::_restore_mod_files "$dir" "$original_mod" "$original_sum" "$had_sum" "$original_work_sum" "$had_work_sum"
            return 1
        fi
    fi

    return 0
}

# piko::go::local_module_index prints "module_path<TAB>absolute_dir" for every
# module in this repository, so intra-repository requirements can be resolved to
# a directory on disk. Modules outside go.work are included as well, such as the
# build-constrained integration tests, because those must never be fetched from
# a proxy either. Only the piko.sh namespace is indexed, which leaves out the
# example fixtures that all share the module name "testmodule".
# Arguments:
#   $1 - Repository root (default: PIKO_ROOT)
piko::go::local_module_index() {
    local root="${1:-${PIKO_ROOT}}"

    local mod_file
    while IFS= read -r mod_file; do
        [[ -z "$mod_file" ]] && continue

        local module_dir
        module_dir=$(cd "$(dirname "$mod_file")" 2>/dev/null && pwd) || continue

        local module_path
        module_path=$(go mod edit -json "${module_dir}/go.mod" | jq -r '.Module.Path')
        [[ -z "$module_path" || "$module_path" == "null" ]] && continue

        case "$module_path" in
            piko.sh/*) ;;
            *) continue ;;
        esac

        printf '%s\t%s\n' "$module_path" "$module_dir"
    done < <(piko::util::find_go_modules "$root")
}

# piko::go::held_dependencies prints the module paths that must not be
# upgraded, read from hack/go/upgrade-hold.txt.
# Arguments:
#   $1 - Repository root (default: PIKO_ROOT)
piko::go::held_dependencies() {
    local root="${1:-${PIKO_ROOT}}"
    local hold_file="${root}/hack/go/upgrade-hold.txt"

    [[ -f "$hold_file" ]] || return 0

    grep -vE '^\s*(#|$)' "$hold_file" | tr -d '[:blank:]'
}

# piko::go::_module_requires prints a module's own requirements.
# Reads go.mod directly rather than asking go list, because under a workspace
# go list -m all reports the union of every module in the workspace.
# Arguments:
#   $1 - Directory containing go.mod
#   $2 - "direct" to exclude indirect requirements, otherwise all
piko::go::_module_requires() {
    local dir="$1"
    local which="${2:-all}"

    if [[ "$which" == "direct" ]]; then
        go mod edit -json "${dir}/go.mod" |
            jq -r '(.Require // [])[] | select(.Indirect | not) | .Path'
    else
        go mod edit -json "${dir}/go.mod" | jq -r '(.Require // [])[].Path'
    fi
}

# piko::go::_module_versions prints a module's direct requirements as
# path@version, sorted.
# Arguments:
#   $1 - Directory containing go.mod
piko::go::_module_versions() {
    local dir="$1"

    go mod edit -json "${dir}/go.mod" |
        jq -r '(.Require // [])[] | select(.Indirect | not) | "\(.Path)@\(.Version)"' | sort
}

# piko::go::_inject_local_replaces adds replace directives pointing at sibling
# modules on disk. go.work supplies these during builds, but go get and go mod
# tidy ignore it, so without them both commands try to fetch intra-repository
# requirements such as piko.sh/piko v0.0.0 from the proxy and fail.
# Arguments:
#   $1 - Directory containing go.mod
# Globals:
#   PIKO_LOCAL_MODULE_INDEX - Read
# Outputs:
#   The module paths that were injected, one per line
piko::go::_inject_local_replaces() {
    local dir="$1"

    piko::go::ensure_module_index

    local own_path existing
    own_path=$(go mod edit -json "${dir}/go.mod" | jq -r '.Module.Path')
    existing=$(go mod edit -json "${dir}/go.mod" | jq -r '(.Replace // [])[].Old.Path')

    local edit_args=() injected=()
    local module_path module_dir
    while IFS=$'\t' read -r module_path module_dir; do
        [[ -z "$module_path" ]] && continue
        [[ "$module_path" == "$own_path" ]] && continue
        grep -qxF "$module_path" <<<"$existing" && continue

        edit_args+=("-replace=${module_path}=${module_dir}")
        injected+=("$module_path")
    done <<<"$PIKO_LOCAL_MODULE_INDEX"

    if [[ ${#edit_args[@]} -eq 0 ]]; then
        return 0
    fi

    (cd "$dir" && go mod edit "${edit_args[@]}")
    printf '%s\n' "${injected[@]}"
}

# piko::go::_drop_local_replaces removes replace directives added by
# piko::go::_inject_local_replaces, leaving any that the module already had.
# Arguments:
#   $1 - Directory containing go.mod
#   $2 - Newline-separated module paths to drop
piko::go::_drop_local_replaces() {
    local dir="$1"
    local injected_paths="$2"

    [[ -z "$injected_paths" ]] && return 0

    local edit_args=()
    local injected_path
    while IFS= read -r injected_path; do
        [[ -z "$injected_path" ]] && continue
        edit_args+=("-dropreplace=${injected_path}")
    done <<<"$injected_paths"

    if [[ ${#edit_args[@]} -gt 0 ]]; then
        (cd "$dir" && go mod edit "${edit_args[@]}")
    fi
}

# piko::go::_normalise_local_requires rewrites sibling requirements back to
# v0.0.0. When go mod tidy adds a requirement on a module that is replaced by a
# directory, it stamps the zero pseudo-version, which no longer resolves once
# the replace directive is removed and go.work takes over again.
# Arguments:
#   $1 - Directory containing go.mod
piko::go::_normalise_local_requires() {
    local dir="$1"

    local edit_args=()
    local require_path
    while IFS= read -r require_path; do
        [[ -z "$require_path" ]] && continue
        edit_args+=("-require=${require_path}@v0.0.0")
    done < <(go mod edit -json "${dir}/go.mod" |
        jq -r '(.Require // [])[] | select(.Version == "v0.0.0-00010101000000-000000000000") | .Path')

    if [[ ${#edit_args[@]} -gt 0 ]]; then
        (cd "$dir" && go mod edit "${edit_args[@]}")
    fi
}

# piko::go::with_local_replaces runs a read-only command inside a module with
# sibling replace directives materialised, then restores go.mod byte for byte.
# Use this for tools that have to resolve the module graph but must not change it.
# Arguments:
#   $1 - Directory containing go.mod
#   $@ - Command and arguments to run inside the module
# Globals:
#   PIKO_LOCAL_MODULE_INDEX - Read
# Returns:
#   The exit status of the command
piko::go::with_local_replaces() {
    local dir="$1"
    shift

    [[ -f "${dir}/go.mod" ]] || return 0

    local original_mod
    original_mod=$(cat "${dir}/go.mod")

    piko::go::_inject_local_replaces "$dir" >/dev/null

    local status=0
    (cd "$dir" && GOWORK=off "$@") || status=$?

    printf '%s\n' "$original_mod" >"${dir}/go.mod"

    return "$status"
}

# piko::go::upgrade_module upgrades all direct dependencies in a module
# and reports what changed.
# Arguments:
#   $1 - Directory containing go.mod
#   $2 - dry_run (true/false, default: false)
# Globals:
#   PIKO_LOCAL_MODULE_INDEX - Read
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

    local original_mod original_sum="" had_sum="false"
    local original_work_sum="" had_work_sum="false"
    original_mod=$(cat "${dir}/go.mod")
    if [[ -f "${dir}/go.sum" ]]; then
        had_sum="true"
        original_sum=$(cat "${dir}/go.sum")
    fi
    if [[ -f "${dir}/go.work.sum" ]]; then
        had_work_sum="true"
        original_work_sum=$(cat "${dir}/go.work.sum")
    fi

    local before
    before=$(piko::go::_module_versions "$dir")

    local injected
    injected=$(piko::go::_inject_local_replaces "$dir")

    local excluded external_deps
    excluded=$(
        cut -f1 <<<"$PIKO_LOCAL_MODULE_INDEX"
        piko::go::held_dependencies
    )
    external_deps=$(piko::go::_module_requires "$dir" direct |
        grep -vxF -f <(grep -v '^$' <<<"$excluded") || true)

    local status=0
    if [[ -n "$external_deps" ]]; then
        if ! (cd "$dir" && GOWORK=off xargs go get -u <<<"$external_deps"); then
            status=1
        fi
    fi

    if [[ $status -eq 0 ]] && ! (cd "$dir" && GOWORK=off go mod tidy); then
        status=1
    fi

    local after
    after=$(piko::go::_module_versions "$dir")

    piko::go::_drop_local_replaces "$dir" "$injected"
    piko::go::_normalise_local_requires "$dir"

    if [[ "$dry_run" == "true" || $status -ne 0 ]]; then
        piko::go::_restore_mod_files "$dir" "$original_mod" "$original_sum" "$had_sum" "$original_work_sum" "$had_work_sum"
        [[ $status -ne 0 ]] && return 1
    elif [[ "$had_sum" != "true" ]]; then
        rm -f "${dir}/go.sum"
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
