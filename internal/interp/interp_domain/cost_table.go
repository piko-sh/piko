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

// CostTable holds the per-opcode cost weights used by the runtime cost
// metering system. Callers can provide a custom table via
// WithCostTable to adjust weights for their environment.
type CostTable [opCount]int64

const (
	// costFree represents zero cost for register moves, constant loads, and extensions.
	costFree int64 = 0

	// costCheap represents low cost for arithmetic, comparisons, jumps, and type conversions.
	costCheap int64 = 1

	// costMedium represents medium cost for string ops, field access, and math intrinsics.
	costMedium int64 = 3

	// costModerate represents moderate cost for index/map
	// access, closures, channels, and defer/panic.
	costModerate int64 = 5

	// costExpensive represents high cost for function calls,
	// goroutine spawn, append, and select.
	costExpensive int64 = 10

	// costVeryHeavy represents very high cost for allocations (make slice/map/chan).
	costVeryHeavy int64 = 20
)

// DefaultCostTable returns the default cost table with weights
// reflecting approximate real-world computation expense.
//
//	0  - free: register moves, constant loads, extensions
//	1  - cheap: arithmetic, comparisons, jumps, type conversions
//	3  - medium: string ops, field access, math intrinsics
//	5  - moderate: index/map access, closures, channels, defer/panic
//	10 - expensive: function calls, goroutine spawn, append, select
//	20 - very expensive: allocations (make slice/map/chan)
//
// Returns CostTable populated with the default cost weights.
func DefaultCostTable() CostTable {
	var table CostTable
	initDefaultCostTable(&table)
	return table
}

// defaultCostTable is the package-level default used when no custom
// table is provided.
var defaultCostTable CostTable

func init() {
	initDefaultCostTable(&defaultCostTable)
}

// initDefaultCostTable populates a CostTable with the default per-opcode cost weights.
//
// Takes table (*CostTable) which is the cost table to initialise.
func initDefaultCostTable(table *CostTable) {
	for i := range table {
		table[i] = costCheap
	}
	assignFreeOps(table)
	assignMediumOps(table)
	assignModerateOps(table)
	assignExpensiveOps(table)
	assignVeryHeavyOps(table)
}

// assignFreeOps sets all zero-cost opcodes in the given cost table.
//
// Takes table (*CostTable) which is the cost table to modify.
func assignFreeOps(table *CostTable) {
	free := []opcode{
		opNop, opExt,
		opMoveInt, opMoveFloat, opMoveString, opMoveGeneral,
		opMoveBool, opMoveUint, opMoveComplex,
		opLoadIntConst, opLoadFloatConst, opLoadBool,
		opLoadIntConstSmall, opLoadStringConst, opLoadGeneralConst,
		opLoadBoolConst, opLoadUintConst, opLoadComplexConst,
		opLoadNil, opLoadZero,
		opReturn, opReturnVoid,
	}
	for _, op := range free {
		table[op] = costFree
	}
}

// assignMediumOps sets all medium-cost opcodes in the given cost table.
//
// Takes table (*CostTable) which is the cost table to modify.
func assignMediumOps(table *CostTable) {
	medium := []opcode{
		opConcatString, opSliceString, opRuneToString,
		opConcatRuneString, opStringIndexToInt,
		opLenStringLtJumpFalse,
		opEqString, opNeString, opLtString, opLeString,
		opGtString, opGeString,
		opConvert, opStringToBytes, opBytesToString,
		opGetField, opSetField, opSetFieldInt, opGetFieldInt,
		opAddr, opDeref,
		opMathSqrt, opMathAbs, opMathFloor, opMathCeil,
		opMathTrunc, opMathRound, opMathPow, opMathExp,
		opMathSin, opMathCos, opMathTan, opMathMod,
		opStrconvItoa, opStrconvFormatBool, opStrconvFormatInt,
		opEqGeneral, opNeGeneral, opLtGeneral, opLeGeneral,
		opGtGeneral, opGeGeneral,
		opAdd, opSub, opMul, opDiv, opRem,
	}
	for _, op := range medium {
		table[op] = costMedium
	}
}

// assignModerateOps sets all moderate-cost opcodes in the given cost table.
//
// Takes table (*CostTable) which is the cost table to modify.
func assignModerateOps(table *CostTable) {
	moderate := []opcode{
		opIndex, opIndexSet, opSliceOp,
		opMapIndex, opMapSet, opMapIndexOk, opMapDelete,
		opMapGetIntInt, opMapSetIntInt,
		opSliceGetInt, opSliceSetInt,
		opSliceGetFloat, opSliceSetFloat,
		opSliceGetString, opSliceSetString,
		opSliceGetBool, opSliceSetBool,
		opSliceGetUint, opSliceSetUint,
		opDefer, opPanic, opRecover,
		opMakeClosure,
		opChanSend, opChanRecv, opChanClose,
		opLen, opCap, opCopy,
		opTypeAssert,
		opAllocIndirect,
		opSetZero,
		opGetGlobal, opSetGlobal,
		opGetGlobalWide, opSetGlobalWide,
		opSpill, opReload,
		opGetMethod, opBindMethod, opMakeMethodExpr,
		opStrContains, opStrContainsRune,
		opStrHasPrefix, opStrHasSuffix, opStrEqualFold,
		opStrIndex, opStrCount,
		opStrToUpper, opStrToLower, opStrTrimSpace,
		opStrTrimPrefix, opStrTrimSuffix, opStrTrim,
		opStrIndexRune, opStrRepeat, opStrLastIndex,
		opStrJoin, opStrSplit, opStrReplaceAll,
		opRangeInit, opRangeNext,
		opGetUpvalue, opSetUpvalue,
		opSyncClosureUpvalues, opResetSharedCell, opWriteSharedCell,
		opPackInterface, opUnpackInterface,
		opMoveIntToGeneral, opMoveGeneralToInt,
		opMoveFloatToGeneral, opMoveGeneralToFloat,
		opMoveStringToGeneral, opMoveGeneralToString,
		opTestNilJumpTrue, opTestNilJumpFalse,
		opEqStringConstJumpFalse,
		opUnsafeString, opUnsafeStringData,
		opUnsafeSlice, opUnsafeSliceData, opUnsafeAdd,
	}
	for _, op := range moderate {
		table[op] = costModerate
	}
}

// assignExpensiveOps sets all expensive-cost opcodes in the given cost table.
//
// Takes table (*CostTable) which is the cost table to modify.
func assignExpensiveOps(table *CostTable) {
	expensive := []opcode{
		opCall, opCallNative, opCallMethod, opCallIIFE,
		opCallBuiltin, opTailCall,
		opGo, opSelect,
		opAppend, opAppendInt, opAppendString,
		opAppendFloat, opAppendBool,
	}
	for _, op := range expensive {
		table[op] = costExpensive
	}
}

// assignVeryHeavyOps sets all very-heavy-cost opcodes in the given cost table.
//
// Takes table (*CostTable) which is the cost table to modify.
func assignVeryHeavyOps(table *CostTable) {
	veryHeavy := []opcode{
		opMakeSlice, opMakeMap, opMakeChan,
	}
	for _, op := range veryHeavy {
		table[op] = costVeryHeavy
	}
}
