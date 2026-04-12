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

//go:build safe || (js && wasm)

package interp_domain

// handleSubOpIncStructFieldInt increments an int-typed struct field via the reflect
// fallback path.
//
// Takes vm (*VM) which owns the VM state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which is the register bank.
// Takes instr (instruction) which carries the field layout in C and the receiver register
// in B.
//
// Returns opResult signalling dispatch continuation.
func handleSubOpIncStructFieldInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instr.c)
	return incDecStructFieldIntFallback(vm, frame, registers, instr, layout, +1)
}

// handleSubOpDecStructFieldInt decrements an int-typed struct field via the reflect
// fallback path.
//
// Takes vm (*VM) which owns the VM state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which is the register bank.
// Takes instr (instruction) which carries the field layout in C and the receiver register
// in B.
//
// Returns opResult signalling dispatch continuation.
func handleSubOpDecStructFieldInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instr.c)
	return incDecStructFieldIntFallback(vm, frame, registers, instr, layout, -1)
}

// handleSubOpIncStructFieldUint increments a uint-typed struct field via the reflect
// fallback path.
//
// Takes vm (*VM) which owns the VM state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which is the register bank.
// Takes instr (instruction) which carries the field layout in C and the receiver register
// in B.
//
// Returns opResult signalling dispatch continuation.
func handleSubOpIncStructFieldUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instr.c)
	return incDecStructFieldUintFallback(vm, frame, registers, instr, layout, +1)
}

// handleSubOpDecStructFieldUint decrements a uint-typed struct field via the reflect
// fallback path.
//
// Takes vm (*VM) which owns the VM state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which is the register bank.
// Takes instr (instruction) which carries the field layout in C and the receiver register
// in B.
//
// Returns opResult signalling dispatch continuation.
func handleSubOpDecStructFieldUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	layout := readTier0StructFieldLayout(frame, instr.c)
	return incDecStructFieldUintFallback(vm, frame, registers, instr, layout, -1)
}

// incDecStructFieldIntFallback performs the reflect-walk write that adjusts an int-typed
// struct field by delta.
//
// Takes registers (*Registers) which is the register bank holding the receiver value.
// Takes instr (instruction) which carries the receiver register in B.
// Takes layout (structFieldLayout) which is the resolved field layout.
// Takes delta (int64) which is the signed adjustment to apply.
//
// Returns opResult signalling dispatch continuation; bails to opContinue if the receiver
// cannot be addressed.
func incDecStructFieldIntFallback(_ *VM, frame *callFrame, registers *Registers, instr instruction, layout structFieldLayout, delta int64) opResult {
	field, ok := structFieldReflectWrite(registers, frame, instr.b, layout)
	if !ok {
		return opContinue
	}
	field.SetInt(field.Int() + delta)
	return opContinue
}

// incDecStructFieldUintFallback performs the reflect-walk write that adjusts a uint-typed
// struct field by delta.
//
// Takes registers (*Registers) which is the register bank holding the receiver value.
// Takes instr (instruction) which carries the receiver register in B.
// Takes layout (structFieldLayout) which is the resolved field layout.
// Takes delta (int64) which is the signed adjustment to apply (cast internally to uint64
// wrap-around semantics).
//
// Returns opResult signalling dispatch continuation; bails to opContinue if the receiver
// cannot be addressed.
func incDecStructFieldUintFallback(_ *VM, frame *callFrame, registers *Registers, instr instruction, layout structFieldLayout, delta int64) opResult {
	field, ok := structFieldReflectWrite(registers, frame, instr.b, layout)
	if !ok {
		return opContinue
	}
	current := field.Uint()
	if delta >= 0 {
		field.SetUint(current + 1)
	} else {
		field.SetUint(current - 1)
	}
	return opContinue
}
