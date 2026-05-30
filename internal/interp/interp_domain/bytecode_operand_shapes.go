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

// operandRole classifies what an instruction operand byte means.
//
// The classification feeds two distinct consumers: the compile-time emit funnel inserts
// coercions for register-bank reads when the static kind disagrees with the contract, and
// the post-compilation verifier asserts that every typed-bank read has been preceded by a
// same-kind write on every dataflow path.
//
// Only register reads and writes carry a bank constraint; other roles (constants,
// immediates, jump offsets, type indices) are opaque to the funnel and verifier and exist
// solely so the table is exhaustive.
type operandRole uint8

const (
	// roleNone marks an unused operand byte.
	roleNone operandRole = iota

	// roleRegInt designates the operand as an int-bank register index.
	roleRegInt

	// roleRegFloat designates the operand as a float-bank register index.
	roleRegFloat

	// roleRegString designates the operand as a string-bank register index.
	roleRegString

	// roleRegBool designates the operand as a bool-bank register index.
	roleRegBool

	// roleRegUint designates the operand as a uint-bank register index.
	roleRegUint

	// roleRegComplex designates the operand as a complex-bank register index.
	roleRegComplex

	// roleRegGeneral designates the operand as a general-bank register index.
	roleRegGeneral

	// roleRegSliceInt designates the operand as a slicesInt-bank register index, holding a
	// typed []int64 slice header. Used by the typed slice opcodes (opMakeSliceInt,
	// opSliceGetIntDirect, opSliceSetIntDirect, opLenSliceIntDirect, opRangeNextSliceInt)
	// that bypass reflect.Value entirely.
	roleRegSliceInt

	// roleRegSliceFloat designates the operand as a slicesFloat-bank register index, holding
	// a typed []float64 slice header.
	roleRegSliceFloat

	// roleRegSliceString designates the operand as a slicesString-bank register index,
	// holding a typed []string slice header.
	roleRegSliceString

	// roleRegSliceBool designates the operand as a slicesBool-bank register index, holding a
	// typed []bool slice header.
	roleRegSliceBool

	// roleRegSliceUint designates the operand as a slicesUint-bank register index, holding a
	// typed []uint64 slice header.
	roleRegSliceUint

	// roleRegDynamic marks a register operand whose bank is selected at runtime by another
	// operand or a following extension word. Used by ops such as opGetUpvalue, opLoadZero
	// and opAddr; the funnel treats these as opaque and the verifier widens them to "any
	// bank".
	roleRegDynamic

	// roleConstIndex marks an operand that indexes into a constant pool.
	roleConstIndex

	// roleFieldIndex marks an operand that selects a struct field by position.
	roleFieldIndex

	// roleImmediate marks a small inline value (e.g. byte literals, flag bits).
	roleImmediate

	// roleTypeIndex marks an operand that indexes into the function's type table.
	roleTypeIndex

	// roleKindMarker marks an operand that names a registerKind value (used by ops like
	// opPackInterface and opUnpackInterface where C names the bank for B or A).
	roleKindMarker

	// roleSubOpcode marks an operand that names a subOpcode value dispatched by
	// opDrillTier1. The verifier treats this as an immediate-style classifier rather than a
	// register reference.
	roleSubOpcode

	// roleJumpOffsetLow marks the low byte of a 16-bit signed jump offset.
	roleJumpOffsetLow

	// roleJumpOffsetHigh marks the high byte of a 16-bit signed jump offset paired with
	// roleJumpOffsetLow.
	roleJumpOffsetHigh

	// roleCallSiteLow marks the low byte of a call-site index.
	roleCallSiteLow

	// roleCallSiteHigh marks the high byte of a call-site index paired with roleCallSiteLow.
	roleCallSiteHigh

	// roleFollowsExtension marks an operand whose meaning is supplied by the following opExt
	// word; the verifier treats both bytes as opaque.
	roleFollowsExtension

	// roleUnknown marks an opcode the table does not describe; treated as opaque (no
	// constraints, no writes recorded) so undescribed opcodes do not block verification of
	// the rest of the function.
	roleUnknown
)

// shapeFlag bitmasks describe whole-instruction properties that individual operand roles
// cannot express.
type shapeFlag uint8

const (
	// shapeFlagDescribed marks the descriptor as authoritative. Without it, the verifier
	// treats the opcode as opaque (matching roleUnknown per-operand entries).
	shapeFlagDescribed shapeFlag = 1 << iota

	// shapeFlagFollowsExtension marks an opcode that consumes one or more opExt words after
	// it. The verifier skips those words rather than treating them as standalone
	// instructions.
	shapeFlagFollowsExtension

	// shapeFlagControlFlow marks ops that change the program counter in non-fallthrough ways
	// (jumps, calls, returns, panics). The verifier uses this to build the CFG.
	shapeFlagControlFlow

	// shapeFlagTerminator marks ops that end a basic block by always transferring control
	// elsewhere (return, panic, jump).
	shapeFlagTerminator
)

const (
	// numInstructionOperands is the fixed number of single-byte operands (a, b, c) carried
	// by every instruction in the four-byte instruction encoding. Tables that span operand
	// positions have this length.
	numInstructionOperands = 3
)

// operandShape describes the per-operand role for one opcode.
//
// The reads array is true for operands that read a register; writes is true for operands
// that write a register. The followsExtension flag (in flags) indicates that this opcode
// consumes the following opExt word; the verifier skips that word rather than decoding
// it.
type operandShape struct {
	// a is the role for operand byte A.
	a operandRole

	// b is the role for operand byte B.
	b operandRole

	// c is the role for operand byte C.
	c operandRole

	// reads marks each operand that reads its named register.
	reads [numInstructionOperands]bool

	// writes marks each operand that writes its named register.
	writes [numInstructionOperands]bool

	// flags holds whole-instruction shape flags (control flow, extension, terminator, etc.).
	flags shapeFlag
}

var (
	// operandShapes maps each opcode to its operand-shape descriptor. Entries without
	// shapeFlagDescribed are treated as opaque by the verifier and emit funnel; this lets
	// coverage grow incrementally without blocking rollout.
	operandShapes [opcodeCount]operandShape
)

// kindForRole returns the registerKind expected by a register-shaped operand role. The
// boolean is false for non-register roles.
//
// Takes role (operandRole) which is the operand's classification.
//
// Returns the corresponding register bank kind and true when the role is register-shaped,
// otherwise zero and false.
func kindForRole(role operandRole) (registerKind, bool) {
	switch role {
	case roleRegInt:
		return registerInt, true
	case roleRegFloat:
		return registerFloat, true
	case roleRegString:
		return registerString, true
	case roleRegBool:
		return registerBool, true
	case roleRegUint:
		return registerUint, true
	case roleRegComplex:
		return registerComplex, true
	case roleRegGeneral:
		return registerGeneral, true
	case roleRegSliceInt:
		return registerSliceInt, true
	case roleRegSliceFloat:
		return registerSliceFloat, true
	case roleRegSliceString:
		return registerSliceString, true
	case roleRegSliceBool:
		return registerSliceBool, true
	case roleRegSliceUint:
		return registerSliceUint, true
	default:
	}
	return 0, false
}

// roleForKind returns the operandRole that names a register operand of the given bank.
//
// Takes kind (registerKind) which is the bank to convert.
//
// Returns the matching operandRole.
func roleForKind(kind registerKind) operandRole {
	switch kind {
	case registerInt:
		return roleRegInt
	case registerFloat:
		return roleRegFloat
	case registerString:
		return roleRegString
	case registerBool:
		return roleRegBool
	case registerUint:
		return roleRegUint
	case registerComplex:
		return roleRegComplex
	case registerSliceInt:
		return roleRegSliceInt
	case registerSliceFloat:
		return roleRegSliceFloat
	case registerSliceString:
		return roleRegSliceString
	case registerSliceBool:
		return roleRegSliceBool
	case registerSliceUint:
		return roleRegSliceUint
	default:
		return roleRegGeneral
	}
}

// described3 records a three-operand opcode with separate roles per position.
// shapeFlagDescribed is added automatically.
//
// Takes op (opcode) which is the opcode being described.
// Takes ra, rb, rc (operandRole) which are the per-operand roles.
// Takes reads, writes (numInstructionOperandsbool) which mark register accesses per
// position.
// Takes flags (shapeFlag) which adds extra whole-instruction flags.
func described3(op opcode, ra, rb, rc operandRole, reads, writes [numInstructionOperands]bool, flags shapeFlag) {
	operandShapes[op] = operandShape{
		a: ra, b: rb, c: rc,
		reads:  reads,
		writes: writes,
		flags:  flags | shapeFlagDescribed,
	}
}

// describeDstSrcSrc records the canonical "writes A, reads B and C" pattern where all
// three operands share the same register kind.
//
// Takes op (opcode) which is the opcode being described.
// Takes kind (operandRole) which is the shared register role.
func describeDstSrcSrc(op opcode, kind operandRole) {
	described3(op, kind, kind, kind,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
}

// describeDstSrcSrcMixed records "writes A in destinationKind, reads B and C in
// sourceKind" (e.g. comparisons that return int but read float).
//
// Takes op (opcode) which is the opcode being described.
// Takes destination (operandRole) which is the destination role.
// Takes source (operandRole) which is the shared source role.
func describeDstSrcSrcMixed(op opcode, destination, source operandRole) {
	described3(op, destination, source, source,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
}

// describeDstSrc records "writes A, reads B" with C unused.
//
// Takes op (opcode) which is the opcode being described.
// Takes destination (operandRole) which is the destination role.
// Takes source (operandRole) which is the source role.
func describeDstSrc(op opcode, destination, source operandRole) {
	described3(op, destination, source, roleNone,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
}

// describeLoadConst records "writes A from a wide constant index in B|(C<<8)".
//
// Takes op (opcode) which is the opcode being described.
// Takes destination (operandRole) which is the destination register role.
func describeLoadConst(op opcode, destination operandRole) {
	described3(op, destination, roleConstIndex, roleConstIndex,
		[numInstructionOperands]bool{false, false, false},
		[numInstructionOperands]bool{true, false, false},
		0)
}

// describeArithConst records "writes A, reads B, takes constant index in C".
//
// Takes op (opcode) which is the opcode being described.
// Takes kind (operandRole) which is the register role for both A and B.
func describeArithConst(op opcode, kind operandRole) {
	described3(op, kind, kind, roleConstIndex,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
}

// describeJumpCondition records "reads A, jump offset in B|(C<<8)".
//
// Takes op (opcode) which is the opcode being described.
// Takes source (operandRole) which is the role for the test register.
func describeJumpCondition(op opcode, source operandRole) {
	described3(op, source, roleJumpOffsetLow, roleJumpOffsetHigh,
		[numInstructionOperands]bool{true, false, false},
		[numInstructionOperands]bool{false, false, false},
		shapeFlagControlFlow)
}

// describeConstJump records "reads A, B is constant index, opExt follows for jump
// offset". Used by fused conditional-and-jump opcodes.
//
// Takes op (opcode) which is the opcode being described.
// Takes source (operandRole) which is the role for the test register.
func describeConstJump(op opcode, source operandRole) {
	described3(op, source, roleConstIndex, roleNone,
		[numInstructionOperands]bool{true, false, false},
		[numInstructionOperands]bool{false, false, false},
		shapeFlagControlFlow|shapeFlagFollowsExtension)
}

// describeRegRegJump records "reads A and B as registers of role source, opExt follows
// for jump offset". Used by fused reg-reg-compare-and-jump opcodes such as
// opLtIntJumpFalse.
//
// Takes op (opcode) which is the opcode being described.
// Takes source (operandRole) which is the role for both operand registers.
func describeRegRegJump(op opcode, source operandRole) {
	described3(op, source, source, roleNone,
		[numInstructionOperands]bool{true, true, false},
		[numInstructionOperands]bool{false, false, false},
		shapeFlagControlFlow|shapeFlagFollowsExtension)
}

// describeTypedSliceGet records "writes A as destinationKind, reads B as general (the
// slice), reads C as int (the index)".
//
// Takes op (opcode) which is the opcode being described.
// Takes destination (operandRole) which is the destination register role.
func describeTypedSliceGet(op opcode, destination operandRole) {
	described3(op, destination, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
}

// describeTypedSliceSet records "reads A as general (the slice), reads B as int (index),
// reads C as srcKind (the value)".
//
// Takes op (opcode) which is the opcode being described.
// Takes source (operandRole) which is the value register role.
func describeTypedSliceSet(op opcode, source operandRole) {
	described3(op, roleRegGeneral, roleRegInt, source,
		[numInstructionOperands]bool{true, true, true},
		[numInstructionOperands]bool{false, false, false},
		0)
}

func init() {
	populateScalarArithShapes()
	populateGenericArithShapes()
	populateComparisonShapes()
	populateConversionMoveShapes()
	populateConstantLoadShapes()
	populateJumpAndFusedShapes()
	populateMathShapes()
	populateCollectionShapes()
	populateAddressTypeShapes()
	populateGlobalUnsafeShapes()
	populateStringShapes()
	populateStringIntrinsicShapes()
	populateComplexAndFusedStringShapes()
	populateSliceAndAppendShapes()
	populateTypedSliceDirectShapes()
	populateMapAndFieldShapes()
	populateUpvalueAndConvertShapes()
	populateTerminatorShapes()
}

// populateScalarArithShapes describes integer, float, uint, and complex arithmetic and
// bitwise opcodes that operate on a single register bank.
func populateScalarArithShapes() {
	describeDstSrcSrc(opAddInt, roleRegInt)
	describeDstSrcSrc(opSubInt, roleRegInt)
	describeDstSrcSrc(opMulInt, roleRegInt)
	describeDstSrcSrc(opDivInt, roleRegInt)
	describeDstSrcSrc(opRemInt, roleRegInt)
	describeDstSrcSrc(opBitAnd, roleRegInt)
	describeDstSrcSrc(opBitOr, roleRegInt)
	describeDstSrcSrc(opBitXor, roleRegInt)
	describeDstSrcSrc(opBitAndNot, roleRegInt)
	describeDstSrcSrc(opShiftLeft, roleRegInt)
	describeDstSrcSrc(opShiftRight, roleRegInt)
	describeDstSrcSrc(opAddFloat, roleRegFloat)
	describeDstSrcSrc(opSubFloat, roleRegFloat)
	describeDstSrcSrc(opMulFloat, roleRegFloat)
	describeDstSrcSrc(opDivFloat, roleRegFloat)
	describeDstSrcSrc(opAddUint, roleRegUint)
	describeDstSrcSrc(opSubUint, roleRegUint)
	describeDstSrcSrc(opMulUint, roleRegUint)
	describeDstSrcSrc(opDivUint, roleRegUint)
	describeDstSrcSrc(opRemUint, roleRegUint)
	describeDstSrcSrc(opBitAndUint, roleRegUint)
	describeDstSrcSrc(opBitOrUint, roleRegUint)
	describeDstSrcSrc(opBitXorUint, roleRegUint)
	describeDstSrcSrc(opBitAndNotUint, roleRegUint)
	describeDstSrcSrc(opShiftLeftUint, roleRegUint)
	describeDstSrcSrc(opShiftRightUint, roleRegUint)
	describeDstSrcSrc(opAddComplex, roleRegComplex)
	describeDstSrcSrc(opSubComplex, roleRegComplex)
	describeDstSrcSrc(opMulComplex, roleRegComplex)
	describeDstSrcSrc(opDivComplex, roleRegComplex)
}

// populateGenericArithShapes describes the reflect-based arithmetic opcodes that operate
// on the general register bank.
func populateGenericArithShapes() {
	describeDstSrcSrc(opAdd, roleRegGeneral)
	describeDstSrcSrc(opSub, roleRegGeneral)
	describeDstSrcSrc(opMul, roleRegGeneral)
	describeDstSrcSrc(opDiv, roleRegGeneral)
	describeDstSrcSrc(opRem, roleRegGeneral)
}

// populateComparisonShapes describes the comparison opcodes that produce an int
// (boolean-as-int) result from typed-bank reads.
func populateComparisonShapes() {
	describeDstSrcSrcMixed(opEqInt, roleRegInt, roleRegInt)
	describeDstSrcSrcMixed(opNeInt, roleRegInt, roleRegInt)
	describeDstSrcSrcMixed(opLtInt, roleRegInt, roleRegInt)
	describeDstSrcSrcMixed(opLeInt, roleRegInt, roleRegInt)
	describeDstSrcSrcMixed(opGtInt, roleRegInt, roleRegInt)
	describeDstSrcSrcMixed(opGeInt, roleRegInt, roleRegInt)
	describeDstSrcSrcMixed(opEqFloat, roleRegInt, roleRegFloat)
	describeDstSrcSrcMixed(opNeFloat, roleRegInt, roleRegFloat)
	describeDstSrcSrcMixed(opLtFloat, roleRegInt, roleRegFloat)
	describeDstSrcSrcMixed(opLeFloat, roleRegInt, roleRegFloat)
	describeDstSrcSrcMixed(opGtFloat, roleRegInt, roleRegFloat)
	describeDstSrcSrcMixed(opGeFloat, roleRegInt, roleRegFloat)
	describeDstSrcSrcMixed(opEqString, roleRegInt, roleRegString)
	describeDstSrcSrcMixed(opNeString, roleRegInt, roleRegString)
	describeDstSrcSrcMixed(opLtString, roleRegInt, roleRegString)
	describeDstSrcSrcMixed(opLeString, roleRegInt, roleRegString)
	describeDstSrcSrcMixed(opGtString, roleRegInt, roleRegString)
	describeDstSrcSrcMixed(opGeString, roleRegInt, roleRegString)
	describeDstSrcSrcMixed(opEqUint, roleRegInt, roleRegUint)
	describeDstSrcSrcMixed(opNeUint, roleRegInt, roleRegUint)
	describeDstSrcSrcMixed(opLtUint, roleRegInt, roleRegUint)
	describeDstSrcSrcMixed(opLeUint, roleRegInt, roleRegUint)
	describeDstSrcSrcMixed(opGtUint, roleRegInt, roleRegUint)
	describeDstSrcSrcMixed(opGeUint, roleRegInt, roleRegUint)
	describeDstSrcSrcMixed(opEqComplex, roleRegInt, roleRegComplex)
	describeDstSrcSrcMixed(opNeComplex, roleRegInt, roleRegComplex)
	describeDstSrcSrcMixed(opEqGeneral, roleRegInt, roleRegGeneral)
	describeDstSrcSrcMixed(opNeGeneral, roleRegInt, roleRegGeneral)
	describeDstSrc(opEqInterfaceNil, roleRegInt, roleRegGeneral)
	describeDstSrc(opNeInterfaceNil, roleRegInt, roleRegGeneral)
	describeDstSrcSrcMixed(opLtGeneral, roleRegInt, roleRegGeneral)
	describeDstSrcSrcMixed(opLeGeneral, roleRegInt, roleRegGeneral)
	describeDstSrcSrcMixed(opGtGeneral, roleRegInt, roleRegGeneral)
	describeDstSrcSrcMixed(opGeGeneral, roleRegInt, roleRegGeneral)
}

// populateConversionMoveShapes describes type-converting moves between scalar register
// banks plus same-bank moves and the cross-bank pack/unpack opcodes.
func populateConversionMoveShapes() {
	described3(opTruncateNarrow, roleRegDynamic, roleImmediate, roleKindMarker,
		[numInstructionOperands]bool{true, false, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	describeDstSrc(opMoveGeneral, roleRegGeneral, roleRegGeneral)
	described3(opPackInterface, roleRegGeneral, roleRegDynamic, roleKindMarker,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opPackTyped, roleRegGeneral, roleRegDynamic, roleKindMarker,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opUnpackInterface, roleRegDynamic, roleRegGeneral, roleKindMarker,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
}

// populateConstantLoadShapes describes the opcodes that materialise constants into typed
// register banks.
func populateConstantLoadShapes() {
	describeLoadConst(opLoadIntConst, roleRegInt)
	describeLoadConst(opLoadFloatConst, roleRegFloat)
	describeLoadConst(opLoadStringConst, roleRegString)
	describeLoadConst(opLoadGeneralConst, roleRegGeneral)
	describeLoadConst(opLoadUintConst, roleRegUint)
	describeLoadConst(opLoadComplexConst, roleRegComplex)
	described3(opLoadBoolConst, roleRegBool, roleConstIndex, roleNone,
		[numInstructionOperands]bool{false, false, false},
		[numInstructionOperands]bool{true, false, false},
		0)
}

// populateJumpAndFusedShapes describes unconditional jumps, conditional branches, and
// fused arithmetic-and-jump opcodes.
func populateJumpAndFusedShapes() {
	describeArithConst(opSubIntConst, roleRegInt)
	describeArithConst(opAddIntConst, roleRegInt)
	describeArithConst(opMulIntConst, roleRegInt)
	describeConstJump(opLeIntConstJumpFalse, roleRegInt)
	describeConstJump(opLtIntConstJumpFalse, roleRegInt)
	describeConstJump(opEqIntConstJumpFalse, roleRegInt)
	describeConstJump(opEqIntConstJumpTrue, roleRegInt)
	describeConstJump(opGeIntConstJumpFalse, roleRegInt)
	describeConstJump(opGtIntConstJumpFalse, roleRegInt)
	describeConstJump(opEqStringConstJumpFalse, roleRegString)
	describeRegRegJump(opLtIntJumpFalse, roleRegInt)
	describeRegRegJump(opLeIntJumpFalse, roleRegInt)
	describeRegRegJump(opGtIntJumpFalse, roleRegInt)
	describeRegRegJump(opGeIntJumpFalse, roleRegInt)
	describeRegRegJump(opEqIntJumpFalse, roleRegInt)
	describeRegRegJump(opNeIntJumpFalse, roleRegInt)
	described3(opAddIntJump, roleRegInt, roleRegInt, roleConstIndex,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension|shapeFlagControlFlow)
	describeJumpCondition(opJumpIfTrue, roleRegInt)
	describeJumpCondition(opJumpIfFalse, roleRegInt)
	describeJumpCondition(opTestNilJumpTrue, roleRegGeneral)
	describeJumpCondition(opTestNilJumpFalse, roleRegGeneral)
}

// populateMathShapes describes the math intrinsic opcodes that operate on float
// registers, plus the umbrella opcode that dispatches to many cold-path handlers via a
// sub-op discriminator.
func populateMathShapes() {
	describeDstSrcSrc(opMathPow, roleRegFloat)

	described3(opDrillTier1, roleSubOpcode, roleRegDynamic, roleRegDynamic,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{false, true, false},
		0)
}

// populateStringShapes describes basic string operations and strconv formatters that
// produce string-bank values from typed-bank inputs.
func populateStringShapes() {
	described3(opStringIndex, roleRegUint, roleRegString, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opStringIndexToInt, roleRegInt, roleRegString, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opStringIndexUnchecked, roleRegUint, roleRegString, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opStringIndexToIntUnchecked, roleRegInt, roleRegString, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	describeDstSrcSrc(opConcatString, roleRegString)
	described3(opConcatRuneString, roleRegString, roleRegString, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
}

// populateStringIntrinsicShapes describes the strings package intrinsic opcodes
// (Contains, ToUpper, etc.).
func populateStringIntrinsicShapes() {
	described3(opStrContainsRune, roleRegBool, roleRegString, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	describeDstSrcSrcMixed(opStrContains, roleRegBool, roleRegString)
	describeDstSrcSrcMixed(opStrHasPrefix, roleRegBool, roleRegString)
	describeDstSrcSrcMixed(opStrHasSuffix, roleRegBool, roleRegString)
	describeDstSrcSrcMixed(opStrEqualFold, roleRegBool, roleRegString)
	describeDstSrcSrcMixed(opStrIndex, roleRegInt, roleRegString)
	describeDstSrcSrcMixed(opStrCount, roleRegInt, roleRegString)
	describeDstSrcSrc(opStrTrimPrefix, roleRegString)
	describeDstSrcSrc(opStrTrimSuffix, roleRegString)
	describeDstSrcSrc(opStrTrim, roleRegString)
	described3(opStrIndexRune, roleRegInt, roleRegString, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opStrRepeat, roleRegString, roleRegString, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	describeDstSrcSrcMixed(opStrLastIndex, roleRegInt, roleRegString)
	described3(opStrJoin, roleRegString, roleRegGeneral, roleRegString,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opStrSplit, roleRegGeneral, roleRegString, roleRegString,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opStrReplaceAll, roleRegString, roleRegString, roleRegString,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
}

// populateComplexAndFusedStringShapes describes complex constructors, accessors, and
// fused string-and-jump opcodes.
func populateComplexAndFusedStringShapes() {
	describeDstSrcSrcMixed(opBuildComplex, roleRegComplex, roleRegFloat)
	described3(opSliceString, roleRegString, roleRegString, roleImmediate,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
}

// populateSliceAndAppendShapes describes typed slice get/set and typed append opcodes.
func populateSliceAndAppendShapes() {
	describeTypedSliceGet(opSliceGetInt, roleRegInt)
	describeTypedSliceGet(opSliceGetFloat, roleRegFloat)
	describeTypedSliceGet(opSliceGetString, roleRegString)
	describeTypedSliceGet(opSliceGetBool, roleRegBool)
	describeTypedSliceGet(opSliceGetUint, roleRegUint)
	describeTypedSliceGet(opSliceGetIntUnchecked, roleRegInt)
	describeTypedSliceSet(opSliceSetInt, roleRegInt)
	describeTypedSliceSet(opSliceSetFloat, roleRegFloat)
	describeTypedSliceSet(opSliceSetString, roleRegString)
	describeTypedSliceSet(opSliceSetBool, roleRegBool)
	describeTypedSliceSet(opSliceSetUint, roleRegUint)
	describeTypedSliceSet(opSliceSetIntUnchecked, roleRegInt)
}

// populateTypedSliceDirectShapes describes typed direct-storage slice opcodes that
// operate against the slicesInt bank without reflect.
func populateTypedSliceDirectShapes() {
	described3(opSliceGetIntDirect, roleRegInt, roleRegSliceInt, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opSliceSetIntDirect, roleRegSliceInt, roleRegInt, roleRegInt,
		[numInstructionOperands]bool{true, true, true},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opRangeNextSliceInt, roleRegInt, roleRegSliceInt, roleRegInt,
		[numInstructionOperands]bool{true, true, false},
		[numInstructionOperands]bool{true, false, true},
		shapeFlagFollowsExtension|shapeFlagControlFlow)
	described3(opSliceGetIntDirectUnchecked, roleRegInt, roleRegSliceInt, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opSliceSetIntDirectUnchecked, roleRegSliceInt, roleRegInt, roleRegInt,
		[numInstructionOperands]bool{true, true, true},
		[numInstructionOperands]bool{false, false, false},
		0)
}

// populateMapAndFieldShapes describes typed map opcodes and typed field accesses.
func populateMapAndFieldShapes() {
	populateTypedMapShapes()
	populateGenericFieldShapes()
	populateStructFieldShapes()
	populateSliceIndexStructFieldShapes()
}

// populateTypedMapShapes describes opcodes that access maps keyed and valued by typed
// registers (int, string).
func populateTypedMapShapes() {
	described3(opMapGetIntInt, roleRegInt, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opMapSetIntInt, roleRegGeneral, roleRegInt, roleRegInt,
		[numInstructionOperands]bool{true, true, true},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opMapGetStringInt, roleRegInt, roleRegGeneral, roleRegString,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opMapSetStringInt, roleRegGeneral, roleRegString, roleRegInt,
		[numInstructionOperands]bool{true, true, true},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opMapAddIntInt, roleRegGeneral, roleRegInt, roleRegInt,
		[numInstructionOperands]bool{true, true, true},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opMapAddStringInt, roleRegGeneral, roleRegString, roleRegInt,
		[numInstructionOperands]bool{true, true, true},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opMapGetStringString, roleRegString, roleRegGeneral, roleRegString,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opMapSetStringString, roleRegGeneral, roleRegString, roleRegString,
		[numInstructionOperands]bool{true, true, true},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opMapGetIntString, roleRegString, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opMapSetIntString, roleRegGeneral, roleRegInt, roleRegString,
		[numInstructionOperands]bool{true, true, true},
		[numInstructionOperands]bool{false, false, false},
		0)
}

// populateGenericFieldShapes describes the generic typed-int field accessors and the
// dynamic get/set field opcodes.
//
// opGetField writes to general[A] unconditionally (see handleGetField); opSetField reads
// from general[C] unconditionally (see handleSetField). Annotated roleRegGeneral so the
// inliner's remapOperands rewrites them when splicing a callee that uses these ops;
// roleRegDynamic would silently skip the remap and leave the byte at its callee slot
// value.
func populateGenericFieldShapes() {
	described3(opGetFieldInt, roleRegInt, roleRegGeneral, roleFieldIndex,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opSetFieldInt, roleRegGeneral, roleFieldIndex, roleRegInt,
		[numInstructionOperands]bool{true, false, true},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opGetField, roleRegGeneral, roleRegGeneral, roleFieldIndex,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opSetField, roleRegGeneral, roleFieldIndex, roleRegGeneral,
		[numInstructionOperands]bool{true, false, true},
		[numInstructionOperands]bool{false, false, false},
		shapeFlagFollowsExtension)
}

// populateStructFieldShapes describes the typed tier-0 struct-field get and set opcodes
// plus the in-place swap.
//
// opSwapStructFieldsGeneralT0 reads the struct in A and mutates two in-place fields; from
// the bytecode's register-allocator POV the struct general register is both read and
// written (the slot's value reflects a swapped state afterwards but the register itself
// keeps holding the same *Struct).
func populateStructFieldShapes() {
	described3(opGetStructFieldIntT0, roleRegInt, roleRegGeneral, roleFieldIndex,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opSetStructFieldIntT0, roleRegGeneral, roleRegInt, roleFieldIndex,
		[numInstructionOperands]bool{true, true, false},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opGetStructFieldUint, roleRegUint, roleRegGeneral, roleFieldIndex,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opSetStructFieldUint, roleRegGeneral, roleRegUint, roleFieldIndex,
		[numInstructionOperands]bool{true, true, false},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opGetStructFieldFloat, roleRegFloat, roleRegGeneral, roleFieldIndex,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opSetStructFieldFloat, roleRegGeneral, roleRegFloat, roleFieldIndex,
		[numInstructionOperands]bool{true, true, false},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opGetStructFieldBool, roleRegBool, roleRegGeneral, roleFieldIndex,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opSetStructFieldBool, roleRegGeneral, roleRegBool, roleFieldIndex,
		[numInstructionOperands]bool{true, true, false},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opGetStructFieldGeneral, roleRegGeneral, roleRegGeneral, roleFieldIndex,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opGetStructFieldRawPointerT0, roleRegGeneral, roleRegGeneral, roleFieldIndex,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opSetStructFieldGeneral, roleRegGeneral, roleRegGeneral, roleFieldIndex,
		[numInstructionOperands]bool{true, true, false},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opSwapStructFieldsGeneralT0, roleRegGeneral, roleFieldIndex, roleFieldIndex,
		[numInstructionOperands]bool{true, false, false},
		[numInstructionOperands]bool{false, false, false},
		0)
}

// populateSliceIndexStructFieldShapes describes the typed fused
// slice-index-of-struct-field super-instructions for every register family.
func populateSliceIndexStructFieldShapes() {
	described3(opSliceIndexStructFieldInt, roleRegInt, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opSliceIndexStructFieldUint, roleRegUint, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opSliceIndexStructFieldFloat, roleRegFloat, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opSliceIndexStructFieldBool, roleRegBool, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opSliceIndexStructFieldString, roleRegString, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
}

// populateUpvalueAndConvertShapes describes upvalue access and the remaining conversion
// opcodes that touch the general bank.
func populateUpvalueAndConvertShapes() {
	described3(opGetUpvalue, roleRegDynamic, roleImmediate, roleKindMarker,
		[numInstructionOperands]bool{false, false, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opSetUpvalue, roleRegDynamic, roleImmediate, roleKindMarker,
		[numInstructionOperands]bool{true, false, false},
		[numInstructionOperands]bool{false, false, false},
		0)
}

// populateTerminatorShapes describes returns, panics, and the trivial nop and ext
// opcodes.
func populateTerminatorShapes() {
	described3(opExt, roleImmediate, roleImmediate, roleImmediate,
		[numInstructionOperands]bool{false, false, false},
		[numInstructionOperands]bool{false, false, false},
		0)
}

// populateCollectionShapes describes the make/index/append/len/cap opcodes that operate
// on slice/map/channel general-bank values.
func populateCollectionShapes() {
	described3(opMakeSlice, roleRegGeneral, roleRegInt, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opIndex, roleRegGeneral, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opIndexSet, roleRegGeneral, roleRegInt, roleRegGeneral,
		[numInstructionOperands]bool{true, true, true},
		[numInstructionOperands]bool{false, false, false},
		shapeFlagFollowsExtension)
	described3(opSliceOp, roleRegGeneral, roleRegGeneral, roleImmediate,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMapIndex, roleRegGeneral, roleRegGeneral, roleRegGeneral,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opMapSet, roleRegGeneral, roleRegGeneral, roleRegGeneral,
		[numInstructionOperands]bool{true, true, true},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opMapIndexOk, roleRegGeneral, roleRegGeneral, roleRegGeneral,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	populateTypedMapIndexOkShapes()
	described3(opAppend, roleRegGeneral, roleRegGeneral, roleImmediate,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opAppendByteFast, roleRegGeneral, roleRegGeneral, roleRegUint,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opAppendByteFastInPlace, roleRegGeneral, roleRegGeneral, roleRegUint,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opAppendInPlace, roleRegGeneral, roleRegGeneral, roleRegGeneral,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opAppendSpreadInPlace, roleRegGeneral, roleRegGeneral, roleRegGeneral,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opAppendSpread, roleRegGeneral, roleRegGeneral, roleRegGeneral,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	describeDstSrcSrcMixed(opCopy, roleRegInt, roleRegGeneral)
	described3(opChannelSend, roleRegGeneral, roleRegGeneral, roleNone,
		[numInstructionOperands]bool{true, true, false},
		[numInstructionOperands]bool{false, false, false},
		0)
}

// populateTypedMapIndexOkShapes describes the typed map-comma-ok opcodes used by the
// compiler when both the key kind and value kind match a primitive register bank,
// eliminating the boxing that the general-bank opMapIndexOk path otherwise pays.
func populateTypedMapIndexOkShapes() {
	described3(opMapIndexOkIntInt, roleRegInt, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMapIndexOkStringInt, roleRegInt, roleRegGeneral, roleRegString,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMapIndexOkStringString, roleRegString, roleRegGeneral, roleRegString,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMapIndexOkIntString, roleRegString, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMapGetIntGeneral, roleRegGeneral, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opMapIndexOkIntGeneral, roleRegGeneral, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMapIndexOkJumpIfFalseIntInt, roleRegInt, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMapIndexOkJumpIfFalseStringInt, roleRegInt, roleRegGeneral, roleRegString,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMapIndexOkJumpIfFalseStringString, roleRegString, roleRegGeneral, roleRegString,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMapIndexOkJumpIfFalseIntString, roleRegString, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMapIndexOkJumpIfFalseIntGeneral, roleRegGeneral, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMapIndexOkJumpIfFalseStringGeneral, roleRegGeneral, roleRegGeneral, roleRegString,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
}

// populateAddressTypeShapes describes address-of, dereference, type-assertion, and
// reflect-driven indirection opcodes.
func populateAddressTypeShapes() {
	described3(opAddr, roleRegGeneral, roleRegDynamic, roleKindMarker,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opDeref, roleRegDynamic, roleRegGeneral, roleKindMarker,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opAllocIndirect, roleRegGeneral, roleRegDynamic, roleKindMarker,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opTypeAssert, roleRegGeneral, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, true},
		shapeFlagFollowsExtension)
	described3(opConvert, roleRegGeneral, roleRegGeneral, roleNone,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opBindMethod, roleRegGeneral, roleRegGeneral, roleImmediate,
		[numInstructionOperands]bool{false, true, false},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opMakeClosure, roleRegGeneral, roleConstIndex, roleConstIndex,
		[numInstructionOperands]bool{false, false, false},
		[numInstructionOperands]bool{true, false, false},
		0)
}

// populateGlobalUnsafeShapes describes package-level variable access and the
// unsafe-package intrinsic opcodes.
func populateGlobalUnsafeShapes() {
	described3(opGetGlobal, roleRegDynamic, roleConstIndex, roleKindMarker,
		[numInstructionOperands]bool{false, false, false},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opSetGlobal, roleRegDynamic, roleConstIndex, roleKindMarker,
		[numInstructionOperands]bool{true, false, false},
		[numInstructionOperands]bool{false, false, false},
		0)
	described3(opGetGlobalWide, roleRegDynamic, roleConstIndex, roleKindMarker,
		[numInstructionOperands]bool{false, false, false},
		[numInstructionOperands]bool{true, false, false},
		shapeFlagFollowsExtension)
	described3(opSetGlobalWide, roleRegDynamic, roleConstIndex, roleKindMarker,
		[numInstructionOperands]bool{true, false, false},
		[numInstructionOperands]bool{false, false, false},
		shapeFlagFollowsExtension)
	described3(opUnsafeString, roleRegString, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opUnsafeSlice, roleRegGeneral, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
	described3(opUnsafeAdd, roleRegGeneral, roleRegGeneral, roleRegInt,
		[numInstructionOperands]bool{false, true, true},
		[numInstructionOperands]bool{true, false, false},
		0)
}
