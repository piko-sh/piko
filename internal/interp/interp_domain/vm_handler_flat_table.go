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

//go:generate go run gen_flat_switch.go

package interp_domain

import (
	"fmt"
	"math/bits"
	"unsafe"
)

const (
	// flatHandlerTier1Base is the start of the tier-1 region in the 1024-slot flat dispatch
	// space. See flatDispatchIndex and the generated flatDispatchSwitch in
	// vm_handler_flat_switch.go for how the layout is consumed.
	flatHandlerTier1Base = 256

	// flatHandlerTier2Base is the start of the tier-2 region.
	flatHandlerTier2Base = 512

	// flatHandlerTier3Base is the start of the tier-3 region.
	flatHandlerTier3Base = 768

	// bitsPerByteShift is log2(8) used by flatDispatchIndexTZCNT.
	//
	// Divides a bit index by 8 (>>3) to convert "trailing-zero count" into "which byte holds
	// the lowest non-zero". Used only by the benchmark variant flatDispatchIndexTZCNT, which
	// is transitively unused in production.
	bitsPerByteShift = 3 //nolint:unused // transitively used by flatDispatchIndexTZCNT (benchmark variant)

	// flatTierByteShift = 8, the bit-width of a single instruction operand byte.
	// flatDispatchIndexTZCNT shifts the tier number into the high byte of the resulting
	// flat-dispatch index.
	flatTierByteShift = 8 //nolint:unused // transitively used by flatDispatchIndexTZCNT (benchmark variant)
)

// handleFlatUnknownTier1 is the flat-dispatch arm for an unrecognised tier-1 sub-op.
//
// Reports the offending sub-op A and panics.
//
// Takes vm (*VM) which receives the eval error.
// Takes instr (instruction) which carries the offending sub-op.
//
// Returns opResult which is opPanicError.
func handleFlatUnknownTier1(vm *VM, _ *callFrame, _ *Registers, instr instruction) opResult {
	vm.evalError = fmt.Errorf("invalid umbrella sub-op: %d", instr.a)
	return opPanicError
}

// handleFlatUnknownTier2 mirrors dispatchTier2's default arm.
//
// Reports the offending sub-op B and panics.
//
// Takes vm (*VM) which receives the eval error.
// Takes instr (instruction) which carries the offending sub-op.
//
// Returns opResult which is opPanicError.
func handleFlatUnknownTier2(vm *VM, _ *callFrame, _ *Registers, instr instruction) opResult {
	vm.evalError = fmt.Errorf("invalid tier-2 sub-op: %d", instr.b)
	return opPanicError
}

// handleFlatUnknownTier3 mirrors dispatchTier3's default arm.
//
// Reports the offending sub-op C and panics.
//
// Takes vm (*VM) which receives the eval error.
// Takes instr (instruction) which carries the offending sub-op.
//
// Returns opResult which is opPanicError.
func handleFlatUnknownTier3(vm *VM, _ *callFrame, _ *Registers, instr instruction) opResult {
	vm.evalError = fmt.Errorf("invalid tier-3 sub-op: %d", instr.c)
	return opPanicError
}

// flatDispatchIndex computes the 0..1023 slot in flatDispatchSwitch (and the ASM-side
// flatJumpTable) that should handle the given instruction.
//
// The encoding convention shared with ASM dispatch is documented at opcode.go
// (opDrillTier1's comment): a zero byte in operand position k means "descend into tier
// k+1; the sub-op lives at position k+1". The lowest non-zero byte therefore picks the
// tier (op==tier-0, a==tier-1, b==tier-2, c==tier-3) and its value is the sub-op id.
//
// The branch chain handles tier-0 (the vast majority of dispatched ops in hot workloads)
// in a single predicted branch and a register move, leaving the TZCNT-equivalent fallback
// for the rare tier-1+ instructions. An earlier version always ran TZCNT to be
// branch-free, but the extra ~3 cycles on every tier-0 dispatch cost more than the branch
// saves on tier-1+ - the branch predictor locks onto the dominant tier-0 path with
// near-perfect accuracy.
//
// Takes instr (instruction) which is the 4-byte word to classify.
//
// Returns the flat-table index in the range [0, 1023]. The all-zero instruction word does
// not arise in practice because every emitted instruction has at least one non-zero byte;
// if it did, the index would land on flatHandlerTier3Base which is registered with a
// tier-3 invalid-sub-op handler that surfaces the encoding error.
func flatDispatchIndex(instr instruction) uint {
	if instr.op != 0 {
		return uint(instr.op)
	}
	if instr.a != 0 {
		return flatHandlerTier1Base + uint(instr.a)
	}
	if instr.b != 0 {
		return flatHandlerTier2Base + uint(instr.b)
	}
	return flatHandlerTier3Base + uint(instr.c)
}

// flatDispatchIndexTZCNT is a bit-twiddling dispatch-index variant.
//
// Retained for benchmarking. Single-instruction TZCNT on amd64 (BSF/TZCNT) and a
// CLZ-based sequence on arm64; pure arithmetic, no branches.
//
// Takes instr (instruction) which is the 4-byte word to classify.
//
// Returns uint which is the flat-table index in the range [0, 1023].
//
//nolint:unused // benchmark variant; flatDispatchIndex is the production path
func flatDispatchIndexTZCNT(instr instruction) uint {
	word := *(*uint32)(unsafe.Pointer(&instr))
	if word == 0 {
		return 0
	}
	tzcnt := bits.TrailingZeros32(word)
	tier := uint(tzcnt) >> bitsPerByteShift
	subop := uint(byte(word >> (tier << bitsPerByteShift))) //nolint:gosec // intentional byte-extraction from a 32-bit instruction word; the byte() conversion masks to 8 bits
	return tier<<flatTierByteShift | subop
}

// flatTier1Registrations lists every tier-1 sub-op handler.
//
// Pattern A (inline body) and SIMD entries point at the extracted handlers in
// vm_handler_flat_tier1.go. Pattern B entries reuse existing handlers as-is because they
// already read operands from the tier-1 positions. Pattern C entries point at the named
// wrappers in vm_handler_flat_wrappers.go (a closure-based variant was slower because Go
// cannot inline through a captured function pointer; named wrappers let the compiler
// statically resolve and inline the inner call).
//
// flatTier1Registrations is intentionally partial: subOpDrillTier2 is the tier-1
// drill-down meta-op that initiates tier-2 dispatch and never lands in tier-1 flat-switch
// directly.
//
//nolint:gochecknoglobals,grouper,unused // dispatch table read by gen_flat_switch.go
var flatTier1Registrations = map[subOpcode]opcodeHandler{ //nolint:exhaustive // partial: subOpDrillTier2 dispatched via specialised path

	subOpMathSin:            handleFlatSubOpMathSin,
	subOpMathCos:            handleFlatSubOpMathCos,
	subOpMathExp:            handleFlatSubOpMathExp,
	subOpMathTan:            handleFlatSubOpMathTan,
	subOpMathMod:            handleFlatSubOpMathMod,
	subOpStrconvFormatBool:  handleFlatSubOpStrconvFormatBool,
	subOpStrconvFormatInt:   handleFlatSubOpStrconvFormatInt,
	subOpStrconvItoa:        handleFlatSubOpStrconvItoa,
	subOpRealComplex:        handleFlatSubOpRealComplex,
	subOpImagComplex:        handleFlatSubOpImagComplex,
	subOpBytesToString:      handleFlatSubOpBytesToString,
	subOpCap:                handleFlatSubOpCap,
	subOpNegComplex:         handleFlatSubOpNegComplex,
	subOpMoveComplex:        handleFlatSubOpMoveComplex,
	subOpLoadIntConstSmall:  handleFlatSubOpLoadIntConstSmall,
	subOpLoadBool:           handleFlatSubOpLoadIntConstSmall,
	subOpLoadUintConstSmall: handleFlatSubOpLoadUintConstSmall,

	subOpMakeMethodExpr:          runMakeMethodExpr,
	subOpMakeSliceInt:            handleSubOpMakeSliceInt,
	subOpLenSliceIntDirect:       handleSubOpLenSliceIntDirect,
	subOpMakeSliceFloat:          handleSubOpMakeSliceFloat,
	subOpSliceGetFloatDirect:     handleSubOpSliceGetFloatDirect,
	subOpSliceSetFloatDirect:     handleSubOpSliceSetFloatDirect,
	subOpLenSliceFloatDirect:     handleSubOpLenSliceFloatDirect,
	subOpMakeSliceString:         handleSubOpMakeSliceString,
	subOpSliceGetStringDirect:    handleSubOpSliceGetStringDirect,
	subOpSliceSetStringDirect:    handleSubOpSliceSetStringDirect,
	subOpLenSliceStringDirect:    handleSubOpLenSliceStringDirect,
	subOpMakeSliceBool:           handleSubOpMakeSliceBool,
	subOpSliceGetBoolDirect:      handleSubOpSliceGetBoolDirect,
	subOpSliceSetBoolDirect:      handleSubOpSliceSetBoolDirect,
	subOpLenSliceBoolDirect:      handleSubOpLenSliceBoolDirect,
	subOpMakeSliceUint:           handleSubOpMakeSliceUint,
	subOpSliceGetUintDirect:      handleSubOpSliceGetUintDirect,
	subOpSliceSetUintDirect:      handleSubOpSliceSetUintDirect,
	subOpLenSliceUintDirect:      handleSubOpLenSliceUintDirect,
	subOpBoxSliceInt:             handleSubOpBoxSliceInt,
	subOpUnboxSliceInt:           handleSubOpUnboxSliceInt,
	subOpMakeSliceByte:           handleSubOpMakeSliceByte,
	subOpSliceGetByteDirect:      handleSliceGetByteDirect,
	subOpSliceSetByteDirect:      handleSliceSetByteDirect,
	subOpLenSliceByteDirect:      handleSubOpLenSliceByteDirect,
	subOpCapSliceIntDirect:       handleSubOpCapSliceIntDirect,
	subOpCapSliceFloatDirect:     handleSubOpCapSliceFloatDirect,
	subOpCapSliceStringDirect:    handleSubOpCapSliceStringDirect,
	subOpCapSliceBoolDirect:      handleSubOpCapSliceBoolDirect,
	subOpCapSliceUintDirect:      handleSubOpCapSliceUintDirect,
	subOpCapSliceByteDirect:      handleSubOpCapSliceByteDirect,
	subOpSliceByteSlice:          handleSubOpSliceByteSlice,
	subOpRangeNextSliceByte:      handleRangeNextSliceByte,
	subOpBoxSliceByte:            handleSubOpBoxSliceByte,
	subOpUnboxSliceByte:          handleSubOpUnboxSliceByte,
	subOpSliceByteToString:       handleSubOpSliceByteToString,
	subOpRangeCheckUintJumpFalse: handleSubOpRangeCheckUintJumpFalse,
	subOpEqUintConstJumpFalse:    handleSubOpEqUintConstJumpFalse,
	subOpJump:                    handleJump,

	subOpGetStructFieldInt:    handleGetStructFieldUnsafeInt,
	subOpGetStructFieldUint:   handleGetStructFieldUnsafeUint,
	subOpGetStructFieldFloat:  handleGetStructFieldUnsafeFloat,
	subOpGetStructFieldBool:   handleGetStructFieldUnsafeBool,
	subOpGetStructFieldString: handleGetStructFieldUnsafeString,
	subOpSetStructFieldInt:    handleSetStructFieldUnsafeInt,
	subOpSetStructFieldUint:   handleSetStructFieldUnsafeUint,
	subOpSetStructFieldFloat:  handleSetStructFieldUnsafeFloat,
	subOpSetStructFieldBool:   handleSetStructFieldUnsafeBool,
	subOpSetStructFieldString: handleSetStructFieldUnsafeString,

	subOpGetStructFieldSliceInt:    handleGetStructFieldUnsafeSliceInt,
	subOpGetStructFieldSliceFloat:  handleGetStructFieldUnsafeSliceFloat,
	subOpGetStructFieldSliceUint:   handleGetStructFieldUnsafeSliceUint,
	subOpGetStructFieldSliceString: handleGetStructFieldUnsafeSliceString,
	subOpGetStructFieldSliceBool:   handleGetStructFieldUnsafeSliceBool,
	subOpGetStructFieldSliceByte:   handleGetStructFieldUnsafeSliceByte,

	subOpSetStructFieldSliceInt:    handleSetStructFieldUnsafeSliceInt,
	subOpSetStructFieldSliceFloat:  handleSetStructFieldUnsafeSliceFloat,
	subOpSetStructFieldSliceUint:   handleSetStructFieldUnsafeSliceUint,
	subOpSetStructFieldSliceString: handleSetStructFieldUnsafeSliceString,
	subOpSetStructFieldSliceBool:   handleSetStructFieldUnsafeSliceBool,
	subOpSetStructFieldSliceByte:   handleSetStructFieldUnsafeSliceByte,

	subOpAppendUint:                 handleSubOpAppendUint,
	subOpAppendInt:                  handleSubOpAppendInt,
	subOpAppendString:               handleSubOpAppendString,
	subOpAppendFloat:                handleSubOpAppendFloat,
	subOpAppendBool:                 handleSubOpAppendBool,
	subOpAppendSliceIntDirect:       handleSubOpAppendSliceIntDirect,
	subOpAppendSliceFloatDirect:     handleSubOpAppendSliceFloatDirect,
	subOpAppendSliceStringDirect:    handleSubOpAppendSliceStringDirect,
	subOpAppendSliceBoolDirect:      handleSubOpAppendSliceBoolDirect,
	subOpAppendSliceUintDirect:      handleSubOpAppendSliceUintDirect,
	subOpAppendSliceByteDirect:      handleSubOpAppendSliceByteDirect,
	subOpSliceSliceIntDirect:        handleSubOpSliceSliceIntDirect,
	subOpSliceSliceFloatDirect:      handleSubOpSliceSliceFloatDirect,
	subOpSliceSliceStringDirect:     handleSubOpSliceSliceStringDirect,
	subOpSliceSliceBoolDirect:       handleSubOpSliceSliceBoolDirect,
	subOpSliceSliceUintDirect:       handleSubOpSliceSliceUintDirect,
	subOpCopySliceIntDirect:         handleSubOpCopySliceIntDirect,
	subOpCopySliceFloatDirect:       handleSubOpCopySliceFloatDirect,
	subOpCopySliceStringDirect:      handleSubOpCopySliceStringDirect,
	subOpCopySliceBoolDirect:        handleSubOpCopySliceBoolDirect,
	subOpCopySliceUintDirect:        handleSubOpCopySliceUintDirect,
	subOpCopySliceByteDirect:        handleSubOpCopySliceByteDirect,
	subOpAdoptGeneralToSlicesInt:    handleSubOpAdoptGeneralToSlicesInt,
	subOpAdoptGeneralToSlicesString: handleSubOpAdoptGeneralToSlicesString,
	subOpAdoptGeneralToSlicesBool:   handleSubOpAdoptGeneralToSlicesBool,
	subOpAdoptGeneralToSlicesUint:   handleSubOpAdoptGeneralToSlicesUint,
	subOpAdoptGeneralToSlicesByte:   handleSubOpAdoptGeneralToSlicesByte,
	subOpBoxSliceFloat:              handleSubOpBoxSliceFloat,
	subOpBoxSliceString:             handleSubOpBoxSliceString,
	subOpBoxSliceBool:               handleSubOpBoxSliceBool,
	subOpBoxSliceUint:               handleSubOpBoxSliceUint,
	subOpStarAppendByteFast:         handleSubOpStarAppendByteFast,
	subOpStarAppendByteSpread:       handleSubOpStarAppendByteSpread,
	subOpAppendUintInPlace:          handleSubOpAppendUintInPlace,
	subOpAppendByteSpreadInPlace:    handleSubOpAppendByteSpreadInPlace,
	subOpIncStructFieldInt:          handleSubOpIncStructFieldInt,
	subOpDecStructFieldInt:          handleSubOpDecStructFieldInt,
	subOpIncStructFieldUint:         handleSubOpIncStructFieldUint,
	subOpDecStructFieldUint:         handleSubOpDecStructFieldUint,

	subOpMoveInt:              handleFlatSubOpMoveInt,
	subOpMoveFloat:            handleFlatSubOpMoveFloat,
	subOpMoveString:           handleFlatSubOpMoveString,
	subOpMoveBool:             handleFlatSubOpMoveBool,
	subOpMoveUint:             handleFlatSubOpMoveUint,
	subOpMoveSliceInt:         handleFlatSubOpMoveSliceInt,
	subOpMoveSliceFloat:       handleFlatSubOpMoveSliceFloat,
	subOpMoveSliceString:      handleFlatSubOpMoveSliceString,
	subOpMoveSliceBool:        handleFlatSubOpMoveSliceBool,
	subOpMoveSliceUint:        handleFlatSubOpMoveSliceUint,
	subOpMoveSliceByte:        handleFlatSubOpMoveSliceByte,
	subOpMoveIntToGeneral:     handleFlatSubOpMoveIntToGeneral,
	subOpMoveGeneralToInt:     handleFlatSubOpMoveGeneralToInt,
	subOpMoveFloatToGeneral:   handleFlatSubOpMoveFloatToGeneral,
	subOpMoveGeneralToFloat:   handleFlatSubOpMoveGeneralToFloat,
	subOpMoveStringToGeneral:  handleFlatSubOpMoveStringToGeneral,
	subOpMoveGeneralToString:  handleFlatSubOpMoveGeneralToString,
	subOpNegInt:               handleFlatSubOpNegInt,
	subOpNegFloat:             handleFlatSubOpNegFloat,
	subOpBitNot:               handleFlatSubOpBitNot,
	subOpBitNotUint:           handleFlatSubOpBitNotUint,
	subOpIntToFloat:           handleFlatSubOpIntToFloat,
	subOpFloatToInt:           handleFlatSubOpFloatToInt,
	subOpNot:                  handleFlatSubOpNot,
	subOpBoolToInt:            handleFlatSubOpBoolToInt,
	subOpIntToBool:            handleFlatSubOpIntToBool,
	subOpIntToUint:            handleFlatSubOpIntToUint,
	subOpUintToInt:            handleFlatSubOpUintToInt,
	subOpUintToFloat:          handleFlatSubOpUintToFloat,
	subOpFloatToUint:          handleFlatSubOpFloatToUint,
	subOpMathSqrt:             handleFlatSubOpMathSqrt,
	subOpMathAbs:              handleFlatSubOpMathAbs,
	subOpMathFloor:            handleFlatSubOpMathFloor,
	subOpMathCeil:             handleFlatSubOpMathCeil,
	subOpMathTrunc:            handleFlatSubOpMathTrunc,
	subOpMathRound:            handleFlatSubOpMathRound,
	subOpLenString:            handleFlatSubOpLenString,
	subOpRuneToString:         handleFlatSubOpRuneToString,
	subOpStrToUpper:           handleFlatSubOpStrToUpper,
	subOpStrToLower:           handleFlatSubOpStrToLower,
	subOpStrTrimSpace:         handleFlatSubOpStrTrimSpace,
	subOpLen:                  handleFlatSubOpLen,
	subOpStringToBytes:        handleFlatSubOpStringToBytes,
	subOpUnsafeStringData:     handleFlatSubOpUnsafeStringData,
	subOpUnsafeSliceData:      handleFlatSubOpUnsafeSliceData,
	subOpAddUintConst:         handleFlatSubOpAddUintConst,
	subOpSubUintConst:         handleFlatSubOpSubUintConst,
	subOpBitAndUintConst:      handleFlatSubOpBitAndUintConst,
	subOpIncIntJumpLt:         handleFlatSubOpIncIntJumpLt,
	subOpLenStringLtJumpFalse: handleFlatSubOpLenStringLtJumpFalse,
	subOpLoadZero:             handleFlatSubOpLoadZero,
	subOpMakeMap:              handleFlatSubOpMakeMap,
	subOpMakeChannel:          handleFlatSubOpMakeChannel,
	subOpMapDelete:            handleFlatSubOpMapDelete,
	subOpChannelReceive:       handleFlatSubOpChannelReceive,
	subOpGetMethod:            handleFlatSubOpGetMethod,
	subOpSpill:                handleFlatSubOpSpill,
	subOpReload:               handleFlatSubOpReload,

	subOpSimdDotProductFloat64:               handleFlatSubOpSimdDotProductFloat64,
	subOpSimdSumSliceFloat64:                 handleFlatSubOpSimdSumSliceFloat64,
	subOpSimdNormSquaredFloat64:              handleFlatSubOpSimdNormSquaredFloat64,
	subOpSimdEuclideanDistanceSquaredFloat64: handleFlatSubOpSimdEuclideanDistanceSquaredFloat64,
	subOpSimdMaxSliceFloat64:                 handleFlatSubOpSimdMaxSliceFloat64,
	subOpSimdMinSliceFloat64:                 handleFlatSubOpSimdMinSliceFloat64,
	subOpSimdAddSliceFloat64:                 handleFlatSubOpSimdAddSliceFloat64,
	subOpSimdSubSliceFloat64:                 handleFlatSubOpSimdSubSliceFloat64,
	subOpSimdMulSliceFloat64:                 handleFlatSubOpSimdMulSliceFloat64,
	subOpSimdAxpyFloat64:                     handleFlatSubOpSimdAxpyFloat64,
	subOpSimdScaleSliceFloat64:               handleFlatSubOpSimdScaleSliceFloat64,
	subOpSimdClearSliceFloat64:               handleFlatSubOpSimdClearSliceFloat64,
	subOpSimdFillSliceFloat64:                handleFlatSubOpSimdFillSliceFloat64,
	subOpAdoptGeneralToSlicesFloat:           handleFlatSubOpAdoptGeneralToSlicesFloat,
}

// flatTier2Registrations lists every tier-2 sub-op handler.
//
// Each entry points at a named wrapper in vm_handler_flat_wrappers.go that reshapes the
// tier-2 encoding ({b: subop, c: operand}) into the underlying handler's canonical {a:
// operand}.
//
// flatTier2Registrations is intentionally partial: subOpTier2DrillTier3 is the tier-2
// drill-down meta-op that initiates tier-3 dispatch and never lands in tier-2 flat-switch
// directly.
//
//nolint:gochecknoglobals,grouper,unused // dispatch table read by gen_flat_switch.go
var flatTier2Registrations = map[subOpcodeTier2]opcodeHandler{ //nolint:exhaustive // partial: subOpTier2DrillTier3 dispatched via specialised path
	subOpTier2IncInt:       handleFlatSubOpTier2IncInt,
	subOpTier2DecInt:       handleFlatSubOpTier2DecInt,
	subOpTier2IncUint:      handleFlatSubOpTier2IncUint,
	subOpTier2DecUint:      handleFlatSubOpTier2DecUint,
	subOpTier2Panic:        handleFlatSubOpTier2Panic,
	subOpTier2Recover:      handleFlatSubOpTier2Recover,
	subOpTier2SetZero:      handleFlatSubOpTier2SetZero,
	subOpTier2ChannelClose: handleFlatSubOpTier2ChannelClose,
	subOpTier2LoadNil:      handleFlatSubOpTier2LoadNil,
	subOpTier2Return:       handleFlatSubOpTier2Return,
}

// handleFlatSubOpTier3Nop is the no-op handler for the tier-3 nop slot.
//
// Returns opResult which is opContinue.
func handleFlatSubOpTier3Nop(_ *VM, _ *callFrame, _ *Registers, _ instruction) opResult {
	return opContinue
}

// flatTier3Registrations lists every tier-3 sub-op handler.
//
// Only two slots are populated today (Nop, ReturnVoid); the rest fall through to
// handleFlatUnknownTier3 via the generated switch's default arm.
//
//nolint:gochecknoglobals,grouper,unused // read by gen_flat_switch.go.
var flatTier3Registrations = map[subOpcodeTier3]opcodeHandler{
	subOpTier3Nop:        handleFlatSubOpTier3Nop,
	subOpTier3ReturnVoid: handleReturnVoid,
}
