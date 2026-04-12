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

const (
	// goSymbolCap is the Plan-9 ASM symbol for the Cap trampoline.
	goSymbolCap = "·asmCallCap(SB)"

	// goSymbolBytesToString is the Plan-9 ASM symbol for the BytesToString trampoline
	// (arena-backed string materialisation).
	goSymbolBytesToString = "·asmCallBytesToString(SB)"

	// goSymbolBoxSliceInt is the Plan-9 ASM symbol for the BoxSliceInt trampoline
	// (reflect.ValueOf on an int slice via vm).
	goSymbolBoxSliceInt = "·asmCallBoxSliceInt(SB)"

	// goSymbolUnboxSliceInt is the Plan-9 ASM symbol for the UnboxSliceInt trampoline
	// (type-asserting general-bank value to []int).
	goSymbolUnboxSliceInt = "·asmCallUnboxSliceInt(SB)"

	// goSymbolMakeSliceInt is the Plan-9 ASM symbol for the MakeSliceInt trampoline
	// (runtime.makeslice for []int64).
	goSymbolMakeSliceInt = "·asmCallMakeSliceInt(SB)"

	// goSymbolMakeSliceFloat is the Plan-9 ASM symbol for the MakeSliceFloat trampoline
	// (runtime.makeslice for []float64).
	goSymbolMakeSliceFloat = "·asmCallMakeSliceFloat(SB)"

	// goSymbolMakeSliceString is the Plan-9 ASM symbol for the MakeSliceString trampoline
	// (runtime.makeslice for []string).
	goSymbolMakeSliceString = "·asmCallMakeSliceString(SB)"

	// goSymbolMakeSliceBool is the Plan-9 ASM symbol for the MakeSliceBool trampoline
	// (runtime.makeslice for []bool).
	goSymbolMakeSliceBool = "·asmCallMakeSliceBool(SB)"

	// goSymbolMakeSliceUint is the Plan-9 ASM symbol for the MakeSliceUint trampoline
	// (runtime.makeslice for []uint64).
	goSymbolMakeSliceUint = "·asmCallMakeSliceUint(SB)"

	// goSymbolMakeSliceByte is the Plan-9 ASM symbol for the MakeSliceByte trampoline
	// (runtime.makeslice for []byte).
	goSymbolMakeSliceByte = "·asmCallMakeSliceByte(SB)"
)

// tier1RuntimeHandlers returns the tier-1 umbrella sub-op handler definitions.
//
// Each entry delegates to a Go runtime intrinsic requiring vm access (Cap reflect
// dispatch, BytesToString arena allocation, Box/Unbox reflect.Value conversions,
// MakeSlice runtime.makeslice). Each op is a single-level NOSPLIT|NOFRAME handler with an
// ADJSP scratch frame around the abi0 CALL. Tier-1 dispatch carries the sub-op identifier
// in operand A and uses operands B and C as the two remaining operand slots.
//
// Returns the 10 handler definitions in stable order.
func tier1RuntimeHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerSubOpCap(),
		handlerSubOpBytesToString(),
		handlerSubOpBoxSliceInt(),
		handlerSubOpUnboxSliceInt(),
		handlerSubOpMakeSliceInt(),
		handlerSubOpMakeSliceFloat(),
		handlerSubOpMakeSliceString(),
		handlerSubOpMakeSliceBool(),
		handlerSubOpMakeSliceUint(),
		handlerSubOpMakeSliceByte(),
	}
}

// 2-operand sub-ops (Cap, BytesToString, BoxSliceInt, UnboxSliceInt).

// handlerSubOpCap returns the handler definition for the Cap sub-op.
//
// Returns a 2-operand shim wrapping asmCallCap.
func handlerSubOpCap() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim("handlerSubOpCap",
		"handlerSubOpCap sets ints[B] = cap(value at C) via asmCallCap (collectionLengthOrCap).",
		goSymbolCap)
}

// handlerSubOpBytesToString returns the handler definition for the BytesToString sub-op.
//
// Returns a 2-operand shim wrapping asmCallBytesToString.
func handlerSubOpBytesToString() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim("handlerSubOpBytesToString",
		"handlerSubOpBytesToString sets strings[B] = string(bytes at C) via asmCallBytesToString (arena-backed).",
		goSymbolBytesToString)
}

// handlerSubOpBoxSliceInt returns the handler definition for the BoxSliceInt sub-op.
//
// Returns a 2-operand shim wrapping asmCallBoxSliceInt.
func handlerSubOpBoxSliceInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim("handlerSubOpBoxSliceInt",
		"handlerSubOpBoxSliceInt boxes []int64 at C into general[B] via asmCallBoxSliceInt (reflect.ValueOf).",
		goSymbolBoxSliceInt)
}

// handlerSubOpUnboxSliceInt returns the handler definition for the UnboxSliceInt sub-op.
//
// Returns a 2-operand shim wrapping asmCallUnboxSliceInt; the inner call panics when the
// source value cannot be asserted to the target slice type.
func handlerSubOpUnboxSliceInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoTwoOperandShim("handlerSubOpUnboxSliceInt",
		"handlerSubOpUnboxSliceInt unboxes general[C] into []int64 at B via asmCallUnboxSliceInt; panics on type-assert failure.",
		goSymbolUnboxSliceInt)
}

// 3-operand sub-ops (MakeSlice variants take length from C and capacity from ext.A).

// handlerSubOpMakeSliceInt returns the handler definition for the MakeSliceInt sub-op.
//
// Returns a 3-operand shim wrapping asmCallMakeSliceInt (make([]int64, length, cap) via
// vm).
func handlerSubOpMakeSliceInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpMakeSliceInt",
		"handlerSubOpMakeSliceInt builds []int64 of length C and cap ext.A into general[B] via asmCallMakeSliceInt.",
		goSymbolMakeSliceInt)
}

// handlerSubOpMakeSliceFloat returns the handler definition for the MakeSliceFloat
// sub-op.
//
// Returns a 3-operand shim wrapping asmCallMakeSliceFloat (make([]float64, length, cap)
// via vm).
func handlerSubOpMakeSliceFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpMakeSliceFloat",
		"handlerSubOpMakeSliceFloat builds []float64 of length C and cap ext.A into general[B] via asmCallMakeSliceFloat.",
		goSymbolMakeSliceFloat)
}

// handlerSubOpMakeSliceString returns the handler definition for the MakeSliceString
// sub-op.
//
// Returns a 3-operand shim wrapping asmCallMakeSliceString (make([]string, length, cap)
// via vm).
func handlerSubOpMakeSliceString() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpMakeSliceString",
		"handlerSubOpMakeSliceString builds []string of length C and cap ext.A into general[B] via asmCallMakeSliceString.",
		goSymbolMakeSliceString)
}

// handlerSubOpMakeSliceBool returns the handler definition for the MakeSliceBool sub-op.
//
// Returns a 3-operand shim wrapping asmCallMakeSliceBool (make([]bool, length, cap) via
// vm).
func handlerSubOpMakeSliceBool() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpMakeSliceBool",
		"handlerSubOpMakeSliceBool builds []bool of length C and cap ext.A into general[B] via asmCallMakeSliceBool.",
		goSymbolMakeSliceBool)
}

// handlerSubOpMakeSliceUint returns the handler definition for the MakeSliceUint sub-op.
//
// Returns a 3-operand shim wrapping asmCallMakeSliceUint (make([]uint64, length, cap) via
// vm).
func handlerSubOpMakeSliceUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpMakeSliceUint",
		"handlerSubOpMakeSliceUint builds []uint64 of length C and cap ext.A into general[B] via asmCallMakeSliceUint.",
		goSymbolMakeSliceUint)
}

// handlerSubOpMakeSliceByte returns the handler definition for the MakeSliceByte sub-op.
//
// Returns a 3-operand shim wrapping asmCallMakeSliceByte (make([]byte, length, cap) via
// vm).
func handlerSubOpMakeSliceByte() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpMakeSliceByte",
		"handlerSubOpMakeSliceByte builds []byte of length C and cap ext.A into general[B] via asmCallMakeSliceByte.",
		goSymbolMakeSliceByte)
}
