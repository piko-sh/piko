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
	// contextOffsetSlicesIntBase names the CTX_SLICES_INT_BASE offset of the slicesInt bank
	// base pointer in DispatchContext. The LoadTypedSliceHeaderLength primitive formats this
	// string directly into the emitted instruction operand.
	contextOffsetSlicesIntBase = "CTX_SLICES_INT_BASE"

	// contextOffsetSlicesFloatBase names the CTX_SLICES_FLOAT_BASE offset of the slicesFloat
	// bank base pointer in DispatchContext.
	contextOffsetSlicesFloatBase = "CTX_SLICES_FLOAT_BASE"

	// contextOffsetSlicesStringBase names the CTX_SLICES_STRING_BASE offset of the
	// slicesString bank base pointer in DispatchContext.
	contextOffsetSlicesStringBase = "CTX_SLICES_STRING_BASE"

	// contextOffsetSlicesBoolBase names the CTX_SLICES_BOOL_BASE offset of the slicesBool
	// bank base pointer in DispatchContext.
	contextOffsetSlicesBoolBase = "CTX_SLICES_BOOL_BASE"

	// contextOffsetSlicesUintBase names the CTX_SLICES_UINT_BASE offset of the slicesUint
	// bank base pointer in DispatchContext.
	contextOffsetSlicesUintBase = "CTX_SLICES_UINT_BASE"

	// contextOffsetSlicesByteBase names the CTX_SLICES_BYTE_BASE offset of the slicesByte
	// bank base pointer in DispatchContext.
	contextOffsetSlicesByteBase = "CTX_SLICES_BYTE_BASE"

	// elementSizeShiftStride1 is the log2 stride for byte-element typed slices.
	//
	// Zero-encodes "shift by 0 = stride 1". Used by EmitTypedSliceSliceSlice to select the
	// per-bank index scaling.
	elementSizeShiftStride1 uint8 = 0

	// elementSizeShiftStride8 is the log2 stride for 8-byte-element typed slices (int64,
	// float64, uint64). Centralised so the per-bank handler factories stay short and the
	// link between bank and stride is auditable in one place.
	elementSizeShiftStride8 uint8 = 3

	// elementSizeShiftStride16 is the log2 stride for 16-byte-element typed slices
	// (complex128).
	elementSizeShiftStride16 uint8 = 4

	// goSymbolAppendSliceIntDirect is the Plan-9 ASM symbol of the
	// asmCallAppendSliceIntDirect Go trampoline (append element to a []int64 typed-bank
	// slice).
	goSymbolAppendSliceIntDirect = "·asmCallAppendSliceIntDirect(SB)"

	// goSymbolAppendSliceFloatDirect is the Plan-9 ASM symbol of the
	// asmCallAppendSliceFloatDirect Go trampoline.
	goSymbolAppendSliceFloatDirect = "·asmCallAppendSliceFloatDirect(SB)"

	// goSymbolAppendSliceStringDirect is the Plan-9 ASM symbol of the
	// asmCallAppendSliceStringDirect Go trampoline.
	goSymbolAppendSliceStringDirect = "·asmCallAppendSliceStringDirect(SB)"

	// goSymbolAppendSliceBoolDirect is the Plan-9 ASM symbol of the
	// asmCallAppendSliceBoolDirect Go trampoline.
	goSymbolAppendSliceBoolDirect = "·asmCallAppendSliceBoolDirect(SB)"

	// goSymbolAppendSliceUintDirect is the Plan-9 ASM symbol of the
	// asmCallAppendSliceUintDirect Go trampoline.
	goSymbolAppendSliceUintDirect = "·asmCallAppendSliceUintDirect(SB)"

	// goSymbolAppendSliceByteDirect is the Plan-9 ASM symbol of the
	// asmCallAppendSliceByteDirect Go trampoline (the element is read from the uint bank and
	// truncated to a byte).
	goSymbolAppendSliceByteDirect = "·asmCallAppendSliceByteDirect(SB)"
)

// tier1SliceTypedHandlers returns the handler definitions for the tier-1 umbrella sub-ops
// that read or update typed-slice register banks (slicesInt, slicesFloat, slicesString,
// slicesBool, slicesUint) without leaving the ASM dispatch loop.
//
// Returns []asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the complete set
// of typed-slice umbrella handler definitions.
func tier1SliceTypedHandlers() []asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return []asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		handlerSubOpLenSliceIntDirect(),
		handlerSubOpLenSliceFloatDirect(),
		handlerSubOpLenSliceStringDirect(),
		handlerSubOpLenSliceBoolDirect(),
		handlerSubOpLenSliceUintDirect(),
		handlerSubOpLenSliceByteDirect(),
		handlerSubOpSliceGetFloatDirect(),
		handlerSubOpSliceSetFloatDirect(),
		handlerSubOpSliceGetUintDirect(),
		handlerSubOpSliceSetUintDirect(),
		handlerSubOpSliceGetBoolDirect(),
		handlerSubOpSliceSetBoolDirect(),
		handlerSubOpSliceGetStringDirect(),
		handlerSubOpSliceSetStringDirect(),
		handlerSubOpSliceGetByteDirect(),
		handlerSubOpSliceSetByteDirect(),
		handlerSubOpSliceByteSlice(),
		handlerSubOpRangeNextSliceByte(),
		handlerSubOpMoveSliceInt(),
		handlerSubOpMoveSliceFloat(),
		handlerSubOpMoveSliceString(),
		handlerSubOpMoveSliceBool(),
		handlerSubOpMoveSliceUint(),
		handlerSubOpMoveSliceByte(),
		handlerSubOpSliceSliceIntDirect(),
		handlerSubOpSliceSliceFloatDirect(),
		handlerSubOpSliceSliceStringDirect(),
		handlerSubOpSliceSliceBoolDirect(),
		handlerSubOpSliceSliceUintDirect(),
		handlerSubOpAppendSliceIntDirect(),
		handlerSubOpAppendSliceFloatDirect(),
		handlerSubOpAppendSliceStringDirect(),
		handlerSubOpAppendSliceBoolDirect(),
		handlerSubOpAppendSliceUintDirect(),
		handlerSubOpAppendSliceByteDirect(),
	}
}

// handlerSubOpAppendSliceIntDirect builds the tier-1 handler for
// subOpAppendSliceIntDirect: slicesInt[B] = append(slicesInt[C], ints[ext.A]). 3-operand
// shim wrapping asmCallAppendSliceIntDirect so the grow path can take Go's mallocgc; the
// trampoline pattern matches the existing make-slice family.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpAppendSliceIntDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpAppendSliceIntDirect",
		"handlerSubOpAppendSliceIntDirect - 3-operand shim wrapping ·asmCallAppendSliceIntDirect (typed-bank []int64 append).",
		goSymbolAppendSliceIntDirect)
}

// handlerSubOpAppendSliceFloatDirect builds the tier-1 handler for
// subOpAppendSliceFloatDirect: slicesFloat[B] = append(slicesFloat[C], floats[ext.A]).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpAppendSliceFloatDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpAppendSliceFloatDirect",
		"handlerSubOpAppendSliceFloatDirect - 3-operand shim wrapping ·asmCallAppendSliceFloatDirect (typed-bank []float64 append).",
		goSymbolAppendSliceFloatDirect)
}

// handlerSubOpAppendSliceStringDirect builds the tier-1 handler for
// subOpAppendSliceStringDirect: slicesString[B] = append(slicesString[C],
// strings[ext.A]). The trampoline routes the element through materialiseString so an
// arena-borrowed header gets a proper backing before being held by the destination slice.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpAppendSliceStringDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpAppendSliceStringDirect",
		"handlerSubOpAppendSliceStringDirect - 3-operand shim wrapping ·asmCallAppendSliceStringDirect (typed-bank []string append).",
		goSymbolAppendSliceStringDirect)
}

// handlerSubOpAppendSliceBoolDirect builds the tier-1 handler for
// subOpAppendSliceBoolDirect: slicesBool[B] = append(slicesBool[C], bools[ext.A]).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpAppendSliceBoolDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpAppendSliceBoolDirect",
		"handlerSubOpAppendSliceBoolDirect - 3-operand shim wrapping ·asmCallAppendSliceBoolDirect (typed-bank []bool append).",
		goSymbolAppendSliceBoolDirect)
}

// handlerSubOpAppendSliceUintDirect builds the tier-1 handler for
// subOpAppendSliceUintDirect: slicesUint[B] = append(slicesUint[C], uints[ext.A]).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpAppendSliceUintDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpAppendSliceUintDirect",
		"handlerSubOpAppendSliceUintDirect - 3-operand shim wrapping ·asmCallAppendSliceUintDirect (typed-bank []uint64 append).",
		goSymbolAppendSliceUintDirect)
}

// handlerSubOpAppendSliceByteDirect builds the tier-1 handler for
// subOpAppendSliceByteDirect: slicesByte[B] = append(slicesByte[C], byte(uints[ext.A])).
// The element is read from the uint bank and truncated to a byte, matching the convention
// slicesByte uses for its element ABI.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpAppendSliceByteDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return inlineGoThreeOperandShim("handlerSubOpAppendSliceByteDirect",
		"handlerSubOpAppendSliceByteDirect - 3-operand shim wrapping ·asmCallAppendSliceByteDirect (typed-bank []byte append).",
		goSymbolAppendSliceByteDirect)
}

// handlerSubOpMoveSliceInt builds the tier-1 handler for subOpMoveSliceInt: slicesInt[B]
// = slicesInt[C]. Pure register-to- register slice-header move; no bounds check.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpMoveSliceInt() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return moveSliceHandler("handlerSubOpMoveSliceInt", contextOffsetSlicesIntBase, "slicesInt")
}

// handlerSubOpMoveSliceFloat builds the tier-1 handler for subOpMoveSliceFloat:
// slicesFloat[B] = slicesFloat[C].
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpMoveSliceFloat() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return moveSliceHandler("handlerSubOpMoveSliceFloat", contextOffsetSlicesFloatBase, "slicesFloat")
}

// handlerSubOpMoveSliceString builds the tier-1 handler for subOpMoveSliceString:
// slicesString[B] = slicesString[C].
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpMoveSliceString() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return moveSliceHandler("handlerSubOpMoveSliceString", contextOffsetSlicesStringBase, "slicesString")
}

// handlerSubOpMoveSliceBool builds the tier-1 handler for subOpMoveSliceBool:
// slicesBool[B] = slicesBool[C].
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpMoveSliceBool() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return moveSliceHandler("handlerSubOpMoveSliceBool", contextOffsetSlicesBoolBase, "slicesBool")
}

// handlerSubOpMoveSliceUint builds the tier-1 handler for subOpMoveSliceUint:
// slicesUint[B] = slicesUint[C].
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpMoveSliceUint() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return moveSliceHandler("handlerSubOpMoveSliceUint", contextOffsetSlicesUintBase, "slicesUint")
}

// handlerSubOpMoveSliceByte builds the tier-1 handler for subOpMoveSliceByte:
// slicesByte[B] = slicesByte[C].
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpMoveSliceByte() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return moveSliceHandler("handlerSubOpMoveSliceByte", contextOffsetSlicesByteBase, "slicesByte")
}

// moveSliceHandler builds a tier-1 handler that copies a 24-byte typed-slice header
// between register slots of the same bank. The six typed-slice banks share identical slot
// layout, so a single EmitTypedSliceMove primitive parameterised on the bank base offset
// emits the correct body for all of them.
//
// Takes name (string) which is the generated symbol name.
// Takes contextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
// Takes bankLabel (string) which appears in the handler's comment to identify which bank
// the move targets (used for diagnostic context in generated assembly listings).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func moveSliceHandler(name, contextOffset, bankLabel string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   name + " copies " + bankLabel + "[C] to " + bankLabel + "[B] for the tier-1 typed-slice header move sub-op.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceMove(emitter, contextOffset)
		},
	}
}

// handlerSubOpSliceSliceIntDirect builds the tier-1 handler for subOpSliceSliceIntDirect:
// slicesInt[A] = slicesInt[C][low:high]. Element stride 8 bytes.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceSliceIntDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return sliceSliceHandler("handlerSubOpSliceSliceIntDirect", contextOffsetSlicesIntBase, "slicesInt", elementSizeShiftStride8)
}

// handlerSubOpSliceSliceFloatDirect builds the tier-1 handler for
// subOpSliceSliceFloatDirect: slicesFloat[A] = slicesFloat[C][low:high]. Element stride 8
// bytes.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceSliceFloatDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return sliceSliceHandler("handlerSubOpSliceSliceFloatDirect", contextOffsetSlicesFloatBase, "slicesFloat", elementSizeShiftStride8)
}

// handlerSubOpSliceSliceStringDirect builds the tier-1 handler for
// subOpSliceSliceStringDirect: slicesString[A] = slicesString[C][low:high]. Element
// stride 16 bytes (Go string header: data pointer + length).
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceSliceStringDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return sliceSliceHandler("handlerSubOpSliceSliceStringDirect", contextOffsetSlicesStringBase, "slicesString", elementSizeShiftStride16)
}

// handlerSubOpSliceSliceBoolDirect builds the tier-1 handler for
// subOpSliceSliceBoolDirect: slicesBool[A] = slicesBool[C][low:high]. Element stride 1
// byte.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceSliceBoolDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return sliceSliceHandler("handlerSubOpSliceSliceBoolDirect", contextOffsetSlicesBoolBase, "slicesBool", elementSizeShiftStride1)
}

// handlerSubOpSliceSliceUintDirect builds the tier-1 handler for
// subOpSliceSliceUintDirect: slicesUint[A] = slicesUint[C][low:high]. Element stride 8
// bytes.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceSliceUintDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return sliceSliceHandler("handlerSubOpSliceSliceUintDirect", contextOffsetSlicesUintBase, "slicesUint", elementSizeShiftStride8)
}

// sliceSliceHandler builds a tier-1 bounds-checked sub-slice handler for a typed-slice
// bank with a configurable element stride. Mirrors handlerSubOpSliceByteSlice but emits
// the stride-aware low-bound adjustment via EmitTypedSliceSliceSlice rather than the
// byte-fused EmitTypedSliceByteSlice primitive.
//
// Takes name (string) which is the generated symbol name.
// Takes contextOffset (string) which is the byte offset of the typed-slice bank base
// pointer within the DispatchContext.
// Takes bankLabel (string) which appears in the handler's comment.
// Takes elementSizeShift (uint8) which is the log2 of the element stride.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func sliceSliceHandler(name, contextOffset, bankLabel string, elementSizeShift uint8) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   name + " performs " + bankLabel + "[A] = " + bankLabel + "[C][low:high] with bounds check (low+high only).",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceSliceSlice(emitter, contextOffset, elementSizeShift)
		},
	}
}

// handlerSubOpSliceGetFloatDirect builds the handler for the tier-1 sub-op
// subOpSliceGetFloatDirect, which sets floats[B] = slicesFloat[C][ints[ext.A]] with
// bounds checking. On out-of-bounds access the ASM body exits to tier2Fallback so the
// Go-side handler produces the proper error message.
//
// Operand layout (Get shape; the flat dispatcher's TZCNT decode already routed here via
// flatJumpTable based on the sub-op byte): A holds the sub-op tag (already decoded by
// DISPATCH_NEXT), B is the destination float register index, C is the source slicesFloat
// register index, and the next instruction word's A field is the int register holding the
// element index.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceGetFloatDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpSliceGetFloatDirect",
		Comment:   "handlerSubOpSliceGetFloatDirect sets floats[B] = slicesFloat[C][ints[ext.A]] with bounds check.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceFloatGet(emitter, contextOffsetSlicesFloatBase)
		},
	}
}

// handlerSubOpSliceSetFloatDirect builds the handler for the tier-1 sub-op
// subOpSliceSetFloatDirect, which sets slicesFloat[B][ints[C]] = floats[ext.A] with
// bounds checking.
//
// Operand layout (Set shape): B is the destination slicesFloat register index, C is the
// int register holding the element index, and the next instruction word's A field is the
// source float register holding the value.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceSetFloatDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpSliceSetFloatDirect",
		Comment:   "handlerSubOpSliceSetFloatDirect sets slicesFloat[B][ints[C]] = floats[ext.A] with bounds check.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceFloatSet(emitter, contextOffsetSlicesFloatBase)
		},
	}
}

// handlerSubOpSliceGetUintDirect builds the handler for the tier-1 sub-op
// subOpSliceGetUintDirect, which sets uints[B] = slicesUint[C][ints[ext.A]] with bounds
// checking. Element size is 8 bytes; the destination uint bank base is loaded from
// CTX_UINTS_BASE.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceGetUintDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpSliceGetUintDirect",
		Comment:   "handlerSubOpSliceGetUintDirect sets uints[B] = slicesUint[C][ints[ext.A]] with bounds check.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceUintGet(emitter, contextOffsetSlicesUintBase)
		},
	}
}

// handlerSubOpSliceSetUintDirect builds the handler for the tier-1 sub-op
// subOpSliceSetUintDirect, which sets slicesUint[B][ints[C]] = uints[ext.A] with bounds
// checking.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceSetUintDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpSliceSetUintDirect",
		Comment:   "handlerSubOpSliceSetUintDirect sets slicesUint[B][ints[C]] = uints[ext.A] with bounds check.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceUintSet(emitter, contextOffsetSlicesUintBase)
		},
	}
}

// handlerSubOpSliceGetBoolDirect builds the handler for the tier-1 sub-op
// subOpSliceGetBoolDirect, which sets bools[B] = slicesBool[C][ints[ext.A]] with bounds
// checking. Element size is 1 byte; the destination bool bank base is loaded from
// CTX_BOOLS_BASE.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceGetBoolDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpSliceGetBoolDirect",
		Comment:   "handlerSubOpSliceGetBoolDirect sets bools[B] = slicesBool[C][ints[ext.A]] with bounds check.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceBoolGet(emitter, contextOffsetSlicesBoolBase)
		},
	}
}

// handlerSubOpSliceSetBoolDirect builds the handler for the tier-1 sub-op
// subOpSliceSetBoolDirect, which sets slicesBool[B][ints[C]] = bools[ext.A] with bounds
// checking.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceSetBoolDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpSliceSetBoolDirect",
		Comment:   "handlerSubOpSliceSetBoolDirect sets slicesBool[B][ints[C]] = bools[ext.A] with bounds check.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceBoolSet(emitter, contextOffsetSlicesBoolBase)
		},
	}
}

// handlerSubOpSliceGetStringDirect builds the handler for the tier-1 sub-op
// subOpSliceGetStringDirect, which sets strings[B] = slicesString[C][ints[ext.A]] with
// bounds checking. Element size is 16 bytes (Go string header: data pointer + length);
// the destination string bank base is loaded from CTX_STRINGS_BASE.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceGetStringDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpSliceGetStringDirect",
		Comment:   "handlerSubOpSliceGetStringDirect sets strings[B] = slicesString[C][ints[ext.A]] with bounds check.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceStringGet(emitter, contextOffsetSlicesStringBase)
		},
	}
}

// handlerSubOpSliceSetStringDirect builds the handler for the tier-1 sub-op
// subOpSliceSetStringDirect, which sets slicesString[B][ints[C]] = strings[ext.A] with
// bounds checking.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceSetStringDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpSliceSetStringDirect",
		Comment:   "handlerSubOpSliceSetStringDirect sets slicesString[B][ints[C]] = strings[ext.A] with bounds check.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceStringSet(emitter, contextOffsetSlicesStringBase)
		},
	}
}

// handlerSubOpLenSliceIntDirect builds the handler for the tier-1 sub-op
// subOpLenSliceIntDirect, which sets ints[B] = int64(len(slicesInt[C])) without crossing
// the ASM/Go boundary.
//
// Operand layout (the flat dispatcher's TZCNT decode already routed here via
// flatJumpTable based on the sub-op byte): A holds the sub-op tag
// (subOpLenSliceIntDirect, already decoded by DISPATCH_NEXT), B is the destination int
// register index, and C is the source slicesInt register index.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] which is the handler
// definition for subOpLenSliceIntDirect.
func handlerSubOpLenSliceIntDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return typedSliceLenHandler(
		"handlerSubOpLenSliceIntDirect",
		"handlerSubOpLenSliceIntDirect sets ints[B] = int64(len(slicesInt[C])).",
		contextOffsetSlicesIntBase,
	)
}

// handlerSubOpLenSliceFloatDirect builds the handler for the tier-1 sub-op
// subOpLenSliceFloatDirect, which sets ints[B] = int64(len(slicesFloat[C])) without
// crossing the ASM/Go boundary.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpLenSliceFloatDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return typedSliceLenHandler(
		"handlerSubOpLenSliceFloatDirect",
		"handlerSubOpLenSliceFloatDirect sets ints[B] = int64(len(slicesFloat[C])).",
		contextOffsetSlicesFloatBase,
	)
}

// handlerSubOpLenSliceStringDirect builds the handler for the tier-1 sub-op
// subOpLenSliceStringDirect, which sets ints[B] = int64(len(slicesString[C])) without
// crossing the ASM/Go boundary.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpLenSliceStringDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return typedSliceLenHandler(
		"handlerSubOpLenSliceStringDirect",
		"handlerSubOpLenSliceStringDirect sets ints[B] = int64(len(slicesString[C])).",
		contextOffsetSlicesStringBase,
	)
}

// handlerSubOpLenSliceBoolDirect builds the handler for the tier-1 sub-op
// subOpLenSliceBoolDirect, which sets ints[B] = int64(len(slicesBool[C])) without
// crossing the ASM/Go boundary.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpLenSliceBoolDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return typedSliceLenHandler(
		"handlerSubOpLenSliceBoolDirect",
		"handlerSubOpLenSliceBoolDirect sets ints[B] = int64(len(slicesBool[C])).",
		contextOffsetSlicesBoolBase,
	)
}

// handlerSubOpLenSliceUintDirect builds the handler for the tier-1 sub-op
// subOpLenSliceUintDirect, which sets ints[B] = int64(len(slicesUint[C])) without
// crossing the ASM/Go boundary.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpLenSliceUintDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return typedSliceLenHandler(
		"handlerSubOpLenSliceUintDirect",
		"handlerSubOpLenSliceUintDirect sets ints[B] = int64(len(slicesUint[C])).",
		contextOffsetSlicesUintBase,
	)
}

// typedSliceLenHandler builds a handler definition for a tier-1 "len of typed slice"
// sub-op. Each typed-slice bank (int/float/string/bool/uint) shares the same handler
// shape (extract B and C operands, read length from the slice header at the bank-specific
// context offset, write to the integer bank); only the context offset differs.
//
// Takes name (string) which is the asmgen-generated symbol name (e.g.,
// "handlerSubOpLenSliceIntDirect").
// Takes comment (string) which is the inline comment emitted into the .s file for the
// handler's TEXT block.
// Takes contextOffset (string) which is the byte offset of the bank base pointer within
// the DispatchContext.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort] ready to register in a
// FileGroup.
func typedSliceLenHandler(name, comment, contextOffset string) asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      name,
		Comment:   comment,
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			scratches := architecture.ScratchRegisters()
			architecture.ExtractB(emitter, scratches[0])
			architecture.ExtractC(emitter, scratches[1])
			temp := architecture.DataTemporary(dataTempScratch0)
			architecture.LoadTypedSliceHeaderLength(emitter, contextOffset, scratches[1], temp)
			architecture.StoreToBank(emitter, asmgen.RegisterBankInteger, temp, scratches[0])
			architecture.DispatchNext(emitter)
		},
	}
}

// handlerSubOpLenSliceByteDirect builds the handler for the tier-1 sub-op
// subOpLenSliceByteDirect, which sets ints[B] = int64(len(slicesByte[C])) without
// crossing the ASM/Go boundary.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpLenSliceByteDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return typedSliceLenHandler(
		"handlerSubOpLenSliceByteDirect",
		"handlerSubOpLenSliceByteDirect sets ints[B] = int64(len(slicesByte[C])).",
		contextOffsetSlicesByteBase,
	)
}

// handlerSubOpSliceGetByteDirect builds the handler for the tier-1 sub-op
// subOpSliceGetByteDirect, which sets uints[B] = uint64(slicesByte[C][ints[ext.A]]) with
// bounds checking. Element size is 1 byte; the load uses MOVBLZX (amd64) / MOVBU (arm64)
// to zero-extend, and the destination uint bank slot is 8 bytes wide.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceGetByteDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpSliceGetByteDirect",
		Comment:   "handlerSubOpSliceGetByteDirect sets uints[B] = uint64(slicesByte[C][ints[ext.A]]) with bounds check.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceByteGet(emitter, contextOffsetSlicesByteBase)
		},
	}
}

// handlerSubOpSliceSetByteDirect builds the handler for the tier-1 sub-op
// subOpSliceSetByteDirect, which sets slicesByte[B][ints[C]] = byte(uints[ext.A]) with
// bounds checking.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceSetByteDirect() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpSliceSetByteDirect",
		Comment:   "handlerSubOpSliceSetByteDirect sets slicesByte[B][ints[C]] = byte(uints[ext.A]) with bounds check.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceByteSet(emitter, contextOffsetSlicesByteBase)
		},
	}
}

// handlerSubOpSliceByteSlice builds the handler for the tier-1 sub-op subOpSliceByteSlice
// (Phase F.3): slicesByte[A] = slicesByte[C][ints[ext.b]:ints[ext.c]]. Pure ASM body for
// the low+high case (sliceLowBoundFlag | sliceHighBoundFlag); other flag shapes
// (max-bound, low-only, high-only, no-flag) defer to the Go fallback via tier2Fallback.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpSliceByteSlice() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpSliceByteSlice",
		Comment:   "handlerSubOpSliceByteSlice sets slicesByte[A] = slicesByte[C][ints[ext.b]:ints[ext.c]] (no max-bound) inline.",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedSliceByteSlice(emitter, contextOffsetSlicesByteBase)
		},
	}
}

// handlerSubOpRangeNextSliceByte builds the handler for the tier-1 sub-op
// subOpRangeNextSliceByte (Phase F.6): typed range-next over slicesByte. Pure ASM; on
// end-of-range applies the 24-bit forward jump offset packed in the next ext word.
//
// Returns asmgen.HandlerDefinition[BytecodeArchitecturePort].
func handlerSubOpRangeNextSliceByte() asmgen.HandlerDefinition[BytecodeArchitecturePort] {
	return asmgen.HandlerDefinition[BytecodeArchitecturePort]{
		Name:      "handlerSubOpRangeNextSliceByte",
		Comment:   "handlerSubOpRangeNextSliceByte advances ints[A]; on end-of-range jumps by ext word; else uints[C] = byte at slicesByte[B][ints[A]].",
		FrameSize: frameSizeZero,
		Flags:     flagsNoSplitNoFrame,
		Emit: func(emitter *asmgen.Emitter, architecture BytecodeArchitecturePort) {
			architecture.EmitTypedRangeNextByte(emitter, contextOffsetSlicesByteBase)
		},
	}
}
