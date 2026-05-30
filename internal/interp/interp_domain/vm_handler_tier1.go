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
	"errors"
	"reflect"
)

const (
	// minStrconvBase is the smallest radix accepted by strconv.FormatInt (and the tier-1
	// FormatInt sub-op).
	minStrconvBase = 2

	// maxStrconvBase is the largest radix accepted by strconv.FormatInt (and the tier-1
	// FormatInt sub-op): digits 0-9 plus letters a-z.
	maxStrconvBase = 36
)

// runMakeMethodExpr is the body of the MakeMethodExpr sub-op.
//
// Kept as a separate function rather than inlined into the switch because it performs
// multi-instruction extension-word reading and reflect.MakeFunc closure construction.
// Operand layout: instruction.b = destination general register, instruction.c = embedded
// field path length. The function index is in the next extension word; per-field
// traversal indices follow.
//
// Takes vm (*VM) which provides the function table.
// Takes frame (*callFrame) which carries extension words.
// Takes registers (*Registers) which receives the closure.
// Takes instruction (instruction) carrying destination (B) and fieldCount (C).
//
// Returns opResult which is the dispatch outcome.
//
// Panics when the generated method-expression closure is invoked without a receiver or
// with an out-of-range embedded field index.
func runMakeMethodExpr(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	fieldCount := int(instruction.c)
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	funcIndex := uint16(extensionWord.a) | uint16(extensionWord.b)<<wideBitShift
	if int(funcIndex) >= len(vm.functions) {
		vmBoundsError(vm, frame, boundsTableFunction, int(funcIndex), len(vm.functions))
		return opPanicError
	}
	var fieldPathBuffer [4]int
	fieldPath := fieldPathBuffer[:0]
	for range fieldCount {
		fieldExtension := frame.function.body[frame.programCounter]
		frame.programCounter++
		fieldPath = append(fieldPath, int(fieldExtension.a))
	}
	callee := vm.functions[funcIndex]
	signature, ok := callee.reflectMethodExprType()
	if !ok {
		vm.evalError = errors.New("cannot create method expression: no type info")
		return opPanicError
	}
	bound := &boundMethodVM{vm: vm, callee: callee, limits: vm.limits}
	boundFieldPath := fieldPath
	registers.general[instruction.b] = reflect.MakeFunc(signature, func(arguments []reflect.Value) []reflect.Value {
		if len(arguments) == 0 || !arguments[0].IsValid() {
			panic(newRuntimePanicError("interp: method expression invoked without a receiver"))
		}
		receiver := resolveMethodExprReceiver(arguments[0], boundFieldPath)
		return bound.invoke(receiver, arguments[1:], elemArg)
	})
	return opContinue
}

// resolveMethodExprReceiver derives the method-expression receiver.
//
// Unwraps the supplied receiver and walks the embedded-field path. Each step panics with
// an interpreted runtime error when the value is not a struct or the index is out of
// range, so a malformed call surfaces to interpreted recover() rather than crashing the
// host.
//
// Takes receiver (reflect.Value) which is the first argument supplied to the
// method-expression closure.
// Takes fieldPath ([]int) which lists the embedded-field indices to traverse from the
// unwrapped receiver.
//
// Returns reflect.Value which is the addressable struct receiver to hand to
// boundMethodVM.invoke.
//
// Panics when the field path traverses a non-struct value or when an embedded-field index
// is out of range; the panic surfaces to interpreted recover().
func resolveMethodExprReceiver(receiver reflect.Value, fieldPath []int) reflect.Value {
	if receiver.Kind() == reflect.Pointer || receiver.Kind() == reflect.Interface {
		receiver = receiver.Elem()
	}
	for _, index := range fieldPath {
		if receiver.Kind() != reflect.Struct {
			panic(newRuntimePanicError("interp: method expression field path traverses non-struct value (%s)", receiver.Kind()))
		}
		if index < 0 || index >= receiver.NumField() {
			panic(newRuntimePanicError("interp: method expression field index %d out of range", index))
		}
		receiver = receiver.Field(index)
	}
	return ensureAddressableStructReceiver(receiver)
}
