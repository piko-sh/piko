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

const (
	// flatJumpTableSize is the number of slots in flatJumpTable.
	//
	// The TZCNT-based dispatch macro computes a flat index in the range [0, 1023]: tier
	// (0..3) packed into bits 8..9, sub-opcode byte (0..255) packed into bits 0..7. The
	// all-zero instruction word (which would otherwise yield tier=4 -> flat_index=1024 and
	// overflow the table) is short-circuited by the macro before TZCNT runs, so 1024 slots
	// are sufficient.
	flatJumpTableSize = 1024
)

var (
	// flatJumpTable is the unified dispatch table for the TZCNT-based flat-dispatch model.
	//
	// Layout:
	//
	// 	slots [0..255]    -> tier-0 handlers (asmJumpTable[0..255])
	// 	slots [256..511]  -> tier-1 handlers (tier1JumpTable[0..255])
	// 	slots [512..767]  -> tier-2 handlers (tier2JumpTable[0..255])
	// 	slots [768..1023] -> tier-3 handlers (tier3JumpTable[0..255])
	//
	// Populated once at init by installFlatJumpTableASM (declared in
	// asm_vm_dispatch_flat_install_{amd64,arm64}.s) which uses memory loads from the four
	// source tables. Those tables hold the .abi0 entry addresses for every handler
	// (initJumpTable and initSubOpJumpTables both use LEAQ to record .abi0 entries
	// directly), so the flat table inherits the no-wrapper-frames property end-to-end.
	//
	// The dispatch loop reads its jump-table base from this global directly (LEAQ
	// *flatJumpTable(SB)), bypassing the per-call DispatchContext.jumpTable field.
	//
	//nolint:gochecknoglobals // ASM-visible dispatch table, initialised once at startup
	flatJumpTable [flatJumpTableSize]uintptr
)

// installFlatJumpTableASM copies handler addresses into flatJumpTable.
//
// Declared in asm_vm_dispatch_flat_install_{amd64,arm64}.s. Copies from asmJumpTable,
// tier1JumpTable, tier2JumpTable, and tier3JumpTable into the four 256-slot regions of
// flatJumpTable. Must be called AFTER the source tables are fully populated (i.e., after
// installTier{1,2,3}Dispatcher seed the tables and initSubOpJumpTables overwrites the
// slots with ASM-tier handlers).
//
//go:noescape
func installFlatJumpTableASM()
