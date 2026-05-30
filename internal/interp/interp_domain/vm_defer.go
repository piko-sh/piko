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
	"fmt"
	"reflect"
)

// runDefers executes all deferred calls registered by the current frame.
//
// Called on normal function return in LIFO order. Each defer is popped from the stack
// BEFORE it runs, so if it panics the remaining (un-run) defers are still on the stack
// for the panic-unwind path (unwindFrame) to run with recover semantics.
func (vm *VM) runDefers() {
	frame := vm.currentFrame()
	base := frame.deferBase
	for len(vm.deferStack) > base {
		call := vm.deferStack[len(vm.deferStack)-1]
		vm.deferStack = vm.deferStack[:len(vm.deferStack)-1]
		vm.executeDeferredCall(call)
	}
}

// runFrameSimpleDefer executes the trivial-defer slot armed on this frame and clears it.
// Used by handleReturn / handleReturnVoid / unwindFrame on the fast path that avoids
// appending to the global deferStack.
//
// The classifier guarantees the deferred call body cannot call recover(), so the slot
// must run unconditionally, even during a panic unwind, where the panic state is
// preserved across the call.
//
// Takes frame (*callFrame) which owns the trivial-defer slot.
func (vm *VM) runFrameSimpleDefer(frame *callFrame) {
	record := frame.simpleDefer
	if record == nil || !record.active {
		return
	}
	record.active = false
	deferred := deferredCall{
		function:       record.target,
		nativeFunction: record.nativeFunction,
		arguments:      record.arguments,
		frameIndex:     vm.framePointer,
	}
	record.target = nil
	record.nativeFunction = reflect.Value{}
	record.arguments = nil
	vm.executeDeferredCall(deferred)
}

// unwindGoexit handles a runtime.Goexit by running every armed defer.
//
// Walks frames from the current frame down to the base frame in LIFO order, popping each.
// recover() inside these deferred functions observes vm.panicking == false and returns
// nil, matching Go's "Because Goexit is not a panic, any recover calls in those deferred
// functions will return nil" semantics. The interpreter never calls the real
// runtime.Goexit; the host goroutine running the VM exits via normal return when vm.run
// finishes.
func (vm *VM) unwindGoexit() {
	for vm.framePointer >= vm.baseFramePointer {
		frame := vm.currentFrame()
		if frame.simpleDefer != nil && frame.simpleDefer.active {
			vm.runFrameSimpleDefer(frame)
		}
		for i := len(vm.deferStack) - 1; i >= frame.deferBase; i-- {
			vm.executeDeferredCall(vm.deferStack[i])
		}
		vm.deferStack = vm.deferStack[:frame.deferBase]
		vm.popFrame()
	}
}

// unwindPanic handles panic unwinding by running deferred calls for each frame in LIFO
// order, catching panics when a deferred call contains a recover().
//
// Returns nil if the panic was recovered, or an error wrapping the panic value otherwise.
func (vm *VM) unwindPanic() error {
	for vm.framePointer >= 0 {
		if vm.unwindFrame() {
			return nil
		}
	}
	if err, ok := vm.panicValue.(error); ok {
		return fmt.Errorf("panic: %w", err)
	}
	return fmt.Errorf("panic: %v", vm.panicValue)
}

// unwindFrame runs deferred calls for the current frame during unwinding.
//
// Each defer is popped from vm.deferStack BEFORE executeDeferredCall runs its body so
// that if the body re-panics (recover() followed by panic()), the nested unwindPanic that
// processes the body's own frame and then returns here cannot observe the same defer
// record still on the stack and re-execute it. Without pop-before-execute,
// recover-then-re-panic is an infinite recursion: each nested unwindFrame on the same
// frame sees the same top defer, runs it again, re-panics, repeats - overflowing the host
// stack instead of propagating the new panic to the next defer.
//
// Pops the frame in either case. The frame's trivial defer slot, if armed, runs first
// because the classifier guarantees that path cannot call recover() and therefore cannot
// cancel the panic; its only job is to release resources before unwind continues.
//
// Returns true if a recover() was found and the panic was caught, or false otherwise.
func (vm *VM) unwindFrame() bool {
	enterFramePointer := vm.framePointer
	frame := vm.currentFrame()
	if frame.simpleDefer != nil && frame.simpleDefer.active {
		vm.runFrameSimpleDefer(frame)
	}
	for len(vm.deferStack) > frame.deferBase {
		call := vm.deferStack[len(vm.deferStack)-1]
		vm.deferStack = vm.deferStack[:len(vm.deferStack)-1]
		vm.executeDeferredCall(call)
		if vm.framePointer < enterFramePointer {
			return !vm.panicking
		}
		if !vm.panicking {
			vm.runRemainingDefers(frame.deferBase)
			vm.syncNamedResults(frame)
			vm.deliverRecoveredReturn(frame)
			vm.popFrame()
			return true
		}
	}
	vm.popFrame()
	return false
}

// runRemainingDefers drains every deferred call above base.
//
// Executes the calls in LIFO order with the same pop-before-execute discipline as
// unwindFrame so nested unwindPanic activity cannot re-run an entry. Called once a
// recover() has caught the panic and the frame is being finalised; the remaining defers
// run in normal (non-panicking) mode.
//
// Takes base (int) which specifies the lowest defer stack index to drain down to.
func (vm *VM) runRemainingDefers(base int) {
	for len(vm.deferStack) > base {
		call := vm.deferStack[len(vm.deferStack)-1]
		vm.deferStack = vm.deferStack[:len(vm.deferStack)-1]
		vm.executeDeferredCall(call)
	}
}

// executeDeferredCall runs a single deferred closure call by pushing a new call frame and
// executing it.
//
// Takes call (deferredCall) which specifies the deferred call record containing the
// closure and arguments.
func (vm *VM) executeDeferredCall(call deferredCall) {
	if call.function == nil {
		vm.executeNativeDeferredCall(call)
		return
	}
	callee := call.function.function
	deferSave, deferRegs := vm.allocateDeferredFrameRegisters(callee.numRegisters)
	newFrame := callFrame{
		registers:      deferRegs,
		function:       callee,
		programCounter: 0,
		deferBase:      len(vm.deferStack),
		arenaSave:      deferSave,
	}

	if call.function.upvalues != nil {
		newFrame.initialiseUpvalues(call.function.upvalues, vm.arena)
	}

	placeReflectArgs(&newFrame.registers, call.arguments, callee.parameterKinds, vm.arena)

	snapshot := vm.swapToClosureRoot(call.function.rootFunction)

	vm.guardCallDepth()
	vm.framePointer++
	if vm.framePointer >= len(vm.callStack) {
		vm.growCallStack()
	}
	vm.callStack[vm.framePointer] = newFrame
	vm.recordFrameSnapshot(vm.framePointer, snapshot)

	priorRecoverEligible := vm.recoverEligibleFrame
	vm.recoverEligibleFrame = vm.framePointer
	result, _ := vm.runDispatchedGuarded(vm.framePointer)
	vm.recoverEligibleFrame = priorRecoverEligible
	if result != nil && vm.evalResult == nil {
		vm.evalResult = result
	}
}
