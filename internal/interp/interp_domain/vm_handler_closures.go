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
	"reflect"
)

const (
	// upvalueKindAsPointer is the destinationKind sentinel for an "address of upvalue" read.
	//
	// The single-byte operand encodes a registerKind; the reserved 0xFF value instructs
	// readIndirectCellValue to store the cell's *T pointer directly into the destination
	// general register without dereferencing. Used by compileAddressOfUpvalue so &upvalue
	// inside a closure resolves to a pointer that aliases the parent's heap cell.
	upvalueKindAsPointer registerKind = 0xFF
)

// handleMakeClosure creates a runtime closure value by capturing upvalues from the
// current frame and storing the result in the destination register.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the current register banks.
// Takes instruction (instruction) which encodes the function index.
//
// Returns opResult indicating the next execution step.
func handleMakeClosure(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	funcIndex := instruction.wideIndex()
	if int(funcIndex) >= len(vm.functions) {
		vmBoundsError(vm, frame, boundsTableFunction, int(funcIndex), len(vm.functions))
		return opPanicError
	}
	compiledFunction := vm.functions[funcIndex]
	if len(compiledFunction.upvalueDescriptors) == 0 {
		return emitCachedZeroUpvalueClosure(vm, registers, instruction.a, funcIndex, compiledFunction)
	}
	closure := buildCapturingClosure(vm, registers, compiledFunction)
	registers.general[instruction.a] = reflect.ValueOf(closure)
	return opContinue
}

// emitCachedZeroUpvalueClosure stores the per-funcIndex cached reflect.Value for a
// zero-upvalue closure into the destination register, populating the cache on first use.
// Zero-upvalue closures are immutable so a single shared *runtimeClosure can back every
// invocation.
//
// Takes vm (*VM) which owns the closure cache.
// Takes registers (*Registers) which receives the cached value.
// Takes destinationRegister (uint8) which is the target general-bank slot.
// Takes funcIndex (uint16) which selects the function in vm.functions.
// Takes compiledFunction (*CompiledFunction) which is the function to wrap on first use.
//
// Returns opContinue once the destination register holds the cached closure value.
func emitCachedZeroUpvalueClosure(vm *VM, registers *Registers, destinationRegister uint8, funcIndex uint16, compiledFunction *CompiledFunction) opResult {
	if int(funcIndex) < len(vm.closureCache) {
		if cached := vm.closureCache[funcIndex]; cached.IsValid() {
			registers.general[destinationRegister] = cached
			return opContinue
		}
	} else {
		grown := make([]reflect.Value, len(vm.functions))
		copy(grown, vm.closureCache)
		vm.closureCache = grown
	}
	reflectValue := reflect.ValueOf(&runtimeClosure{function: compiledFunction, rootFunction: vm.rootFunction})
	vm.closureCache[funcIndex] = reflectValue
	registers.general[destinationRegister] = reflectValue
	return opContinue
}

// buildCapturingClosure allocates and populates a *runtimeClosure for a function that
// captures at least one upvalue.
//
// Inline backing storage is used when the capture count fits inlineUpvalueCells. The
// parent frame's sharedCells dedup map is rented from the pool on first use.
//
// Takes vm (*VM) which owns the call stack and pools.
// Takes registers (*Registers) which provides captured values for local upvalues.
// Takes compiledFunction (*CompiledFunction) which describes the captured upvalues.
//
// Returns the populated *runtimeClosure ready for invocation.
func buildCapturingClosure(vm *VM, registers *Registers, compiledFunction *CompiledFunction) *runtimeClosure {
	parentFrame := &vm.callStack[vm.framePointer]
	if parentFrame.sharedCells == nil {
		parentFrame.sharedCells = acquireSharedCellMap()
	}
	closure := &runtimeClosure{function: compiledFunction, rootFunction: vm.rootFunction}
	upvalueCount := len(compiledFunction.upvalueDescriptors)
	if upvalueCount <= inlineUpvalueCells {
		closure.upvalues = closure.inlineCells[:upvalueCount]
	} else {
		closure.upvalues = make([]*upvalueCell, upvalueCount)
	}
	for i, descriptor := range compiledFunction.upvalueDescriptors {
		closure.upvalues[i] = resolveUpvalueCell(vm, registers, parentFrame, descriptor)
	}
	return closure
}

// resolveUpvalueCell returns the upvalueCell for one descriptor, reusing the parent
// frame's upvalue when the descriptor is non-local, otherwise looking up or constructing
// the cell via the parent's sharedCells dedup map.
//
// Takes vm (*VM) which owns the call stack and registers.
// Takes registers (*Registers) which sources captured local values.
// Takes parentFrame (*callFrame) which owns the dedup map.
// Takes descriptor (UpvalueDescriptor) which identifies the capture.
//
// Returns the resolved *upvalueCell for the descriptor.
func resolveUpvalueCell(vm *VM, registers *Registers, parentFrame *callFrame, descriptor UpvalueDescriptor) *upvalueCell {
	if !descriptor.isLocal && parentFrame.upvalues != nil {
		return parentFrame.upvalues[descriptor.index].value
	}
	key := joinWide(uint8(descriptor.kind), descriptor.index)
	if existing, ok := parentFrame.sharedCells.get(key); ok {
		return existing
	}
	cell := buildUpvalueCellFromDescriptor(vm, registers, descriptor)
	parentFrame.sharedCells.set(key, cell)
	return cell
}

// buildUpvalueCellFromDescriptor builds an upvalueCell for a capture.
//
// Indirect descriptors carry the *T heap pointer in generalValue and record the
// originalKind so closure-side reads can deref to the right typed bank; direct
// descriptors snapshot the parent register's value into the matching cell field.
// Centralising the per-kind dispatch keeps handleMakeClosure under the
// cognitive-complexity budget.
//
// Takes vm (*VM) which provides the arena for string materialisation.
// Takes registers (*Registers) which is the parent frame's register set being captured.
// Takes descriptor (UpvalueDescriptor) which describes which register to read and the
// closure-side cell semantics.
//
// Returns the populated upvalueCell ready to be inserted into the closure's cells slice
// and the parent frame's sharedCells map.
func buildUpvalueCellFromDescriptor(vm *VM, registers *Registers, descriptor UpvalueDescriptor) *upvalueCell {
	cell := &upvalueCell{kind: descriptor.kind}
	if descriptor.isIndirect {
		cell.isIndirect = true
		cell.originalKind = descriptor.originalKind
		cell.generalValue = registers.general[descriptor.index]
		return cell
	}
	switch descriptor.kind {
	case registerInt:
		cell.intValue = registers.ints[descriptor.index]
	case registerFloat:
		cell.floatValue = registers.floats[descriptor.index]
	case registerString:
		cell.stringValue = materialiseStringUnconditional(vm.arena, registers.strings[descriptor.index])
	case registerGeneral:
		cell.generalValue = materialiseArenaValueUnconditional(vm.arena, registers.general[descriptor.index])
	case registerBool:
		cell.boolValue = registers.bools[descriptor.index]
	case registerUint:
		cell.uintValue = registers.uints[descriptor.index]
	case registerComplex:
		cell.complexValue = registers.complex[descriptor.index]
	case registerSliceInt:
		cell.sliceIntValue = registers.slicesInt[descriptor.index]
	case registerSliceFloat:
		cell.sliceFloatValue = registers.slicesFloat[descriptor.index]
	case registerSliceString:
		cell.sliceStringValue = registers.slicesString[descriptor.index]
	case registerSliceBool:
		cell.sliceBoolValue = registers.slicesBool[descriptor.index]
	case registerSliceUint:
		cell.sliceUintValue = registers.slicesUint[descriptor.index]
	case registerSliceByte:
		cell.sliceByteValue = registers.slicesByte[descriptor.index]
	default:
	}
	return cell
}

// handleGetUpvalue reads a captured upvalue cell and copies its value into the
// appropriate typed register bank of the current frame.
//
// Takes frame (*callFrame) which provides the upvalue array.
// Takes registers (*Registers) which is the destination register set.
// Takes instruction (instruction) which encodes the upvalue and register.
//
// Returns opResult indicating the next execution step.
func handleGetUpvalue(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	cell := frame.upvalues[instruction.b].value
	if cell.isIndirect {
		readIndirectCellValue(registers, cell, instruction.a, registerKind(instruction.c))
		return opContinue
	}
	syncCellToRegister(registers, cell, varLocation{register: instruction.a, kind: registerKind(instruction.c)})
	return opContinue
}

// readIndirectCellValue dereferences an indirect cell's *T pointer.
//
// Stores the pointed-to value into the destination register's typed bank, unless
// destinationKind is the upvalueKindAsPointer sentinel. The destination kind names the
// bank the closure body expects; for general-bank reads we keep the dereferenced reflect
// value as-is so methods on the captured value see the heap memory, otherwise we unbox
// via .Int()/.Float()/etc.
//
// Takes registers (*Registers) which is the destination register set.
// Takes cell (*upvalueCell) which is the indirect cell carrying the pointer.
// Takes destination (uint8) which is the destination register index.
// Takes destinationKind (registerKind) which selects the bank, or upvalueKindAsPointer to
// copy the cell's *T pointer untouched.
func readIndirectCellValue(registers *Registers, cell *upvalueCell, destination uint8, destinationKind registerKind) {
	pointer := cell.generalValue
	if !pointer.IsValid() {
		return
	}
	if destinationKind == upvalueKindAsPointer {
		registers.general[destination] = pointer
		return
	}
	target := pointer.Elem()
	switch destinationKind {
	case registerInt:
		registers.ints[destination] = target.Int()
	case registerFloat:
		registers.floats[destination] = target.Float()
	case registerString:
		registers.strings[destination] = target.String()
	case registerGeneral:
		registers.general[destination] = target
	case registerBool:
		registers.bools[destination] = target.Bool()
	case registerUint:
		registers.uints[destination] = target.Uint()
	case registerComplex:
		registers.complex[destination] = target.Complex()
	default:
	}
}

// handleSetUpvalue writes a register value into a captured upvalue cell, materialising
// arena strings to ensure they outlive the current frame.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the upvalue array.
// Takes registers (*Registers) which holds the source register banks.
// Takes instruction (instruction) which encodes the upvalue and register.
//
// Returns opResult indicating the next execution step.
func handleSetUpvalue(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	cell := frame.upvalues[instruction.b].value
	writeRegisterToCell(vm, vm.arena, cell, registers, registerKind(instruction.c), instruction.a)
	return opContinue
}

// writeRegisterToCell copies the value from the specified register bank at the given
// index into the upvalue cell.
//
// Takes arena (*RegisterArena) which provides string materialisation.
// Takes cell (*upvalueCell) which is the destination upvalue cell.
// Takes registers (*Registers) which holds the source register banks.
// Takes kind (registerKind) which selects the register bank to read from.
// Takes registerIndex (byte) which is the index within the selected register bank.
func writeRegisterToCell(vm *VM, arena *RegisterArena, cell *upvalueCell, registers *Registers, kind registerKind, registerIndex byte) {
	if cell.isIndirect {
		writeRegisterThroughIndirectCell(vm, arena, cell, registers, kind, registerIndex)
		return
	}
	switch kind {
	case registerInt:
		cell.intValue = registers.ints[registerIndex]
	case registerFloat:
		cell.floatValue = registers.floats[registerIndex]
	case registerString:
		cell.stringValue = materialiseStringUnconditional(arena, registers.strings[registerIndex])
	case registerGeneral:
		cell.generalValue = materialiseArenaValueUnconditional(arena, registers.general[registerIndex])
		if !arenaUsesUnsafeSlabs {
			cell.generalValue = safeBuildDetachSliceHeader(cell.generalValue)
		}
	case registerBool:
		cell.boolValue = registers.bools[registerIndex]
	case registerUint:
		cell.uintValue = registers.uints[registerIndex]
	case registerComplex:
		cell.complexValue = registers.complex[registerIndex]
	case registerSliceInt:
		cell.sliceIntValue = registers.slicesInt[registerIndex]
	case registerSliceFloat:
		cell.sliceFloatValue = registers.slicesFloat[registerIndex]
	case registerSliceString:
		cell.sliceStringValue = registers.slicesString[registerIndex]
	case registerSliceBool:
		cell.sliceBoolValue = registers.slicesBool[registerIndex]
	case registerSliceUint:
		cell.sliceUintValue = registers.slicesUint[registerIndex]
	case registerSliceByte:
		cell.sliceByteValue = registers.slicesByte[registerIndex]
	default:
	}
}

// writeRegisterThroughIndirectCell stores the source register's value through the cell's
// *T pointer so every reader of the cell observes the new value, both the declaring frame
// and any other closure that captured the same variable. Mirrors readIndirectCellValue:
// the source kind selects the typed bank to read from.
//
// Takes arena (*RegisterArena) which is unused for non-string banks and provides string
// materialisation for strings.
// Takes cell (*upvalueCell) which carries the *T pointer.
// Takes registers (*Registers) which is the source register set.
// Takes kind (registerKind) which selects the source bank.
// Takes registerIndex (byte) which is the source register index.
func writeRegisterThroughIndirectCell(vm *VM, arena *RegisterArena, cell *upvalueCell, registers *Registers, kind registerKind, registerIndex byte) {
	pointer := cell.generalValue
	if !pointer.IsValid() {
		return
	}
	target := pointer.Elem()
	switch kind {
	case registerInt:
		target.SetInt(registers.ints[registerIndex])
	case registerFloat:
		target.SetFloat(registers.floats[registerIndex])
	case registerString:
		target.SetString(materialiseStringUnconditional(arena, registers.strings[registerIndex]))
	case registerGeneral:
		src := materialiseArenaValueUnconditional(arena, coerceValue(vm, registers.general[registerIndex], target.Type()))
		if !arenaUsesUnsafeSlabs {
			src = safeBuildDetachSliceHeader(src)
		}
		target.Set(src)
	case registerBool:
		target.SetBool(registers.bools[registerIndex])
	case registerUint:
		target.SetUint(registers.uints[registerIndex])
	case registerComplex:
		target.SetComplex(registers.complex[registerIndex])
	default:
	}
}

// handleSyncClosureUpvalues synchronises upvalue cells back to the caller's registers
// after a callee has modified shared captured variables.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the caller's register banks.
// Takes instruction (instruction) which selects the sync mode and register.
//
// Returns opResult indicating the next execution step.
func handleSyncClosureUpvalues(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	if instruction.b == 1 {
		syncCalleeUpvalues(vm, registers)
		return opContinue
	}
	if vm.callStack[vm.framePointer].sharedCells == nil {
		return opContinue
	}
	syncClosureSharedUpvalues(vm, registers, instruction.a)
	return opContinue
}

// syncCalleeUpvalues copies modified upvalue cells from the just-returned callee frame
// back into the caller's registers (post-IIFE writeback).
//
// Takes vm (*VM) which provides access to the call stack.
// Takes registers (*Registers) which is the caller's register set.
func syncCalleeUpvalues(vm *VM, registers *Registers) {
	calleeFrame := &vm.callStack[vm.framePointer+1]
	for i, descriptor := range calleeFrame.function.upvalueDescriptors {
		if !descriptor.isLocal {
			continue
		}
		syncUpvalueCellToRegister(registers, descriptor, calleeFrame.upvalues[i].value)
	}
}

// syncClosureSharedUpvalues writes back upvalue cells that are shared between the caller
// frame and a closure, updating the caller's registers with any values mutated by the
// closure during execution.
//
// Takes vm (*VM) which provides access to the call stack.
// Takes registers (*Registers) which is the caller's register set.
// Takes closureRegister (uint8) which holds the closure to sync upvalues for.
func syncClosureSharedUpvalues(vm *VM, registers *Registers, closureRegister uint8) {
	closure, ok := reflect.TypeAssert[*runtimeClosure](registers.general[closureRegister])
	if !ok {
		return
	}
	callerFrame := &vm.callStack[vm.framePointer]
	if callerFrame.sharedCells == nil {
		return
	}
	for i, descriptor := range closure.function.upvalueDescriptors {
		if !descriptor.isLocal {
			continue
		}
		key := joinWide(uint8(descriptor.kind), descriptor.index)
		if !callerFrame.sharedCells.has(key) {
			continue
		}
		syncUpvalueCellToRegister(registers, descriptor, closure.upvalues[i])
	}
}

// syncUpvalueCellToRegister copies the value from an upvalue cell back into the caller's
// register at the index described by the upvalue descriptor.
//
// Takes registers (*Registers) which is the destination register set.
// Takes descriptor (UpvalueDescriptor) which describes the kind and index.
// Takes cell (*upvalueCell) which holds the current upvalue state.
func syncUpvalueCellToRegister(registers *Registers, descriptor UpvalueDescriptor, cell *upvalueCell) {
	switch descriptor.kind {
	case registerInt:
		registers.ints[descriptor.index] = cell.intValue
	case registerFloat:
		registers.floats[descriptor.index] = cell.floatValue
	case registerString:
		registers.strings[descriptor.index] = cell.stringValue
	case registerGeneral:
		registers.general[descriptor.index] = cell.generalValue
	case registerBool:
		registers.bools[descriptor.index] = cell.boolValue
	case registerUint:
		registers.uints[descriptor.index] = cell.uintValue
	case registerComplex:
		registers.complex[descriptor.index] = cell.complexValue
	default:
	}
}

// handleResetSharedCell removes a shared upvalue cell entry so that subsequent writes to
// that register no longer propagate to closures.
//
// Takes frame (*callFrame) which holds the shared cell map to update.
// Takes instruction (instruction) which encodes the kind and register index.
//
// Returns opResult indicating the next execution step.
func handleResetSharedCell(_ *VM, frame *callFrame, _ *Registers, instruction instruction) opResult {
	if frame.sharedCells != nil {
		key := joinWide(instruction.b, instruction.a)
		frame.sharedCells.remove(key)
	}
	return opContinue
}

// handleWriteSharedCell copies the current register value into the corresponding shared
// upvalue cell, keeping the cell in sync after a parent-frame write to a captured
// variable.
//
// Takes vm (*VM) which provides the arena for string materialisation.
// Takes frame (*callFrame) which holds the shared cell map.
// Takes registers (*Registers) which holds the source register banks.
// Takes instruction (instruction) which encodes the register (A) and kind (B).
//
// Returns opResult indicating the next execution step.
func handleWriteSharedCell(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	if frame.sharedCells == nil {
		return opContinue
	}
	key := joinWide(instruction.b, instruction.a)
	cell, ok := frame.sharedCells.get(key)
	if !ok {
		return opContinue
	}
	writeRegisterToCell(vm, vm.arena, cell, registers, registerKind(instruction.b), instruction.a)
	return opContinue
}
