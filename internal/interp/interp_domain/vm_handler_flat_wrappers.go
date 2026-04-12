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

// handleFlatSubOpMoveInt reshapes a Pattern C move and dispatches to handleMoveInt.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveInt(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveFloat reshapes a Pattern C move and dispatches to handleMoveFloat.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveFloat(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveFloat(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveString reshapes a Pattern C move and dispatches to handleMoveString.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveString(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveString(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveBool reshapes a Pattern C move and dispatches to handleMoveBool.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveBool(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveBool(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveUint reshapes a Pattern C move and dispatches to handleMoveUint.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveUint(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveSliceInt reshapes a Pattern C move for handleMoveSliceInt.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveSliceInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveSliceInt(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveSliceFloat reshapes a Pattern C move for handleMoveSliceFloat.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveSliceFloat(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveSliceFloat(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveSliceString reshapes a Pattern C move for handleMoveSliceString.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveSliceString(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveSliceString(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveSliceBool reshapes a Pattern C move for handleMoveSliceBool.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveSliceBool(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveSliceBool(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveSliceUint reshapes a Pattern C move for handleMoveSliceUint.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveSliceUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveSliceUint(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveSliceByte reshapes a Pattern C move for handleMoveSliceByte.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveSliceByte(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveSliceByte(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveIntToGeneral reshapes a Pattern C move for handleMoveIntToGeneral.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveIntToGeneral(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveIntToGeneral(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveGeneralToInt reshapes a Pattern C move for handleMoveGeneralToInt.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveGeneralToInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveGeneralToInt(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveFloatToGeneral reshapes a Pattern C move for
// handleMoveFloatToGeneral.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveFloatToGeneral(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveFloatToGeneral(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveGeneralToFloat reshapes a Pattern C move for
// handleMoveGeneralToFloat.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveGeneralToFloat(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveGeneralToFloat(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveStringToGeneral reshapes a Pattern C string-to-general move.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveStringToGeneral(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveStringToGeneral(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMoveGeneralToString reshapes a Pattern C general-to-string move.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMoveGeneralToString(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMoveGeneralToString(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpNegInt reshapes a Pattern C negation and dispatches to handleNegInt.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpNegInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleNegInt(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpNegFloat reshapes a Pattern C negation and dispatches to handleNegFloat.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpNegFloat(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleNegFloat(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpBitNot reshapes a Pattern C bitwise not and dispatches to handleBitNot.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpBitNot(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleBitNot(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpBitNotUint reshapes a Pattern C bitwise not for handleBitNotUint.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpBitNotUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleBitNotUint(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpIntToFloat reshapes a Pattern C conversion for handleIntToFloat.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpIntToFloat(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleIntToFloat(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpFloatToInt reshapes a Pattern C conversion for handleFloatToInt.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpFloatToInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleFloatToInt(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpNot reshapes a Pattern C logical not and dispatches to handleNot.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpNot(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleNot(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpBoolToInt reshapes a Pattern C conversion for handleBoolToInt.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpBoolToInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleBoolToInt(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpIntToBool reshapes a Pattern C conversion for handleIntToBool.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpIntToBool(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleIntToBool(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpIntToUint reshapes a Pattern C conversion for handleIntToUint.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpIntToUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleIntToUint(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpUintToInt reshapes a Pattern C conversion for handleUintToInt.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpUintToInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleUintToInt(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpUintToFloat reshapes a Pattern C conversion for handleUintToFloat.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpUintToFloat(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleUintToFloat(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpFloatToUint reshapes a Pattern C conversion for handleFloatToUint.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpFloatToUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleFloatToUint(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMathSqrt reshapes a Pattern C maths call and dispatches to
// handleMathSqrt.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMathSqrt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMathSqrt(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMathAbs reshapes a Pattern C maths call and dispatches to handleMathAbs.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMathAbs(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMathAbs(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMathFloor reshapes a Pattern C maths call for handleMathFloor.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMathFloor(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMathFloor(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMathCeil reshapes a Pattern C maths call and dispatches to
// handleMathCeil.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMathCeil(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMathCeil(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMathTrunc reshapes a Pattern C maths call for handleMathTrunc.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMathTrunc(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMathTrunc(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMathRound reshapes a Pattern C maths call for handleMathRound.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMathRound(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMathRound(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpLenString reshapes a Pattern C length op and dispatches to
// handleLenString.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpLenString(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleLenString(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpRuneToString reshapes a Pattern C conversion for handleRuneToString.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpRuneToString(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleRuneToString(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpStrToUpper reshapes a Pattern C string op for handleStrToUpper.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpStrToUpper(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleStrToUpper(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpStrToLower reshapes a Pattern C string op for handleStrToLower.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpStrToLower(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleStrToLower(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpStrTrimSpace reshapes a Pattern C string op for handleStrTrimSpace.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpStrTrimSpace(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleStrTrimSpace(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpLen reshapes a Pattern C length op and dispatches to handleLen.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpLen(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleLen(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpStringToBytes reshapes a Pattern C conversion for handleStringToBytes.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpStringToBytes(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleStringToBytes(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpUnsafeStringData reshapes a Pattern C unsafe op for
// handleUnsafeStringData.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpUnsafeStringData(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleUnsafeStringData(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpUnsafeSliceData reshapes a Pattern C unsafe op for
// handleUnsafeSliceData.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpUnsafeSliceData(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleUnsafeSliceData(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpAddUintConst reshapes a Pattern C constant-add for handleAddUintConst.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpAddUintConst(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleAddUintConst(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpSubUintConst reshapes a Pattern C constant-sub for handleSubUintConst.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpSubUintConst(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleSubUintConst(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpBitAndUintConst reshapes a Pattern C masking op for
// handleBitAndUintConst.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpBitAndUintConst(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleBitAndUintConst(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpIncIntJumpLt reshapes a Pattern C fused increment for
// handleIncIntJumpLt.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpIncIntJumpLt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleIncIntJumpLt(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpLenStringLtJumpFalse reshapes a Pattern C compare-jump dispatch.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpLenStringLtJumpFalse(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleLenStringLtJumpFalse(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpLoadZero reshapes a Pattern C load-zero and dispatches to
// handleLoadZero.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpLoadZero(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleLoadZero(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMakeMap reshapes a Pattern C map-make and dispatches to handleMakeMap.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMakeMap(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMakeMap(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMakeChannel reshapes a Pattern C channel-make for handleMakeChannel.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMakeChannel(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMakeChannel(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpMapDelete reshapes a Pattern C map delete for handleMapDelete.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpMapDelete(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleMapDelete(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpChannelReceive reshapes a Pattern C receive for handleChannelReceive.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpChannelReceive(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleChannelReceive(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpGetMethod reshapes a Pattern C method lookup for handleGetMethod.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpGetMethod(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleGetMethod(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpSpill reshapes a Pattern C spill and dispatches to handleSpill.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpSpill(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleSpill(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpReload reshapes a Pattern C reload and dispatches to handleReload.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operands.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpReload(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleReload(vm, frame, registers, instruction{a: instr.b, b: instr.c})
}

// handleFlatSubOpTier2IncInt reshapes a tier-2 Pattern C op and dispatches to
// handleIncInt.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operand.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpTier2IncInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleIncInt(vm, frame, registers, instruction{a: instr.c})
}

// handleFlatSubOpTier2DecInt reshapes a tier-2 Pattern C op and dispatches to
// handleDecInt.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operand.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpTier2DecInt(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleDecInt(vm, frame, registers, instruction{a: instr.c})
}

// handleFlatSubOpTier2IncUint reshapes a tier-2 Pattern C op for handleIncUint.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operand.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpTier2IncUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleIncUint(vm, frame, registers, instruction{a: instr.c})
}

// handleFlatSubOpTier2DecUint reshapes a tier-2 Pattern C op for handleDecUint.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operand.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpTier2DecUint(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleDecUint(vm, frame, registers, instruction{a: instr.c})
}

// handleFlatSubOpTier2Panic reshapes a tier-2 Pattern C op and dispatches to handlePanic.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operand.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpTier2Panic(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handlePanic(vm, frame, registers, instruction{a: instr.c})
}

// handleFlatSubOpTier2Recover reshapes a tier-2 Pattern C op for handleRecover.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operand.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpTier2Recover(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleRecover(vm, frame, registers, instruction{a: instr.c})
}

// handleFlatSubOpTier2SetZero reshapes a tier-2 Pattern C op for handleSetZero.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operand.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpTier2SetZero(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleSetZero(vm, frame, registers, instruction{a: instr.c})
}

// handleFlatSubOpTier2ChannelClose reshapes a tier-2 Pattern C op for handleChannelClose.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operand.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpTier2ChannelClose(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleChannelClose(vm, frame, registers, instruction{a: instr.c})
}

// handleFlatSubOpTier2LoadNil reshapes a tier-2 Pattern C op for handleLoadNil.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operand.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpTier2LoadNil(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleLoadNil(vm, frame, registers, instruction{a: instr.c})
}

// handleFlatSubOpTier2Return reshapes a tier-2 Pattern C op and dispatches to
// handleReturn.
//
// Takes vm (*VM) which owns the executing program state.
// Takes frame (*callFrame) which is the current call frame.
// Takes registers (*Registers) which holds the active register banks.
// Takes instr (instruction) which carries the Pattern C operand.
//
// Returns opResult which propagates the inner handler outcome.
func handleFlatSubOpTier2Return(vm *VM, frame *callFrame, registers *Registers, instr instruction) opResult {
	return handleReturn(vm, frame, registers, instruction{a: instr.c})
}
