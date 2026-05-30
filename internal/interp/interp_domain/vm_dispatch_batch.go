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

const (
	// batchedTier2Limit caps consecutive Go-fallback opcodes per ASM exit.
	//
	// processExitTier2 stops batching after this many ops in a single Go entry before
	// returning to ASM. The limit is a safety net against pathological long sequences that
	// would skip safe-point checks (cancellation, preemption) for too long.
	batchedTier2Limit = 256

	// exitAddressMapInitialCapacity is the expected number of distinct ASM-side exit stubs
	// (tier2Fallback plus the per-op direct exits). Used to pre-size the deduplication map
	// in initOpNeedsGoFallback.
	exitAddressMapInitialCapacity = 8

	// flatTableTier1Base is the start of the tier-1 region in flatJumpTable. Slots
	// [tier1Base..tier1Base+255] hold tier-1 handler addresses.
	flatTableTier1Base = 256

	// flatTableTier2Base is the start of the tier-2 region in flatJumpTable. Slots
	// [tier2Base..tier2Base+255] hold tier-2 handler addresses.
	flatTableTier2Base = 512

	// flatTableTier3Base is the start of the tier-3 region in flatJumpTable. Slots
	// [tier3Base..tier3Base+255] hold tier-3 handler addresses.
	flatTableTier3Base = 768
)

var (
	// flatGoFallback[index] is true when flatJumpTable[index] points at tier2Fallback - i.e.
	// dispatching this index will exit ASM to Go.
	//
	// The 1024-slot layout mirrors flatJumpTable:
	//
	// 	slots [0..255]    -> tier-0 handlers
	// 	slots [256..511]  -> tier-1 handlers
	// 	slots [512..767]  -> tier-2 handlers
	// 	slots [768..1023] -> tier-3 handlers
	//
	// Populated once at package init by initOpNeedsGoFallback (called at the end of the
	// arch-specific init() in vm_dispatch_{amd64,arm64}.go, after installFlatJumpTableASM
	// has finalised the source tables and copied them into flatJumpTable).
	//
	// Used by processExitTier2 to batch consecutive Go-bodied opcodes in a single Go entry,
	// avoiding one ASM<->Go round-trip per batched op.
	//
	//nolint:gochecknoglobals // dispatch lookup table, written once at init
	flatGoFallback [flatJumpTableSize]bool
)

// initOpNeedsGoFallback builds flatGoFallback from the finalised flatJumpTable.
//
// A slot is marked as a Go-fallback dispatch site when its handler address matches any of
// the asm-side exit stubs - tier2Fallback (the generic Go exit) or any per-op direct exit
// (handlerSetFieldExit, ...). This is the set of dispatch targets that hand control back
// to Go via handleDispatchExit, so processExitTier2's batching loop knows when an
// adjacent op can be run in Go too.
//
// Must run after installFlatJumpTableASM (and after every other installer that may patch
// jumptable slots with direct ASM handler addresses). The arch-specific init() in
// vm_dispatch_{amd64,arm64}.go is the canonical caller.
func initOpNeedsGoFallback() {
	exitAddresses := make(map[uintptr]struct{}, exitAddressMapInitialCapacity)
	exitAddresses[reflect.ValueOf(tier2Fallback).Pointer()] = struct{}{}
	for _, address := range directExitHandlerAddresses() {
		exitAddresses[address] = struct{}{}
	}
	for i := range flatJumpTable {
		_, fallback := exitAddresses[flatJumpTable[i]]
		flatGoFallback[i] = fallback
	}
}

// instructionWouldTrampoline reports whether ASM dispatch would exit to Go.
//
// Mirrors the DISPATCH_NEXT tier extraction in asm_dispatch_{amd64,arm64}.h: the first
// non-zero byte position picks the tier (op == tier-0, a == tier-1, b == tier-2, c ==
// tier-3), and the byte value at that position is the sub-op index within the tier.
// Inlined into processExitTier2's batch loop; branches are highly predictable per call
// site because the bytecode for a given hot-loop fragment has a stable tier mix.
//
// Takes instr (instruction) which is the 4-byte word to inspect.
//
// Returns true when dispatch of instr lands at tier2Fallback.
func instructionWouldTrampoline(instr instruction) bool {
	if instr.op != 0 {
		return flatGoFallback[uint(instr.op)]
	}
	if instr.a != 0 {
		return flatGoFallback[flatTableTier1Base+uint(instr.a)]
	}
	if instr.b != 0 {
		return flatGoFallback[flatTableTier2Base+uint(instr.b)]
	}
	if instr.c != 0 {
		return flatGoFallback[flatTableTier3Base+uint(instr.c)]
	}
	return false
}
