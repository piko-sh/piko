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

# hack/lib/logging.sh - Standardised logging utilities for Piko scripts
#
# This file should be sourced, not executed directly.
# All functions are namespaced with piko::log::

# Prevent double-sourcing
if [[ -n "${_PIKO_LOGGING_LOADED:-}" ]]; then
    return 0
fi
readonly _PIKO_LOGGING_LOADED=1

# Colour codes (only if terminal supports it)
if [[ -t 2 ]]; then
    readonly PIKO_RED='\033[0;31m'
    readonly PIKO_GREEN='\033[0;32m'
    readonly PIKO_YELLOW='\033[1;33m'
    readonly PIKO_BLUE='\033[0;34m'
    readonly PIKO_CYAN='\033[0;36m'
    readonly PIKO_BOLD='\033[1m'
    readonly PIKO_NC='\033[0m'
else
    readonly PIKO_RED=''
    readonly PIKO_GREEN=''
    readonly PIKO_YELLOW=''
    readonly PIKO_BLUE=''
    readonly PIKO_CYAN=''
    readonly PIKO_BOLD=''
    readonly PIKO_NC=''
fi

# piko::log::info prints an informational message
# Arguments:
#   $@ - Message to print
piko::log::info() {
    echo -e "${PIKO_BLUE}[INFO]${PIKO_NC} $*" >&2
}

# piko::log::success prints a success message
# Arguments:
#   $@ - Message to print
piko::log::success() {
    echo -e "${PIKO_GREEN}[OK]${PIKO_NC} $*" >&2
}

# piko::log::warn prints a warning message
# Arguments:
#   $@ - Message to print
piko::log::warn() {
    echo -e "${PIKO_YELLOW}[WARN]${PIKO_NC} $*" >&2
}

# piko::log::error prints an error message
# Arguments:
#   $@ - Message to print
piko::log::error() {
    echo -e "${PIKO_RED}[ERROR]${PIKO_NC} $*" >&2
}

# piko::log::fatal prints an error message and exits with code 1
# Arguments:
#   $@ - Message to print
piko::log::fatal() {
    echo -e "${PIKO_RED}[FATAL]${PIKO_NC} $*" >&2
    exit 1
}

# piko::log::header prints a prominent header
# Arguments:
#   $@ - Header text
piko::log::header() {
    echo "========================================================================" >&2
    echo -e " ${PIKO_BOLD}$*${PIKO_NC}" >&2
    echo "========================================================================" >&2
}

# piko::log::footer prints a footer separator
piko::log::footer() {
    echo "------------------------------------------------------------------------" >&2
    echo >&2
}

# piko::log::step prints a step indicator for multi-step processes
# Arguments:
#   $1 - Step number
#   $2 - Total steps
#   $3 - Step description
piko::log::step() {
    local step="$1"
    local total="$2"
    shift 2
    echo -e "${PIKO_CYAN}[${step}/${total}]${PIKO_NC} $*" >&2
}

# piko::log::detail prints supplementary detail without a prefix.
# Use for indented instructions or examples under a main log message.
# Arguments:
#   $@ - Detail text to print
piko::log::detail() {
    echo -e "  $*" >&2
}

# piko::log::blank prints a blank line for visual spacing.
piko::log::blank() {
    echo >&2
}

# piko::log::run prints a command before executing it (for debugging)
# Arguments:
#   $@ - Command to print and execute
piko::log::run() {
    echo -e "${PIKO_BLUE}+${PIKO_NC} $*" >&2
    "$@"
}
