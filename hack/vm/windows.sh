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

# hack/vm/windows.sh - Windows VM management and testing
#
# This script is called by Makefile targets:
#   make vm-windows-up         Start and provision Windows VM
#   make vm-windows-halt       Stop Windows VM
#   make vm-windows-destroy    Destroy Windows VM
#   make vm-windows-test       Cross-compile and run Go tests on Windows
#   make vm-windows-outlook    Render emails in Outlook and capture screenshots
#   make vm-windows-status     Show Windows VM status
#   make vm-windows-ssh        Open SSH session to Windows VM

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the Windows VM directory.
VM_DIR="$(piko::vm::vagrant_dir)"

# Email test data containing golden HTML files.
EMAIL_TESTDATA="${PIKO_ROOT}/internal/render/render_test/email/testdata"

# Directory for Outlook screenshots (actual, golden, diff).
SCREENSHOT_DIR="${VM_DIR}/screenshots"

# transfer_licences copies licence key files to the Windows guest.
# Globals:
#   VM_DIR - Read
transfer_licences() {
    local licence_dir="${VM_DIR}/licences"

    if ! compgen -G "${licence_dir}/*.key" > /dev/null 2>&1; then
        return 0
    fi

    piko::log::info "Transferring licence files to guest..."
    piko::vm::ssh "New-Item -ItemType Directory -Force -Path C:\Users\vagrant\licences | Out-Null"

    for keyfile in "${licence_dir}"/*.key; do
        local filename
        filename="$(basename "$keyfile")"
        piko::vm::scp_to "$keyfile" "C:\\Users\\vagrant\\licences\\${filename}"
        piko::log::info "  Transferred ${filename}"
    done
}

# validate_ssh checks that the Windows VM is reachable over SSH.
validate_ssh() {
    piko::log::info "Validating SSH connectivity..."

    if piko::vm::ssh "Write-Output 'SSH connection successful'" > /dev/null 2>&1; then
        piko::log::success "Windows VM is running and accessible via SSH"
    else
        piko::log::fatal "VM is running but SSH connection failed"
    fi
}

# start_vm starts and provisions the Windows VM.
# Globals:
#   VM_DIR - Read
start_vm() {
    piko::vm::require_vagrant
    piko::vm::ensure_licences_dir

    piko::log::header "Starting Windows VM"

    cd "$VM_DIR" || exit 1

    piko::log::info "Running vagrant up (this may take a while on first run)..."
    vagrant up --provider=libvirt

    piko::log::info "Refreshing SSH config..."
    piko::vm::ssh_config

    transfer_licences

    # Re-provision if licence files were transferred (they need to be picked
    # up by the provisioning scripts). Skip if no licences were found.
    local licence_dir="${VM_DIR}/licences"
    if compgen -G "${licence_dir}/*.key" > /dev/null 2>&1; then
        piko::log::info "Re-provisioning to apply licence files..."
        cd "$VM_DIR" || exit 1
        vagrant provision
    fi

    validate_ssh
}

# stop_vm stops the Windows VM gracefully.
# Globals:
#   VM_DIR - Read
stop_vm() {
    piko::log::header "Stopping Windows VM"

    cd "$VM_DIR" || exit 1
    vagrant halt

    piko::log::success "Windows VM stopped"
}

# destroy_vm destroys the Windows VM and all state.
# Globals:
#   VM_DIR - Read
destroy_vm() {
    piko::log::header "Destroying Windows VM"

    cd "$VM_DIR" || exit 1
    vagrant destroy -f
    rm -f "${PIKO_VM_SSH_CONFIG}"

    piko::log::success "Windows VM destroyed"
}

# show_status prints the current VM status.
# Globals:
#   VM_DIR - Read
show_status() {
    cd "$VM_DIR" || exit 1
    vagrant status
}

# open_ssh opens an interactive SSH session to the VM.
open_ssh() {
    piko::vm::require_running
    piko::vm::ssh
}

# cross_compile_tests builds Go test binaries for Windows.
# Globals:
#   VM_DIR - Read
# Arguments:
#   $1 - Package pattern
# Outputs:
#   Sets COMPILE_DIR, SUCCESSFUL_BINARIES, COMPILED_COUNT, SKIPPED_COUNT
cross_compile_tests() {
    local package_pattern="$1"

    piko::log::info "Discovering test packages..."
    local packages
    packages=$(go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' "$package_pattern" 2>/dev/null)

    if [[ -z "$packages" ]]; then
        piko::log::warn "No test packages found matching: $package_pattern"
        return 1
    fi

    local package_count
    package_count=$(echo "$packages" | wc -l)
    piko::log::info "Found ${package_count} packages with tests"

    piko::log::info "Cross-compiling test binaries for windows/amd64..."
    COMPILE_DIR="${VM_DIR}/tmp"
    mkdir -p "$COMPILE_DIR"
    rm -f "${COMPILE_DIR}"/*.test.exe

    COMPILED_COUNT=0
    SKIPPED_COUNT=0
    SUCCESSFUL_BINARIES=()
    local failed_packages=()

    while IFS= read -r package; do
        local binary_name
        binary_name=$(echo "$package" | tr '/' '_')
        local output="${COMPILE_DIR}/${binary_name}.test.exe"

        if GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o "$output" "$package" 2>/dev/null; then
            COMPILED_COUNT=$((COMPILED_COUNT + 1))
            SUCCESSFUL_BINARIES+=("$binary_name")
        else
            SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
            failed_packages+=("$package")
        fi
    done <<< "$packages"

    piko::log::info "Compiled: ${COMPILED_COUNT}, Skipped: ${SKIPPED_COUNT}"

    if [[ ${#failed_packages[@]} -gt 0 ]]; then
        piko::log::warn "Packages that failed to cross-compile:"
        for package in "${failed_packages[@]}"; do
            piko::log::detail "$package"
        done
    fi

    if [[ $COMPILED_COUNT -eq 0 ]]; then
        piko::log::warn "No test binaries compiled successfully"
        return 1
    fi
}

# transfer_test_binaries copies compiled test binaries to the Windows VM.
# Globals:
#   COMPILE_DIR - Read
#   SUCCESSFUL_BINARIES - Read
transfer_test_binaries() {
    piko::log::info "Transferring ${COMPILED_COUNT} test binaries to Windows VM..."
    piko::vm::ssh "Remove-Item -Path C:\piko-tests\*.test.exe -Force -ErrorAction SilentlyContinue"

    for binary_name in "${SUCCESSFUL_BINARIES[@]}"; do
        piko::vm::scp_to "${COMPILE_DIR}/${binary_name}.test.exe" "C:\\piko-tests\\${binary_name}.test.exe"
    done
}

# run_remote_tests executes test binaries on the Windows VM and collects results.
# Globals:
#   SUCCESSFUL_BINARIES - Read
#   COMPILED_COUNT - Read
# Arguments:
#   $1 - Short flag (e.g. "-test.short" or "")
run_remote_tests() {
    local short_flag="$1"

    piko::log::info "Running tests on Windows..."
    piko::log::footer

    local passed=0
    local total_failed=0
    local failed_binaries=()

    for binary_name in "${SUCCESSFUL_BINARIES[@]}"; do
        local display_name
        display_name=$(echo "$binary_name" | sed 's/piko\.sh_piko_//' | tr '_' '/')

        piko::log::info "Testing: ${display_name}"

        if piko::vm::ssh "& 'C:\\piko-tests\\${binary_name}.test.exe' '-test.v' ${short_flag:+'-test.short'}"; then
            passed=$((passed + 1))
        else
            total_failed=$((total_failed + 1))
            failed_binaries+=("$display_name")
        fi

        echo
    done

    piko::log::header "Windows Test Results"
    piko::log::info "Passed: ${passed}/${COMPILED_COUNT}"

    if [[ $total_failed -gt 0 ]]; then
        piko::log::error "Failed: ${total_failed}/${COMPILED_COUNT}"
        for name in "${failed_binaries[@]}"; do
            piko::log::detail "FAIL: $name"
        done
        exit 1
    fi

    piko::log::success "All Windows tests passed"

    rm -rf "$COMPILE_DIR"
}

# discover_email_cases finds email test HTML files matching a filter.
# Globals:
#   EMAIL_TESTDATA - Read
# Arguments:
#   $1 - Filter pattern (optional)
# Outputs:
#   Sets HTML_FILES, TEST_NAMES
discover_email_cases() {
    local filter_pattern="${1:-}"

    HTML_FILES=()
    TEST_NAMES=()

    for testdir in "${EMAIL_TESTDATA}"/*/; do
        local dirname
        dirname="$(basename "$testdir")"
        local html_file="${testdir}golden/actual.html"

        if [[ ! -f "$html_file" ]]; then
            continue
        fi

        if [[ -n "$filter_pattern" ]] && [[ "$dirname" != *"$filter_pattern"* ]]; then
            continue
        fi

        HTML_FILES+=("$html_file")
        TEST_NAMES+=("$dirname")
    done
}

# transfer_email_files copies HTML emails and the screenshot script to the guest.
# Globals:
#   HTML_FILES - Read
#   TEST_NAMES - Read
#   VM_DIR - Read
transfer_email_files() {
    piko::log::info "Transferring files to Windows VM..."
    piko::vm::ssh "Remove-Item -Path C:\piko-emails\* -Force -ErrorAction SilentlyContinue"
    piko::vm::ssh "Remove-Item -Path C:\piko-screenshots\* -Force -ErrorAction SilentlyContinue"

    for i in "${!HTML_FILES[@]}"; do
        piko::vm::scp_to "${HTML_FILES[$i]}" "C:\\piko-emails\\${TEST_NAMES[$i]}.html"
    done

    piko::vm::scp_to "${VM_DIR}/scripts/screenshot-email.ps1" "C:\\piko-scripts\\screenshot-email.ps1"
}

# capture_screenshots runs the Outlook screenshot script for each email.
# Globals:
#   TEST_NAMES - Read
# Outputs:
#   Sets CAPTURED_COUNT, CAPTURE_FAILED_COUNT
capture_screenshots() {
    piko::log::info "Capturing Outlook screenshots..."
    CAPTURED_COUNT=0
    CAPTURE_FAILED_COUNT=0

    for i in "${!TEST_NAMES[@]}"; do
        local name="${TEST_NAMES[$i]}"
        piko::log::step "$((i + 1))" "${#TEST_NAMES[@]}" "$name"

        piko::vm::ssh "Remove-Item -Path C:\piko-screenshots\${name}.done -Force -ErrorAction SilentlyContinue" 2>/dev/null

        piko::vm::ssh "schtasks /Create /TN PikoScreenshot /TR 'powershell -ExecutionPolicy Bypass -File C:\piko-scripts\screenshot-email.ps1 -HtmlPath C:\piko-emails\${name}.html -OutputPath C:\piko-screenshots\${name}.png' /SC ONCE /ST 00:00 /RU vagrant /IT /F" > /dev/null 2>&1

        piko::vm::ssh "schtasks /Run /TN PikoScreenshot" > /dev/null 2>&1

        local timeout=30
        local elapsed=0
        while [[ $elapsed -lt $timeout ]]; do
            if piko::vm::ssh "Test-Path C:\piko-screenshots\${name}.done" 2>/dev/null | grep -qi "true"; then
                break
            fi
            sleep 1
            elapsed=$((elapsed + 1))
        done

        if [[ $elapsed -ge $timeout ]]; then
            piko::log::error "  Timeout capturing $name"
            CAPTURE_FAILED_COUNT=$((CAPTURE_FAILED_COUNT + 1))
        else
            CAPTURED_COUNT=$((CAPTURED_COUNT + 1))
        fi
    done

    piko::vm::ssh "schtasks /Delete /TN PikoScreenshot /F" > /dev/null 2>&1
}

# retrieve_screenshots copies captured PNGs from the guest to the host.
# Globals:
#   TEST_NAMES - Read
#   SCREENSHOT_DIR - Read
retrieve_screenshots() {
    piko::log::info "Retrieving screenshots..."
    mkdir -p "${SCREENSHOT_DIR}/actual"

    for name in "${TEST_NAMES[@]}"; do
        piko::vm::scp_from "C:\\piko-screenshots\\${name}.png" "${SCREENSHOT_DIR}/actual/${name}.png" 2>/dev/null
    done
}

# update_golden_screenshots copies actual screenshots to the golden directory.
# Globals:
#   TEST_NAMES - Read
#   SCREENSHOT_DIR - Read
update_golden_screenshots() {
    piko::log::info "Updating golden screenshots..."
    mkdir -p "${SCREENSHOT_DIR}/golden"

    for name in "${TEST_NAMES[@]}"; do
        local actual="${SCREENSHOT_DIR}/actual/${name}.png"
        if [[ -f "$actual" ]]; then
            cp "$actual" "${SCREENSHOT_DIR}/golden/${name}.png"
        fi
    done

    piko::log::success "Golden screenshots updated"
}

# compare_screenshots checks actual screenshots against golden files.
# Globals:
#   TEST_NAMES - Read
#   SCREENSHOT_DIR - Read
compare_screenshots() {
    if [[ ! -d "${SCREENSHOT_DIR}/golden" ]] || [[ -z "$(ls -A "${SCREENSHOT_DIR}/golden" 2>/dev/null)" ]]; then
        piko::log::warn "No golden screenshots to compare against"
        piko::log::detail "Generate them with: make vm-windows-outlook UPDATE=1"
        return 0
    fi

    piko::log::info "Comparing against golden screenshots..."

    if ! command -v compare &>/dev/null; then
        piko::log::warn "ImageMagick 'compare' not found - skipping visual comparison"
        piko::log::detail "Install with: sudo dnf install ImageMagick"
        return 0
    fi

    local threshold="${PIKO_VM_SCREENSHOT_THRESHOLD:-0.05}"
    local comparison_passed=0
    local comparison_failed=0
    local comparison_failures=()

    mkdir -p "${SCREENSHOT_DIR}/diff"

    for name in "${TEST_NAMES[@]}"; do
        local actual="${SCREENSHOT_DIR}/actual/${name}.png"
        local golden="${SCREENSHOT_DIR}/golden/${name}.png"
        local diff_file="${SCREENSHOT_DIR}/diff/${name}.png"

        if [[ ! -f "$actual" ]]; then
            continue
        fi

        if [[ ! -f "$golden" ]]; then
            piko::log::warn "  No golden file for: $name"
            continue
        fi

        local rmse
        rmse=$(compare -metric RMSE "$actual" "$golden" "$diff_file" 2>&1 | grep -oP '\([\d.]+\)' | tr -d '()')

        if [[ -z "$rmse" ]]; then
            piko::log::error "  Failed to compare: $name"
            comparison_failed=$((comparison_failed + 1))
            comparison_failures+=("$name")
            continue
        fi

        local exceeds
        exceeds=$(awk "BEGIN { print ($rmse > $threshold) }")

        if [[ "$exceeds" == "1" ]]; then
            piko::log::error "  DIFF: $name (RMSE: $rmse, threshold: $threshold)"
            piko::log::detail "  See: ${diff_file}"
            comparison_failed=$((comparison_failed + 1))
            comparison_failures+=("$name")
        else
            comparison_passed=$((comparison_passed + 1))
        fi
    done

    piko::log::footer
    piko::log::info "Comparison: ${comparison_passed} passed, ${comparison_failed} failed"

    if [[ $comparison_failed -gt 0 ]]; then
        piko::log::error "Screenshot comparison failures:"
        for name in "${comparison_failures[@]}"; do
            piko::log::detail "  $name"
        done
        piko::log::detail ""
        piko::log::detail "View diffs in: ${SCREENSHOT_DIR}/diff/"
        piko::log::detail "Update golden files with: make vm-windows-outlook UPDATE=1"
        exit 1
    fi

    piko::log::success "All screenshots match golden files"
}

# main handles VM commands.
# Arguments:
#   $1 - Command to execute: up, halt, destroy, status, ssh, test, outlook
main() {
    local command="${1:-up}"
    shift || true

    case "$command" in
        up)
            start_vm
            ;;

        halt)
            stop_vm
            ;;

        destroy)
            destroy_vm
            ;;

        status)
            show_status
            ;;

        ssh)
            open_ssh
            ;;

        test)
            piko::vm::require_running

            local short_flag="-test.short"
            local package_pattern="piko.sh/piko/..."

            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --short) short_flag="-test.short"; shift ;;
                    --full)  short_flag=""; shift ;;
                    *)       package_pattern="$1"; shift ;;
                esac
            done

            piko::log::header "Windows Go Tests"

            cross_compile_tests "$package_pattern"
            transfer_test_binaries
            run_remote_tests "$short_flag"
            ;;

        outlook)
            piko::vm::require_running

            local update_golden=false
            local filter_pattern=""

            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --update) update_golden=true; shift ;;
                    *)        filter_pattern="$1"; shift ;;
                esac
            done

            piko::log::header "Outlook Email Rendering"

            discover_email_cases "$filter_pattern"

            if [[ ${#HTML_FILES[@]} -eq 0 ]]; then
                piko::log::warn "No email test cases found"
                if [[ -n "$filter_pattern" ]]; then
                    piko::log::detail "Filter: $filter_pattern"
                fi
                return 0
            fi

            piko::log::info "Found ${#HTML_FILES[@]} email test cases"

            transfer_email_files
            capture_screenshots
            retrieve_screenshots

            piko::log::info "Captured: ${CAPTURED_COUNT}, Failed: ${CAPTURE_FAILED_COUNT}"

            if [[ "$update_golden" == "true" ]]; then
                update_golden_screenshots
            else
                compare_screenshots
            fi
            ;;

        *)
            piko::log::fatal "Unknown command: $command (expected: up, halt, destroy, status, ssh, test, outlook)"
            ;;
    esac
}

main "$@"
