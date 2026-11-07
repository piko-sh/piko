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

# hack/lib/lsp.sh - LSP build utilities for Piko scripts
#
# This file should be sourced, not executed directly.
# All functions are namespaced with piko::lsp::

# Prevent double-sourcing.
if [[ -n "${_PIKO_LSP_LOADED:-}" ]]; then
    return 0
fi
readonly _PIKO_LSP_LOADED=1

# Version to embed in LSP binaries via ldflags. Override with PIKO_VERSION env var.
PIKO_LSP_VERSION="${PIKO_VERSION:-dev}"

# Ldflags for LSP builds: strip symbols and embed version.
PIKO_LSP_LDFLAGS="-s -w -X main.Version=${PIKO_LSP_VERSION}"

# Default platforms for LSP builds.
readonly PIKO_LSP_PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

# piko::lsp::build builds the LSP binary for a single platform.
# Arguments:
#   $1 - GOOS
#   $2 - GOARCH
#   $3 - Output directory
piko::lsp::build() {
    local goos="$1"
    local goarch="$2"
    local output_dir="$3"

    local binary_name="pikopls"
    if [[ "$goos" == "windows" ]]; then
        binary_name="pikopls.exe"
    fi

    local output_path="${output_dir}/${goos}-${goarch}/${binary_name}"

    piko::log::info "Building LSP for ${goos}/${goarch}..."
    piko::util::ensure_dir "$(dirname "$output_path")"

    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
        go build -trimpath -ldflags="$PIKO_LSP_LDFLAGS" \
        -o "$output_path" \
        "${PIKO_ROOT}/cmd/pikopls"

    local size
    size=$(du -h "$output_path" | cut -f1)
    piko::log::success "Built ${goos}/${goarch} (${size})"
}

# piko::lsp::remove_stale_binaries deletes LSP binaries left by earlier builds,
# including the old piko-lsp name, so they are never packaged again. Only the
# known binary names are removed, so other files in the folder are kept.
# Arguments:
#   $1 - Output directory
piko::lsp::remove_stale_binaries() {
    local output_dir="$1"
    [[ -d "$output_dir" ]] || return 0

    piko::log::info "Removing earlier LSP binaries from ${output_dir}..."
    find "$output_dir" -maxdepth 2 -type f \
        \( -name 'piko-lsp' -o -name 'piko-lsp.exe' -o -name 'pikopls' -o -name 'pikopls.exe' \) \
        -delete
}

# piko::lsp::build_all builds LSP binaries for all default platforms.
# Arguments:
#   $1 - Output directory
piko::lsp::build_all() {
    local output_dir="$1"

    piko::log::header "Building Piko LSP binaries"

    piko::lsp::remove_stale_binaries "$output_dir"

    local platform
    for platform in "${PIKO_LSP_PLATFORMS[@]}"; do
        IFS='/' read -r goos goarch <<<"$platform"
        piko::lsp::build "$goos" "$goarch" "$output_dir"
    done

    piko::log::footer
    piko::log::success "All LSP binaries built in ${output_dir}"

    echo
    echo "Directory structure:"
    if command -v tree &>/dev/null; then
        tree "$output_dir"
    else
        find "$output_dir" -type f
    fi
}

# piko::lsp::build_current builds LSP binary for current platform only.
# Arguments:
#   $1 - Output directory (binary placed directly in this directory)
piko::lsp::build_current() {
    local output_dir="$1"
    local goos goarch
    goos="$(piko::util::host_os)"
    goarch="$(piko::util::host_arch)"

    local binary_name="pikopls"
    if [[ "$goos" == "windows" ]]; then
        binary_name="pikopls.exe"
    fi

    local output_path="${output_dir}/${binary_name}"

    piko::util::ensure_dir "$output_dir"
    piko::go::build "$output_path" "${PIKO_ROOT}/cmd/pikopls" "$goos" "$goarch" "$PIKO_LSP_LDFLAGS"
}
