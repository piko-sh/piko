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

# hack/lib/cli.sh - CLI build utilities for Piko scripts
#
# This file should be sourced, not executed directly.
# All functions are namespaced with piko::cli::

# Prevent double-sourcing.
if [[ -n "${_PIKO_CLI_LOADED:-}" ]]; then
    return 0
fi
readonly _PIKO_CLI_LOADED=1

# Version to embed in CLI binaries via ldflags. Override with PIKO_VERSION env var.
PIKO_CLI_VERSION="${PIKO_VERSION:-0.1.0-alpha}"

# Ldflags for CLI builds: strip symbols and embed version.
PIKO_CLI_LDFLAGS="-s -w -X main.Version=${PIKO_CLI_VERSION}"

# Default platforms for CLI builds.
readonly PIKO_CLI_PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

# piko::cli::build builds the CLI binary for a single platform.
# Arguments:
#   $1 - GOOS
#   $2 - GOARCH
#   $3 - Output directory
piko::cli::build() {
    local goos="$1"
    local goarch="$2"
    local output_dir="$3"

    local binary_name="piko"
    if [[ "$goos" == "windows" ]]; then
        binary_name="piko.exe"
    fi

    local output_path="${output_dir}/${goos}-${goarch}/${binary_name}"

    piko::util::ensure_dir "$(dirname "$output_path")"
    piko::go::build "$output_path" "${PIKO_ROOT}/cmd/piko" "$goos" "$goarch" "$PIKO_CLI_LDFLAGS"
}

# piko::cli::build_all builds CLI binaries for all default platforms.
# Arguments:
#   $1 - Output directory
piko::cli::build_all() {
    local output_dir="$1"

    piko::log::header "Building Piko CLI binaries"

    local platform
    for platform in "${PIKO_CLI_PLATFORMS[@]}"; do
        IFS='/' read -r goos goarch <<<"$platform"
        piko::cli::build "$goos" "$goarch" "$output_dir"
    done

    piko::log::footer
    piko::log::success "All CLI binaries built in ${output_dir}"

    echo
    echo "Directory structure:"
    if command -v tree &>/dev/null; then
        tree "$output_dir"
    else
        find "$output_dir" -type f
    fi
}

# piko::cli::build_current builds CLI binary for current platform only.
# Arguments:
#   $1 - Output path (full file path, e.g. bin/piko)
piko::cli::build_current() {
    local output="$1"
    local goos goarch
    goos="$(piko::util::host_os)"
    goarch="$(piko::util::host_arch)"

    piko::util::ensure_dir "$(dirname "$output")"
    piko::go::build "$output" "${PIKO_ROOT}/cmd/piko" "$goos" "$goarch" "$PIKO_CLI_LDFLAGS"
}
