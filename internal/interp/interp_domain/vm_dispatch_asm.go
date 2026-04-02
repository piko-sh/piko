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

//go:build !safe && !(js && wasm) && (amd64 || arm64)

package interp_domain

import "unsafe"

// dispatchAction indicates what the dispatch loop should do after
// handling an exit from the ASM dispatch loop.
type dispatchAction int

const (
	// loopRebuild instructs the dispatch loop to rebuild context and
	// re-enter dispatch.
	loopRebuild dispatchAction = iota

	// loopContinue instructs the dispatch loop to skip rebuild and
	// re-enter dispatch.
	loopContinue

	// loopReturn instructs the dispatch loop to return result and error to caller.
	loopReturn
)

// runDispatched executes bytecode starting from baseFramePointer using
// the ASM threaded dispatch loop for Tier 1 opcodes, falling back to
// Go for Tier 2 opcodes via a trampoline pattern.
//
// Execution alternates between ASM (fast path for ~67 pure register
// opcodes) and Go (for complex opcodes involving strings, reflect.Value,
// frame changes, closures, etc.).
//
// Takes baseFramePointer (int) which specifies the call stack frame to
// return from when execution completes.
//
// Returns the execution result and any error encountered.
func (vm *VM) runDispatched(baseFramePointer int) (any, error) {
	if vm.limits.forceGoDispatch || (vm.debugActive != nil && vm.debugActive.Load() != 0) {
		return vm.run(baseFramePointer)
	}

	savedBaseFp := vm.baseFramePointer
	vm.baseFramePointer = baseFramePointer
	defer func() { vm.baseFramePointer = savedBaseFp }()

	frame := &vm.callStack[vm.framePointer]
	registers := &frame.registers

	var ctx DispatchContext

	vm.buildDispatchContext(&ctx, &asmJumpTable)
	vm.saveCurrentDispatchRegisters(&ctx)

	for {
		dispatchLoop(&ctx)

		vm.syncCallContextFromASM(&ctx)
		frame = &vm.callStack[vm.framePointer]
		registers = &frame.registers
		frame.programCounter = int(ctx.programCounter)

		if vm.cancelled.Load() != 0 {
			return nil, vm.ctx.Err()
		}

		result, action, err := vm.handleDispatchExit(
			&ctx, frame, registers, baseFramePointer,
		)
		if action == loopReturn {
			return result, err
		}
		if action == loopContinue {
			continue
		}

		frame = &vm.callStack[vm.framePointer]
		registers = &frame.registers
		vm.rebuildDispatchPointers(&ctx, frame, registers)
	}
}

// handleDispatchExit routes an ASM dispatch exit to the appropriate
// handler and returns the action the main loop should take.
//
// Takes ctx (*DispatchContext) which provides the dispatch context.
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes baseFramePointer (int) which specifies the base frame for
// return detection.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) handleDispatchExit(
	ctx *DispatchContext,
	frame *callFrame,
	registers *Registers,
	baseFramePointer int,
) (any, dispatchAction, error) {
	switch ctx.exitReason {
	case exitEndOfCode:
		return vm.processEndOfCode(frame, baseFramePointer)
	case exitDivByZero:
		return nil, loopReturn, errDivisionByZero
	case exitCallOverflow:
		return nil, loopReturn, errStackOverflow
	case exitCall:
		return vm.processExitCall(ctx, frame, registers)
	case exitReturn:
		return vm.processExitReturn(frame, registers)
	case exitReturnVoid:
		return vm.processExitReturnVoid(frame, registers)
	case exitTailCall:
		vm.processExitTailCall(frame, registers)
		return nil, loopRebuild, nil
	case exitTier2:
		return vm.processExitTier2(frame, registers, ctx)
	default:
		return nil, loopRebuild, nil
	}
}

// processEndOfCode handles the end-of-code exit by running pending
// defers, popping the frame, and returning the result at the base frame.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes baseFramePointer (int) which specifies the base frame for
// return detection.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processEndOfCode(
	frame *callFrame,
	baseFramePointer int,
) (any, dispatchAction, error) {
	if len(vm.deferStack) > frame.deferBase {
		vm.runDefers()
	}
	if vm.framePointer == baseFramePointer {
		result, err := vm.extractResult(frame)
		vm.popFrame()
		return result, loopReturn, err
	}
	vm.popFrame()
	if vm.framePointer < baseFramePointer {
		return nil, loopReturn, nil
	}
	return nil, loopRebuild, nil
}

// processExitCall handles a compiled function call exit from the ASM
// dispatch loop.
//
// Takes ctx (*DispatchContext) which provides the dispatch context.
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitCall(
	ctx *DispatchContext,
	frame *callFrame,
	registers *Registers,
) (any, dispatchAction, error) {
	vm.saveCurrentDispatchRegisters(ctx)
	instruction := frame.function.body[frame.programCounter]
	frame.programCounter++
	if handleCall(vm, frame, registers, instruction) == opStackOverflow {
		return nil, loopReturn, errStackOverflow
	}
	vm.updateASMCallInfoBase()
	return nil, loopRebuild, nil
}

// processExitReturn handles a return instruction exit from the ASM
// dispatch loop.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitReturn(
	frame *callFrame,
	registers *Registers,
) (any, dispatchAction, error) {
	instruction := frame.function.body[frame.programCounter]
	frame.programCounter++
	if handleReturn(vm, frame, registers, instruction) == opDone {
		result := vm.evalResult
		vm.evalResult = nil
		return result, loopReturn, nil
	}
	vm.updateASMCallInfoBase()
	return nil, loopRebuild, nil
}

// processExitReturnVoid handles a void return exit from the ASM
// dispatch loop.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitReturnVoid(
	frame *callFrame,
	registers *Registers,
) (any, dispatchAction, error) {
	frame.programCounter++
	if handleReturnVoid(vm, frame, registers, instruction{}) == opDone {
		return nil, loopReturn, nil
	}
	vm.updateASMCallInfoBase()
	return nil, loopRebuild, nil
}

// processExitTailCall handles a tail call exit from the ASM dispatch
// loop.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
func (vm *VM) processExitTailCall(frame *callFrame, registers *Registers) {
	instruction := frame.function.body[frame.programCounter]
	frame.programCounter++
	handleTailCall(vm, frame, registers, instruction)
	vm.updateASMCallInfoBase()
}

// processExitTier2 handles a Tier 2 opcode exit by dispatching through
// the Go handler table.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitTier2(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	instruction := frame.function.body[frame.programCounter]
	frame.programCounter++
	rc := handlerTable[instruction.op](vm, frame, registers, instruction)
	switch rc {
	case opContinue:
		ctx.programCounter = int64(frame.programCounter)
		ctx.deferStackLength = int64(len(vm.deferStack))
		return nil, loopContinue, nil
	case opDone:
		result := vm.evalResult
		vm.evalResult = nil
		return result, loopReturn, nil
	case opDivByZero:
		return nil, loopReturn, errDivisionByZero
	case opStackOverflow:
		return nil, loopReturn, errStackOverflow
	case opPanicError:
		err := vm.evalError
		vm.evalError = nil
		return nil, loopReturn, err
	default:
		vm.updateASMCallInfoBase()
		return nil, loopRebuild, nil
	}
}

// rebuildDispatchPointers updates the ASM dispatch context pointers
// after a frame change (call, return, tier 2 handler, etc.).
//
// Takes ctx (*DispatchContext) which provides the dispatch context to
// update.
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
func (vm *VM) rebuildDispatchPointers(
	ctx *DispatchContext,
	frame *callFrame,
	registers *Registers,
) {
	body := frame.function.body
	if len(body) > 0 {
		ctx.codeBase = uintptr(unsafe.Pointer(&body[0]))
	}
	ctx.codeLength = int64(len(body))
	ctx.programCounter = int64(frame.programCounter)
	if len(registers.ints) > 0 {
		ctx.intsBase = uintptr(unsafe.Pointer(&registers.ints[0]))
	}
	if len(registers.floats) > 0 {
		ctx.floatsBase = uintptr(unsafe.Pointer(&registers.floats[0]))
	}
	if len(registers.strings) > 0 {
		ctx.stringsBase = uintptr(unsafe.Pointer(&registers.strings[0]))
	}
	if len(registers.uints) > 0 {
		ctx.uintsBase = uintptr(unsafe.Pointer(&registers.uints[0]))
	}
	if len(registers.bools) > 0 {
		ctx.boolsBase = uintptr(unsafe.Pointer(&registers.bools[0]))
	}
	if len(frame.function.intConstants) > 0 {
		ctx.intConstantsBase = uintptr(unsafe.Pointer(&frame.function.intConstants[0]))
	}
	if len(frame.function.floatConstants) > 0 {
		ctx.floatConstantsBase = uintptr(unsafe.Pointer(&frame.function.floatConstants[0]))
	}
	vm.refreshCallContext(ctx)
	vm.saveCurrentDispatchRegisters(ctx)
}
