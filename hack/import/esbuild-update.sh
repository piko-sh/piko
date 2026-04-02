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

# hack/import/esbuild-update.sh - Update vendored esbuild internal code
#
# Usage:
#   ./hack/import/esbuild-update.sh [version]
#
# If version is not provided, the script will prompt for it.
# Check releases at: https://github.com/evanw/esbuild/releases

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Target directory for vendored esbuild code.
TARGET_BASE_DIR="${PIKO_ROOT}/internal/esbuild"

# Internal directories to copy from esbuild source.
ESBUILD_INTERNAL_DIRS_TO_COPY=(
    "ast"
    "css_ast"
    "css_lexer"
    "css_parser"
    "css_printer"
    "compat"
    "logger"
    "helpers"
    "sourcemap"
    "config"
    "fs"
    "js_ast"
    "js_lexer"
    "js_parser"
    "js_printer"
    "renamer"
    "runtime"
)

# Original import path in esbuild source.
OLD_IMPORT_PATH="github.com/evanw/esbuild/internal"

# New import path for vendored code.
NEW_IMPORT_PATH="piko.sh/piko/internal/esbuild"

# Temporary directory for downloads.
TEMP_DIR=""

# Version of esbuild to download.
ESBUILD_VERSION=""

# URL to download esbuild source.
DOWNLOAD_URL=""

# Name of the downloaded tarball.
TARBALL_NAME=""

# Name of the extracted directory.
EXTRACTED_DIR_NAME=""

# cleanup removes the temporary download directory.
# Globals:
#   TEMP_DIR - Read
cleanup() {
    piko::log::info "Cleaning up temporary directory: $TEMP_DIR"
    rm -rf "$TEMP_DIR"
}

# get_version prompts for or validates the esbuild version.
# Globals:
#   ESBUILD_VERSION - Set
#   DOWNLOAD_URL - Set
#   TARBALL_NAME - Set
#   EXTRACTED_DIR_NAME - Set
# Arguments:
#   $1 - Optional version string
get_version() {
    piko::log::header "esbuild Internal Code Updater"
    piko::log::info "ESBuild releases: https://github.com/evanw/esbuild/releases"

    ESBUILD_VERSION="${1:-}"
    if [[ -z "$ESBUILD_VERSION" ]]; then
        read -rp "Enter the esbuild version to download (e.g., 0.20.0 or 0.21.4): " ESBUILD_VERSION
    fi

    if [[ -z "$ESBUILD_VERSION" ]]; then
        piko::log::fatal "No version entered. Exiting."
    fi

    DOWNLOAD_URL="https://github.com/evanw/esbuild/archive/refs/tags/v${ESBUILD_VERSION}.tar.gz"
    TARBALL_NAME="v${ESBUILD_VERSION}.tar.gz"
    EXTRACTED_DIR_NAME="esbuild-${ESBUILD_VERSION}"
}

# download_esbuild downloads the esbuild source tarball.
# Globals:
#   DOWNLOAD_URL - Read
#   TEMP_DIR - Read
#   TARBALL_NAME - Read
#   ESBUILD_VERSION - Read
download_esbuild() {
    piko::log::info "Downloading esbuild version ${ESBUILD_VERSION}..."
    piko::log::detail "URL: $DOWNLOAD_URL"
    piko::log::blank

    if ! curl -L "$DOWNLOAD_URL" -o "$TEMP_DIR/$TARBALL_NAME"; then
        piko::log::fatal "Failed to download esbuild version ${ESBUILD_VERSION}."
    fi

    piko::log::success "Download complete."
}

# extract_tarball extracts the downloaded tarball.
# Globals:
#   TEMP_DIR - Read
#   TARBALL_NAME - Read
#   EXTRACTED_DIR_NAME - Read/Modified
#   ESBUILD_VERSION - Read
extract_tarball() {
    piko::log::info "Extracting $TARBALL_NAME..."
    if ! tar -xzf "$TEMP_DIR/$TARBALL_NAME" -C "$TEMP_DIR"; then
        piko::log::fatal "Failed to extract the tarball."
    fi

    if [[ ! -d "$TEMP_DIR/$EXTRACTED_DIR_NAME" ]]; then
        local alt_name="esbuild-${ESBUILD_VERSION#v}"
        if [[ -d "$TEMP_DIR/$alt_name" ]]; then
            EXTRACTED_DIR_NAME="$alt_name"
        else
            piko::log::error "Extracted directory not found."
            ls -l "$TEMP_DIR"
            piko::log::fatal "Please check the archive structure for version ${ESBUILD_VERSION}."
        fi
    fi

    piko::log::success "Extraction complete: $TEMP_DIR/$EXTRACTED_DIR_NAME"
}

# copy_internal_dirs copies selected internal packages to vendor location.
# Globals:
#   TEMP_DIR - Read
#   EXTRACTED_DIR_NAME - Read
#   TARGET_BASE_DIR - Read
#   ESBUILD_INTERNAL_DIRS_TO_COPY - Read
copy_internal_dirs() {
    piko::log::info "Copying internal directories to $TARGET_BASE_DIR..."

    piko::util::ensure_dir "$TARGET_BASE_DIR"

    for internal_dir in "${ESBUILD_INTERNAL_DIRS_TO_COPY[@]}"; do
        local source_path="$TEMP_DIR/$EXTRACTED_DIR_NAME/internal/$internal_dir"
        local dest_path="$TARGET_BASE_DIR/$internal_dir"

        if [[ -d "$source_path" ]]; then
            piko::log::detail "Copying: $internal_dir"
            rm -rf "$dest_path"
            mkdir -p "$dest_path"

            if rsync -a \
                --prune-empty-dirs \
                --include='*/' \
                --exclude='*_test.go' \
                --include='*.go' \
                --exclude='*' \
                "$source_path/" "$dest_path/"; then
                piko::log::success "Copied $internal_dir"
            else
                piko::log::error "Failed to copy $internal_dir"
            fi
        else
            piko::log::warn "Source directory $internal_dir not found. Skipping."
        fi
    done
}

# update_imports rewrites import paths in copied Go files.
# Globals:
#   TARGET_BASE_DIR - Read
#   OLD_IMPORT_PATH - Read
#   NEW_IMPORT_PATH - Read
update_imports() {
    piko::log::info "Updating import paths in copied .go files..."
    piko::log::detail "Replacing: \"${OLD_IMPORT_PATH}\""
    piko::log::detail "     With: \"${NEW_IMPORT_PATH}\""

    local sed_old
    local sed_new
    sed_old=$(printf '%s\n' "$OLD_IMPORT_PATH" | sed 's:[][\\/.^$*]:\\&:g')
    sed_new=$(printf '%s\n' "$NEW_IMPORT_PATH" | sed 's:[][\\/.^$*]:\\&:g')

    if [[ -d "$TARGET_BASE_DIR" ]]; then
        find "$TARGET_BASE_DIR" -type f -name "*.go" -print0 | xargs -0 sed -i "s/${sed_old}/${sed_new}/g"
        piko::log::success "Import paths updated."
    fi
}

# finalise copies the licence and creates a version file.
# Globals:
#   TEMP_DIR - Read
#   EXTRACTED_DIR_NAME - Read
#   TARGET_BASE_DIR - Read
#   ESBUILD_VERSION - Read
finalise() {
    local license_source="$TEMP_DIR/$EXTRACTED_DIR_NAME/LICENSE.md"
    local license_dest="$TARGET_BASE_DIR/LICENSE.esbuild.md"

    if [[ -f "$license_source" ]]; then
        cp "$license_source" "$license_dest"
        piko::log::success "Copied esbuild LICENSE.md"
    else
        piko::log::warn "esbuild LICENSE.md not found"
    fi

    local version_file="$TARGET_BASE_DIR/esbuild.version"
    echo "$ESBUILD_VERSION" >"$version_file"
    piko::log::success "Created version file: $ESBUILD_VERSION"

    piko::log::footer
    piko::log::header "Update Complete"
    piko::log::info "Files copied to: $TARGET_BASE_DIR"
    piko::log::info "Please review the changes and update your Go module dependencies if necessary."
}

# main orchestrates the esbuild vendoring process.
# Arguments:
#   $@ - Optional version string passed to get_version
main() {
    TEMP_DIR=$(mktemp -d)
    trap cleanup SIGINT SIGTERM

    get_version "$@"
    download_esbuild
    extract_tarball
    copy_internal_dirs
    update_imports
    finalise
    cleanup
}

main "$@"
