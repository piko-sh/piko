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
	"fmt"
	"strings"

	"piko.sh/asmgen"
)

// DispatchContextOffsets carries DispatchContext field offsets baked into the generated
// asm_dispatch_offsets.h.
//
// The offsets are supplied by the caller (typically via unsafe.Offsetof on the live Go
// struct in package interp_domain) so the header is always derived from the actual layout
// rather than hardcoded literals that can silently desync.
type DispatchContextOffsets struct {
	// CodeBase is the byte offset of the codeBase field (pointer to the first instruction in
	// the active body).
	CodeBase uintptr

	// CodeLength is the byte offset of the codeLength field (number of instructions in the
	// body).
	CodeLength uintptr

	// ProgramCounter is the byte offset of the programCounter field (current instruction
	// index).
	ProgramCounter uintptr

	// IntsBase is the byte offset of the intsBase field (pointer to the int64 register
	// bank).
	IntsBase uintptr

	// IntsLength is the byte offset of the intsLength field (number of int64 registers
	// allocated).
	IntsLength uintptr

	// FloatsBase is the byte offset of the floatsBase field (pointer to the float64 register
	// bank).
	FloatsBase uintptr

	// FloatsLength is the byte offset of the floatsLength field (number of float64 registers
	// allocated).
	FloatsLength uintptr

	// IntConstantsBase is the byte offset of the intConstantsBase field (pointer to the
	// int64 constant pool).
	IntConstantsBase uintptr

	// IntConstantsLength is the byte offset of the intConstantsLength field.
	IntConstantsLength uintptr

	// FloatConstantsBase is the byte offset of the floatConstantsBase field (pointer to the
	// float64 constant pool).
	FloatConstantsBase uintptr

	// FloatConstantsLength is the byte offset of the floatConstantsLength field.
	FloatConstantsLength uintptr

	// JumpTable is the byte offset of the jumpTable field (pointer to the flat dispatch
	// table).
	JumpTable uintptr

	// ExitReason is the byte offset of the exitReason field, written by ASM before returning
	// to Go to indicate why dispatch exited.
	ExitReason uintptr

	// ExitProgramCounter is the byte offset of the exitProgramCounter field, holding the PC
	// at which dispatch exited.
	ExitProgramCounter uintptr

	// AsmCallInfoBase is the byte offset of the asmCallInfoBase field (active function's
	// asmCallInfo table base pointer).
	AsmCallInfoBase uintptr

	// CallStackBase is the byte offset of the callStackBase field (pointer to
	// vm.callStack[0]).
	CallStackBase uintptr

	// CallStackLength is the byte offset of the callStackLength field.
	CallStackLength uintptr

	// FramePointer is the byte offset of the framePointer field (current frame index within
	// the call stack).
	FramePointer uintptr

	// BaseFramePointer is the byte offset of the baseFramePointer field (frame index of the
	// outermost dispatched frame).
	BaseFramePointer uintptr

	// CallDepthLimit is the byte offset of the callDepthLimit field.
	CallDepthLimit uintptr

	// ArenaIntSlab is the byte offset of the arenaIntSlab field (pointer to the int register
	// arena slab base).
	ArenaIntSlab uintptr

	// ArenaIntCapacity is the byte offset of the arenaIntCapacity field (total slot count of
	// the int slab).
	ArenaIntCapacity uintptr

	// ArenaIntIndex is the byte offset of the arenaIntIndex field (current bump position;
	// read-write by ASM).
	ArenaIntIndex uintptr

	// ArenaFloatSlab is the byte offset of the arenaFloatSlab field (pointer to the float
	// register arena slab base).
	ArenaFloatSlab uintptr

	// ArenaFloatCapacity is the byte offset of the arenaFloatCapacity field.
	ArenaFloatCapacity uintptr

	// ArenaFloatIndex is the byte offset of the arenaFloatIndex field (current bump
	// position; read-write by ASM).
	ArenaFloatIndex uintptr

	// ArenaStringIndex is the byte offset of the arenaStringIndex field (current bump
	// position; read-write by ASM).
	ArenaStringIndex uintptr

	// ArenaGeneralIndex is the byte offset of the arenaGeneralIndex field (read-only by
	// ASM).
	ArenaGeneralIndex uintptr

	// ArenaBoolIndex is the byte offset of the arenaBoolIndex field (read-write by ASM).
	ArenaBoolIndex uintptr

	// ArenaUintIndex is the byte offset of the arenaUintIndex field (read-write by ASM).
	ArenaUintIndex uintptr

	// ArenaComplexIndex is the byte offset of the arenaComplexIndex field (read-only by
	// ASM).
	ArenaComplexIndex uintptr

	// DeferStackLength is the byte offset of the deferStackLength field (vm.deferStack entry
	// count).
	DeferStackLength uintptr

	// AsmCallInfoBasesPtr is the byte offset of the asmCallInfoBasesPointer field (pointer
	// to the asmCallInfoBases slice base).
	AsmCallInfoBasesPtr uintptr

	// DispatchSavesPtr is the byte offset of the dispatchSavesPointer field (pointer to the
	// asmDispatchSaves slice base).
	DispatchSavesPtr uintptr

	// StringsBase is the byte offset of the stringsBase field (pointer to the string
	// register bank, 16-byte headers).
	StringsBase uintptr

	// UintsBase is the byte offset of the uintsBase field (pointer to the uint64 register
	// bank).
	UintsBase uintptr

	// BoolsBase is the byte offset of the boolsBase field (pointer to the bool register
	// bank).
	BoolsBase uintptr

	// ArenaStringSlab is the byte offset of the arenaStringSlab field (pointer to the string
	// register arena slab base).
	ArenaStringSlab uintptr

	// ArenaStringCapacity is the byte offset of the arenaStringCapacity field.
	ArenaStringCapacity uintptr

	// ArenaBoolSlab is the byte offset of the arenaBoolSlab field (pointer to the bool
	// register arena slab base).
	ArenaBoolSlab uintptr

	// ArenaBoolCapacity is the byte offset of the arenaBoolCapacity field.
	ArenaBoolCapacity uintptr

	// ArenaUintSlab is the byte offset of the arenaUintSlab field (pointer to the uint
	// register arena slab base).
	ArenaUintSlab uintptr

	// ArenaUintCapacity is the byte offset of the arenaUintCapacity field.
	ArenaUintCapacity uintptr

	// SlicesIntBase is the byte offset of the slicesIntBase field (pointer to the first
	// []int64 slice header in the registers.slicesInt bank; each slot is 24 bytes).
	SlicesIntBase uintptr

	// SlicesFloatBase is the byte offset of the slicesFloatBase field (pointer to the first
	// []float64 slice header).
	SlicesFloatBase uintptr

	// SlicesStringBase is the byte offset of the slicesStringBase field (pointer to the
	// first []string slice header).
	SlicesStringBase uintptr

	// SlicesBoolBase is the byte offset of the slicesBoolBase field (pointer to the first
	// []bool slice header).
	SlicesBoolBase uintptr

	// SlicesUintBase is the byte offset of the slicesUintBase field (pointer to the first
	// []uint64 slice header).
	SlicesUintBase uintptr

	// ComplexBase is the byte offset of the complexBase field (pointer to the first
	// complex128 element; 16 bytes per slot).
	ComplexBase uintptr

	// StringConstantsBase is the byte offset of the stringConstantsBase field (pointer to
	// the string constant pool; 16-byte headers).
	StringConstantsBase uintptr

	// StringConstantsLength is the byte offset of the stringConstantsLength field.
	StringConstantsLength uintptr

	// BoolConstantsBase is the byte offset of the boolConstantsBase field (pointer to the
	// bool constant pool; 1-byte entries).
	BoolConstantsBase uintptr

	// BoolConstantsLength is the byte offset of the boolConstantsLength field.
	BoolConstantsLength uintptr

	// SavedPC is the byte offset of the spill slot used by tier-1 inline-call shims to stash
	// R14 (piko PC) across a Go-trampoline CALL.
	//
	// Spilled to ctx rather than the handler's local frame so the frame can declare
	// NO_LOCAL_POINTERS truthfully. See the asmgen emitters EmitInlineGoCallTwoOperandShim
	// and EmitInlineGoCallThreeOperandShim for the spill discipline.
	SavedPC uintptr

	// Tier2Result is the byte offset of the single-byte opResult slot that tier-2 ASM-call
	// shims read after CALLing their Go trampoline.
	//
	// "Tier-2" here names the handler tier (a Go-side handler that returns opContinue /
	// opFrameChanged / etc.), not the operand-arity tier; many shimmed handlers are actually
	// tier-0-encoded in the dispatch table. Emitted as CTX_TIER2_RESULT. See
	// asmgen_arch_*.EmitTier2CallShim's branch-on-opContinue.
	Tier2Result uintptr

	// StructLayoutTableBase is the byte offset of the structLayoutTableBase field (active
	// function's structLayoutTable base pointer).
	//
	// Read by tier-1 struct-field sub-ops (e.g. handlerSubOpStructFieldIntT0) to compute
	// &layoutTable[opC] for the field offset + Kind. Refreshed on every frame change in
	// rebuildDispatchPointers.
	StructLayoutTableBase uintptr

	// StructLayoutTableLength is the byte offset of the structLayoutTableLength field;
	// informational only at runtime.
	StructLayoutTableLength uintptr

	// TypeTableBase is the byte offset of the typeTableBase field (active function's
	// typeTable base pointer).
	//
	// Used by typed-nil and type-driven sub-ops. Refreshed on every frame change in
	// rebuildDispatchPointers.
	TypeTableBase uintptr

	// TypeTableLength is the byte offset of the typeTableLength field; informational only at
	// runtime.
	TypeTableLength uintptr

	// SlicesByteBase is the byte offset of the slicesByteBase field (pointer to
	// registers.slicesByte[0]).
	//
	// Used by tier-1 byte-slice sub-ops for get/set/slice/len/range and the make-byte
	// trampoline.
	SlicesByteBase uintptr

	// ArenaSliceByteSlab is the byte offset of the arenaSliceByteSlab field (pointer to the
	// typed []byte arena slab base).
	//
	// Read by the inline-call allocator to bump-allocate the callee's slicesByte register
	// bank from the dispatch context.
	ArenaSliceByteSlab uintptr

	// ArenaSliceByteCapacity is the byte offset of the arenaSliceByteCapacity field (total
	// slot count of the typed []byte slab).
	ArenaSliceByteCapacity uintptr

	// ArenaSliceByteIndex is the byte offset of the arenaSliceByteIndex field (current bump
	// position of the typed []byte slab; read-write by ASM).
	ArenaSliceByteIndex uintptr

	// ArenaSliceIntIndex is the byte offset of the slicesInt bump-index mirror field.
	//
	// Read by the inline-call frame-push shim so the new frame's arenaSave block carries the
	// correct save-point values for every typed-slice bank (not just slicesByte). ASM does
	// not bump these indices itself; they exist solely so the frame-push shim's arenaSave
	// write covers all 13 ArenaSavePoint slots.
	ArenaSliceIntIndex uintptr

	// ArenaSliceFloatIndex is the byte offset of the slicesFloat bump-index mirror field.
	ArenaSliceFloatIndex uintptr

	// ArenaSliceStringIndex is the byte offset of the slicesString bump-index mirror field.
	ArenaSliceStringIndex uintptr

	// ArenaSliceBoolIndex is the byte offset of the slicesBool bump-index mirror field.
	ArenaSliceBoolIndex uintptr

	// ArenaSliceUintIndex is the byte offset of the slicesUint bump-index mirror field.
	ArenaSliceUintIndex uintptr
}

// CallFrameOffsets carries the size of the runtime callFrame struct and byte offsets of
// every callFrame field referenced by the inline call and return ASM handlers.
//
// Values are supplied by the caller so the emitted header reflects the live struct
// layout; the field order in callFrame itself is pinned (TestCallFrameOffsets verifies
// it).
type CallFrameOffsets struct {
	// Size is the total byte size of the callFrame struct (CALLFRAME_SIZE in the generated
	// header).
	Size uintptr

	// RegsIntsPtr is the byte offset of the registers.ints slice pointer within the frame.
	RegsIntsPtr uintptr

	// RegsIntsLen is the byte offset of the registers.ints slice length.
	RegsIntsLen uintptr

	// RegsIntsCap is the byte offset of the registers.ints slice capacity.
	RegsIntsCap uintptr

	// RegsFloatsPtr is the byte offset of the registers.floats slice pointer.
	RegsFloatsPtr uintptr

	// RegsFloatsLen is the byte offset of the registers.floats slice length.
	RegsFloatsLen uintptr

	// RegsFloatsCap is the byte offset of the registers.floats slice capacity.
	RegsFloatsCap uintptr

	// RegsStringsPtr is the byte offset of the registers.strings slice pointer.
	RegsStringsPtr uintptr

	// RegsStringsLen is the byte offset of the registers.strings slice length.
	RegsStringsLen uintptr

	// RegsStringsCap is the byte offset of the registers.strings slice capacity.
	RegsStringsCap uintptr

	// RegsGeneralPtr is the byte offset of the registers.general slice pointer. Not
	// allocated on the inline-call fast path so the ASM only zeros these fields; included as
	// macros so the callFrame struct stays free to reorder.
	RegsGeneralPtr uintptr

	// RegsGeneralLen is the byte offset of the registers.general slice length.
	RegsGeneralLen uintptr

	// RegsGeneralCap is the byte offset of the registers.general slice capacity.
	RegsGeneralCap uintptr

	// RegsBoolsPtr is the byte offset of the registers.bools slice pointer.
	RegsBoolsPtr uintptr

	// RegsBoolsLen is the byte offset of the registers.bools slice length.
	RegsBoolsLen uintptr

	// RegsBoolsCap is the byte offset of the registers.bools slice capacity.
	RegsBoolsCap uintptr

	// RegsUintsPtr is the byte offset of the registers.uints slice pointer.
	RegsUintsPtr uintptr

	// RegsUintsLen is the byte offset of the registers.uints slice length.
	RegsUintsLen uintptr

	// RegsUintsCap is the byte offset of the registers.uints slice capacity.
	RegsUintsCap uintptr

	// RegsComplexPtr is the byte offset of the registers.complex slice pointer.
	RegsComplexPtr uintptr

	// RegsComplexLen is the byte offset of the registers.complex slice length.
	RegsComplexLen uintptr

	// RegsComplexCap is the byte offset of the registers.complex slice capacity.
	RegsComplexCap uintptr

	// RegsSlicesIntPtr is the byte offset of the registers.slicesInt slice header pointer.
	RegsSlicesIntPtr uintptr

	// RegsSlicesIntLen is the byte offset of the registers.slicesInt slice length.
	RegsSlicesIntLen uintptr

	// RegsSlicesIntCap is the byte offset of the registers.slicesInt slice capacity.
	RegsSlicesIntCap uintptr

	// RegsSlicesFloatPtr is the byte offset of the registers.slicesFloat slice header
	// pointer.
	RegsSlicesFloatPtr uintptr

	// RegsSlicesFloatLen is the byte offset of the registers.slicesFloat slice length.
	RegsSlicesFloatLen uintptr

	// RegsSlicesFloatCap is the byte offset of the registers.slicesFloat slice capacity.
	RegsSlicesFloatCap uintptr

	// RegsSlicesStringPtr is the byte offset of the registers.slicesString slice header
	// pointer.
	RegsSlicesStringPtr uintptr

	// RegsSlicesStringLen is the byte offset of the registers.slicesString slice length.
	RegsSlicesStringLen uintptr

	// RegsSlicesStringCap is the byte offset of the registers.slicesString slice capacity.
	RegsSlicesStringCap uintptr

	// RegsSlicesBoolPtr is the byte offset of the registers.slicesBool slice header pointer.
	RegsSlicesBoolPtr uintptr

	// RegsSlicesBoolLen is the byte offset of the registers.slicesBool slice length.
	RegsSlicesBoolLen uintptr

	// RegsSlicesBoolCap is the byte offset of the registers.slicesBool slice capacity.
	RegsSlicesBoolCap uintptr

	// RegsSlicesUintPtr is the byte offset of the registers.slicesUint slice header pointer.
	RegsSlicesUintPtr uintptr

	// RegsSlicesUintLen is the byte offset of the registers.slicesUint slice length.
	RegsSlicesUintLen uintptr

	// RegsSlicesUintCap is the byte offset of the registers.slicesUint slice capacity.
	RegsSlicesUintCap uintptr

	// RegsSliceBytePtr is the byte offset of the registers.slicesByte slice header pointer.
	//
	// The inline-call allocator allocates the callee's slicesByte bank directly from the
	// typed-byte arena slab without crossing into Go; RegsSliceByteLen and RegsSliceByteCap
	// accompany this to address the three slice-header words.
	RegsSliceBytePtr uintptr

	// RegsSliceByteLen is the byte offset of the registers.slicesByte slice length.
	RegsSliceByteLen uintptr

	// RegsSliceByteCap is the byte offset of the registers.slicesByte slice capacity.
	RegsSliceByteCap uintptr

	// Function is the byte offset of the function field (pointer to the compiled function
	// being executed).
	Function uintptr

	// SharedCells is the byte offset of the sharedCells field (closure-cell dedup map;
	// lazily allocated).
	SharedCells uintptr

	// Upvalues is the byte offset of the upvalues slice header.
	Upvalues uintptr

	// ReturnDestPtr is the byte offset of the returnDestination slice pointer.
	ReturnDestPtr uintptr

	// ReturnDestLen is the byte offset of the returnDestination slice length.
	ReturnDestLen uintptr

	// ReturnDestCap is the byte offset of the returnDestination slice capacity.
	ReturnDestCap uintptr

	// ProgramCounter is the byte offset of the programCounter field (the per-frame PC,
	// distinct from the dispatch context's PC).
	ProgramCounter uintptr

	// DeferBase is the byte offset of the deferBase field (the vm.deferStack index at which
	// this frame's defers start).
	DeferBase uintptr

	// ArenaSave is the byte offset of the arenaSave field (snapshot of arena state taken at
	// frame push and restored on pop).
	ArenaSave uintptr

	// HasGeneralAlloc is the byte offset of the hasGeneralAlloc byte (non-zero when this
	// frame allocated general-bank slots, telling the return handler whether to run the
	// clear+restore trampoline).
	HasGeneralAlloc uintptr
}

// ASMCallInfoOffsets carries asmCallInfo field offsets walked by the inline call
// dispatcher.
//
// Values are supplied by the caller for the same drift-resistance reason as the other
// offset structs.
type ASMCallInfoOffsets struct {
	// CalleeFunction is the byte offset of the callee's CompiledFunction pointer.
	CalleeFunction uintptr

	// CalleeBody is the byte offset of the callee body pointer (first instruction).
	CalleeBody uintptr

	// CalleeBodyLen is the byte offset of the callee body instruction count.
	CalleeBodyLen uintptr

	// CalleeIntConsts is the byte offset of the callee's int constant table pointer.
	CalleeIntConsts uintptr

	// CalleeFltConsts is the byte offset of the callee's float constant table pointer.
	CalleeFltConsts uintptr

	// CalleeNumInts is the byte offset of the callee's int register count.
	CalleeNumInts uintptr

	// CalleeNumFloats is the byte offset of the callee's float register count.
	CalleeNumFloats uintptr

	// NumIntArgs is the byte offset of the int argument count.
	NumIntArgs uintptr

	// IntArgSrcs is the byte offset of the int argument source array (caller register
	// indices, 8 slots).
	IntArgSrcs uintptr

	// NumFloatArgs is the byte offset of the float argument count.
	NumFloatArgs uintptr

	// FloatArgSrcs is the byte offset of the float argument source array (caller register
	// indices, 8 slots).
	FloatArgSrcs uintptr

	// NumReturns is the byte offset of the return value count (0 or 1).
	NumReturns uintptr

	// RetDestKind is the byte offset of the return-destination register kind.
	RetDestKind uintptr

	// RetDestReg is the byte offset of the return-destination register index in the caller.
	RetDestReg uintptr

	// RetDestPtr is the byte offset of the return-destination descriptor pointer.
	RetDestPtr uintptr

	// RetDestLen is the byte offset of the return-destination descriptor length.
	RetDestLen uintptr

	// CalleeCallInfo is the byte offset of the callee's asmCallInfo table base pointer.
	CalleeCallInfo uintptr

	// IsFastPath is the byte offset of the inline-dispatch mode tag: 0 not eligible, 1
	// eligible (string/bool/uint), 2 lean (int/float only), 3 Phase B-lite general-bank via
	// trampoline.
	IsFastPath uintptr

	// CalleeNumStrings is the byte offset of the callee's string register count.
	CalleeNumStrings uintptr

	// CalleeNumBools is the byte offset of the callee's bool register count.
	CalleeNumBools uintptr

	// CalleeNumUints is the byte offset of the callee's uint register count.
	CalleeNumUints uintptr

	// NumStringArgs is the byte offset of the string argument count.
	NumStringArgs uintptr

	// StringArgSrcs is the byte offset of the string argument source array (8 slots).
	StringArgSrcs uintptr

	// NumBoolArgs is the byte offset of the bool argument count.
	NumBoolArgs uintptr

	// BoolArgSrcs is the byte offset of the bool argument source array (8 slots).
	BoolArgSrcs uintptr

	// NumUintArgs is the byte offset of the uint argument count.
	NumUintArgs uintptr

	// UintArgSrcs is the byte offset of the uint argument source array (8 slots).
	UintArgSrcs uintptr

	// CalleeStrConsts is the byte offset of the callee's string constant table pointer
	// (16-byte headers).
	CalleeStrConsts uintptr

	// CalleeBoolConsts is the byte offset of the callee's bool constant table pointer
	// (1-byte entries).
	CalleeBoolConsts uintptr

	// CalleeNumGeneral is the byte offset of the callee's general (reflect.Value) register
	// count.
	CalleeNumGeneral uintptr

	// NumGeneralArgs is the byte offset of the general argument count.
	NumGeneralArgs uintptr

	// GeneralArgSrcs is the byte offset of the general argument source array (8 slots).
	GeneralArgSrcs uintptr

	// CalleeNumSliceByte is the byte offset of the callee's slicesByte register count.
	//
	// The inline-call allocator and copy block use this to size and bind the callee's
	// slicesByte bank from the dispatch context's typed-byte arena slab; companions are
	// NumSliceByteArgs and SliceByteArgSrcs.
	CalleeNumSliceByte uintptr

	// NumSliceByteArgs is the byte offset of the []byte argument count.
	NumSliceByteArgs uintptr

	// SliceByteArgSrcs is the byte offset of the []byte argument source array (8 slots).
	SliceByteArgSrcs uintptr

	// SizeShift is the log2 of sizeof(asmCallInfo) so the dispatcher can index per-site
	// entries as (siteIndex << SizeShift).
	SizeShift int
}

// VarLocationOffsets carries varLocation field offsets read by the return handler to
// materialise the caller's destination register kind.
type VarLocationOffsets struct {
	// UpvalueIndex is the byte offset of the upvalueIndex field.
	UpvalueIndex uintptr

	// Register is the byte offset of the register index field.
	Register uintptr

	// Kind is the byte offset of the register-bank kind field.
	Kind uintptr

	// IsUpvalue is the byte offset of the isUpvalue flag.
	IsUpvalue uintptr
}

const (
	// callFrameSizeComment documents the current CALLFRAME_SIZE that the generator emits
	// into the dispatch header. Kept in the generator source rather than as an asm-level
	// comment so it stays beside the struct it describes.
	callFrameSizeComment = `// callFrame size in bytes (verified by TestCallFrameOffsets).
//
// The 488-byte layout holds the scalar register banks plus six
// typed-slice header banks (Int, Float, String, Bool, Uint, Byte).
// Each typed-slice bank contributes a 24-byte slice header and an
// 8-byte arena save-point index.
`
)

// offsetDefineEntry is a single #define name/value pair.
type offsetDefineEntry struct {
	// name is the macro identifier emitted after "#define ".
	name string

	// value is the integer value the macro expands to.
	value uintptr
}

// offsetDefineList is an ordered slice of offset #define entries used to format aligned
// #define columns in the generated header.
type offsetDefineList []offsetDefineEntry

// add appends a #define entry to the list.
//
// Takes name (string) which is the macro identifier.
// Takes value (uintptr) which is the integer expansion.
func (list *offsetDefineList) add(name string, value uintptr) {
	*list = append(*list, offsetDefineEntry{name: name, value: value})
}

// format returns the #define block as a string with names padded to the longest name in
// the list so the value columns align.
//
// Returns the formatted #define block.
func (list offsetDefineList) format() string {
	width := 0
	for _, entry := range list {
		if len(entry.name) > width {
			width = len(entry.name)
		}
	}
	var builder strings.Builder
	for _, entry := range list {
		builder.WriteString(formatDefineLine(entry.name, entry.value, width))
		builder.WriteString("\n")
	}
	return builder.String()
}

// formatExcept returns the #define block excluding any entry whose name matches
// excludeName.
//
// Used by emitOffsetsCallFrame to keep the CALLFRAME_SIZE define above the field-offset
// block while reusing the same alignment routine.
//
// Takes excludeName (string) which names the entry to skip.
//
// Returns the formatted #define block with the named entry omitted.
func (list offsetDefineList) formatExcept(excludeName string) string {
	width := 0
	for _, entry := range list {
		if entry.name == excludeName {
			continue
		}
		if len(entry.name) > width {
			width = len(entry.name)
		}
	}
	var builder strings.Builder
	for _, entry := range list {
		if entry.name == excludeName {
			continue
		}
		builder.WriteString(formatDefineLine(entry.name, entry.value, width))
		builder.WriteString("\n")
	}
	return builder.String()
}

// lineFor returns a single formatted #define line for the named entry.
//
// The name is padded only to its own length so the standalone line stays outside any
// aligned column block.
//
// Takes name (string) which identifies the entry to format.
//
// Returns the formatted "#define name value" line, or "" when no entry matches.
func (list offsetDefineList) lineFor(name string) string {
	for _, entry := range list {
		if entry.name == name {
			return formatDefineLine(entry.name, entry.value, len(entry.name))
		}
	}
	return ""
}

// HeaderFiles returns the HeaderFile definitions for the interp_domain dispatch headers.
//
// The offsets are supplied by the caller so the emitted asm_dispatch_offsets.h is always
// derived from the current Go struct layout rather than hardcoded literals.
//
// Takes contextOffsets (*DispatchContextOffsets) which carries the DispatchContext field
// offsets.
// Takes frameOffsets (CallFrameOffsets) which carries callFrame size and field offsets.
// Takes callInfoOffsets (*ASMCallInfoOffsets) which carries asmCallInfo field offsets.
// Takes varLocationOffsets (VarLocationOffsets) which carries varLocation field offsets.
//
// Returns []asmgen.HeaderFile holding the offsets header and the per-architecture
// dispatch macro headers.
func HeaderFiles(
	contextOffsets *DispatchContextOffsets,
	frameOffsets *CallFrameOffsets,
	callInfoOffsets *ASMCallInfoOffsets,
	varLocationOffsets VarLocationOffsets,
) []asmgen.HeaderFile {
	return []asmgen.HeaderFile{
		{
			Name: "asm_dispatch_offsets.h",
			Dir:  dispatchOutputDir,
			Emit: func(_ []asmgen.ArchitecturePort) string {
				return emitDispatchOffsetsHeader(
					contextOffsets,
					frameOffsets,
					callInfoOffsets,
					varLocationOffsets,
				)
			},
		},
		{
			Name: "asm_dispatch_amd64.h",
			Dir:  dispatchOutputDir,
			Emit: func(archs []asmgen.ArchitecturePort) string {
				return dispatchMacrosForArchitecture(archs, asmgen.ArchitectureAMD64)
			},
		},
		{
			Name: "asm_dispatch_arm64.h",
			Dir:  dispatchOutputDir,
			Emit: func(archs []asmgen.ArchitecturePort) string {
				return dispatchMacrosForArchitecture(archs, asmgen.ArchitectureARM64)
			},
		},
	}
}

// dispatchMacrosForArchitecture finds the BytecodeArchitecturePort matching the given
// architecture and returns its dispatch macros. If no matching architecture is found, it
// returns an empty string.
//
// Takes archs ([]asmgen.ArchitecturePort) which is the set of registered architecture
// adapters.
// Takes target (asmgen.Architecture) which identifies the desired architecture.
//
// Returns string which is the dispatch macro text for the target architecture.
func dispatchMacrosForArchitecture(archs []asmgen.ArchitecturePort, target asmgen.Architecture) string {
	for _, arch := range archs {
		if arch.Arch() != target {
			continue
		}
		if bytecodeArch, ok := arch.(BytecodeArchitecturePort); ok {
			return bytecodeArch.DispatchMacros()
		}
	}
	return ""
}

// emitDispatchOffsetsHeader returns the full dispatch offsets header content.
//
// Concatenates the licence preamble, exit reasons, callFrame offsets, core and extended
// DispatchContext offsets, asmCallInfo offsets, and varLocation offsets in the order the
// .h file expects.
//
// Takes contextOffsets (*DispatchContextOffsets) which carries the DispatchContext field
// offsets.
// Takes frameOffsets (CallFrameOffsets) which carries callFrame size and field offsets.
// Takes callInfoOffsets (*ASMCallInfoOffsets) which carries asmCallInfo field offsets.
// Takes varLocationOffsets (VarLocationOffsets) which carries varLocation field offsets.
//
// Returns the complete header text.
func emitDispatchOffsetsHeader(
	contextOffsets *DispatchContextOffsets,
	frameOffsets *CallFrameOffsets,
	callInfoOffsets *ASMCallInfoOffsets,
	varLocationOffsets VarLocationOffsets,
) string {
	return emitOffsetsLicenceAndPreamble() +
		emitOffsetsExitReasons() +
		emitOffsetsCallFrame(frameOffsets) +
		emitOffsetsDispatchContextCore(contextOffsets) +
		emitOffsetsDispatchContextExtended(contextOffsets) +
		emitOffsetsASMCallInfo(callInfoOffsets) +
		emitOffsetsVarLocation(varLocationOffsets)
}

// emitOffsetsDispatchContextCore returns the #define block for the core DispatchContext
// field offsets.
//
// Covers codeBase, programCounter, the constant-pool pointers, jumpTable, and the exit
// reason/PC pair - the fields touched by DISPATCH_NEXT and the per-handler fast paths.
// Emitted as CTX_* defines so the underlying struct may be reordered without rewriting
// assembly.
//
// Takes offsets (*DispatchContextOffsets) which supplies the byte offsets to emit.
//
// Returns the core CTX_* #define block.
func emitOffsetsDispatchContextCore(offsets *DispatchContextOffsets) string {
	defines := offsetDefineList{}
	defines.add("CTX_CODE_BASE", offsets.CodeBase)
	defines.add("CTX_CODE_LEN", offsets.CodeLength)
	defines.add("CTX_PC", offsets.ProgramCounter)
	defines.add("CTX_INTS_BASE", offsets.IntsBase)
	defines.add("CTX_INTS_LEN", offsets.IntsLength)
	defines.add("CTX_FLOATS_BASE", offsets.FloatsBase)
	defines.add("CTX_FLOATS_LEN", offsets.FloatsLength)
	defines.add("CTX_INT_CONSTS_BASE", offsets.IntConstantsBase)
	defines.add("CTX_INT_CONSTS_LEN", offsets.IntConstantsLength)
	defines.add("CTX_FLT_CONSTS_BASE", offsets.FloatConstantsBase)
	defines.add("CTX_FLT_CONSTS_LEN", offsets.FloatConstantsLength)
	defines.add("CTX_JUMP_TABLE", offsets.JumpTable)
	defines.add("CTX_EXIT_REASON", offsets.ExitReason)
	defines.add("CTX_EXIT_PC", offsets.ExitProgramCounter)

	return "// DispatchContext core field offsets (referenced by DISPATCH_NEXT and\n" +
		"// the per-handler dispatch fast paths):\n" + defines.format()
}

// emitOffsetsLicenceAndPreamble returns the Apache 2.0 licence header, the
// DispatchContext field offset documentation, and the instruction encoding comment that
// appear at the top of the generated offsets header file.
//
// Returns string which is the licence and preamble text.
func emitOffsetsLicenceAndPreamble() string {
	return `// Code generated by cmd/asmgen; DO NOT EDIT.

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

// dispatch_offsets.h -- architecture-independent constants shared by both
// vm_dispatch_*_amd64.s and vm_dispatch_*_arm64.s.
//
// All offsets are verified at test time by vm_dispatch_offsets_test.go.

// DispatchContext field offsets (verified by TestDispatchContextOffsets):
//   codeBase      =   0    uintptr  pointer to Body[0]
//   codeLen       =   8    int64    number of instructions
//   pc            =  16    int64    current program counter
//   intsBase      =  24    uintptr  pointer to regs.Ints[0]
//   intsLen       =  32    int64    number of int registers
//   floatsBase    =  40    uintptr  pointer to regs.Floats[0]
//   floatsLen     =  48    int64    number of float registers
//   intConstsBase =  56    uintptr  pointer to IntConstants[0]
//   intConstsLen  =  64    int64    number of int constants
//   fltConstsBase =  72    uintptr  pointer to FloatConstants[0]
//   fltConstsLen  =  80    int64    number of float constants
//   jumpTable     =  88    uintptr  pointer to jump table[0]
//   exitReason    =  96    int64    exit reason code
//   exitPC        = 104    int64    PC at exit point

// Instruction encoding: {Op uint8, A uint8, B uint8, C uint8} = 4 bytes

`
}

// emitOffsetsExitReasons returns the #define block for exit reason constants used by the
// assembly dispatch loop to signal why control returned to Go.
//
// Returns string which is the exit reason #define block.
func emitOffsetsExitReasons() string {
	return `// Exit reason constants:
#define EXIT_END_OF_CODE    0
#define EXIT_TIER2          1
#define EXIT_DIV_BY_ZERO    2
#define EXIT_CALL           3
#define EXIT_RETURN         4
#define EXIT_RETURN_VOID    5
#define EXIT_TAIL_CALL      6
#define EXIT_CALL_OVERFLOW  7

`
}

// emitOffsetsCallFrame returns the #define block for callFrame size and field offsets
// used by the inline call and return handlers.
//
// CALLFRAME_SIZE is emitted ahead of the field block, with the layout history comment
// attached, so future struct growth has a natural place to extend the history.
//
// Takes offsets (CallFrameOffsets) which supplies the size and byte offsets to emit.
//
// Returns the callFrame CF_* #define block.
func emitOffsetsCallFrame(offsets *CallFrameOffsets) string {
	defines := offsetDefineList{
		{name: "CALLFRAME_SIZE", value: offsets.Size},
	}
	defines.add("CF_REGS_INTS_PTR", offsets.RegsIntsPtr)
	defines.add("CF_REGS_INTS_LEN", offsets.RegsIntsLen)
	defines.add("CF_REGS_INTS_CAP", offsets.RegsIntsCap)
	defines.add("CF_REGS_FLOATS_PTR", offsets.RegsFloatsPtr)
	defines.add("CF_REGS_FLOATS_LEN", offsets.RegsFloatsLen)
	defines.add("CF_REGS_FLOATS_CAP", offsets.RegsFloatsCap)
	defines.add("CF_REGS_STRINGS_PTR", offsets.RegsStringsPtr)
	defines.add("CF_REGS_STRINGS_LEN", offsets.RegsStringsLen)
	defines.add("CF_REGS_STRINGS_CAP", offsets.RegsStringsCap)
	defines.add("CF_REGS_GENERAL_PTR", offsets.RegsGeneralPtr)
	defines.add("CF_REGS_GENERAL_LEN", offsets.RegsGeneralLen)
	defines.add("CF_REGS_GENERAL_CAP", offsets.RegsGeneralCap)
	defines.add("CF_REGS_BOOLS_PTR", offsets.RegsBoolsPtr)
	defines.add("CF_REGS_BOOLS_LEN", offsets.RegsBoolsLen)
	defines.add("CF_REGS_BOOLS_CAP", offsets.RegsBoolsCap)
	defines.add("CF_REGS_UINTS_PTR", offsets.RegsUintsPtr)
	defines.add("CF_REGS_UINTS_LEN", offsets.RegsUintsLen)
	defines.add("CF_REGS_UINTS_CAP", offsets.RegsUintsCap)
	defines.add("CF_REGS_COMPLEX_PTR", offsets.RegsComplexPtr)
	defines.add("CF_REGS_COMPLEX_LEN", offsets.RegsComplexLen)
	defines.add("CF_REGS_COMPLEX_CAP", offsets.RegsComplexCap)
	defines.add("CF_REGS_SLICESINT_PTR", offsets.RegsSlicesIntPtr)
	defines.add("CF_REGS_SLICESINT_LEN", offsets.RegsSlicesIntLen)
	defines.add("CF_REGS_SLICESINT_CAP", offsets.RegsSlicesIntCap)
	defines.add("CF_REGS_SLICESFLOAT_PTR", offsets.RegsSlicesFloatPtr)
	defines.add("CF_REGS_SLICESFLOAT_LEN", offsets.RegsSlicesFloatLen)
	defines.add("CF_REGS_SLICESFLOAT_CAP", offsets.RegsSlicesFloatCap)
	defines.add("CF_REGS_SLICESSTRING_PTR", offsets.RegsSlicesStringPtr)
	defines.add("CF_REGS_SLICESSTRING_LEN", offsets.RegsSlicesStringLen)
	defines.add("CF_REGS_SLICESSTRING_CAP", offsets.RegsSlicesStringCap)
	defines.add("CF_REGS_SLICESBOOL_PTR", offsets.RegsSlicesBoolPtr)
	defines.add("CF_REGS_SLICESBOOL_LEN", offsets.RegsSlicesBoolLen)
	defines.add("CF_REGS_SLICESBOOL_CAP", offsets.RegsSlicesBoolCap)
	defines.add("CF_REGS_SLICESUINT_PTR", offsets.RegsSlicesUintPtr)
	defines.add("CF_REGS_SLICESUINT_LEN", offsets.RegsSlicesUintLen)
	defines.add("CF_REGS_SLICESUINT_CAP", offsets.RegsSlicesUintCap)
	defines.add("CF_REGS_SLICEBYTE_PTR", offsets.RegsSliceBytePtr)
	defines.add("CF_REGS_SLICEBYTE_LEN", offsets.RegsSliceByteLen)
	defines.add("CF_REGS_SLICEBYTE_CAP", offsets.RegsSliceByteCap)
	defines.add("CF_FUNCTION", offsets.Function)
	defines.add("CF_SHARED_CELLS", offsets.SharedCells)
	defines.add("CF_UPVALUES_PTR", offsets.Upvalues)
	defines.add("CF_RETURNDEST_PTR", offsets.ReturnDestPtr)
	defines.add("CF_RETURNDEST_LEN", offsets.ReturnDestLen)
	defines.add("CF_RETURNDEST_CAP", offsets.ReturnDestCap)
	defines.add("CF_PROGRAM_COUNTER", offsets.ProgramCounter)
	defines.add("CF_DEFERBASE", offsets.DeferBase)
	defines.add("CF_ARENA_SAVE", offsets.ArenaSave)
	defines.add("CF_HAS_GENERAL_ALLOC", offsets.HasGeneralAlloc)

	preamble := callFrameSizeComment + defines.lineFor("CALLFRAME_SIZE") + "\n" +
		"\n// callFrame field offsets:\n"
	return preamble + defines.formatExcept("CALLFRAME_SIZE") + "\n"
}

// emitOffsetsDispatchContextExtended returns the #define block for DispatchContext fields
// beyond the core dispatch set.
//
// Covers inline call state, arena indices, extended register bank bases, and the Phase
// B/E/J.3 sub-blocks (struct-layout/type tables, typed []byte register bank, typed []byte
// arena slab). The Phase preambles are emitted alongside their defines so the explanatory
// comments stay adjacent to the values they describe; appends the static
// structFieldLayout/reflect block at the end.
//
// Takes offsets (*DispatchContextOffsets) which supplies the byte offsets to emit.
//
// Returns the extended CTX_* #define block followed by the static layout/reflect defines.
func emitOffsetsDispatchContextExtended(offsets *DispatchContextOffsets) string {
	preamble := "// DispatchContext extended field offsets (inline call/return):\n"
	body := buildExtendedDispatchContextDefines(offsets).format()
	phaseB := emitPhaseBStructFieldDefines(offsets)
	phaseE := emitPhaseESliceByteBankDefines(offsets)
	phaseJ3 := emitPhaseJ3SliceByteArenaDefines(offsets)
	return preamble + body + phaseB + phaseE + phaseJ3 + "\n" + emitOffsetsStructFieldLayout()
}

// buildExtendedDispatchContextDefines assembles the main CTX_* define block emitted at
// the top of emitOffsetsDispatchContextExtended.
//
// Split out so the parent function stays under the function-length limit; the dense table
// is easier to scan when it lives in its own helper.
//
// Takes offsets (*DispatchContextOffsets) which supplies the byte offsets to emit.
//
// Returns the populated offsetDefineList holding the extended CTX_* entries.
func buildExtendedDispatchContextDefines(offsets *DispatchContextOffsets) offsetDefineList {
	defines := offsetDefineList{}
	defines.add("CTX_ASM_CALL_INFO_BASE", offsets.AsmCallInfoBase)
	defines.add("CTX_CSTACK_BASE", offsets.CallStackBase)
	defines.add("CTX_CSTACK_LEN", offsets.CallStackLength)
	defines.add("CTX_FRAME_POINTER", offsets.FramePointer)
	defines.add("CTX_BASE_FRAME_POINTER", offsets.BaseFramePointer)
	defines.add("CTX_DEPTH_LIMIT", offsets.CallDepthLimit)
	defines.add("CTX_ARENA_INT_SLAB", offsets.ArenaIntSlab)
	defines.add("CTX_ARENA_INT_CAP", offsets.ArenaIntCapacity)
	defines.add("CTX_ARENA_INT_IDX", offsets.ArenaIntIndex)
	defines.add("CTX_ARENA_FLT_SLAB", offsets.ArenaFloatSlab)
	defines.add("CTX_ARENA_FLT_CAP", offsets.ArenaFloatCapacity)
	defines.add("CTX_ARENA_FLT_IDX", offsets.ArenaFloatIndex)
	defines.add("CTX_ARENA_STR_IDX", offsets.ArenaStringIndex)
	defines.add("CTX_ARENA_GEN_IDX", offsets.ArenaGeneralIndex)
	defines.add("CTX_ARENA_BOOL_IDX", offsets.ArenaBoolIndex)
	defines.add("CTX_ARENA_UINT_IDX", offsets.ArenaUintIndex)
	defines.add("CTX_ARENA_CPLX_IDX", offsets.ArenaComplexIndex)
	defines.add("CTX_DEFER_STACK_LEN", offsets.DeferStackLength)
	defines.add("CTX_ASM_CI_PTRS", offsets.AsmCallInfoBasesPtr)
	defines.add("CTX_DISPATCH_SAVES", offsets.DispatchSavesPtr)
	defines.add("CTX_STRINGS_BASE", offsets.StringsBase)
	defines.add("CTX_UINTS_BASE", offsets.UintsBase)
	defines.add("CTX_BOOLS_BASE", offsets.BoolsBase)
	defines.add("CTX_ARENA_STR_SLAB", offsets.ArenaStringSlab)
	defines.add("CTX_ARENA_STR_CAP", offsets.ArenaStringCapacity)
	defines.add("CTX_ARENA_BOOL_SLAB", offsets.ArenaBoolSlab)
	defines.add("CTX_ARENA_BOOL_CAP", offsets.ArenaBoolCapacity)
	defines.add("CTX_ARENA_UINT_SLAB", offsets.ArenaUintSlab)
	defines.add("CTX_ARENA_UINT_CAP", offsets.ArenaUintCapacity)
	defines.add("CTX_SLICES_INT_BASE", offsets.SlicesIntBase)
	defines.add("CTX_SLICES_FLOAT_BASE", offsets.SlicesFloatBase)
	defines.add("CTX_SLICES_STRING_BASE", offsets.SlicesStringBase)
	defines.add("CTX_SLICES_BOOL_BASE", offsets.SlicesBoolBase)
	defines.add("CTX_SLICES_UINT_BASE", offsets.SlicesUintBase)
	defines.add("CTX_COMPLEX_BASE", offsets.ComplexBase)
	defines.add("CTX_STR_CONSTS_BASE", offsets.StringConstantsBase)
	defines.add("CTX_STR_CONSTS_LEN", offsets.StringConstantsLength)
	defines.add("CTX_BOOL_CONSTS_BASE", offsets.BoolConstantsBase)
	defines.add("CTX_BOOL_CONSTS_LEN", offsets.BoolConstantsLength)
	defines.add("CTX_SAVED_PC", offsets.SavedPC)
	defines.add("CTX_TIER2_RESULT", offsets.Tier2Result)
	return defines
}

// emitPhaseBStructFieldDefines emits the Phase B preamble + define block for struct-field
// ASM handler table bases.
//
// Kept adjacent to the comment so future readers see why these four defines are grouped
// together.
//
// Takes offsets (*DispatchContextOffsets) which supplies the byte offsets to emit.
//
// Returns the Phase B preamble plus its #define block.
func emitPhaseBStructFieldDefines(offsets *DispatchContextOffsets) string {
	preamble := "\n// Phase B: pinned table bases for struct-field ASM handlers.\n" +
		"// structLayoutTable holds compile-time-resolved {offset, kind, ...}\n" +
		"// per field-access site; ASM reads the entry at body[pc].c to\n" +
		"// compute the unsafe field pointer without a Go round-trip.\n" +
		"// typeTable holds reflect.Type slots used by tier-1 typed-nil and\n" +
		"// type-driven ops.\n"
	defines := offsetDefineList{}
	defines.add("CTX_STRUCT_LAYOUT_TABLE_BASE", offsets.StructLayoutTableBase)
	defines.add("CTX_STRUCT_LAYOUT_TABLE_LEN", offsets.StructLayoutTableLength)
	defines.add("CTX_TYPE_TABLE_BASE", offsets.TypeTableBase)
	defines.add("CTX_TYPE_TABLE_LEN", offsets.TypeTableLength)
	return preamble + defines.format()
}

// emitPhaseESliceByteBankDefines emits the Phase E preamble + define for the typed []byte
// register bank base.
//
// Takes offsets (*DispatchContextOffsets) which supplies the byte offset to emit.
//
// Returns the Phase E preamble plus its #define block.
func emitPhaseESliceByteBankDefines(offsets *DispatchContextOffsets) string {
	preamble := "\n// Phase E: typed []byte register bank base.\n" +
		"// Each slot is a 24-byte slice header (ptr+len+cap). Used by\n" +
		"// tier-1 byte-slice ASM sub-ops for chunk[i] / chunk[s:e] /\n" +
		"// make([]byte) / range over []byte without leaving the loop.\n"
	defines := offsetDefineList{}
	defines.add("CTX_SLICES_BYTE_BASE", offsets.SlicesByteBase)
	return preamble + defines.format()
}

// emitPhaseJ3SliceByteArenaDefines emits the Phase J.3 preamble + defines for the typed
// []byte arena slab used by the inline-call allocator.
//
// Takes offsets (*DispatchContextOffsets) which supplies the byte offsets to emit.
//
// Returns the Phase J.3 preamble plus its #define block.
func emitPhaseJ3SliceByteArenaDefines(offsets *DispatchContextOffsets) string {
	preamble := "\n// Phase J.3: typed []byte arena slab (24-byte slice headers).\n" +
		"// Read by the inline-call allocator to bump-allocate the callee's\n" +
		"// slicesByte register bank from the dispatch context.\n"
	defines := offsetDefineList{}
	defines.add("CTX_ARENA_SLICEBYTE_SLAB", offsets.ArenaSliceByteSlab)
	defines.add("CTX_ARENA_SLICEBYTE_CAP", offsets.ArenaSliceByteCapacity)
	defines.add("CTX_ARENA_SLICEBYTE_IDX", offsets.ArenaSliceByteIndex)
	defines.add("CTX_ARENA_SLICEINT_IDX", offsets.ArenaSliceIntIndex)
	defines.add("CTX_ARENA_SLICEFLT_IDX", offsets.ArenaSliceFloatIndex)
	defines.add("CTX_ARENA_SLICESTR_IDX", offsets.ArenaSliceStringIndex)
	defines.add("CTX_ARENA_SLICEBOOL_IDX", offsets.ArenaSliceBoolIndex)
	defines.add("CTX_ARENA_SLICEUINT_IDX", offsets.ArenaSliceUintIndex)
	return preamble + defines.format()
}

// emitOffsetsStructFieldLayout returns the static #define block covering the
// structFieldLayout struct's field offsets and the reflect.Kind / reflect.Value flag
// constants the Phase B ASM handlers compare against. These values are fixed (stable
// across Go releases for the foreseeable future) and pinned by
// TestStructFieldLayoutOffsets / TestReflectKindConstants / TestReflectValueFlagConstants
// in vm_dispatch_offsets_test.go.
//
// Returns string which is the static layout/reflect #define block.
func emitOffsetsStructFieldLayout() string {
	return `// structFieldLayout is 16 bytes. Verified by
// TestStructFieldLayoutSize. The shift exists so ASM can compute
// ` + "`&layoutTable[idx]`" + ` as ` + "`base + (idx << LAYOUT_SIZE_SHIFT)`" + `.
#define LAYOUT_SIZE_SHIFT 4

// structFieldLayout field offsets (verified by
// TestStructFieldLayoutOffsets). Only the fields ASM reads are
// exposed; the Path[] / PathLength fields are safe-build-only and
// not addressed from ASM.
//
//   Offset         uint32 at +0   field byte offset within deref'd struct
//   TypeIndex      uint16 at +4   typeTable index of struct type (panic msgs)
//   Path[4]        4×uint8 at +6  reflect.Field index walk (safe build)
//   PathLength     uint8  at +10
//   Kind           uint8  at +11  reflect.Kind of leaf field
//   RegisterKind   uint8  at +12  registerKind of leaf field
//   Flags          uint8  at +13
//   FieldTypeIndex uint16 at +14  typeTable index of leaf type (registerGeneral)
#define OFF_LAYOUT_OFFSET           0
#define OFF_LAYOUT_TYPE_INDEX       4
#define OFF_LAYOUT_KIND             11
#define OFF_LAYOUT_REGISTER_KIND    12
#define OFF_LAYOUT_FLAGS            13
#define OFF_LAYOUT_FIELD_TYPE_INDEX 14

// reflect.Kind enum values (from $GOROOT/src/reflect/type.go).
// Pinned by TestReflectKindConstants. Stable across Go 1.x; Phase B
// ASM handlers compare layout.Kind against these to dispatch the
// typed read/write.
#define REFLECT_INVALID 0
#define REFLECT_BOOL    1
#define REFLECT_INT     2
#define REFLECT_INT8    3
#define REFLECT_INT16   4
#define REFLECT_INT32   5
#define REFLECT_INT64   6
#define REFLECT_UINT    7
#define REFLECT_UINT8   8
#define REFLECT_UINT16  9
#define REFLECT_UINT32  10
#define REFLECT_UINT64  11
#define REFLECT_UINTPTR 12
#define REFLECT_FLOAT32 13
#define REFLECT_FLOAT64 14
#define REFLECT_INTERFACE 20
#define REFLECT_POINTER 22

// reflect.Value flag word bits (from $GOROOT/src/reflect/value.go,
// mirrored in interp_domain/reflect_value_unsafe.go). Pinned by
// TestReflectValueFlagConstants. Phase B ASM handlers extract the
// receiver's Kind via flag & FLAG_KIND_MASK and check the
// FLAG_INDIR bit to decide whether ptr is *T (storage) or T itself.
#define FLAG_KIND_MASK 0x1F
#define FLAG_INDIR     0x80
#define FLAG_ADDR      0x100

`
}

// emitOffsetsASMCallInfo returns the #define block for asmCallInfo field offsets plus the
// size-shift constant used by the inline call handler to index into the call-info array.
//
// Emits the literal sizeof comment line so the chosen shift is self-documenting alongside
// ACI_SIZE_SHIFT.
//
// Takes offsets (*ASMCallInfoOffsets) which supplies the byte offsets and the SizeShift.
//
// Returns the asmCallInfo ACI_* #define block plus ACI_SIZE_SHIFT.
func emitOffsetsASMCallInfo(offsets *ASMCallInfoOffsets) string {
	defines := offsetDefineList{}
	defines.add("ACI_CALLEE_FUNCTION", offsets.CalleeFunction)
	defines.add("ACI_CALLEE_BODY", offsets.CalleeBody)
	defines.add("ACI_CALLEE_BODY_LEN", offsets.CalleeBodyLen)
	defines.add("ACI_CALLEE_INT_CONSTS", offsets.CalleeIntConsts)
	defines.add("ACI_CALLEE_FLT_CONSTS", offsets.CalleeFltConsts)
	defines.add("ACI_CALLEE_NUM_INTS", offsets.CalleeNumInts)
	defines.add("ACI_CALLEE_NUM_FLOATS", offsets.CalleeNumFloats)
	defines.add("ACI_NUM_INT_ARGS", offsets.NumIntArgs)
	defines.add("ACI_INT_ARG_SRCS", offsets.IntArgSrcs)
	defines.add("ACI_NUM_FLOAT_ARGS", offsets.NumFloatArgs)
	defines.add("ACI_FLOAT_ARG_SRCS", offsets.FloatArgSrcs)
	defines.add("ACI_NUM_RETURNS", offsets.NumReturns)
	defines.add("ACI_RET_DEST_KIND", offsets.RetDestKind)
	defines.add("ACI_RET_DEST_REG", offsets.RetDestReg)
	defines.add("ACI_RET_DEST_PTR", offsets.RetDestPtr)
	defines.add("ACI_RET_DEST_LEN", offsets.RetDestLen)
	defines.add("ACI_CALLEE_CALL_INFO", offsets.CalleeCallInfo)
	defines.add("ACI_IS_FAST_PATH", offsets.IsFastPath)
	defines.add("ACI_CALLEE_NUM_STRINGS", offsets.CalleeNumStrings)
	defines.add("ACI_CALLEE_NUM_BOOLS", offsets.CalleeNumBools)
	defines.add("ACI_CALLEE_NUM_UINTS", offsets.CalleeNumUints)
	defines.add("ACI_NUM_STRING_ARGS", offsets.NumStringArgs)
	defines.add("ACI_STRING_ARG_SRCS", offsets.StringArgSrcs)
	defines.add("ACI_NUM_BOOL_ARGS", offsets.NumBoolArgs)
	defines.add("ACI_BOOL_ARG_SRCS", offsets.BoolArgSrcs)
	defines.add("ACI_NUM_UINT_ARGS", offsets.NumUintArgs)
	defines.add("ACI_UINT_ARG_SRCS", offsets.UintArgSrcs)
	defines.add("ACI_CALLEE_STR_CONSTS", offsets.CalleeStrConsts)
	defines.add("ACI_CALLEE_BOOL_CONSTS", offsets.CalleeBoolConsts)
	defines.add("ACI_CALLEE_NUM_GENERAL", offsets.CalleeNumGeneral)
	defines.add("ACI_NUM_GENERAL_ARGS", offsets.NumGeneralArgs)
	defines.add("ACI_GENERAL_ARG_SRCS", offsets.GeneralArgSrcs)
	defines.add("ACI_CALLEE_NUM_SLICEBYTE", offsets.CalleeNumSliceByte)
	defines.add("ACI_NUM_SLICEBYTE_ARGS", offsets.NumSliceByteArgs)
	defines.add("ACI_SLICEBYTE_ARG_SRCS", offsets.SliceByteArgSrcs)

	return "// asmCallInfo field offsets (verified by TestASMCallInfoOffsets):\n" +
		defines.format() + "\n" +
		fmt.Sprintf("// sizeof(asmCallInfo) = %d (power of 2, use shift).\n", 1<<offsets.SizeShift) +
		fmt.Sprintf("#define ACI_SIZE_SHIFT %d\n\n", offsets.SizeShift)
}

// emitOffsetsVarLocation returns the #define block for varLocation field offsets read by
// the return handler to recover the caller's destination register kind.
//
// Takes offsets (VarLocationOffsets) which supplies the byte offsets to emit.
//
// Returns the varLocation VL_* #define block.
func emitOffsetsVarLocation(offsets VarLocationOffsets) string {
	defines := offsetDefineList{}
	defines.add("VL_UPVALUE_INDEX", offsets.UpvalueIndex)
	defines.add("VL_REGISTER", offsets.Register)
	defines.add("VL_KIND", offsets.Kind)
	defines.add("VL_IS_UPVALUE", offsets.IsUpvalue)

	return "// varLocation field offsets:\n" + defines.format()
}

// formatDefineLine formats one "#define NAME value" line with the name padded so values
// within the same section align in a single column.
//
// The longest name in the section receives a single space before its value; shorter names
// are padded out to match.
//
// Takes name (string) which is the macro identifier.
// Takes value (uintptr) which is the integer expansion.
// Takes nameWidth (int) which is the longest name length in the section, used for column
// alignment.
//
// Returns the formatted "#define name value" line.
func formatDefineLine(name string, value uintptr, nameWidth int) string {
	padding := max(nameWidth-len(name), 0)
	return "#define " + name + strings.Repeat(" ", padding+1) + fmt.Sprintf("%d", value)
}
