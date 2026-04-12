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

package asm

import (
	"piko.sh/piko/wdk/asmgen"
)

// Tier-1 inc/dec struct-field ASM lifts.
//
// Single-level NOSPLIT|NOFRAME handlers installed in tier1JumpTable. ADJSP $32 / $-32
// frames the abi0 CALL to the Go trampoline asmCall{Inc,Dec}StructField{Int,Uint}, which
// returns ctx. After the CALL the shim reloads dispatch registers and DISPATCH_NEXTs
// without ever leaving the ASM dispatch loop.

const (
	// goSymbolIncStructFieldInt names the Go trampoline target the inc-int-struct-field
	// handler CALLs.
	goSymbolIncStructFieldInt = "·asmCallIncStructFieldInt(SB)"

	// goSymbolDecStructFieldInt names the Go trampoline target the dec-int-struct-field
	// handler CALLs.
	goSymbolDecStructFieldInt = "·asmCallDecStructFieldInt(SB)"

	// goSymbolIncStructFieldUint names the Go trampoline target the inc-uint-struct-field
	// handler CALLs.
	goSymbolIncStructFieldUint = "·asmCallIncStructFieldUint(SB)"

	// goSymbolDecStructFieldUint names the Go trampoline target the dec-uint-struct-field
	// handler CALLs.
	goSymbolDecStructFieldUint = "·asmCallDecStructFieldUint(SB)"
)

// tier1StructFieldIncDecHandlers returns the four single-level handler definitions for
// the inc/dec struct field super-instructions.
//
// Returns the inc/dec int and uint struct-field handler definitions.
func tier1StructFieldIncDecHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerSubOpIncStructFieldInt(),
		handlerSubOpDecStructFieldInt(),
		handlerSubOpIncStructFieldUint(),
		handlerSubOpDecStructFieldUint(),
	}
}

// handlerSubOpIncStructFieldInt builds the tier-1 inc-int-struct-field handler.
//
// Returns the handler definition that calls asmCallIncStructFieldInt.
func handlerSubOpIncStructFieldInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim(
		"handlerSubOpIncStructFieldInt",
		"handlerSubOpIncStructFieldInt increments the int field on the struct at ints[B] by one via asmCallIncStructFieldInt.",
		goSymbolIncStructFieldInt,
	)
}

// handlerSubOpDecStructFieldInt builds the tier-1 dec-int-struct-field handler.
//
// Returns the handler definition that calls asmCallDecStructFieldInt.
func handlerSubOpDecStructFieldInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim(
		"handlerSubOpDecStructFieldInt",
		"handlerSubOpDecStructFieldInt decrements the int field on the struct at ints[B] by one via asmCallDecStructFieldInt.",
		goSymbolDecStructFieldInt,
	)
}

// handlerSubOpIncStructFieldUint builds the tier-1 inc-uint-struct-field handler.
//
// Returns the handler definition that calls asmCallIncStructFieldUint.
func handlerSubOpIncStructFieldUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim(
		"handlerSubOpIncStructFieldUint",
		"handlerSubOpIncStructFieldUint increments the uint field on the struct at ints[B] by one via asmCallIncStructFieldUint.",
		goSymbolIncStructFieldUint,
	)
}

// handlerSubOpDecStructFieldUint builds the tier-1 dec-uint-struct-field handler.
//
// Returns the handler definition that calls asmCallDecStructFieldUint.
func handlerSubOpDecStructFieldUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim(
		"handlerSubOpDecStructFieldUint",
		"handlerSubOpDecStructFieldUint single-level shim wrapping asmCallDecStructFieldUint.",
		goSymbolDecStructFieldUint,
	)
}
