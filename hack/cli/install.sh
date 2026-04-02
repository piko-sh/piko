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

# hack/cli/install.sh - Build and install the Piko CLI binary
#
# This script is called by Makefile targets:
#   make install-cli  Build and install CLI to PATH

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the locally-built binary.
BUILT_BINARY="${PIKO_ROOT}/bin/piko"

# Candidate installation directories, in priority order.
INSTALL_CANDIDATES=(
    "${HOME}/.local/bin"
    "/usr/local/bin"
)

# Whether we are running interactively (have a TTY).
INTERACTIVE=true
if [[ ! -t 0 ]] && [[ ! -e /dev/tty ]]; then
    INTERACTIVE=false
fi

# show_usage displays help information.
show_usage() {
    piko::log::info "Usage: make install-cli"
    piko::log::blank
    piko::log::info "Build and install the Piko CLI binary to your PATH."
    piko::log::detail "Detects existing installations and suggests overwriting."
    piko::log::detail "Falls back to ~/.local/bin or /usr/local/bin."
}

# prompt_yes_no asks the user a yes/no question.
# Arguments:
#   $1 - Prompt text
#   $2 - Default answer: y or n (default: n)
# Returns:
#   0 for yes, 1 for no
# Globals:
#   INTERACTIVE - Read
prompt_yes_no() {
    local prompt="$1"
    local default="${2:-n}"

    if [[ "$INTERACTIVE" == "false" ]]; then
        case "$default" in
            [yY]) return 0 ;;
            *) return 1 ;;
        esac
    fi

    local yn_hint
    if [[ "$default" == "y" ]]; then
        yn_hint="[Y/n]"
    else
        yn_hint="[y/N]"
    fi

    local answer
    read -r -p "$(echo -e "  ${PIKO_BOLD}${prompt}${PIKO_NC} ${yn_hint} ")" answer </dev/tty

    if [[ -z "$answer" ]]; then
        answer="$default"
    fi

    case "$answer" in
        [yY]|[yY][eE][sS]) return 0 ;;
        *) return 1 ;;
    esac
}

# find_existing_install checks if piko is already in PATH.
# Outputs:
#   The path to the existing binary, if found.
# Returns:
#   0 if found, 1 if not found.
find_existing_install() {
    command -v piko 2>/dev/null
}

# find_install_dir finds a suitable installation directory from PATH.
# Globals:
#   INSTALL_CANDIDATES - Read
# Outputs:
#   The chosen directory path.
find_install_dir() {
    for candidate in "${INSTALL_CANDIDATES[@]}"; do
        if [[ ":${PATH}:" == *":${candidate}:"* ]]; then
            echo "$candidate"
            return 0
        fi
    done

    local fallback="${HOME}/.local/bin"
    piko::log::warn "No standard install directory found in PATH."
    piko::log::info "Will use ${fallback} (you may need to add it to your PATH)."
    echo "$fallback"
}

# needs_sudo checks if a directory requires root permissions to write.
# Arguments:
#   $1 - Directory path
# Returns:
#   0 if sudo is needed, 1 if not needed.
needs_sudo() {
    local dir="$1"
    if [[ -w "$dir" ]]; then
        return 1
    fi
    return 0
}

# install_binary copies the built binary to the target path.
# Arguments:
#   $1 - Target file path
# Globals:
#   BUILT_BINARY - Read
install_binary() {
    local target="$1"
    local target_dir
    target_dir="$(dirname "$target")"

    if [[ ! -d "$target_dir" ]]; then
        piko::log::info "Creating directory: ${target_dir}"
        if needs_sudo "$(dirname "$target_dir")"; then
            sudo mkdir -p "$target_dir"
        else
            mkdir -p "$target_dir"
        fi
    fi

    piko::log::info "Installing piko to ${target}..."
    if needs_sudo "$target_dir"; then
        piko::log::info "Root permissions required for ${target_dir}"
        sudo cp "$BUILT_BINARY" "$target"
        sudo chmod 755 "$target"
    else
        cp "$BUILT_BINARY" "$target"
        chmod 755 "$target"
    fi
}

# ensure_in_path checks if a directory is in PATH and prints instructions if not.
# Arguments:
#   $1 - Directory to check
ensure_in_path() {
    local dir="$1"

    if [[ ":${PATH}:" == *":${dir}:"* ]]; then
        return 0
    fi

    piko::log::blank
    piko::log::warn "${dir} is not in your PATH."
    piko::log::info "Add it to your shell configuration:"
    piko::log::detail "bash:  echo 'export PATH=\"${dir}:\$PATH\"' >> ~/.bashrc"
    piko::log::detail "zsh:   echo 'export PATH=\"${dir}:\$PATH\"' >> ~/.zshrc"
    piko::log::detail "fish:  fish_add_path ${dir}"
    piko::log::blank
    piko::log::info "Then restart your shell or run:"
    piko::log::detail "source ~/.bashrc  # or ~/.zshrc"
}

# verify_installation runs the installed binary to confirm it works.
verify_installation() {
    if command -v piko &>/dev/null; then
        if piko help &>/dev/null; then
            piko::log::success "Verified: piko is available in PATH"
        else
            piko::log::warn "piko was installed but 'piko help' returned an error"
        fi
    else
        piko::log::warn "piko is not yet available in PATH (you may need to restart your shell)"
    fi
}

# main orchestrates the build and install process.
# Globals:
#   BUILT_BINARY - Read
main() {
    if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
        show_usage
        exit 0
    fi

    piko::log::header "Installing Piko CLI"

    piko::cli::build_current "$BUILT_BINARY"

    piko::log::blank

    local existing_path
    existing_path=$(find_existing_install) || true

    local install_path=""

    if [[ -n "$existing_path" ]]; then
        piko::log::info "Found existing installation: ${existing_path}"
        if prompt_yes_no "Overwrite ${existing_path}?" "y"; then
            install_path="$existing_path"
        fi
    fi

    if [[ -z "$install_path" ]]; then
        local install_dir
        install_dir=$(find_install_dir)

        local goos
        goos="$(piko::util::host_os)"
        local binary_name="piko"
        if [[ "$goos" == "windows" ]]; then
            binary_name="piko.exe"
        fi

        install_path="${install_dir}/${binary_name}"

        piko::log::info "Install location: ${install_path}"
        if ! prompt_yes_no "Install to ${install_path}?" "y"; then
            piko::log::info "Installation cancelled."
            exit 0
        fi
    fi

    install_binary "$install_path"

    local install_dir
    install_dir="$(dirname "$install_path")"
    ensure_in_path "$install_dir"

    verify_installation

    piko::log::blank
    piko::log::success "Piko CLI installed to ${install_path}"
    piko::log::blank
    piko::log::info "Get started:"
    piko::log::detail "piko new      # Run the project creation wizard"
    piko::log::detail "piko help     # Show available commands"
}

main "$@"
