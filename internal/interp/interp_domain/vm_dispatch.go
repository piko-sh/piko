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

import "unsafe"

// DispatchContext is the flat struct passed to the ASM dispatch loop. It serves as
// the sole interface between Go and assembly; the ASM code reads and writes only
// through the context, never touching Go object internals directly.
//
// Field offsets are hardcoded in the ASM handlers and verified by
// TestDispatchContextOffsets. If you add, remove, or reorder fields,
// update both the ASM constants and the offset test.
type DispatchContext struct {
	// codeBase is the pointer to the first instruction in the
	// compiled function body (unsafe.Pointer to &body[0]).
	codeBase uintptr

	// codeLength is the number of instructions in the body.
	codeLength int64

	// programCounter is the current program counter (instruction index).
	// Updated by ASM before returning to Go.
	programCounter int64

	// intsBase is the pointer to the int64 register bank
	// (unsafe.Pointer to &registers.ints[0]).
	intsBase uintptr

	// intsLength is the number of int64 registers allocated.
	intsLength int64

	// floatsBase is the pointer to the float64 register bank
	// (unsafe.Pointer to &registers.floats[0]).
	floatsBase uintptr

	// floatsLength is the number of float64 registers allocated.
	floatsLength int64

	// intConstantsBase is the pointer to the int64 constant table
	// (unsafe.Pointer to &fn.intConstants[0]).
	intConstantsBase uintptr

	// intConstantsLength is the number of int64 constants.
	intConstantsLength int64

	// floatConstantsBase is the pointer to the float64 constant table
	// (unsafe.Pointer to &fn.floatConstants[0]).
	floatConstantsBase uintptr

	// floatConstantsLength is the number of float64 constants.
	floatConstantsLength int64

	// jumpTable is the pointer to the 256-entry dispatch table
	// (unsafe.Pointer to &jumpTable[0]). Each entry is a uintptr
	// holding the absolute address of the handler for that opcode.
	jumpTable uintptr

	// exitReason is written by ASM before returning to indicate why
	// the dispatch loop exited.
	//
	//   0 = end of code (pc >= codeLength)
	//   1 = tier 2 opcode (needs Go handling)
	//   2 = division by zero error.
	exitReason int64

	// exitProgramCounter is the program counter at which the dispatch loop
	// exited. For tier 2 exits, this is the PC of the instruction
	// that needs Go handling.
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

	// arenaIntSlab is the pointer to the first element of the int
	// register arena slab.
	arenaIntSlab uintptr

	// arenaIntCapacity is the total capacity of the int register arena slab.
	arenaIntCapacity int64

	// arenaIntIndex is the current allocation index into the int arena
	// slab, read-write by ASM.
	arenaIntIndex int64

	// arenaFloatSlab is the pointer to the first element of the float
	// register arena slab.
	arenaFloatSlab uintptr

	// arenaFloatCapacity is the total capacity of the float register arena slab.
	arenaFloatCapacity int64

	// arenaFloatIndex is the current allocation index into the float
	// arena slab, read-write by ASM.
	arenaFloatIndex int64

	// arenaStringIndex is the current string arena allocation index,
	// read-write by ASM.
	arenaStringIndex int64

	// arenaGeneralIndex is the current general arena allocation index,
	// read-only by ASM.
	arenaGeneralIndex int64

	// arenaBoolIndex is the current bool arena allocation index, read-write by ASM.
	arenaBoolIndex int64

	// arenaUintIndex is the current uint arena allocation index, read-write by ASM.
	arenaUintIndex int64

	// arenaComplexIndex is the current complex arena allocation index,
	// read-only by ASM.
	arenaComplexIndex int64

	// deferStackLength is the number of entries in the VM defer stack.
	deferStackLength int64

	// asmCallInfoBasesPointer is the pointer to the first element of
	// the asmCallInfoBases slice.
	asmCallInfoBasesPointer uintptr

	// dispatchSavesPointer is the pointer to the first element of
	// the asmDispatchSaves slice.
	dispatchSavesPointer uintptr

	// stringsBase is the pointer to the string register bank
	// (unsafe.Pointer to &registers.strings[0]).
	// Each string is 16 bytes: {Data uintptr, Len int}.
	stringsBase uintptr

	// uintsBase is the pointer to the uint64 register bank
	// (unsafe.Pointer to &registers.uints[0]).
	uintsBase uintptr

	// boolsBase is the pointer to the bool register bank
	// (unsafe.Pointer to &registers.bools[0]).
	boolsBase uintptr

	// arenaStringSlab is the pointer to the first element of the string
	// register arena slab.
	arenaStringSlab uintptr

	// arenaStringCapacity is the total capacity of the string register
	// arena slab.
	arenaStringCapacity int64

	// arenaBoolSlab is the pointer to the first element of the bool
	// register arena slab.
	arenaBoolSlab uintptr

	// arenaBoolCapacity is the total capacity of the bool register
	// arena slab.
	arenaBoolCapacity int64

	// arenaUintSlab is the pointer to the first element of the uint
	// register arena slab.
	arenaUintSlab uintptr

	// arenaUintCapacity is the total capacity of the uint register
	// arena slab.
	arenaUintCapacity int64
}

const (
	// exitEndOfCode indicates the dispatch loop exited because the
	// program counter reached the end of the code.
	exitEndOfCode int64 = 0

	// exitTier2 indicates the dispatch loop exited because a tier 2
	// opcode requires Go-side handling.
	exitTier2 int64 = 1

	// exitDivByZero indicates the dispatch loop exited due to a
	// division by zero error.
	exitDivByZero int64 = 2

	// exitCall indicates the dispatch loop exited to perform a
	// function call via Go.
	exitCall int64 = 3

	// exitReturn indicates the dispatch loop exited to perform a
	// function return via Go.
	exitReturn int64 = 4

	// exitReturnVoid indicates the dispatch loop exited to perform a
	// void function return via Go.
	exitReturnVoid int64 = 5

	// exitTailCall indicates the dispatch loop exited to perform a
	// tail call via Go.
	exitTailCall int64 = 6

	// exitCallOverflow indicates the dispatch loop exited because the
	// call depth limit was exceeded.
	exitCallOverflow int64 = 7

	// maxASMIntArgs is the maximum number of int arguments the ASM
	// fast-path call handler can copy.
	maxASMIntArgs = 8

	// maxASMFloatArgs is the maximum number of float arguments the ASM
	// fast-path call handler can copy.
	maxASMFloatArgs = 8

	// maxASMStringArgs is the maximum number of string arguments the
	// ASM fast-path call handler can copy.
	maxASMStringArgs = 8

	// maxASMBoolArgs is the maximum number of bool arguments the ASM
	// fast-path call handler can copy.
	maxASMBoolArgs = 8

	// maxASMUintArgs is the maximum number of uint arguments the ASM
	// fast-path call handler can copy.
	maxASMUintArgs = 8
)

// asmCallInfo holds pre-computed metadata for one call site, enabling
// the ASM dispatch loop to handle interpreted-to-interpreted calls
// without
// trampolining to Go. Built once per function by buildASMCallInfoTables.
//
// Field offsets are hardcoded in the ASM handlers and verified by
// TestASMCallInfoOffsets.
type asmCallInfo struct {
	// calleeFunction is the unsafe pointer to the callee CompiledFunction.
	calleeFunction uintptr

	// calleeBody is the pointer to the first instruction of the callee
	// function body.
	calleeBody uintptr

	// calleeBodyLength is the number of instructions in the callee
	// function body.
	calleeBodyLength int64

	// calleeIntConstants is the pointer to the callee function's int
	// constant table.
	calleeIntConstants uintptr

	// calleeFloatConstants is the pointer to the callee function's
	// float constant table.
	calleeFloatConstants uintptr

	// calleeIntCount is the number of int registers required by the
	// callee function.
	calleeIntCount int64

	// calleeFloatCount is the number of float registers required by
	// the callee function.
	calleeFloatCount int64

	// intArgumentCount is the number of integer arguments to copy
	// from caller to callee.
	intArgumentCount int64

	// intArgumentSources holds the caller int register index for each
	// integer argument.
	intArgumentSources [8]int64

	// floatArgumentCount is the number of float arguments to copy
	// from caller to callee.
	floatArgumentCount int64

	// floatArgumentSources holds the caller float register index for
	// each float argument.
	floatArgumentSources [8]int64

	// returnCount is the number of return values, either zero or one.
	returnCount int64

	// returnDestinationKind is the register kind for the return
	// destination, int or float.
	returnDestinationKind int64

	// returnDestinationReg is the caller register index for storing
	// the return value.
	returnDestinationReg int64

	// returnDestinationPtr is the pointer to the first return location descriptor.
	returnDestinationPtr uintptr

	// returnDestinationLen is the number of entries in the return
	// location descriptor slice.
	returnDestinationLen int64

	// calleeCallInfo is the pointer to the callee function's
	// asmCallInfo table base.
	calleeCallInfo uintptr

	// isFastPath indicates the ASM inline dispatch mode.
	//
	//   0 = not eligible (Go fallback)
	//   1 = eligible, callee uses string/bool/uint registers
	//   2 = eligible, callee uses only int/float (lean path)
	isFastPath int64

	// calleeStringCount is the number of string registers required by
	// the callee function.
	calleeStringCount int64

	// calleeBoolCount is the number of bool registers required by the
	// callee function.
	calleeBoolCount int64

	// calleeUintCount is the number of uint registers required by the
	// callee function.
	calleeUintCount int64

	// stringArgumentCount is the number of string arguments to copy
	// from caller to callee.
	stringArgumentCount int64

	// stringArgumentSources holds the caller string register index for
	// each string argument.
	stringArgumentSources [8]int64

	// boolArgumentCount is the number of bool arguments to copy from
	// caller to callee.
	boolArgumentCount int64

	// boolArgumentSources holds the caller bool register index for
	// each bool argument.
	boolArgumentSources [8]int64

	// uintArgumentCount is the number of uint arguments to copy from
	// caller to callee.
	uintArgumentCount int64

	// uintArgumentSources holds the caller uint register index for
	// each uint argument.
	uintArgumentSources [8]int64

	// _padding ensures the struct size is exactly 512 bytes (power of 2).
	_padding [2]int64
}

// asmDispatchSave stores the dispatch register values that must be
// preserved across inline call/return. Saved by the call handler
// and restored by the return handler.
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

	// floatConstantsBase is the pointer to the saved function's float
	// constant table.
	floatConstantsBase uintptr
}

// buildDispatchContext populates a DispatchContext from the current
// VM frame state. The context is only valid for the lifetime of the
// current frame; after opCall or opReturn it must be rebuilt.
//
// Takes ctx (*DispatchContext) which is the dispatch context struct to
// populate with current frame state.
// Takes jumpTable (*[opcodeTableSize]uintptr) which is the opcode
// dispatch table mapping opcodes to handler addresses.
func (vm *VM) buildDispatchContext(ctx *DispatchContext, jumpTable *[opcodeTableSize]uintptr) {
	frame := &vm.callStack[vm.framePointer]
	body := frame.function.body
	registers := &frame.registers

	if len(body) > 0 {
		ctx.codeBase = uintptr(unsafe.Pointer(&body[0]))
	}
	ctx.codeLength = int64(len(body))
	ctx.programCounter = int64(frame.programCounter)

	if len(registers.ints) > 0 {
		ctx.intsBase = uintptr(unsafe.Pointer(&registers.ints[0]))
	}
	ctx.intsLength = int64(len(registers.ints))

	if len(registers.floats) > 0 {
		ctx.floatsBase = uintptr(unsafe.Pointer(&registers.floats[0]))
	}
	ctx.floatsLength = int64(len(registers.floats))

	if len(frame.function.intConstants) > 0 {
		ctx.intConstantsBase = uintptr(unsafe.Pointer(&frame.function.intConstants[0]))
	}
	ctx.intConstantsLength = int64(len(frame.function.intConstants))

	if len(frame.function.floatConstants) > 0 {
		ctx.floatConstantsBase = uintptr(unsafe.Pointer(&frame.function.floatConstants[0]))
	}
	ctx.floatConstantsLength = int64(len(frame.function.floatConstants))

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

// populateExtendedBases writes string, uint, and bool register base
// pointers plus ASM metadata pointers into the dispatch context.
//
// Takes ctx (*DispatchContext) which is the dispatch context to
// populate with extended register base pointers.
// Takes registers (*Registers) which provides the string, uint, and
// bool register slices.
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

	if len(vm.asmCallInfoBases) > 0 {
		ctx.asmCallInfoBasesPointer = uintptr(unsafe.Pointer(&vm.asmCallInfoBases[0]))
		ctx.asmCallInfoBase = vm.asmCallInfoBases[vm.framePointer]
	}
	if len(vm.asmDispatchSaves) > 0 {
		ctx.dispatchSavesPointer = uintptr(unsafe.Pointer(&vm.asmDispatchSaves[0]))
	}
}

// populateArenaContext writes the arena slab pointers and indices
// into the dispatch context so that ASM can allocate registers inline.
//
// Takes ctx (*DispatchContext) which is the dispatch context to
// populate with arena state.
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
}

// syncCallContextFromASM updates VM state from the DispatchContext
// after the ASM loop returns. ASM may have modified fp, arenaIntIndex,
// and arenaFloatIndex via inline call/return.
//
// Takes ctx (*DispatchContext) which is the dispatch context containing
// the updated ASM state.
func (vm *VM) syncCallContextFromASM(ctx *DispatchContext) {
	vm.framePointer = int(ctx.framePointer)
	if vm.arena != nil {
		vm.arena.intIndex = int(ctx.arenaIntIndex)
		vm.arena.floatIndex = int(ctx.arenaFloatIndex)
		vm.arena.stringIndex = int(ctx.arenaStringIndex)
		vm.arena.boolIndex = int(ctx.arenaBoolIndex)
		vm.arena.uintIndex = int(ctx.arenaUintIndex)
	}
}

// refreshCallContext updates the call-related fields in the
// DispatchContext after a Go-side frame change (push/pop).
//
// Takes ctx (*DispatchContext) which is the dispatch context to
// refresh with current call stack state.
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

// refreshArenaSlabs updates the arena slab pointers, capacities, and
// indices in the dispatch context from the current arena state.
//
// Takes ctx (*DispatchContext) which is the dispatch context to
// refresh with current arena slab state.
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
}

// saveCurrentDispatchRegisters writes the current frame's dispatch
// register values into dispSaves[vm.framePointer]. This must be called
// before entering the ASM dispatch loop so that the inline return
// handler can restore the caller's dispatch state even when the call
// went through Go.
//
// Takes ctx (*DispatchContext) which is the dispatch context containing
// the current register values to save.
func (vm *VM) saveCurrentDispatchRegisters(ctx *DispatchContext) {
	if vm.asmDispatchSaves != nil && vm.framePointer >= 0 && vm.framePointer < len(vm.asmDispatchSaves) {
		save := &vm.asmDispatchSaves[vm.framePointer]
		save.codeBase = ctx.codeBase
		save.codeLength = ctx.codeLength
		save.intConstantsBase = ctx.intConstantsBase
		save.floatConstantsBase = ctx.floatConstantsBase
	}
}

// buildASMCallInfoTables pre-computes asmCallInfo tables for all
// functions in the program.
//
// Takes rootFunction (*CompiledFunction) which is the entry-point
// function for the program.
// Takes functions ([]*CompiledFunction) which is the complete list of
// compiled functions in the program.
//
// Returns a map from function pointer to its asmCallInfo table, and
// the root function's table for direct use by the dispatch loop.
func buildASMCallInfoTables(rootFunction *CompiledFunction, functions []*CompiledFunction) (map[*CompiledFunction][]asmCallInfo, []asmCallInfo) {
	tables := make(map[*CompiledFunction][]asmCallInfo)
	buildASMCallInfoTableFor(rootFunction, functions, tables)
	for _, compiledFunction := range functions {
		if _, ok := tables[compiledFunction]; !ok {
			buildASMCallInfoTableFor(compiledFunction, functions, tables)
		}
	}
	for compiledFunction, table := range tables {
		for i := range table {
			if table[i].isFastPath == 0 {
				continue
			}
			site := &compiledFunction.callSites[i]
			callee := functions[site.funcIndex]
			if calleeTable, ok := tables[callee]; ok && len(calleeTable) > 0 {
				table[i].calleeCallInfo = uintptr(unsafe.Pointer(&calleeTable[0]))
			}
		}
	}
	return tables, tables[rootFunction]
}

// buildASMCallInfoTableFor builds the asmCallInfo table for a single
// compiled function and stores it in the tables map.
//
// Takes function (*CompiledFunction) which is the function to build
// the table for.
// Takes functions ([]*CompiledFunction) which is the complete list of
// compiled functions for callee resolution.
// Takes tables (map[*CompiledFunction][]asmCallInfo) which is the map
// to store the resulting table in.
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

// buildOneASMCallInfo populates a single asmCallInfo entry for a call
// site if it is eligible for ASM fast-path dispatch.
//
// Takes info (*asmCallInfo) which is the asmCallInfo entry to populate.
// Takes site (*callSite) which is the call site descriptor from the
// caller function.
// Takes functions ([]*CompiledFunction) which is the complete list of
// compiled functions for callee resolution.
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

// resolveASMCallee returns the callee function if the call site is
// eligible for ASM dispatch, or nil if it should be skipped.
//
// Takes site (*callSite) which is the call site descriptor to evaluate
// for eligibility.
// Takes functions ([]*CompiledFunction) which is the complete list of
// compiled functions for index lookup.
//
// Returns the resolved callee CompiledFunction, or nil if the call site
// is ineligible due to closures, native calls, variadic signatures, or
// non-int/float registers.
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
	if callee.numRegisters[registerGeneral] > 0 ||
		callee.numRegisters[registerComplex] > 0 {
		return nil
	}
	return callee
}

// maxASMArgsByKind maps each register kind to its maximum ASM argument
// count. Unsupported kinds (general, complex) have zero capacity.
var maxASMArgsByKind = [7]int{
	registerInt:     maxASMIntArgs,
	registerFloat:   maxASMFloatArgs,
	registerString:  maxASMStringArgs,
	registerGeneral: 0,
	registerBool:    maxASMBoolArgs,
	registerUint:    maxASMUintArgs,
	registerComplex: 0,
}

// asmArgumentSourceSlice returns the argument source array for the
// given register kind, or nil for unsupported kinds.
//
// Takes info (*asmCallInfo) which is the call info entry containing
// the per-kind argument source arrays.
// Takes kind (registerKind) which selects the register kind whose
// source array to return.
//
// Returns []int64 which is the argument source slice for the given
// kind, or nil if the kind is unsupported.
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
	default:
		return nil
	}
}

// mapASMArguments maps the call site's arguments to the ASM info's
// source arrays and writes the per-kind argument counts into info.
//
// Takes info (*asmCallInfo) which is the asmCallInfo entry to populate
// with argument source indices and counts.
// Takes site (*callSite) which is the call site descriptor containing
// the argument locations.
// Takes callee (*CompiledFunction) which is the target function whose
// parameter kinds must match.
//
// Returns true if all arguments were successfully mapped, or false if
// any argument kind does not match or exceeds the maximum count.
func mapASMArguments(info *asmCallInfo, site *callSite, callee *CompiledFunction) bool {
	var counts [7]int
	for argumentIndex, argumentLocation := range site.arguments {
		if argumentIndex >= len(callee.paramKinds) {
			break
		}
		paramKind := callee.paramKinds[argumentIndex]
		if argumentLocation.kind != paramKind {
			return false
		}
		sources := asmArgumentSourceSlice(info, paramKind)
		if sources == nil || counts[paramKind] >= maxASMArgsByKind[paramKind] {
			return false
		}
		sources[counts[paramKind]] = int64(argumentLocation.register)
		counts[paramKind]++
	}
	info.intArgumentCount = int64(counts[registerInt])
	info.floatArgumentCount = int64(counts[registerFloat])
	info.stringArgumentCount = int64(counts[registerString])
	info.boolArgumentCount = int64(counts[registerBool])
	info.uintArgumentCount = int64(counts[registerUint])
	return true
}

// configureASMReturn validates and configures the return destination
// for ASM fast-path dispatch.
//
// Takes info (*asmCallInfo) which is the asmCallInfo entry to populate
// with return metadata.
// Takes site (*callSite) which is the call site descriptor containing
// return location descriptors.
// Takes callee (*CompiledFunction) which is the target function whose
// result kinds are validated.
//
// Returns true if the return shape is compatible with ASM dispatch, or
// false if there are multiple returns or incompatible register kinds.
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
	info.returnDestinationReg = int64(returnLocation.register)
	return true
}

// populateASMCalleeFields fills the callee-specific pointer and size
// fields that the ASM dispatch loop needs to set up an inline call
// frame. Argument counts must already be written to info by
// mapASMArguments.
//
// Takes info (*asmCallInfo) which is the asmCallInfo entry to populate
// with callee pointers and sizes.
// Takes site (*callSite) which is the call site descriptor containing
// return location data.
// Takes callee (*CompiledFunction) which is the target compiled
// function providing body and constant pointers.
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
	info.calleeIntCount = int64(callee.numRegisters[registerInt])
	info.calleeFloatCount = int64(callee.numRegisters[registerFloat])
	info.calleeStringCount = int64(callee.numRegisters[registerString])
	info.calleeBoolCount = int64(callee.numRegisters[registerBool])
	info.calleeUintCount = int64(callee.numRegisters[registerUint])
	if len(site.returns) > 0 {
		info.returnDestinationPtr = uintptr(unsafe.Pointer(&site.returns[0]))
		info.returnDestinationLen = int64(len(site.returns))
	}
	if callee.numRegisters[registerString] == 0 &&
		callee.numRegisters[registerBool] == 0 &&
		callee.numRegisters[registerUint] == 0 {
		info.isFastPath = 2
	} else {
		info.isFastPath = 1
	}
}
