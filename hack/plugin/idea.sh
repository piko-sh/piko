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

# hack/plugin/idea.sh - Build and release IntelliJ plugin
#
# This script is called by Makefile targets:
#   make plugin-idea-build    Build and package
#   make plugin-idea-install  Build, package, and install
#   make plugin-idea-run      Run sandbox IDE with plugin
#   make plugin-idea-clean    Clean build artefacts

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the IntelliJ plugin directory.
IDEA_DIR="${PIKO_ROOT}/plugins/idea"

# Central location for built LSP binaries.
LSP_BIN_DIR="${PIKO_ROOT}/bin/lsp"

# Target directory for LSP binaries within the plugin.
PLUGIN_BIN_DIR="${IDEA_DIR}/src/main/resources/bin"

# check_java_21 verifies Java 21 is available and sets JAVA_HOME.
# Uses piko::java::require_21 from hack/lib/java.sh
check_java_21() {
    piko::log::header "Checking Java 21"
    piko::java::require_21
}

# build_lsp_binaries builds LSP binaries and copies to plugin directory.
# Globals:
#   LSP_BIN_DIR - Read
#   PLUGIN_BIN_DIR - Read
build_lsp_binaries() {
    piko::lsp::build_all "$LSP_BIN_DIR"

    piko::log::header "Copying LSP Binaries to Plugin"
    piko::util::ensure_dir "$PLUGIN_BIN_DIR"

    # Remove binaries from earlier builds (including the old piko-lsp name) so
    # stale files are never packaged into the plugin.
    rm -rf "${PLUGIN_BIN_DIR:?}"/*

    piko::log::info "Copying from $LSP_BIN_DIR to $PLUGIN_BIN_DIR..."
    if cp -r "$LSP_BIN_DIR"/* "$PLUGIN_BIN_DIR/"; then
        piko::log::success "LSP binaries copied to plugin"
    else
        piko::log::fatal "Failed to copy LSP binaries"
    fi
}

# build_plugin builds the IntelliJ plugin with Gradle.
# Globals:
#   IDEA_DIR - Read
build_plugin() {
    piko::log::header "Building IntelliJ Plugin"

    check_java_21

    cd "$IDEA_DIR" || exit 1

    piko::log::info "Running Gradle build..."
    ./gradlew build --no-daemon

    piko::log::success "Plugin built"
}

# package_plugin creates a distribution ZIP.
# Globals:
#   IDEA_DIR - Read
package_plugin() {
    piko::log::header "Packaging Plugin"

    cd "$IDEA_DIR" || exit 1

    piko::log::info "Creating distribution..."
    ./gradlew buildPlugin -x buildSearchableOptions --no-daemon

    local zip_file
    zip_file=$(find_zip_file)

    if [[ -n "$zip_file" ]] && [[ -f "$zip_file" ]]; then
        local size
        size=$(du -h "$zip_file" | cut -f1)
        piko::log::success "Plugin packaged: $zip_file ($size)"
        piko::log::detail "Location: $zip_file"
    else
        piko::log::fatal "No distribution ZIP found"
    fi
}

# find_zip_file locates the most recently built distribution ZIP.
#
# Older builds can leave differently named ZIPs behind, so this picks the
# newest file by modification time rather than by name.
# Globals:
#   IDEA_DIR - Read
# Outputs:
#   Writes the ZIP path to stdout
find_zip_file() {
    cd "$IDEA_DIR" || exit 1

    local newest=""
    local candidate
    shopt -s nullglob
    for candidate in build/distributions/*.zip; do
        if [[ -z "$newest" ]] || [[ "$candidate" -nt "$newest" ]]; then
            newest="$candidate"
        fi
    done
    shopt -u nullglob

    if [[ -n "$newest" ]]; then
        echo "$newest"
    fi
}

# Products whose configuration folders can host the plugin.
IDE_PRODUCTS=(GoLand IntelliJIdea WebStorm)

# Lowest IDE version the plugin supports (sinceBuild 253 is 2025.3).
MINIMUM_IDE_VERSION="2025.3"

# Old install folder names that must be removed before a fresh install.
LEGACY_INSTALL_NAMES=(piko piko-idea piko-lsp-bin pikopls-bin)

# ide_config_roots prints the base folders that hold per-version IDE folders.
# Outputs:
#   Writes one folder per line to stdout
ide_config_roots() {
    if [[ "$(uname -s)" == "Darwin" ]]; then
        echo "${HOME}/Library/Application Support/JetBrains"
    else
        echo "${XDG_DATA_HOME:-${HOME}/.local/share}/JetBrains"
    fi
}

# plugins_dir_for prints the plugins folder for one per-version IDE folder.
#
# On Linux, plugins live directly in the per-version data folder. On macOS
# they live in a plugins subfolder.
# Arguments:
#   $1 - Per-version IDE folder, for example ~/.local/share/JetBrains/GoLand2026.1
# Outputs:
#   Writes the plugins folder to stdout
plugins_dir_for() {
    local version_dir="$1"
    if [[ "$(uname -s)" == "Darwin" ]]; then
        echo "${version_dir}/plugins"
    else
        echo "${version_dir}"
    fi
}

# version_at_least succeeds when the first year.release version is not lower
# than the second.
# Arguments:
#   $1 - Version to test, for example 2026.1
#   $2 - Minimum version, for example 2025.3
version_at_least() {
    local lowest
    lowest=$(printf '%s\n%s\n' "$1" "$2" | sort -V | head -n1)
    [[ "$lowest" == "$2" ]]
}

# find_ide_plugin_dirs prints the plugins folder of the newest supported
# version of each installed IDE product.
#
# PIKO_IDE_PLUGINS_DIR overrides the search with one explicit folder.
# Outputs:
#   Writes one plugins folder per line to stdout
find_ide_plugin_dirs() {
    if [[ -n "${PIKO_IDE_PLUGINS_DIR:-}" ]]; then
        echo "${PIKO_IDE_PLUGINS_DIR}"
        return
    fi

    local root product candidate version newest_dir newest_version
    while IFS= read -r root; do
        [[ -d "$root" ]] || continue
        for product in "${IDE_PRODUCTS[@]}"; do
            newest_dir=""
            newest_version=""
            shopt -s nullglob
            for candidate in "${root}/${product}"[0-9]*; do
                [[ -d "$candidate" ]] || continue
                version="${candidate##*/"${product}"}"
                [[ "$version" =~ ^[0-9]+\.[0-9]+$ ]] || continue
                version_at_least "$version" "$MINIMUM_IDE_VERSION" || continue
                if [[ -z "$newest_version" ]] || ! version_at_least "$newest_version" "$version"; then
                    newest_version="$version"
                    newest_dir="$candidate"
                fi
            done
            shopt -u nullglob
            if [[ -n "$newest_dir" ]]; then
                plugins_dir_for "$newest_dir"
            fi
        done
    done < <(ide_config_roots)
}

# install_into_plugins_dir unpacks the plugin ZIP into one plugins folder,
# removing earlier installs first.
# Arguments:
#   $1 - Plugin ZIP path
#   $2 - IDE plugins folder
install_into_plugins_dir() {
    local zip_file="$1"
    local plugins_dir="$2"

    if ! mkdir -p "$plugins_dir"; then
        piko::log::error "Could not create plugins folder: $plugins_dir"
        return 1
    fi

    local top_level
    top_level=$(unzip -Z1 "$zip_file" | head -n1)
    top_level="${top_level%%/*}"
    if [[ -z "$top_level" ]]; then
        piko::log::error "Plugin ZIP has no top-level folder: $zip_file"
        return 1
    fi

    local name
    for name in "${LEGACY_INSTALL_NAMES[@]}" "$top_level"; do
        rm -rf "${plugins_dir:?}/${name}"
    done

    if ! unzip -q -o "$zip_file" -d "$plugins_dir"; then
        piko::log::error "Could not unpack $zip_file into $plugins_dir"
        return 1
    fi

    piko::log::success "Installed into: $plugins_dir"
}

# install_plugin installs the plugin into every supported IDE it finds.
# Globals:
#   IDEA_DIR - Read
install_plugin() {
    piko::log::header "Installing Plugin"

    cd "$IDEA_DIR" || exit 1

    local zip_file
    zip_file=$(find_zip_file)

    if [[ -z "$zip_file" ]] || [[ ! -f "$zip_file" ]]; then
        piko::log::fatal "No distribution ZIP found. Run 'make plugin-idea-build' first."
    fi
    zip_file="${IDEA_DIR}/${zip_file}"

    local -a plugin_dirs=()
    local plugin_dir
    while IFS= read -r plugin_dir; do
        [[ -n "$plugin_dir" ]] && plugin_dirs+=("$plugin_dir")
    done < <(find_ide_plugin_dirs)

    if [[ ${#plugin_dirs[@]} -eq 0 ]]; then
        piko::log::error "Could not find a GoLand, IntelliJ IDEA or WebStorm ${MINIMUM_IDE_VERSION}+ configuration folder."
        piko::log::info "Set PIKO_IDE_PLUGINS_DIR to the IDE plugins folder, or install manually from: $zip_file"
        piko::log::detail "Go to: Settings > Plugins > gear icon > Install Plugin from Disk"
        exit 1
    fi

    local failures=0
    for plugin_dir in "${plugin_dirs[@]}"; do
        install_into_plugins_dir "$zip_file" "$plugin_dir" || failures=$((failures + 1))
    done

    if [[ $failures -gt 0 ]]; then
        piko::log::fatal "Plugin install failed for ${failures} IDE folder(s)"
    fi

    piko::log::blank
    piko::log::warn "Please restart your IDE to activate the plugin"
}

# run_ide launches a sandbox IDE with the plugin for testing.
# Globals:
#   IDEA_DIR - Read
run_ide() {
    build_lsp_binaries

    piko::log::header "Running IntelliJ Sandbox IDE"

    check_java_21

    cd "$IDEA_DIR" || exit 1

    piko::log::info "Running Gradle clean + build + runIde..."
    ./gradlew clean generatePKLexer build runIde --no-daemon
}

# clean_build removes all build artefacts.
# Globals:
#   IDEA_DIR - Read
#   PLUGIN_BIN_DIR - Read
clean_build() {
    piko::log::header "Cleaning Build Artefacts"

    cd "$IDEA_DIR" || exit 1

    piko::log::info "Removing plugin binaries..."
    rm -rf "${PLUGIN_BIN_DIR:?}"/*

    piko::log::info "Removing Gradle build..."
    rm -rf build/

    piko::log::success "Build artefacts cleaned"
}

# main handles build, install, or clean commands.
# Arguments:
#   $1 - Command to execute: build (default), install, or clean
main() {
    local command="${1:-build}"

    case "$command" in
        build)
            build_lsp_binaries
            build_plugin
            package_plugin
            piko::log::blank
            piko::log::success "Build complete!"
            piko::log::blank
            piko::log::info "To install: make plugin-idea-install"
            piko::log::blank
            piko::log::info "Or manually install from:"
            piko::log::detail "Settings > Plugins > gear icon > Install Plugin from Disk"
            ;;

        install)
            build_lsp_binaries
            build_plugin
            package_plugin
            piko::log::blank
            install_plugin
            ;;

        run)
            run_ide
            ;;

        clean)
            clean_build
            ;;

        *)
            piko::log::fatal "Unknown command: $command (expected: build, install, run, clean)"
            ;;
    esac
}

main "$@"
