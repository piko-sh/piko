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

package asmgen_arch_arm64

import (
	"piko.sh/piko/wdk/asmgen"
)

// itoaBufCap is the initial capacity for the itoa conversion buffer.
const itoaBufCap = 4

// itoaBase is the numeric base used for itoa conversion.
const itoaBase = 10

// Jump table byte offsets for each opcode, computed as opcodeIndex * 8.
const (
	// jtOffsetNop is the jump table byte offset for the Nop opcode.
	jtOffsetNop = 0

	// jtOffsetMoveInt is the jump table byte offset for the MoveInt opcode.
	jtOffsetMoveInt = 16

	// jtOffsetMoveFloat is the jump table byte offset for the MoveFloat opcode.
	jtOffsetMoveFloat = 24

	// jtOffsetLoadIntConst is the jump table byte offset for the LoadIntConst opcode.
	jtOffsetLoadIntConst = 32

	// jtOffsetLoadFloatConst is the jump table byte offset for the LoadFloatConst opcode.
	jtOffsetLoadFloatConst = 40

	// jtOffsetLoadBool is the jump table byte offset for the LoadBool opcode.
	jtOffsetLoadBool = 48

	// jtOffsetLoadIntConstSmall is the jump table byte offset for the LoadIntConstSmall opcode.
	jtOffsetLoadIntConstSmall = 56

	// jtOffsetAddInt is the jump table byte offset for the AddInt opcode.
	jtOffsetAddInt = 64

	// jtOffsetSubInt is the jump table byte offset for the SubInt opcode.
	jtOffsetSubInt = 72

	// jtOffsetMulInt is the jump table byte offset for the MulInt opcode.
	jtOffsetMulInt = 80

	// jtOffsetDivInt is the jump table byte offset for the DivInt opcode.
	jtOffsetDivInt = 88

	// jtOffsetRemInt is the jump table byte offset for the RemInt opcode.
	jtOffsetRemInt = 96

	// jtOffsetNegInt is the jump table byte offset for the NegInt opcode.
	jtOffsetNegInt = 104

	// jtOffsetIncInt is the jump table byte offset for the IncInt opcode.
	jtOffsetIncInt = 112

	// jtOffsetDecInt is the jump table byte offset for the DecInt opcode.
	jtOffsetDecInt = 120

	// jtOffsetBitAnd is the jump table byte offset for the BitAnd opcode.
	jtOffsetBitAnd = 128

	// jtOffsetBitOr is the jump table byte offset for the BitOr opcode.
	jtOffsetBitOr = 136

	// jtOffsetBitXor is the jump table byte offset for the BitXor opcode.
	jtOffsetBitXor = 144

	// jtOffsetBitAndNot is the jump table byte offset for the BitAndNot opcode.
	jtOffsetBitAndNot = 152

	// jtOffsetBitNot is the jump table byte offset for the BitNot opcode.
	jtOffsetBitNot = 160

	// jtOffsetShiftLeft is the jump table byte offset for the ShiftLeft opcode.
	jtOffsetShiftLeft = 168

	// jtOffsetShiftRight is the jump table byte offset for the ShiftRight opcode.
	jtOffsetShiftRight = 176

	// jtOffsetAddFloat is the jump table byte offset for the AddFloat opcode.
	jtOffsetAddFloat = 184

	// jtOffsetSubFloat is the jump table byte offset for the SubFloat opcode.
	jtOffsetSubFloat = 192

	// jtOffsetMulFloat is the jump table byte offset for the MulFloat opcode.
	jtOffsetMulFloat = 200

	// jtOffsetDivFloat is the jump table byte offset for the DivFloat opcode.
	jtOffsetDivFloat = 208

	// jtOffsetNegFloat is the jump table byte offset for the NegFloat opcode.
	jtOffsetNegFloat = 216

	// jtOffsetEqInt is the jump table byte offset for the EqInt opcode.
	jtOffsetEqInt = 224

	// jtOffsetNeInt is the jump table byte offset for the NeInt opcode.
	jtOffsetNeInt = 232

	// jtOffsetLtInt is the jump table byte offset for the LtInt opcode.
	jtOffsetLtInt = 240

	// jtOffsetLeInt is the jump table byte offset for the LeInt opcode.
	jtOffsetLeInt = 248

	// jtOffsetGtInt is the jump table byte offset for the GtInt opcode.
	jtOffsetGtInt = 256

	// jtOffsetGeInt is the jump table byte offset for the GeInt opcode.
	jtOffsetGeInt = 264

	// jtOffsetEqFloat is the jump table byte offset for the EqFloat opcode.
	jtOffsetEqFloat = 272

	// jtOffsetNeFloat is the jump table byte offset for the NeFloat opcode.
	jtOffsetNeFloat = 280

	// jtOffsetLtFloat is the jump table byte offset for the LtFloat opcode.
	jtOffsetLtFloat = 288

	// jtOffsetLeFloat is the jump table byte offset for the LeFloat opcode.
	jtOffsetLeFloat = 296

	// jtOffsetGtFloat is the jump table byte offset for the GtFloat opcode.
	jtOffsetGtFloat = 304

	// jtOffsetGeFloat is the jump table byte offset for the GeFloat opcode.
	jtOffsetGeFloat = 312

	// jtOffsetIntToFloat is the jump table byte offset for the IntToFloat opcode.
	jtOffsetIntToFloat = 320

	// jtOffsetFloatToInt is the jump table byte offset for the FloatToInt opcode.
	jtOffsetFloatToInt = 328

	// jtOffsetNot is the jump table byte offset for the Not opcode.
	jtOffsetNot = 336

	// jtOffsetJump is the jump table byte offset for the Jump opcode.
	jtOffsetJump = 344

	// jtOffsetJumpIfTrue is the jump table byte offset for the JumpIfTrue opcode.
	jtOffsetJumpIfTrue = 352

	// jtOffsetJumpIfFalse is the jump table byte offset for the JumpIfFalse opcode.
	jtOffsetJumpIfFalse = 360

	// jtOffsetCallInline is the jump table byte offset for the CallInline opcode.
	jtOffsetCallInline = 368

	// jtOffsetReturnInline is the jump table byte offset for the ReturnInline opcode.
	jtOffsetReturnInline = 376

	// jtOffsetReturnVoidInline is the jump table byte offset for the ReturnVoidInline opcode.
	jtOffsetReturnVoidInline = 384

	// jtOffsetTailCallExit is the jump table byte offset for the TailCallExit opcode.
	jtOffsetTailCallExit = 392

	// jtOffsetSubIntConst is the jump table byte offset for the SubIntConst super-instruction.
	jtOffsetSubIntConst = 400

	// jtOffsetAddIntConst is the jump table byte offset for the AddIntConst super-instruction.
	jtOffsetAddIntConst = 408

	// jtOffsetLeIntConstJF is the jump table offset for
	// the LeIntConstJumpFalse super-instruction.
	jtOffsetLeIntConstJF = 416

	// jtOffsetLtIntConstJF is the jump table offset for
	// the LtIntConstJumpFalse super-instruction.
	jtOffsetLtIntConstJF = 424

	// jtOffsetEqIntConstJF is the jump table offset for
	// the EqIntConstJumpFalse super-instruction.
	jtOffsetEqIntConstJF = 432

	// jtOffsetEqIntConstJT is the jump table offset for
	// the EqIntConstJumpTrue super-instruction.
	jtOffsetEqIntConstJT = 440

	// jtOffsetGeIntConstJF is the jump table offset for
	// the GeIntConstJumpFalse super-instruction.
	jtOffsetGeIntConstJF = 448

	// jtOffsetGtIntConstJF is the jump table offset for
	// the GtIntConstJumpFalse super-instruction.
	jtOffsetGtIntConstJF = 456

	// jtOffsetMulIntConst is the jump table byte offset for the MulIntConst super-instruction.
	jtOffsetMulIntConst = 464

	// jtOffsetAddIntJump is the jump table byte offset for the AddIntJump super-instruction.
	jtOffsetAddIntJump = 472

	// jtOffsetIncIntJumpLt is the jump table byte offset for the IncIntJumpLt super-instruction.
	jtOffsetIncIntJumpLt = 480

	// jtOffsetMathSqrt is the jump table byte offset for the MathSqrt opcode.
	jtOffsetMathSqrt = 488

	// jtOffsetMathAbs is the jump table byte offset for the MathAbs opcode.
	jtOffsetMathAbs = 496

	// jtOffsetMathFloor is the jump table byte offset for the MathFloor opcode.
	jtOffsetMathFloor = 504

	// jtOffsetMathCeil is the jump table byte offset for the MathCeil opcode.
	jtOffsetMathCeil = 512

	// jtOffsetMathTrunc is the jump table byte offset for the MathTrunc opcode.
	jtOffsetMathTrunc = 520

	// jtOffsetMathRound is the jump table byte offset for the MathRound opcode.
	jtOffsetMathRound = 528

	// jtOffsetLenString is the jump table byte offset for the LenString opcode.
	jtOffsetLenString = 536

	// jtOffsetStringIndex is the jump table byte offset for the StringIndex opcode.
	jtOffsetStringIndex = 544

	// jtOffsetEqString is the jump table byte offset for the EqString opcode.
	jtOffsetEqString = 552

	// jtOffsetNeString is the jump table byte offset for the NeString opcode.
	jtOffsetNeString = 560

	// jtOffsetSliceString is the jump table byte offset for the SliceString opcode.
	jtOffsetSliceString = 568

	// jtOffsetStringIndexToInt is the jump table byte offset for the StringIndexToInt opcode.
	jtOffsetStringIndexToInt = 576

	// jtOffsetLenStringLtJF is the jump table byte offset for the LenStringLtJumpFalse opcode.
	jtOffsetLenStringLtJF = 584
)

// arm64InitOps implements InitialisationOperationsPort for ARM 64-bit
// Plan 9 assembly. Each method emits the complete handler body for
// dispatch loop initialisation, jump table setup, and exit handlers.
type arm64InitOps struct{}

// Ensure arm64InitOps implements InitialisationOperationsPort at compile time.
var _ asmgen.InitialisationOperationsPort = (*arm64InitOps)(nil)

// initArm64JumpTableEntry emits a MOVD pair that patches one entry in
// the jump table.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes handler (string) which is the handler function symbol name.
// Takes offset (int) which is the byte offset into the jump table.
func initArm64JumpTableEntry(e *asmgen.Emitter, handler string, offset int) {
	inst5(e, mnemonicMOVD, "$\xc2\xb7"+handler+"(SB), R1")
	inst5(e, mnemonicMOVD, "R1, "+itoa(offset)+"(R0)")
}

// itoa converts a small non-negative integer to its decimal string
// representation without importing strconv.
//
// Takes n (int) which is the non-negative integer to convert.
//
// Returns string which is the decimal string representation.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, itoaBufCap)
	for n > 0 {
		buf = append(buf, byte('0'+n%itoaBase))
		n /= itoaBase
	}

	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

// EmitInitJumpTable emits the full initJumpTable function body, filling all
// 256 entries with tier2Fallback then patching each Tier 1 opcode handler.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (o *arm64InitOps) EmitInitJumpTable(e *asmgen.Emitter) {
	o.emitJumpTableFillLoop(e)
	o.emitJumpTablePatchDataMovement(e)
	o.emitJumpTablePatchIntegerArithmetic(e)
	o.emitJumpTablePatchBitwise(e)
	o.emitJumpTablePatchFloatArithmetic(e)
	o.emitJumpTablePatchComparison(e)
	o.emitJumpTablePatchConversionAndLogic(e)
	o.emitJumpTablePatchControlFlow(e)
	o.emitJumpTablePatchSuperInstructions(e)
	o.emitJumpTablePatchMathBuiltins(e)
	o.emitJumpTablePatchStringOperations(e)

	e.Instruction("RET")
}

// EmitInitJumpTableSSE41 is a no-op on arm64, existing solely to
// satisfy the InitialisationOperationsPort interface.
func (*arm64InitOps) EmitInitJumpTableSSE41(_ *asmgen.Emitter) {}

// EmitDispatchLoop emits the dispatchLoop function body, loading DispatchContext
// fields into callee-saved registers and performing the first dispatch.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) EmitDispatchLoop(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "ctx+0(FP), R19")
	inst5(e, mnemonicMOVD, "0(R19), R22")
	inst5(e, mnemonicMOVD, "8(R19), R21")
	inst5(e, mnemonicMOVD, "16(R19), R20")
	inst5(e, mnemonicMOVD, "24(R19), R23")
	inst5(e, mnemonicMOVD, "40(R19), R24")
	inst5(e, mnemonicMOVD, "56(R19), R26")
	inst5(e, mnemonicMOVD, "88(R19), R25")
	e.Instruction("DISPATCH_NEXT()")
}

// EmitTier2Fallback emits the tier2Fallback handler body, decrementing the
// program counter and returning to Go with EXIT_TIER2.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) EmitTier2Fallback(e *asmgen.Emitter) {
	inst5(e, mnemonicSUB, "$1, R20, R20")
	inst5(e, mnemonicMOVD, "R20, 16(R19)")
	inst5(e, mnemonicMOVD, "$EXIT_TIER2, R0")
	inst5(e, mnemonicMOVD, "R0, 96(R19)")
	inst5(e, mnemonicMOVD, "R20, 104(R19)")
	e.Instruction("RET")
}

// EmitExitHandler emits an exit handler body for the given exit constant,
// decrementing the program counter and returning to Go with the specified reason.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
// Takes exitConstant (string) which is the exit reason constant name (e.g. "EXIT_CALL").
func (*arm64InitOps) EmitExitHandler(e *asmgen.Emitter, exitConstant string) {
	inst5(e, mnemonicSUB, "$1, R20, R20")
	inst5(e, mnemonicMOVD, "R20, 16(R19)")
	inst5(e, mnemonicMOVD, "$"+exitConstant+", R0")
	inst5(e, mnemonicMOVD, "R0, 96(R19)")
	inst5(e, mnemonicMOVD, "R20, 104(R19)")
	e.Instruction("RET")
}

// emitJumpTableFillLoop emits the loop that fills all 256 jump table
// entries with the tier2Fallback handler address.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTableFillLoop(e *asmgen.Emitter) {
	inst5(e, mnemonicMOVD, "table+0(FP), R0")
	e.Blank()
	inst5(e, mnemonicMOVD, "$\xc2\xb7tier2Fallback(SB), R1")
	inst5(e, mnemonicMOVD, "$256, R2")
	e.Blank()
	e.Label("initjt_fill")
	inst5(e, mnemonicMOVD, "R1, (R0)")
	inst5(e, mnemonicADD, "$8, R0, R0")
	inst5(e, mnemonicSUB, "$1, R2, R2")
	inst5(e, "CBNZ", "R2, initjt_fill")
	e.Blank()
	inst5(e, mnemonicMOVD, "table+0(FP), R0")
	e.Blank()
}

// emitJumpTablePatchDataMovement patches the jump table entries for
// data movement opcodes.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTablePatchDataMovement(e *asmgen.Emitter) {
	initArm64JumpTableEntry(e, "handlerNop", jtOffsetNop)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerMoveInt", jtOffsetMoveInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerMoveFloat", jtOffsetMoveFloat)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerLoadIntConst", jtOffsetLoadIntConst)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerLoadFloatConst", jtOffsetLoadFloatConst)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerLoadBool", jtOffsetLoadBool)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerLoadIntConstSmall", jtOffsetLoadIntConstSmall)
	e.Blank()
}

// emitJumpTablePatchIntegerArithmetic patches the jump table entries
// for integer arithmetic opcodes.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTablePatchIntegerArithmetic(e *asmgen.Emitter) {
	initArm64JumpTableEntry(e, "handlerAddInt", jtOffsetAddInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerSubInt", jtOffsetSubInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerMulInt", jtOffsetMulInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerDivInt", jtOffsetDivInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerRemInt", jtOffsetRemInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerNegInt", jtOffsetNegInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerIncInt", jtOffsetIncInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerDecInt", jtOffsetDecInt)
	e.Blank()
}

// emitJumpTablePatchBitwise patches the jump table entries for bitwise opcodes.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTablePatchBitwise(e *asmgen.Emitter) {
	initArm64JumpTableEntry(e, "handlerBitAnd", jtOffsetBitAnd)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerBitOr", jtOffsetBitOr)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerBitXor", jtOffsetBitXor)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerBitAndNot", jtOffsetBitAndNot)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerBitNot", jtOffsetBitNot)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerShiftLeft", jtOffsetShiftLeft)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerShiftRight", jtOffsetShiftRight)
	e.Blank()
}

// emitJumpTablePatchFloatArithmetic patches the jump table entries for
// floating-point arithmetic opcodes.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTablePatchFloatArithmetic(e *asmgen.Emitter) {
	initArm64JumpTableEntry(e, "handlerAddFloat", jtOffsetAddFloat)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerSubFloat", jtOffsetSubFloat)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerMulFloat", jtOffsetMulFloat)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerDivFloat", jtOffsetDivFloat)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerNegFloat", jtOffsetNegFloat)
	e.Blank()
}

// emitJumpTablePatchComparison patches the jump table entries for comparison opcodes.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTablePatchComparison(e *asmgen.Emitter) {
	initArm64JumpTableEntry(e, "handlerEqInt", jtOffsetEqInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerNeInt", jtOffsetNeInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerLtInt", jtOffsetLtInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerLeInt", jtOffsetLeInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerGtInt", jtOffsetGtInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerGeInt", jtOffsetGeInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerEqFloat", jtOffsetEqFloat)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerNeFloat", jtOffsetNeFloat)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerLtFloat", jtOffsetLtFloat)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerLeFloat", jtOffsetLeFloat)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerGtFloat", jtOffsetGtFloat)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerGeFloat", jtOffsetGeFloat)
	e.Blank()
}

// emitJumpTablePatchConversionAndLogic patches the jump table entries
// for type conversion and logic opcodes.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTablePatchConversionAndLogic(e *asmgen.Emitter) {
	initArm64JumpTableEntry(e, "handlerIntToFloat", jtOffsetIntToFloat)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerFloatToInt", jtOffsetFloatToInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerNot", jtOffsetNot)
	e.Blank()
}

// emitJumpTablePatchControlFlow patches the jump table entries for
// control flow opcodes.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTablePatchControlFlow(e *asmgen.Emitter) {
	initArm64JumpTableEntry(e, "handlerJump", jtOffsetJump)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerJumpIfTrue", jtOffsetJumpIfTrue)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerJumpIfFalse", jtOffsetJumpIfFalse)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerCallInline", jtOffsetCallInline)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerReturnInline", jtOffsetReturnInline)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerReturnVoidInline", jtOffsetReturnVoidInline)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerTailCallExit", jtOffsetTailCallExit)
	e.Blank()
}

// emitJumpTablePatchSuperInstructions patches the jump table entries
// for super-instructions (fused opcode pairs).
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTablePatchSuperInstructions(e *asmgen.Emitter) {
	initArm64JumpTableEntry(e, "handlerSubIntConst", jtOffsetSubIntConst)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerAddIntConst", jtOffsetAddIntConst)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerLeIntConstJumpFalse", jtOffsetLeIntConstJF)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerLtIntConstJumpFalse", jtOffsetLtIntConstJF)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerEqIntConstJumpFalse", jtOffsetEqIntConstJF)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerEqIntConstJumpTrue", jtOffsetEqIntConstJT)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerGeIntConstJumpFalse", jtOffsetGeIntConstJF)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerGtIntConstJumpFalse", jtOffsetGtIntConstJF)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerMulIntConst", jtOffsetMulIntConst)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerAddIntJump", jtOffsetAddIntJump)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerIncIntJumpLt", jtOffsetIncIntJumpLt)
	e.Blank()
}

// emitJumpTablePatchMathBuiltins patches the jump table entries for
// math built-in opcodes.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTablePatchMathBuiltins(e *asmgen.Emitter) {
	initArm64JumpTableEntry(e, "handlerMathSqrt", jtOffsetMathSqrt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerMathAbs", jtOffsetMathAbs)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerMathFloor", jtOffsetMathFloor)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerMathCeil", jtOffsetMathCeil)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerMathTrunc", jtOffsetMathTrunc)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerMathRound", jtOffsetMathRound)
	e.Blank()
}

// emitJumpTablePatchStringOperations patches the jump table entries for
// string opcodes.
//
// Takes e (*asmgen.Emitter) which receives the emitted instructions.
func (*arm64InitOps) emitJumpTablePatchStringOperations(e *asmgen.Emitter) {
	initArm64JumpTableEntry(e, "handlerLenString", jtOffsetLenString)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerStringIndex", jtOffsetStringIndex)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerEqString", jtOffsetEqString)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerNeString", jtOffsetNeString)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerSliceString", jtOffsetSliceString)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerStringIndexToInt", jtOffsetStringIndexToInt)
	e.Blank()
	initArm64JumpTableEntry(e, "handlerLenStringLtJumpFalse", jtOffsetLenStringLtJF)
	e.Blank()
}
