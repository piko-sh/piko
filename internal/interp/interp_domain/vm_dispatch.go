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
)

const (
	// exitEndOfCode indicates the dispatch loop exited because the program counter reached
	// the end of the code.
	exitEndOfCode int64 = 0

	// exitTier2 indicates the dispatch loop exited because a tier 2 opcode requires Go-side
	// handling.
	exitTier2 int64 = 1

	// exitDivByZero indicates the dispatch loop exited due to a division by zero error.
	exitDivByZero int64 = 2

	// exitCall indicates the dispatch loop exited to perform a function call via Go.
	exitCall int64 = 3

	// exitReturn indicates the dispatch loop exited to perform a function return via Go.
	exitReturn int64 = 4

	// exitReturnVoid indicates the dispatch loop exited to perform a void function return
	// via Go.
	exitReturnVoid int64 = 5

	// exitTailCall indicates the dispatch loop exited to perform a tail call via Go.
	exitTailCall int64 = 6

	// exitCallOverflow indicates the dispatch loop exited because the call depth limit was
	// exceeded.
	exitCallOverflow int64 = 7

	// exitSetField is the per-op direct exit for opSetField.
	//
	// Routed by handlerSetFieldExit in asm_vm_dispatch_direct_exits_*.s, installed into
	// asmJumpTable[opSetField] at init. Go-side path is processExitSetField, which calls
	// handleSetField directly without going through handlerTable[op].
	exitSetField int64 = 8

	// exitGetField is the per-op direct exit for opGetField. Mirrors exitSetField; saves the
	// handlerTable[op] indirect call on the trampoline's first op.
	exitGetField int64 = 9

	// exitMapIndex is the per-op direct exit for opMapIndex. Mirrors exitSetField; saves the
	// handlerTable[op] indirect call on the trampoline's first op.
	exitMapIndex int64 = 10

	// exitAppend is the per-op direct exit for opAppend. Mirrors exitSetField; saves the
	// handlerTable[op] indirect call on the trampoline's first op.
	exitAppend int64 = 11

	// exitAppendByteFast is the tier-0 specialised []byte-append direct-exit reason emitted
	// by compileBuiltinAppend when the slice element is statically `byte`.
	exitAppendByteFast int64 = 12

	// exitGetStructFieldIntT0 is the direct-exit reason for the tier-0 int struct-field
	// reader. The matching processExit dispatcher calls the existing Go handler directly,
	// skipping the handlerTable[op] indirect.
	exitGetStructFieldIntT0 int64 = 13

	// exitGetStructFieldUintT0 is the direct-exit reason for the tier-0 uint struct-field
	// reader.
	exitGetStructFieldUintT0 int64 = 14

	// exitGetStructFieldFloatT0 is the direct-exit reason for the tier-0 float struct-field
	// reader.
	exitGetStructFieldFloatT0 int64 = 15

	// exitGetStructFieldBoolT0 is the direct-exit reason for the tier-0 bool struct-field
	// reader.
	exitGetStructFieldBoolT0 int64 = 16

	// exitSetStructFieldIntT0 is the direct-exit reason for the tier-0 int struct-field
	// writer. Primitive kind only; no GC barrier required (primitives carry no pointers).
	exitSetStructFieldIntT0 int64 = 17

	// exitSetStructFieldUintT0 is the direct-exit reason for the tier-0 uint struct-field
	// writer.
	exitSetStructFieldUintT0 int64 = 18

	// exitSetStructFieldFloatT0 is the direct-exit reason for the tier-0 float struct-field
	// writer.
	exitSetStructFieldFloatT0 int64 = 19

	// exitSetStructFieldBoolT0 is the direct-exit reason for the tier-0 bool struct-field
	// writer.
	exitSetStructFieldBoolT0 int64 = 20

	// exitFrameChanged is the tier-2 ASM-lift exit code for opFrameChanged.
	//
	// Written by the ASM-side tier-2 call shim when the Go handler returned opFrameChanged.
	// The shim CALLs the Go handler, checks the returned opResult byte, and on a non-zero
	// value writes the matching exitReason here and RETs to handleDispatchExit, routing
	// every shimmed tier-2 op's cold path through a single Go-side dispatch. The shim
	// epilogue maps opFrameChanged to exitFrameChanged (21), opDone to exitDoneTier2 (24),
	// opDivByZero to exitDivByZero (2, reusing the existing code), opStackOverflow to
	// exitStackOverflowTier2 (23), and opPanicError to exitPanicErrorTier2 (22).
	exitFrameChanged int64 = 21

	// exitPanicErrorTier2 is the tier-2 ASM-lift exit code for opPanicError. See
	// exitFrameChanged for the opResult-to-exitReason mapping table.
	exitPanicErrorTier2 int64 = 22

	// exitStackOverflowTier2 is the tier-2 ASM-lift exit code for opStackOverflow. See
	// exitFrameChanged for the opResult-to-exitReason mapping table.
	exitStackOverflowTier2 int64 = 23

	// exitDoneTier2 is the tier-2 ASM-lift exit code for opDone. See exitFrameChanged for
	// the opResult-to-exitReason mapping table.
	exitDoneTier2 int64 = 24

	// exitGetStructFieldGeneralT0 is the direct-exit reason for the tier-0 general-bank
	// (pointer/interface) struct-field reader. The matching
	// processExitGetStructFieldGeneralT0 calls handleGetStructFieldGeneralT0 directly,
	// skipping the handlerTable[op] indirect on the trampoline's first op.
	exitGetStructFieldGeneralT0 int64 = 25

	// exitSetStructFieldGeneralT0 is the direct-exit reason for the tier-0 general-bank
	// struct-field writer. The handler still needs runtime_typedmemmove for the
	// pointer/interface store (no GC-safe NOSPLIT|NOFRAME path); the direct exit only saves
	// the handlerTable[op] indirect on the trampoline's first op.
	exitSetStructFieldGeneralT0 int64 = 26

	// exitTestNilJumpFalse is the direct-exit reason for the tier-2 nil-test-and-jump op
	// (branch taken when the tested pointer is non-nil). Skipping the handlerTable[op]
	// indirect on the trampoline's first op pays off for tree- and list-style
	// pointer-walking workloads.
	exitTestNilJumpFalse int64 = 27

	// exitTestNilJumpTrue mirrors exitTestNilJumpFalse for the inverted-sense nil test.
	exitTestNilJumpTrue int64 = 28

	// maxASMIntArgs is the maximum number of int arguments the ASM fast-path call handler
	// can copy.
	maxASMIntArgs = 8

	// maxASMFloatArgs is the maximum number of float arguments the ASM fast-path call
	// handler can copy.
	maxASMFloatArgs = 8

	// maxASMStringArgs is the maximum number of string arguments the ASM fast-path call
	// handler can copy.
	maxASMStringArgs = 8

	// maxASMBoolArgs is the maximum number of bool arguments the ASM fast-path call handler
	// can copy.
	maxASMBoolArgs = 8

	// maxASMUintArgs is the maximum number of uint arguments the ASM fast-path call handler
	// can copy.
	maxASMUintArgs = 8

	// maxASMSliceByteArgs is the maximum number of []byte arguments the ASM fast-path call
	// handler can copy.
	maxASMSliceByteArgs = 8

	// maxASMGeneralArgs is the maximum number of general (reflect.Value) arguments the ASM
	// fast-path call handler can copy via the handlerCallInlineSetupGeneralBank trampoline.
	// Matches the 8-entry generalArgumentSources array in asmCallInfo.
	maxASMGeneralArgs = 8

	// fastPathIneligible marks a call site that cannot use the inline ASM dispatcher and
	// must trampoline to Go.
	fastPathIneligible int64 = 0

	// fastPathExtendedBanks marks a call site whose callee uses string/bool/uint registers;
	// the inline path allocates each bank.
	fastPathExtendedBanks int64 = 1

	// fastPathPureIntFloat marks a call site whose callee uses only int and float registers;
	// the inline path zeros extended bank headers and skips per-bank allocation (lean path).
	fastPathPureIntFloat int64 = 2

	// fastPathGeneralBank marks a call site whose callee uses the reflect.Value general
	// bank; the inline path allocates the standard banks and CALLs
	// handlerCallInlineSetupGeneralBank for the general bank.
	fastPathGeneralBank int64 = 3
)

var (
	// maxASMArgsByKind maps each register kind to its maximum ASM argument count.
	// registerComplex and the typed-slice banks other than registerSliceByte are unsupported
	// by the ASM fast path.
	maxASMArgsByKind = [NumRegisterKinds]int{
		registerInt:       maxASMIntArgs,
		registerFloat:     maxASMFloatArgs,
		registerString:    maxASMStringArgs,
		registerGeneral:   maxASMGeneralArgs,
		registerBool:      maxASMBoolArgs,
		registerUint:      maxASMUintArgs,
		registerComplex:   0,
		registerSliceByte: maxASMSliceByteArgs,
	}
)

// DispatchContext is the per-goroutine VM dispatch state shared with the ASM threaded
// dispatcher: program counter, code base, register file pointer, exit reason, and limit
// counters. Reads and writes from .s assembly use the offsets generated by cmd/asmgen so
// reordering fields requires regenerating asm_dispatch_offsets.h.
type DispatchContext struct {
	// vm is a back-pointer to the executing VM.
	//
	// Used by Go-side trampolines for tier-1 sub-ops that need access to vm.arena,
	// vm.evalError, or registers.general (the general reflect.Value bank, which is not
	// safely accessible via raw pointer arithmetic from ASM). Not read by ASM; only by Go
	// code in asm_call_trampolines.go.
	vm *VM

	// codeBase is the pointer to the first instruction in the compiled function body
	// (unsafe.Pointer to &body[0]).
	codeBase uintptr

	// codeLength is the number of instructions in the body.
	codeLength int64

	// programCounter is the current program counter (instruction index). Updated by ASM
	// before returning to Go.
	programCounter int64

	// intsBase is the pointer to the int64 register bank (unsafe.Pointer to
	// &registers.ints[0]).
	intsBase uintptr

	// intsLength is the number of int64 registers allocated.
	intsLength int64

	// floatsBase is the pointer to the float64 register bank (unsafe.Pointer to
	// &registers.floats[0]).
	floatsBase uintptr

	// floatsLength is the number of float64 registers allocated.
	floatsLength int64

	// intConstantsBase is the pointer to the int64 constant table (unsafe.Pointer to
	// &fn.intConstants[0]).
	intConstantsBase uintptr

	// intConstantsLength is the number of int64 constants.
	intConstantsLength int64

	// floatConstantsBase is the pointer to the float64 constant table (unsafe.Pointer to
	// &fn.floatConstants[0]).
	floatConstantsBase uintptr

	// floatConstantsLength is the number of float64 constants.
	floatConstantsLength int64

	// jumpTable is the pointer to the 256-entry dispatch table (unsafe.Pointer to
	// &jumpTable[0]). Each entry is a uintptr holding the absolute address of the handler
	// for that opcode.
	jumpTable uintptr

	// exitReason is written by ASM before returning to indicate why the dispatch loop
	// exited: 0 for end of code (pc >= codeLength), 1 for a tier 2 opcode that needs Go
	// handling, and 2 for a division-by-zero error.
	exitReason int64

	// exitProgramCounter is the program counter at which the dispatch loop exited. For tier
	// 2 exits, this is the PC of the instruction that needs Go handling.
	exitProgramCounter int64

	// asmCallInfoBase is the current function's asmCallInfo table base pointer.
	asmCallInfoBase uintptr

	// callStackBase is the pointer to the first element of the VM call stack.
	callStackBase uintptr

	// callStackLength is the number of entries in the VM call stack.
	callStackLength int64

	// framePointer is the current frame pointer index within the call stack.
	framePointer int64

	// baseFramePointer is the base frame pointer established by runDispatched.
	baseFramePointer int64

	// callDepthLimit is the maximum call depth allowed before overflow.
	callDepthLimit int64

	// arenaIntSlab is the pointer to the first element of the int register arena slab.
	arenaIntSlab uintptr

	// arenaIntCapacity is the total capacity of the int register arena slab.
	arenaIntCapacity int64

	// arenaIntIndex is the current allocation index into the int arena slab, read-write by
	// ASM.
	arenaIntIndex int64

	// arenaFloatSlab is the pointer to the first element of the float register arena slab.
	arenaFloatSlab uintptr

	// arenaFloatCapacity is the total capacity of the float register arena slab.
	arenaFloatCapacity int64

	// arenaFloatIndex is the current allocation index into the float arena slab, read-write
	// by ASM.
	arenaFloatIndex int64

	// arenaStringIndex is the current string arena allocation index, read-write by ASM.
	arenaStringIndex int64

	// arenaGeneralIndex is the current general arena allocation index, read-only by ASM.
	arenaGeneralIndex int64

	// arenaBoolIndex is the current bool arena allocation index, read-write by ASM.
	arenaBoolIndex int64

	// arenaUintIndex is the current uint arena allocation index, read-write by ASM.
	arenaUintIndex int64

	// arenaComplexIndex is the current complex arena allocation index, read-only by ASM.
	arenaComplexIndex int64

	// deferStackLength is the number of entries in the VM defer stack.
	deferStackLength int64

	// asmCallInfoBasesPointer is the pointer to the first element of the asmCallInfoBases
	// slice.
	asmCallInfoBasesPointer uintptr

	// dispatchSavesPointer is the pointer to the first element of the asmDispatchSaves
	// slice.
	dispatchSavesPointer uintptr

	// stringsBase is the pointer to the string register bank (unsafe.Pointer to
	// &registers.strings[0]). Each string is 16 bytes: {Data uintptr, Len int}.
	stringsBase uintptr

	// uintsBase is the pointer to the uint64 register bank (unsafe.Pointer to
	// &registers.uints[0]).
	uintsBase uintptr

	// boolsBase is the pointer to the bool register bank (unsafe.Pointer to
	// &registers.bools[0]).
	boolsBase uintptr

	// arenaStringSlab is the pointer to the first element of the string register arena slab.
	arenaStringSlab uintptr

	// arenaStringCapacity is the total capacity of the string register arena slab.
	arenaStringCapacity int64

	// arenaBoolSlab is the pointer to the first element of the bool register arena slab.
	arenaBoolSlab uintptr

	// arenaBoolCapacity is the total capacity of the bool register arena slab.
	arenaBoolCapacity int64

	// arenaUintSlab is the pointer to the first element of the uint register arena slab.
	arenaUintSlab uintptr

	// arenaUintCapacity is the total capacity of the uint register arena slab.
	arenaUintCapacity int64

	// slicesIntBase is the pointer to the first []int64 slice header in the typed slicesInt
	// register bank.
	//
	// Equal to unsafe.Pointer(&registers.slicesInt[0]). Each slot is 24 bytes (slice header:
	// ptr+len+cap). Used by tier-1 ASM sub-ops that read or update typed []int64 slices
	// without leaving the dispatch loop.
	slicesIntBase uintptr

	// slicesFloatBase is the pointer to the first []float64 slice header in the typed
	// slicesFloat register bank. Same 24-byte slot layout as slicesIntBase.
	slicesFloatBase uintptr

	// slicesStringBase is the pointer to the first []string slice header in the typed
	// slicesString register bank.
	slicesStringBase uintptr

	// slicesBoolBase is the pointer to the first []bool slice header in the typed slicesBool
	// register bank.
	slicesBoolBase uintptr

	// slicesUintBase is the pointer to the first []uint64 slice header in the typed
	// slicesUint register bank.
	slicesUintBase uintptr

	// complexBase is the pointer to the first complex128 element in the complex register
	// bank.
	//
	// Equal to unsafe.Pointer(&registers.complex[0]). Each slot is 16 bytes (two contiguous
	// float64s: real then imag). Used by the tier-1 ASM sub-ops handlerSubOpRealComplex /
	// handlerSubOpImagComplex / handlerSubOpMoveComplex / handlerSubOpNegComplex.
	complexBase uintptr

	// stringConstantsBase is the pointer to the first 16-byte string header in the active
	// function's string constant table.
	//
	// Equal to unsafe.Pointer(&fn.stringConstants[0]). Read by the ASM
	// handlerLoadStringConst on every opLoadStringConst dispatch. Mirrors intConstantsBase /
	// floatConstantsBase for the int / float constant pools.
	stringConstantsBase uintptr

	// stringConstantsLength is the number of entries in the string constant table. Kept for
	// verifier panics; the runtime does not bounds-check the const-load fast path because
	// the compiler only emits indices in range.
	stringConstantsLength int64

	// boolConstantsBase is the pointer to the first 1-byte entry in the active function's
	// bool constant table. Read by handlerLoadBoolConst.
	boolConstantsBase uintptr

	// boolConstantsLength is the number of entries in the bool constant table.
	boolConstantsLength int64

	// savedPC is the spill slot for the bytecode PC across a CALL.
	//
	// The tier-1 ASM real handlers use this to stash R14 (piko's bytecode PC, a uintptr
	// index) across a CALL into a Go trampoline. Go's ABI clobbers R14 (the runtime sets it
	// to G), so the value must be preserved somewhere and restored after the CALL. Spilling
	// to this ctx field instead of into the handler's local frame is what lets the handler
	// declare NO_LOCAL_POINTERS truthfully: the frame is left with no Go pointer locals at
	// all.
	//
	// Declared as uintptr (not unsafe.Pointer) so Go's GC explicitly does not scan through
	// it; the field stores an integer index, not a heap pointer.
	savedPC uintptr

	// tier2Result is the opResult byte returned from the most recent tier-2 ASM-call shim's
	// trampoline. The trampoline stores the Go handler's opResult here so the ASM shim can
	// branch on it (opContinue -> resume DISPATCH_NEXT; non-zero -> RET to Go).
	//
	// Why a ctx field instead of a Go multi-return: returning two values from the trampoline
	// (ctx *DispatchContext, rc opResult) would require the ASM shim to read the return
	// value from a stack slot AND recover R15 from another slot - two memory accesses across
	// what should be a single-branch hot path. Storing the result on ctx lets us return only
	// ctx (recovers R15 in one MOVQ from the return slot) and read the byte with one MOVBLZX
	// from CTX_TIER2_RESULT.
	//
	// On non-opContinue return codes the trampoline ALSO writes ctx.exitReason and
	// ctx.exitProgramCounter so the ASM shim's non-continue branch is just `RET`; Go's
	// handleDispatchExit dispatches based on the exit reason the trampoline already set.
	//
	// Single byte; padded to 8 in the struct layout.
	tier2Result uint8

	// structLayoutTableBase is the pointer to the first structFieldLayout entry.
	//
	// Equal to unsafe.Pointer(&fn.structLayoutTable[0]) of the active function. Read by ASM
	// tier-1 struct-field handlers (e.g. handlerSubOpStructFieldIntT0) to compute
	// &layoutTable[opC] for the field offset + kind. Mirrors intConstantsBase /
	// floatConstantsBase / stringConstantsBase / boolConstantsBase for the constant pools;
	// refreshed on every frame change in rebuildDispatchPointers. Zero when the active
	// function has no resolved struct-field layouts; the compiler only emits
	// layoutTable-indexed opcodes when at least one entry exists, so ASM never reads through
	// a zero base.
	//
	// Each layoutTable entry is 16 bytes; LAYOUT_SIZE_SHIFT = 4. Field offsets within
	// structFieldLayout are exposed as OFF_LAYOUT_* defines in asm_dispatch_offsets.h.
	structLayoutTableBase uintptr

	// structLayoutTableLength is the number of entries in the active function's
	// structLayoutTable, exposed as the informational length companion via the
	// CTX_STRUCT_LAYOUT_TABLE_LEN .h define (the compiler-emit invariant guarantees ASM-side
	// reads are in bounds).
	structLayoutTableLength int64

	// typeTableBase is the pointer to the first reflect.Type slot.
	//
	// Equal to unsafe.Pointer(&fn.typeTable[0]) of the active function. reflect.Type is a
	// 16-byte interface ({typ, data}); the data word is the *abi.Type pointer that
	// downstream ASM handlers (e.g. handlerSubOpLoadNil) read when materialising typed nil
	// values. Refreshed on every frame change in rebuildDispatchPointers; zero when the
	// active function has no recorded types.
	typeTableBase uintptr

	// typeTableLength is the number of entries in the active function's typeTable. Length
	// companion to typeTableBase; same informational semantics as structLayoutTableLength.
	typeTableLength int64

	// slicesByteBase is the pointer to the first []byte slice header in the typed slicesByte
	// register bank.
	//
	// Equal to unsafe.Pointer(&registers.slicesByte[0]). Each slot is 24 bytes (slice
	// header: ptr+len+cap). Used by tier-1 ASM sub-ops that read or update typed []byte
	// slices without leaving the dispatch loop: chunk[i], chunk[s:e], make([]byte).
	slicesByteBase uintptr

	// arenaSliceByteSlab is the pointer to the first slot of RegisterArena.slicesByteSlab.
	//
	// Each slot is a 24-byte slice header. Read by the ASM inline-call allocator to
	// bump-allocate the callee's slicesByte register bank.
	arenaSliceByteSlab uintptr

	// arenaSliceByteCapacity is the total slot count of RegisterArena.slicesByteSlab. The
	// inline-call allocator checks (current index + requested) against this cap before
	// bumping.
	arenaSliceByteCapacity int64

	// arenaSliceByteIndex is the current bump position of the typed byte-slice arena slab.
	// ASM updates this in place after a successful allocation; the Go-side restore path
	// reads it to know how far the arena has advanced.
	arenaSliceByteIndex int64

	// arenaSliceIntIndex mirrors RegisterArena.slicesIntIndex so the ASM inline-call shim
	// writes the correct save-point value into the new frame's arenaSave block.
	arenaSliceIntIndex int64

	// arenaSliceFloatIndex mirrors RegisterArena.slicesFloatIndex.
	arenaSliceFloatIndex int64

	// arenaSliceStringIndex mirrors RegisterArena.slicesStringIndex.
	arenaSliceStringIndex int64

	// arenaSliceBoolIndex mirrors RegisterArena.slicesBoolIndex.
	arenaSliceBoolIndex int64

	// arenaSliceUintIndex mirrors RegisterArena.slicesUintIndex.
	arenaSliceUintIndex int64
}

// asmCallInfo holds pre-computed metadata for one call site, enabling the ASM dispatch
// loop to handle interpreted-to-interpreted calls without trampolining to Go. Built once
// per function by buildASMCallInfoTables.
//
// Field offsets are hardcoded in the ASM handlers and verified by TestASMCallInfoOffsets.
type asmCallInfo struct {
	// calleeFunction is the unsafe pointer to the callee CompiledFunction.
	calleeFunction uintptr

	// calleeBody is the pointer to the first instruction of the callee function body.
	calleeBody uintptr

	// calleeBodyLength is the number of instructions in the callee function body.
	calleeBodyLength int64

	// calleeIntConstants is the pointer to the callee function's int constant table.
	calleeIntConstants uintptr

	// calleeFloatConstants is the pointer to the callee function's float constant table.
	calleeFloatConstants uintptr

	// calleeIntCount is the number of int registers required by the callee function.
	calleeIntCount int64

	// calleeFloatCount is the number of float registers required by the callee function.
	calleeFloatCount int64

	// intArgumentCount is the number of integer arguments to copy from caller to callee.
	intArgumentCount int64

	// intArgumentSources holds the caller int register index for each integer argument.
	intArgumentSources [8]int64

	// floatArgumentCount is the number of float arguments to copy from caller to callee.
	floatArgumentCount int64

	// floatArgumentSources holds the caller float register index for each float argument.
	floatArgumentSources [8]int64

	// returnCount is the number of return values, either zero or one.
	returnCount int64

	// returnDestinationKind is the register kind for the return destination, int or float.
	returnDestinationKind int64

	// returnDestinationRegister is the caller register index for storing the return value.
	returnDestinationRegister int64

	// returnDestinationPointer is the pointer to the first return location descriptor.
	returnDestinationPointer uintptr

	// returnDestinationLen is the number of entries in the return location descriptor slice.
	returnDestinationLen int64

	// calleeCallInfo is the pointer to the callee function's asmCallInfo table base.
	calleeCallInfo uintptr

	// isFastPath indicates the ASM inline dispatch mode. Values are defined by the fastPath*
	// constants.
	isFastPath int64

	// calleeStringCount is the number of string registers required by the callee function.
	calleeStringCount int64

	// calleeBoolCount is the number of bool registers required by the callee function.
	calleeBoolCount int64

	// calleeUintCount is the number of uint registers required by the callee function.
	calleeUintCount int64

	// stringArgumentCount is the number of string arguments to copy from caller to callee.
	stringArgumentCount int64

	// stringArgumentSources holds the caller string register index for each string argument.
	stringArgumentSources [8]int64

	// boolArgumentCount is the number of bool arguments to copy from caller to callee.
	boolArgumentCount int64

	// boolArgumentSources holds the caller bool register index for each bool argument.
	boolArgumentSources [8]int64

	// uintArgumentCount is the number of uint arguments to copy from caller to callee.
	uintArgumentCount int64

	// uintArgumentSources holds the caller uint register index for each uint argument.
	uintArgumentSources [8]int64

	// calleeStringConstants is a pointer to the callee function's string constant table
	// (16-byte Go string headers). Loaded into the dispatch context by the inline ASM call
	// handler so the callee's opLoadStringConst handler reads the right table.
	calleeStringConstants uintptr

	// calleeBoolConstants is a pointer to the callee function's bool constant table (1-byte
	// entries). Loaded into the dispatch context by the inline ASM call handler.
	calleeBoolConstants uintptr

	// calleeGeneralCount is the number of general (reflect.Value) registers required by the
	// callee function. isFastPath=3 uses this to size the callee's general-bank arena
	// allocation from the dispatch context's arenaGeneralSlab.
	calleeGeneralCount int64

	// generalArgumentCount is the number of general-bank arguments to copy from caller to
	// callee.
	generalArgumentCount int64

	// generalArgumentSources holds the caller general register index for each general
	// argument.
	generalArgumentSources [8]int64

	// calleeSliceByteCount is the number of slicesByte (typed []byte) registers required by
	// the callee function. The inline byte-slice path uses this to size the callee's
	// slicesByte bank from arena.slicesByteSlab via the dispatch context.
	calleeSliceByteCount int64

	// sliceByteArgumentCount is the number of []byte arguments to copy from caller to
	// callee.
	sliceByteArgumentCount int64

	// sliceByteArgumentSources holds the caller slicesByte register index for each []byte
	// argument.
	sliceByteArgumentSources [8]int64

	// _padding extends the struct size to 1024 bytes so the inline call dispatcher indexes
	// the per-site table via (siteIndex << ACI_SIZE_SHIFT) where the shift is 10. The fields
	// above account for 672 bytes; the remaining 352 are reserved.
	_padding [352]byte
}

// asmDispatchSave stores the dispatch register values that must be preserved across
// inline call/return. Saved by the call handler and restored by the return handler.
//
// Field offsets are hardcoded in the ASM handlers and verified by
// TestAsmDispatchSaveOffsets.
type asmDispatchSave struct {
	// codeBase is the pointer to the first instruction of the saved function body.
	codeBase uintptr

	// codeLength is the number of instructions in the saved function body.
	codeLength int64

	// intConstantsBase is the pointer to the saved function's int constant table.
	intConstantsBase uintptr

	// floatConstantsBase is the pointer to the saved function's float constant table.
	floatConstantsBase uintptr

	// stringConstantsBase is the pointer to the saved function's string constant table. Each
	// entry is a 16-byte Go string header.
	stringConstantsBase uintptr

	// boolConstantsBase is the pointer to the saved function's bool constant table. Each
	// entry is a single byte.
	boolConstantsBase uintptr

	// _reserved0 is reserved padding to keep the struct size stable so ASM offsets baked
	// into asm_dispatch_offsets.h do not shift when new fields are added.
	_reserved0 uintptr

	// _reserved1 is reserved padding; same rationale as _reserved0.
	_reserved1 uintptr
}

// buildDispatchContext populates a DispatchContext from the current VM frame state. The
// context is only valid for the lifetime of the current frame; after opCall or opReturn
// it must be rebuilt.
//
// Takes ctx (*DispatchContext) which is the dispatch context struct to populate with
// current frame state.
// Takes jumpTable (*[opcodeTableSize]uintptr) which is the opcode dispatch table mapping
// opcodes to handler addresses.
func (vm *VM) buildDispatchContext(ctx *DispatchContext, jumpTable *[opcodeTableSize]uintptr) {
	frame := &vm.callStack[vm.framePointer]
	body := frame.function.body
	registers := &frame.registers

	if len(body) > 0 {
		ctx.codeBase = uintptr(unsafe.Pointer(&body[0]))
	}
	ctx.codeLength = int64(len(body))
	ctx.programCounter = int64(frame.programCounter)

	populateRegisterBases(ctx, registers)
	populateConstantTableBases(ctx, frame.function)
	populateAuxiliaryTableBases(ctx, frame.function)

	if jumpTable != nil {
		ctx.jumpTable = uintptr(unsafe.Pointer(&jumpTable[0]))
	}

	ctx.exitReason = 0
	ctx.exitProgramCounter = 0

	ctx.callStackBase = uintptr(unsafe.Pointer(&vm.callStack[0]))
	ctx.callStackLength = int64(len(vm.callStack))
	ctx.framePointer = int64(vm.framePointer)
	ctx.baseFramePointer = int64(vm.baseFramePointer)
	ctx.callDepthLimit = int64(vm.callDepthLimit())
	ctx.deferStackLength = int64(len(vm.deferStack))

	vm.populateArenaContext(ctx)
	vm.populateExtendedBases(ctx, registers)
}

// populateExtendedBases writes string, uint, and bool register base pointers plus ASM
// metadata pointers into the dispatch context.
//
// Takes ctx (*DispatchContext) which is the dispatch context to populate with extended
// register base pointers.
// Takes registers (*Registers) which provides the string, uint, and bool register slices.
func (vm *VM) populateExtendedBases(ctx *DispatchContext, registers *Registers) {
	if len(registers.strings) > 0 {
		ctx.stringsBase = uintptr(unsafe.Pointer(&registers.strings[0]))
	}
	if len(registers.uints) > 0 {
		ctx.uintsBase = uintptr(unsafe.Pointer(&registers.uints[0]))
	}
	if len(registers.bools) > 0 {
		ctx.boolsBase = uintptr(unsafe.Pointer(&registers.bools[0]))
	}
	if len(registers.slicesInt) > 0 {
		ctx.slicesIntBase = uintptr(unsafe.Pointer(&registers.slicesInt[0]))
	}
	if len(registers.slicesFloat) > 0 {
		ctx.slicesFloatBase = uintptr(unsafe.Pointer(&registers.slicesFloat[0]))
	}
	if len(registers.slicesString) > 0 {
		ctx.slicesStringBase = uintptr(unsafe.Pointer(&registers.slicesString[0]))
	}
	if len(registers.slicesBool) > 0 {
		ctx.slicesBoolBase = uintptr(unsafe.Pointer(&registers.slicesBool[0]))
	}
	if len(registers.slicesUint) > 0 {
		ctx.slicesUintBase = uintptr(unsafe.Pointer(&registers.slicesUint[0]))
	}
	if len(registers.slicesByte) > 0 {
		ctx.slicesByteBase = uintptr(unsafe.Pointer(&registers.slicesByte[0]))
	}
	if len(registers.complex) > 0 {
		ctx.complexBase = uintptr(unsafe.Pointer(&registers.complex[0]))
	}
	ctx.vm = vm

	if len(vm.asmCallInfoBases) > 0 {
		ctx.asmCallInfoBasesPointer = uintptr(unsafe.Pointer(&vm.asmCallInfoBases[0]))
		ctx.asmCallInfoBase = vm.asmCallInfoBases[vm.framePointer]
	}
	if len(vm.asmDispatchSaves) > 0 {
		ctx.dispatchSavesPointer = uintptr(unsafe.Pointer(&vm.asmDispatchSaves[0]))
	}
}

// populateArenaContext writes the arena slab pointers and indices into the dispatch
// context so that ASM can allocate registers inline.
//
// Takes ctx (*DispatchContext) which is the dispatch context to populate with arena
// state.
func (vm *VM) populateArenaContext(ctx *DispatchContext) {
	if vm.arena == nil {
		return
	}
	if len(vm.arena.intSlab) > 0 {
		ctx.arenaIntSlab = uintptr(unsafe.Pointer(&vm.arena.intSlab[0]))
	}
	ctx.arenaIntCapacity = int64(len(vm.arena.intSlab))
	ctx.arenaIntIndex = int64(vm.arena.intIndex)
	if len(vm.arena.floatSlab) > 0 {
		ctx.arenaFloatSlab = uintptr(unsafe.Pointer(&vm.arena.floatSlab[0]))
	}
	ctx.arenaFloatCapacity = int64(len(vm.arena.floatSlab))
	ctx.arenaFloatIndex = int64(vm.arena.floatIndex)
	ctx.arenaStringIndex = int64(vm.arena.stringIndex)
	ctx.arenaGeneralIndex = int64(vm.arena.generalIndex)
	ctx.arenaBoolIndex = int64(vm.arena.boolIndex)
	ctx.arenaUintIndex = int64(vm.arena.uintIndex)
	ctx.arenaComplexIndex = int64(vm.arena.complexIndex)
	if len(vm.arena.stringSlab) > 0 {
		ctx.arenaStringSlab = uintptr(unsafe.Pointer(&vm.arena.stringSlab[0]))
	}
	ctx.arenaStringCapacity = int64(len(vm.arena.stringSlab))
	if len(vm.arena.boolSlab) > 0 {
		ctx.arenaBoolSlab = uintptr(unsafe.Pointer(&vm.arena.boolSlab[0]))
	}
	ctx.arenaBoolCapacity = int64(len(vm.arena.boolSlab))
	if len(vm.arena.uintSlab) > 0 {
		ctx.arenaUintSlab = uintptr(unsafe.Pointer(&vm.arena.uintSlab[0]))
	}
	ctx.arenaUintCapacity = int64(len(vm.arena.uintSlab))
	ctx.arenaSliceByteIndex = int64(vm.arena.slicesByteIndex)
	if len(vm.arena.slicesByteSlab) > 0 {
		ctx.arenaSliceByteSlab = uintptr(unsafe.Pointer(&vm.arena.slicesByteSlab[0]))
	}
	ctx.arenaSliceByteCapacity = int64(len(vm.arena.slicesByteSlab))
	ctx.arenaSliceIntIndex = int64(vm.arena.slicesIntIndex)
	ctx.arenaSliceFloatIndex = int64(vm.arena.slicesFloatIndex)
	ctx.arenaSliceStringIndex = int64(vm.arena.slicesStringIndex)
	ctx.arenaSliceBoolIndex = int64(vm.arena.slicesBoolIndex)
	ctx.arenaSliceUintIndex = int64(vm.arena.slicesUintIndex)
}

// syncCallContextFromASM updates VM state from the DispatchContext after the ASM loop
// returns. ASM may have modified fp, arenaIntIndex, and arenaFloatIndex via inline
// call/return.
//
// Skips per-bank writes whose ctx value matches vm.arena's existing value. On hot inner
// loops that never run handlerCallInline, every arena bank index sits at the same value
// at ASM entry and exit, so the compare-then-skip pattern elides six unconditional stores
// per ASM-loop exit.
//
// Takes ctx (*DispatchContext) which is the dispatch context containing the updated ASM
// state.
func (vm *VM) syncCallContextFromASM(ctx *DispatchContext) {
	newFp := int(ctx.framePointer)
	vm.applyPoppedFrameSnapshots(newFp)
	vm.framePointer = newFp
	if vm.arena != nil {
		syncArenaIndicesFromASM(vm.arena, ctx)
	}
}

// applyPoppedFrameSnapshots restores popped-frame dispatch state.
//
// ASM-side inline return handlers (handlerReturnInline / handlerReturnVoidInline) update
// CTX_FRAME_POINTER and arena indices without going through Go's vm.popFrame; the Go-side
// vm.rootFunction / vm.functions tables and the parallel rootSnapshots slice are
// untouched. When a popped frame had a non-nil snapshot (recorded because a cross-bundle
// closure or external-method swap happened during its push), the caller's dispatch tables
// would otherwise remain pointing at the callee's bundle. Walk the popped range
// oldest-to-newest so restoration matches an equivalent series of vm.popFrame calls.
//
// Takes newFp (int) which is the new frame-pointer value after the ASM-side pop.
func (vm *VM) applyPoppedFrameSnapshots(newFp int) {
	if vm.framePointer <= newFp || newFp < -1 {
		return
	}
	for fp := newFp + 1; fp <= vm.framePointer && fp < len(vm.rootSnapshots); fp++ {
		snapshot := vm.rootSnapshots[fp]
		if snapshot == nil {
			continue
		}
		vm.functions = snapshot.functions
		vm.rootFunction = snapshot.rootFunction
		vm.rootSnapshots[fp] = nil
	}
}

// refreshCallContext updates the call-related fields in the DispatchContext after a
// Go-side frame change (push/pop).
//
// Takes ctx (*DispatchContext) which is the dispatch context to refresh with current call
// stack state.
func (vm *VM) refreshCallContext(ctx *DispatchContext) {
	ctx.callStackBase = uintptr(unsafe.Pointer(&vm.callStack[0]))
	ctx.callStackLength = int64(len(vm.callStack))
	ctx.framePointer = int64(vm.framePointer)
	ctx.deferStackLength = int64(len(vm.deferStack))
	if vm.arena != nil {
		vm.refreshArenaSlabs(ctx)
	}
	if len(vm.asmCallInfoBases) > 0 {
		if vm.framePointer >= 0 && vm.framePointer < len(vm.asmCallInfoBases) {
			ctx.asmCallInfoBase = vm.asmCallInfoBases[vm.framePointer]
		}
		ctx.asmCallInfoBasesPointer = uintptr(unsafe.Pointer(&vm.asmCallInfoBases[0]))
	}
	if len(vm.asmDispatchSaves) > 0 {
		ctx.dispatchSavesPointer = uintptr(unsafe.Pointer(&vm.asmDispatchSaves[0]))
	}
}

// refreshArenaSlabs updates the arena slab pointers, capacities, and indices in the
// dispatch context from the current arena state.
//
// Takes ctx (*DispatchContext) which is the dispatch context to refresh with current
// arena slab state.
func (vm *VM) refreshArenaSlabs(ctx *DispatchContext) {
	arena := vm.arena
	ctx.arenaIntIndex = int64(arena.intIndex)
	ctx.arenaFloatIndex = int64(arena.floatIndex)
	if len(arena.intSlab) > 0 {
		ctx.arenaIntSlab = uintptr(unsafe.Pointer(&arena.intSlab[0]))
	}
	ctx.arenaIntCapacity = int64(len(arena.intSlab))
	if len(arena.floatSlab) > 0 {
		ctx.arenaFloatSlab = uintptr(unsafe.Pointer(&arena.floatSlab[0]))
	}
	ctx.arenaFloatCapacity = int64(len(arena.floatSlab))
	ctx.arenaStringIndex = int64(arena.stringIndex)
	ctx.arenaGeneralIndex = int64(arena.generalIndex)
	ctx.arenaBoolIndex = int64(arena.boolIndex)
	ctx.arenaUintIndex = int64(arena.uintIndex)
	ctx.arenaComplexIndex = int64(arena.complexIndex)
	if len(arena.stringSlab) > 0 {
		ctx.arenaStringSlab = uintptr(unsafe.Pointer(&arena.stringSlab[0]))
	}
	ctx.arenaStringCapacity = int64(len(arena.stringSlab))
	if len(arena.boolSlab) > 0 {
		ctx.arenaBoolSlab = uintptr(unsafe.Pointer(&arena.boolSlab[0]))
	}
	ctx.arenaBoolCapacity = int64(len(arena.boolSlab))
	if len(arena.uintSlab) > 0 {
		ctx.arenaUintSlab = uintptr(unsafe.Pointer(&arena.uintSlab[0]))
	}
	ctx.arenaUintCapacity = int64(len(arena.uintSlab))
	ctx.arenaSliceByteIndex = int64(arena.slicesByteIndex)
	if len(arena.slicesByteSlab) > 0 {
		ctx.arenaSliceByteSlab = uintptr(unsafe.Pointer(&arena.slicesByteSlab[0]))
	}
	ctx.arenaSliceByteCapacity = int64(len(arena.slicesByteSlab))
	ctx.arenaSliceIntIndex = int64(arena.slicesIntIndex)
	ctx.arenaSliceFloatIndex = int64(arena.slicesFloatIndex)
	ctx.arenaSliceStringIndex = int64(arena.slicesStringIndex)
	ctx.arenaSliceBoolIndex = int64(arena.slicesBoolIndex)
	ctx.arenaSliceUintIndex = int64(arena.slicesUintIndex)
}

// saveCurrentDispatchRegisters writes the current frame's dispatch register values into
// dispSaves[vm.framePointer]. This must be called before entering the ASM dispatch loop
// so that the inline return handler can restore the caller's dispatch state even when the
// call went through Go.
//
// Takes ctx (*DispatchContext) which is the dispatch context containing the current
// register values to save.
func (vm *VM) saveCurrentDispatchRegisters(ctx *DispatchContext) {
	if vm.asmDispatchSaves != nil && vm.framePointer >= 0 && vm.framePointer < len(vm.asmDispatchSaves) {
		save := &vm.asmDispatchSaves[vm.framePointer]
		save.codeBase = ctx.codeBase
		save.codeLength = ctx.codeLength
		save.intConstantsBase = ctx.intConstantsBase
		save.floatConstantsBase = ctx.floatConstantsBase
		save.stringConstantsBase = ctx.stringConstantsBase
		save.boolConstantsBase = ctx.boolConstantsBase
	}
}

// populateRegisterBases writes the int and float register base pointers and their lengths
// into the dispatch context.
//
// Takes ctx (*DispatchContext) which receives the base addresses.
// Takes registers (*Registers) which is the current frame's register set.
func populateRegisterBases(ctx *DispatchContext, registers *Registers) {
	if len(registers.ints) > 0 {
		ctx.intsBase = uintptr(unsafe.Pointer(&registers.ints[0]))
	}
	ctx.intsLength = int64(len(registers.ints))
	if len(registers.floats) > 0 {
		ctx.floatsBase = uintptr(unsafe.Pointer(&registers.floats[0]))
	}
	ctx.floatsLength = int64(len(registers.floats))
}

// populateConstantTableBases writes the int, float, string, and bool constant-table base
// pointers and lengths into the dispatch context.
//
// Takes ctx (*DispatchContext) which receives the base addresses.
// Takes function (*CompiledFunction) which holds the constant tables.
func populateConstantTableBases(ctx *DispatchContext, function *CompiledFunction) {
	if len(function.intConstants) > 0 {
		ctx.intConstantsBase = uintptr(unsafe.Pointer(&function.intConstants[0]))
	}
	ctx.intConstantsLength = int64(len(function.intConstants))
	if len(function.floatConstants) > 0 {
		ctx.floatConstantsBase = uintptr(unsafe.Pointer(&function.floatConstants[0]))
	}
	ctx.floatConstantsLength = int64(len(function.floatConstants))
	if len(function.stringConstants) > 0 {
		ctx.stringConstantsBase = uintptr(unsafe.Pointer(&function.stringConstants[0]))
	}
	ctx.stringConstantsLength = int64(len(function.stringConstants))
	if len(function.boolConstants) > 0 {
		ctx.boolConstantsBase = uintptr(unsafe.Pointer(&function.boolConstants[0]))
	}
	ctx.boolConstantsLength = int64(len(function.boolConstants))
}

// populateAuxiliaryTableBases writes the struct-layout and type-table base pointers and
// lengths into the dispatch context. These fields are shared with rebuildDispatchPointers
// and must be initialised on the first dispatch entry so trampolines reading
// ctx.structLayoutTableBase do not dereference a bogus pointer.
//
// Takes ctx (*DispatchContext) which receives the base addresses.
// Takes function (*CompiledFunction) which holds the layout tables.
func populateAuxiliaryTableBases(ctx *DispatchContext, function *CompiledFunction) {
	if len(function.structLayoutTable) > 0 {
		ctx.structLayoutTableBase = uintptr(unsafe.Pointer(&function.structLayoutTable[0]))
	}
	ctx.structLayoutTableLength = int64(len(function.structLayoutTable))
	if len(function.typeTable) > 0 {
		ctx.typeTableBase = uintptr(unsafe.Pointer(&function.typeTable[0]))
	}
	ctx.typeTableLength = int64(len(function.typeTable))
}

// syncArenaIndicesFromASM copies arena cursors back from the context.
//
// Takes arena (*RegisterArena) which receives the cursor values.
// Takes ctx (*DispatchContext) which holds the ASM-advanced cursors.
func syncArenaIndicesFromASM(arena *RegisterArena, ctx *DispatchContext) {
	arena.intIndex = int(ctx.arenaIntIndex)
	arena.floatIndex = int(ctx.arenaFloatIndex)
	arena.stringIndex = int(ctx.arenaStringIndex)
	arena.boolIndex = int(ctx.arenaBoolIndex)
	arena.uintIndex = int(ctx.arenaUintIndex)
	arena.slicesByteIndex = int(ctx.arenaSliceByteIndex)
	arena.slicesIntIndex = int(ctx.arenaSliceIntIndex)
	arena.slicesFloatIndex = int(ctx.arenaSliceFloatIndex)
	arena.slicesStringIndex = int(ctx.arenaSliceStringIndex)
	arena.slicesBoolIndex = int(ctx.arenaSliceBoolIndex)
	arena.slicesUintIndex = int(ctx.arenaSliceUintIndex)
}

// ensureASMCallInfoTables returns the asmCallInfoTables map for a root function, building
// it on first use under the function's asmCallInfoTablesOnce.
//
// The sync.Once gate makes the build race-free: interpreted `go` statements launch child
// VMs that share the same root function, and the per-CompiledFunction state mutated
// during build cannot be raced. Routing every caller through this helper guarantees one
// build per root, with subsequent callers reading the published map.
//
// Takes rootFunction (*CompiledFunction) whose asmCallInfoTablesOnce guards the build and
// whose asmCallInfoTables receives the result.
//
// Returns the function-pointer-keyed asmCallInfo table map. Empty when the root has no
// functions or no callsites.
func ensureASMCallInfoTables(rootFunction *CompiledFunction) map[*CompiledFunction][]asmCallInfo {
	rootFunction.asmCallInfoTablesOnce.Do(func() {
		rootFunction.asmCallInfoTables, _ = buildASMCallInfoTables(rootFunction, rootFunction.functions)
	})
	return rootFunction.asmCallInfoTables
}

// buildASMCallInfoTables pre-computes asmCallInfo tables for all functions in the
// program.
//
// Callers MUST go through ensureASMCallInfoTables rather than invoking this directly. The
// function is only exported within the package so the sync.Once helper can call it;
// concurrent direct invocation for the same root races on the shared per-function
// table-slice state populated below.
//
// Takes rootFunction (*CompiledFunction) which is the entry-point function for the
// program.
// Takes functions ([]*CompiledFunction) which is the complete list of compiled functions
// in the program.
//
// Returns a map from function pointer to its asmCallInfo table, and the root function's
// table for direct use by the dispatch loop.
func buildASMCallInfoTables(rootFunction *CompiledFunction, functions []*CompiledFunction) (map[*CompiledFunction][]asmCallInfo, []asmCallInfo) {
	tables := make(map[*CompiledFunction][]asmCallInfo)
	buildASMCallInfoTableFor(rootFunction, functions, tables)
	for _, compiledFunction := range functions {
		if _, ok := tables[compiledFunction]; !ok {
			buildASMCallInfoTableFor(compiledFunction, functions, tables)
		}
	}
	linkCalleeCallInfoPointers(functions, tables)
	publishAsmCallInfoBases(tables)
	return tables, tables[rootFunction]
}

// linkCalleeCallInfoPointers fills each asmCallInfo entry's calleeCallInfo with the base
// pointer of the callee's own table, so the ASM dispatcher can chain into the callee's
// fast-path slots without re-looking up the table.
//
// Takes functions ([]*CompiledFunction) which is the indexed list of compiled functions
// used to resolve callees.
// Takes tables (map[*CompiledFunction][]asmCallInfo) which is the per-function
// asmCallInfo tables to link in place.
func linkCalleeCallInfoPointers(functions []*CompiledFunction, tables map[*CompiledFunction][]asmCallInfo) {
	for compiledFunction, table := range tables {
		for i := range table {
			if table[i].isFastPath == fastPathIneligible {
				continue
			}
			site := &compiledFunction.callSites[i]
			callee := functions[site.funcIndex]
			if calleeTable, ok := tables[callee]; ok && len(calleeTable) > 0 {
				table[i].calleeCallInfo = uintptr(unsafe.Pointer(&calleeTable[0]))
			}
		}
	}
}

// publishAsmCallInfoBases caches each function's own asmCallInfo table base.
//
// Stores the base directly on the CompiledFunction so updateASMCallInfoBase can read a
// single uintptr field instead of probing a map. Safe because ensureASMCallInfoTables's
// sync.Once gates the entire build: each root function's tables are built exactly once
// and the resulting writes are observed as stable values by every subsequent reader (the
// sync.Once provides the publication barrier).
//
// Takes tables (map[*CompiledFunction][]asmCallInfo) which is the per-function
// asmCallInfo tables whose bases are published.
func publishAsmCallInfoBases(tables map[*CompiledFunction][]asmCallInfo) {
	for compiledFunction, table := range tables {
		if len(table) > 0 {
			compiledFunction.asmCallInfoBase = uintptr(unsafe.Pointer(&table[0]))
		} else {
			compiledFunction.asmCallInfoBase = 0
		}
	}
}

// buildASMCallInfoTableFor builds the asmCallInfo table for a single compiled function
// and stores it in the tables map.
//
// Takes function (*CompiledFunction) which is the function to build the table for.
// Takes functions ([]*CompiledFunction) which is the complete list of compiled functions
// for callee resolution.
// Takes tables (map[*CompiledFunction][]asmCallInfo) which is the map to store the
// resulting table in.
func buildASMCallInfoTableFor(function *CompiledFunction, functions []*CompiledFunction, tables map[*CompiledFunction][]asmCallInfo) {
	if len(function.callSites) == 0 {
		tables[function] = nil
		return
	}
	table := make([]asmCallInfo, len(function.callSites))
	for i := range function.callSites {
		buildOneASMCallInfo(&table[i], &function.callSites[i], functions)
	}
	tables[function] = table
}

// buildOneASMCallInfo populates a single asmCallInfo entry for a call site if it is
// eligible for ASM fast-path dispatch.
//
// Takes info (*asmCallInfo) which is the asmCallInfo entry to populate.
// Takes site (*callSite) which is the call site descriptor from the caller function.
// Takes functions ([]*CompiledFunction) which is the complete list of compiled functions
// for callee resolution.
func buildOneASMCallInfo(info *asmCallInfo, site *callSite, functions []*CompiledFunction) {
	callee := resolveASMCallee(site, functions)
	if callee == nil {
		return
	}
	if !mapASMArguments(info, site, callee) {
		return
	}
	if !configureASMReturn(info, site, callee) {
		return
	}
	populateASMCalleeFields(info, site, callee)
}

// resolveASMCallee returns the callee function if the call site is eligible for ASM
// dispatch, or nil if it should be skipped.
//
// Takes site (*callSite) which is the call site descriptor to evaluate for eligibility.
// Takes functions ([]*CompiledFunction) which is the complete list of compiled functions
// for index lookup.
//
// Returns the resolved callee CompiledFunction, or nil if the call site is ineligible due
// to closures, native calls, variadic signatures, or non-int/float registers.
func resolveASMCallee(site *callSite, functions []*CompiledFunction) *CompiledFunction {
	if site.isClosure || site.isNative {
		return nil
	}
	if int(site.funcIndex) >= len(functions) {
		return nil
	}
	callee := functions[site.funcIndex]
	if callee.isVariadic || len(callee.upvalueDescriptors) > 0 {
		return nil
	}
	if callee.numRegisters[registerComplex] > 0 {
		return nil
	}
	if asmCalleeParameterLayoutPerturbed(callee) {
		return nil
	}
	return callee
}

// asmCalleeParameterLayoutPerturbed reports whether the callee's parameter slots are
// scattered relative to the naive per-bank counter (slot 0 for the first param of kind K,
// slot 1 for the second, ...).
//
// The ASM fast-path argument-copy trampolines write each argument to
// calleeRegisters[bank][i] for the i-th argument of that bank, with no awareness of
// CompiledFunction.parameterRegisters. promoteToIndirect (invoked during
// compileFuncParams when a parameter has its address taken or is captured by a closure)
// inserts an extra general-bank allocation BETWEEN parameter declareVar calls, shifting
// later same-bank parameters off their naive slot. The mismatch corrupts the callee
// frame: the heap-promote pointer slot receives the caller's argument and the real
// parameter slot stays zero. Refusing ASM fast path here forces the call through Go's
// handleCall, whose argCopyProgram already consults parameterRegisters.
//
// Takes callee (*CompiledFunction) whose parameter layout is examined.
//
// Returns true when parameterRegisters disagrees with the naive layout.
func asmCalleeParameterLayoutPerturbed(callee *CompiledFunction) bool {
	if len(callee.parameterRegisters) != len(callee.parameterKinds) {
		return false
	}
	var bankCounter [NumRegisterKinds]uint8
	for i, paramKind := range callee.parameterKinds {
		expected := bankCounter[paramKind]
		bankCounter[paramKind]++
		if callee.parameterRegisters[i] != expected {
			return true
		}
	}
	return false
}

// asmArgumentSourceSlice returns the argument source array for the given register kind,
// or nil for unsupported kinds.
//
// Takes info (*asmCallInfo) which is the call info entry containing the per-kind argument
// source arrays.
// Takes kind (registerKind) which selects the register kind whose source array to return.
//
// Returns []int64 which is the argument source slice for the given kind, or nil if the
// kind is unsupported.
func asmArgumentSourceSlice(info *asmCallInfo, kind registerKind) []int64 {
	switch kind {
	case registerInt:
		return info.intArgumentSources[:]
	case registerFloat:
		return info.floatArgumentSources[:]
	case registerString:
		return info.stringArgumentSources[:]
	case registerBool:
		return info.boolArgumentSources[:]
	case registerUint:
		return info.uintArgumentSources[:]
	case registerGeneral:
		return info.generalArgumentSources[:]
	case registerSliceByte:
		return info.sliceByteArgumentSources[:]
	default:
		return nil
	}
}

// mapASMArguments maps the call site's arguments to the ASM info's source arrays and
// writes the per-kind argument counts into info.
//
// Takes info (*asmCallInfo) which is the asmCallInfo entry to populate with argument
// source indices and counts.
// Takes site (*callSite) which is the call site descriptor containing the argument
// locations.
// Takes callee (*CompiledFunction) which is the target function whose parameter kinds
// must match.
//
// Returns true if all arguments were successfully mapped, or false if any argument kind
// does not match or exceeds the maximum count.
func mapASMArguments(info *asmCallInfo, site *callSite, callee *CompiledFunction) bool {
	var counts [NumRegisterKinds]int
	for argumentIndex, argumentLocation := range site.arguments {
		if argumentIndex >= len(callee.parameterKinds) {
			break
		}
		parameterKind := callee.parameterKinds[argumentIndex]
		if argumentLocation.kind != parameterKind {
			return false
		}
		sources := asmArgumentSourceSlice(info, parameterKind)
		if sources == nil || counts[parameterKind] >= maxASMArgsByKind[parameterKind] {
			return false
		}
		sources[counts[parameterKind]] = int64(argumentLocation.register)
		counts[parameterKind]++
	}
	info.intArgumentCount = int64(counts[registerInt])
	info.floatArgumentCount = int64(counts[registerFloat])
	info.stringArgumentCount = int64(counts[registerString])
	info.boolArgumentCount = int64(counts[registerBool])
	info.uintArgumentCount = int64(counts[registerUint])
	info.generalArgumentCount = int64(counts[registerGeneral])
	info.sliceByteArgumentCount = int64(counts[registerSliceByte])
	return true
}

// configureASMReturn validates and configures the return destination for ASM fast-path
// dispatch.
//
// Takes info (*asmCallInfo) which is the asmCallInfo entry to populate with return
// metadata.
// Takes site (*callSite) which is the call site descriptor containing return location
// descriptors.
// Takes callee (*CompiledFunction) which is the target function whose result kinds are
// validated.
//
// Returns true if the return shape is compatible with ASM dispatch, or false if there are
// multiple returns or incompatible register kinds.
func configureASMReturn(info *asmCallInfo, site *callSite, callee *CompiledFunction) bool {
	if len(site.returns) > 1 {
		return false
	}
	if len(site.returns) == 0 {
		return true
	}
	returnLocation := site.returns[0]
	if returnLocation.isUpvalue || len(callee.resultKinds) == 0 {
		return false
	}
	resultKind := callee.resultKinds[0]
	if resultKind != registerInt && resultKind != registerFloat &&
		resultKind != registerString && resultKind != registerBool &&
		resultKind != registerUint {
		return false
	}
	if returnLocation.kind != resultKind {
		return false
	}
	info.returnCount = 1
	info.returnDestinationKind = int64(returnLocation.kind)
	info.returnDestinationRegister = int64(returnLocation.register)
	return true
}

// populateASMCalleeFields fills the callee-specific pointer and size fields that the ASM
// dispatch loop needs to set up an inline call frame. Argument counts must already be
// written to info by mapASMArguments.
//
// Takes info (*asmCallInfo) which is the asmCallInfo entry to populate with callee
// pointers and sizes.
// Takes site (*callSite) which is the call site descriptor containing return location
// data.
// Takes callee (*CompiledFunction) which is the target compiled function providing body
// and constant pointers.
func populateASMCalleeFields(info *asmCallInfo, site *callSite, callee *CompiledFunction) {
	info.calleeFunction = uintptr(unsafe.Pointer(callee))
	if len(callee.body) > 0 {
		info.calleeBody = uintptr(unsafe.Pointer(&callee.body[0]))
	}
	info.calleeBodyLength = int64(len(callee.body))
	if len(callee.intConstants) > 0 {
		info.calleeIntConstants = uintptr(unsafe.Pointer(&callee.intConstants[0]))
	}
	if len(callee.floatConstants) > 0 {
		info.calleeFloatConstants = uintptr(unsafe.Pointer(&callee.floatConstants[0]))
	}
	if len(callee.stringConstants) > 0 {
		info.calleeStringConstants = uintptr(unsafe.Pointer(&callee.stringConstants[0]))
	}
	if len(callee.boolConstants) > 0 {
		info.calleeBoolConstants = uintptr(unsafe.Pointer(&callee.boolConstants[0]))
	}
	info.calleeIntCount = int64(callee.numRegisters[registerInt])
	info.calleeFloatCount = int64(callee.numRegisters[registerFloat])
	info.calleeStringCount = int64(callee.numRegisters[registerString])
	info.calleeBoolCount = int64(callee.numRegisters[registerBool])
	info.calleeUintCount = int64(callee.numRegisters[registerUint])
	info.calleeGeneralCount = int64(callee.numRegisters[registerGeneral])
	info.calleeSliceByteCount = int64(callee.numRegisters[registerSliceByte])
	if len(site.returns) > 0 {
		info.returnDestinationPointer = uintptr(unsafe.Pointer(&site.returns[0]))
		info.returnDestinationLen = int64(len(site.returns))
	}
	if calleeUsesTypedSliceBanks(callee) {
		info.isFastPath = fastPathIneligible
		return
	}
	switch {
	case callee.numRegisters[registerGeneral] > 0:
		info.isFastPath = fastPathGeneralBank
	case callee.numRegisters[registerString] == 0 &&
		callee.numRegisters[registerBool] == 0 &&
		callee.numRegisters[registerUint] == 0 &&
		callee.numRegisters[registerSliceByte] == 0:
		info.isFastPath = fastPathPureIntFloat
	default:
		info.isFastPath = fastPathExtendedBanks
	}
}

// calleeUsesTypedSliceBanks reports whether callee uses an unsupported typed-slice bank.
//
// Detects when callee allocates any register in a typed-slice bank that the ASM inline
// call path does not know how to size and bind. registerSliceByte is supported by the ASM
// path: the inline call/return blocks pivot CTX_SLICES_BYTE_BASE to the callee's
// slicesByte slab on entry and restore the caller's on return, so byte-touching tier-1
// handlers index a valid bank for the active frame.
//
// Takes callee (*CompiledFunction) whose register watermark is inspected.
//
// Returns true when at least one ASM-unsupported typed-slice bank has nonzero register
// count.
func calleeUsesTypedSliceBanks(callee *CompiledFunction) bool {
	return callee.numRegisters[registerSliceInt] > 0 ||
		callee.numRegisters[registerSliceFloat] > 0 ||
		callee.numRegisters[registerSliceString] > 0 ||
		callee.numRegisters[registerSliceBool] > 0 ||
		callee.numRegisters[registerSliceUint] > 0
}
