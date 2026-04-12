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

import (
	"reflect"
)

// Per-op direct exits.
//
// High-volume Go-bodied opcodes (opSetField, opGetField, opMapIndex, opAppend,
// opAppendByteFast) route through dedicated ASM exit stubs instead of the generic
// tier2Fallback. Each stub writes a distinct exit reason; handleDispatchExit then
// dispatches directly to a dedicated processExitXxx handler that calls the specific Go
// body without the handlerTable[op] indirect. The saving is ~3-5 cycles per trampoline.

// handlerSetFieldExit is the ASM exit stub for opSetField.
//
//go:noescape
func handlerSetFieldExit() //nolint:unused // ASM handler - JMP target via asmJumpTable[opSetField]

// handlerGetFieldExit is the ASM exit stub for opGetField.
//
//go:noescape
func handlerGetFieldExit() //nolint:unused // ASM handler - opGetField

// handlerMapIndexExit is the ASM exit stub for opMapIndex.
//
//go:noescape
func handlerMapIndexExit() //nolint:unused // ASM handler - opMapIndex

// handlerAppendExit is the ASM exit stub for opAppend.
//
//go:noescape
func handlerAppendExit() //nolint:unused // ASM handler - opAppend

// handlerAppendByteFastExit is the ASM exit stub for opAppendByteFast.
//
//go:noescape
func handlerAppendByteFastExit() //nolint:unused // ASM handler - opAppendByteFast

// installPerOpDirectExits patches asmJumpTable to route specific Go-fallback opcodes to
// dedicated exit stubs. Must run after initJumpTable (which fills every slot with
// tier2Fallback) and before installFlatJumpTableASM (which snapshots asmJumpTable into
// the runtime-hot flatJumpTable).
//
// Tier-0 struct-field primitive readers/writers, general-bank struct-field
// readers/writers, and the testNil-jump pair live on the tier-2 ASM-call shim path
// (tier2_shim_registry.go) and are installed from there; installPerOpDirectExits does not
// patch them.
func installPerOpDirectExits() {
	asmJumpTable[opSetField] = reflect.ValueOf(handlerSetFieldExit).Pointer()
	asmJumpTable[opGetField] = reflect.ValueOf(handlerGetFieldExit).Pointer()
	asmJumpTable[opMapIndex] = reflect.ValueOf(handlerMapIndexExit).Pointer()
	asmJumpTable[opAppend] = reflect.ValueOf(handlerAppendExit).Pointer()
	asmJumpTable[opAppendByteFast] = reflect.ValueOf(handlerAppendByteFastExit).Pointer()
}

// directExitHandlerAddresses returns the set of ASM-side exit-stub addresses installed by
// installPerOpDirectExits. initOpNeedsGoFallback consults this list (alongside
// tier2Fallback) so flatGoFallback correctly marks per-op direct-exit slots as
// Go-fallback sites for batching.
//
// Slots backed by tier-2 ASM-call shims (struct-field T0 readers/writers, general-bank
// struct-field T0, testNil-jump pair) must not appear here: they stay in ASM via
// DISPATCH_NEXT, and listing them would force spurious tier-2 batching.
//
// Returns a []uintptr of the per-op direct-exit handler addresses.
func directExitHandlerAddresses() []uintptr {
	return []uintptr{
		reflect.ValueOf(handlerSetFieldExit).Pointer(),
		reflect.ValueOf(handlerGetFieldExit).Pointer(),
		reflect.ValueOf(handlerMapIndexExit).Pointer(),
		reflect.ValueOf(handlerAppendExit).Pointer(),
		reflect.ValueOf(handlerAppendByteFastExit).Pointer(),
	}
}
