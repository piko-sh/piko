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

# find_zip_file locates the most recent distribution ZIP.
# Globals:
#   IDEA_DIR - Read
# Outputs:
#   Writes the ZIP path to stdout
find_zip_file() {
    cd "$IDEA_DIR" || exit 1

    local -a zip_files
    shopt -s nullglob
    zip_files=(build/distributions/*.zip)
    shopt -u nullglob

    if [[ ${#zip_files[@]} -gt 0 ]]; then
        echo "${zip_files[-1]}"
    fi
}

# install_plugin installs the plugin into the user's IDE.
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

    local ide_plugins_dir=""

    for dir in ~/.local/share/JetBrains/GoLand*/plugins \
        ~/.local/share/JetBrains/IntelliJIdea*/plugins \
        ~/Library/Application\ Support/JetBrains/GoLand*/plugins \
        ~/Library/Application\ Support/JetBrains/IntelliJIdea*/plugins; do
        if [[ -d "$dir" ]]; then
            ide_plugins_dir="$dir"
            break
        fi
    done

    if [[ -z "$ide_plugins_dir" ]]; then
        piko::log::error "Could not find IDE plugins directory."
        piko::log::info "Please manually install the plugin from: $zip_file"
        piko::log::detail "Go to: Settings > Plugins > gear icon > Install Plugin from Disk"
        exit 1
    fi

    piko::log::info "Installing to: $ide_plugins_dir"

    rm -rf "$ide_plugins_dir/piko"
    unzip -q -o "$zip_file" -d "$ide_plugins_dir"

    piko::log::success "Plugin installed"
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
