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

//go:build !safe && !(js && wasm) && amd64

package interp_domain

import "golang.org/x/sys/cpu"

// asmJumpTable is the global dispatch table, initialised once.
var asmJumpTable [opcodeTableSize]uintptr

// dispatchLoop is the ASM threaded dispatch loop that executes Tier 1 opcodes
// directly in assembly, modifying registers through the DispatchContext.
// When it
// encounters a Tier 2 opcode or reaches the end of the code body, it writes the
// exit reason and returns to Go.
//
// Takes ctx (*DispatchContext) which provides the register file and program
// counter state for dispatch.
//
//go:noescape
func dispatchLoop(ctx *DispatchContext)

// initJumpTable populates a 256-entry dispatch table with handler
// addresses for each opcode. Tier 1 opcodes get ASM handler addresses;
// all other entries point to the Tier 2 fallback handler.
//
// Takes table (*[opcodeTableSize]uintptr) which is the fixed-size array to
// populate with handler addresses.
//
//go:noescape
func initJumpTable(table *[opcodeTableSize]uintptr)

// tier2Fallback and related functions are ASM handlers declared
// here for linker resolution. Each is a TEXT symbol in
// vm_dispatch_amd64.s - jump targets, not called from Go.
//
//go:noescape
func tier2Fallback() //nolint:unused // ASM handler

// handlerNop performs a no-operation instruction in the virtual machine
// dispatch loop.
//
//go:noescape
func handlerNop() //nolint:unused // ASM handler

// handlerMoveInt copies an integer value between registers in the
// dispatch loop.
//
//go:noescape
func handlerMoveInt() //nolint:unused // ASM handler

// handlerMoveFloat copies a floating-point value between registers in the
// dispatch loop.
//
//go:noescape
func handlerMoveFloat() //nolint:unused // ASM handler

// handlerLoadIntConst loads an integer constant into a register.
//
//go:noescape
func handlerLoadIntConst() //nolint:unused // ASM handler

// handlerLoadBool loads a boolean constant into a register.
//
//go:noescape
func handlerLoadBool() //nolint:unused // ASM handler

// handlerAddInt performs integer addition of two registers in the
// dispatch loop.
//
//go:noescape
func handlerAddInt() //nolint:unused // ASM handler

// handlerSubInt performs integer subtraction of two registers in the
// dispatch loop.
//
//go:noescape
func handlerSubInt() //nolint:unused // ASM handler

// handlerMulInt performs integer multiplication of two registers in the
// dispatch loop.
//
//go:noescape
func handlerMulInt() //nolint:unused // ASM handler

// handlerDivInt performs integer division of two registers in the
// dispatch loop.
//
//go:noescape
func handlerDivInt() //nolint:unused // ASM handler

// handlerRemInt performs integer remainder of two registers in the
// dispatch loop.
//
//go:noescape
func handlerRemInt() //nolint:unused // ASM handler

// handlerNegInt negates an integer value in a register in the dispatch loop.
//
//go:noescape
func handlerNegInt() //nolint:unused // ASM handler

// handlerBitAnd performs bitwise AND of two registers in the dispatch loop.
//
//go:noescape
func handlerBitAnd() //nolint:unused // ASM handler

// handlerBitOr performs bitwise OR of two registers in the dispatch loop.
//
//go:noescape
func handlerBitOr() //nolint:unused // ASM handler

// handlerBitXor performs bitwise XOR of two registers in the dispatch loop.
//
//go:noescape
func handlerBitXor() //nolint:unused // ASM handler

// handlerBitAndNot performs bitwise AND-NOT of two registers in the
// dispatch loop.
//
//go:noescape
func handlerBitAndNot() //nolint:unused // ASM handler

// handlerBitNot performs bitwise NOT on a register value in the dispatch loop.
//
//go:noescape
func handlerBitNot() //nolint:unused // ASM handler

// handlerShiftLeft performs a left bit shift on a register in the
// dispatch loop.
//
//go:noescape
func handlerShiftLeft() //nolint:unused // ASM handler

// handlerShiftRight performs a right bit shift on a register in the
// dispatch loop.
//
//go:noescape
func handlerShiftRight() //nolint:unused // ASM handler

// handlerAddFloat performs floating-point addition of two registers in the
// dispatch loop.
//
//go:noescape
func handlerAddFloat() //nolint:unused // ASM handler

// handlerSubFloat performs floating-point subtraction of two registers in
// the dispatch loop.
//
//go:noescape
func handlerSubFloat() //nolint:unused // ASM handler

// handlerMulFloat performs floating-point multiplication of two registers
// in the dispatch loop.
//
//go:noescape
func handlerMulFloat() //nolint:unused // ASM handler

// handlerDivFloat performs floating-point division of two registers in the
// dispatch loop.
//
//go:noescape
func handlerDivFloat() //nolint:unused // ASM handler

// handlerNegFloat negates a floating-point value in a register in the
// dispatch loop.
//
//go:noescape
func handlerNegFloat() //nolint:unused // ASM handler

// handlerEqInt performs integer equality comparison of two registers in
// the dispatch loop.
//
//go:noescape
func handlerEqInt() //nolint:unused // ASM handler

// handlerNeInt performs integer not-equal comparison of two registers in
// the dispatch loop.
//
//go:noescape
func handlerNeInt() //nolint:unused // ASM handler

// handlerLtInt performs integer less-than comparison of two registers in
// the dispatch loop.
//
//go:noescape
func handlerLtInt() //nolint:unused // ASM handler

// handlerLeInt performs integer less-than-or-equal comparison of two
// registers in the dispatch loop.
//
//go:noescape
func handlerLeInt() //nolint:unused // ASM handler

// handlerGtInt performs integer greater-than comparison of two registers
// in the dispatch loop.
//
//go:noescape
func handlerGtInt() //nolint:unused // ASM handler

// handlerGeInt performs integer greater-than-or-equal comparison of two
// registers in the dispatch loop.
//
//go:noescape
func handlerGeInt() //nolint:unused // ASM handler

// handlerNot performs logical NOT on a boolean register value in the
// dispatch loop.
//
//go:noescape
func handlerNot() //nolint:unused // ASM handler

// handlerJump performs an unconditional jump to a target offset in the
// dispatch loop.
//
//go:noescape
func handlerJump() //nolint:unused // ASM handler

// handlerJumpIfTrue performs a conditional jump when the boolean register
// is true.
//
//go:noescape
func handlerJumpIfTrue() //nolint:unused // ASM handler

// handlerJumpIfFalse performs a conditional jump when the boolean register
// is false.
//
//go:noescape
func handlerJumpIfFalse() //nolint:unused // ASM handler

// handlerCallExit exits the dispatch loop to perform a function call in Go.
//
//go:noescape
func handlerCallExit() //nolint:unused // ASM handler

// handlerReturnExit exits the dispatch loop to return a value to the caller.
//
//go:noescape
func handlerReturnExit() //nolint:unused // ASM handler

// handlerReturnVoidExit exits the dispatch loop to perform a void return
// to the caller.
//
//go:noescape
func handlerReturnVoidExit() //nolint:unused // ASM handler

// handlerTailCallExit exits the dispatch loop to perform a tail call in Go.
//
//go:noescape
func handlerTailCallExit() //nolint:unused // ASM handler

// handlerCallInline performs an inline function call within the dispatch loop.
//
//go:noescape
func handlerCallInline() //nolint:unused // ASM handler

// handlerReturnInline performs an inline return with a value within the
// dispatch loop.
//
//go:noescape
func handlerReturnInline() //nolint:unused // ASM handler

// handlerReturnVoidInline performs an inline void return within the
// dispatch loop.
//
//go:noescape
func handlerReturnVoidInline() //nolint:unused // ASM handler

// handlerSubIntConst subtracts an immediate integer constant from a
// register value.
//
//go:noescape
func handlerSubIntConst() //nolint:unused // ASM handler

// handlerAddIntConst adds an immediate integer constant to a register value.
//
//go:noescape
func handlerAddIntConst() //nolint:unused // ASM handler

// handlerLeIntConstJumpFalse compares a register against a constant and
// jumps if not less-or-equal.
//
//go:noescape
func handlerLeIntConstJumpFalse() //nolint:unused // ASM handler

// handlerLtIntConstJumpFalse compares a register against a constant and
// jumps if not less-than.
//
//go:noescape
func handlerLtIntConstJumpFalse() //nolint:unused // ASM handler

// handlerIncInt increments an integer register value by one in the
// dispatch loop.
//
//go:noescape
func handlerIncInt() //nolint:unused // ASM handler

// handlerDecInt decrements an integer register value by one in the
// dispatch loop.
//
//go:noescape
func handlerDecInt() //nolint:unused // ASM handler

// handlerEqIntConstJumpFalse compares a register against a constant and
// jumps if not equal.
//
//go:noescape
func handlerEqIntConstJumpFalse() //nolint:unused // ASM handler

// handlerEqIntConstJumpTrue compares a register against a constant and
// jumps if equal.
//
//go:noescape
func handlerEqIntConstJumpTrue() //nolint:unused // ASM handler

// handlerGeIntConstJumpFalse compares a register against a constant and
// jumps if not greater-or-equal.
//
//go:noescape
func handlerGeIntConstJumpFalse() //nolint:unused // ASM handler

// handlerGtIntConstJumpFalse compares a register against a constant and
// jumps if not greater-than.
//
//go:noescape
func handlerGtIntConstJumpFalse() //nolint:unused // ASM handler

// handlerMulIntConst multiplies a register value by an immediate integer
// constant.
//
//go:noescape
func handlerMulIntConst() //nolint:unused // ASM handler

// handlerAddIntJump performs integer addition followed by an
// unconditional jump.
//
//go:noescape
func handlerAddIntJump() //nolint:unused // ASM handler

// handlerIncIntJumpLt increments a register and jumps if the result is
// less than a limit.
//
//go:noescape
func handlerIncIntJumpLt() //nolint:unused // ASM handler

// handlerLoadIntConstSmall loads a small integer constant encoded in the
// instruction operand.
//
//go:noescape
func handlerLoadIntConstSmall() //nolint:unused // ASM handler

// handlerLoadFloatConst loads a floating-point constant into a register.
//
//go:noescape
func handlerLoadFloatConst() //nolint:unused // ASM handler

// handlerEqFloat performs floating-point equality comparison of two
// registers in the dispatch loop.
//
//go:noescape
func handlerEqFloat() //nolint:unused // ASM handler

// handlerNeFloat performs floating-point not-equal comparison of two
// registers in the dispatch loop.
//
//go:noescape
func handlerNeFloat() //nolint:unused // ASM handler

// handlerLtFloat performs floating-point less-than comparison of two
// registers in the dispatch loop.
//
//go:noescape
func handlerLtFloat() //nolint:unused // ASM handler

// handlerLeFloat performs floating-point less-than-or-equal comparison
// of two registers in the dispatch loop.
//
//go:noescape
func handlerLeFloat() //nolint:unused // ASM handler

// handlerGtFloat performs floating-point greater-than comparison of two
// registers in the dispatch loop.
//
//go:noescape
func handlerGtFloat() //nolint:unused // ASM handler

// handlerGeFloat performs floating-point greater-than-or-equal comparison
// of two registers in the dispatch loop.
//
//go:noescape
func handlerGeFloat() //nolint:unused // ASM handler

// handlerIntToFloat converts an integer register value to floating-point
// in the dispatch loop.
//
//go:noescape
func handlerIntToFloat() //nolint:unused // ASM handler

// handlerFloatToInt converts a floating-point register value to integer
// in the dispatch loop.
//
//go:noescape
func handlerFloatToInt() //nolint:unused // ASM handler

// handlerMathSqrt computes the square root of a floating-point register value.
//
//go:noescape
func handlerMathSqrt() //nolint:unused // ASM handler

// handlerMathAbs computes the absolute value of a floating-point register
// value.
//
//go:noescape
func handlerMathAbs() //nolint:unused // ASM handler

// handlerLenString computes the length of a string register value.
//
//go:noescape
func handlerLenString() //nolint:unused // ASM handler

// handlerStringIndex loads a byte from a string as a uint64.
//
//go:noescape
func handlerStringIndex() //nolint:unused // ASM handler

// handlerEqString compares two strings for equality.
//
//go:noescape
func handlerEqString() //nolint:unused // ASM handler

// handlerNeString compares two strings for inequality.
//
//go:noescape
func handlerNeString() //nolint:unused // ASM handler

// handlerSliceString slices a string with optional low and high bounds.
//
//go:noescape
func handlerSliceString() //nolint:unused // ASM handler

// handlerStringIndexToInt loads a byte from a string as an int64.
//
//go:noescape
func handlerStringIndexToInt() //nolint:unused // ASM handler

// handlerLenStringLtJumpFalse compares an int against a string length and
// conditionally jumps.
//
//go:noescape
func handlerLenStringLtJumpFalse() //nolint:unused // ASM handler

// handlerMathFloor computes the floor of a floating-point register value.
//
//go:noescape
func handlerMathFloor() //nolint:unused // ASM handler

// handlerMathCeil computes the ceiling of a floating-point register value.
//
//go:noescape
func handlerMathCeil() //nolint:unused // ASM handler

// handlerMathTrunc truncates a floating-point register value toward zero.
//
//go:noescape
func handlerMathTrunc() //nolint:unused // ASM handler

// initJumpTableSSE41 patches Floor, Ceil, Trunc entries into the
// dispatch table. Only called when the CPU supports SSE4.1.
//
// Takes table (*[opcodeTableSize]uintptr) which is the fixed-size array to
// patch with SSE4.1 handler addresses.
//
//go:noescape
func initJumpTableSSE41(table *[opcodeTableSize]uintptr)

func init() {
	initJumpTable(&asmJumpTable)
	if cpu.X86.HasSSE41 {
		initJumpTableSSE41(&asmJumpTable)
	}
}
