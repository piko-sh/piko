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
	"unsafe"

	"piko.sh/piko/internal/interp/interp_domain/asm"
)

const (
	// offsetSliceLength is the byte offset of the length field within a Go slice header
	// (data pointer + length + capacity = 8 + 8 + 8).
	offsetSliceLength = 8

	// offsetSliceCapacity is the byte offset of the capacity field within a Go slice header.
	offsetSliceCapacity = 16
)

// ProvideDispatchContextOffsets returns the DispatchContext byte offsets.
//
// The asmgen header generator embeds these values as CTX_* defines in
// asm_dispatch_offsets.h. Computed via unsafe.Offsetof on the live struct so the
// generated header always reflects the current layout.
//
// Returns asm.DispatchContextOffsets populated from the current DispatchContext struct.
// Returned by pointer because the struct is large (~496 bytes) and gets threaded through
// several emitter helpers.
func ProvideDispatchContextOffsets() *asm.DispatchContextOffsets {
	var ctx DispatchContext
	offsets := &asm.DispatchContextOffsets{}
	populateDispatchCoreOffsets(&ctx, offsets)
	populateDispatchCallStackOffsets(&ctx, offsets)
	populateDispatchArenaAndRegisterOffsets(&ctx, offsets)
	return offsets
}

// populateDispatchCoreOffsets fills the program counter, code base, register length, jump
// table and exit fields.
//
// Takes ctx (*DispatchContext) whose field offsets are measured.
// Takes offsets (*asm.DispatchContextOffsets) which is the destination offsets table that
// receives the values.
func populateDispatchCoreOffsets(ctx *DispatchContext, offsets *asm.DispatchContextOffsets) {
	offsets.CodeBase = unsafe.Offsetof(ctx.codeBase)
	offsets.CodeLength = unsafe.Offsetof(ctx.codeLength)
	offsets.ProgramCounter = unsafe.Offsetof(ctx.programCounter)
	offsets.IntsBase = unsafe.Offsetof(ctx.intsBase)
	offsets.IntsLength = unsafe.Offsetof(ctx.intsLength)
	offsets.FloatsBase = unsafe.Offsetof(ctx.floatsBase)
	offsets.FloatsLength = unsafe.Offsetof(ctx.floatsLength)
	offsets.IntConstantsBase = unsafe.Offsetof(ctx.intConstantsBase)
	offsets.IntConstantsLength = unsafe.Offsetof(ctx.intConstantsLength)
	offsets.FloatConstantsBase = unsafe.Offsetof(ctx.floatConstantsBase)
	offsets.FloatConstantsLength = unsafe.Offsetof(ctx.floatConstantsLength)
	offsets.JumpTable = unsafe.Offsetof(ctx.jumpTable)
	offsets.ExitReason = unsafe.Offsetof(ctx.exitReason)
	offsets.ExitProgramCounter = unsafe.Offsetof(ctx.exitProgramCounter)
}

// populateDispatchCallStackOffsets fills the ASM call info, call stack, frame pointer and
// call depth fields.
//
// Takes ctx (*DispatchContext) whose field offsets are measured.
// Takes offsets (*asm.DispatchContextOffsets) which is the destination offsets table that
// receives the values.
func populateDispatchCallStackOffsets(ctx *DispatchContext, offsets *asm.DispatchContextOffsets) {
	offsets.AsmCallInfoBase = unsafe.Offsetof(ctx.asmCallInfoBase)
	offsets.CallStackBase = unsafe.Offsetof(ctx.callStackBase)
	offsets.CallStackLength = unsafe.Offsetof(ctx.callStackLength)
	offsets.FramePointer = unsafe.Offsetof(ctx.framePointer)
	offsets.BaseFramePointer = unsafe.Offsetof(ctx.baseFramePointer)
	offsets.CallDepthLimit = unsafe.Offsetof(ctx.callDepthLimit)
	offsets.DeferStackLength = unsafe.Offsetof(ctx.deferStackLength)
	offsets.AsmCallInfoBasesPtr = unsafe.Offsetof(ctx.asmCallInfoBasesPointer)
	offsets.DispatchSavesPtr = unsafe.Offsetof(ctx.dispatchSavesPointer)
}

// populateDispatchArenaAndRegisterOffsets fills the arena slab, capacity and index fields
// for every register family, plus the per-type register slab base pointers and the
// constant pool / struct layout / type table fields. Bundling the categories together
// avoids the dupl detector flagging two short, similarly shaped offset-population
// helpers.
//
// Takes ctx (*DispatchContext) whose field offsets are measured.
// Takes offsets (*asm.DispatchContextOffsets) which is the destination offsets table that
// receives the values.
func populateDispatchArenaAndRegisterOffsets(ctx *DispatchContext, offsets *asm.DispatchContextOffsets) {
	offsets.ArenaIntSlab = unsafe.Offsetof(ctx.arenaIntSlab)
	offsets.ArenaIntCapacity = unsafe.Offsetof(ctx.arenaIntCapacity)
	offsets.ArenaIntIndex = unsafe.Offsetof(ctx.arenaIntIndex)
	offsets.ArenaFloatSlab = unsafe.Offsetof(ctx.arenaFloatSlab)
	offsets.ArenaFloatCapacity = unsafe.Offsetof(ctx.arenaFloatCapacity)
	offsets.ArenaFloatIndex = unsafe.Offsetof(ctx.arenaFloatIndex)
	offsets.ArenaStringIndex = unsafe.Offsetof(ctx.arenaStringIndex)
	offsets.ArenaGeneralIndex = unsafe.Offsetof(ctx.arenaGeneralIndex)
	offsets.ArenaBoolIndex = unsafe.Offsetof(ctx.arenaBoolIndex)
	offsets.ArenaUintIndex = unsafe.Offsetof(ctx.arenaUintIndex)
	offsets.ArenaComplexIndex = unsafe.Offsetof(ctx.arenaComplexIndex)
	offsets.ArenaStringSlab = unsafe.Offsetof(ctx.arenaStringSlab)
	offsets.ArenaStringCapacity = unsafe.Offsetof(ctx.arenaStringCapacity)
	offsets.ArenaBoolSlab = unsafe.Offsetof(ctx.arenaBoolSlab)
	offsets.ArenaBoolCapacity = unsafe.Offsetof(ctx.arenaBoolCapacity)
	offsets.ArenaUintSlab = unsafe.Offsetof(ctx.arenaUintSlab)
	offsets.ArenaUintCapacity = unsafe.Offsetof(ctx.arenaUintCapacity)
	offsets.ArenaSliceByteSlab = unsafe.Offsetof(ctx.arenaSliceByteSlab)
	offsets.ArenaSliceByteCapacity = unsafe.Offsetof(ctx.arenaSliceByteCapacity)
	offsets.ArenaSliceByteIndex = unsafe.Offsetof(ctx.arenaSliceByteIndex)
	offsets.ArenaSliceIntIndex = unsafe.Offsetof(ctx.arenaSliceIntIndex)
	offsets.ArenaSliceFloatIndex = unsafe.Offsetof(ctx.arenaSliceFloatIndex)
	offsets.ArenaSliceStringIndex = unsafe.Offsetof(ctx.arenaSliceStringIndex)
	offsets.ArenaSliceBoolIndex = unsafe.Offsetof(ctx.arenaSliceBoolIndex)
	offsets.ArenaSliceUintIndex = unsafe.Offsetof(ctx.arenaSliceUintIndex)
	offsets.StringsBase = unsafe.Offsetof(ctx.stringsBase)
	offsets.UintsBase = unsafe.Offsetof(ctx.uintsBase)
	offsets.BoolsBase = unsafe.Offsetof(ctx.boolsBase)
	offsets.SlicesIntBase = unsafe.Offsetof(ctx.slicesIntBase)
	offsets.SlicesFloatBase = unsafe.Offsetof(ctx.slicesFloatBase)
	offsets.SlicesStringBase = unsafe.Offsetof(ctx.slicesStringBase)
	offsets.SlicesBoolBase = unsafe.Offsetof(ctx.slicesBoolBase)
	offsets.SlicesUintBase = unsafe.Offsetof(ctx.slicesUintBase)
	offsets.SlicesByteBase = unsafe.Offsetof(ctx.slicesByteBase)
	offsets.ComplexBase = unsafe.Offsetof(ctx.complexBase)
	offsets.StringConstantsBase = unsafe.Offsetof(ctx.stringConstantsBase)
	offsets.StringConstantsLength = unsafe.Offsetof(ctx.stringConstantsLength)
	offsets.BoolConstantsBase = unsafe.Offsetof(ctx.boolConstantsBase)
	offsets.BoolConstantsLength = unsafe.Offsetof(ctx.boolConstantsLength)
	offsets.SavedPC = unsafe.Offsetof(ctx.savedPC)
	offsets.Tier2Result = unsafe.Offsetof(ctx.tier2Result)
	offsets.StructLayoutTableBase = unsafe.Offsetof(ctx.structLayoutTableBase)
	offsets.StructLayoutTableLength = unsafe.Offsetof(ctx.structLayoutTableLength)
	offsets.TypeTableBase = unsafe.Offsetof(ctx.typeTableBase)
	offsets.TypeTableLength = unsafe.Offsetof(ctx.typeTableLength)
}

// ProvideCallFrameOffsets returns the size of the runtime callFrame struct and the byte
// offsets of every callFrame field accessed by the inline call and return ASM handlers.
// Values are computed via unsafe.Offsetof so the asmgen-generated header always reflects
// the current layout.
//
// Returns *asm.CallFrameOffsets populated from the current callFrame struct. Returned by
// pointer because the struct is several hundred bytes; the lint hugeParam rule otherwise
// fires at every consumer.
func ProvideCallFrameOffsets() *asm.CallFrameOffsets {
	var frame callFrame
	offsets := &asm.CallFrameOffsets{
		Size:            unsafe.Sizeof(frame),
		Function:        unsafe.Offsetof(frame.function),
		SharedCells:     unsafe.Offsetof(frame.sharedCells),
		Upvalues:        unsafe.Offsetof(frame.upvalues),
		ReturnDestPtr:   unsafe.Offsetof(frame.returnDestination),
		ReturnDestLen:   unsafe.Offsetof(frame.returnDestination) + offsetSliceLength,
		ReturnDestCap:   unsafe.Offsetof(frame.returnDestination) + offsetSliceCapacity,
		ProgramCounter:  unsafe.Offsetof(frame.programCounter),
		DeferBase:       unsafe.Offsetof(frame.deferBase),
		ArenaSave:       unsafe.Offsetof(frame.arenaSave),
		HasGeneralAlloc: unsafe.Offsetof(frame.hasGeneralAlloc),
	}
	populateCallFrameRegisterBankOffsets(&frame, offsets)
	return offsets
}

// populateCallFrameRegisterBankOffsets fills the per-bank offsets.
//
// Writes the 13 register-bank ptr/len/cap triples on offsets from the live callFrame
// layout. Split out from ProvideCallFrameOffsets to keep that constructor under the
// per-function line limit; the helper computes one base offset per bank and derives the
// slice-header triples from it.
//
// Takes frame (*callFrame) which is the live callFrame layout source.
// Takes offsets (*asm.CallFrameOffsets) which receives the per-bank offsets.
func populateCallFrameRegisterBankOffsets(frame *callFrame, offsets *asm.CallFrameOffsets) {
	registersOffset := unsafe.Offsetof(frame.registers)
	bankOffset := func(field uintptr) uintptr {
		return registersOffset + field
	}
	intsOffset := bankOffset(unsafe.Offsetof(frame.registers.ints))
	floatsOffset := bankOffset(unsafe.Offsetof(frame.registers.floats))
	stringsOffset := bankOffset(unsafe.Offsetof(frame.registers.strings))
	generalOffset := bankOffset(unsafe.Offsetof(frame.registers.general))
	boolsOffset := bankOffset(unsafe.Offsetof(frame.registers.bools))
	uintsOffset := bankOffset(unsafe.Offsetof(frame.registers.uints))
	complexOffset := bankOffset(unsafe.Offsetof(frame.registers.complex))
	slicesIntOffset := bankOffset(unsafe.Offsetof(frame.registers.slicesInt))
	slicesFloatOffset := bankOffset(unsafe.Offsetof(frame.registers.slicesFloat))
	slicesStringOffset := bankOffset(unsafe.Offsetof(frame.registers.slicesString))
	slicesBoolOffset := bankOffset(unsafe.Offsetof(frame.registers.slicesBool))
	slicesUintOffset := bankOffset(unsafe.Offsetof(frame.registers.slicesUint))
	slicesByteOffset := bankOffset(unsafe.Offsetof(frame.registers.slicesByte))
	offsets.RegsIntsPtr, offsets.RegsIntsLen, offsets.RegsIntsCap = intsOffset, intsOffset+offsetSliceLength, intsOffset+offsetSliceCapacity
	offsets.RegsFloatsPtr, offsets.RegsFloatsLen, offsets.RegsFloatsCap = floatsOffset, floatsOffset+offsetSliceLength, floatsOffset+offsetSliceCapacity
	offsets.RegsStringsPtr, offsets.RegsStringsLen, offsets.RegsStringsCap = stringsOffset, stringsOffset+offsetSliceLength, stringsOffset+offsetSliceCapacity
	offsets.RegsGeneralPtr, offsets.RegsGeneralLen, offsets.RegsGeneralCap = generalOffset, generalOffset+offsetSliceLength, generalOffset+offsetSliceCapacity
	offsets.RegsBoolsPtr, offsets.RegsBoolsLen, offsets.RegsBoolsCap = boolsOffset, boolsOffset+offsetSliceLength, boolsOffset+offsetSliceCapacity
	offsets.RegsUintsPtr, offsets.RegsUintsLen, offsets.RegsUintsCap = uintsOffset, uintsOffset+offsetSliceLength, uintsOffset+offsetSliceCapacity
	offsets.RegsComplexPtr, offsets.RegsComplexLen, offsets.RegsComplexCap = complexOffset, complexOffset+offsetSliceLength, complexOffset+offsetSliceCapacity
	offsets.RegsSlicesIntPtr, offsets.RegsSlicesIntLen, offsets.RegsSlicesIntCap = slicesIntOffset, slicesIntOffset+offsetSliceLength, slicesIntOffset+offsetSliceCapacity
	offsets.RegsSlicesFloatPtr, offsets.RegsSlicesFloatLen, offsets.RegsSlicesFloatCap = slicesFloatOffset, slicesFloatOffset+offsetSliceLength, slicesFloatOffset+offsetSliceCapacity
	offsets.RegsSlicesStringPtr, offsets.RegsSlicesStringLen, offsets.RegsSlicesStringCap = slicesStringOffset, slicesStringOffset+offsetSliceLength, slicesStringOffset+offsetSliceCapacity
	offsets.RegsSlicesBoolPtr, offsets.RegsSlicesBoolLen, offsets.RegsSlicesBoolCap = slicesBoolOffset, slicesBoolOffset+offsetSliceLength, slicesBoolOffset+offsetSliceCapacity
	offsets.RegsSlicesUintPtr, offsets.RegsSlicesUintLen, offsets.RegsSlicesUintCap = slicesUintOffset, slicesUintOffset+offsetSliceLength, slicesUintOffset+offsetSliceCapacity
	offsets.RegsSliceBytePtr, offsets.RegsSliceByteLen, offsets.RegsSliceByteCap = slicesByteOffset, slicesByteOffset+offsetSliceLength, slicesByteOffset+offsetSliceCapacity
}

// ProvideASMCallInfoOffsets returns the size and field offsets of the runtime asmCallInfo
// struct that the inline call dispatcher walks. Values are computed via unsafe.Offsetof
// so drift is impossible.
//
// Returns asm.ASMCallInfoOffsets populated from the current asmCallInfo struct. Returned
// by pointer because the struct is ~288 bytes.
func ProvideASMCallInfoOffsets() *asm.ASMCallInfoOffsets {
	var info asmCallInfo
	return &asm.ASMCallInfoOffsets{
		CalleeFunction:   unsafe.Offsetof(info.calleeFunction),
		CalleeBody:       unsafe.Offsetof(info.calleeBody),
		CalleeBodyLen:    unsafe.Offsetof(info.calleeBodyLength),
		CalleeIntConsts:  unsafe.Offsetof(info.calleeIntConstants),
		CalleeFltConsts:  unsafe.Offsetof(info.calleeFloatConstants),
		CalleeNumInts:    unsafe.Offsetof(info.calleeIntCount),
		CalleeNumFloats:  unsafe.Offsetof(info.calleeFloatCount),
		NumIntArgs:       unsafe.Offsetof(info.intArgumentCount),
		IntArgSrcs:       unsafe.Offsetof(info.intArgumentSources),
		NumFloatArgs:     unsafe.Offsetof(info.floatArgumentCount),
		FloatArgSrcs:     unsafe.Offsetof(info.floatArgumentSources),
		NumReturns:       unsafe.Offsetof(info.returnCount),
		RetDestKind:      unsafe.Offsetof(info.returnDestinationKind),
		RetDestReg:       unsafe.Offsetof(info.returnDestinationRegister),
		RetDestPtr:       unsafe.Offsetof(info.returnDestinationPointer),
		RetDestLen:       unsafe.Offsetof(info.returnDestinationLen),
		CalleeCallInfo:   unsafe.Offsetof(info.calleeCallInfo),
		IsFastPath:       unsafe.Offsetof(info.isFastPath),
		CalleeNumStrings: unsafe.Offsetof(info.calleeStringCount),
		CalleeNumBools:   unsafe.Offsetof(info.calleeBoolCount),
		CalleeNumUints:   unsafe.Offsetof(info.calleeUintCount),
		NumStringArgs:    unsafe.Offsetof(info.stringArgumentCount),
		StringArgSrcs:    unsafe.Offsetof(info.stringArgumentSources),
		NumBoolArgs:      unsafe.Offsetof(info.boolArgumentCount),
		BoolArgSrcs:      unsafe.Offsetof(info.boolArgumentSources),
		NumUintArgs:      unsafe.Offsetof(info.uintArgumentCount),
		UintArgSrcs:      unsafe.Offsetof(info.uintArgumentSources),
		CalleeStrConsts:  unsafe.Offsetof(info.calleeStringConstants),
		CalleeBoolConsts: unsafe.Offsetof(info.calleeBoolConstants),
		CalleeNumGeneral: unsafe.Offsetof(info.calleeGeneralCount),
		NumGeneralArgs:   unsafe.Offsetof(info.generalArgumentCount),
		GeneralArgSrcs:   unsafe.Offsetof(info.generalArgumentSources),

		CalleeNumSliceByte: unsafe.Offsetof(info.calleeSliceByteCount),
		NumSliceByteArgs:   unsafe.Offsetof(info.sliceByteArgumentCount),
		SliceByteArgSrcs:   unsafe.Offsetof(info.sliceByteArgumentSources),

		SizeShift: asmCallInfoSizeShift(unsafe.Sizeof(info)),
	}
}

// ProvideVarLocationOffsets returns the field offsets of the runtime varLocation struct
// used by the return handler. Values are computed via unsafe.Offsetof.
//
// Returns asm.VarLocationOffsets populated from the current varLocation struct.
func ProvideVarLocationOffsets() asm.VarLocationOffsets {
	var loc varLocation
	return asm.VarLocationOffsets{
		UpvalueIndex: unsafe.Offsetof(loc.upvalueIndex),
		Register:     unsafe.Offsetof(loc.register),
		Kind:         unsafe.Offsetof(loc.kind),
		IsUpvalue:    unsafe.Offsetof(loc.isUpvalue),
	}
}

// asmCallInfoSizeShift returns the bit shift such that (1 << shift) == size. Panics if
// size is not a power of two; asmCallInfo is required to be a power-of-two size so the
// inline call dispatcher can index the table with a shift instead of a multiply.
//
// Takes size (uintptr) which is the byte size of the asmCallInfo struct.
//
// Returns int which is the corresponding bit shift.
func asmCallInfoSizeShift(size uintptr) int {
	if size == 0 || size&(size-1) != 0 {
		panic("asmCallInfo size is not a power of two; adjust _padding")
	}
	shift := 0
	for size > 1 {
		size >>= 1
		shift++
	}
	return shift
}
