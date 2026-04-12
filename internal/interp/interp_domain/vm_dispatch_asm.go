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

import (
	"unsafe"
)

// dispatchAction indicates what the dispatch loop should do after handling an exit from
// the ASM dispatch loop.
type dispatchAction int

const (
	// loopRebuild instructs the dispatch loop to rebuild context and re-enter dispatch.
	loopRebuild dispatchAction = iota

	// loopContinue instructs the dispatch loop to skip rebuild and re-enter dispatch.
	loopContinue

	// loopReturn instructs the dispatch loop to return result and error to caller.
	loopReturn
)

// runDispatched executes bytecode starting from baseFramePointer using the ASM threaded
// dispatch loop for tier-1 opcodes, falling back to Go for tier-2 opcodes via a
// trampoline pattern.
//
// Execution alternates between ASM (fast path for pure-register opcodes) and Go (for
// opcodes involving strings, reflect.Value, frame changes, closures, etc.).
//
// A shimmed tier-2 handler may unwind every frame in-place (for example,
// handleCallBuiltin invoking runtime.Goexit), leaving vm.framePointer = -1; the loop
// preserves the prior frame/registers pointers in that case because the downstream
// error-path exits do not dereference them.
//
// Takes baseFramePointer (int) which specifies the call stack frame to return from when
// execution completes.
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
		if vm.framePointer >= 0 {
			frame = &vm.callStack[vm.framePointer]
			registers = &frame.registers
			frame.programCounter = int(ctx.programCounter)
		}

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

// handleDispatchExit routes an ASM dispatch exit to the appropriate handler and returns
// the action the main loop should take.
//
// Takes ctx (*DispatchContext) which provides the dispatch context.
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes baseFramePointer (int) which specifies the base frame for return detection.
//
// Returns the result, the dispatch action, and any error.
//
//nolint:revive // function-length,cyclomatic: hot dispatch switch.
func (vm *VM) handleDispatchExit(
	ctx *DispatchContext,
	frame *callFrame,
	registers *Registers,
	baseFramePointer int,
) (any, dispatchAction, error) {
	switch ctx.exitReason {
	case exitEndOfCode:
		return vm.processEndOfCode(frame, baseFramePointer)
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
	case exitSetField:
		return vm.processExitSetField(frame, registers, ctx)
	case exitGetField:
		return vm.processExitGetField(frame, registers, ctx)
	case exitMapIndex:
		return vm.processExitMapIndex(frame, registers, ctx)
	case exitAppend:
		return vm.processExitAppend(frame, registers, ctx)
	case exitAppendByteFast:
		return vm.processExitAppendByteFast(frame, registers, ctx)
	case exitTestNilJumpFalse:
		return vm.processExitTestNilJumpFalse(frame, registers, ctx)
	case exitTestNilJumpTrue:
		return vm.processExitTestNilJumpTrue(frame, registers, ctx)
	case exitFrameChanged:
		vm.updateASMCallInfoBase()
		return nil, loopRebuild, nil
	case exitGetStructFieldIntT0:
		return vm.processExitGetStructFieldIntT0(frame, registers, ctx)
	case exitGetStructFieldUintT0:
		return vm.processExitGetStructFieldUintT0(frame, registers, ctx)
	case exitGetStructFieldFloatT0:
		return vm.processExitGetStructFieldFloatT0(frame, registers, ctx)
	case exitGetStructFieldBoolT0:
		return vm.processExitGetStructFieldBoolT0(frame, registers, ctx)
	case exitSetStructFieldIntT0:
		return vm.processExitSetStructFieldIntT0(frame, registers, ctx)
	case exitSetStructFieldUintT0:
		return vm.processExitSetStructFieldUintT0(frame, registers, ctx)
	case exitSetStructFieldFloatT0:
		return vm.processExitSetStructFieldFloatT0(frame, registers, ctx)
	case exitSetStructFieldBoolT0:
		return vm.processExitSetStructFieldBoolT0(frame, registers, ctx)
	case exitGetStructFieldGeneralT0:
		return vm.processExitGetStructFieldGeneralT0(frame, registers, ctx)
	case exitSetStructFieldGeneralT0:
		return vm.processExitSetStructFieldGeneralT0(frame, registers, ctx)
	case exitDivByZero:
		return vm.raiseDivByZeroForDispatch()
	case exitCallOverflow, exitStackOverflowTier2:
		return nil, loopReturn, vm.stackOverflowError()
	case exitPanicErrorTier2:
		err := vm.evalError
		vm.evalError = nil
		return nil, loopReturn, err
	case exitDoneTier2:
		result := vm.evalResult
		vm.evalResult = nil
		return result, loopReturn, nil
	default:
		return nil, loopRebuild, nil
	}
}

// processExitSetField is the direct-exit Go handler for opSetField.
//
// The asm stub handlerSetFieldExit (installed at asmJumpTable[opSetField]) routes here
// instead of the generic tier2Fallback -> processExitTier2 path, saving the
// handlerTable[op] indirect call on the first op of the trampoline run. Subsequent
// trampolining ops in the same Go entry fall back to the shared batch loop (which uses
// handlerTable[op]) so a long Go-fallback stretch still avoids redundant ASM <-> Go
// round-trips.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitSetField(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleSetField(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitGetField is the direct-exit Go handler for opGetField.
//
// Mirrors processExitSetField: skips handlerTable[op] on the first op, then continues
// batching for subsequent trampolining ops.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitGetField(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleGetField(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitMapIndex is the direct-exit Go handler for opMapIndex.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitMapIndex(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleMapIndex(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitAppend is the direct-exit Go handler for opAppend.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitAppend(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleAppend(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitAppendByteFast is the direct-exit Go handler for the specialised
// byte-builder opcode.
//
// Skips both the handlerTable[op] indirect call and the generic tier-2 processor.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitAppendByteFast(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleAppendByteFast(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// Tier-0 struct-field READER direct-exit Go handlers follow. Each one mirrors
// processExitGetField's shape: read the current instruction, advance PC, dispatch
// directly to the matching tier-0 Go handler (skipping the handlerTable[op] indirect
// lookup), then run the shared post-first-op batching loop. Subsequent trampolining ops
// in the same Go entry still go through the generic indirect-dispatch loop, so the saving
// is the single handlerTable[op] miss the trampoline's entry op would otherwise pay.

// processExitGetStructFieldIntT0 is the direct-exit Go handler for opGetStructFieldIntT0.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitGetStructFieldIntT0(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleGetStructFieldIntT0(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitGetStructFieldUintT0 is the direct-exit Go handler for
// opGetStructFieldUintT0.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitGetStructFieldUintT0(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleGetStructFieldUintT0(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitGetStructFieldFloatT0 is the direct-exit Go handler for
// opGetStructFieldFloatT0.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitGetStructFieldFloatT0(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleGetStructFieldFloatT0(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitGetStructFieldBoolT0 is the direct-exit Go handler for
// opGetStructFieldBoolT0.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitGetStructFieldBoolT0(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleGetStructFieldBoolT0(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// Tier-0 struct-field WRITER direct-exit Go handlers follow (primitives only; the
// General-bank Set stays on the generic processExitTier2 path because it needs
// runtimeTypedmemmove).

// processExitSetStructFieldIntT0 is the direct-exit Go handler for opSetStructFieldIntT0.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitSetStructFieldIntT0(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleSetStructFieldIntT0(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitSetStructFieldUintT0 is the direct-exit Go handler for
// opSetStructFieldUintT0.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitSetStructFieldUintT0(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleSetStructFieldUintT0(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitSetStructFieldFloatT0 is the direct-exit Go handler for
// opSetStructFieldFloatT0.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitSetStructFieldFloatT0(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleSetStructFieldFloatT0(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitSetStructFieldBoolT0 is the direct-exit Go handler for
// opSetStructFieldBoolT0.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitSetStructFieldBoolT0(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleSetStructFieldBoolT0(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitGetStructFieldGeneralT0 is the direct-exit Go handler for
// opGetStructFieldGeneral (the tier-0 general-bank reader).
//
// It mirrors processExitGetStructFieldIntT0 by saving the handlerTable[op] indirect call
// on the trampoline's first op; subsequent batched ops in the same Go entry fall back to
// the shared batching loop in runBatchedTier2Loop.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitGetStructFieldGeneralT0(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleGetStructFieldGeneralT0(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitSetStructFieldGeneralT0 is the direct-exit Go handler for
// opSetStructFieldGeneral. Same pattern as the reader; the handler still uses
// runtime_typedmemmove internally for the pointer/interface store - the direct exit only
// saves the handlerTable[op] indirect.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitSetStructFieldGeneralT0(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleSetStructFieldGeneralT0(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitTestNilJumpFalse is the direct-exit Go handler for opTestNilJumpFalse.
//
// The handler body runs in Go; the direct exit saves only the handlerTable[op] indirect
// on the trampoline's first op and fires once per recursive step in any *node-style
// pointer walker.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitTestNilJumpFalse(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleTestNilJumpFalse(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// processExitTestNilJumpTrue mirrors processExitTestNilJumpFalse for the inverted-sense
// nil test.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitTestNilJumpTrue(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
) (any, dispatchAction, error) {
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	instruction := body[frame.programCounter]
	frame.programCounter++
	rc := handleTestNilJumpTrue(vm, frame, registers, instruction)
	return vm.runBatchedTier2Loop(frame, registers, ctx, codeLength, body, rc)
}

// runBatchedTier2Loop is the shared post-first-op batching loop used by all per-op
// direct-exit handlers. After the first op was handled inline (without the
// handlerTable[op] indirect), subsequent trampolining ops fall back to the generic
// indirect-dispatch loop.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
// Takes ctx (*DispatchContext) which provides the dispatch context.
// Takes codeLength (int) which is the cached length of body.
// Takes body ([]instruction) which is the bytecode body of the current function.
// Takes rc (opResult) which is the return code from the inline first op.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) runBatchedTier2Loop(
	frame *callFrame,
	registers *Registers,
	ctx *DispatchContext,
	codeLength int,
	body []instruction,
	rc opResult,
) (any, dispatchAction, error) {
	for range batchedTier2Limit {
		switch rc {
		case opContinue:
			if frame.programCounter >= codeLength {
				ctx.programCounter = int64(frame.programCounter)
				ctx.deferStackLength = int64(len(vm.deferStack))
				return nil, loopContinue, nil
			}
			if !instructionWouldTrampoline(body[frame.programCounter]) {
				ctx.programCounter = int64(frame.programCounter)
				ctx.deferStackLength = int64(len(vm.deferStack))
				return nil, loopContinue, nil
			}
			instruction := body[frame.programCounter]
			frame.programCounter++
			rc = flatDispatchSwitch(vm, frame, registers, instruction)
		case opDone:
			result := vm.evalResult
			vm.evalResult = nil
			return result, loopReturn, nil
		case opDivByZero:
			return vm.raiseDivByZeroForDispatch()
		case opStackOverflow:
			return nil, loopReturn, vm.stackOverflowError()
		case opPanicError:
			err := vm.evalError
			vm.evalError = nil
			return nil, loopReturn, err
		default:
			vm.updateASMCallInfoBase()
			return nil, loopRebuild, nil
		}
	}
	ctx.programCounter = int64(frame.programCounter)
	ctx.deferStackLength = int64(len(vm.deferStack))
	return nil, loopContinue, nil
}

// processEndOfCode handles the end-of-code exit by running pending defers, popping the
// frame, and returning the result at the base frame.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes baseFramePointer (int) which specifies the base frame for return detection.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processEndOfCode(
	frame *callFrame,
	baseFramePointer int,
) (any, dispatchAction, error) {
	atBase, atRoot, result, err := vm.finaliseFrameAtEnd(frame, baseFramePointer)
	switch {
	case atBase:
		return result, loopReturn, err
	case atRoot:
		return nil, loopReturn, nil
	default:
		return nil, loopRebuild, nil
	}
}

// processExitCall handles a compiled function call exit from the ASM dispatch loop.
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
	switch handleCall(vm, frame, registers, instruction) {
	case opStackOverflow:
		return nil, loopReturn, vm.stackOverflowError()
	case opDone:
		result := vm.evalResult
		vm.evalResult = nil
		return result, loopReturn, nil
	case opPanicError:
		err := vm.evalError
		vm.evalError = nil
		return nil, loopReturn, err
	default:
	}
	vm.updateASMCallInfoBase()
	return nil, loopRebuild, nil
}

// processExitReturn handles a return instruction exit from the ASM dispatch loop.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
//
// Returns the result, the dispatch action, and any error.
func (vm *VM) processExitReturn(
	frame *callFrame,
	registers *Registers,
) (any, dispatchAction, error) {
	tier2Instruction := frame.function.body[frame.programCounter]
	frame.programCounter++
	syntheticReturn := instruction{a: tier2Instruction.c}
	if handleReturn(vm, frame, registers, syntheticReturn) == opDone {
		result := vm.evalResult
		vm.evalResult = nil
		return result, loopReturn, nil
	}
	vm.updateASMCallInfoBase()
	return nil, loopRebuild, nil
}

// processExitReturnVoid handles a void return exit from the ASM dispatch loop.
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

// processExitTailCall handles a tail call exit from the ASM dispatch loop.
//
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
func (vm *VM) processExitTailCall(frame *callFrame, registers *Registers) {
	instruction := frame.function.body[frame.programCounter]
	frame.programCounter++
	handleTailCall(vm, frame, registers, instruction)
	vm.updateASMCallInfoBase()
}

// processExitTier2 handles a tier-2 opcode exit by dispatching through the Go handler
// table.
//
// When the executed op returns opContinue and the next instruction is also a Go-fallback
// op (instructionWouldTrampoline returns true), the loop stays in Go for up to
// batchedTier2Limit consecutive ops, saving one ASM/Go round-trip per batched op. On
// reaching the cap the loop hands control back to ASM so dispatchLoop runs its
// cancellation check and re-enters Go on the next trampoline exit.
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
	codeLength := int(ctx.codeLength)
	body := frame.function.body
	for range batchedTier2Limit {
		instruction := body[frame.programCounter]
		frame.programCounter++
		rc := flatDispatchSwitch(vm, frame, registers, instruction)
		switch rc {
		case opContinue:
			if frame.programCounter >= codeLength {
				ctx.programCounter = int64(frame.programCounter)
				ctx.deferStackLength = int64(len(vm.deferStack))
				return nil, loopContinue, nil
			}
			if !instructionWouldTrampoline(body[frame.programCounter]) {
				ctx.programCounter = int64(frame.programCounter)
				ctx.deferStackLength = int64(len(vm.deferStack))
				return nil, loopContinue, nil
			}
		case opDone:
			result := vm.evalResult
			vm.evalResult = nil
			return result, loopReturn, nil
		case opDivByZero:
			return nil, loopReturn, errDivisionByZero
		case opStackOverflow:
			return nil, loopReturn, vm.stackOverflowError()
		case opPanicError:
			err := vm.evalError
			vm.evalError = nil
			return nil, loopReturn, err
		default:
			vm.updateASMCallInfoBase()
			return nil, loopRebuild, nil
		}
	}
	ctx.programCounter = int64(frame.programCounter)
	ctx.deferStackLength = int64(len(vm.deferStack))
	return nil, loopContinue, nil
}

// raiseDivByZeroForDispatch raises a divide-by-zero runtime panic.
//
// Converts an ASM/handler-table reported opDivByZero / exitDivByZero into an
// interpreted-side runtime panic (matching Go's "runtime error: integer divide by zero"
// message) and maps the resulting opResult to the dispatch loop's (result, action, error)
// tuple. Used wherever the dispatch boundary observed a terminal divide-by-zero result
// code so interpreted defer/recover() can catch it.
//
// Returns the dispatch tuple: the result value when the unwind landed at the base frame,
// loopRebuild when a recover() reset the frame pointer and the dispatcher should reload,
// and the propagated error when the panic escaped the interpreter.
func (vm *VM) raiseDivByZeroForDispatch() (any, dispatchAction, error) {
	result := raiseNativePanicAsInterpreted(vm, newRuntimePanicError("runtime error: integer divide by zero"))
	switch result {
	case opDone:
		r := vm.evalResult
		vm.evalResult = nil
		return r, loopReturn, nil
	case opPanicError:
		err := vm.evalError
		vm.evalError = nil
		return nil, loopReturn, err
	case opFrameChanged:
		return nil, loopRebuild, nil
	default:
	}
	return nil, loopRebuild, nil
}

// rebuildDispatchPointers updates the ASM dispatch context pointers after a frame change
// (call, return, tier-2 handler, etc.).
//
// Takes ctx (*DispatchContext) which provides the dispatch context to update.
// Takes frame (*callFrame) which specifies the current call frame.
// Takes registers (*Registers) which provides the register file.
func (vm *VM) rebuildDispatchPointers(
	ctx *DispatchContext,
	frame *callFrame,
	registers *Registers,
) {
	function := frame.function
	body := function.body
	if len(body) > 0 {
		ctx.codeBase = uintptr(unsafe.Pointer(&body[0]))
	}
	ctx.codeLength = int64(len(body))
	ctx.programCounter = int64(frame.programCounter)
	rebuildRegisterBaseScalars(ctx, registers, function.nonZeroBankMask)
	rebuildRegisterBaseSlices(ctx, registers, function.nonZeroBankMask)
	vm.repairRegisterBasesFromCallers(ctx, function.nonZeroBankMask)
	rebuildConstantBases(ctx, function)
	rebuildLayoutAndTypeTableBases(ctx, function)
	vm.refreshCallContext(ctx)
	vm.saveCurrentDispatchRegisters(ctx)
}

// repairRegisterBasesFromCallers restores bank base pointers that the current frame did
// not refresh by walking up the call stack to find the nearest ancestor frame whose bank
// is populated.
//
// Background: when a deeper callee with bank X is invoked, ctx's base for X is refreshed
// to that callee's bank. When the callee returns to an intermediate frame whose mask has
// bit X clear, rebuildRegisterBaseSlices skips X - leaving ctx pointing at the callee's
// (now-deallocated) bank rather than the original caller's live bank. The original caller
// may then dereference ctx[X] and read stale memory.
//
// This repair walks up callFrame[framePointer-1..baseFramePointer] looking for a frame
// whose function's mask DOES include each bank the current frame's mask DOES NOT include.
// The first such frame owns the live bank that the current frame will return into; ctx is
// refreshed to it so any inline op that crosses this bank reads the right memory.
//
// Banks whose bit IS set in currentMask are already correct from the preceding
// rebuildRegisterBaseSlices call; skip them.
//
// Takes ctx (*DispatchContext) which holds the base pointers being repaired.
// Takes currentMask (uint16) which is the current frame's nonZeroBankMask; bits already
// set here are already correct.
func (vm *VM) repairRegisterBasesFromCallers(ctx *DispatchContext, currentMask uint16) {
	if !vm.usesTypedSliceBanks {
		return
	}
	missing := typedSliceBankMask & ^currentMask
	if missing == 0 {
		return
	}
	for fp := vm.framePointer - 1; fp >= 0 && missing != 0; fp-- {
		ancestor := &vm.callStack[fp]
		if ancestor.function == nil {
			continue
		}
		bits := ancestor.function.nonZeroBankMask & missing
		if bits == 0 {
			continue
		}
		refreshCtxBasesFromAncestor(ctx, &ancestor.registers, bits)
		missing &^= bits
	}
}

// refreshCtxBasesFromAncestor updates ctx's bank-base pointers to the banks present in
// ancestorRegs whose bit is set in bits. The seven per-bank refresh checks (slicesInt,
// slicesFloat, slicesString, slicesBool, slicesUint, slicesByte, complex) are factored
// out of repairRegisterBasesFromCallers so the parent's cognitive complexity stays inside
// the linter limit.
//
// Each `len(regs.X) > 0` guard protects against the empty-bank case where indexing
// &regs.X[0] would panic; ancestor frames with the nonZeroBankMask bit set should always
// have a non-empty bank, but the guard makes the helper safe to call from non-strict
// callers.
//
// Takes ctx (*DispatchContext) whose base fields are refreshed.
// Takes ancestorRegs (*Registers) which holds the ancestor frame's banks.
// Takes bits (uint16) which is the subset of bank masks to refresh.
func refreshCtxBasesFromAncestor(ctx *DispatchContext, ancestorRegs *Registers, bits uint16) {
	if bits&allocMaskSliceInt != 0 && len(ancestorRegs.slicesInt) > 0 {
		ctx.slicesIntBase = uintptr(unsafe.Pointer(&ancestorRegs.slicesInt[0]))
	}
	if bits&allocMaskSliceFloat != 0 && len(ancestorRegs.slicesFloat) > 0 {
		ctx.slicesFloatBase = uintptr(unsafe.Pointer(&ancestorRegs.slicesFloat[0]))
	}
	if bits&allocMaskSliceString != 0 && len(ancestorRegs.slicesString) > 0 {
		ctx.slicesStringBase = uintptr(unsafe.Pointer(&ancestorRegs.slicesString[0]))
	}
	if bits&allocMaskSliceBool != 0 && len(ancestorRegs.slicesBool) > 0 {
		ctx.slicesBoolBase = uintptr(unsafe.Pointer(&ancestorRegs.slicesBool[0]))
	}
	if bits&allocMaskSliceUint != 0 && len(ancestorRegs.slicesUint) > 0 {
		ctx.slicesUintBase = uintptr(unsafe.Pointer(&ancestorRegs.slicesUint[0]))
	}
	if bits&allocMaskSliceByte != 0 && len(ancestorRegs.slicesByte) > 0 {
		ctx.slicesByteBase = uintptr(unsafe.Pointer(&ancestorRegs.slicesByte[0]))
	}
	if bits&allocMaskComplex != 0 && len(ancestorRegs.complex) > 0 {
		ctx.complexBase = uintptr(unsafe.Pointer(&ancestorRegs.complex[0]))
	}
}

// rebuildRegisterBaseScalars refreshes the scalar bank base pointers
// (int/float/string/uint/bool) on ctx for the active frame.
//
// Banks whose bit is clear in bankMask are left at their existing value so inline-return
// ASM that does not refresh every base never observes a stale-NULL.
//
// Takes ctx (*DispatchContext) which is the dispatch context whose base fields are
// refreshed.
// Takes registers (*Registers) which are the active frame's register banks.
// Takes bankMask (uint16) which is the bitmap of banks that must be refreshed for this
// frame.
func rebuildRegisterBaseScalars(ctx *DispatchContext, registers *Registers, bankMask uint16) {
	if bankMask&allocMaskInt != 0 {
		ctx.intsBase = uintptr(unsafe.Pointer(&registers.ints[0]))
	}
	if bankMask&allocMaskFloat != 0 {
		ctx.floatsBase = uintptr(unsafe.Pointer(&registers.floats[0]))
	}
	if bankMask&allocMaskString != 0 {
		ctx.stringsBase = uintptr(unsafe.Pointer(&registers.strings[0]))
	}
	if bankMask&allocMaskUint != 0 {
		ctx.uintsBase = uintptr(unsafe.Pointer(&registers.uints[0]))
	}
	if bankMask&allocMaskBool != 0 {
		ctx.boolsBase = uintptr(unsafe.Pointer(&registers.bools[0]))
	}
}

// rebuildRegisterBaseSlices refreshes typed-slice and complex bank base pointers on ctx
// for the active frame.
//
// Banks whose bit is clear in bankMask are intentionally not zeroed because the
// inline-return ASM does not refresh them; leaving the caller's still-valid base in place
// is correct because the compiler only emits typed-slice ASM ops when the current frame
// has matching slots.
//
// Takes ctx (*DispatchContext) which is the dispatch context whose base fields are
// refreshed.
// Takes registers (*Registers) which are the active frame's register banks.
// Takes bankMask (uint16) which is the bitmap of banks that must be refreshed for this
// frame.
func rebuildRegisterBaseSlices(ctx *DispatchContext, registers *Registers, bankMask uint16) {
	if bankMask&allocMaskSliceInt != 0 {
		ctx.slicesIntBase = uintptr(unsafe.Pointer(&registers.slicesInt[0]))
	}
	if bankMask&allocMaskSliceFloat != 0 {
		ctx.slicesFloatBase = uintptr(unsafe.Pointer(&registers.slicesFloat[0]))
	}
	if bankMask&allocMaskSliceString != 0 {
		ctx.slicesStringBase = uintptr(unsafe.Pointer(&registers.slicesString[0]))
	}
	if bankMask&allocMaskSliceBool != 0 {
		ctx.slicesBoolBase = uintptr(unsafe.Pointer(&registers.slicesBool[0]))
	}
	if bankMask&allocMaskSliceUint != 0 {
		ctx.slicesUintBase = uintptr(unsafe.Pointer(&registers.slicesUint[0]))
	}
	if bankMask&allocMaskSliceByte != 0 {
		ctx.slicesByteBase = uintptr(unsafe.Pointer(&registers.slicesByte[0]))
	}
	if bankMask&allocMaskComplex != 0 {
		ctx.complexBase = uintptr(unsafe.Pointer(&registers.complex[0]))
	}
}

// rebuildConstantBases refreshes the per-function constant-pool base pointers
// (int/float/string/bool).
//
// String and bool pools also reset their length fields and explicitly zero the base when
// empty so a cross-frame stale pointer never leaks into ASM.
//
// Takes ctx (*DispatchContext) which is the dispatch context whose constant-pool fields
// are refreshed.
// Takes function (*CompiledFunction) which is the active compiled function whose
// constants supply the bases.
func rebuildConstantBases(ctx *DispatchContext, function *CompiledFunction) {
	constMask := function.nonEmptyConstantMask
	if constMask&constMaskInt != 0 {
		ctx.intConstantsBase = uintptr(unsafe.Pointer(&function.intConstants[0]))
	}
	if constMask&constMaskFloat != 0 {
		ctx.floatConstantsBase = uintptr(unsafe.Pointer(&function.floatConstants[0]))
	}
	if constMask&constMaskString != 0 {
		ctx.stringConstantsBase = uintptr(unsafe.Pointer(&function.stringConstants[0]))
	} else {
		ctx.stringConstantsBase = 0
	}
	ctx.stringConstantsLength = int64(len(function.stringConstants))
	if constMask&constMaskBool != 0 {
		ctx.boolConstantsBase = uintptr(unsafe.Pointer(&function.boolConstants[0]))
	} else {
		ctx.boolConstantsBase = 0
	}
	ctx.boolConstantsLength = int64(len(function.boolConstants))
}

// rebuildLayoutAndTypeTableBases refreshes the structLayoutTable and typeTable base
// pointers on ctx, explicitly zeroing them on empty tables so stale cross-frame leaks
// cannot reach ASM tier-1 struct-field handlers.
//
// Takes ctx (*DispatchContext) which is the dispatch context whose table base fields are
// refreshed.
// Takes function (*CompiledFunction) which is the active compiled function whose tables
// supply the bases.
func rebuildLayoutAndTypeTableBases(ctx *DispatchContext, function *CompiledFunction) {
	constMask := function.nonEmptyConstantMask
	if constMask&constMaskStructLayoutTable != 0 {
		ctx.structLayoutTableBase = uintptr(unsafe.Pointer(&function.structLayoutTable[0]))
	} else {
		ctx.structLayoutTableBase = 0
	}
	ctx.structLayoutTableLength = int64(len(function.structLayoutTable))
	if constMask&constMaskTypeTable != 0 {
		ctx.typeTableBase = uintptr(unsafe.Pointer(&function.typeTable[0]))
	} else {
		ctx.typeTableBase = 0
	}
	ctx.typeTableLength = int64(len(function.typeTable))
}
