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
	"math"
	"reflect"
	"unsafe"

	"piko.sh/piko/wdk/safeconv"
)

// asmCallMathSin is the trampoline for handlerSubOpMathSin.
//
// Reads floats[sourceIndex] from the float bank exposed via ctx.floatsBase, computes
// math.Sin, writes to floats[destinationIndex], and returns ctx so the ASM caller can
// restore R15.
//
// Takes ctx (*DispatchContext) which carries the live float bank base pointer.
// Takes destinationIndex (int64) which is the float register receiving the result.
// Takes sourceIndex (int64) which is the float register holding the operand.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallMathSin
//go:nosplit
func asmCallMathSin(ctx *DispatchContext, destinationIndex, sourceIndex int64) *DispatchContext {
	base := unsafe.Pointer(ctx.floatsBase)
	source := (*float64)(unsafe.Add(base, sourceIndex*8))
	destination := (*float64)(unsafe.Add(base, destinationIndex*8))
	*destination = math.Sin(*source)
	return ctx
}

// asmCallMathCos is the trampoline for handlerSubOpMathCos.
//
// Reads floats[sourceIndex], computes math.Cos, writes to floats[destinationIndex], and
// returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the float bank base pointer.
// Takes destinationIndex (int64) which is the float register receiving the result.
// Takes sourceIndex (int64) which is the float register holding the operand.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallMathCos
//go:nosplit
func asmCallMathCos(ctx *DispatchContext, destinationIndex, sourceIndex int64) *DispatchContext {
	base := unsafe.Pointer(ctx.floatsBase)
	source := (*float64)(unsafe.Add(base, sourceIndex*8))
	destination := (*float64)(unsafe.Add(base, destinationIndex*8))
	*destination = math.Cos(*source)
	return ctx
}

// asmCallMathExp is the trampoline for handlerSubOpMathExp.
//
// Reads floats[sourceIndex], computes math.Exp, writes to floats[destinationIndex], and
// returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the float bank base pointer.
// Takes destinationIndex (int64) which is the float register receiving the result.
// Takes sourceIndex (int64) which is the float register holding the operand.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallMathExp
//go:nosplit
func asmCallMathExp(ctx *DispatchContext, destinationIndex, sourceIndex int64) *DispatchContext {
	base := unsafe.Pointer(ctx.floatsBase)
	source := (*float64)(unsafe.Add(base, sourceIndex*8))
	destination := (*float64)(unsafe.Add(base, destinationIndex*8))
	*destination = math.Exp(*source)
	return ctx
}

// asmCallMathTan is the trampoline for handlerSubOpMathTan.
//
// Reads floats[sourceIndex], computes math.Tan, writes to floats[destinationIndex], and
// returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the float bank base pointer.
// Takes destinationIndex (int64) which is the float register receiving the result.
// Takes sourceIndex (int64) which is the float register holding the operand.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallMathTan
//go:nosplit
func asmCallMathTan(ctx *DispatchContext, destinationIndex, sourceIndex int64) *DispatchContext {
	base := unsafe.Pointer(ctx.floatsBase)
	source := (*float64)(unsafe.Add(base, sourceIndex*8))
	destination := (*float64)(unsafe.Add(base, destinationIndex*8))
	*destination = math.Tan(*source)
	return ctx
}

// asmCallMathMod is the trampoline for handlerSubOpMathMod.
//
// Computes floats[destinationIndex] = math.Mod(floats[source1Index],
// floats[source2Index]). This is a three-operand sub-op (tier 0 arity); source2Index is
// read from the next instruction word's A field by the real handler before the call.
// Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the float bank base pointer.
// Takes destinationIndex (int64) which is the float register receiving the result.
// Takes source1Index (int64) which is the float register holding the dividend.
// Takes source2Index (int64) which is the float register holding the divisor.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallMathMod
//go:nosplit
func asmCallMathMod(ctx *DispatchContext, destinationIndex, source1Index, source2Index int64) *DispatchContext {
	base := unsafe.Pointer(ctx.floatsBase)
	source1 := (*float64)(unsafe.Add(base, source1Index*8))
	source2 := (*float64)(unsafe.Add(base, source2Index*8))
	destination := (*float64)(unsafe.Add(base, destinationIndex*8))
	*destination = math.Mod(*source1, *source2)
	return ctx
}

// asmCallStrconvItoa is the trampoline for handlerSubOpStrconvItoa.
//
// Writes strings[destinationIndex] = arenaItoaString(ints[sourceIndex]). The result is
// backed by the arena's byte slab so the string write does not trigger a Go heap
// allocation in steady state. Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the int and string bank base pointers plus
// the arena handle.
// Takes destinationIndex (int64) which is the string register receiving the formatted
// text.
// Takes sourceIndex (int64) which is the int register holding the value to format.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallStrconvItoa
//go:nosplit
func asmCallStrconvItoa(ctx *DispatchContext, destinationIndex, sourceIndex int64) *DispatchContext {
	intsBase := unsafe.Pointer(ctx.intsBase)
	source := *(*int64)(unsafe.Add(intsBase, sourceIndex*8))
	stringsBase := unsafe.Pointer(ctx.stringsBase)
	destination := (*string)(unsafe.Add(stringsBase, destinationIndex*16))
	*destination = arenaItoaString(ctx.vm.arena, source)
	return ctx
}

// asmCallStrconvFormatInt is the trampoline for handlerSubOpStrconvFormatInt.
//
// Writes strings[destinationIndex] = arenaFormatIntString(ints[source1Index],
// int(ints[source2Index])). Three-operand sub-op (tier 0 arity); the numeric base in
// source2Index is read from the next instruction word's A field by the real handler. Like
// asmCallStrconvItoa the result is arena-backed, so there is no Go heap allocation in
// steady state (only when the byte slab grows, which is amortised by EnsureCapacity at
// warmup). Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the int and string bank base pointers plus
// the arena handle.
// Takes destinationIndex (int64) which is the string register receiving the formatted
// text.
// Takes source1Index (int64) which is the int register holding the value to format.
// Takes source2Index (int64) which is the int register holding the numeric base.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallStrconvFormatInt
//go:nosplit
func asmCallStrconvFormatInt(ctx *DispatchContext, destinationIndex, source1Index, source2Index int64) *DispatchContext {
	intsBase := unsafe.Pointer(ctx.intsBase)
	source1 := *(*int64)(unsafe.Add(intsBase, source1Index*8))
	source2 := *(*int64)(unsafe.Add(intsBase, source2Index*8))
	stringsBase := unsafe.Pointer(ctx.stringsBase)
	destination := (*string)(unsafe.Add(stringsBase, destinationIndex*16))
	*destination = arenaFormatIntString(ctx.vm.arena, source1, int(source2))
	return ctx
}

// vmRegistersForCtx returns the *Registers for the frame the VM is dispatching.
//
// Used by trampolines that need to access the general (reflect.Value) bank, which cannot
// be safely indexed via raw pointer arithmetic. Reads ctx.framePointer (the live ASM-side
// fp) rather than ctx.vm.framePointer (the Go-side mirror). They normally match, but
// inside an ASM inline call the Go-side mirror lags behind: the inline-call ASM bumps
// only ctx.framePointer when descending into the callee, and syncCallContextFromASM
// writes the value back to vm.framePointer only when the dispatch loop exits. Tier-1 Go
// trampolines invoked during inline dispatch (e.g. asmCallMakeSliceByte for `make([]byte,
// ...)` inside an inline-called function) must therefore consult ctx.framePointer to find
// the active frame.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and the VM
// back-reference.
//
// Returns the *Registers for the frame at ctx.framePointer in vm.callStack.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT vmRegistersForCtx
//go:nosplit
func vmRegistersForCtx(ctx *DispatchContext) *Registers {
	frame := &ctx.vm.callStack[ctx.framePointer]
	return &frame.registers
}

// asmCallCap is the trampoline for handlerSubOpCap.
//
// Writes ints[destinationIndex] = collectionLengthOrCap(general[sourceIndex],
// reflect.Value.Cap). Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer used to locate the
// general bank.
// Takes destinationIndex (int64) which is the int register receiving the capacity.
// Takes sourceIndex (int64) which is the general register holding the collection value.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallCap
//go:nosplit
func asmCallCap(ctx *DispatchContext, destinationIndex, sourceIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	registers.ints[destinationIndex] = collectionLengthOrCap(registers.general[sourceIndex], reflect.Value.Cap)
	return ctx
}

// asmCallBytesToString is the trampoline for handlerSubOpBytesToString.
//
// Writes strings[destinationIndex] = arenaBytesToString(vm.arena,
// general[sourceIndex].Bytes()). Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer used to locate the
// general bank and the arena.
// Takes destinationIndex (int64) which is the string register receiving the converted
// text.
// Takes sourceIndex (int64) which is the general register holding the byte slice.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallBytesToString
//go:nosplit
func asmCallBytesToString(ctx *DispatchContext, destinationIndex, sourceIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	registers.strings[destinationIndex] = arenaBytesToString(ctx.vm.arena, registers.general[sourceIndex].Bytes())
	return ctx
}

// asmCallBoxSliceInt is the trampoline for handlerSubOpBoxSliceInt.
//
// Writes general[destinationIndex] = reflect.ValueOf(slicesInt[sourceIndex]), boxing a
// typed-bank int slice into the general (reflect.Value) bank. Returns ctx so the ASM
// caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer used to locate both
// banks.
// Takes destinationIndex (int64) which is the general register receiving the boxed slice.
// Takes sourceIndex (int64) which is the slicesInt register holding the source slice.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallBoxSliceInt
//go:nosplit
func asmCallBoxSliceInt(ctx *DispatchContext, destinationIndex, sourceIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	registers.general[destinationIndex] = reflect.ValueOf(registers.slicesInt[sourceIndex])
	return ctx
}

// asmCallUnboxSliceInt is the trampoline for handlerSubOpUnboxSliceInt.
//
// Writes slicesInt[destinationIndex] = reflect.TypeAssert[[]int64](general[sourceIndex]),
// unboxing a general-bank reflect.Value back into the typed slicesInt bank. The bank is
// left untouched when the type assertion fails. Returns ctx so the ASM caller can restore
// R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer used to locate both
// banks.
// Takes destinationIndex (int64) which is the slicesInt register receiving the unboxed
// slice.
// Takes sourceIndex (int64) which is the general register holding the boxed
// reflect.Value.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallUnboxSliceInt
//go:nosplit
func asmCallUnboxSliceInt(ctx *DispatchContext, destinationIndex, sourceIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	if asInts, ok := reflect.TypeAssert[[]int64](registers.general[sourceIndex]); ok {
		registers.slicesInt[destinationIndex] = asInts
	}
	return ctx
}

// asmCallMakeSliceInt is the trampoline for handlerSubOpMakeSliceInt.
//
// Writes slicesInt[destinationIndex] = arena-backed make([]int64, ints[lengthIndex],
// ints[capacityIndex]). Three-operand sub-op (tier 0 arity); capacityIndex is read from
// the next instruction word's A field by the real handler. Bounds violations panic via
// Go's runtime ("makeslice: len out of range") which propagates through the dispatch
// chain. The slice's backing array is bump-allocated from the arena's intBacking slab so
// the initial make does not trigger mallocgc; append-grow on the result also routes
// through arenaAppendInt (see vm_handler_append.go). Returns ctx so the ASM caller can
// restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesInt register receiving the new slice.
// Takes lengthIndex (int64) which is the int register holding the requested length.
// Takes capacityIndex (int64) which is the int register holding the requested capacity.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallMakeSliceInt
//go:nosplit
func asmCallMakeSliceInt(ctx *DispatchContext, destinationIndex, lengthIndex, capacityIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	length := registers.ints[lengthIndex]
	capacity := registers.ints[capacityIndex]
	backing := ctx.vm.arena.AllocIntBacking(int(capacity))
	clear(backing)
	registers.slicesInt[destinationIndex] = backing[:length:capacity]
	return ctx
}

// asmCallMakeSliceFloat is the trampoline for handlerSubOpMakeSliceFloat.
//
// Writes slicesFloat[destinationIndex] = arena-backed make([]float64, ints[lengthIndex],
// ints[capacityIndex]). Three-operand sub-op (tier 0 arity) backed by the arena's
// floatBacking slab. Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesFloat register receiving the new
// slice.
// Takes lengthIndex (int64) which is the int register holding the requested length.
// Takes capacityIndex (int64) which is the int register holding the requested capacity.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallMakeSliceFloat
//go:nosplit
func asmCallMakeSliceFloat(ctx *DispatchContext, destinationIndex, lengthIndex, capacityIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	length := registers.ints[lengthIndex]
	capacity := registers.ints[capacityIndex]
	backing := ctx.vm.arena.AllocFloatBacking(int(capacity))
	clear(backing)
	registers.slicesFloat[destinationIndex] = backing[:length:capacity]
	return ctx
}

// asmCallMakeSliceString is the trampoline for handlerSubOpMakeSliceString.
//
// Writes slicesString[destinationIndex] = arena-backed make([]string, ints[lengthIndex],
// ints[capacityIndex]). Three-operand sub-op (tier 0 arity) backed by the arena's
// stringBacking slab. Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesString register receiving the new
// slice.
// Takes lengthIndex (int64) which is the int register holding the requested length.
// Takes capacityIndex (int64) which is the int register holding the requested capacity.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallMakeSliceString
//go:nosplit
func asmCallMakeSliceString(ctx *DispatchContext, destinationIndex, lengthIndex, capacityIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	length := registers.ints[lengthIndex]
	capacity := registers.ints[capacityIndex]
	backing := ctx.vm.arena.AllocStringBacking(int(capacity))
	clear(backing)
	registers.slicesString[destinationIndex] = backing[:length:capacity]
	return ctx
}

// asmCallMakeSliceBool is the trampoline for handlerSubOpMakeSliceBool.
//
// Writes slicesBool[destinationIndex] = arena-backed make([]bool, ints[lengthIndex],
// ints[capacityIndex]). Three-operand sub-op (tier 0 arity) backed by the arena's
// boolBacking slab. Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesBool register receiving the new
// slice.
// Takes lengthIndex (int64) which is the int register holding the requested length.
// Takes capacityIndex (int64) which is the int register holding the requested capacity.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallMakeSliceBool
//go:nosplit
func asmCallMakeSliceBool(ctx *DispatchContext, destinationIndex, lengthIndex, capacityIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	length := registers.ints[lengthIndex]
	capacity := registers.ints[capacityIndex]
	backing := ctx.vm.arena.AllocBoolBacking(int(capacity))
	clear(backing)
	registers.slicesBool[destinationIndex] = backing[:length:capacity]
	return ctx
}

// asmCallMakeSliceUint is the trampoline for handlerSubOpMakeSliceUint.
//
// Writes slicesUint[destinationIndex] = arena-backed make([]uint64, ints[lengthIndex],
// ints[capacityIndex]). Three-operand sub-op (tier 0 arity) backed by the arena's
// uintBacking slab. Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesUint register receiving the new
// slice.
// Takes lengthIndex (int64) which is the int register holding the requested length.
// Takes capacityIndex (int64) which is the int register holding the requested capacity.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallMakeSliceUint
//go:nosplit
func asmCallMakeSliceUint(ctx *DispatchContext, destinationIndex, lengthIndex, capacityIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	length := registers.ints[lengthIndex]
	capacity := registers.ints[capacityIndex]
	backing := ctx.vm.arena.AllocUintBacking(int(capacity))
	clear(backing)
	registers.slicesUint[destinationIndex] = backing[:length:capacity]
	return ctx
}

// asmCallMakeSliceByte is the trampoline for handlerSubOpMakeSliceByte.
//
// Writes slicesByte[destinationIndex] = arena-backed make([]byte, ints[lengthIndex],
// ints[capacityIndex]). Three-operand sub-op (tier 0 arity); the byte backing comes from
// the arena's shared byteSlab (same path arenaMakeSliceBacking takes for the general-bank
// route). The body clears the live prefix to match Go's make semantics for typed byte
// slices. Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesByte register receiving the new
// slice.
// Takes lengthIndex (int64) which is the int register holding the requested length.
// Takes capacityIndex (int64) which is the int register holding the requested capacity.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallMakeSliceByte
//go:nosplit
func asmCallMakeSliceByte(ctx *DispatchContext, destinationIndex, lengthIndex, capacityIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	length := registers.ints[lengthIndex]
	capacity := registers.ints[capacityIndex]
	backing := ctx.vm.arena.AllocByteBacking(int(capacity))
	if length > 0 {
		clear(backing[:length])
	}
	registers.slicesByte[destinationIndex] = backing[:length:capacity]
	return ctx
}

// asmCallAppendSliceIntDirect is the trampoline for handlerSubOpAppendSliceIntDirect.
//
// Reads slicesInt[sourceIndex] and ints[elementIndex], appends the element to the source,
// and writes the resulting slice to slicesInt[destinationIndex]. Mirrors
// handleSubOpAppendSliceIntDirect's body without the maxAllocSize gate (the policy is
// enforced on the make-slice path that establishes the initial capacity; matches the
// established no-limit-in-trampoline convention used by every asmCallMakeSlice<Kind>).
// Append-grow routes through arenaAppendInt so the unsafe build can intercept allocation
// if it elects to. Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesInt register receiving the appended
// slice.
// Takes sourceIndex (int64) which is the slicesInt register holding the source slice.
// Takes elementIndex (int64) which is the int register holding the element value.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallAppendSliceIntDirect
//go:nosplit
func asmCallAppendSliceIntDirect(ctx *DispatchContext, destinationIndex, sourceIndex, elementIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	source := registers.slicesInt[sourceIndex]
	element := registers.ints[elementIndex]
	registers.slicesInt[destinationIndex] = arenaAppendInt(ctx.vm.arena, source, element)
	return ctx
}

// asmCallAppendSliceFloatDirect is the trampoline for handlerSubOpAppendSliceFloatDirect.
//
// Same shape as asmCallAppendSliceIntDirect but for []float64 with the element read from
// the float bank.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesFloat register receiving the appended
// slice.
// Takes sourceIndex (int64) which is the slicesFloat register holding the source slice.
// Takes elementIndex (int64) which is the float register holding the element value.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallAppendSliceFloatDirect
//go:nosplit
func asmCallAppendSliceFloatDirect(ctx *DispatchContext, destinationIndex, sourceIndex, elementIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	source := registers.slicesFloat[sourceIndex]
	element := registers.floats[elementIndex]
	registers.slicesFloat[destinationIndex] = arenaAppendFloat(ctx.vm.arena, source, element)
	return ctx
}

// asmCallAppendSliceStringDirect is the trampoline for
// handlerSubOpAppendSliceStringDirect.
//
// Reads slicesString[sourceIndex] and strings[elementIndex], routes the element through
// materialiseString (so an arena-borrowed header gets a proper backing if needed before
// being held by the destination slice), appends, and writes to
// slicesString[destinationIndex].
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesString register receiving the
// appended slice.
// Takes sourceIndex (int64) which is the slicesString register holding the source slice.
// Takes elementIndex (int64) which is the string register holding the element value.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallAppendSliceStringDirect
//go:nosplit
func asmCallAppendSliceStringDirect(ctx *DispatchContext, destinationIndex, sourceIndex, elementIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	source := registers.slicesString[sourceIndex]
	element := materialiseString(ctx.vm.arena, registers.strings[elementIndex])
	registers.slicesString[destinationIndex] = arenaAppendString(ctx.vm.arena, source, element)
	return ctx
}

// asmCallAppendSliceBoolDirect is the trampoline for handlerSubOpAppendSliceBoolDirect.
//
// Same shape as asmCallAppendSliceIntDirect but for []bool with the element read from the
// bool bank.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesBool register receiving the appended
// slice.
// Takes sourceIndex (int64) which is the slicesBool register holding the source slice.
// Takes elementIndex (int64) which is the bool register holding the element value.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallAppendSliceBoolDirect
//go:nosplit
func asmCallAppendSliceBoolDirect(ctx *DispatchContext, destinationIndex, sourceIndex, elementIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	source := registers.slicesBool[sourceIndex]
	element := registers.bools[elementIndex]
	registers.slicesBool[destinationIndex] = arenaAppendBool(ctx.vm.arena, source, element)
	return ctx
}

// asmCallAppendSliceUintDirect is the trampoline for handlerSubOpAppendSliceUintDirect.
//
// Same shape as asmCallAppendSliceIntDirect but for []uint64 with the element read from
// the uint bank.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesUint register receiving the appended
// slice.
// Takes sourceIndex (int64) which is the slicesUint register holding the source slice.
// Takes elementIndex (int64) which is the uint register holding the element value.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallAppendSliceUintDirect
//go:nosplit
func asmCallAppendSliceUintDirect(ctx *DispatchContext, destinationIndex, sourceIndex, elementIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	source := registers.slicesUint[sourceIndex]
	element := registers.uints[elementIndex]
	registers.slicesUint[destinationIndex] = arenaAppendUint(ctx.vm.arena, source, element)
	return ctx
}

// asmCallAppendSliceByteDirect is the trampoline for handlerSubOpAppendSliceByteDirect.
//
// Reads slicesByte[sourceIndex] and uints[elementIndex], truncates the uint64 element to
// a byte (matching handleSubOpAppendSliceByteDirect), appends, and writes to
// slicesByte[destinationIndex].
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and arena handle.
// Takes destinationIndex (int64) which is the slicesByte register receiving the appended
// slice.
// Takes sourceIndex (int64) which is the slicesByte register holding the source slice.
// Takes elementIndex (int64) which is the uint register whose low byte is appended.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallAppendSliceByteDirect
//go:nosplit
func asmCallAppendSliceByteDirect(ctx *DispatchContext, destinationIndex, sourceIndex, elementIndex int64) *DispatchContext {
	registers := vmRegistersForCtx(ctx)
	source := registers.slicesByte[sourceIndex]
	element := safeconv.Uint64ToByteTruncate(registers.uints[elementIndex])
	registers.slicesByte[destinationIndex] = arenaAppendByte(ctx.vm.arena, source, element)
	return ctx
}

// asmCallSetupGeneralBank finishes an inline call's general-bank setup.
//
// Invoked from handlerCallInlineSetupGeneralBank when the callee's isFastPath == 3. The
// ASM has already allocated the int/float/string/bool/uint banks for the callee frame at
// ctx.framePointer (already incremented to point at the callee). The trampoline slices
// arena.generalSlab into the callee frame's registers.general, copies the general-bank
// arguments from the caller frame using Go's typed assignment so GC write barriers fire
// on the reflect.Value payload (the ASM side cannot do this safely with raw MOVQs), and
// stashes the callee's freshly entered dispatch state into asmDispatchSaves[callee_fp] so
// any deeper Go-side push can later restore the callee correctly. Returns the ctx pointer
// so the ASM caller can recover R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer, arena handle, and
// dispatch-save table.
// Takes callInfo (*asmCallInfo) which holds the callee's general count, argument count,
// source register indices, and code/constant base pointers.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmCallSetupGeneralBank
func asmCallSetupGeneralBank(ctx *DispatchContext, callInfo *asmCallInfo) *DispatchContext {
	fp := int(ctx.framePointer)
	ctx.vm.callStack[fp].hasGeneralAlloc = true
	arena := ctx.vm.arena
	asmSetupCalleeGeneralBank(ctx, callInfo, arena, fp)
	asmCopyGeneralArguments(ctx, callInfo, arena, fp)
	asmSaveCalleeDispatchState(ctx, callInfo, fp)
	return ctx
}

// asmSetupCalleeGeneralBank slices the callee's general-bank registers out of the arena's
// general slab, growing the slab when the callee's register count would not fit.
//
// Takes ctx (*DispatchContext) which owns the call stack.
// Takes callInfo (*asmCallInfo) which describes the callee's register layout.
// Takes arena (*RegisterArena) which provides the general slab storage.
// Takes fp (int) which is the frame pointer index of the callee on the call stack.
//
//nolint:unused // called only from asmCallSetupGeneralBank
//go:nosplit
func asmSetupCalleeGeneralBank(ctx *DispatchContext, callInfo *asmCallInfo, arena *RegisterArena, fp int) {
	count := callInfo.calleeGeneralCount
	if count == 0 {
		return
	}
	start := arena.generalIndex
	end := start + int(count)
	if end > len(arena.generalSlab) {
		arena.growGeneralSlab(end)
	}
	ctx.vm.callStack[fp].registers.general = arena.generalSlab[start:end]
	arena.generalIndex = end
	ctx.arenaGeneralIndex = int64(end)
}

// asmCopyGeneralArguments copies the general-bank arguments from the caller frame (at
// fp-1) into the callee frame using Go's typed assignment so GC write barriers fire
// correctly for the reflect.Value payload. The boundary-copy guard for Struct/Array kinds
// is inlined so the 99% recursive *node case stays branch-light.
//
// Takes ctx (*DispatchContext) which owns the call stack.
// Takes callInfo (*asmCallInfo) which describes argument source slots.
// Takes arena (*RegisterArena) which is used for boundary-copy slab growth.
// Takes fp (int) which is the frame pointer index of the callee on the call stack.
//
//nolint:unused // called only from asmCallSetupGeneralBank
//go:nosplit
func asmCopyGeneralArguments(ctx *DispatchContext, callInfo *asmCallInfo, arena *RegisterArena, fp int) {
	argumentCount := callInfo.generalArgumentCount
	if argumentCount == 0 {
		return
	}
	callerGen := ctx.vm.callStack[fp-1].registers.general
	calleeGen := ctx.vm.callStack[fp].registers.general
	n := int(argumentCount)
	for i := range n {
		sourceIndex := int(callInfo.generalArgumentSources[i])
		source := callerGen[sourceIndex]
		if !source.IsValid() {
			calleeGen[i] = source
			continue
		}
		k := source.Kind()
		if k != reflect.Struct && k != reflect.Array {
			calleeGen[i] = source
			continue
		}
		calleeGen[i] = valueCopyForBoundaryArenaWithVM(arena, ctx.vm, source)
	}
}

// asmSaveCalleeDispatchState stashes the callee's freshly entered dispatch state into
// asmDispatchSaves[callee_fp] so any deeper Go-side push can later restore the callee
// correctly.
//
// Takes ctx (*DispatchContext) which owns asmDispatchSaves.
// Takes callInfo (*asmCallInfo) which describes the callee's code and constants.
// Takes fp (int) which is the frame pointer index of the callee on the call stack.
//
//nolint:unused // called only from asmCallSetupGeneralBank
//go:nosplit
func asmSaveCalleeDispatchState(ctx *DispatchContext, callInfo *asmCallInfo, fp int) {
	if ctx.vm.asmDispatchSaves == nil || fp < 0 || fp >= len(ctx.vm.asmDispatchSaves) {
		return
	}
	save := &ctx.vm.asmDispatchSaves[fp]
	save.codeBase = callInfo.calleeBody
	save.codeLength = callInfo.calleeBodyLength
	save.intConstantsBase = callInfo.calleeIntConstants
	save.floatConstantsBase = callInfo.calleeFloatConstants
	save.stringConstantsBase = callInfo.calleeStringConstants
	save.boolConstantsBase = callInfo.calleeBoolConstants
}

// asmReturnClearGeneralBank releases an inline callee's general-bank allocation.
//
// Invoked from handlerCallInlineClearGeneralBank when the returning callee allocated a
// general bank. The emit site fires this BEFORE ctx.framePointer is decremented, so
// ctx.framePointer is still the callee fp at entry. The trampoline clears the
// general-slab range used by the callee frame so the GC no longer scans stale
// reflect.Value entries (the slab is GC-visible), restores arena.generalIndex from the
// callee's arenaSave so the space returns to the bump allocator, and updates
// ctx.arenaGeneralIndex to keep the ASM's view in sync so subsequent inline-calls
// snapshot the right pre-call value.
//
// Takes ctx (*DispatchContext) which carries the live callee frame pointer and arena
// handle.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmReturnClearGeneralBank
//go:nosplit
func asmReturnClearGeneralBank(ctx *DispatchContext) *DispatchContext {
	fp := int(ctx.framePointer)
	callee := &ctx.vm.callStack[fp]
	savedIdx := callee.arenaSave.generalIndex
	arena := ctx.vm.arena
	if savedIdx < arena.generalIndex {
		clear(arena.generalSlab[savedIdx:arena.generalIndex])
	}
	arena.generalIndex = savedIdx
	ctx.arenaGeneralIndex = int64(savedIdx)
	return ctx
}

// asmTailCallExecute performs a tail call from the ASM dispatch loop without exiting
// through the Go-side exit-reason switch.
//
// Mirrors processExitTailCall's body plus the loopRebuild branch's
// rebuildDispatchPointers call, bundled behind a single CALL from handlerTailCallInline.
// The inline path skips the exit-reason routing overhead, the syncCallContextFromASM
// round-trip, and the dispatch loop iteration's setup. The expensive parts
// (handleTailCall's arena restore/save/alloc plus arg snapshot/place) run in Go to keep
// correctness for the alias-bearing argument copy.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and (un-decremented)
// PC. The PC convention matches the exit handler: the caller writes the un-advanced PC
// (pointing AT the opTailCall instruction) into CTX_PC before calling.
//
// Returns the same ctx so the ASM caller can reload R15 from the trampoline's return
// slot.
//
//nolint:unused // called from generated Plan-9 ASM as TEXT asmTailCallExecute
func asmTailCallExecute(ctx *DispatchContext) *DispatchContext {
	vm := ctx.vm
	vm.syncCallContextFromASM(ctx)
	fp := int(ctx.framePointer)
	frame := &vm.callStack[fp]
	registers := &frame.registers
	frame.programCounter = int(ctx.programCounter)
	instruction := frame.function.body[frame.programCounter]
	frame.programCounter++
	handleTailCall(vm, frame, registers, instruction)
	vm.updateASMCallInfoBase()
	vm.refreshCallContext(ctx)
	vm.rebuildDispatchPointers(ctx, frame, registers)
	return ctx
}

// frameForCtx returns the *callFrame the ASM dispatcher is currently executing.
//
// Uses ctx.framePointer (the ASM-maintained int64 in the dispatch context) rather than
// ctx.vm.framePointer (the Go-side mirror, only refreshed via syncCallContextFromASM at
// dispatch exit). During a fully-ASM call sequence the two diverge: handlerCallInline
// bumps ctx.framePointer to the callee but does not touch vm.framePointer until the loop
// exits, so trampolines reached mid-call see vm.framePointer pointing at the caller.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and the VM
// back-reference.
//
// Returns the *callFrame at vm.callStack[ctx.framePointer].
//
//nolint:unused // called from generated Plan-9 ASM as TEXT frameForCtx
//go:nosplit
func frameForCtx(ctx *DispatchContext) *callFrame {
	return &ctx.vm.callStack[ctx.framePointer]
}

// trampolinePointerBase extracts the unsafe base for the Pointer-receiver fast path.
//
// Reads general[recvReg] without taking the address of a stack-resident reflect.Value,
// which Go's escape analysis pessimises when the caller is reached via an ASM trampoline;
// every call would heap-allocate the 24-byte reflect.Value. The body reads the {typ, ptr,
// flag} triplet directly from the general bank's backing memory via the slice header's
// data pointer.
//
// Takes registers (*Registers) which owns the general bank slice header.
// Takes recvReg (uint8) which is the general register index of the receiver.
//
// Returns the receiver's base pointer when the value's kind is reflect.Pointer.
// Returns false when the slot is empty, the kind is not Pointer, or the indirected
// pointer is nil; the caller should route to the reflect-walk slow path.
//
//nolint:unused // called from generated Plan-9 ASM via TEXT trampolinePointerBase
//go:nosplit
func trampolinePointerBase(registers *Registers, recvReg uint8) (unsafe.Pointer, bool) {
	slot := (*unsafeReflectValue)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(registers.general)), uintptr(recvReg)*24))
	if slot.typ == nil {
		return nil, false
	}
	if reflect.Kind(slot.flag&flagKindMask) != reflect.Pointer {
		return nil, false
	}
	if slot.flag&flagIndir != 0 {
		inner := *(*unsafe.Pointer)(slot.ptr)
		if inner == nil {
			return nil, false
		}
		return inner, true
	}
	if slot.ptr == nil {
		return nil, false
	}
	return slot.ptr, true
}

// asmCallIncStructFieldInt increments a signed-int struct field via the fast path.
//
// Tier-1 sub-op (op=opDrillTier1, A=sub-op id, B=recvReg, C= layoutIdx low byte) invoked
// through EmitInlineGoCallTwoOperandShim so the ASM caller passes (ctx, recvReg,
// layoutIdx). Resolves the field's unsafe address via trampolinePointerBase and INC's the
// appropriately sized integer in place; on receiver-shape mismatch or non-int Kind the
// body routes through incStructFieldIntSlowPath which calls the existing reflect-walk
// fallback. Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and structLayout
// table.
// Takes recvReg (int64) which is the general register holding the pointer receiver.
// Takes layoutIdx (int64) which is the index into the function's structLayoutTable.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:dupl,unused // Inc/Dec twin; merging blocks hot-path inline
//go:nosplit
func asmCallIncStructFieldInt(ctx *DispatchContext, recvReg, layoutIdx int64) *DispatchContext {
	frame := frameForCtx(ctx)
	layout := frame.function.structLayoutTable[layoutIdx]
	registers := &frame.registers
	base, ok := trampolinePointerBase(registers, safeconv.Int64ToUint8(recvReg))
	if !ok {
		incStructFieldIntSlowPath(ctx, frame, registers, recvReg, layout, +1)
		return ctx
	}
	fieldPtr := unsafe.Add(base, uintptr(layout.Offset))
	switch reflect.Kind(layout.Kind) {
	case reflect.Int:
		*(*int)(fieldPtr)++
	case reflect.Int8:
		*(*int8)(fieldPtr)++
	case reflect.Int16:
		*(*int16)(fieldPtr)++
	case reflect.Int32:
		*(*int32)(fieldPtr)++
	case reflect.Int64:
		*(*int64)(fieldPtr)++
	default:
		incStructFieldIntSlowPath(ctx, frame, registers, recvReg, layout, +1)
	}
	return ctx
}

// asmCallDecStructFieldInt decrements a signed-int struct field via the fast path.
//
// Sign-flipped twin of asmCallIncStructFieldInt kept separate so the hot-path INC and DEC
// bodies both remain inlinable. See that function for the dispatch shape and operand
// layout. Slow path routes through incStructFieldIntSlowPath with delta=-1.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and structLayout
// table.
// Takes recvReg (int64) which is the general register holding the pointer receiver.
// Takes layoutIdx (int64) which is the index into the function's structLayoutTable.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:dupl,unused // see asmCallIncStructFieldInt; kept inline for the hot path
//go:nosplit
func asmCallDecStructFieldInt(ctx *DispatchContext, recvReg, layoutIdx int64) *DispatchContext {
	frame := frameForCtx(ctx)
	layout := frame.function.structLayoutTable[layoutIdx]
	registers := &frame.registers
	base, ok := trampolinePointerBase(registers, safeconv.Int64ToUint8(recvReg))
	if !ok {
		incStructFieldIntSlowPath(ctx, frame, registers, recvReg, layout, -1)
		return ctx
	}
	fieldPtr := unsafe.Add(base, uintptr(layout.Offset))
	switch reflect.Kind(layout.Kind) {
	case reflect.Int:
		*(*int)(fieldPtr)--
	case reflect.Int8:
		*(*int8)(fieldPtr)--
	case reflect.Int16:
		*(*int16)(fieldPtr)--
	case reflect.Int32:
		*(*int32)(fieldPtr)--
	case reflect.Int64:
		*(*int64)(fieldPtr)--
	default:
		incStructFieldIntSlowPath(ctx, frame, registers, recvReg, layout, -1)
	}
	return ctx
}

// asmCallIncStructFieldUint increments an unsigned-int struct field via the fast path.
//
// Unsigned counterpart of asmCallIncStructFieldInt, dispatched the same way
// (op=opDrillTier1, A=sub-op id, B=recvReg, C=layoutIdx low byte) through
// EmitInlineGoCallTwoOperandShim. Resolves the field's unsafe address via
// trampolinePointerBase and INC's the appropriately sized unsigned integer in place; on
// receiver-shape mismatch or non-uint Kind the body routes through
// incStructFieldUintSlowPath. Returns ctx so the ASM caller can restore R15.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and structLayout
// table.
// Takes recvReg (int64) which is the general register holding the pointer receiver.
// Takes layoutIdx (int64) which is the index into the function's structLayoutTable.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:dupl,unused // Inc/Dec twin; merging blocks hot-path inline
//go:nosplit
func asmCallIncStructFieldUint(ctx *DispatchContext, recvReg, layoutIdx int64) *DispatchContext {
	frame := frameForCtx(ctx)
	layout := frame.function.structLayoutTable[layoutIdx]
	registers := &frame.registers
	base, ok := trampolinePointerBase(registers, safeconv.Int64ToUint8(recvReg))
	if !ok {
		incStructFieldUintSlowPath(ctx, frame, registers, recvReg, layout, +1)
		return ctx
	}
	fieldPtr := unsafe.Add(base, uintptr(layout.Offset))
	switch reflect.Kind(layout.Kind) {
	case reflect.Uint:
		*(*uint)(fieldPtr)++
	case reflect.Uint8:
		*(*uint8)(fieldPtr)++
	case reflect.Uint16:
		*(*uint16)(fieldPtr)++
	case reflect.Uint32:
		*(*uint32)(fieldPtr)++
	case reflect.Uint64:
		*(*uint64)(fieldPtr)++
	case reflect.Uintptr:
		*(*uintptr)(fieldPtr)++
	default:
		incStructFieldUintSlowPath(ctx, frame, registers, recvReg, layout, +1)
	}
	return ctx
}

// asmCallDecStructFieldUint decrements an unsigned-int struct field via the fast path.
//
// Sign-flipped twin of asmCallIncStructFieldUint kept separate so the hot-path INC and
// DEC bodies both remain inlinable. See that function for the dispatch shape and operand
// layout. Slow path routes through incStructFieldUintSlowPath with delta=-1.
//
// Takes ctx (*DispatchContext) which carries the live frame pointer and structLayout
// table.
// Takes recvReg (int64) which is the general register holding the pointer receiver.
// Takes layoutIdx (int64) which is the index into the function's structLayoutTable.
//
// Returns the same ctx so the ASM caller can reload R15 from AX.
//
//nolint:dupl,unused // see asmCallIncStructFieldUint; kept inline for the hot path
//go:nosplit
func asmCallDecStructFieldUint(ctx *DispatchContext, recvReg, layoutIdx int64) *DispatchContext {
	frame := frameForCtx(ctx)
	layout := frame.function.structLayoutTable[layoutIdx]
	registers := &frame.registers
	base, ok := trampolinePointerBase(registers, safeconv.Int64ToUint8(recvReg))
	if !ok {
		incStructFieldUintSlowPath(ctx, frame, registers, recvReg, layout, -1)
		return ctx
	}
	fieldPtr := unsafe.Add(base, uintptr(layout.Offset))
	switch reflect.Kind(layout.Kind) {
	case reflect.Uint:
		*(*uint)(fieldPtr)--
	case reflect.Uint8:
		*(*uint8)(fieldPtr)--
	case reflect.Uint16:
		*(*uint16)(fieldPtr)--
	case reflect.Uint32:
		*(*uint32)(fieldPtr)--
	case reflect.Uint64:
		*(*uint64)(fieldPtr)--
	case reflect.Uintptr:
		*(*uintptr)(fieldPtr)--
	default:
		incStructFieldUintSlowPath(ctx, frame, registers, recvReg, layout, -1)
	}
	return ctx
}

// incStructFieldIntSlowPath routes a failed Inc/Dec to the reflect-walk fallback.
//
// Called from the int Inc/Dec trampolines when the receiver is not the expected Pointer
// shape or the layout's Kind is not a sized signed integer. The synthesised instruction
// word keeps the fallback handler's existing parsing contract (op=opDrillTier1, A=Inc or
// Dec sub-op id selected from delta's sign, B=receiver register).
//
// Takes ctx (*DispatchContext) which carries the live VM pointer.
// Takes frame (*callFrame) which is the dispatching frame.
// Takes registers (*Registers) which is the frame's register bundle.
// Takes recvReg (int64) which is the general register holding the receiver.
// Takes layout (structFieldLayout) which describes the target field.
// Takes delta (int64) which is +1 for Inc and -1 for Dec.
//
//nolint:unused // called from generated Plan-9 ASM via TEXT incStructFieldIntSlowPath
//go:nosplit
func incStructFieldIntSlowPath(ctx *DispatchContext, frame *callFrame, registers *Registers, recvReg int64, layout structFieldLayout, delta int64) {
	instr := instruction{op: opDrillTier1, a: uint8(subOpIncStructFieldInt), b: safeconv.Int64ToUint8(recvReg)}
	if delta < 0 {
		instr.a = uint8(subOpDecStructFieldInt)
	}
	incDecStructFieldIntFallback(ctx.vm, frame, registers, instr, layout, delta)
}

// incStructFieldUintSlowPath routes a failed Inc/Dec to the reflect-walk fallback.
//
// Unsigned counterpart of incStructFieldIntSlowPath, called from the uint Inc/Dec
// trampolines when the fast path's shape check fails. The synthesised instruction word
// keeps the unsigned fallback's parsing contract (op=opDrillTier1, A=Inc or Dec sub-op id
// selected from delta's sign, B=receiver register).
//
// Takes ctx (*DispatchContext) which carries the live VM pointer.
// Takes frame (*callFrame) which is the dispatching frame.
// Takes registers (*Registers) which is the frame's register bundle.
// Takes recvReg (int64) which is the general register holding the receiver.
// Takes layout (structFieldLayout) which describes the target field.
// Takes delta (int64) which is +1 for Inc and -1 for Dec.
//
//nolint:unused // called from generated Plan-9 ASM via TEXT incStructFieldUintSlowPath
//go:nosplit
func incStructFieldUintSlowPath(ctx *DispatchContext, frame *callFrame, registers *Registers, recvReg int64, layout structFieldLayout, delta int64) {
	instr := instruction{op: opDrillTier1, a: uint8(subOpIncStructFieldUint), b: safeconv.Int64ToUint8(recvReg)}
	if delta < 0 {
		instr.a = uint8(subOpDecStructFieldUint)
	}
	incDecStructFieldUintFallback(ctx.vm, frame, registers, instr, layout, delta)
}
