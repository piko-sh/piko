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

//go:build piko_bce_paranoid

// Paranoid (safety-harness) variants of the bounds-check-elimination
// handlers. The default fast variants live in
// vm_handler_bce_unchecked.go and skip the bounds check based on a
// compile-time proof emitted by the BCE pass. A defective proof there
// is undefined behaviour at runtime.
//
// Building with -tags piko_bce_paranoid swaps in this file: every
// "unchecked" handler regains an explicit bounds check and routes
// out-of-range accesses through vm.evalError + opPanicError, mirroring
// the checked variants exactly. CI enables this tag for the
// adversarial-bytecode fuzz runs so a bad proof surfaces as a clean
// verifier-style failure rather than a Go runtime panic or memory
// corruption. Production builds leave the tag off.

package interp_domain

import (
	"fmt"
	"reflect"
)

// handleSliceGetIntDirectUnchecked is the paranoid variant of handleSliceGetIntDirect. It
// reinstates the runtime bounds check dropped by the fast variant so an incorrect
// compile-time BCE proof is reported as a regular interpreter error.
//
// Takes vm (*VM) which receives the bounds-check error when triggered.
// Takes registers (*Registers) which holds the slicesInt collection and the int index.
// Takes instruction (instruction) which encodes the destination int register A, the
// source slicesInt register B, and the index int register C.
//
// Returns opContinue on success or opPanicError when the index is outside [0,
// len(slice)).
func handleSliceGetIntDirectUnchecked(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	slice := registers.slicesInt[instruction.b]
	index := registers.ints[instruction.c]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	registers.ints[instruction.a] = slice[index]
	return opContinue
}

// handleSliceSetIntDirectUnchecked is the paranoid variant of handleSliceSetIntDirect.
// The runtime bounds check is restored to surface bad BCE proofs as interpreter errors.
//
// Takes vm (*VM) which receives the bounds-check error when triggered.
// Takes registers (*Registers) which holds the slicesInt collection, the index, and the
// value.
// Takes instruction (instruction) which encodes the destination slicesInt register A, the
// index int register B, and the value int register C.
//
// Returns opContinue on success or opPanicError when the index is outside [0,
// len(slice)).
func handleSliceSetIntDirectUnchecked(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	slice := registers.slicesInt[instruction.a]
	index := registers.ints[instruction.b]
	if uint64(index) >= uint64(len(slice)) { //nolint:gosec // unsigned compare catches negatives as overflow
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(slice))
		return opPanicError
	}
	slice[index] = registers.ints[instruction.c]
	return opContinue
}

// handleStringIndexUnchecked is the paranoid variant of handleStringIndex. The runtime
// range check is restored so a defective BCE proof surfaces as a normal interpreter error
// rather than a Go runtime panic.
//
// Takes vm (*VM) which receives the bounds-check error when triggered.
// Takes registers (*Registers) which holds the string, index, and destination registers.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opContinue on success or opPanicError when the index is outside [0,
// len(string)).
func handleStringIndexUnchecked(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.strings[instruction.b]
	index := int(registers.ints[instruction.c])
	if index < 0 || index >= len(s) {
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(s))
		return opPanicError
	}
	registers.uints[instruction.a] = uint64(s[index])
	return opContinue
}

// handleStringIndexToIntUnchecked is the paranoid variant of handleStringIndexToInt. The
// runtime range check is restored.
//
// Takes vm (*VM) which receives the bounds-check error when triggered.
// Takes registers (*Registers) which holds the string, index, and destination registers.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opContinue on success or opPanicError when the index is outside [0,
// len(string)).
func handleStringIndexToIntUnchecked(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	s := registers.strings[instruction.b]
	index := int(registers.ints[instruction.c])
	if index < 0 || index >= len(s) {
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, len(s))
		return opPanicError
	}
	registers.ints[instruction.a] = int64(s[index])
	return opContinue
}

// handleSliceGetIntUnchecked is the paranoid variant of the reflect-bank bounds-elided
// get. The runtime range check is restored so a defective BCE proof on a general-bank
// slice surfaces through vm.evalError instead of letting reflect.Value.Index panic.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection and index.
// Takes instruction (instruction) which encodes the destination register.
//
// Returns opContinue on success or opPanicError when the collection is not indexable or
// the index falls outside [0, len).
func handleSliceGetIntUnchecked(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.b])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.c])
	if index < 0 || index >= collection.Len() {
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, collection.Len())
		return opPanicError
	}
	element := unwrapInterfaceElement(collection.Index(index))
	switch element.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		registers.ints[instruction.a] = element.Int()
	default:
		registers.ints[instruction.a] = int64(element.Uint()) //nolint:gosec // intentional reinterpret
	}
	return opContinue
}

// handleSliceSetIntUnchecked is the paranoid variant of the reflect-bank bounds-elided
// set. The runtime range check is restored.
//
// Takes vm (*VM) which is the virtual machine executing the instruction.
// Takes registers (*Registers) which holds the collection, index and value.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opContinue on success or opPanicError when the collection is not indexable or
// the index falls outside [0, len).
func handleSliceSetIntUnchecked(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	collection, indexPanicResult, ok := resolveIndexCollection(vm, registers.general[instruction.a])
	if !ok {
		return indexPanicResult
	}
	index := int(registers.ints[instruction.b])
	if index < 0 || index >= collection.Len() {
		vm.evalError = fmt.Errorf(errIdxOutOfRangeFmt, errIndexOutOfRange, index, collection.Len())
		return opPanicError
	}
	element := collection.Index(index)
	switch element.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		element.SetInt(registers.ints[instruction.c])
	default:
		element.SetUint(uint64(registers.ints[instruction.c])) //nolint:gosec // intentional reinterpret
	}
	return opContinue
}
