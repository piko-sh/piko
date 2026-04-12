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
)

// opResult encodes the outcome of an opcode handler as a single byte. The zero value
// (opContinue) is the common case and enables a fast branch-predicted check in the
// dispatch loop.
type opResult uint8

const (
	// opContinue signals the dispatch loop to advance to the next instruction. It is the
	// zero value and the common case.
	opContinue opResult = iota

	// opFrameChanged signals that a frame was pushed or popped and the dispatch loop must
	// reload its frame and register pointers.
	opFrameChanged

	// opDone signals that execution has finished successfully.
	opDone

	// opDivByZero signals a division by zero error.
	opDivByZero

	// opStackOverflow signals that the call stack exceeded maxCallDepth.
	opStackOverflow

	// opPanicError signals a runtime panic; the error is stored in vm.evalError.
	opPanicError
)

// opcodeHandler is the function signature for all opcode handlers. flatDispatchSwitch
// (Path-B fallback) and the asmgen-emitted Path-A trampolines share this signature so a
// single handler body covers both dispatch paths.
type opcodeHandler func(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult

// handleInvalidOpcode is the default handler for unregistered opcodes. It stores an
// errInvalidOpcode error in vm.evalError and signals a panic so the dispatch loop
// terminates cleanly.
//
// Takes vm (*VM) which receives the formatted error in vm.evalError.
// Takes instruction (instruction) which provides the invalid opcode for the error
// message.
//
// Returns opResult which is always opPanicError, terminating execution.
func handleInvalidOpcode(vm *VM, _ *callFrame, _ *Registers, instruction instruction) opResult {
	vm.evalError = fmt.Errorf("%w: %s (%d)", errInvalidOpcode, instruction.op, instruction.op)
	return opPanicError
}
