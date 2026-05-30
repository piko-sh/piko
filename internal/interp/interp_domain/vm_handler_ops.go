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

import (
	"fmt"
	"reflect"
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// registerBitWidth is the underlying width of the int and uint register banks. Used as
	// the high anchor when sign-extending narrow integers in opTruncateNarrow: the value is
	// shifted up by (registerBitWidth - declaredWidth), then arithmetic-shifted back down so
	// the sign bit of the declared width propagates correctly.
	registerBitWidth uint = 64

	// smallIntBoxLow is the inclusive lower bound of the pre-allocated reflect.Value box
	// cache for int64 values; mirrors CPython's small-int cache extended down to -128 to
	// cover common sentinels.
	smallIntBoxLow = -128

	// smallIntBoxHigh is the inclusive upper bound of the pre-allocated reflect.Value box
	// cache; 256 keeps the full byte range and small loop counters in cache.
	smallIntBoxHigh = 256

	// smallIntBoxSize is the entry count for smallIntBoxCache (inclusive of both bounds).
	smallIntBoxSize = smallIntBoxHigh - smallIntBoxLow + 1

	// smallUintBoxCacheSize covers byte-range uints (0..255), which is the common output of
	// string and []byte indexing. Index == value.
	smallUintBoxCacheSize = 256

	// extensionWideIndexHighByteShift is the left shift placing the extension word's B byte
	// into the high half of a uint16 wide index.
	extensionWideIndexHighByteShift = 8

	// integerDivideByZeroMessage is the panic message raised when an integer division or
	// remainder operation has a zero divisor. It mirrors the Go runtime's own wording so
	// interpreted programs observe identical panic text.
	integerDivideByZeroMessage = "runtime error: integer divide by zero"

	// moveGeneralModeDynamic dispatches to valueCopyForBoundaryArena's runtime kind switch.
	//
	// Used when the source's static type is interface, type parameter, or unavailable.
	// Encoded as zero so existing bytecode (and any compiler emission paths that have not
	// been threaded with a static type yet) preserves current behaviour byte-for-byte.
	moveGeneralModeDynamic uint8 = 0

	// moveGeneralModeAlias performs a direct reflect.Value header copy. Emitted when the
	// source's static type is alias-safe.
	moveGeneralModeAlias uint8 = 1

	// moveGeneralModeSnapshot unconditionally invokes the snapshot helper. Emitted when the
	// source's static type is struct or array.
	moveGeneralModeSnapshot uint8 = 2
)

var (
	// smallIntBoxCache holds pre-allocated reflect.Value boxes for int64 values in
	// [smallIntBoxLow, smallIntBoxHigh]. Index = v - smallIntBoxLow.
	smallIntBoxCache [smallIntBoxSize]reflect.Value

	// smallUintBoxCache holds pre-allocated reflect.Value boxes for uint64 values in [0,
	// smallUintBoxCacheSize-1].
	smallUintBoxCache [smallUintBoxCacheSize]reflect.Value

	// boolBoxCache holds the two pre-allocated reflect.Value boxes for bool values. Index 0
	// is false, index 1 is true.
	boolBoxCache [2]reflect.Value

	// emptyStringBox is the pre-allocated reflect.Value for the empty string, which appears
	// frequently as a zero-value initialiser and loop sentinel.
	emptyStringBox = reflect.ValueOf("")

	// stringTypeABIType caches the *abi.Type for `string` so the arena- backed box
	// constructor can skip the reflect.Type to *abi.Type extract on the hot path. Populated
	// in init() once per process.
	stringTypeABIType = reflectValueABIType(reflect.TypeFor[string]())

	// int64TypeABIType is the cached *abi.Type pointer for int64, used by the arena-backed
	// boxInt64ToGeneral fast path.
	int64TypeABIType = reflectValueABIType(reflect.TypeFor[int64]())

	// float64TypeABIType is the cached *abi.Type pointer for float64, used by the
	// arena-backed boxFloat64ToGeneral fast path.
	float64TypeABIType = reflectValueABIType(reflect.TypeFor[float64]())

	// uint64TypeABIType is the cached *abi.Type pointer for uint64, used by the arena-backed
	// boxUint64ToGeneral fast path.
	uint64TypeABIType = reflectValueABIType(reflect.TypeFor[uint64]())

	// complex128TypeABIType is the cached *abi.Type pointer for complex128, used by the
	// arena-backed boxComplex128ToGeneral fast path to replace the per-call reflect.ValueOf
	// mallocgc.
	complex128TypeABIType = reflectValueABIType(reflect.TypeFor[complex128]())
)

func init() {
	for i := range smallIntBoxCache {
		smallIntBoxCache[i] = reflect.ValueOf(int64(i + smallIntBoxLow))
	}
}

// boxInt64ToGeneral returns a reflect.Value wrapping v.
//
// For values in [smallIntBoxLow, smallIntBoxHigh] uses the static cache. For others, when
// arena is supplied, bump-allocates from arena.intBoxSlab; when arena is nil, falls back
// to reflect.ValueOf (allocates).
//
// Takes arena (*RegisterArena) which is the per-VM bump arena; may be nil.
// Takes v (int64) which is the value to box.
//
// Returns a reflect.Value of dynamic type int64 wrapping v.
func boxInt64ToGeneral(arena *RegisterArena, v int64) reflect.Value {
	if v >= smallIntBoxLow && v <= smallIntBoxHigh {
		return smallIntBoxCache[v-smallIntBoxLow]
	}
	if arena != nil {
		slot := arena.AllocIntBox(v)
		return unsafeNewAt(int64TypeABIType, unsafe.Pointer(slot), reflect.Int64)
	}
	return reflect.ValueOf(v)
}

func init() {
	for i := range smallUintBoxCache {
		smallUintBoxCache[i] = reflect.ValueOf(uint64(i))
	}
	boolBoxCache[0] = reflect.ValueOf(false)
	boolBoxCache[1] = reflect.ValueOf(true)
}

// boxUint64ToGeneral returns a reflect.Value wrapping v.
//
// For values in [0, smallUintBoxCacheSize) uses the static cache. For larger values, when
// arena is supplied, bump-allocates from arena.uintBoxSlab; when arena is nil, falls back
// to reflect.ValueOf.
//
// Takes arena (*RegisterArena) which is the per-VM bump arena; may be nil.
// Takes v (uint64) which is the value to box.
//
// Returns a reflect.Value of dynamic type uint64 wrapping v.
func boxUint64ToGeneral(arena *RegisterArena, v uint64) reflect.Value {
	if v < smallUintBoxCacheSize {
		return smallUintBoxCache[v]
	}
	if arena != nil {
		slot := arena.AllocUintBox(v)
		return unsafeNewAt(uint64TypeABIType, unsafe.Pointer(slot), reflect.Uint64)
	}
	return reflect.ValueOf(v)
}

// boxFloat64ToGeneral returns a reflect.Value wrapping v.
//
// No static cache (floats don't have a clean small-value enumeration); arena-allocates
// when supplied, else reflect.ValueOf.
//
// Takes arena (*RegisterArena) which is the per-VM bump arena; may be nil.
// Takes v (float64) which is the value to box.
//
// Returns a reflect.Value of dynamic type float64 wrapping v.
func boxFloat64ToGeneral(arena *RegisterArena, v float64) reflect.Value {
	if arena != nil {
		slot := arena.AllocFloatBox(v)
		return unsafeNewAt(float64TypeABIType, unsafe.Pointer(slot), reflect.Float64)
	}
	return reflect.ValueOf(v)
}

// boxComplex128ToGeneral returns a reflect.Value wrapping v.
//
// Sibling of boxInt64ToGeneral / boxFloat64ToGeneral. No static cache (complex128 doesn't
// have a clean small-value enumeration). When an arena is supplied, bump-allocates from
// arena.complexBoxSlab; when arena is nil, falls back to reflect.ValueOf.
//
// Takes arena (*RegisterArena) which is the per-VM bump arena; may be nil.
// Takes v (complex128) which is the value to box.
//
// Returns a reflect.Value of dynamic type complex128 wrapping v.
func boxComplex128ToGeneral(arena *RegisterArena, v complex128) reflect.Value {
	if arena != nil {
		slot := arena.AllocComplexBox(v)
		return unsafeNewAt(complex128TypeABIType, unsafe.Pointer(slot), reflect.Complex128)
	}
	return reflect.ValueOf(v)
}

// boxBoolToGeneral returns a pre-cached reflect.Value wrapping v.
//
// Cost is one branch and one slice load - no allocation.
//
// Takes v (bool) which is the value to box.
//
// Returns a reflect.Value of dynamic type bool wrapping v.
func boxBoolToGeneral(v bool) reflect.Value {
	if v {
		return boolBoxCache[1]
	}
	return boolBoxCache[0]
}

// boxStringToGeneral returns a reflect.Value wrapping s.
//
// Empty strings return the pre-allocated singleton; non-empty strings route through the
// arena's stringBoxSlab when an arena is supplied, replacing reflect.ValueOf(s)'s
// per-call mallocgc with a bump-allocated slot. The arena slab clears on arena.Reset so
// the slots are reclaimed in bulk between Execute() boundaries. When arena is nil
// (callers without arena context), falls back to reflect.ValueOf - preserves the
// pre-existing behaviour.
//
// Takes arena (*RegisterArena) which is the per-VM bump arena; may be nil.
// Takes s (string) which is the value to box.
//
// Returns a reflect.Value of dynamic type string wrapping s.
func boxStringToGeneral(arena *RegisterArena, s string) reflect.Value {
	if len(s) == 0 {
		return emptyStringBox
	}
	if arena != nil {
		slot := arena.AllocStringBox(s)

		return unsafeNewAt(stringTypeABIType, unsafe.Pointer(slot), reflect.String)
	}
	return reflect.ValueOf(s)
}

// handleExt is the handler for an extension opcode slot; it is a no-op that continues
// dispatch.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleExt(_ *VM, _ *callFrame, _ *Registers, _ instruction) opResult { return opContinue }

// conditionalJump reads the extension word from the bytecode stream and advances the
// program counter by the encoded offset when shouldJump is true. This is a small helper
// shared by all const-compare-and-branch handlers to eliminate duplicated jump logic.
//
// Takes frame (*callFrame) which provides the bytecode body and program counter.
// Takes shouldJump (bool) which indicates whether the branch should be taken.
//
// Returns opResult which signals the VM dispatch loop to continue.
func conditionalJump(frame *callFrame, shouldJump bool) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	if shouldJump {
		offset := joinOffset(extensionWord.a, extensionWord.b)
		frame.programCounter += int(offset)
	}
	return opContinue
}

// intConstBoundsCheck validates that instruction.b is within the integer constant pool
// and returns the constant value. When the index is out of bounds it triggers a VM bounds
// error and returns ok=false.
//
// Takes vm (*VM) which provides bounds-error reporting.
// Takes frame (*callFrame) which provides the integer constant pool.
// Takes instruction (instruction) which encodes the constant pool index in field b.
//
// Returns constantValue (int64) which is the constant value when ok is true.
// Returns errResult (opResult) which is the error result when ok is false.
// Returns ok (bool) which indicates whether the bounds check passed.
func intConstBoundsCheck(vm *VM, frame *callFrame, instruction instruction) (int64, opResult, bool) {
	if int(instruction.b) >= len(frame.function.intConstants) {
		vmBoundsError(vm, frame, boundsTableIntConstant, int(instruction.b), len(frame.function.intConstants))
		return 0, opPanicError, false
	}
	return frame.function.intConstants[instruction.b], opContinue, true
}

// stringConstBoundsCheck validates that instruction.b is within the string constant pool
// and returns the constant value. When the index is out of bounds it triggers a VM bounds
// error and returns ok=false.
//
// Takes vm (*VM) which provides bounds-error reporting.
// Takes frame (*callFrame) which provides the string constant pool.
// Takes instruction (instruction) which encodes the constant pool index in field b.
//
// Returns constantValue (string) which is the constant value when ok is true.
// Returns errResult (opResult) which is the error result when ok is false.
// Returns ok (bool) which indicates whether the bounds check passed.
func stringConstBoundsCheck(vm *VM, frame *callFrame, instruction instruction) (string, opResult, bool) {
	if int(instruction.b) >= len(frame.function.stringConstants) {
		vmBoundsError(vm, frame, boundsTableStringConstant, int(instruction.b), len(frame.function.stringConstants))
		return "", opPanicError, false
	}
	return frame.function.stringConstants[instruction.b], opContinue, true
}

// handleMoveInt copies a signed integer value between virtual machine registers.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes source and destination register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b]
	return opContinue
}

// handleMoveFloat copies a floating-point value between virtual machine registers.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes source and destination register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = registers.floats[instruction.b]
	return opContinue
}

// handleMoveString copies a string value between virtual machine registers.
//
// Takes registers (*Registers) which provides the string register banks.
// Takes instruction (instruction) which encodes source and destination register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = registers.strings[instruction.b]
	return opContinue
}

// handleMoveGeneral copies a general-purpose reflect.Value between registers.
//
// The instruction's C operand encodes a snapshot mode chosen by the compiler from the
// source operand's static type (see moveGeneralMode constants). Mode zero (dynamic)
// preserves the pre-existing valueCopyForBoundary path so existing serialised bytecode
// continues to behave identically.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes source and destination register indices
// and the snapshot mode in operand C.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveGeneral(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	source := registers.general[instruction.b]
	switch instruction.c {
	case moveGeneralModeAlias:
		registers.general[instruction.a] = source
	case moveGeneralModeSnapshot:
		if !source.IsValid() {
			registers.general[instruction.a] = source
		} else {
			registers.general[instruction.a] = copyReflectValueArena(vm.arena, source)
		}
	default:
		registers.general[instruction.a] = valueCopyForBoundaryArena(vm.arena, source)
	}
	return opContinue
}

// handleMoveSliceInt copies a slicesInt slice header.
//
// Used by the bytecode inliner during splice register remapping. Same-bank only;
// cross-bank conversions route through copyOneCallArgument or the dedicated adoption
// opcodes (subOpAdoptGeneralToSlicesFloat etc.).
//
// Takes registers (*Registers).
// Takes instruction (instruction) which encodes destination index in operand A and source
// index in operand B.
//
// Returns opResult signalling the VM dispatch loop to continue.
func handleMoveSliceInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.slicesInt[instruction.a] = registers.slicesInt[instruction.b]
	return opContinue
}

// handleMoveSliceFloat copies a slicesFloat slice header between typed-slice registers.
//
// Takes registers (*Registers).
// Takes instruction (instruction).
//
// Returns opResult.
func handleMoveSliceFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.slicesFloat[instruction.a] = registers.slicesFloat[instruction.b]
	return opContinue
}

// handleMoveSliceString copies a slicesString slice header between typed-slice registers.
//
// Takes registers (*Registers).
// Takes instruction (instruction).
//
// Returns opResult.
func handleMoveSliceString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.slicesString[instruction.a] = registers.slicesString[instruction.b]
	return opContinue
}

// handleMoveSliceBool copies a slicesBool slice header between typed-slice registers.
//
// Takes registers (*Registers).
// Takes instruction (instruction).
//
// Returns opResult.
func handleMoveSliceBool(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.slicesBool[instruction.a] = registers.slicesBool[instruction.b]
	return opContinue
}

// handleMoveSliceUint copies a slicesUint slice header between typed-slice registers.
//
// Takes registers (*Registers).
// Takes instruction (instruction).
//
// Returns opResult.
func handleMoveSliceUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.slicesUint[instruction.a] = registers.slicesUint[instruction.b]
	return opContinue
}

// handleMoveSliceByte copies a slicesByte slice header between typed-slice registers.
//
// Takes registers (*Registers).
// Takes instruction (instruction).
//
// Returns opResult.
func handleMoveSliceByte(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.slicesByte[instruction.a] = registers.slicesByte[instruction.b]
	return opContinue
}

// handleLoadIntConst loads a signed integer constant from the function constant pool into
// a register.
//
// Takes frame (*callFrame) which provides access to the function constant pool.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination register and constant
// pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadIntConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := instruction.wideIndex()
	if int(index) >= len(frame.function.intConstants) {
		vmBoundsError(vm, frame, boundsTableIntConstant, int(index), len(frame.function.intConstants))
		return opPanicError
	}
	registers.ints[instruction.a] = frame.function.intConstants[index]
	return opContinue
}

// handleLoadFloatConst loads a floating-point constant from the function constant pool
// into a register.
//
// Takes frame (*callFrame) which provides access to the function constant pool.
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination register and constant
// pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadFloatConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := instruction.wideIndex()
	if int(index) >= len(frame.function.floatConstants) {
		vmBoundsError(vm, frame, boundsTableFloatConstant, int(index), len(frame.function.floatConstants))
		return opPanicError
	}
	registers.floats[instruction.a] = frame.function.floatConstants[index]
	return opContinue
}

// handleLoadStringConst loads a string constant from the function constant pool into a
// register.
//
// Takes frame (*callFrame) which provides access to the function constant pool.
// Takes registers (*Registers) which provides the string register banks.
// Takes instruction (instruction) which encodes the destination register and constant
// pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadStringConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := instruction.wideIndex()
	if int(index) >= len(frame.function.stringConstants) {
		vmBoundsError(vm, frame, boundsTableStringConstant, int(index), len(frame.function.stringConstants))
		return opPanicError
	}
	registers.strings[instruction.a] = frame.function.stringConstants[index]
	return opContinue
}

// handleLoadGeneralConst loads a general constant from the function constant pool into a
// register.
//
// When the constant is a struct, a fresh addressable copy is created so that each
// invocation gets its own mutable value and pointer-receiver methods can be called.
//
// Takes frame (*callFrame) which provides access to the function constant pool.
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination register and constant
// pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadGeneralConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := instruction.wideIndex()
	if int(index) >= len(frame.function.generalConstants) {
		vmBoundsError(vm, frame, boundsTableGeneralConstant, int(index), len(frame.function.generalConstants))
		return opPanicError
	}
	v := frame.function.generalConstants[index]

	if v.Kind() == reflect.Struct {
		cp := reflect.New(v.Type()).Elem()
		cp.Set(v)
		v = cp
	}
	registers.general[instruction.a] = v
	return opContinue
}

// handleLoadNil loads an invalid reflect.Value representing nil into a general register.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination register index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadNil(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.general[instruction.a] = reflect.Value{}
	return opContinue
}

// handleLoadZero stores the zero value for the register kind specified by instruction.b
// into the destination register.
//
// Takes registers (*Registers) which provides all typed register banks.
// Takes instruction (instruction) which encodes the destination register and the register
// kind in operand B.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadZero(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	switch registerKind(instruction.b) {
	case registerInt:
		registers.ints[instruction.a] = 0
	case registerFloat:
		registers.floats[instruction.a] = 0
	case registerString:
		registers.strings[instruction.a] = ""
	case registerGeneral:
		registers.general[instruction.a] = reflect.Value{}
	case registerBool:
		registers.bools[instruction.a] = false
	case registerUint:
		registers.uints[instruction.a] = 0
	case registerComplex:
		registers.complex[instruction.a] = 0
	default:
	}
	return opContinue
}

// handleAddInt performs signed integer addition of two register operands in the virtual
// machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAddInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] + registers.ints[instruction.c]
	return opContinue
}

// handleSubInt performs signed integer subtraction of two register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] - registers.ints[instruction.c]
	return opContinue
}

// handleMulInt performs signed integer multiplication of two register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMulInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] * registers.ints[instruction.c]
	return opContinue
}

// handleDivInt performs signed integer division of two register operands in the virtual
// machine.
//
// When the divisor is zero, raises an interpreted integer-divide-by-zero panic instead of
// continuing.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue, or the result of
// raising an interpreted divide-by-zero panic when the divisor register holds zero.
func handleDivInt(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	divisor := registers.ints[instruction.c]
	if divisor == 0 {
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError(integerDivideByZeroMessage))
	}
	registers.ints[instruction.a] = registers.ints[instruction.b] / divisor
	return opContinue
}

// handleRemInt computes the signed integer remainder of two register operands in the
// virtual machine.
//
// When the divisor is zero, raises an interpreted integer-divide-by-zero panic instead of
// continuing.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue, or the result of
// raising an interpreted divide-by-zero panic when the divisor register holds zero.
func handleRemInt(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	divisor := registers.ints[instruction.c]
	if divisor == 0 {
		return raiseNativePanicAsInterpreted(vm, newRuntimePanicError(integerDivideByZeroMessage))
	}
	registers.ints[instruction.a] = registers.ints[instruction.b] % divisor
	return opContinue
}

// handleNegInt negates a signed integer register value in the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNegInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = -registers.ints[instruction.b]
	return opContinue
}

// handleBitAnd performs a bitwise AND of two signed integer register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitAnd(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] & registers.ints[instruction.c]
	return opContinue
}

// handleBitOr performs a bitwise OR of two signed integer register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitOr(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] | registers.ints[instruction.c]
	return opContinue
}

// handleBitXor performs a bitwise XOR of two signed integer register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitXor(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] ^ registers.ints[instruction.c]
	return opContinue
}

// handleBitAndNot performs a bitwise AND NOT of two signed integer register operands in
// the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitAndNot(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] &^ registers.ints[instruction.c]
	return opContinue
}

// handleBitNot performs a bitwise complement of a signed integer register value in the
// virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBitNot(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = ^registers.ints[instruction.b]
	return opContinue
}

// handleShiftLeft performs a left bit shift of a signed integer register by the amount in
// another register.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, value, and shift-amount
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleShiftLeft(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] << uint(registers.ints[instruction.c]) //nolint:gosec // register shift
	return opContinue
}

// handleShiftRight performs a right bit shift of a signed integer register by the amount
// in another register.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, value, and shift-amount
// register indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleShiftRight(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.ints[instruction.b] >> uint(registers.ints[instruction.c]) //nolint:gosec // register shift
	return opContinue
}

// handleSubIntConst subtracts a constant pool integer from a register value in the
// virtual machine.
//
// Takes frame (*callFrame) which provides access to the function constant pool.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, source, and constant
// pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubIntConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if int(instruction.c) >= len(frame.function.intConstants) {
		vmBoundsError(vm, frame, boundsTableIntConstant, int(instruction.c), len(frame.function.intConstants))
		return opPanicError
	}
	registers.ints[instruction.a] = registers.ints[instruction.b] - frame.function.intConstants[instruction.c]
	return opContinue
}

// handleAddIntConst adds a constant pool integer to a register value in the virtual
// machine.
//
// Takes frame (*callFrame) which provides access to the function constant pool.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination, source, and constant
// pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAddIntConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if int(instruction.c) >= len(frame.function.intConstants) {
		vmBoundsError(vm, frame, boundsTableIntConstant, int(instruction.c), len(frame.function.intConstants))
		return opPanicError
	}
	registers.ints[instruction.a] = registers.ints[instruction.b] + frame.function.intConstants[instruction.c]
	return opContinue
}

// handleLeIntConstJumpFalse compares a register against an integer constant and branches
// when the less-or-equal condition is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and program
// counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and constant pool
// index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeIntConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constantValue, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] > constantValue)
}

// handleLtIntConstJumpFalse compares a register against an integer constant and branches
// when the less-than condition is false.
//
// Takes frame (*callFrame) which provides access to the bytecode body and program
// counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the source register and constant pool
// index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtIntConstJumpFalse(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	constantValue, errResult, ok := intConstBoundsCheck(vm, frame, instruction)
	if !ok {
		return errResult
	}
	return conditionalJump(frame, registers.ints[instruction.a] >= constantValue)
}

// handleLtIntJumpFalse compares two register operands and branches by the extension-word
// offset when the less-than condition is false. Fuses opLtInt + opJumpIfFalse.
//
// Takes frame and registers (the comparison operands live in ints[A] and ints[B]) and
// instruction (which encodes the operand registers in A and B).
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtIntJumpFalse(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return conditionalJump(frame, registers.ints[instruction.a] >= registers.ints[instruction.b])
}

// handleLeIntJumpFalse compares two register operands and branches by the extension-word
// offset when the less-or-equal condition is false. Fuses opLeInt + opJumpIfFalse.
//
// Takes frame (*callFrame) which provides the bytecode body and PC.
// Takes registers (*Registers) which holds the integer operands.
// Takes instruction (instruction) which encodes the operand register indices in A and B.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeIntJumpFalse(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return conditionalJump(frame, registers.ints[instruction.a] > registers.ints[instruction.b])
}

// handleGtIntJumpFalse compares two register operands and branches by the extension-word
// offset when the greater-than condition is false. Fuses opGtInt + opJumpIfFalse.
//
// Takes frame (*callFrame) which provides the bytecode body and PC.
// Takes registers (*Registers) which holds the integer operands.
// Takes instruction (instruction) which encodes the operand register indices in A and B.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtIntJumpFalse(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return conditionalJump(frame, registers.ints[instruction.a] <= registers.ints[instruction.b])
}

// handleGeIntJumpFalse compares two register operands and branches by the extension-word
// offset when the greater-or-equal condition is false. Fuses opGeInt + opJumpIfFalse.
//
// Takes frame (*callFrame) which provides the bytecode body and PC.
// Takes registers (*Registers) which holds the integer operands.
// Takes instruction (instruction) which encodes the operand register indices in A and B.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeIntJumpFalse(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return conditionalJump(frame, registers.ints[instruction.a] < registers.ints[instruction.b])
}

// handleEqIntJumpFalse compares two register operands and branches by the extension-word
// offset when the equality condition is false. Fuses opEqInt + opJumpIfFalse.
//
// Takes frame (*callFrame) which provides the bytecode body and PC.
// Takes registers (*Registers) which holds the integer operands.
// Takes instruction (instruction) which encodes the operand register indices in A and B.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqIntJumpFalse(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return conditionalJump(frame, registers.ints[instruction.a] != registers.ints[instruction.b])
}

// handleNeIntJumpFalse compares two register operands and branches by the extension-word
// offset when the inequality condition is false. Fuses opNeInt + opJumpIfFalse.
//
// Takes frame (*callFrame) which provides the bytecode body and PC.
// Takes registers (*Registers) which holds the integer operands.
// Takes instruction (instruction) which encodes the operand register indices in A and B.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeIntJumpFalse(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	return conditionalJump(frame, registers.ints[instruction.a] == registers.ints[instruction.b])
}

// handleAddFloat performs floating-point addition of two register operands in the virtual
// machine.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAddFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = registers.floats[instruction.b] + registers.floats[instruction.c]
	return opContinue
}

// handleSubFloat performs floating-point subtraction of two register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = registers.floats[instruction.b] - registers.floats[instruction.c]
	return opContinue
}

// handleMulFloat performs floating-point multiplication of two register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMulFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = registers.floats[instruction.b] * registers.floats[instruction.c]
	return opContinue
}

// handleDivFloat performs floating-point division of two register operands in the virtual
// machine.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleDivFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = registers.floats[instruction.b] / registers.floats[instruction.c]
	return opContinue
}

// handleNegFloat negates a floating-point register value in the virtual machine.
//
// Takes registers (*Registers) which provides the float register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNegFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = -registers.floats[instruction.b]
	return opContinue
}

// handleConcatString concatenates two string register values using the arena allocator.
//
// Takes vm (*VM) which provides access to the arena allocator.
// Takes registers (*Registers) which provides the string register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleConcatString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a := registers.strings[instruction.b]
	b := registers.strings[instruction.c]
	if vm.limits.maxStringSize > 0 && len(a)+len(b) > vm.limits.maxStringSize {
		vm.evalError = fmt.Errorf("%w: concat result %d bytes exceeds limit %d",
			errStringLimit, len(a)+len(b), vm.limits.maxStringSize)
		return opPanicError
	}
	registers.strings[instruction.a] = arenaConcatString(vm.arena, a, b)
	return opContinue
}

// handleLenString computes the byte length of a string register value and stores it as an
// integer.
//
// Takes registers (*Registers) which provides the string and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLenString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(len(registers.strings[instruction.b]))
	return opContinue
}

// handleAdd performs addition on two general register operands using reflection-based
// type dispatch.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAdd(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a, b := registers.general[instruction.b], registers.general[instruction.c]
	registers.general[instruction.a] = reflectBinaryOp(a, b, func(x, y int64) int64 { return x + y },
		func(x, y float64) float64 { return x + y }, func(x, y string) string { return x + y })
	return opContinue
}

// handleSub performs subtraction on two general register operands using reflection-based
// type dispatch.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSub(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a, b := registers.general[instruction.b], registers.general[instruction.c]
	registers.general[instruction.a] = reflectBinaryOp(a, b, func(x, y int64) int64 { return x - y },
		func(x, y float64) float64 { return x - y }, nil)
	return opContinue
}

// handleMul performs multiplication on two general register operands using
// reflection-based type dispatch.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMul(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a, b := registers.general[instruction.b], registers.general[instruction.c]
	registers.general[instruction.a] = reflectBinaryOp(a, b, func(x, y int64) int64 { return x * y },
		func(x, y float64) float64 { return x * y }, nil)
	return opContinue
}

// handleDiv performs division on two general register operands using reflection-based
// type dispatch.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleDiv(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a, b := registers.general[instruction.b], registers.general[instruction.c]
	registers.general[instruction.a] = reflectBinaryOp(a, b, func(x, y int64) int64 { return x / y },
		func(x, y float64) float64 { return x / y }, nil)
	return opContinue
}

// handleRem computes the remainder of two general register operands using
// reflection-based type dispatch.
//
// Takes registers (*Registers) which provides the general register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleRem(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	a, b := registers.general[instruction.b], registers.general[instruction.c]
	registers.general[instruction.a] = reflectBinaryOp(a, b, func(x, y int64) int64 { return x % y }, nil, nil)
	return opContinue
}

// handleEqInt tests equality of two signed integer register values and stores the boolean
// result as an int.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] == registers.ints[instruction.c])
	return opContinue
}

// handleNeInt tests inequality of two signed integer register values and stores the
// boolean result as an int.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] != registers.ints[instruction.c])
	return opContinue
}

// handleLtInt tests whether the first signed integer register is less than the second and
// stores the result.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] < registers.ints[instruction.c])
	return opContinue
}

// handleLeInt tests whether the first signed integer register is less than or equal to
// the second and stores the result.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] <= registers.ints[instruction.c])
	return opContinue
}

// handleGtInt tests whether the first signed integer register is greater than the second
// and stores the result.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] > registers.ints[instruction.c])
	return opContinue
}

// handleGeInt tests whether the first signed integer register is greater than or equal to
// the second and stores the result.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] >= registers.ints[instruction.c])
	return opContinue
}

// handleEqFloat tests equality of two floating-point register values and stores the
// boolean result as an int.
//
// Takes registers (*Registers) which provides the float and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] == registers.floats[instruction.c])
	return opContinue
}

// handleLtFloat tests whether the first float register is less than the second and stores
// the result as an int.
//
// Takes registers (*Registers) which provides the float and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] < registers.floats[instruction.c])
	return opContinue
}

// handleLeFloat tests whether the first float register is less than or equal to the
// second and stores the result.
//
// Takes registers (*Registers) which provides the float and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] <= registers.floats[instruction.c])
	return opContinue
}

// handleEqString tests equality of two string register values and stores the boolean
// result as an int.
//
// Takes registers (*Registers) which provides the string and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] == registers.strings[instruction.c])
	return opContinue
}

// handleLtString tests whether the first string register is lexicographically less than
// the second and stores the result.
//
// Takes registers (*Registers) which provides the string and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] < registers.strings[instruction.c])
	return opContinue
}

// handleLeString tests whether the first string register is lexicographically less than
// or equal to the second.
//
// Takes registers (*Registers) which provides the string and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] <= registers.strings[instruction.c])
	return opContinue
}

// handleEqInterfaceNil sets ints[A] = 1 when general[B] is the zero-value interface (no
// dynamic type, no dynamic value), and 0 otherwise. This matches Go's "interface holding
// typed nil != nil" rule that would otherwise be lost in reflectEqual's typed-nil fast
// path.
//
// Takes registers (*Registers) which provides the general and integer register banks.
// Takes instruction (instruction) which encodes the destination int register in operand A
// and the source general register in operand B.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqInterfaceNil(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	value := registers.general[instruction.b]
	if !value.IsValid() {
		registers.ints[instruction.a] = 1
		return opContinue
	}
	if value.Kind() == reflect.Interface && value.IsNil() {
		registers.ints[instruction.a] = 1
		return opContinue
	}
	registers.ints[instruction.a] = 0
	return opContinue
}

// handleNeInterfaceNil is the != mirror of handleEqInterfaceNil.
//
// Takes registers (*Registers) which provides the general and integer register banks.
// Takes instruction (instruction) which encodes the destination int register in operand A
// and the source general register in operand B.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeInterfaceNil(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	value := registers.general[instruction.b]
	if !value.IsValid() {
		registers.ints[instruction.a] = 0
		return opContinue
	}
	if value.Kind() == reflect.Interface && value.IsNil() {
		registers.ints[instruction.a] = 0
		return opContinue
	}
	registers.ints[instruction.a] = 1
	return opContinue
}

// handleEqGeneral compares two general-bank values via reflectEqual.
//
// Stores 1 in the destination int register when the values compare equal under Go's ==
// semantics for interfaces, 0 otherwise. Used for the general path of == when the operand
// static types do not collapse to a typed bank.
//
// Takes registers (*Registers) which holds the operands and destination.
// Takes instruction (instruction) which encodes the destination int register in A and the
// source general registers in B and C.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(reflectEqual(
		registers.general[instruction.b], registers.general[instruction.c]))
	return opContinue
}

// isNilableAndNil reports whether v is a nil-able kind (func, pointer, interface, slice,
// map, channel) and currently holds a nil value.
//
// Takes v (reflect.Value) which is the value to inspect for nil-ability and nil state.
//
// Returns true if v is a nil-able kind and currently nil, false otherwise.
func isNilableAndNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Pointer, reflect.Interface,
		reflect.Slice, reflect.Map, reflect.Chan:
		return v.IsNil()
	default:
	}
	return false
}

// handleLtGeneral tests whether the first general register is less than the second using
// reflection comparison.
//
// Takes registers (*Registers) which provides the general and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLtGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(reflectCompare(registers.general[instruction.b], registers.general[instruction.c]) < 0)
	return opContinue
}

// handleLeGeneral tests whether the first general register is less than or equal to the
// second using reflection.
//
// Takes registers (*Registers) which provides the general and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLeGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(reflectCompare(registers.general[instruction.b], registers.general[instruction.c]) <= 0)
	return opContinue
}

// handleGtGeneral tests whether the first general register is greater than the second
// using reflection comparison.
//
// Takes registers (*Registers) which provides the general and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(reflectCompare(registers.general[instruction.b], registers.general[instruction.c]) > 0)
	return opContinue
}

// handleGeGeneral tests whether the first general register is greater than or equal to
// the second using reflection.
//
// Takes registers (*Registers) which provides the general and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(reflectCompare(registers.general[instruction.b], registers.general[instruction.c]) >= 0)
	return opContinue
}

// handleNot performs a logical NOT on an integer register, storing 1 if the value is zero
// and 0 otherwise.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNot(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.ints[instruction.b] == 0)
	return opContinue
}

// handleJump performs an unconditional branch by adding a signed offset to the program
// counter.
//
// Takes frame (*callFrame) which provides access to the program counter.
// Takes instruction (instruction) which encodes the signed branch offset.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleJump(_ *VM, frame *callFrame, _ *Registers, instruction instruction) opResult {
	frame.programCounter += int(instruction.signedOffset())
	return opContinue
}

// handleJumpIfTrue performs a conditional branch when the integer condition register is
// non-zero.
//
// Takes frame (*callFrame) which provides access to the program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the condition register and branch offset.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleJumpIfTrue(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if registers.ints[instruction.a] != 0 {
		frame.programCounter += int(instruction.signedOffset())
	}
	return opContinue
}

// handleJumpIfFalse performs a conditional branch when the integer condition register is
// zero.
//
// Takes frame (*callFrame) which provides access to the program counter.
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the condition register and branch offset.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleJumpIfFalse(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if registers.ints[instruction.a] == 0 {
		frame.programCounter += int(instruction.signedOffset())
	}
	return opContinue
}

// handleUnpackInterface extracts a concrete value from an interface in a general register
// into a typed register.
//
// When the source value is invalid or nil, the destination register is set to its zero
// value.
//
// Takes registers (*Registers) which provides all typed register banks.
// Takes instruction (instruction) which encodes the source and destination register
// indices, and the target register kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUnpackInterface(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	v := registers.general[instruction.b]
	if v.IsValid() && v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if !v.IsValid() {
		unpackInterfaceZero(registers, instruction)
	} else {
		unpackInterfaceValue(registers, instruction, v)
	}
	return opContinue
}

// unpackInterfaceZero writes the zero value for the destination kind when the source
// reflect.Value is invalid (nil interface).
//
// Takes registers (*Registers) which provides the register file to write the zero value
// into.
// Takes instruction (instruction) which encodes the destination register index and the
// target register kind.
func unpackInterfaceZero(registers *Registers, instruction instruction) {
	switch registerKind(instruction.c) {
	case registerInt:
		registers.ints[instruction.a] = 0
	case registerFloat:
		registers.floats[instruction.a] = 0
	case registerString:
		registers.strings[instruction.a] = ""
	case registerGeneral:
		registers.general[instruction.a] = reflect.Value{}
	case registerBool:
		registers.bools[instruction.a] = false
	case registerUint:
		registers.uints[instruction.a] = 0
	case registerComplex:
		registers.complex[instruction.a] = 0
	default:
	}
}

// unpackInterfaceValue extracts a concrete value from a valid reflect.Value into the
// destination register bank.
//
// Takes registers (*Registers) which provides the register file to write the extracted
// value into.
// Takes instruction (instruction) which encodes the destination register index and the
// target register kind.
// Takes value (reflect.Value) which is the concrete value to extract from the interface.
func unpackInterfaceValue(registers *Registers, instruction instruction, value reflect.Value) {
	switch registerKind(instruction.c) {
	case registerInt:
		unpackInterfaceInt(registers, instruction.a, value)
	case registerFloat:
		registers.floats[instruction.a] = value.Float()
	case registerString:
		registers.strings[instruction.a] = value.String()
	case registerGeneral:
		registers.general[instruction.a] = value
	case registerBool:
		registers.bools[instruction.a] = value.Bool()
	case registerUint:
		registers.uints[instruction.a] = value.Uint()
	case registerComplex:
		registers.complex[instruction.a] = value.Complex()
	default:
	}
}

// unpackInterfaceInt handles the registerInt case which requires checking multiple
// numeric kinds (signed, unsigned, bool).
//
// Takes registers (*Registers) which provides the register file to write the integer
// value into.
// Takes destination (uint8) which is the index of the target integer register.
// Takes value (reflect.Value) which is the value to extract the integer from.
func unpackInterfaceInt(registers *Registers, destination uint8, value reflect.Value) {
	if value.CanInt() {
		registers.ints[destination] = value.Int()
	} else if value.CanUint() {
		registers.ints[destination] = int64(value.Uint()) //nolint:gosec // unsigned->signed reinterpret
	} else if value.Kind() == reflect.Bool {
		registers.ints[destination] = boolToInt64(value.Bool())
	}
}

// handlePackInterface wraps a typed register value into a reflect.Value and stores it in
// a general register.
//
// Takes vm (*VM) which provides access to the arena allocator for string materialisation.
// Takes registers (*Registers) which provides all typed register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices, and the source register kind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handlePackInterface(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	switch registerKind(instruction.c) {
	case registerInt:
		registers.general[instruction.a] = boxInt64ToGeneral(vm.arena, registers.ints[instruction.b])
	case registerFloat:
		registers.general[instruction.a] = boxFloat64ToGeneral(vm.arena, registers.floats[instruction.b])
	case registerString:
		registers.general[instruction.a] = boxStringToGeneral(vm.arena, materialiseString(vm.arena, registers.strings[instruction.b]))
	case registerGeneral:

		registers.general[instruction.a] = materialiseArenaValue(vm.arena, registers.general[instruction.b])
	case registerBool:
		registers.general[instruction.a] = boxBoolToGeneral(registers.bools[instruction.b])
	case registerUint:
		registers.general[instruction.a] = boxUint64ToGeneral(vm.arena, registers.uints[instruction.b])
	case registerComplex:
		registers.general[instruction.a] = boxComplex128ToGeneral(vm.arena, registers.complex[instruction.b])
	case registerSliceInt:
		registers.general[instruction.a] = packTypedSliceToGeneral(vm.arena, registers.slicesInt[instruction.b], intSliceReflectType)
	case registerSliceFloat:
		registers.general[instruction.a] = packTypedSliceFloatToGeneral(vm.arena, registers.slicesFloat[instruction.b])
	case registerSliceString:
		registers.general[instruction.a] = packTypedSliceStringToGeneral(vm.arena, registers.slicesString[instruction.b])
	case registerSliceBool:
		registers.general[instruction.a] = packTypedSliceBoolToGeneral(vm.arena, registers.slicesBool[instruction.b])
	case registerSliceUint:
		registers.general[instruction.a] = packTypedSliceUintToGeneral(vm.arena, registers.slicesUint[instruction.b])
	case registerSliceByte:
		if vm.arena != nil {
			registers.general[instruction.a] = arenaWrapByteSlice(vm.arena, registers.slicesByte[instruction.b])
		} else {
			registers.general[instruction.a] = reflect.ValueOf(registers.slicesByte[instruction.b])
		}
	default:
	}
	return opContinue
}

// packTypedSliceToGeneral wraps an []int64 slice into a reflect.Value.
//
// Avoids the per-call mallocgc that reflect.ValueOf incurs. Falls back to reflect.ValueOf
// when no arena is available (test paths). Mirrors arenaWrapByteSlice but parameterised
// by reflect.Type for the non-byte typed-slice variants.
//
// Takes arena (*RegisterArena) which provides the slice-header slab.
// Takes s ([]int64) which is the slice to wrap.
// Takes sliceType (reflect.Type) which is the cached reflect.Type for []int64 (or
// whatever element type the caller is wrapping).
//
// Returns the wrapped reflect.Value.
func packTypedSliceToGeneral(arena *RegisterArena, s []int64, sliceType reflect.Type) reflect.Value {
	if arena == nil {
		return reflect.ValueOf(s)
	}
	var data unsafe.Pointer
	if cap(s) > 0 {
		data = unsafe.Pointer(&s[:1][0])
	}
	return arenaWrapTypedSlice(arena, data, len(s), cap(s), sliceType)
}

// packTypedSliceFloatToGeneral wraps an []float64 slice via the arena slice-header pool.
// See packTypedSliceToGeneral.
//
// Takes arena (*RegisterArena) which provides the slice-header slab.
// Takes s ([]float64) which is the slice to wrap.
//
// Returns reflect.Value which wraps s with the cached float-slice type.
func packTypedSliceFloatToGeneral(arena *RegisterArena, s []float64) reflect.Value {
	if arena == nil {
		return reflect.ValueOf(s)
	}
	var data unsafe.Pointer
	if cap(s) > 0 {
		data = unsafe.Pointer(&s[:1][0])
	}
	return arenaWrapTypedSlice(arena, data, len(s), cap(s), floatSliceReflectType)
}

// packTypedSliceStringToGeneral wraps an []string slice via the arena slice-header pool.
// See packTypedSliceToGeneral.
//
// Takes arena (*RegisterArena) which provides the slice-header slab.
// Takes s ([]string) which is the slice to wrap.
//
// Returns reflect.Value which wraps s with the cached string-slice type.
func packTypedSliceStringToGeneral(arena *RegisterArena, s []string) reflect.Value {
	if arena == nil {
		return reflect.ValueOf(s)
	}
	var data unsafe.Pointer
	if cap(s) > 0 {
		data = unsafe.Pointer(&s[:1][0])
	}
	return arenaWrapTypedSlice(arena, data, len(s), cap(s), stringSliceReflectType)
}

// packTypedSliceBoolToGeneral wraps an []bool slice via the arena slice-header pool. See
// packTypedSliceToGeneral.
//
// Takes arena (*RegisterArena) which provides the slice-header slab.
// Takes s ([]bool) which is the slice to wrap.
//
// Returns reflect.Value which wraps s with the cached bool-slice type.
func packTypedSliceBoolToGeneral(arena *RegisterArena, s []bool) reflect.Value {
	if arena == nil {
		return reflect.ValueOf(s)
	}
	var data unsafe.Pointer
	if cap(s) > 0 {
		data = unsafe.Pointer(&s[:1][0])
	}
	return arenaWrapTypedSlice(arena, data, len(s), cap(s), boolSliceReflectType)
}

// packTypedSliceUintToGeneral wraps an []uint64 slice via the arena slice-header pool.
// See packTypedSliceToGeneral.
//
// Takes arena (*RegisterArena) which provides the slice-header slab.
// Takes s ([]uint64) which is the slice to wrap.
//
// Returns reflect.Value which wraps s with the cached uint-slice type.
func packTypedSliceUintToGeneral(arena *RegisterArena, s []uint64) reflect.Value {
	if arena == nil {
		return reflect.ValueOf(s)
	}
	var data unsafe.Pointer
	if cap(s) > 0 {
		data = unsafe.Pointer(&s[:1][0])
	}
	return arenaWrapTypedSlice(arena, data, len(s), cap(s), uintSliceReflectType)
}

// handlePackTyped boxes a typed-bank register value into the general bank while
// preserving its exact source-level reflect.Type.
//
// The instruction is two words wide: the second word is an opExt carrying a 16-bit
// typeTable index. handlePackTyped reads the index, resolves the reflect.Type, and
// reconstructs the boxed value with that precise type rather than the canonical
// int64/float64/etc. that handlePackInterface produces. This is what lets `int64(42)` and
// the untyped literal `5` survive as distinct dynamic types through an interface, so a
// type assertion can tell them apart (bug 686).
//
// Takes vm (*VM) which provides the arena allocator.
// Takes frame (*callFrame) which provides the typeTable and the extension word.
// Takes registers (*Registers) which provides the typed register banks.
// Takes instruction (instruction) which encodes A=general dest, B=source register,
// C=source registerKind.
//
// Returns opResult which signals the dispatch loop to continue.
func handlePackTyped(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	typeIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(typeIndex) >= len(frame.function.typeTable) {
		vmBoundsError(vm, frame, boundsTableTypeTable, int(typeIndex), len(frame.function.typeTable))
		return opPanicError
	}
	reflectType := frame.function.typeTable[typeIndex]
	if reflectType == nil {
		registers.general[instruction.a] = packInterfaceFallback(vm, registers, instruction)
		return opContinue
	}
	registers.general[instruction.a] = boxTypedToGeneral(vm.arena, reflectType, registers, instruction.b, registerKind(instruction.c))
	return opContinue
}

// packInterfaceFallback boxes a register value using the canonical (int64/float64/...)
// representation. Used by handlePackTyped when the recorded type index resolved to a nil
// reflect.Type, so behaviour degrades to the legacy opPackInterface semantics rather than
// panicking.
//
// Takes vm (*VM) which provides the arena.
// Takes registers (*Registers) which provides the source banks.
// Takes instruction (instruction) which encodes B=source, C=kind.
//
// Returns the boxed reflect.Value.
func packInterfaceFallback(vm *VM, registers *Registers, instruction instruction) reflect.Value {
	switch registerKind(instruction.c) {
	case registerInt:
		return boxInt64ToGeneral(vm.arena, registers.ints[instruction.b])
	case registerFloat:
		return boxFloat64ToGeneral(vm.arena, registers.floats[instruction.b])
	case registerString:
		return boxStringToGeneral(vm.arena, materialiseString(vm.arena, registers.strings[instruction.b]))
	case registerBool:
		return boxBoolToGeneral(registers.bools[instruction.b])
	case registerUint:
		return boxUint64ToGeneral(vm.arena, registers.uints[instruction.b])
	case registerComplex:
		return reflect.ValueOf(registers.complex[instruction.b])
	default:
		return registers.general[instruction.b]
	}
}

// boxTypedToGeneral reconstructs a register value as a reflect.Value of the exact
// reflect.Type reflectType. The source register is read via the registerKind-appropriate
// bank and the scalar is narrowed into a fresh addressable value of reflectType.
//
// When arena is non-nil, the fast path routes through the existing arena box slabs and
// builds the reflect.Value via unsafeNewAt - zero mallocgc per call. Sub-width integer
// widths exploit the LE aliasing already validated by saturatingFloatToIntConvert.
// Float32 and Complex64 cannot alias the float64/complex128 slabs (bit layout differs) so
// fall through to the reflect.New path.
//
// Takes arena (*RegisterArena) which provides the bump-allocated box slabs; may be nil in
// test contexts.
// Takes reflectType (reflect.Type) which is the precise source-level type to clothe the
// value in.
// Takes registers (*Registers) which provides the typed banks.
// Takes sourceRegister (uint8) which is the source slot.
// Takes sourceKind (registerKind) which selects the source bank.
//
// Returns the boxed reflect.Value carrying reflectType identity.
func boxTypedToGeneral(arena *RegisterArena, reflectType reflect.Type, registers *Registers, sourceRegister uint8, sourceKind registerKind) reflect.Value {
	kind := reflectType.Kind()
	if arena != nil {
		if value, ok := boxScalarToArenaBox(arena, reflectType, kind, registers, sourceRegister); ok {
			return value
		}
	}
	out := reflect.New(reflectType).Elem()
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		out.SetInt(registers.ints[sourceRegister])
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		out.SetUint(registers.uints[sourceRegister])
	case reflect.Float32, reflect.Float64:
		out.SetFloat(registers.floats[sourceRegister])
	case reflect.String:
		out.SetString(registers.strings[sourceRegister])
	case reflect.Bool:
		out.SetBool(registers.bools[sourceRegister])
	case reflect.Complex64, reflect.Complex128:
		out.SetComplex(registers.complex[sourceRegister])
	default:
		switch sourceKind {
		case registerInt:
			return boxInt64ToGeneral(nil, registers.ints[sourceRegister])
		case registerFloat:
			return boxFloat64ToGeneral(nil, registers.floats[sourceRegister])
		case registerString:
			return reflect.ValueOf(registers.strings[sourceRegister])
		case registerBool:
			return boxBoolToGeneral(registers.bools[sourceRegister])
		case registerUint:
			return boxUint64ToGeneral(nil, registers.uints[sourceRegister])
		default:
			return registers.general[sourceRegister]
		}
	}
	return out
}

// boxScalarToArenaBox is the arena-allocating scalar boxing path.
//
// Routes scalar kinds with a bump-allocated arena slot variant to their typed slab;
// compound/non-scalar kinds (Slice, Struct, Map, Pointer, Interface, Func, Chan, Array,
// UnsafePointer) plus the invalid/untyped slots fall through to the reflect.New path.
//
// Takes arena (*RegisterArena) which provides the bump-allocated per-kind box slabs.
// Takes reflectType (reflect.Type) which is the declared destination type used to mint
// the result reflect.Value's ABI token.
// Takes kind (reflect.Kind) which is reflectType.Kind() pre-computed by the caller.
// Takes registers (*Registers) which provides the typed banks.
// Takes sourceRegister (uint8) which is the source slot.
//
// Returns reflect.Value which is the boxed scalar on match, or the zero Value for
// compound kinds.
// Returns bool which is true on a scalar match and false for compound kinds.
//
//nolint:cyclop // structural switch over reflect.Kind
func boxScalarToArenaBox(arena *RegisterArena, reflectType reflect.Type, kind reflect.Kind, registers *Registers, sourceRegister uint8) (reflect.Value, bool) {
	abiType := reflectValueABIType(reflectType)
	switch kind { //nolint:exhaustive // compound kinds (Slice/Struct/Map/...) return false for the caller's reflect.New fallback
	case reflect.Int:
		slot := arena.AllocIntBox(registers.ints[sourceRegister])
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Int), true
	case reflect.Int8:
		//nolint:gosec // intentional narrowing matching reflect.Value.SetInt wrap semantics
		slot := arena.AllocIntBox(int64(int8(registers.ints[sourceRegister])))
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Int8), true
	case reflect.Int16:
		//nolint:gosec // intentional narrowing matching reflect.Value.SetInt wrap semantics
		slot := arena.AllocIntBox(int64(int16(registers.ints[sourceRegister])))
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Int16), true
	case reflect.Int32:
		//nolint:gosec // intentional narrowing matching reflect.Value.SetInt wrap semantics
		slot := arena.AllocIntBox(int64(int32(registers.ints[sourceRegister])))
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Int32), true
	case reflect.Int64:
		slot := arena.AllocIntBox(registers.ints[sourceRegister])
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Int64), true
	case reflect.Uint:
		slot := arena.AllocUintBox(registers.uints[sourceRegister])
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Uint), true
	case reflect.Uint8:
		//nolint:gosec // intentional narrowing matching reflect.Value.SetUint wrap semantics
		slot := arena.AllocUintBox(uint64(uint8(registers.uints[sourceRegister])))
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Uint8), true
	case reflect.Uint16:
		//nolint:gosec // intentional narrowing matching reflect.Value.SetUint wrap semantics
		slot := arena.AllocUintBox(uint64(uint16(registers.uints[sourceRegister])))
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Uint16), true
	case reflect.Uint32:
		//nolint:gosec // intentional narrowing matching reflect.Value.SetUint wrap semantics
		slot := arena.AllocUintBox(uint64(uint32(registers.uints[sourceRegister])))
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Uint32), true
	case reflect.Uint64:
		slot := arena.AllocUintBox(registers.uints[sourceRegister])
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Uint64), true
	case reflect.Uintptr:
		slot := arena.AllocUintBox(registers.uints[sourceRegister])
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Uintptr), true
	case reflect.Float64:
		slot := arena.AllocFloatBox(registers.floats[sourceRegister])
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Float64), true
	case reflect.String:
		slot := arena.AllocStringBox(registers.strings[sourceRegister])
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.String), true
	case reflect.Bool:
		return boxBoolToGeneral(registers.bools[sourceRegister]), true
	case reflect.Complex128:
		slot := arena.AllocComplexBox(registers.complex[sourceRegister])
		return unsafeNewAt(abiType, unsafe.Pointer(slot), reflect.Complex128), true
	}
	return reflect.Value{}, false
}

// handleIntToFloat converts a signed integer register value to float64 and stores it in a
// float register.
//
// Takes registers (*Registers) which provides the integer and float register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleIntToFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = float64(registers.ints[instruction.b])
	return opContinue
}

// handleFloatToInt converts a floating-point register value to int64 and stores it in an
// integer register.
//
// Takes registers (*Registers) which provides the float and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleFloatToInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(registers.floats[instruction.b])
	return opContinue
}

// handleMoveBool copies a boolean value between virtual machine registers.
//
// Takes registers (*Registers) which provides the boolean register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMoveBool(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.bools[instruction.a] = registers.bools[instruction.b]
	return opContinue
}

// handleLoadBoolConst loads a boolean constant from the function constant pool into a
// register.
//
// Takes frame (*callFrame) which provides access to the function constant pool.
// Takes registers (*Registers) which provides the boolean register banks.
// Takes instruction (instruction) which encodes the destination register and constant
// pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadBoolConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if int(instruction.b) >= len(frame.function.boolConstants) {
		vmBoundsError(vm, frame, boundsTableBoolConstant, int(instruction.b), len(frame.function.boolConstants))
		return opPanicError
	}
	registers.bools[instruction.a] = frame.function.boolConstants[instruction.b]
	return opContinue
}

// handleLoadComplexConst loads a complex number constant from the function constant pool
// into a register.
//
// Takes frame (*callFrame) which provides access to the function constant pool.
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination register and constant
// pool index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleLoadComplexConst(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	index := instruction.wideIndex()
	if int(index) >= len(frame.function.complexConstants) {
		vmBoundsError(vm, frame, boundsTableComplexConstant, int(index), len(frame.function.complexConstants))
		return opPanicError
	}
	registers.complex[instruction.a] = frame.function.complexConstants[index]
	return opContinue
}

// handleAddComplex performs complex number addition of two register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleAddComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = registers.complex[instruction.b] + registers.complex[instruction.c]
	return opContinue
}

// handleSubComplex performs complex number subtraction of two register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleSubComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = registers.complex[instruction.b] - registers.complex[instruction.c]
	return opContinue
}

// handleMulComplex performs complex number multiplication of two register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleMulComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = registers.complex[instruction.b] * registers.complex[instruction.c]
	return opContinue
}

// handleDivComplex performs complex number division of two register operands in the
// virtual machine.
//
// Takes registers (*Registers) which provides the complex register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleDivComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = registers.complex[instruction.b] / registers.complex[instruction.c]
	return opContinue
}

// handleEqComplex tests equality of two complex register values and stores the boolean
// result as an int.
//
// Takes registers (*Registers) which provides the complex and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleEqComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.complex[instruction.b] == registers.complex[instruction.c])
	return opContinue
}

// handleNeComplex tests inequality of two complex register values and stores the boolean
// result as an int.
//
// Takes registers (*Registers) which provides the complex and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.complex[instruction.b] != registers.complex[instruction.c])
	return opContinue
}

// handleIntToUint converts a signed integer register value to uint64 and stores it in an
// unsigned register.
//
// Takes registers (*Registers) which provides the integer and unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleIntToUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = safeconv.Int64ToUint64Reinterpret(registers.ints[instruction.b])
	return opContinue
}

// handleUintToInt converts an unsigned integer register value to int64 and stores it in a
// signed register.
//
// Takes registers (*Registers) which provides the unsigned integer and integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUintToInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = safeconv.Uint64ToInt64Reinterpret(registers.uints[instruction.b])
	return opContinue
}

// handleUintToFloat converts an unsigned integer register value to float64 and stores it
// in a float register.
//
// Takes registers (*Registers) which provides the unsigned integer and float register
// banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleUintToFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = float64(registers.uints[instruction.b])
	return opContinue
}

// handleFloatToUint converts a floating-point register value to uint64 and stores it in
// an unsigned register.
//
// Takes registers (*Registers) which provides the float and unsigned integer register
// banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleFloatToUint(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = uint64(registers.floats[instruction.b])
	return opContinue
}

// handleTruncateNarrow truncates a narrow integer register to its declared width.
//
// Operates in place on the bank selected by operand C (registerUint applies a zero-fill
// mask, registerInt masks then sign-extends to preserve signed wrap semantics for
// int8/int16/int32). Operand B is the bit width (8, 16, or 32); width 64 is a no-op so
// the compiler does not emit it.
//
// Takes registers (*Registers) which provides the narrow integer banks.
// Takes instruction (instruction) which encodes the register index, the bit width, and
// the bank registerKind.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleTruncateNarrow(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	bitWidth := uint(instruction.b)
	if registerKind(instruction.c) == registerUint {
		mask := uint64(1)<<bitWidth - 1
		registers.uints[instruction.a] &= mask
		return opContinue
	}
	shift := registerBitWidth - bitWidth
	value := registers.ints[instruction.a]
	registers.ints[instruction.a] = (value << shift) >> shift
	return opContinue
}

// handleBoolToInt converts a boolean register value to an integer representation and
// stores it in an int register.
//
// Takes registers (*Registers) which provides the boolean and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBoolToInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.bools[instruction.b])
	return opContinue
}

// handleIntToBool converts a signed integer register value to a boolean and stores it in
// a bool register.
//
// Takes registers (*Registers) which provides the integer and boolean register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleIntToBool(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.bools[instruction.a] = registers.ints[instruction.b] != 0
	return opContinue
}

// handleBuildComplex constructs a complex number from two float register values and
// stores it in a complex register.
//
// Takes registers (*Registers) which provides the float and complex register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleBuildComplex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.complex[instruction.a] = complex(registers.floats[instruction.b], registers.floats[instruction.c])
	return opContinue
}

// handleIncInt increments a signed integer register value by one in the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the target register index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleIncInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a]++
	return opContinue
}

// handleDecInt decrements a signed integer register value by one in the virtual machine.
//
// Takes registers (*Registers) which provides the integer register banks.
// Takes instruction (instruction) which encodes the target register index.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleDecInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a]--
	return opContinue
}

// handleNeFloat tests inequality of two floating-point register values and stores the
// boolean result as an int.
//
// Takes registers (*Registers) which provides the float and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] != registers.floats[instruction.c])
	return opContinue
}

// handleGtFloat tests whether the first float register is greater than the second and
// stores the result as an int.
//
// Takes registers (*Registers) which provides the float and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] > registers.floats[instruction.c])
	return opContinue
}

// handleGeFloat tests whether the first float register is greater than or equal to the
// second and stores the result.
//
// Takes registers (*Registers) which provides the float and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeFloat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.floats[instruction.b] >= registers.floats[instruction.c])
	return opContinue
}

// handleNeString tests inequality of two string register values and stores the boolean
// result as an int.
//
// Takes registers (*Registers) which provides the string and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] != registers.strings[instruction.c])
	return opContinue
}

// handleGtString tests whether the first string register is lexicographically greater
// than the second.
//
// Takes registers (*Registers) which provides the string and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGtString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] > registers.strings[instruction.c])
	return opContinue
}

// handleGeString tests whether the first string register is lexicographically greater
// than or equal to the second.
//
// Takes registers (*Registers) which provides the string and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleGeString(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(registers.strings[instruction.b] >= registers.strings[instruction.c])
	return opContinue
}

// reflectEqual compares two reflect.Value operands for equality using type-appropriate
// comparison strategies.
//
// When both values are invalid, they are considered equal. When only one is invalid,
// equality holds only if the valid value is a nil-able kind currently holding nil.
//
// Takes a (reflect.Value) which is the first operand to compare.
// Takes b (reflect.Value) which is the second operand to compare.
//
// Returns true if the two values are considered equal, false otherwise.
func reflectEqual(a, b reflect.Value) bool {
	if !a.IsValid() && !b.IsValid() {
		return true
	}
	if !a.IsValid() || !b.IsValid() {
		return reflectEqualOneInvalid(a, b)
	}
	if matched, equal := reflectEqualComparable(a, b); matched {
		return equal
	}
	if a.Kind() == reflect.Pointer && b.Kind() == reflect.Pointer {
		return a.Pointer() == b.Pointer()
	}
	if a.Kind() == reflect.Struct && b.Kind() == reflect.Struct && a.Type() == b.Type() {
		if a.Comparable() && b.Comparable() {
			return a.Equal(b)
		}
	}
	return reflect.DeepEqual(a.Interface(), b.Interface())
}

// reflectEqualOneInvalid handles equality when exactly one operand is invalid.
//
// Takes a (reflect.Value) which is the first operand.
// Takes b (reflect.Value) which is the second operand.
//
// Returns bool indicating whether the valid operand is a nilable nil.
func reflectEqualOneInvalid(a, b reflect.Value) bool {
	valid := a
	if !a.IsValid() {
		valid = b
	}
	return isNilableAndNil(valid)
}

// reflectEqualComparable attempts a fast-path comparison for numeric, string, and boolean
// reflect values.
//
// Takes a (reflect.Value) which is the first operand.
// Takes b (reflect.Value) which is the second operand.
//
// Returns matched (bool) which indicates whether a fast-path applied.
// Returns equal (bool) which holds the comparison result when matched is true.
func reflectEqualComparable(a, b reflect.Value) (matched bool, equal bool) {
	if a.CanInt() && b.CanInt() {
		return true, a.Int() == b.Int()
	}
	if a.CanUint() && b.CanUint() {
		return true, a.Uint() == b.Uint()
	}
	if a.CanFloat() && b.CanFloat() {
		return true, a.Float() == b.Float()
	}
	if a.Kind() == reflect.String && b.Kind() == reflect.String {
		return true, a.String() == b.String()
	}
	if a.Kind() == reflect.Bool && b.Kind() == reflect.Bool {
		return true, a.Bool() == b.Bool()
	}
	return false, false
}

// handleNeGeneral tests inequality of two general register values using reflection and
// stores the result.
//
// Takes registers (*Registers) which provides the general and integer register banks.
// Takes instruction (instruction) which encodes the destination and source register
// indices.
//
// Returns opResult which signals the VM dispatch loop to continue.
func handleNeGeneral(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = boolToInt64(!reflectEqual(
		registers.general[instruction.b], registers.general[instruction.c]))
	return opContinue
}
