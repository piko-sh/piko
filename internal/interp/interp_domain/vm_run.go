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
	"fmt"
	"runtime"
)

// runDispatchedGuarded wraps runDispatched with a recover() so that a native Go panic
// raised by an opcode handler (for example the invalid-register and not-a-struct
// diagnostics in vm_bounds_check.go, or any reflect operation that panics) is contained
// inside the interpreter instead of unwinding into and crashing the host.
//
// Takes baseFramePointer (int) which specifies the frame index at which this invocation
// should stop and return results.
//
// Returns the execution result and any error encountered during dispatch.
func (vm *VM) runDispatchedGuarded(baseFramePointer int) (result any, err error) {
	if vm.lockInterpreter() {
		defer vm.unlockInterpreter()
	}
	defer func() { vm.handleRecoveredHandlerPanic(recover(), &result, &err) }()
	return vm.runDispatched(baseFramePointer)
}

// runGuarded wraps the pure-Go run loop with the same handler-panic recovery as
// runDispatchedGuarded. Used by the variable-initialiser path, which dispatches via run
// directly rather than runDispatched.
//
// Takes baseFramePointer (int) which specifies the frame index at which this invocation
// should stop and return results.
//
// Returns the execution result and any error encountered during dispatch.
func (vm *VM) runGuarded(baseFramePointer int) (result any, err error) {
	if vm.lockInterpreter() {
		defer vm.unlockInterpreter()
	}
	defer func() { vm.handleRecoveredHandlerPanic(recover(), &result, &err) }()
	return vm.run(baseFramePointer)
}

// handleRecoveredHandlerPanic forwards a recovered handler panic.
//
// The deferred body shared by runDispatchedGuarded and runGuarded. Receives the value
// already recovered by the enclosing deferred closure, converts a native Go panic raised
// by an opcode handler into an interpreted panic so interpreted defer/recover can observe
// it; an arena-budget panic is re-raised so execute's own recover continues to surface it
// as an error.
//
// Takes recovered (any) which is the value returned by recover() in the deferred closure,
// or nil when no panic is pending.
// Takes result (*any) which receives the recovered execution result.
// Takes err (*error) which receives the surfaced error, if any.
func (vm *VM) handleRecoveredHandlerPanic(recovered any, result *any, err *error) {
	if recovered == nil {
		return
	}
	if budgetErr, ok := recovered.(error); ok && errors.Is(budgetErr, errArenaBudgetExceeded) {
		panic(recovered)
	}
	switch raiseNativePanicAsInterpreted(vm, nativeHandlerPanicValue(recovered)) {
	case opDone, opFrameChanged:
		*result = vm.evalResult
		vm.evalResult = nil
		*err = nil
	default:
		*result = nil
		*err = vm.evalError
		vm.evalError = nil
	}
}

// run is the main execution loop, dispatching all opcodes via flatDispatchSwitch (Path-B)
// defined in vm_handler_flat_switch.go. The ASM dispatch loop (Path-A) covers most
// opcodes inline and only returns to this loop on exit reasons that need a Go-side
// handler.
//
// Takes baseFramePointer (int) which specifies the frame index at which this invocation
// should stop and return results.
//
// Returns the execution result and any error encountered during dispatch.
//
//revive:disable:cognitive-complexity // VM dispatch loops are inherently complex.
func (vm *VM) run(baseFramePointer int) (any, error) {
	savedBaseFp := vm.baseFramePointer
	vm.baseFramePointer = baseFramePointer
	defer func() { vm.baseFramePointer = savedBaseFp }()

	frame := &vm.callStack[vm.framePointer]
	registers := &frame.registers

	var ops uint32
	for {
		ops++
		if ops&cancellationCheckMask == 0 {
			if done, result, err := vm.runPeriodicChecks(); done {
				return result, err
			}
			if vm.checkpointFlags != 0 || (vm.arena != nil && vm.arena.gcShouldRun()) {
				vm.runPendingCheckpoints()
				frame = &vm.callStack[vm.framePointer]
				registers = &frame.registers
			}
			vm.yieldInterpreterLock()
		}
		if vm.shouldStopDebug(frame) {
			return nil, ErrDebuggerStop
		}
		if frame.programCounter >= len(frame.function.body) {
			done, result, err := vm.handleEndOfBody(frame, baseFramePointer)
			if done {
				return result, err
			}
			frame = &vm.callStack[vm.framePointer]
			registers = &frame.registers
			continue
		}

		instruction := frame.function.body[frame.programCounter]
		frame.programCounter++

		if exhausted, err := vm.accountForInstructionCost(instruction.op); exhausted {
			return nil, err
		}
		vm.maybeYield()

		rc := flatDispatchSwitch(vm, frame, registers, instruction)
		if rc == opContinue {
			continue
		}
		result, terminal, err := vm.handleOpResult(rc)
		if terminal {
			return result, err
		}
		frame = &vm.callStack[vm.framePointer]
		registers = &frame.registers
	}
}

// accountForInstructionCost decrements the VM's remaining cost budget by the per-opcode
// cost when metering is enabled, reporting budget exhaustion via the returned pair. The
// caller must abort the dispatch loop when the exhausted flag is true.
//
// Takes op (opcode) which identifies the cost-table entry to charge.
//
// Returns true when the budget is now non-positive (exhausted) and the associated error
// to surface from run.
func (vm *VM) accountForInstructionCost(op opcode) (bool, error) {
	if vm.costRemaining <= 0 {
		return false, nil
	}
	vm.costRemaining -= vm.limits.costTable[op]
	if vm.costRemaining <= 0 {
		return true, errCostBudgetExceeded
	}
	return false, nil
}

// maybeYield optionally calls runtime.Gosched() when the yield interval is enabled and
// the per-VM counter aligns with it. The yield is a cooperative scheduling hint; it never
// blocks the VM.
func (vm *VM) maybeYield() {
	if vm.limits.yieldInterval == 0 {
		return
	}
	vm.yieldCounter++
	if vm.yieldCounter&(vm.limits.yieldInterval-1) == 0 {
		runtime.Gosched()
	}
}

// runPeriodicChecks inspects the cancellation and panic flags.
//
// When a terminal condition is observed, returns (true, result, err); the dispatch loop
// must then exit. Otherwise returns (false, nil, nil) to indicate execution may continue.
//
// Returns done (bool) which is true when the dispatch loop should exit immediately.
// Returns result (any) which is the propagated result value to surface from run.
// Returns err (error) which is any error encountered during the periodic checks.
func (vm *VM) runPeriodicChecks() (done bool, result any, err error) {
	if vm.cancelled.Load() != 0 {
		return true, nil, vm.ctx.Err()
	}
	if info := vm.globals.goroutinePanic.Load(); info != nil {
		return true, nil, fmt.Errorf("goroutine panicked: %v", info.value)
	}
	return false, nil, nil
}

// handleEndOfBody processes the end-of-body condition for the current frame by running
// defers, then either returning the result for the base frame or popping the frame and
// continuing.
//
// Takes frame (*callFrame) which specifies the current call frame being completed.
// Takes baseFramePointer (int) which specifies the frame index that marks the bottom of
// this run invocation.
//
// Returns (true, result, err) when the caller should return, or (false, _, _) to continue
// the loop.
func (vm *VM) handleEndOfBody(frame *callFrame, baseFramePointer int) (bool, any, error) {
	atBase, atRoot, result, err := vm.finaliseFrameAtEnd(frame, baseFramePointer)
	if atBase {
		return true, result, err
	}
	if atRoot {
		return true, nil, nil
	}
	return false, nil, nil
}

// finaliseFrameAtEnd runs deferred work for a returning frame, pops the frame, and
// reports the post-pop position relative to the base frame pointer. The caller's
// interpretation of (atBase, atRoot) determines whether the dispatch loop should keep
// running, return, or extract the final result.
//
// Takes frame (*callFrame) which is the returning call frame.
// Takes baseFramePointer (int) which is the base frame pointer of the current dispatch
// invocation.
//
// Returns atBase (true when the popped frame was the base frame and the caller should
// extract a result), atRoot (true when the post- pop frame pointer is below the base,
// signalling unwind), the extracted result when atBase is true, and any extraction error.
func (vm *VM) finaliseFrameAtEnd(frame *callFrame, baseFramePointer int) (atBase bool, atRoot bool, result any, err error) {
	if frame.simpleDefer != nil && frame.simpleDefer.active {
		vm.runFrameSimpleDefer(frame)
	}
	if len(vm.deferStack) > frame.deferBase {
		vm.runDefers()
	}
	if vm.framePointer == baseFramePointer {
		result, err = vm.extractResult(frame)
		vm.popFrame()
		return true, false, result, err
	}
	vm.popFrame()
	if vm.framePointer < baseFramePointer {
		return false, true, nil, nil
	}
	return false, false, nil, nil
}

// handleOpResult translates an opcode handler return code into either a terminal result
// or a signal that the frame pointer changed and local variables must be refreshed.
//
// Takes rc (opResult) which specifies the opcode handler return code to translate.
//
// Returns the result value, a terminal flag indicating whether execution should stop, and
// any error.
func (vm *VM) handleOpResult(rc opResult) (result any, terminal bool, err error) {
	switch rc {
	case opDone:
		result = vm.evalResult
		vm.evalResult = nil
		return result, true, nil
	case opDivByZero:
		result := raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: integer divide by zero"))
		return vm.handleOpResult(result)
	case opStackOverflow:
		return nil, true, vm.stackOverflowError()
	case opPanicError:
		err = vm.evalError
		vm.evalError = nil
		return nil, true, err
	default:
		return nil, false, nil
	}
}

// nativeHandlerPanicValue normalises a recovered host panic into a value suitable for
// interpreted recover(). String panics raised by the VM diagnostics in vm_bounds_check.go
// are wrapped in a runtimePanicError so fmt formatting matches Go's runtime error
// surface; error and other values pass through unchanged.
//
// Takes recovered (any) which is the non-nil value returned by the host recover().
//
// Returns the value to hand to raiseNativePanicAsInterpreted.
func nativeHandlerPanicValue(recovered any) any {
	if message, ok := recovered.(string); ok {
		return &runtimePanicError{message: message}
	}
	return recovered
}
