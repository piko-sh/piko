// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package interp_domain

// Hook for the asmgen tool. Re-emits the Plan 9 dispatch assembly and
// asm_dispatch_offsets.h header file from the live struct layouts in this package, so
// reordering varLocation, callFrame, DispatchContext, or asmCallInfo for alignment is
// safe - the new offsets propagate automatically into every .s file that includes the
// header.
//
// Run `go generate ./internal/interp/interp_domain/...` after any field-layout change to
// the structs whose offsets the asmgen providers expose (see
// ProvideDispatchContextOffsets, ProvideCallFrameOffsets, ProvideASMCallInfoOffsets,
// ProvideVarLocationOffsets). The tool's `-validate` mode runs as part of the test suite
// and fails fast on drift.

//go:generate go run piko.sh/piko/cmd/asmgen
