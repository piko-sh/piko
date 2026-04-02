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

# hack/lib/java.sh - Java utilities for Piko hack/ scripts
#
# This file provides Java-related helper functions.
# It is sourced by hack/lib/init.sh and available in all hack/ scripts.

# Prevent double-sourcing
if [[ -n "${_PIKO_JAVA_LOADED:-}" ]]; then
    return 0
fi
readonly _PIKO_JAVA_LOADED=1

# Common Java 21 locations by platform.
PIKO_JAVA_21_PATHS=(
    "/usr/lib/jvm/java-21-openjdk"
    "/usr/lib/jvm/java-21"
    "/usr/lib/jvm/java-21-openjdk-amd64"
    "/usr/lib/jvm/java-21-openjdk-arm64"
    "/usr/lib/jvm/temurin-21"
    "/usr/lib/jvm/zulu-21"
    "/opt/homebrew/opt/openjdk@21"
    "/usr/local/opt/openjdk@21"
    "$HOME/.sdkman/candidates/java/21-open"
    "$HOME/.sdkman/candidates/java/21-tem"
    "$HOME/.asdf/installs/java/openjdk-21"
)

# piko::java::find_21 finds Java 21 installation path.
# Globals:
#   PIKO_JAVA_21_PATHS - Read
#   JAVA_HOME - Read (checked first if set)
# Outputs:
#   Writes JAVA_HOME path to stdout if found, empty otherwise
# Returns:
#   0 if found, 1 otherwise
piko::java::find_21() {
    if [[ -n "${JAVA_HOME:-}" ]]; then
        local version
        version=$("$JAVA_HOME/bin/java" -version 2>&1 | head -1)
        if [[ "$version" == *"21"* ]]; then
            echo "$JAVA_HOME"
            return 0
        fi
    fi

    for path in "${PIKO_JAVA_21_PATHS[@]}"; do
        if [[ -d "$path" ]] && [[ -x "$path/bin/java" ]]; then
            local version
            version=$("$path/bin/java" -version 2>&1 | head -1)
            if [[ "$version" == *"21"* ]]; then
                echo "$path"
                return 0
            fi
        fi
    done

    return 1
}

# piko::java::require_21 ensures Java 21 is available, exiting with helpful message if not.
# Globals:
#   JAVA_HOME - May be set
# Returns:
#   Exits with code 1 if Java 21 is not found
piko::java::require_21() {
    local java_home
    java_home=$(piko::java::find_21)

    if [[ -z "$java_home" ]]; then
        piko::log::error "Java 21 not found!"
        piko::log::blank
        piko::log::info "This operation requires Java 21."
        piko::log::blank
        piko::log::info "To fix this, either:"
        piko::log::blank
        piko::log::detail "1. Set JAVA_HOME before running this script:"
        piko::log::detail "   export JAVA_HOME=/path/to/java-21"
        piko::log::blank
        piko::log::detail "2. Install Java 21:"
        piko::log::detail "   - Fedora: sudo dnf install java-21-openjdk-devel"
        piko::log::detail "   - Ubuntu: sudo apt install openjdk-21-jdk"
        piko::log::detail "   - macOS:  brew install openjdk@21"
        piko::log::detail "   - SDKMAN: sdk install java 21-open"
        piko::log::blank
        exit 1
    fi

    export JAVA_HOME="$java_home"
    piko::log::success "Using Java 21: $JAVA_HOME"
}
