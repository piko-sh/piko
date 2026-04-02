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

# hack/tools/download.sh - Download all required development tools
#
# Downloads all code generation tools (flatc, protoc) to the .tools/
# directory. Each tool is downloaded only if not already present at the
# correct version.
#
# Usage:
#   ./hack/tools/download.sh

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the tools scripts.
TOOLS_DIR="${PIKO_ROOT}/hack/tools"

# download_all downloads all required tools.
# Globals:
#   TOOLS_DIR - Read
download_all() {
    piko::log::header "Downloading development tools"

    piko::log::info "Downloading flatc..."
    "${TOOLS_DIR}/flatc.sh"

    piko::log::blank
    piko::log::info "Downloading protoc..."
    "${TOOLS_DIR}/protoc.sh"

    piko::log::footer
    piko::log::success "All tools downloaded successfully!"
    piko::log::blank
    piko::log::info "Tools installed to: ${PIKO_ROOT}/.tools/"
    piko::log::detail "- flatc"
    piko::log::detail "- protoc"
}

# main downloads all tools.
main() {
    download_all
}

main "$@"
