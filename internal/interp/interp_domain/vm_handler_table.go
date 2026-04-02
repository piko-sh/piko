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

// opResult encodes the outcome of an opcode handler as a single byte.
// The zero value (opContinue) is the common case and enables a fast
// branch-predicted check in the dispatch loop.
type opResult uint8

const (
	// opContinue signals the dispatch loop to advance to the next
	// instruction. It is the zero value and the common case.
	opContinue opResult = iota

	// opFrameChanged signals that a frame was pushed or popped and
	// the dispatch loop must reload its frame and register pointers.
	opFrameChanged

	// opDone signals that execution has finished successfully.
	opDone

	// opDivByZero signals a division by zero error.
	opDivByZero

	// opStackOverflow signals that the call stack exceeded maxCallDepth.
	opStackOverflow

	// opPanicError signals a runtime panic; the error is stored in
	// vm.evalError.
	opPanicError
)

// opcodeHandler is the function signature for all opcode handlers.
type opcodeHandler func(vm *VM, frame *callFrame, registers *Registers, instruction instruction) opResult

// handlerTable is the complete dispatch table mapping every opcode to
// its handler function. Populated once at init time.
var handlerTable [opCount]opcodeHandler

// handleInvalidOpcode is the default handler for unregistered opcodes.
// It stores an errInvalidOpcode error in vm.evalError and signals a
// panic so the dispatch loop terminates cleanly.
//
// Takes vm (*VM) which receives the formatted error in vm.evalError.
// Takes instruction (instruction) which provides the invalid opcode for
// the error message.
//
// Returns opResult which is always opPanicError, terminating execution.
func handleInvalidOpcode(vm *VM, _ *callFrame, _ *Registers, instruction instruction) opResult {
	vm.evalError = fmt.Errorf("%w: %s (%d)", errInvalidOpcode, instruction.op, instruction.op)
	return opPanicError
}

// handlerRegistrations maps each opcode to its handler. The init function
// copies these into handlerTable.
var handlerRegistrations = map[opcode]opcodeHandler{
	opNop:                    handleNop,
	opExt:                    handleExt,
	opMoveInt:                handleMoveInt,
	opMoveFloat:              handleMoveFloat,
	opMoveString:             handleMoveString,
	opMoveGeneral:            handleMoveGeneral,
	opLoadIntConst:           handleLoadIntConst,
	opLoadFloatConst:         handleLoadFloatConst,
	opLoadStringConst:        handleLoadStringConst,
	opLoadGeneralConst:       handleLoadGeneralConst,
	opLoadNil:                handleLoadNil,
	opLoadBool:               handleLoadBool,
	opLoadZero:               handleLoadZero,
	opAddInt:                 handleAddInt,
	opSubInt:                 handleSubInt,
	opMulInt:                 handleMulInt,
	opDivInt:                 handleDivInt,
	opRemInt:                 handleRemInt,
	opNegInt:                 handleNegInt,
	opBitAnd:                 handleBitAnd,
	opBitOr:                  handleBitOr,
	opBitXor:                 handleBitXor,
	opBitAndNot:              handleBitAndNot,
	opBitNot:                 handleBitNot,
	opShiftLeft:              handleShiftLeft,
	opShiftRight:             handleShiftRight,
	opSubIntConst:            handleSubIntConst,
	opAddIntConst:            handleAddIntConst,
	opLeIntConstJumpFalse:    handleLeIntConstJumpFalse,
	opLtIntConstJumpFalse:    handleLtIntConstJumpFalse,
	opAddFloat:               handleAddFloat,
	opSubFloat:               handleSubFloat,
	opMulFloat:               handleMulFloat,
	opDivFloat:               handleDivFloat,
	opNegFloat:               handleNegFloat,
	opConcatString:           handleConcatString,
	opLenString:              handleLenString,
	opAdd:                    handleAdd,
	opSub:                    handleSub,
	opMul:                    handleMul,
	opDiv:                    handleDiv,
	opRem:                    handleRem,
	opEqInt:                  handleEqInt,
	opNeInt:                  handleNeInt,
	opLtInt:                  handleLtInt,
	opLeInt:                  handleLeInt,
	opGtInt:                  handleGtInt,
	opGeInt:                  handleGeInt,
	opEqFloat:                handleEqFloat,
	opLtFloat:                handleLtFloat,
	opLeFloat:                handleLeFloat,
	opEqString:               handleEqString,
	opLtString:               handleLtString,
	opLeString:               handleLeString,
	opEqGeneral:              handleEqGeneral,
	opLtGeneral:              handleLtGeneral,
	opLeGeneral:              handleLeGeneral,
	opGtGeneral:              handleGtGeneral,
	opGeGeneral:              handleGeGeneral,
	opNot:                    handleNot,
	opJump:                   handleJump,
	opJumpIfTrue:             handleJumpIfTrue,
	opJumpIfFalse:            handleJumpIfFalse,
	opPackInterface:          handlePackInterface,
	opUnpackInterface:        handleUnpackInterface,
	opIntToFloat:             handleIntToFloat,
	opFloatToInt:             handleFloatToInt,
	opCall:                   handleCall,
	opCallNative:             handleCallNative,
	opCallBuiltin:            handleCallBuiltin,
	opCallIIFE:               handleCallIIFE,
	opReturn:                 handleReturn,
	opReturnVoid:             handleReturnVoid,
	opTailCall:               handleTailCall,
	opMakeClosure:            handleMakeClosure,
	opGetUpvalue:             handleGetUpvalue,
	opSetUpvalue:             handleSetUpvalue,
	opSyncClosureUpvalues:    handleSyncClosureUpvalues,
	opDefer:                  handleDefer,
	opPanic:                  handlePanic,
	opRecover:                handleRecover,
	opGo:                     handleGo,
	opMakeSlice:              handleMakeSlice,
	opMakeMap:                handleMakeMap,
	opMakeChan:               handleMakeChan,
	opIndex:                  handleIndex,
	opIndexSet:               handleIndexSet,
	opSliceGetInt:            handleSliceGetInt,
	opSliceSetInt:            handleSliceSetInt,
	opSliceGetFloat:          handleSliceGetFloat,
	opSliceSetFloat:          handleSliceSetFloat,
	opMapIndex:               handleMapIndex,
	opMapIndexOk:             handleMapIndexOk,
	opMapSet:                 handleMapSet,
	opMapDelete:              handleMapDelete,
	opLen:                    handleLen,
	opAppend:                 handleAppend,
	opSliceOp:                handleSliceOp,
	opCopy:                   handleCopy,
	opCap:                    handleCap,
	opGetField:               handleGetField,
	opSetField:               handleSetField,
	opGetMethod:              handleGetMethod,
	opAddr:                   handleAddr,
	opDeref:                  handleDeref,
	opConvert:                handleConvert,
	opStringToBytes:          handleStringToBytes,
	opBytesToString:          handleBytesToString,
	opRangeInit:              handleRangeInit,
	opRangeNext:              handleRangeNext,
	opTypeAssert:             handleTypeAssert,
	opChanSend:               handleChanSend,
	opChanRecv:               handleChanRecv,
	opChanClose:              handleChanClose,
	opSelect:                 handleSelect,
	opCallMethod:             handleCallMethod,
	opAllocIndirect:          handleAllocIndirect,
	opResetSharedCell:        handleResetSharedCell,
	opWriteSharedCell:        handleWriteSharedCell,
	opMoveBool:               handleMoveBool,
	opLoadBoolConst:          handleLoadBoolConst,
	opMoveUint:               handleMoveUint,
	opLoadUintConst:          handleLoadUintConst,
	opAddUint:                handleAddUint,
	opSubUint:                handleSubUint,
	opMulUint:                handleMulUint,
	opDivUint:                handleDivUint,
	opRemUint:                handleRemUint,
	opBitAndUint:             handleBitAndUint,
	opBitOrUint:              handleBitOrUint,
	opBitXorUint:             handleBitXorUint,
	opBitAndNotUint:          handleBitAndNotUint,
	opBitNotUint:             handleBitNotUint,
	opShiftLeftUint:          handleShiftLeftUint,
	opShiftRightUint:         handleShiftRightUint,
	opEqUint:                 handleEqUint,
	opNeUint:                 handleNeUint,
	opLtUint:                 handleLtUint,
	opLeUint:                 handleLeUint,
	opGtUint:                 handleGtUint,
	opGeUint:                 handleGeUint,
	opIncUint:                handleIncUint,
	opDecUint:                handleDecUint,
	opMoveComplex:            handleMoveComplex,
	opLoadComplexConst:       handleLoadComplexConst,
	opAddComplex:             handleAddComplex,
	opSubComplex:             handleSubComplex,
	opMulComplex:             handleMulComplex,
	opDivComplex:             handleDivComplex,
	opNegComplex:             handleNegComplex,
	opEqComplex:              handleEqComplex,
	opNeComplex:              handleNeComplex,
	opIntToUint:              handleIntToUint,
	opUintToInt:              handleUintToInt,
	opUintToFloat:            handleUintToFloat,
	opFloatToUint:            handleFloatToUint,
	opBoolToInt:              handleBoolToInt,
	opIntToBool:              handleIntToBool,
	opRealComplex:            handleRealComplex,
	opImagComplex:            handleImagComplex,
	opBuildComplex:           handleBuildComplex,
	opUnsafeString:           handleUnsafeString,
	opUnsafeStringData:       handleUnsafeStringData,
	opUnsafeSlice:            handleUnsafeSlice,
	opUnsafeSliceData:        handleUnsafeSliceData,
	opUnsafeAdd:              handleUnsafeAdd,
	opGetGlobal:              handleGetGlobal,
	opSetGlobal:              handleSetGlobal,
	opBindMethod:             handleBindMethod,
	opMakeMethodExpr:         handleMakeMethodExpr,
	opStringIndex:            handleStringIndex,
	opRuneToString:           handleRuneToString,
	opSliceString:            handleSliceString,
	opStrContainsRune:        handleStrContainsRune,
	opStrContains:            handleStrContains,
	opStrHasPrefix:           handleStrHasPrefix,
	opStrHasSuffix:           handleStrHasSuffix,
	opStrEqualFold:           handleStrEqualFold,
	opStrIndex:               handleStrIndex,
	opStrCount:               handleStrCount,
	opStrToUpper:             handleStrToUpper,
	opStrToLower:             handleStrToLower,
	opStrTrimSpace:           handleStrTrimSpace,
	opStrTrimPrefix:          handleStrTrimPrefix,
	opStrTrimSuffix:          handleStrTrimSuffix,
	opStrTrim:                handleStrTrim,
	opStrIndexRune:           handleStrIndexRune,
	opMathAbs:                handleMathAbs,
	opMathSqrt:               handleMathSqrt,
	opMathFloor:              handleMathFloor,
	opMathCeil:               handleMathCeil,
	opMathRound:              handleMathRound,
	opStrconvItoa:            handleStrconvItoa,
	opStrconvFormatBool:      handleStrconvFormatBool,
	opStrconvFormatInt:       handleStrconvFormatInt,
	opMathPow:                handleMathPow,
	opMathExp:                handleMathExp,
	opMathSin:                handleMathSin,
	opMathCos:                handleMathCos,
	opMathTan:                handleMathTan,
	opMathMod:                handleMathMod,
	opMathTrunc:              handleMathTrunc,
	opSpill:                  handleSpill,
	opReload:                 handleReload,
	opStrRepeat:              handleStrRepeat,
	opStrLastIndex:           handleStrLastIndex,
	opAppendString:           handleAppendString,
	opAppendFloat:            handleAppendFloat,
	opAppendBool:             handleAppendBool,
	opMapGetIntInt:           handleMapGetIntInt,
	opMapSetIntInt:           handleMapSetIntInt,
	opAppendInt:              handleAppendInt,
	opGetFieldInt:            handleGetFieldInt,
	opSetFieldInt:            handleSetFieldInt,
	opIncInt:                 handleIncInt,
	opDecInt:                 handleDecInt,
	opNeFloat:                handleNeFloat,
	opGtFloat:                handleGtFloat,
	opGeFloat:                handleGeFloat,
	opNeString:               handleNeString,
	opGtString:               handleGtString,
	opGeString:               handleGeString,
	opNeGeneral:              handleNeGeneral,
	opEqIntConstJumpFalse:    handleEqIntConstJumpFalse,
	opEqIntConstJumpTrue:     handleEqIntConstJumpTrue,
	opGeIntConstJumpFalse:    handleGeIntConstJumpFalse,
	opGtIntConstJumpFalse:    handleGtIntConstJumpFalse,
	opAddIntJump:             handleAddIntJump,
	opIncIntJumpLt:           handleIncIntJumpLt,
	opMulIntConst:            handleMulIntConst,
	opSliceGetString:         handleSliceGetString,
	opSliceSetString:         handleSliceSetString,
	opSliceGetBool:           handleSliceGetBool,
	opSliceSetBool:           handleSliceSetBool,
	opSliceGetUint:           handleSliceGetUint,
	opSliceSetUint:           handleSliceSetUint,
	opLoadIntConstSmall:      handleLoadIntConstSmall,
	opEqStringConstJumpFalse: handleEqStringConstJumpFalse,
	opMoveIntToGeneral:       handleMoveIntToGeneral,
	opMoveGeneralToInt:       handleMoveGeneralToInt,
	opMoveFloatToGeneral:     handleMoveFloatToGeneral,
	opMoveGeneralToFloat:     handleMoveGeneralToFloat,
	opMoveStringToGeneral:    handleMoveStringToGeneral,
	opMoveGeneralToString:    handleMoveGeneralToString,
	opTestNilJumpTrue:        handleTestNilJumpTrue,
	opTestNilJumpFalse:       handleTestNilJumpFalse,
	opConcatRuneString:       handleConcatRuneString,
	opStrJoin:                handleStrJoin,
	opStrSplit:               handleStrSplit,
	opStrReplaceAll:          handleStrReplaceAll,
	opSetZero:                handleSetZero,
	opGetGlobalWide:          handleGetGlobalWide,
	opSetGlobalWide:          handleSetGlobalWide,
	opStringIndexToInt:       handleStringIndexToInt,
	opLenStringLtJumpFalse:   handleLenStringLtJumpFalse,
}

func init() {
	for i := range handlerTable {
		handlerTable[i] = handleInvalidOpcode
	}

	for op, h := range handlerRegistrations {
		handlerTable[op] = h
	}
}
