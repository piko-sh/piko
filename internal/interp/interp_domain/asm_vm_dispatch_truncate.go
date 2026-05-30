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

//go:build !safe && !(js && wasm) && (amd64 || arm64)

package interp_domain

var (
	_ = handlerTruncateNarrow
)

// handlerTruncateNarrow is the ASM-only handler for opTruncateNarrow.
//
// Body is in asm_vm_dispatch_truncate_{amd64,arm64}.s. The Go declaration is a NOSPLIT
// stub with no body; the ASM file provides the implementation.
// ProvideAsmHandlerJumpTableEntries installs this handler's address into
// asmJumpTable[opTruncateNarrow] so DISPATCH_NEXT JMPs here without leaving the ASM
// dispatch loop.
//
// The body is emitted by the EmitTruncateNarrow architecture-port primitive, which
// encodes the runtime branch on the registerKind operand (uint zero-masking vs int
// sign-extending path). See the .s file for the full body and register-allocation notes.
//
//go:noescape
func handlerTruncateNarrow()
