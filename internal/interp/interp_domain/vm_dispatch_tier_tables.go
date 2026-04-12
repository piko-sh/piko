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

// Per-tier source tables for the TZCNT-based flat dispatch model.
//
// Each tier table holds the .abi0 entry address of the handler for every sub-opcode in
// that tier. Slots that don't have an ASM body hold tier2Fallback so the instruction
// exits to Go via the flat dispatcher and is handled by flatDispatchSwitch.
//
// Population flow at startup:
//
//  1. installTier{1,2,3}Dispatcher seed every slot with
//     reflect.ValueOf(tier2Fallback).Pointer().
//  2. The asmgen-emitted initSubOpJumpTables overwrites slots that have an ASM body with
//     their LEAQ .abi0 entry.
//  3. installFlatJumpTableASM copies the three tier tables (and asmJumpTable) into
//     flatJumpTable[0..1023]. From that point on the dispatch loop reads only
//     flatJumpTable; the tier tables exist solely as the canonical source of truth.
//
// All three tables are 256 entries because each tier's sub-op byte is read as a uint8
// from the instruction word.

const (
	// tier1OpcodeTableSize sets the upper bound on tier-1 sub-opcode indices. The flat
	// dispatcher decodes operand A as a uint8, so 256 entries cover the whole tier-1
	// sub-opcode space.
	tier1OpcodeTableSize = 256

	// tier2OpcodeTableSize sets the upper bound on tier-2 sub-opcode indices. Operand B is
	// read as a uint8.
	tier2OpcodeTableSize = 256

	// tier3OpcodeTableSize sets the upper bound on tier-3 sub-opcode indices. Operand C is
	// read as a uint8.
	tier3OpcodeTableSize = 256
)

var (
	// tier1JumpTable holds tier-1 sub-opcode handler addresses. Copied into
	// flatJumpTable[256..511] by installFlatJumpTableASM.
	//
	//nolint:gochecknoglobals // ASM-visible dispatch table seeded once at startup
	tier1JumpTable [tier1OpcodeTableSize]uintptr

	// tier2JumpTable holds tier-2 sub-opcode handler addresses. Copied into
	// flatJumpTable[512..767] by installFlatJumpTableASM.
	//
	//nolint:gochecknoglobals // ASM-visible dispatch table seeded once at startup
	tier2JumpTable [tier2OpcodeTableSize]uintptr

	// tier3JumpTable holds tier-3 sub-opcode handler addresses. Copied into
	// flatJumpTable[768..1023] by installFlatJumpTableASM.
	//
	//nolint:gochecknoglobals // ASM-visible dispatch table seeded once at startup
	tier3JumpTable [tier3OpcodeTableSize]uintptr
)

// installTier1Dispatcher seeds tier1JumpTable with the fallback.
//
// Every slot gets the tier2Fallback address. The asmgen-emitted initSubOpJumpTables later
// overwrites individual slots with the .abi0 entries of tier-1 sub-op ASM bodies;
// remaining slots stay at tier2Fallback so a tier-1 instruction without an ASM body exits
// to Go via the flat dispatcher and is handled by flatDispatchSwitch.
//
// Called once at startup after initJumpTable. The patching is idempotent; repeated calls
// leave the table in the same state.
func installTier1Dispatcher() {
	fallbackAddress := reflect.ValueOf(tier2Fallback).Pointer()

	for i := range tier1JumpTable {
		tier1JumpTable[i] = fallbackAddress
	}
}

// installTier2Dispatcher seeds tier2JumpTable with the default tier2Fallback address in
// every slot. Mirror of installTier1Dispatcher for the tier-2 sub-opcode space.
//
// Called once at startup. The patching is idempotent; repeated calls leave the table in
// the same state.
func installTier2Dispatcher() {
	fallbackAddress := reflect.ValueOf(tier2Fallback).Pointer()

	for i := range tier2JumpTable {
		tier2JumpTable[i] = fallbackAddress
	}
}

// installTier3Dispatcher seeds tier3JumpTable with the default tier2Fallback address in
// every slot. Mirror of installTier1Dispatcher for the tier-3 sub-opcode space.
//
// Called once at startup after installTier2Dispatcher. The patching is idempotent;
// repeated calls leave the table in the same state.
func installTier3Dispatcher() {
	fallbackAddress := reflect.ValueOf(tier2Fallback).Pointer()

	for i := range tier3JumpTable {
		tier3JumpTable[i] = fallbackAddress
	}
}
