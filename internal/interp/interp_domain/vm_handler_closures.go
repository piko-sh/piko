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

import "reflect"

// handleMakeClosure creates a runtime closure value by capturing upvalues
// from the current frame and storing the result in the destination register.
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
		if cached, ok := vm.closureCache[funcIndex]; ok {
			registers.general[instruction.a] = cached
			return opContinue
		}
		reflectValue := reflect.ValueOf(&runtimeClosure{function: compiledFunction})
		if vm.closureCache == nil {
			vm.closureCache = make(map[uint16]reflect.Value)
		}
		vm.closureCache[funcIndex] = reflectValue
		registers.general[instruction.a] = reflectValue
		return opContinue
	}

	parentFrame := &vm.callStack[vm.framePointer]
	if parentFrame.sharedCells == nil {
		parentFrame.sharedCells = make(map[uint16]*upvalueCell, len(compiledFunction.upvalueDescriptors))
	}
	cells := make([]*upvalueCell, len(compiledFunction.upvalueDescriptors))
	for i, descriptor := range compiledFunction.upvalueDescriptors {
		if !descriptor.isLocal && parentFrame.upvalues != nil {
			cells[i] = parentFrame.upvalues[descriptor.index].value
			continue
		}
		key := joinWide(uint8(descriptor.kind), descriptor.index)
		if existing, ok := parentFrame.sharedCells[key]; ok {
			cells[i] = existing
			continue
		}
		cell := &upvalueCell{kind: descriptor.kind}
		switch descriptor.kind {
		case registerInt:
			cell.intValue = registers.ints[descriptor.index]
		case registerFloat:
			cell.floatValue = registers.floats[descriptor.index]
		case registerString:
			cell.stringValue = materialiseString(vm.arena, registers.strings[descriptor.index])
		case registerGeneral:
			cell.generalValue = registers.general[descriptor.index]
		case registerBool:
			cell.boolValue = registers.bools[descriptor.index]
		case registerUint:
			cell.uintValue = registers.uints[descriptor.index]
		case registerComplex:
			cell.complexValue = registers.complex[descriptor.index]
		}
		cells[i] = cell
		parentFrame.sharedCells[key] = cell
	}
	registers.general[instruction.a] = reflect.ValueOf(&runtimeClosure{function: compiledFunction, upvalues: cells})
	return opContinue
}

// handleGetUpvalue reads a captured upvalue cell and copies its value into
// the appropriate typed register bank of the current frame.
//
// Takes frame (*callFrame) which provides the upvalue array.
// Takes registers (*Registers) which is the destination register set.
// Takes instruction (instruction) which encodes the upvalue and register.
//
// Returns opResult indicating the next execution step.
func handleGetUpvalue(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	cell := frame.upvalues[instruction.b].value
	switch registerKind(instruction.c) {
	case registerInt:
		registers.ints[instruction.a] = cell.intValue
	case registerFloat:
		registers.floats[instruction.a] = cell.floatValue
	case registerString:
		registers.strings[instruction.a] = cell.stringValue
	case registerGeneral:
		registers.general[instruction.a] = cell.generalValue
	case registerBool:
		registers.bools[instruction.a] = cell.boolValue
	case registerUint:
		registers.uints[instruction.a] = cell.uintValue
	case registerComplex:
		registers.complex[instruction.a] = cell.complexValue
	}
	return opContinue
}

// handleSetUpvalue writes a register value into a captured upvalue cell,
// materialising arena strings to ensure they outlive the current frame.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes frame (*callFrame) which provides the upvalue array.
// Takes registers (*Registers) which holds the source register banks.
// Takes instruction (instruction) which encodes the upvalue and register.
//
// Returns opResult indicating the next execution step.
func handleSetUpvalue(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	cell := frame.upvalues[instruction.b].value
	writeRegisterToCell(vm.arena, cell, registers, registerKind(instruction.c), instruction.a)
	return opContinue
}

// writeRegisterToCell copies the value from the specified register
// bank at the given index into the upvalue cell.
//
// Takes arena (*RegisterArena) which provides string
// materialisation.
// Takes cell (*upvalueCell) which is the destination upvalue cell.
// Takes registers (*Registers) which holds the source register
// banks.
// Takes kind (registerKind) which selects the register bank to
// read from.
// Takes registerIndex (byte) which is the index within the
// selected register bank.
func writeRegisterToCell(arena *RegisterArena, cell *upvalueCell, registers *Registers, kind registerKind, registerIndex byte) {
	switch kind {
	case registerInt:
		cell.intValue = registers.ints[registerIndex]
	case registerFloat:
		cell.floatValue = registers.floats[registerIndex]
	case registerString:
		cell.stringValue = materialiseString(arena, registers.strings[registerIndex])
	case registerGeneral:
		cell.generalValue = registers.general[registerIndex]
	case registerBool:
		cell.boolValue = registers.bools[registerIndex]
	case registerUint:
		cell.uintValue = registers.uints[registerIndex]
	case registerComplex:
		cell.complexValue = registers.complex[registerIndex]
	}
}

// handleSyncClosureUpvalues synchronises upvalue cells back to the caller's
// registers after a callee has modified shared captured variables.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the caller's register banks.
// Takes instruction (instruction) which selects the sync mode and register.
//
// Returns opResult indicating the next execution step.
func handleSyncClosureUpvalues(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	if instruction.b == 1 {
		syncCalleeUpvalues(vm, registers)
	} else {
		syncClosureSharedUpvalues(vm, registers, instruction.a)
	}
	return opContinue
}

// syncCalleeUpvalues copies modified upvalue cells from the just-returned
// callee frame back into the caller's registers (post-IIFE writeback).
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

// syncClosureSharedUpvalues writes back upvalue cells that are shared between
// the caller frame and a closure, updating the caller's registers with any
// values mutated by the closure during execution.
//
// Takes vm (*VM) which provides access to the call stack.
// Takes registers (*Registers) which is the caller's register set.
// Takes closureReg (uint8) which holds the closure to sync upvalues for.
func syncClosureSharedUpvalues(vm *VM, registers *Registers, closureReg uint8) {
	closure, ok := reflect.TypeAssert[*runtimeClosure](registers.general[closureReg])
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
		if _, shared := callerFrame.sharedCells[key]; !shared {
			continue
		}
		syncUpvalueCellToRegister(registers, descriptor, closure.upvalues[i])
	}
}

// syncUpvalueCellToRegister copies the value from an upvalue cell back into
// the caller's register at the index described by the upvalue descriptor.
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
	}
}

// handleResetSharedCell removes a shared upvalue cell entry so that
// subsequent writes to that register no longer propagate to closures.
//
// Takes frame (*callFrame) which holds the shared cell map to update.
// Takes instruction (instruction) which encodes the kind and register index.
//
// Returns opResult indicating the next execution step.
func handleResetSharedCell(_ *VM, frame *callFrame, _ *Registers, instruction instruction) opResult {
	if frame.sharedCells != nil {
		key := joinWide(instruction.b, instruction.a)
		delete(frame.sharedCells, key)
	}
	return opContinue
}

// handleWriteSharedCell copies the current register value into the
// corresponding shared upvalue cell, keeping the cell in sync after
// a parent-frame write to a captured variable.
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
	cell, ok := frame.sharedCells[key]
	if !ok {
		return opContinue
	}
	writeRegisterToCell(vm.arena, cell, registers, registerKind(instruction.b), instruction.a)
	return opContinue
}
