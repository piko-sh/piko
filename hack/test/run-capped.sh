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

# hack/test/run-capped.sh - Run a heavy command under a memory cap
#
# Runs packages serially with Go's soft memory limit near 80% of the cap.
# Where user systemd scopes are available, MemoryMax enforces the cap without swap.
# The command runs in the caller's working directory.
#
# Usage:
#   ./hack/test/run-capped.sh <cap> <command...>
#
# The cap is a whole number of gibibytes with a G suffix, e.g. 6G.
#
# Example:
#   ./hack/test/run-capped.sh 6G go test -race -tags integration ./tests/integration/orchestrator_pipeline/...

# Working directory of the caller, captured before init.sh moves to PIKO_ROOT.
CALLER_DIR="$(pwd -P)"
readonly CALLER_DIR

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Mebibytes of GOMEMLIMIT per gibibyte of cap, about 80%.
readonly GOMEMLIMIT_MIB_PER_GIB=820

# show_usage displays help information.
show_usage() {
    piko::log::info "Usage: ./hack/test/run-capped.sh <cap> <command...>"
    piko::log::blank
    piko::log::detail "cap      Whole gibibytes with a G suffix, e.g. 6G"
    piko::log::detail "command  The command to run under the cap"
}

# user_scopes_available reports whether transient user systemd scopes can be created.
# Returns:
#   0 when systemd-run can create a user scope, 1 otherwise
user_scopes_available() {
    command -v systemd-run > /dev/null 2>&1 && systemd-run --user --scope -q true > /dev/null 2>&1
}

# main validates the cap and runs the command under it.
# Globals:
#   CALLER_DIR - Read
#   GOMEMLIMIT_MIB_PER_GIB - Read
# Arguments:
#   $1 - Memory cap, e.g. 6G
#   $@ - Command and its arguments
main() {
    if [[ $# -lt 2 ]]; then
        show_usage
        exit 2
    fi

    local cap="$1"
    shift
    if [[ ! "$cap" =~ ^[0-9]+G$ ]]; then
        piko::log::error "Cap must look like 6G, got ${cap}"
        exit 2
    fi

    local gibibytes="${cap%G}"
    export GOMEMLIMIT="$((gibibytes * GOMEMLIMIT_MIB_PER_GIB))MiB"
    export GOFLAGS="${GOFLAGS:+${GOFLAGS} }-p=1"

    cd "$CALLER_DIR" || exit 1
    if user_scopes_available; then
        exec systemd-run --user --scope -q -p MemoryMax="$cap" -p MemorySwapMax=0 -- "$@"
    fi
    exec "$@"
}

main "$@"
