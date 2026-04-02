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

# hack/lint/licences.sh - Check dependency licences for compatibility
#
# Verifies that all Go and npm dependency licences are compatible with
# Apache-2.0 using google/go-licenses and license-checker-rseidelsohn.
#
# Usage:
#   ./hack/lint/licences.sh

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Comma-separated list of allowed licence types (Go).
ALLOWED_LICENCES_GO="Apache-2.0,MIT,BSD-2-Clause,BSD-3-Clause,ISC,Unlicense"

# Semicolon-separated list of allowed licence types (npm).
ALLOWED_LICENCES_NPM="Apache-2.0;MIT;MIT-0;BSD-2-Clause;BSD-3-Clause;ISC;0BSD;CC0-1.0;CC-BY-3.0;Python-2.0;BlueOak-1.0.0;WTFPL;Artistic-2.0;Zlib"

# Packages with non-standard licence identifiers that have been manually reviewed.
# @vscode/vsce-sign: Microsoft proprietary, dev-only tool for extension signing.
EXCLUDED_NPM_PACKAGES="@vscode/vsce-sign;@vscode/vsce-sign-linux-x64;@vscode/vsce-sign-linux-arm64;@vscode/vsce-sign-darwin-x64;@vscode/vsce-sign-darwin-arm64;@vscode/vsce-sign-win32-x64;@vscode/vsce-sign-win32-arm64;@vscode/vsce-sign-alpine-x64;@vscode/vsce-sign-alpine-arm64"

# Frontend package directories to check.
NPM_PACKAGES=(
    "${PIKO_ROOT}/frontend/core"
    "${PIKO_ROOT}/frontend/playground"
    "${PIKO_ROOT}/frontend/extensions/analytics"
    "${PIKO_ROOT}/plugins/vscode"
)

# verify_go_licences checks that go-licenses is installed.
# Returns:
#   Exits with code 1 if not found
verify_go_licences() {
    if ! piko::util::verify_binary "go-licenses" "go install github.com/google/go-licenses/v2@v2.0.1"; then
        exit 1
    fi
}

# verify_npx checks that npx is installed.
# Returns:
#   Exits with code 1 if not found
verify_npx() {
    if ! piko::util::verify_binary "npx" "install Node.js (https://nodejs.org)"; then
        exit 1
    fi
}

# check_go_licences runs go-licenses check against all modules.
# Globals:
#   PIKO_ROOT - Read
#   ALLOWED_LICENCES_GO - Read
check_go_licences() {
    piko::log::header "Checking Go dependency licences"
    piko::log::info "Allowed licences: ${ALLOWED_LICENCES_GO}"
    piko::log::blank

    cd "$PIKO_ROOT" || exit 1

    if go-licenses check ./... --allowed_licenses="$ALLOWED_LICENCES_GO"; then
        piko::log::blank
        piko::log::success "All Go dependency licences are compatible"
    else
        piko::log::blank
        piko::log::error "Some Go dependencies have incompatible or unknown licences"
        piko::log::info "Review the output above and update ALLOWED_LICENCES_GO if appropriate"
        exit 1
    fi
}

# check_npm_licences runs license-checker against frontend packages.
# Globals:
#   NPM_PACKAGES - Read
#   ALLOWED_LICENCES_NPM - Read
check_npm_licences() {
    piko::log::header "Checking npm dependency licences"
    piko::log::info "Allowed licences: ${ALLOWED_LICENCES_NPM}"
    piko::log::blank

    for pkg_dir in "${NPM_PACKAGES[@]}"; do
        local rel_path
        rel_path=$(piko::util::relative_path "$pkg_dir")

        if [[ ! -d "${pkg_dir}/node_modules" ]]; then
            piko::log::warn "Skipping ${rel_path} (no node_modules, run npm install first)"
            continue
        fi

        piko::log::info "Checking ${rel_path}..."

        if npx --yes license-checker-rseidelsohn --start "$pkg_dir" --onlyAllow "$ALLOWED_LICENCES_NPM" --excludePackages "$EXCLUDED_NPM_PACKAGES" > /dev/null 2>&1; then
            piko::log::success "${rel_path} passed"
        else
            piko::log::blank
            piko::log::error "${rel_path} has incompatible or unknown licences:"
            npx --yes license-checker-rseidelsohn --start "$pkg_dir" --onlyAllow "$ALLOWED_LICENCES_NPM" --excludePackages "$EXCLUDED_NPM_PACKAGES" 2>&1
            exit 1
        fi
    done

    piko::log::blank
    piko::log::success "All npm dependency licences are compatible"
}

# main checks all dependency licences.
# Arguments:
#   $@ - Unused
main() {
    verify_go_licences
    verify_npx

    check_go_licences
    check_npm_licences
}

main "$@"
