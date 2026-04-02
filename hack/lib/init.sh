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

# hack/lib/init.sh - Main initialisation script for Piko hack/ scripts
#
# This file should be sourced at the top of every hack/ script:
#   source "$(dirname "$0")/lib/init.sh"
#
# It sets up:
#   - Strict error handling (errexit, nounset, pipefail)
#   - PIKO_ROOT environment variable
#   - Sources all library files (logging.sh, util.sh, go.sh, java.sh, lsp.sh)

# Prevent double-sourcing
if [[ -n "${_PIKO_INIT_LOADED:-}" ]]; then
    return 0
fi
readonly _PIKO_INIT_LOADED=1

set -o errexit
set -o nounset
set -o pipefail

if [[ -z "${PIKO_ROOT:-}" ]]; then
    if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
        PIKO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
    else
        PIKO_ROOT="$(pwd)"
        while [[ "$PIKO_ROOT" != "/" ]] && [[ ! -f "${PIKO_ROOT}/go.mod" ]]; do
            PIKO_ROOT="$(dirname "$PIKO_ROOT")"
        done
        if [[ "$PIKO_ROOT" == "/" ]]; then
            echo "Error: Could not determine PIKO_ROOT. Set it manually." >&2
            if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
                exit 1
            else
                return 1
            fi
        fi
    fi
fi
export PIKO_ROOT

PIKO_LIB_DIR="${PIKO_ROOT}/hack/lib"
export PIKO_LIB_DIR

# shellcheck source=./logging.sh
source "${PIKO_LIB_DIR}/logging.sh"
# shellcheck source=./util.sh
source "${PIKO_LIB_DIR}/util.sh"
# shellcheck source=./go.sh
source "${PIKO_LIB_DIR}/go.sh"
# shellcheck source=./java.sh
source "${PIKO_LIB_DIR}/java.sh"
# shellcheck source=./lsp.sh
source "${PIKO_LIB_DIR}/lsp.sh"
# shellcheck source=./cli.sh
source "${PIKO_LIB_DIR}/cli.sh"
# shellcheck source=./vm.sh
source "${PIKO_LIB_DIR}/vm.sh"

cd "${PIKO_ROOT}"
