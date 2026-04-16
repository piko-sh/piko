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

package asmgen_arch_amd64

import (
	"fmt"

	"piko.sh/piko/wdk/asmgen"
)

const (
	// mnemonicRET represents the RET assembly mnemonic.
	mnemonicRET = "RET"

	// jtOffsetMoveInt is the jump table byte offset for the MoveInt handler.
	jtOffsetMoveInt = 16

	// jtOffsetMoveFloat is the jump table byte offset for the MoveFloat handler.
	jtOffsetMoveFloat = 24

	// jtOffsetLoadIntConst is the jump table byte offset for the LoadIntConst handler.
	jtOffsetLoadIntConst = 32

	// jtOffsetLoadFloatConst is the jump table byte offset for the LoadFloatConst handler.
	jtOffsetLoadFloatConst = 40

	// jtOffsetLoadBool is the jump table byte offset for the LoadBool handler.
	jtOffsetLoadBool = 48

	// jtOffsetLoadIntConstSmall is the jump table byte offset for the LoadIntConstSmall handler.
	jtOffsetLoadIntConstSmall = 56

	// jtOffsetAddInt is the jump table byte offset for the AddInt handler.
	jtOffsetAddInt = 64

	// jtOffsetSubInt is the jump table byte offset for the SubInt handler.
	jtOffsetSubInt = 72

	// jtOffsetMulInt is the jump table byte offset for the MulInt handler.
	jtOffsetMulInt = 80

	// jtOffsetDivInt is the jump table byte offset for the DivInt handler.
	jtOffsetDivInt = 88

	// jtOffsetRemInt is the jump table byte offset for the RemInt handler.
	jtOffsetRemInt = 96

	// jtOffsetNegInt is the jump table byte offset for the NegInt handler.
	jtOffsetNegInt = 104

	// jtOffsetIncInt is the jump table byte offset for the IncInt handler.
	jtOffsetIncInt = 112

	// jtOffsetDecInt is the jump table byte offset for the DecInt handler.
	jtOffsetDecInt = 120

	// jtOffsetBitAnd is the jump table byte offset for the BitAnd handler.
	jtOffsetBitAnd = 128

	// jtOffsetBitOr is the jump table byte offset for the BitOr handler.
	jtOffsetBitOr = 136

	// jtOffsetBitXor is the jump table byte offset for the BitXor handler.
	jtOffsetBitXor = 144

	// jtOffsetBitAndNot is the jump table byte offset for the BitAndNot handler.
	jtOffsetBitAndNot = 152

	// jtOffsetBitNot is the jump table byte offset for the BitNot handler.
	jtOffsetBitNot = 160

	// jtOffsetShiftLeft is the jump table byte offset for the ShiftLeft handler.
	jtOffsetShiftLeft = 168

	// jtOffsetShiftRight is the jump table byte offset for the ShiftRight handler.
	jtOffsetShiftRight = 176

	// jtOffsetAddFloat is the jump table byte offset for the AddFloat handler.
	jtOffsetAddFloat = 184

	// jtOffsetSubFloat is the jump table byte offset for the SubFloat handler.
	jtOffsetSubFloat = 192

	// jtOffsetMulFloat is the jump table byte offset for the MulFloat handler.
	jtOffsetMulFloat = 200

	// jtOffsetDivFloat is the jump table byte offset for the DivFloat handler.
	jtOffsetDivFloat = 208

	// jtOffsetNegFloat is the jump table byte offset for the NegFloat handler.
	jtOffsetNegFloat = 216

	// jtOffsetEqInt is the jump table byte offset for the EqInt handler.
	jtOffsetEqInt = 224

	// jtOffsetNeInt is the jump table byte offset for the NeInt handler.
	jtOffsetNeInt = 232

	// jtOffsetLtInt is the jump table byte offset for the LtInt handler.
	jtOffsetLtInt = 240

	// jtOffsetLeInt is the jump table byte offset for the LeInt handler.
	jtOffsetLeInt = 248

	// jtOffsetGtInt is the jump table byte offset for the GtInt handler.
	jtOffsetGtInt = 256

	// jtOffsetGeInt is the jump table byte offset for the GeInt handler.
	jtOffsetGeInt = 264

	// jtOffsetEqFloat is the jump table byte offset for the EqFloat handler.
	jtOffsetEqFloat = 272

	// jtOffsetNeFloat is the jump table byte offset for the NeFloat handler.
	jtOffsetNeFloat = 280

	// jtOffsetLtFloat is the jump table byte offset for the LtFloat handler.
	jtOffsetLtFloat = 288

	// jtOffsetLeFloat is the jump table byte offset for the LeFloat handler.
	jtOffsetLeFloat = 296

	// jtOffsetGtFloat is the jump table byte offset for the GtFloat handler.
	jtOffsetGtFloat = 304

	// jtOffsetGeFloat is the jump table byte offset for the GeFloat handler.
	jtOffsetGeFloat = 312

	// jtOffsetIntToFloat is the jump table byte offset for the IntToFloat handler.
	jtOffsetIntToFloat = 320

	// jtOffsetFloatToInt is the jump table byte offset for the FloatToInt handler.
	jtOffsetFloatToInt = 328

	// jtOffsetNot is the jump table byte offset for the Not handler.
	jtOffsetNot = 336

	// jtOffsetJump is the jump table byte offset for the Jump handler.
	jtOffsetJump = 344

	// jtOffsetJumpIfTrue is the jump table byte offset for the JumpIfTrue handler.
	jtOffsetJumpIfTrue = 352

	// jtOffsetJumpIfFalse is the jump table byte offset for the JumpIfFalse handler.
	jtOffsetJumpIfFalse = 360

	// jtOffsetCallInline is the jump table byte offset for the CallInline handler.
	jtOffsetCallInline = 368

	// jtOffsetReturnInline is the jump table byte offset for the ReturnInline handler.
	jtOffsetReturnInline = 376

	// jtOffsetReturnVoidInline is the jump table byte offset for the ReturnVoidInline handler.
	jtOffsetReturnVoidInline = 384

	// jtOffsetTailCallExit is the jump table byte offset for the TailCallExit handler.
	jtOffsetTailCallExit = 392

	// jtOffsetSubIntConst is the jump table byte offset for the SubIntConst super-instruction.
	jtOffsetSubIntConst = 400

	// jtOffsetAddIntConst is the jump table byte offset for the AddIntConst super-instruction.
	jtOffsetAddIntConst = 408

	// jtOffsetLeIntConstJumpFalse is the jump table offset
	// for the LeIntConstJumpFalse super-instruction.
	jtOffsetLeIntConstJumpFalse = 416

	// jtOffsetLtIntConstJumpFalse is the jump table offset
	// for the LtIntConstJumpFalse super-instruction.
	jtOffsetLtIntConstJumpFalse = 424

	// jtOffsetEqIntConstJumpFalse is the jump table offset
	// for the EqIntConstJumpFalse super-instruction.
	jtOffsetEqIntConstJumpFalse = 432

	// jtOffsetEqIntConstJumpTrue is the jump table offset
	// for the EqIntConstJumpTrue super-instruction.
	jtOffsetEqIntConstJumpTrue = 440

	// jtOffsetGeIntConstJumpFalse is the jump table offset
	// for the GeIntConstJumpFalse super-instruction.
	jtOffsetGeIntConstJumpFalse = 448

	// jtOffsetGtIntConstJumpFalse is the jump table offset
	// for the GtIntConstJumpFalse super-instruction.
	jtOffsetGtIntConstJumpFalse = 456

	// jtOffsetMulIntConst is the jump table byte offset for the MulIntConst super-instruction.
	jtOffsetMulIntConst = 464

	// jtOffsetAddIntJump is the jump table byte offset for the AddIntJump super-instruction.
	jtOffsetAddIntJump = 472

	// jtOffsetIncIntJumpLt is the jump table byte offset for the IncIntJumpLt super-instruction.
	jtOffsetIncIntJumpLt = 480

	// jtOffsetMathSqrt is the jump table byte offset for the MathSqrt handler.
	jtOffsetMathSqrt = 488

	// jtOffsetMathAbs is the jump table byte offset for the MathAbs handler.
	jtOffsetMathAbs = 496

	// jtOffsetMathFloor is the jump table byte offset for the MathFloor handler.
	jtOffsetMathFloor = 504

	// jtOffsetMathCeil is the jump table byte offset for the MathCeil handler.
	jtOffsetMathCeil = 512

	// jtOffsetMathTrunc is the jump table byte offset for the MathTrunc handler.
	jtOffsetMathTrunc = 520

	// jtOffsetLenString is the jump table byte offset for the LenString handler.
	jtOffsetLenString = 536

	// jtOffsetStringIndex is the jump table byte offset for the StringIndex handler.
	jtOffsetStringIndex = 544

	// jtOffsetEqString is the jump table byte offset for the EqString handler.
	jtOffsetEqString = 552

	// jtOffsetNeString is the jump table byte offset for the NeString handler.
	jtOffsetNeString = 560

	// jtOffsetSliceString is the jump table byte offset for the SliceString handler.
	jtOffsetSliceString = 568

	// jtOffsetStringIndexToInt is the jump table byte offset for the StringIndexToInt handler.
	jtOffsetStringIndexToInt = 576

	// jtOffsetLenStringLtJumpFalse is the jump table offset
	// for the LenStringLtJumpFalse handler.
	jtOffsetLenStringLtJumpFalse = 584
)

// amd64InitOps implements InitialisationOperationsPort for x86-64.
// Each method emits the complete handler body for an initialisation
// or exit operation.
type amd64InitOps struct{}

var _ asmgen.InitialisationOperationsPort = (*amd64InitOps)(nil)

// initJumpTableEntry emits a LEAQ/MOVQ pair that patches one entry
// in the jump table. The handler symbol is prefixed with the Plan 9
// middle-dot package separator.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes handler (string) which is the handler function symbol name.
// Takes offset (int) which is the byte offset into the jump table.
func initJumpTableEntry(e *asmgen.Emitter, handler string, offset int) {
	inst(e, "LEAQ", fmt.Sprintf("\xc2\xb7%s(SB), AX", handler))
	inst(e, mnemonicMOVQ, fmt.Sprintf("AX, %d(DI)", offset))
}

// EmitInitJumpTable emits the body of initJumpTable, which populates
// all 256 dispatch table entries. It first fills every slot with the
// tier2Fallback address, then patches the tier-1 opcode entries with
// their specific handler addresses.
//
// The patching is organised by opcode category for readability: data
// movement, integer arithmetic, bitwise operations, floating-point
// arithmetic, comparison operations, type conversion and logic, control
// flow, super-instructions, math built-ins, and string operations.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InitOps) EmitInitJumpTable(e *asmgen.Emitter) {
	emitJumpTableFillLoop(e)
	emitJumpTablePatchDataMovement(e)
	emitJumpTablePatchIntegerArithmetic(e)
	emitJumpTablePatchBitwise(e)
	emitJumpTablePatchFloatArithmetic(e)
	emitJumpTablePatchComparison(e)
	emitJumpTablePatchConversionAndLogic(e)
	emitJumpTablePatchControlFlow(e)
	emitJumpTablePatchSuperInstructions(e)
	emitJumpTablePatchMathBuiltins(e)
	emitJumpTablePatchStringOperations(e)

	inst(e, mnemonicRET, "")
}

// EmitInitJumpTableSSE41 emits the body of initJumpTableSSE41, which
// patches the three ROUNDSD-based handler entries (Floor, Ceil, Trunc)
// into the jump table. Only called when the CPU supports SSE4.1.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InitOps) EmitInitJumpTableSSE41(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "table+0(FP), DI")
	e.Blank()

	initJumpTableEntry(e, "handlerMathFloor", jtOffsetMathFloor)
	e.Blank()
	initJumpTableEntry(e, "handlerMathCeil", jtOffsetMathCeil)
	e.Blank()
	initJumpTableEntry(e, "handlerMathTrunc", jtOffsetMathTrunc)
	e.Blank()

	inst(e, mnemonicRET, "")
}

// EmitDispatchLoop emits the body of dispatchLoop, which loads the
// DispatchContext fields into pinned registers and performs the first
// dispatch via the DISPATCH_NEXT macro.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InitOps) EmitDispatchLoop(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "ctx+0(FP), R15")
	inst(e, mnemonicMOVQ, "0(R15), R12")
	inst(e, mnemonicMOVQ, "8(R15), R13")
	inst(e, mnemonicMOVQ, "16(R15), R14")
	inst(e, mnemonicMOVQ, "24(R15), R8")
	inst(e, mnemonicMOVQ, "40(R15), R9")
	inst(e, mnemonicMOVQ, "56(R15), R11")
	inst(e, mnemonicMOVQ, "88(R15), R10")
	e.Instruction("DISPATCH_NEXT()")
}

// EmitTier2Fallback emits the body of tier2Fallback, which
// un-advances pc and returns to Go with EXIT_TIER2.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func (*amd64InitOps) EmitTier2Fallback(e *asmgen.Emitter) {
	inst(e, "DECQ", "R14")
	inst(e, mnemonicMOVQ, "R14, 16(R15)")
	inst(e, mnemonicMOVQ, "$EXIT_TIER2, 96(R15)")
	inst(e, mnemonicMOVQ, "R14, 104(R15)")
	inst(e, mnemonicRET, "")
}

// EmitExitHandler emits an exit handler body that un-advances pc and
// returns to Go with the provided exit constant.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
// Takes exitConstant (string) which is the exit reason constant name.
func (*amd64InitOps) EmitExitHandler(e *asmgen.Emitter, exitConstant string) {
	inst(e, "DECQ", "R14")
	inst(e, mnemonicMOVQ, "R14, 16(R15)")
	inst(e, mnemonicMOVQ, "$"+exitConstant+", 96(R15)")
	inst(e, mnemonicMOVQ, "R14, 104(R15)")
	inst(e, mnemonicRET, "")
}

// emitJumpTableFillLoop emits the loop that fills all 256 jump table
// entries with the tier2Fallback handler address.
//
// This ensures that any opcode not explicitly patched with a tier-1
// handler will fall back to the interpreter tier. The loop uses a
// simple MOVQ/ADDQ/DECQ/JNZ pattern that iterates 256 times.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitJumpTableFillLoop(e *asmgen.Emitter) {
	inst(e, mnemonicMOVQ, "table+0(FP), DI")
	e.Blank()

	inst(e, "LEAQ", "\xc2\xb7tier2Fallback(SB), AX")
	inst(e, mnemonicMOVQ, "$256, CX")
	e.Blank()

	e.Label("initjt_fill")
	inst(e, mnemonicMOVQ, "AX, (DI)")
	inst(e, "ADDQ", "$8, DI")
	inst(e, "DECQ", "CX")
	inst(e, "JNZ", "initjt_fill")
	e.Blank()

	inst(e, mnemonicMOVQ, "table+0(FP), DI")
	e.Blank()
}

// emitJumpTablePatchDataMovement patches the jump table entries for
// data movement opcodes: Nop, MoveInt, MoveFloat, LoadIntConst,
// LoadFloatConst, LoadBool, and LoadIntConstSmall.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitJumpTablePatchDataMovement(e *asmgen.Emitter) {
	initJumpTableEntry(e, "handlerNop", 0)
	e.Blank()
	initJumpTableEntry(e, "handlerMoveInt", jtOffsetMoveInt)
	e.Blank()
	initJumpTableEntry(e, "handlerMoveFloat", jtOffsetMoveFloat)
	e.Blank()
	initJumpTableEntry(e, "handlerLoadIntConst", jtOffsetLoadIntConst)
	e.Blank()
	initJumpTableEntry(e, "handlerLoadFloatConst", jtOffsetLoadFloatConst)
	e.Blank()
	initJumpTableEntry(e, "handlerLoadBool", jtOffsetLoadBool)
	e.Blank()
	initJumpTableEntry(e, "handlerLoadIntConstSmall", jtOffsetLoadIntConstSmall)
	e.Blank()
}

// emitJumpTablePatchIntegerArithmetic patches the jump table entries
// for integer arithmetic opcodes: AddInt, SubInt, MulInt, DivInt,
// RemInt, NegInt, IncInt, and DecInt.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitJumpTablePatchIntegerArithmetic(e *asmgen.Emitter) {
	initJumpTableEntry(e, "handlerAddInt", jtOffsetAddInt)
	e.Blank()
	initJumpTableEntry(e, "handlerSubInt", jtOffsetSubInt)
	e.Blank()
	initJumpTableEntry(e, "handlerMulInt", jtOffsetMulInt)
	e.Blank()
	initJumpTableEntry(e, "handlerDivInt", jtOffsetDivInt)
	e.Blank()
	initJumpTableEntry(e, "handlerRemInt", jtOffsetRemInt)
	e.Blank()
	initJumpTableEntry(e, "handlerNegInt", jtOffsetNegInt)
	e.Blank()
	initJumpTableEntry(e, "handlerIncInt", jtOffsetIncInt)
	e.Blank()
	initJumpTableEntry(e, "handlerDecInt", jtOffsetDecInt)
	e.Blank()
}

// emitJumpTablePatchBitwise patches the jump table entries for bitwise
// opcodes: BitAnd, BitOr, BitXor, BitAndNot, BitNot, ShiftLeft, and
// ShiftRight.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitJumpTablePatchBitwise(e *asmgen.Emitter) {
	initJumpTableEntry(e, "handlerBitAnd", jtOffsetBitAnd)
	e.Blank()
	initJumpTableEntry(e, "handlerBitOr", jtOffsetBitOr)
	e.Blank()
	initJumpTableEntry(e, "handlerBitXor", jtOffsetBitXor)
	e.Blank()
	initJumpTableEntry(e, "handlerBitAndNot", jtOffsetBitAndNot)
	e.Blank()
	initJumpTableEntry(e, "handlerBitNot", jtOffsetBitNot)
	e.Blank()
	initJumpTableEntry(e, "handlerShiftLeft", jtOffsetShiftLeft)
	e.Blank()
	initJumpTableEntry(e, "handlerShiftRight", jtOffsetShiftRight)
	e.Blank()
}

// emitJumpTablePatchFloatArithmetic patches the jump table entries for
// floating-point arithmetic opcodes: AddFloat, SubFloat, MulFloat,
// DivFloat, and NegFloat.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitJumpTablePatchFloatArithmetic(e *asmgen.Emitter) {
	initJumpTableEntry(e, "handlerAddFloat", jtOffsetAddFloat)
	e.Blank()
	initJumpTableEntry(e, "handlerSubFloat", jtOffsetSubFloat)
	e.Blank()
	initJumpTableEntry(e, "handlerMulFloat", jtOffsetMulFloat)
	e.Blank()
	initJumpTableEntry(e, "handlerDivFloat", jtOffsetDivFloat)
	e.Blank()
	initJumpTableEntry(e, "handlerNegFloat", jtOffsetNegFloat)
	e.Blank()
}

// emitJumpTablePatchComparison patches the jump table entries for
// comparison opcodes: EqInt, NeInt, LtInt, LeInt, GtInt, GeInt,
// EqFloat, NeFloat, LtFloat, LeFloat, GtFloat, and GeFloat.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitJumpTablePatchComparison(e *asmgen.Emitter) {
	initJumpTableEntry(e, "handlerEqInt", jtOffsetEqInt)
	e.Blank()
	initJumpTableEntry(e, "handlerNeInt", jtOffsetNeInt)
	e.Blank()
	initJumpTableEntry(e, "handlerLtInt", jtOffsetLtInt)
	e.Blank()
	initJumpTableEntry(e, "handlerLeInt", jtOffsetLeInt)
	e.Blank()
	initJumpTableEntry(e, "handlerGtInt", jtOffsetGtInt)
	e.Blank()
	initJumpTableEntry(e, "handlerGeInt", jtOffsetGeInt)
	e.Blank()
	initJumpTableEntry(e, "handlerEqFloat", jtOffsetEqFloat)
	e.Blank()
	initJumpTableEntry(e, "handlerNeFloat", jtOffsetNeFloat)
	e.Blank()
	initJumpTableEntry(e, "handlerLtFloat", jtOffsetLtFloat)
	e.Blank()
	initJumpTableEntry(e, "handlerLeFloat", jtOffsetLeFloat)
	e.Blank()
	initJumpTableEntry(e, "handlerGtFloat", jtOffsetGtFloat)
	e.Blank()
	initJumpTableEntry(e, "handlerGeFloat", jtOffsetGeFloat)
	e.Blank()
}

// emitJumpTablePatchConversionAndLogic patches the jump table entries
// for type conversion and logic opcodes: IntToFloat, FloatToInt, and
// Not.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitJumpTablePatchConversionAndLogic(e *asmgen.Emitter) {
	initJumpTableEntry(e, "handlerIntToFloat", jtOffsetIntToFloat)
	e.Blank()
	initJumpTableEntry(e, "handlerFloatToInt", jtOffsetFloatToInt)
	e.Blank()
	initJumpTableEntry(e, "handlerNot", jtOffsetNot)
	e.Blank()
}

// emitJumpTablePatchControlFlow patches the jump table entries for
// control flow opcodes: Jump, JumpIfTrue, JumpIfFalse, CallInline,
// ReturnInline, ReturnVoidInline, and TailCallExit.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitJumpTablePatchControlFlow(e *asmgen.Emitter) {
	initJumpTableEntry(e, "handlerJump", jtOffsetJump)
	e.Blank()
	initJumpTableEntry(e, "handlerJumpIfTrue", jtOffsetJumpIfTrue)
	e.Blank()
	initJumpTableEntry(e, "handlerJumpIfFalse", jtOffsetJumpIfFalse)
	e.Blank()
	initJumpTableEntry(e, "handlerCallInline", jtOffsetCallInline)
	e.Blank()
	initJumpTableEntry(e, "handlerReturnInline", jtOffsetReturnInline)
	e.Blank()
	initJumpTableEntry(e, "handlerReturnVoidInline", jtOffsetReturnVoidInline)
	e.Blank()
	initJumpTableEntry(e, "handlerTailCallExit", jtOffsetTailCallExit)
	e.Blank()
}

// emitJumpTablePatchSuperInstructions patches the jump table entries
// for super-instructions (fused opcode pairs): SubIntConst, AddIntConst,
// LeIntConstJumpFalse, LtIntConstJumpFalse, EqIntConstJumpFalse,
// EqIntConstJumpTrue, GeIntConstJumpFalse, GtIntConstJumpFalse,
// MulIntConst, AddIntJump, and IncIntJumpLt.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitJumpTablePatchSuperInstructions(e *asmgen.Emitter) {
	initJumpTableEntry(e, "handlerSubIntConst", jtOffsetSubIntConst)
	e.Blank()
	initJumpTableEntry(e, "handlerAddIntConst", jtOffsetAddIntConst)
	e.Blank()
	initJumpTableEntry(e, "handlerLeIntConstJumpFalse", jtOffsetLeIntConstJumpFalse)
	e.Blank()
	initJumpTableEntry(e, "handlerLtIntConstJumpFalse", jtOffsetLtIntConstJumpFalse)
	e.Blank()
	initJumpTableEntry(e, "handlerEqIntConstJumpFalse", jtOffsetEqIntConstJumpFalse)
	e.Blank()
	initJumpTableEntry(e, "handlerEqIntConstJumpTrue", jtOffsetEqIntConstJumpTrue)
	e.Blank()
	initJumpTableEntry(e, "handlerGeIntConstJumpFalse", jtOffsetGeIntConstJumpFalse)
	e.Blank()
	initJumpTableEntry(e, "handlerGtIntConstJumpFalse", jtOffsetGtIntConstJumpFalse)
	e.Blank()
	initJumpTableEntry(e, "handlerMulIntConst", jtOffsetMulIntConst)
	e.Blank()
	initJumpTableEntry(e, "handlerAddIntJump", jtOffsetAddIntJump)
	e.Blank()
	initJumpTableEntry(e, "handlerIncIntJumpLt", jtOffsetIncIntJumpLt)
	e.Blank()
}

// emitJumpTablePatchMathBuiltins patches the jump table entries for
// math built-in opcodes: MathSqrt and MathAbs.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
//
// Note: MathFloor, MathCeil, and MathTrunc require SSE4.1 on amd64
// and are patched separately by EmitInitJumpTableSSE41.
func emitJumpTablePatchMathBuiltins(e *asmgen.Emitter) {
	initJumpTableEntry(e, "handlerMathSqrt", jtOffsetMathSqrt)
	e.Blank()
	initJumpTableEntry(e, "handlerMathAbs", jtOffsetMathAbs)
	e.Blank()
}

// emitJumpTablePatchStringOperations patches the jump table entries for
// string opcodes: LenString, StringIndex, EqString, NeString,
// SliceString, StringIndexToInt, and LenStringLtJumpFalse.
//
// Takes e (*asmgen.Emitter) which is the assembly emitter to write to.
func emitJumpTablePatchStringOperations(e *asmgen.Emitter) {
	initJumpTableEntry(e, "handlerLenString", jtOffsetLenString)
	e.Blank()
	initJumpTableEntry(e, "handlerStringIndex", jtOffsetStringIndex)
	e.Blank()
	initJumpTableEntry(e, "handlerEqString", jtOffsetEqString)
	e.Blank()
	initJumpTableEntry(e, "handlerNeString", jtOffsetNeString)
	e.Blank()
	initJumpTableEntry(e, "handlerSliceString", jtOffsetSliceString)
	e.Blank()
	initJumpTableEntry(e, "handlerStringIndexToInt", jtOffsetStringIndexToInt)
	e.Blank()
	initJumpTableEntry(e, "handlerLenStringLtJumpFalse", jtOffsetLenStringLtJumpFalse)
	e.Blank()
}
