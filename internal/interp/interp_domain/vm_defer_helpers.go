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

// allocateDeferredFrameRegisters reserves the register file for a deferred frame. Reuses
// the arena when available so the deferred frame's registers are reclaimed at frame exit;
// otherwise allocates fresh slices.
//
// Takes numRegs ([NumRegisterKinds]uint32) which is the per-bank count for the deferred
// callee's register frame.
//
// Returns the arena save point (zero when no arena is in use) and the freshly populated
// Registers struct.
func (vm *VM) allocateDeferredFrameRegisters(numRegs [NumRegisterKinds]uint32) (ArenaSavePoint, Registers) {
	if vm.arena == nil {
		return ArenaSavePoint{}, newRegisters(numRegs)
	}
	var save ArenaSavePoint
	vm.arena.SaveInto(&save)
	var regs Registers
	vm.arena.AllocRegistersInto(&regs, numRegs)
	return save, regs
}

// executeNativeDeferredCall runs a deferred call whose target is a native Go callable (a
// reflect.MakeFunc result, a bound method on a native type, or a function value imported
// from the symbol registry). Recovers panics from the native call and records them on vm
// so the outer dispatch loop can continue unwinding.
//
// Takes d (deferredCall) which specifies the deferred call record containing
// nativeFunction and arguments.
func (vm *VM) executeNativeDeferredCall(d deferredCall) {
	defer func() {
		if recovered := recover(); recovered != nil {
			vm.panicValue = materialiseAnyForArena(vm.arena, recovered)
			vm.panicking = true
		}
	}()
	target := d.nativeFunction
	if !target.IsValid() {
		return
	}
	arguments := coerceDeferredCallArguments(vm, target, d.arguments)
	target.Call(arguments)
}

// coerceDeferredCallArguments applies interface-adapter coercion at defer execution.
//
// Runs the same coercion buildReflectArgs performs at normal native-call dispatch time.
// Without it, a `defer fmt.Fprintf(&pikoWriter, ...)` schedules the call with the raw
// piko-synthesised *stringWriter pointer and the deferred `target.Call` panics with
// `reflect: Call using *struct{...} as type io.Writer` because the pikoWriterAdapter wrap
// never gets attached. The panic is then captured by the function's recover() and the
// deferred call writes nothing.
//
// Coercion uses an empty argumentTypeContext because static-type metadata is not captured
// across the defer-scheduling boundary; the fallback path inside tryBuildInterfaceAdapter
// consults pikoTypeName for piko-synthesised types, which is sufficient for the common
// case of piko-defined structs satisfying stdlib interfaces (io.Writer, io.Reader,
// fmt.Stringer, error).
//
// Takes vm (*VM) which provides the method-table and symbol-registry context the adapter
// builders consult.
// Takes target (reflect.Value) which is the bound native function so we can read its
// parameter types via Type().In(i).
// Takes arguments ([]reflect.Value) which is the captured argument list from when the
// defer was scheduled.
//
// Returns a fresh []reflect.Value with each argument coerced to the target's expected
// parameter type when an adapter applies, or the argument unchanged when none does.
func coerceDeferredCallArguments(vm *VM, target reflect.Value, arguments []reflect.Value) []reflect.Value {
	if !target.IsValid() || len(arguments) == 0 {
		return arguments
	}
	targetType := target.Type()
	if targetType.Kind() != reflect.Func {
		return arguments
	}
	numIn := targetType.NumIn()
	variadic := targetType.IsVariadic()
	out := make([]reflect.Value, len(arguments))
	copy(out, arguments)
	for i := range out {
		var parameterType reflect.Type
		switch {
		case variadic && i >= numIn-1:
			parameterType = targetType.In(numIn - 1).Elem()
		case i < numIn:
			parameterType = targetType.In(i)
		default:
			continue
		}
		out[i] = coerceReflectArgument(vm, out[i], parameterType, argumentTypeContext{})
	}
	return out
}
