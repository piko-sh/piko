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

# hack/generate/qt.sh - Generate Go code from quicktemplate files
#
# Generates Go code from .qtpl quicktemplate files. Installs qtc
# automatically if not present.
#
# Usage:
#   ./hack/generate/qt.sh

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Directories containing .qtpl files.
QT_DIRS=(
    "internal/render/render_templates"
)

# ensure_qtc installs the quicktemplate compiler if not already present.
ensure_qtc() {
    if ! command -v qtc &>/dev/null; then
        piko::log::info "Installing qtc..."
        go install github.com/valyala/quicktemplate/qtc@latest
    fi
}

# generate_qt runs qtc for all template directories.
# Globals:
#   QT_DIRS - Read
#   PIKO_ROOT - Read
generate_qt() {
    local dir_count=${#QT_DIRS[@]}
    local current=0

    for dir in "${QT_DIRS[@]}"; do
        current=$((current + 1))
        local dir_path="${PIKO_ROOT}/${dir}"

        piko::log::step "$current" "$dir_count" "Generating: $dir"

        if [[ ! -d "$dir_path" ]]; then
            piko::log::warn "Directory not found: $dir_path"
            continue
        fi

        cd "$dir_path" || piko::log::fatal "Failed to cd to $dir_path"

        if qtc; then
            piko::log::success "Generated: $dir"
        else
            piko::log::error "Failed: $dir"
        fi

        cd "$PIKO_ROOT" || exit 1
    done
}

# main generates quicktemplate code.
main() {
    piko::log::header "Generating quicktemplate code"

    ensure_qtc
    generate_qt

    piko::log::footer
    piko::log::success "quicktemplate generation complete!"
}

main "$@"
