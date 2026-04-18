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

# hack/generate/protoc.sh - Generate Go code from Protocol Buffers
#
# Generates Go code from all .proto Protocol Buffers files in the project.
# Downloads protoc automatically if not present.
#
# Usage:
#   ./hack/generate/protoc.sh

# shellcheck source=../lib/init.sh
source "$(dirname "$0")/../lib/init.sh"

# Path to the protoc binary.
PROTOC="${PIKO_ROOT}/.tools/protoc"

# Pinned protoc-gen-go version for reproducible code generation.
PROTOC_GEN_GO_VERSION="v1.36.6"

# Pinned protoc-gen-go-grpc version for reproducible code generation.
PROTOC_GEN_GO_GRPC_VERSION="v1.6.0"

# Array of Protocol Buffers file paths.
PROTO_FILES=(
    "internal/monitoring/monitoring_api/monitoring.proto"
)

# ensure_protoc downloads protoc if not already present.
ensure_protoc() {
    if [[ ! -x "$PROTOC" ]]; then
        piko::log::info "protoc not found, downloading..."
        "${PIKO_ROOT}/hack/tools/protoc.sh"
    fi
}

# ensure_protoc_go_plugin checks protoc-gen-go and protoc-gen-go-grpc are
# installed at the pinned versions. Reinstalls if the version does not match.
ensure_protoc_go_plugin() {
    local current_go_version
    current_go_version=$(protoc-gen-go --version 2>/dev/null | awk '{print $2}')
    if [[ "$current_go_version" != "$PROTOC_GEN_GO_VERSION" ]]; then
        piko::log::info "Installing protoc-gen-go@${PROTOC_GEN_GO_VERSION} (current: ${current_go_version:-not found})..."
        if ! go install "google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}"; then
            piko::log::fatal "Failed to install protoc-gen-go@${PROTOC_GEN_GO_VERSION}"
        fi
    fi

    local current_grpc_version
    current_grpc_version=$(protoc-gen-go-grpc --version 2>/dev/null | awk '{print $2}')
    if [[ "$current_grpc_version" != "$PROTOC_GEN_GO_GRPC_VERSION" ]]; then
        piko::log::info "Installing protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION} (current: ${current_grpc_version:-not found})..."
        if ! go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}"; then
            piko::log::fatal "Failed to install protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}"
        fi
    fi
}

# generate_protoc runs protoc for all proto files.
# Globals:
#   PROTOC - Read
#   PROTO_FILES - Read
#   PIKO_ROOT - Read
generate_protoc() {
    local proto_count=${#PROTO_FILES[@]}
    local current=0

    for proto in "${PROTO_FILES[@]}"; do
        current=$((current + 1))
        local proto_path="${PIKO_ROOT}/${proto}"
        local proto_dir
        proto_dir=$(dirname "$proto_path")

        piko::log::step "$current" "$proto_count" "Generating: $proto"

        if [[ ! -f "$proto_path" ]]; then
            piko::log::warn "Proto file not found: $proto_path"
            continue
        fi

        if "$PROTOC" \
            --proto_path="$proto_dir" \
            --go_out="$PIKO_ROOT" \
            --go_opt=module=piko.sh/piko \
            --go-grpc_out="$PIKO_ROOT" \
            --go-grpc_opt=module=piko.sh/piko \
            "$proto_path"; then
            piko::log::success "Generated: $proto"
        else
            piko::log::error "Failed: $proto"
        fi
    done
}

# main generates Protocol Buffers code.
main() {
    piko::log::header "Generating Protocol Buffers code"

    ensure_protoc
    ensure_protoc_go_plugin
    generate_protoc

    piko::log::footer
    piko::log::success "Protocol Buffers generation complete!"
}

main "$@"
