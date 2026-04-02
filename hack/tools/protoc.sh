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

# hack/tools/protoc.sh - Download protoc (Protocol Buffers compiler) binary
#
# Downloads the protoc binary to .tools/ if not already present at the correct
# version. Supports Linux and macOS on amd64 and arm64.
#
# Usage:
#   ./hack/tools/protoc.sh

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Pinned protoc version.
PROTOC_VERSION="33.5"

# Output directory for tools.
TOOLS_DIR="${PIKO_ROOT}/.tools"

# download_protoc downloads the protoc binary for the current platform.
# Globals:
#   PROTOC_VERSION - Read
#   TOOLS_DIR - Read
download_protoc() {
    local os arch url binary_name platform_suffix

    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$os" in
        darwin) os="osx" ;;
        linux) os="linux" ;;
        *)
            piko::log::fatal "Unsupported operating system: $os"
            ;;
    esac

    case "$arch" in
        x86_64) arch="x86_64" ;;
        aarch64|arm64) arch="aarch_64" ;;
        *)
            piko::log::fatal "Unsupported architecture: $arch"
            ;;
    esac

    platform_suffix="${os}-${arch}"
    binary_name="protoc-${PROTOC_VERSION}"
    local target="${TOOLS_DIR}/${binary_name}"

    if [[ -x "$target" ]]; then
        piko::log::info "protoc ${PROTOC_VERSION} already installed at ${target}"
        return 0
    fi

    url="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${platform_suffix}.zip"

    piko::log::info "Downloading protoc ${PROTOC_VERSION} for ${platform_suffix}..."
    piko::log::detail "URL: ${url}"

    piko::util::ensure_dir "$TOOLS_DIR"

    local tmp_dir
    tmp_dir=$(mktemp -d)

    if ! curl -fsSL "$url" -o "${tmp_dir}/protoc.zip"; then
        rm -rf "$tmp_dir"
        piko::log::fatal "Failed to download protoc from ${url}"
    fi

    if ! unzip -q -o "${tmp_dir}/protoc.zip" bin/protoc -d "$tmp_dir"; then
        rm -rf "$tmp_dir"
        piko::log::fatal "Failed to extract protoc"
    fi

    mv "${tmp_dir}/bin/protoc" "$target"
    chmod +x "$target"
    rm -rf "$tmp_dir"

    ln -sf "$binary_name" "${TOOLS_DIR}/protoc"

    piko::log::success "Installed protoc ${PROTOC_VERSION} to ${target}"
}

# main downloads protoc if needed.
main() {
    download_protoc
}

main "$@"
