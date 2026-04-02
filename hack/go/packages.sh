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

# hack/go/packages.sh - List all Go packages excluding generated and test-only code
#
# Finds directories containing non-test .go files, filtering out testdata,
# generated code, vendored dependencies, and test-only packages. Package paths
# are printed to stdout, one per line.
#
# This script is called by Makefile targets:
#   make go-packages-list    List all Go packages

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# is_excluded checks whether a directory should be excluded from the package list.
# Directories are excluded if they are (or are children of) testdata, tmp, gen,
# esbuild, or tests directories, or if they end with _gen or _templates.
# Arguments:
#   $1 - Directory path to check
# Returns:
#   0 if excluded, 1 if included
is_excluded() {
    local dir="$1"

    case "$dir" in
        */testdata|*/testdata/*)                       return 0 ;;
        */testdata-modules|*/testdata-modules/*)       return 0 ;;
        */tmp|*/tmp/*)                                 return 0 ;;
        */gen|*/gen/*)                                 return 0 ;;
        */db|*/db/*)                                   return 0 ;;
        */scenarios|*/scenarios/*)                     return 0 ;;
        *_db|*_db/*)                                   return 0 ;;
        */symbols|*/symbols/*)                         return 0 ;;
        *_gen|*_gen/*)                                 return 0 ;;
        *_test|*_test/*)                               return 0 ;;
        *_schema|*_schema/*)                           return 0 ;;
        */schema|*/schema/*)                           return 0 ;;
        *_templates|*_templates/*)                     return 0 ;;
        */esbuild|*/esbuild/*)                         return 0 ;;
        */tests|*/tests/*)                             return 0 ;;
    esac

    return 1
}

# has_non_test_go_files checks whether a directory contains at least one
# non-test .go file.
# Arguments:
#   $1 - Directory path to check
# Returns:
#   0 if non-test .go files exist, 1 otherwise
has_non_test_go_files() {
    local dir="$1"

    local -a go_files
    shopt -s nullglob
    go_files=("${dir}"/*.go)
    shopt -u nullglob

    local f
    for f in "${go_files[@]}"; do
        if [[ "$f" != *_test.go ]]; then
            return 0
        fi
    done

    return 1
}

# main finds all Go packages and prints them to stdout.
# Arguments:
#   $@ - Unused
main() {
    local -a all_dirs
    while IFS= read -r dir; do
        all_dirs+=("$dir")
    done < <(find . -type f -name "*.go" -print0 | xargs -0 dirname | sort -u)

    local dir
    for dir in "${all_dirs[@]}"; do
        if is_excluded "$dir"; then
            continue
        fi

        if has_non_test_go_files "$dir"; then
            echo "$dir"
        fi
    done
}

main "$@"
