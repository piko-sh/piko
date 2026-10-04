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

# hack/lint/go.sh - Lint all Go modules in the workspace with golangci-lint
#
# golangci-lint does not support go.work natively, so this script parses the
# workspace file and runs the linter against each module individually.
#
# Usage:
#   ./hack/lint/go.sh              # CI mode: skip build-constrained modules
#   ./hack/lint/go.sh --all        # Local mode: include all build tags and build variants

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Build tags to pass to golangci-lint via --build-tags.
# When --all is used, this includes all custom build tags so that modules
# gated behind constraints (integration, vips, ffmpeg, etc.) are also linted.
BUILD_TAGS=""

# All custom build tags used in the project. Platform tags (linux, darwin,
# windows) and Go version tags (go1.25) are excluded since they are handled
# automatically by the toolchain.
readonly ALL_BUILD_TAGS="integration,vips,ffmpeg,bench,fuzz,smoke"

# Array of every module path listed in go.work.
WORKSPACE_MODULES=()

# Array of workspace module paths.
MODULES=()

# Array of module paths that require GOOS/GOARCH cross-compilation for linting.
# These are handled separately from normal modules because golangci-lint's
# --build-tags flag cannot set GOOS/GOARCH.
WASM_MODULES=()

# Arguments shared by every golangci-lint invocation. Parallel runners are allowed so
# that a lint pass can run alongside an editor or another checkout's lint.
readonly LINT_ARGS=("run" "--allow-parallel-runners")

# Build variants that select files the native pass never compiles, in order of
# preference. A file is linted under the first variant whose constraints it satisfies:
# notags (the host platform with no custom build tags, for files that negate a custom tag
# such as !bench, !integration or !vips), safe (the host platform plus the safe tag),
# then one representative GOOS per operating system family, then js/wasm last so that
# negated constraints such as !linux or !unix land on a server platform. A GOOS variant
# matching the host platform selects nothing extra. Only the notags variant lints test
# files, so a constrained test file is assigned to notags or to no variant.
readonly BUILD_VARIANTS=("notags" "safe" "linux" "windows" "darwin" "freebsd" "solaris" "js")

# GOOS values that satisfy the "unix" build constraint.
readonly UNIX_GOOS=" aix android darwin dragonfly freebsd hurd illumos ios linux netbsd openbsd solaris "

# Host platform settings, read from go env by discover_variant_packages.
HOST_GOOS=""
HOST_GOARCH=""
HOST_CGO_ENABLED=""
GO_MINOR_VERSION=0

# Array of "variant|module|package" entries for packages holding files that only a build
# variant compiles.
VARIANT_PACKAGES=()

# Array of "variant|module" entries, one golangci-lint run each.
VARIANT_RUNS=()

# Array of module paths that had lint failures.
FAILED=()

# Count of modules skipped due to build constraints.
SKIPPED=0

# parse_args processes command-line arguments.
# Globals:
#   BUILD_TAGS - Set when --all is passed
# Arguments:
#   $@ - Command-line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --all)
                BUILD_TAGS="$ALL_BUILD_TAGS"
                shift
                ;;
            --tags)
                BUILD_TAGS="$2"
                shift 2
                ;;
            -h|--help)
                piko::log::info "Usage: $0 [--all] [--tags <tags>]"
                piko::log::detail "--all         Include all custom build tags (integration,vips,ffmpeg,...) and build variants (safe, js/wasm, other GOOS)"
                piko::log::detail "--tags <tags>  Specify custom build tags (comma-separated)"
                exit 0
                ;;
            *)
                piko::log::fatal "Unknown argument: $1 (use --help for usage)"
                ;;
        esac
    done
}

# verify_tools checks that required tools are installed.
# Returns:
#   Exits with code 1 if any tool is missing
verify_tools() {
    if ! piko::util::verify_binary "golangci-lint" "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0"; then
        exit 1
    fi
    if ! piko::util::verify_binary "jq" "brew install jq OR apt install jq"; then
        exit 1
    fi
}

# is_wasm_module checks if a module requires GOOS=js GOARCH=wasm to lint.
# Arguments:
#   $1 - Module path relative to PIKO_ROOT
# Returns:
#   0 if the module contains only js/wasm build-constrained files, 1 otherwise
is_wasm_module() {
    local mod_path="$1"
    local abs_path="${PIKO_ROOT}/${mod_path}"

    # Check if all .go files in the module have //go:build js && wasm
    local total_files
    total_files=$(find "$abs_path" -maxdepth 1 -name '*.go' | wc -l)
    if [[ "$total_files" -eq 0 ]]; then
        return 1
    fi

    local wasm_files
    wasm_files=$(grep -rl '//go:build js && wasm' "$abs_path"/*.go 2>/dev/null | wc -l)

    [[ "$total_files" -eq "$wasm_files" ]]
}

# discover_modules reads workspace module paths from go.work.
# Modules that require GOOS=js GOARCH=wasm are separated into WASM_MODULES
# so they can be linted with the correct cross-compilation environment.
# Globals:
#   PIKO_ROOT - Read
#   BUILD_TAGS - Read
#   WORKSPACE_MODULES - Set to array of every go.work module path
#   MODULES - Set to array of module paths
#   WASM_MODULES - Set to array of wasm module paths (only when --all)
discover_modules() {
    local gowork="${PIKO_ROOT}/go.work"

    if [[ ! -f "$gowork" ]]; then
        piko::log::fatal "No go.work file found at ${gowork}"
    fi

    while IFS= read -r mod_path; do
        WORKSPACE_MODULES+=("$mod_path")
    done < <(go work edit -json | jq -r '.Use[].DiskPath')

    if [[ ${#WORKSPACE_MODULES[@]} -eq 0 ]]; then
        piko::log::fatal "No modules found in go.work"
    fi

    # When --all is used, separate wasm modules for cross-compilation linting.
    for mod_path in "${WORKSPACE_MODULES[@]}"; do
        if [[ -n "$BUILD_TAGS" ]] && is_wasm_module "$mod_path"; then
            WASM_MODULES+=("$mod_path")
        else
            MODULES+=("$mod_path")
        fi
    done

    piko::log::info "Found ${#MODULES[@]} modules in go.work"
    if [[ ${#WASM_MODULES[@]} -gt 0 ]]; then
        piko::log::info "Found ${#WASM_MODULES[@]} wasm modules (will lint with GOOS=js GOARCH=wasm)"
    fi
}

# contains_element checks whether a value appears in a list.
# Arguments:
#   $1 - Value to look for
#   $@ - Remaining arguments form the list to search
# Returns:
#   0 if the value is present, 1 otherwise
contains_element() {
    local needle="$1"
    shift

    local element
    for element in "$@"; do
        if [[ "$element" == "$needle" ]]; then
            return 0
        fi
    done
    return 1
}

# owning_module prints the workspace module that contains a package directory, choosing
# the deepest module whose path is a prefix of the directory.
# Globals:
#   WORKSPACE_MODULES - Read
# Arguments:
#   $1 - Package directory relative to PIKO_ROOT (e.g. ./internal/colour)
owning_module() {
    local dir="$1"
    local best=""

    local mod_path
    for mod_path in "${WORKSPACE_MODULES[@]}"; do
        if [[ "$mod_path" != "." && "$dir" != "$mod_path" && "$dir" != "${mod_path}/"* ]]; then
            continue
        fi
        if [[ ${#mod_path} -gt ${#best} ]]; then
            best="$mod_path"
        fi
    done

    echo "$best"
}

# variant_settings prints the GOOS, GOARCH and comma-separated custom build tags ("-" for
# none) of a build variant. Every variant except notags builds with the requested
# BUILD_TAGS.
# Globals:
#   BUILD_TAGS - Read
#   HOST_GOOS, HOST_GOARCH - Read
# Arguments:
#   $1 - Variant name from BUILD_VARIANTS
variant_settings() {
    local tags="${BUILD_TAGS:--}"
    case "$1" in
        notags) echo "${HOST_GOOS} ${HOST_GOARCH} -" ;;
        safe) echo "${HOST_GOOS} ${HOST_GOARCH} ${BUILD_TAGS:+${BUILD_TAGS},}safe" ;;
        js) echo "js wasm ${tags}" ;;
        linux) echo "linux amd64 ${tags}" ;;
        windows) echo "windows amd64 ${tags}" ;;
        darwin) echo "darwin arm64 ${tags}" ;;
        freebsd) echo "freebsd amd64 ${tags}" ;;
        solaris) echo "solaris amd64 ${tags}" ;;
        *) piko::log::fatal "Unknown build variant: $1" ;;
    esac
}

# variant_lints_tests reports whether a build variant lints test files. Only notags does:
# it builds the host platform as an untagged lint would, so its tests type-check, whereas
# tests in the other variants' packages call helpers that only the native build defines.
# Arguments:
#   $1 - Variant name from BUILD_VARIANTS
# Returns:
#   0 when the variant lints test files, 1 otherwise
variant_lints_tests() {
    [[ "$1" == "notags" ]]
}

# platform_tags prints the build tags satisfied on a platform, space-separated and padded
# with a space at each end so callers can match " tag ".
# Globals:
#   UNIX_GOOS - Read
#   HOST_GOOS, HOST_GOARCH, HOST_CGO_ENABLED - Read
#   GO_MINOR_VERSION - Read
# Arguments:
#   $1 - GOOS
#   $2 - GOARCH
#   $3 - Comma-separated custom build tags, or "-" for none
platform_tags() {
    local goos="$1"
    local goarch="$2"
    local build_tags="$3"
    local tags=" ${goos} ${goarch} gc "
    local minor

    if [[ "$build_tags" != "-" ]]; then
        tags+="${build_tags//,/ } "
    fi
    if [[ "$UNIX_GOOS" == *" ${goos} "* ]]; then
        tags+="unix "
    fi
    if [[ "$goos" == "$HOST_GOOS" && "$goarch" == "$HOST_GOARCH" && "$HOST_CGO_ENABLED" == "1" ]]; then
        tags+="cgo "
    fi
    for ((minor = 1; minor <= GO_MINOR_VERSION; minor++)); do
        tags+="go1.${minor} "
    done

    echo "$tags"
}

# tokenise_constraint prints a //go:build expression with every operator and parenthesis
# separated by spaces, ready for constraint_holds.
# Arguments:
#   $1 - Build constraint expression (without the //go:build prefix)
tokenise_constraint() {
    sed -e 's/&&/ \&\& /g' -e 's/||/ || /g' -e 's/[()!]/ & /g' <<<"$1"
}

# constraint_holds evaluates a tokenised build constraint against a set of satisfied
# tags by rewriting each tag as 1 or 0 and evaluating the result arithmetically.
# Arguments:
#   $1 - Tokenised constraint from tokenise_constraint
#   $2 - Satisfied tags from platform_tags
# Returns:
#   0 when the constraint is satisfied, 1 otherwise
constraint_holds() {
    local satisfied="$2"
    local arithmetic=""
    local -a tokens
    local token

    read -ra tokens <<<"$1"
    for token in "${tokens[@]}"; do
        case "$token" in
            "&&"|"||"|"("|")"|"!")
                arithmetic+=" ${token}"
                ;;
            *)
                if [[ "$satisfied" == *" ${token} "* ]]; then
                    arithmetic+=" 1"
                else
                    arithmetic+=" 0"
                fi
                ;;
        esac
    done

    if (( arithmetic )); then
        return 0
    fi
    return 1
}

# list_constrained_files prints "directory<TAB>kind<TAB>expression" for each Go file with
# a //go:build line, where kind is "test" for a _test.go file and "source" otherwise.
# VCS, vendored, node_modules, testdata and testdata-modules trees are skipped, as are
# internal/esbuild, which golangci-lint excludes, and anything git ignores. Constraints
# implied only by a _GOOS or _GOARCH file name suffix are not seen, so such files must
# also carry a //go:build line.
list_constrained_files() {
    local file expression kind
    while IFS= read -r file; do
        expression=$(grep -m1 '^//go:build ' "$file")
        kind="source"
        if [[ "$file" == *_test.go ]]; then
            kind="test"
        fi
        printf '%s\t%s\t%s\n' "${file%/*}" "$kind" "${expression#//go:build }"
    done < <(find . \( -name .git -o -name node_modules -o -name vendor -o -name testdata -o -name testdata-modules -o -path ./internal/esbuild \) -prune \
        -o -type f -name '*.go' -exec grep -l '^//go:build ' {} + \
        | piko::util::reject_ignored_paths)
}

# discover_variant_packages finds the packages holding files that the native pass never
# compiles and assigns each such file to the first build variant that selects it and,
# for a test file, also lints tests. Packages are grouped per variant and owning module.
# Wasm modules are skipped because lint_wasm_module already lints all of their files
# under GOOS=js GOARCH=wasm.
# Globals:
#   BUILD_TAGS - Read
#   BUILD_VARIANTS - Read
#   WASM_MODULES - Read
#   HOST_GOOS, HOST_GOARCH, HOST_CGO_ENABLED, GO_MINOR_VERSION - Set from go env
#   VARIANT_PACKAGES - Set to "variant|module|package" entries
#   VARIANT_RUNS - Set to "variant|module" entries
discover_variant_packages() {
    local go_version
    {
        read -r HOST_GOOS
        read -r HOST_GOARCH
        read -r HOST_CGO_ENABLED
        read -r go_version
    } < <(go env GOOS GOARCH CGO_ENABLED GOVERSION)
    go_version="${go_version#go1.}"
    GO_MINOR_VERSION="${go_version%%[!0-9]*}"

    local native_tags variant goos goarch build_tags
    native_tags=$(platform_tags "$HOST_GOOS" "$HOST_GOARCH" "${BUILD_TAGS:--}")
    local -a variant_tags=()
    for variant in "${BUILD_VARIANTS[@]}"; do
        read -r goos goarch build_tags < <(variant_settings "$variant")
        variant_tags+=("$(platform_tags "$goos" "$goarch" "$build_tags")")
    done

    local dir kind expression tokens mod_path index entry
    while IFS=$'\t' read -r dir kind expression; do
        tokens=$(tokenise_constraint "$expression")
        if constraint_holds "$tokens" "$native_tags"; then
            continue
        fi

        mod_path=$(owning_module "$dir")
        if [[ -z "$mod_path" ]] || contains_element "$mod_path" "${WASM_MODULES[@]}"; then
            continue
        fi

        for index in "${!BUILD_VARIANTS[@]}"; do
            variant="${BUILD_VARIANTS[$index]}"
            if [[ "$kind" == "test" ]] && ! variant_lints_tests "$variant"; then
                continue
            fi
            if ! constraint_holds "$tokens" "${variant_tags[$index]}"; then
                continue
            fi
            entry="${variant}|${mod_path}|${dir}"
            if ! contains_element "$entry" "${VARIANT_PACKAGES[@]}"; then
                VARIANT_PACKAGES+=("$entry")
            fi
            if ! contains_element "${variant}|${mod_path}" "${VARIANT_RUNS[@]}"; then
                VARIANT_RUNS+=("${variant}|${mod_path}")
            fi
            break
        done
    done < <(list_constrained_files)

    if [[ ${#VARIANT_RUNS[@]} -gt 0 ]]; then
        piko::log::info "Found ${#VARIANT_PACKAGES[@]} package and build-variant pairs (will lint in ${#VARIANT_RUNS[@]} extra runs)"
    fi
}

# lint_module runs golangci-lint against a single module path.
# Modules whose files are all excluded by build constraints (e.g. //go:build
# integration, wasm, vips, ffmpeg) produce a "no go files to analyze" error
# from golangci-lint. These are skipped rather than counted as failures.
# Globals:
#   BUILD_TAGS - Read
#   LINT_ARGS - Read
#   FAILED - Modified with failed module paths
#   SKIPPED - Incremented for build-constrained modules
# Arguments:
#   $1 - Module path relative to PIKO_ROOT
lint_module() {
    local mod_path="$1"
    local output
    local rc=0
    local -a args=("${LINT_ARGS[@]}")

    if [[ -n "$BUILD_TAGS" ]]; then
        args+=("--build-tags" "$BUILD_TAGS")
    fi
    args+=("${mod_path}/...")

    output=$(golangci-lint "${args[@]}" 2>&1) || rc=$?

    if [[ $rc -eq 0 ]]; then
        piko::log::success "$mod_path"
        return
    fi

    if echo "$output" | grep -q "no go files to analyze"; then
        piko::log::warn "$mod_path (skipped: no files match build constraints)"
        SKIPPED=$((SKIPPED + 1))
        return
    fi

    echo "$output"
    piko::log::error "$mod_path"
    FAILED+=("$mod_path")
}

# lint_wasm_module runs golangci-lint against a wasm module using GOOS=js
# GOARCH=wasm cross-compilation environment.
# Globals:
#   BUILD_TAGS - Read
#   LINT_ARGS - Read
#   FAILED - Modified with failed module paths
# Arguments:
#   $1 - Module path relative to PIKO_ROOT
lint_wasm_module() {
    local mod_path="$1"
    local output
    local rc=0
    local -a args=("${LINT_ARGS[@]}")

    if [[ -n "$BUILD_TAGS" ]]; then
        args+=("--build-tags" "$BUILD_TAGS")
    fi
    args+=("${mod_path}/...")

    output=$(GOOS=js GOARCH=wasm golangci-lint "${args[@]}" 2>&1) || rc=$?

    if [[ $rc -eq 0 ]]; then
        piko::log::success "$mod_path (GOOS=js GOARCH=wasm)"
        return
    fi

    echo "$output"
    piko::log::error "$mod_path (GOOS=js GOARCH=wasm)"
    FAILED+=("$mod_path")
}

# lint_variant_run runs golangci-lint once over the packages a build variant selects
# within one module, using that variant's GOOS, GOARCH and build tags.
#
# Outside the notags variant, test files are excluded because some tests in these
# packages call helpers from the native build and so do not type-check under the variant,
# and the unused linter is disabled because helpers shared by both builds look dead when
# only the variant's callers are compiled; the native pass already reports code that is
# genuinely unused. The notags variant builds the host platform as an untagged lint
# would, so it lints test files with every linter enabled.
# Globals:
#   LINT_ARGS - Read
#   VARIANT_PACKAGES - Read
#   FAILED - Modified with failed runs
# Arguments:
#   $1 - Variant name from BUILD_VARIANTS
#   $2 - Module path relative to PIKO_ROOT
lint_variant_run() {
    local variant="$1"
    local mod_path="$2"
    local goos goarch tags output entry
    local rc=0

    read -r goos goarch tags < <(variant_settings "$variant")

    local -a args=("${LINT_ARGS[@]}")
    if variant_lints_tests "$variant"; then
        args+=("--tests=true")
    else
        args+=("--tests=false" "--disable" "unused")
    fi
    if [[ "$tags" != "-" ]]; then
        args+=("--build-tags" "$tags")
    fi
    for entry in "${VARIANT_PACKAGES[@]}"; do
        if [[ "${entry%|*}" == "${variant}|${mod_path}" ]]; then
            args+=("${entry##*|}")
        fi
    done

    local tags_label="$tags"
    if [[ "$tags" == "-" ]]; then
        tags_label="none"
    fi
    local label="${mod_path} (${variant}: GOOS=${goos} GOARCH=${goarch} tags=${tags_label})"
    output=$(GOOS="$goos" GOARCH="$goarch" golangci-lint "${args[@]}" 2>&1) || rc=$?

    if [[ $rc -eq 0 ]]; then
        piko::log::success "$label"
        return
    fi

    echo "$output"
    piko::log::error "$label"
    FAILED+=("$label")
}

# lint_modules runs golangci-lint against each workspace module.
# Wasm modules and the build-variant packages of native modules are linted separately
# with their cross-compilation environment.
# Globals:
#   MODULES - Read
#   WASM_MODULES - Read
#   VARIANT_RUNS - Read
#   FAILED - Modified with failed module paths
#   SKIPPED - Modified with skipped module count
lint_modules() {
    local i=0
    local total=$(( ${#MODULES[@]} + ${#WASM_MODULES[@]} + ${#VARIANT_RUNS[@]} ))

    for mod_path in "${MODULES[@]}"; do
        i=$((i + 1))
        piko::log::step "$i" "$total" "$mod_path"
        lint_module "$mod_path"
    done

    for mod_path in "${WASM_MODULES[@]}"; do
        i=$((i + 1))
        piko::log::step "$i" "$total" "$mod_path (wasm)"
        lint_wasm_module "$mod_path"
    done

    local run
    for run in "${VARIANT_RUNS[@]}"; do
        i=$((i + 1))
        piko::log::step "$i" "$total" "${run#*|} (${run%%|*} variant)"
        lint_variant_run "${run%%|*}" "${run#*|}"
    done
}

# print_summary displays lint results across all modules.
# Globals:
#   MODULES - Read
#   WASM_MODULES - Read
#   VARIANT_RUNS - Read
#   FAILED - Read
#   SKIPPED - Read
print_summary() {
    local total=$(( ${#MODULES[@]} + ${#WASM_MODULES[@]} + ${#VARIANT_RUNS[@]} ))
    local failed=${#FAILED[@]}
    local passed=$((total - failed - SKIPPED))

    piko::log::blank
    piko::log::header "Summary"
    piko::log::info "Total runs: $total (${#MODULES[@]} modules, ${#WASM_MODULES[@]} wasm modules, ${#VARIANT_RUNS[@]} build-variant runs)"
    piko::log::info "Passed: $passed"
    piko::log::info "Skipped: $SKIPPED (build constraints)"
    piko::log::info "Failed: $failed"

    if [[ $failed -gt 0 ]]; then
        piko::log::blank
        piko::log::error "The following modules have lint issues:"
        for mod in "${FAILED[@]}"; do
            piko::log::detail "- $mod"
        done
        piko::log::blank
        piko::log::info "Run golangci-lint directly on failing modules for details:"
        piko::log::detail "golangci-lint run <module>/..."
        piko::log::detail "golangci-lint run <package>    (notags variant)"
        piko::log::detail "GOOS=<os> GOARCH=<arch> golangci-lint run --tests=false --disable unused --build-tags <tags> <package>"
        exit 1
    fi

    piko::log::success "All modules passed golangci-lint!"
}

# main lints all Go workspace modules.
# Arguments:
#   $@ - Optional flags (--all, --tags)
main() {
    parse_args "$@"
    verify_tools

    piko::log::header "Linting Go workspace modules with golangci-lint"
    if [[ -n "$BUILD_TAGS" ]]; then
        piko::log::info "Build tags: $BUILD_TAGS"
    fi

    discover_modules
    if [[ -n "$BUILD_TAGS" ]]; then
        discover_variant_packages
    fi
    piko::log::blank
    lint_modules
    print_summary
}

main "$@"
