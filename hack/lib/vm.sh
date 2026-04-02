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

# hack/lib/vm.sh - VM management utilities for Piko scripts
#
# This file should be sourced, not executed directly.
# All functions are namespaced with piko::vm::

# Prevent double-sourcing.
if [[ -n "${_PIKO_VM_LOADED:-}" ]]; then
    return 0
fi
readonly _PIKO_VM_LOADED=1

readonly PIKO_VM_WINDOWS_DIR="${PIKO_ROOT}/tests/vm/windows"
readonly PIKO_VM_SSH_CONFIG="/tmp/piko-vm-ssh-config"

# Required Vagrant plugins for Windows VM support.
readonly PIKO_VM_REQUIRED_PLUGINS=(
    "vagrant-libvirt"
    "winrm"
    "winrm-fs"
    "winrm-elevated"
)

# piko::vm::require_vagrant checks that Vagrant and all required plugins are installed.
piko::vm::require_vagrant() {
    piko::util::verify_binary "vagrant" "See https://developer.hashicorp.com/vagrant/install" || return 1
    piko::util::verify_binary "sshpass" "Install with: sudo dnf install sshpass" || return 1

    local installed
    installed=$(vagrant plugin list 2>/dev/null)
    local missing=()

    for plugin in "${PIKO_VM_REQUIRED_PLUGINS[@]}"; do
        if ! echo "$installed" | grep -q "$plugin"; then
            missing+=("$plugin")
        fi
    done

    if [[ ${#missing[@]} -gt 0 ]]; then
        piko::log::error "Missing required Vagrant plugins: ${missing[*]}"
        piko::log::detail "Install with: vagrant plugin install ${missing[*]}"
        return 1
    fi
}

# piko::vm::vagrant_dir returns the path to the Windows VM directory.
piko::vm::vagrant_dir() {
    echo "${PIKO_VM_WINDOWS_DIR}"
}

# piko::vm::require_running checks that the Windows VM is in "running" state.
piko::vm::require_running() {
    cd "$(piko::vm::vagrant_dir)" || piko::log::fatal "Cannot cd to VM directory"

    if ! vagrant status --machine-readable 2>/dev/null \
        | grep -q "state,running"; then
        piko::log::fatal "Windows VM is not running. Start it with: make vm-windows-up"
    fi

    cd "${PIKO_ROOT}" || true
}

# piko::vm::ssh_config extracts the VM IP from Vagrant and writes a clean
# SSH config that uses password auth via sshpass.
piko::vm::ssh_config() {
    cd "$(piko::vm::vagrant_dir)" || piko::log::fatal "Cannot cd to VM directory"

    local vm_ip
    vm_ip=$(vagrant ssh-config 2>/dev/null | grep "HostName" | awk '{print $2}')

    if [[ -z "$vm_ip" ]]; then
        piko::log::fatal "Failed to get VM IP address from Vagrant"
    fi

    cat > "${PIKO_VM_SSH_CONFIG}" <<EOF
Host default
  HostName ${vm_ip}
  User vagrant
  Port 22
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
  LogLevel ERROR
EOF

    cd "${PIKO_ROOT}" || true
}

# piko::vm::ssh executes a command on the Windows VM via SSH.
# If no command is given, opens an interactive session.
# Arguments:
#   $@ - Command to execute (optional)
piko::vm::ssh() {
    if [[ ! -f "${PIKO_VM_SSH_CONFIG}" ]]; then
        piko::vm::ssh_config
    fi

    if [[ $# -eq 0 ]]; then
        sshpass -p vagrant ssh -F "${PIKO_VM_SSH_CONFIG}" default
    else
        sshpass -p vagrant ssh -F "${PIKO_VM_SSH_CONFIG}" default "$@"
    fi
}

# piko::vm::scp_to copies files from the host to the Windows VM.
# Arguments:
#   $1 - Local source path
#   $2 - Remote destination path
piko::vm::scp_to() {
    local source="$1"
    local destination="$2"

    if [[ ! -f "${PIKO_VM_SSH_CONFIG}" ]]; then
        piko::vm::ssh_config
    fi

    sshpass -p vagrant scp -F "${PIKO_VM_SSH_CONFIG}" \
        "$source" "default:${destination}"
}

# piko::vm::scp_from copies files from the Windows VM to the host.
# Arguments:
#   $1 - Remote source path
#   $2 - Local destination path
piko::vm::scp_from() {
    local source="$1"
    local destination="$2"

    if [[ ! -f "${PIKO_VM_SSH_CONFIG}" ]]; then
        piko::vm::ssh_config
    fi

    sshpass -p vagrant scp -F "${PIKO_VM_SSH_CONFIG}" \
        "default:${source}" "$destination"
}

# piko::vm::ensure_licences_dir checks for licence files and informs the user
# about evaluation mode if none are present. Never fails - licences are optional.
piko::vm::ensure_licences_dir() {
    local licence_dir="${PIKO_VM_WINDOWS_DIR}/licences"

    if [[ ! -d "$licence_dir" ]]; then
        piko::log::info "No licences directory - using Windows evaluation mode (90 days)"
        piko::log::detail "Go testing works without a licence. Outlook requires Office activation."
        return 0
    fi

    local has_keys=false
    for file in "${licence_dir}"/*.key "${licence_dir}"/*.txt; do
        if [[ -f "$file" ]]; then
            has_keys=true
            break
        fi
    done

    if [[ "$has_keys" == "false" ]]; then
        piko::log::info "No licence files found - using Windows evaluation mode (90 days)"
        piko::log::detail "Go testing works without a licence. Outlook requires Office activation."
        piko::log::detail "Place windows.key and/or office.key in ${licence_dir} to activate."
    fi
}
