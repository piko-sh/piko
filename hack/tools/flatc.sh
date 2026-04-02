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

# hack/tools/flatc.sh - Download flatc (FlatBuffers compiler) binary
#
# Downloads the flatc binary to .tools/ if not already present at the correct
# version. Supports Linux and macOS on amd64 and arm64.
#
# Usage:
#   ./hack/tools/flatc.sh

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Pinned flatc version.
FLATC_VERSION="25.12.19"

# Output directory for tools.
TOOLS_DIR="${PIKO_ROOT}/.tools"

# resolve_flatc_archive returns the zip filename for the current platform.
#
# FlatBuffers releases use non-standard archive names per platform:
#   Linux x86_64:  Linux.flatc.binary.clang++-18.zip
#   Linux aarch64: Linux.flatc.binary.g++-13.zip
#   macOS arm64:   Mac.flatc.binary.zip
#   macOS x86_64:  MacIntel.flatc.binary.zip
#   Windows:       Windows.flatc.binary.zip
#
# Outputs the archive filename to stdout.
resolve_flatc_archive() {
    local os arch
    os=$(uname -s)
    arch=$(uname -m)

    case "$os" in
        Darwin)
            case "$arch" in
                arm64)  echo "Mac.flatc.binary.zip" ;;
                x86_64) echo "MacIntel.flatc.binary.zip" ;;
                *)      piko::log::fatal "Unsupported macOS architecture: $arch" ;;
            esac
            ;;
        Linux)
            case "$arch" in
                x86_64)  echo "Linux.flatc.binary.clang++-18.zip" ;;
                aarch64) echo "Linux.flatc.binary.g++-13.zip" ;;
                *)       piko::log::fatal "Unsupported Linux architecture: $arch" ;;
            esac
            ;;
        *)
            piko::log::fatal "Unsupported operating system: $os"
            ;;
    esac
}

# download_flatc downloads the flatc binary for the current platform.
# Globals:
#   FLATC_VERSION - Read
#   TOOLS_DIR - Read
download_flatc() {
    local archive url binary_name

    binary_name="flatc-${FLATC_VERSION}"
    local target="${TOOLS_DIR}/${binary_name}"

    if [[ -x "$target" ]]; then
        piko::log::info "flatc ${FLATC_VERSION} already installed at ${target}"
        return 0
    fi

    archive=$(resolve_flatc_archive)
    url="https://github.com/google/flatbuffers/releases/download/v${FLATC_VERSION}/${archive}"

    piko::log::info "Downloading flatc ${FLATC_VERSION} (${archive})..."
    piko::log::detail "URL: ${url}"

    piko::util::ensure_dir "$TOOLS_DIR"

    local tmp_dir
    tmp_dir=$(mktemp -d)

    if ! curl -fsSL "$url" -o "${tmp_dir}/flatc.zip"; then
        rm -rf "$tmp_dir"
        piko::log::fatal "Failed to download flatc from ${url}"
    fi

    if ! unzip -q -o "${tmp_dir}/flatc.zip" -d "$tmp_dir"; then
        rm -rf "$tmp_dir"
        piko::log::fatal "Failed to extract flatc"
    fi

    mv "${tmp_dir}/flatc" "$target"
    chmod +x "$target"
    rm -rf "$tmp_dir"

    ln -sf "$binary_name" "${TOOLS_DIR}/flatc"

    piko::log::success "Installed flatc ${FLATC_VERSION} to ${target}"
}

# main downloads flatc if needed.
main() {
    download_flatc
}

main "$@"
