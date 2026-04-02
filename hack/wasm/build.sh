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

# hack/wasm/build.sh - Build, optimise, and compress the Piko WASM binary
#
# This script is called by Makefile targets:
#   make build-wasm        Build, optimise, and compress
#   make build-wasm-clean  Clean build artefacts

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Output directory for WASM build artefacts.
WASM_OUT_DIR="${PIKO_ROOT}/bin/wasm"

# Path to the embedded stdlib binary used by the WASM module.
STDLIB_BIN="${PIKO_ROOT}/internal/wasm/wasm_data/stdlib.bin"

# Path to the raw (unoptimised) WASM binary.
WASM_RAW="${WASM_OUT_DIR}/piko.wasm"

# Path to the optimised WASM binary.
WASM_OPT="${WASM_OUT_DIR}/piko.opt.wasm"

# Path to the final WASM binary (optimised if wasm-opt is available, raw otherwise).
WASM_FINAL="${WASM_OUT_DIR}/piko.final.wasm"

# report_size prints the file size of a build artefact.
# Arguments:
#   $1 - Label to display
#   $2 - Path to the file
report_size() {
    local label="$1"
    local file="$2"

    if [[ -f "$file" ]]; then
        local size
        size=$(du -h "$file" | cut -f1)
        piko::log::detail "$label: $size"
    fi
}

# build_stdlib generates the embedded standard library type data.
# This must run before WASM compilation so the binary is up to date.
build_stdlib() {
    piko::log::header "Building Embedded Stdlib Types"

    piko::log::info "Generating stdlib.bin..."
    if go run ./cmd/stdlib-generator -output "$STDLIB_BIN"; then
        report_size "stdlib.bin" "$STDLIB_BIN"
        piko::log::success "Stdlib types generated"
    else
        piko::log::fatal "Failed to generate stdlib types"
    fi
}

# compile_wasm builds the WASM binary with stripped debug info.
# Uses -ldflags="-s -w" to remove symbol table and DWARF debug info.
compile_wasm() {
    piko::log::header "Compiling WASM Binary"

    piko::util::ensure_dir "$WASM_OUT_DIR"

    piko::log::info "Building with GOOS=js GOARCH=wasm -trimpath -ldflags=\"-s -w\"..."
    if GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o "$WASM_RAW" ./cmd/wasm; then
        report_size "Raw WASM" "$WASM_RAW"
        piko::log::success "WASM binary compiled"
    else
        piko::log::fatal "Failed to compile WASM binary"
    fi
}

# optimise_wasm runs wasm-opt to shrink the binary.
# Falls back gracefully if wasm-opt is not installed.
optimise_wasm() {
    piko::log::header "Optimising WASM Binary"

    if ! command -v wasm-opt &>/dev/null; then
        piko::log::warn "wasm-opt not found - skipping optimisation"
        piko::log::info "Install binaryen for wasm-opt: https://github.com/WebAssembly/binaryen"
        cp "$WASM_RAW" "$WASM_FINAL"
        return
    fi

    # Go's WASM output uses post-MVP features: bulk memory operations
    # (memory.copy/fill), non-trapping float-to-int conversions
    # (i64.trunc_sat_f64_s), sign extension, and mutable globals.
    piko::log::info "Running wasm-opt -Oz..."
    if wasm-opt -Oz \
        --enable-bulk-memory \
        --enable-nontrapping-float-to-int \
        --enable-sign-ext \
        --enable-mutable-globals \
        -o "$WASM_OPT" "$WASM_RAW"; then
        report_size "Optimised WASM" "$WASM_OPT"
        cp "$WASM_OPT" "$WASM_FINAL"
        piko::log::success "WASM binary optimised"
    else
        piko::log::warn "wasm-opt failed - using unoptimised binary"
        cp "$WASM_RAW" "$WASM_FINAL"
    fi
}

# compress_wasm creates gzip and brotli compressed variants.
# Brotli is skipped gracefully if brotli is not installed.
compress_wasm() {
    piko::log::header "Compressing WASM Binary"

    piko::log::info "Creating gzip compressed variant..."
    if gzip -9 -k -f "$WASM_FINAL"; then
        report_size "gzip" "${WASM_FINAL}.gz"
        piko::log::success "gzip compression complete"
    else
        piko::log::warn "gzip compression failed"
    fi

    if ! command -v brotli &>/dev/null; then
        piko::log::warn "brotli not found - skipping brotli compression"
        piko::log::info "Install brotli: sudo apt install brotli (or brew install brotli)"
        return
    fi

    piko::log::info "Creating brotli compressed variant..."
    if brotli -9 -k -f "$WASM_FINAL"; then
        report_size "brotli" "${WASM_FINAL}.br"
        piko::log::success "brotli compression complete"
    else
        piko::log::warn "brotli compression failed"
    fi
}

# copy_wasm_exec copies wasm_exec.js from GOROOT to the output directory.
copy_wasm_exec() {
    piko::log::header "Copying wasm_exec.js"

    local goroot
    goroot="$(go env GOROOT)"

    local wasm_exec=""
    if [[ -f "$goroot/lib/wasm/wasm_exec.js" ]]; then
        wasm_exec="$goroot/lib/wasm/wasm_exec.js"
    elif [[ -f "$goroot/misc/wasm/wasm_exec.js" ]]; then
        wasm_exec="$goroot/misc/wasm/wasm_exec.js"
    fi

    if [[ -z "$wasm_exec" ]]; then
        piko::log::warn "wasm_exec.js not found in GOROOT ($goroot)"
        return
    fi

    # The source file may be read-only, so remove any existing copy first.
    rm -f "$WASM_OUT_DIR/wasm_exec.js"
    cp "$wasm_exec" "$WASM_OUT_DIR/"
    report_size "wasm_exec.js" "$WASM_OUT_DIR/wasm_exec.js"
    piko::log::success "Copied wasm_exec.js"

    piko::log::info "Compressing wasm_exec.js..."
    if gzip -9 -k -f "$WASM_OUT_DIR/wasm_exec.js"; then
        report_size "wasm_exec.js gzip" "$WASM_OUT_DIR/wasm_exec.js.gz"
    fi
    if command -v brotli &>/dev/null; then
        if brotli -9 -k -f "$WASM_OUT_DIR/wasm_exec.js"; then
            report_size "wasm_exec.js brotli" "$WASM_OUT_DIR/wasm_exec.js.br"
        fi
    fi
}

# print_summary displays the final build artefact sizes.
print_summary() {
    piko::log::header "Build Summary"

    report_size "Raw WASM       " "$WASM_RAW"
    report_size "Optimised WASM " "$WASM_OPT"
    report_size "Final WASM     " "$WASM_FINAL"
    report_size "gzip           " "${WASM_FINAL}.gz"
    report_size "brotli         " "${WASM_FINAL}.br"

    piko::log::blank
    piko::log::info "Output directory: $WASM_OUT_DIR"
}

# clean_build removes all WASM build artefacts.
clean_build() {
    piko::log::header "Cleaning WASM Build Artefacts"

    if [[ -d "$WASM_OUT_DIR" ]]; then
        rm -rf "${WASM_OUT_DIR:?}"
        piko::log::success "Build artefacts cleaned"
    else
        piko::log::info "Nothing to clean"
    fi
}

# main handles build or clean commands.
# Arguments:
#   $1 - Command to execute: build (default) or clean
main() {
    local command="${1:-build}"

    case "$command" in
        build)
            build_stdlib
            compile_wasm
            optimise_wasm
            compress_wasm
            copy_wasm_exec
            print_summary
            piko::log::blank
            piko::log::success "WASM build complete!"
            ;;

        clean)
            clean_build
            ;;

        *)
            piko::log::fatal "Unknown command: $command (expected: build, clean)"
            ;;
    esac
}

main "$@"
