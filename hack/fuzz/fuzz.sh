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

# hack/fuzz/fuzz.sh - Run the Piko fuzz supervisor under kernel-enforced limits
#
# Builds piko-fuzz and runs it in the caller's working directory. Commands that need
# verified limits use a transient systemd user scope when available. Otherwise the
# supervisor checks the limits of its existing cgroup.
#
# Usage:
#   ./hack/fuzz/fuzz.sh workspace [--local-pipit]
#   ./hack/fuzz/fuzz.sh <command> [flags]
#
# Commands:
#   workspace  Print the workspace to build in, writing the Pipit overlay when asked
#   help       List the supervisor's commands and targets
#   doctor, campaign, replay, reduce, resume, gen, promote, coverage, inventory,
#   preview, status, report
#
# Limits:
#   -cpus 2 -mem 4GiB -min-free SIZE -disk 20GiB -state DIR
#
# Example:
#   ./hack/fuzz/fuzz.sh campaign -suite soak -duration 1h -cpus 8 -mem 12GiB

# Working directory of the caller, captured before init.sh moves to PIKO_ROOT.
CALLER_DIR="$(pwd -P)"
readonly CALLER_DIR

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Environment variable that selects the sibling Pipit checkout.
readonly LOCAL_PIPIT_ENV="PIKO_PIPIT_LOCAL"

# Pipit modules the overlay replaces with the sibling checkout, as module=path pairs.
readonly OVERLAY_REPLACES=(
    "pipit.sh/pipit=../pipit"
    "pipit.sh/pipit/sdk/extract=../pipit/sdk/extract"
    "pipit.sh/pipit/sdk/link=../pipit/sdk/link"
    "pipit.sh/pipit/sdk/stdlib=../pipit/sdk/stdlib"
)

# Memory cap for building the supervisor, applied when user scopes are available.
readonly BUILD_MEMORY_MAX="4G"

# Soft Go memory limit and parallelism for building the supervisor.
readonly BUILD_GOMEMLIMIT="1GiB"
readonly BUILD_PARALLELISM=2

# Seconds allowed for building the supervisor.
readonly BUILD_TIMEOUT=600

# Plan keys the supervisor may set, beyond the PIKO_FUZZ_ environment.
readonly PLAN_KEYS=" SCOPED CPU_LIST CPU_QUOTA MEMORY_MAX MEMORY_HIGH TASKS_MAX "

# Launch plan read from the supervisor.
declare -A PLAN=()

# show_usage displays help information.
show_usage() {
    sed -n '/^# Usage:/,/^# Example:/p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' >&2
}

# workspace_file prints the path to the Go workspace.
# With PIKO_PIPIT_LOCAL=1 it writes an overlay for the sibling Pipit checkout.
# Globals:
#   PIKO_ROOT - Read
#   LOCAL_PIPIT_ENV - Read
#   OVERLAY_REPLACES - Read
# Outputs:
#   The workspace path on stdout
workspace_file() {
    local base="${PIKO_ROOT}/go.work"
    if [[ "${!LOCAL_PIPIT_ENV:-}" != "1" ]]; then
        echo "$base"
        return
    fi

    local replaces=""
    local pair
    for pair in "${OVERLAY_REPLACES[@]}"; do
        if [[ -f "${PIKO_ROOT}/${pair#*=}/go.mod" ]]; then
            replaces+=$'\t'"${pair%%=*} => ${pair#*=}"$'\n'
        fi
    done
    if [[ -z "$replaces" ]]; then
        piko::log::fatal "${LOCAL_PIPIT_ENV}=1 but ../pipit is not checked out"
    fi

    local target="${PIKO_ROOT}/go.fuzz.work"
    local overlay
    overlay="$(cat "$base")"$'\n\n'"replace ("$'\n'"${replaces})"$'\n'
    if [[ ! -f "$target" ]] || [[ "$(cat "$target")"$'\n' != "$overlay" ]]; then
        printf '%s' "$overlay" > "${target}.tmp"
        mv "${target}.tmp" "$target"
    fi
    echo "$target"
}


# state_directory resolves the campaign state directory relative to the caller.
# Globals:
#   PIKO_ROOT - Read
#   CALLER_DIR - Read
# Arguments:
#   $@ - Supervisor arguments
# Outputs:
#   The absolute state directory on stdout
state_directory() {
    local state="${PIKO_FUZZ_STATE:-${PIKO_ROOT}/.piko/fuzz}"
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -state | --state)
                state="${2:-}"
                shift
                ;;
            -state=* | --state=*)
                state="${1#*=}"
                ;;
        esac
        shift
    done
    (cd "$CALLER_DIR" && realpath -m -- "$state")
}

# configure_build sets up the Go environment for the supervisor.
#
# The build cache and temporary files live under the state directory. Go uses the
# existing module cache with GOPROXY=off.
# Globals:
#   GOWORK - Set to the selected workspace unless already absolute
# Arguments:
#   $1 - Absolute state directory
configure_build() {
    local state="$1"
    local cache="${state}/.cache"

    mkdir -p "${cache}/tmp"
    chmod 0700 "$cache" "${cache}/tmp"
    if [[ ! -d "${cache}/go-build" ]]; then
        mkdir -m 0700 "${cache}/go-build"
        touch "${cache}/go-build/.piko-fuzz-owned"
    fi

    if [[ "${GOWORK:-}" != /* ]]; then
        GOWORK="$(workspace_file)"
    fi
    export GOWORK
    export GOCACHE="${cache}/go-build"
    export GOTMPDIR="${cache}/tmp"
    export TMPDIR="${cache}/tmp"
    export GOPROXY="off"
    export GOTOOLCHAIN="local"
    export PIKO_FUZZ_ROOT="${PIKO_ROOT}"
}

# build_supervisor builds piko-fuzz into the state cache, under a memory cap where user
# systemd scopes are available.
# Globals:
#   PIKO_ROOT - Read
#   BUILD_MEMORY_MAX - Read
#   BUILD_GOMEMLIMIT - Read
#   BUILD_PARALLELISM - Read
#   BUILD_TIMEOUT - Read
# Arguments:
#   $1 - Absolute state directory
# Outputs:
#   The supervisor path on stdout
build_supervisor() {
    local binary="${1}/.cache/piko-fuzz"
    local scratch="${binary}.$$"
    local command=(timeout "$BUILD_TIMEOUT" go build -o "$scratch" ./cmd/piko-fuzz)

    if user_scopes_available; then
        command=(systemd-run --user --scope --quiet -p MemoryMax="$BUILD_MEMORY_MAX" -p MemorySwapMax=0 -- "${command[@]}")
    fi

    if ! (cd "${PIKO_ROOT}/tests/fuzz" &&
        GOMEMLIMIT="$BUILD_GOMEMLIMIT" GOMAXPROCS="$BUILD_PARALLELISM" \
            GOFLAGS="${GOFLAGS:+${GOFLAGS} }-p=${BUILD_PARALLELISM}" "${command[@]}" >&2); then
        rm -f "$scratch"
        piko::log::fatal "supervisor build failed; check the Go version, local module cache and GOWORK overrides"
    fi
    mv "$scratch" "$binary"
    echo "$binary"
}

# user_scopes_available reports whether transient user systemd scopes can be created.
# Returns:
#   0 when systemd-run can create a user scope, 1 otherwise
user_scopes_available() {
    command -v systemd-run > /dev/null 2>&1 && systemd-run --user --scope --quiet true > /dev/null 2>&1
}

# read_plan loads launch settings and exports campaign variables.
# Globals:
#   PLAN - Set from the plan's limit keys
#   PLAN_KEYS - Read
#   PIKO_FUZZ_* - Exported from the plan
# Arguments:
#   $1 - Supervisor binary
#   $2 onwards - Supervisor arguments
read_plan() {
    local binary="$1"
    shift

    local output
    if ! output="$("$binary" _plan "$@")"; then
        exit 2
    fi

    local key value
    while IFS='=' read -r key value; do
        if [[ "$key" == PIKO_FUZZ_* ]]; then
            export "${key}=${value}"
        elif [[ "$PLAN_KEYS" == *" ${key} "* ]]; then
            PLAN["$key"]="$value"
        else
            piko::log::fatal "unexpected launch plan key: ${key}"
        fi
    done <<< "$output"
}

# run_owned_scope runs the supervisor in a transient scope and stops the scope when the
# supervisor exits, so workers in their own process groups do not outlive it.
# Globals:
#   PLAN - Read
# Arguments:
#   $1 - Supervisor binary
#   $@ - Supervisor arguments
# Returns:
#   The systemd scope command's exit status
run_owned_scope() {
    local binary="$1"
    shift
    local unit="piko-fuzz-$$.scope"

    piko::log::info "Starting the capped scope ${unit} (CPUs ${PLAN[CPU_LIST]}, memory ${PLAN[MEMORY_MAX]} bytes)"
    systemd-run --user --scope --quiet --collect --unit="$unit" \
        -p CPUQuota="${PLAN[CPU_QUOTA]}" -p CPUWeight=20 \
        -p MemoryMax="${PLAN[MEMORY_MAX]}" -p MemoryHigh="${PLAN[MEMORY_HIGH]}" -p MemorySwapMax=0 \
        -p OOMPolicy=continue -p TasksMax="${PLAN[TASKS_MAX]}" -p TimeoutStopSec=3s \
        -- taskset -c "${PLAN[CPU_LIST]}" "$binary" "$@" &
    local child=$!

    trap 'kill -INT "$child" 2> /dev/null || true' INT TERM
    local status=0
    wait "$child" || status=$?
    while kill -0 "$child" 2> /dev/null; do
        wait "$child" || status=$?
    done
    trap - INT TERM

    systemctl --user stop "$unit" > /dev/null 2>&1 || true
    return "$status"
}

# launch builds the supervisor, plans the command and runs it.
# Globals:
#   CALLER_DIR - Read
#   PLAN - Read
# Arguments:
#   $@ - Supervisor command and arguments
launch() {
    local state
    state="$(state_directory "$@")"
    configure_build "$state"

    local binary
    binary="$(build_supervisor "$state")"
    read_plan "$binary" "$@"

    cd "$CALLER_DIR" || exit 1
    if [[ "${PLAN[SCOPED]}" != "1" ]]; then
        exec "$binary" "$@"
    fi

    export GOMAXPROCS="${PIKO_FUZZ_CPUS}"
    export GOMEMLIMIT="$((PIKO_FUZZ_MEMORY_BYTES / 4))"
    if user_scopes_available; then
        run_owned_scope "$binary" "$@"
        return
    fi
    exec taskset -c "${PLAN[CPU_LIST]}" "$binary" "$@"
}

# main dispatches the launcher command.
# Globals:
#   PIKO_PIPIT_LOCAL - Set when --local-pipit is given
# Arguments:
#   $1 - Command
#   $@ - Command arguments
main() {
    case "${1:-}" in
        "" | -h | --help)
            show_usage
            ;;
        workspace)
            if [[ "${2:-}" == "--local-pipit" ]]; then
                export "${LOCAL_PIPIT_ENV}=1"
            fi
            workspace_file
            ;;
        *)
            launch "$@"
            ;;
    esac
}

main "$@"
