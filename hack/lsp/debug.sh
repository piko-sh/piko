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

# hack/lsp/debug.sh - Start LSP with Delve debugger for debugging
#
# Usage:
#   ./hack/lsp/debug.sh /path/to/your/test/project
#
# This builds a debug version of the LSP and starts it with Delve
# listening on port 2345 for remote debugging.

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the project directory to debug.
TARGET_DIR=""

# Path to the debug binary output.
DEBUG_BINARY="/tmp/piko-lsp-debug"
trap 'rm -f "$DEBUG_BINARY"' EXIT

# validate_args checks that a valid project directory was provided.
# Globals:
#   TARGET_DIR - Set to the project path
# Arguments:
#   $1 - Path to the project directory
validate_args() {
    if [[ -z "$1" || "$1" == "-h" || "$1" == "--help" ]]; then
        piko::log::error "You must provide the path to the project you want to debug."
        piko::log::info "Usage: ./hack/lsp/debug.sh /path/to/your/test/project"
        exit 1
    fi

    TARGET_DIR="$1"

    if [[ ! -d "$TARGET_DIR" ]]; then
        piko::log::fatal "Directory not found: $TARGET_DIR"
    fi
}

# verify_delve checks that the Delve debugger is installed.
# Returns:
#   Exits with code 1 if dlv is not found
verify_delve() {
    if ! piko::util::verify_binary "dlv" "go install github.com/go-delve/delve/cmd/dlv@latest"; then
        exit 1
    fi
}

# build_debug_binary compiles the LSP with debug symbols.
# Globals:
#   DEBUG_BINARY - Read (output path)
#   PIKO_ROOT - Read
build_debug_binary() {
    piko::log::info "Building debug version of piko-lsp..."
    go build -o "$DEBUG_BINARY" -gcflags="all=-N -l" "${PIKO_ROOT}/cmd/lsp"
    piko::log::success "Debug binary built at $DEBUG_BINARY"
    piko::log::footer
}

# run_debugger starts Delve in headless mode for remote debugging.
# Globals:
#   TARGET_DIR - Read
#   DEBUG_BINARY - Read
run_debugger() {
    cd "$TARGET_DIR" || exit 1
    piko::log::info "Set Current Working Directory to: $(pwd)"

    export PIKO_LSP_TCP_ADDR="127.0.0.1:4389"
    export PIKO_LSP_DRIVER="tcp"
    export PIKO_DISABLE_CONSOLE_LOG="true"

    piko::log::info "LSP will listen on: $PIKO_LSP_TCP_ADDR"
    piko::log::info "Starting Delve debugger server on: 127.0.0.1:2345"
    piko::log::footer

    dlv exec "$DEBUG_BINARY" --headless --listen=:2345 --api-version=2 --accept-multiclient
}

# main starts the LSP debugger session.
# Arguments:
#   $@ - Project directory path
main() {
    validate_args "$@"
    verify_delve
    build_debug_binary
    run_debugger
}

main "$@"
