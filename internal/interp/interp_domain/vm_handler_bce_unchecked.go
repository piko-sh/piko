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

//go:build !piko_bce_paranoid

package interp_domain

import (
	"reflect"
)

// Bounds-elided slice and string access handlers. These mirror their checked counterparts
// byte-for-byte except for the dropped bounds check. The compiler emits them only at
// access sites the bounds-check-elimination pass proved are safe (range-loop body,
// post-conditional gate, constant index against a known length).
//
// Soundness depends entirely on the compile-time proof: an out-of- range index reaching
// an unchecked handler is undefined behaviour. Go will panic with index out of range from
// the underlying slice access, but the recovery story is weaker than the checked path's
// explicit vm.evalError. The BCE pass MUST refuse to emit these opcodes unless the proof
// obligations in `function_bce_pass.go` are met.

// handleSliceGetIntDirectUnchecked is the bounds-elided variant of
// handleSliceGetIntDirect. Reads ints[A] = slicesInt[B][ints[C]] without the
// `uint64(index) >= uint64(len(slice))` check.
//
// Takes registers (*Registers) which holds the slicesInt collection and the int index.
// Takes instruction (instruction) which encodes the destination int register A, the
// source slicesInt register B, and the index int register C.
//
// Returns opContinue on success. The Go runtime's slice access will panic if the
// compile-time proof was wrong.
func handleSliceGetIntDirectUnchecked(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = registers.slicesInt[instruction.b][registers.ints[instruction.c]]
	return opContinue
}

// handleSliceSetIntDirectUnchecked is the bounds-elided variant of
// handleSliceSetIntDirect. Writes slicesInt[A][ints[B]] = ints[C] without the runtime
// bounds check.
//
// Takes registers (*Registers) which holds the slicesInt collection, the index, and the
// value.
// Takes instruction (instruction) which encodes the destination slicesInt register A, the
// index int register B, and the value int register C.
//
// Returns opContinue on success.
func handleSliceSetIntDirectUnchecked(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.slicesInt[instruction.a][registers.ints[instruction.b]] = registers.ints[instruction.c]
	return opContinue
}

// handleStringIndexUnchecked is the bounds-elided variant of handleStringIndex. Reads
// uints[A] = uint64(strings[B][ints[C]]) without the runtime range check.
//
// Takes registers (*Registers) which holds the string, index, and destination registers.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opContinue on success.
func handleStringIndexUnchecked(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.uints[instruction.a] = uint64(registers.strings[instruction.b][registers.ints[instruction.c]])
	return opContinue
}

// handleStringIndexToIntUnchecked is the bounds-elided variant of handleStringIndexToInt.
// Reads ints[A] = int64(strings[B][ints[C]]) without the runtime range check.
//
// Takes registers (*Registers) which holds the string, index, and destination registers.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opContinue on success.
func handleStringIndexToIntUnchecked(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(registers.strings[instruction.b][registers.ints[instruction.c]])
	return opContinue
}

// handleSliceGetIntUnchecked is the bounds-elided reflect-bank variant of
// handleSliceGetInt for ints[A] = general[B].Index(ints[C]).Int().
//
// Operates on the reflect (general) register bank without the runtime range check.
//
// Soundness: the BCE pass MUST prove the index is in [0, len(general[B])) before emitting
// this opcode. An out-of-range access lets the underlying reflect.Value.Index call panic
// instead of routing the panic through vm.evalError.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opContinue on success or opPanicError if the collection is not a valid
// indexable value.
func handleSliceGetIntUnchecked(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return indexPanicResult
	}
	element := unwrapInterfaceElement(collection.Index(int(registers.ints[instruction.c])))
	switch element.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		registers.ints[instruction.a] = element.Int()
	default:
		registers.ints[instruction.a] = int64(element.Uint()) //nolint:gosec // intentional reinterpret
	}
	return opContinue
}

// handleSliceSetIntUnchecked is the bounds-elided reflect-bank variant of
// handleSliceSetInt for general[A].Index(ints[B]).SetInt(ints[C]).
//
// Operates on the reflect (general) register bank without the runtime range check.
//
// Soundness: same proof obligation as handleSliceGetIntUnchecked.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opContinue on success or opPanicError if the collection is not a valid
// indexable value.
func handleSliceSetIntUnchecked(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return indexPanicResult
	}
	element := collection.Index(int(registers.ints[instruction.b]))
	switch element.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		element.SetInt(registers.ints[instruction.c])
	default:
		element.SetUint(uint64(registers.ints[instruction.c])) //nolint:gosec // intentional reinterpret
	}
	return opContinue
}
