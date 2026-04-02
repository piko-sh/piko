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

// opcode identifies a virtual machine operation.
type opcode uint8

const (
	// opNop performs no operation.
	opNop opcode = iota

	// opExt is an extension word carrying a 24-bit payload formed from
	// A|(B<<8)|(C<<16). Used for wide jump offsets, large constant
	// indices, and multi-word encodings.
	opExt

	// opMoveInt copies ints[B] to ints[A].
	opMoveInt

	// opMoveFloat copies floats[B] to floats[A].
	opMoveFloat

	// opLoadIntConst loads intConstants[B|(C<<8)] into ints[A].
	opLoadIntConst

	// opLoadFloatConst loads floatConstants[B|(C<<8)] into floats[A].
	opLoadFloatConst

	// opLoadBool sets ints[A] to B (0 for false, 1 for true).
	opLoadBool

	// opLoadIntConstSmall sets ints[A] = int64(B). Embeds the
	// immediate value directly (0-255), avoiding constant table lookup.
	opLoadIntConstSmall

	// opAddInt sets ints[A] = ints[B] + ints[C].
	opAddInt

	// opSubInt sets ints[A] = ints[B] - ints[C].
	opSubInt

	// opMulInt sets ints[A] = ints[B] * ints[C].
	opMulInt

	// opDivInt sets ints[A] = ints[B] / ints[C]. Panics on zero divisor.
	opDivInt

	// opRemInt sets ints[A] = ints[B] % ints[C]. Panics on zero divisor.
	opRemInt

	// opNegInt sets ints[A] = -ints[B].
	opNegInt

	// opIncInt increments ints[A] by one.
	opIncInt

	// opDecInt decrements ints[A] by one.
	opDecInt

	// opBitAnd sets ints[A] = ints[B] & ints[C].
	opBitAnd

	// opBitOr sets ints[A] = ints[B] | ints[C].
	opBitOr

	// opBitXor sets ints[A] = ints[B] ^ ints[C].
	opBitXor

	// opBitAndNot sets ints[A] = ints[B] &^ ints[C].
	opBitAndNot

	// opBitNot sets ints[A] = ^ints[B].
	opBitNot

	// opShiftLeft sets ints[A] = ints[B] << uint(ints[C]).
	opShiftLeft

	// opShiftRight sets ints[A] = ints[B] >> uint(ints[C]).
	opShiftRight

	// opAddFloat sets floats[A] = floats[B] + floats[C].
	opAddFloat

	// opSubFloat sets floats[A] = floats[B] - floats[C].
	opSubFloat

	// opMulFloat sets floats[A] = floats[B] * floats[C].
	opMulFloat

	// opDivFloat sets floats[A] = floats[B] / floats[C].
	opDivFloat

	// opNegFloat sets floats[A] = -floats[B].
	opNegFloat

	// opEqInt sets ints[A] = (ints[B] == ints[C]) ? 1 : 0.
	opEqInt

	// opNeInt sets ints[A] = (ints[B] != ints[C]) ? 1 : 0.
	opNeInt

	// opLtInt sets ints[A] = (ints[B] < ints[C]) ? 1 : 0.
	opLtInt

	// opLeInt sets ints[A] = (ints[B] <= ints[C]) ? 1 : 0.
	opLeInt

	// opGtInt sets ints[A] = (ints[B] > ints[C]) ? 1 : 0.
	opGtInt

	// opGeInt sets ints[A] = (ints[B] >= ints[C]) ? 1 : 0.
	opGeInt

	// opEqFloat sets ints[A] = (floats[B] == floats[C]) ? 1 : 0.
	opEqFloat

	// opNeFloat sets ints[A] = (floats[B] != floats[C]) ? 1 : 0.
	opNeFloat

	// opLtFloat sets ints[A] = (floats[B] < floats[C]) ? 1 : 0.
	opLtFloat

	// opLeFloat sets ints[A] = (floats[B] <= floats[C]) ? 1 : 0.
	opLeFloat

	// opGtFloat sets ints[A] = (floats[B] > floats[C]) ? 1 : 0.
	opGtFloat

	// opGeFloat sets ints[A] = (floats[B] >= floats[C]) ? 1 : 0.
	opGeFloat

	// opIntToFloat converts ints[B] to floats[A].
	opIntToFloat

	// opFloatToInt converts floats[B] to ints[A].
	opFloatToInt

	// opNot sets ints[A] = (ints[B] == 0) ? 1 : 0 (logical not).
	opNot

	// opJump unconditionally jumps by signed offset encoded in B|(C<<8).
	opJump

	// opJumpIfTrue jumps if ints[A] != 0. Offset in B|(C<<8).
	opJumpIfTrue

	// opJumpIfFalse jumps if ints[A] == 0. Offset in B|(C<<8).
	opJumpIfFalse

	// opCall calls a compiled function. general[A] holds the callee,
	// B is the argument count, and C encodes return destination info.
	opCall

	// opReturn returns from the current function with A return values.
	opReturn

	// opReturnVoid returns from the current function with no values.
	opReturnVoid

	// opTailCall performs a tail call (optimisation, future).
	opTailCall

	// opSubIntConst sets ints[A] = ints[B] - intConstants[C].
	// Fuses opLoadIntConst + opSubInt when the constant index fits in 8 bits.
	opSubIntConst

	// opAddIntConst sets ints[A] = ints[B] + intConstants[C].
	// Fuses opLoadIntConst + opAddInt when the constant index fits in 8 bits.
	opAddIntConst

	// opLeIntConstJumpFalse compares ints[A] <= intConstants[B] and jumps
	// by offset in the following opExt if false (ints[A] > constant).
	// Fuses opLoadIntConst + opLeInt + opJumpIfFalse.
	opLeIntConstJumpFalse

	// opLtIntConstJumpFalse compares ints[A] < intConstants[B] and jumps
	// by offset in the following opExt if false (ints[A] >= constant).
	// Fuses opLoadIntConst + opLtInt + opJumpIfFalse.
	opLtIntConstJumpFalse

	// opEqIntConstJumpFalse compares ints[A] == intConstants[B] and
	// jumps by offset in the following opExt if false (not equal).
	// Fuses opLoadIntConst + opEqInt + opJumpIfFalse.
	opEqIntConstJumpFalse

	// opEqIntConstJumpTrue compares ints[A] == intConstants[B] and
	// jumps by offset in the following opExt if true (equal).
	// Fuses opLoadIntConst + opEqInt + opJumpIfTrue.
	opEqIntConstJumpTrue

	// opGeIntConstJumpFalse compares ints[A] >= intConstants[B] and
	// jumps by offset in the following opExt if false (less than).
	// Fuses opLoadIntConst + opGeInt + opJumpIfFalse.
	opGeIntConstJumpFalse

	// opGtIntConstJumpFalse compares ints[A] > intConstants[B] and
	// jumps by offset in the following opExt if false (less or equal).
	// Fuses opLoadIntConst + opGtInt + opJumpIfFalse.
	opGtIntConstJumpFalse

	// opMulIntConst sets ints[A] = ints[B] * intConstants[C].
	// Fuses opLoadIntConst + opMulInt when the constant index fits
	// in 8 bits.
	opMulIntConst

	// opAddIntJump sets ints[A] = ints[B] + intConstants[C] and
	// unconditionally jumps by offset in the following opExt.
	// Fuses opAddIntConst + opJump.
	opAddIntJump

	// opIncIntJumpLt increments ints[A] and jumps by offset in the
	// following opExt if ints[A] < ints[B]. Fuses opIncInt + opLtInt
	// + opJumpIfTrue for the canonical for-loop back-edge.
	opIncIntJumpLt

	// opMathSqrt sets floats[A] = math.Sqrt(floats[B]).
	opMathSqrt

	// opMathAbs sets floats[A] = math.Abs(floats[B]).
	opMathAbs

	// opMathFloor sets floats[A] = math.Floor(floats[B]).
	opMathFloor

	// opMathCeil sets floats[A] = math.Ceil(floats[B]).
	opMathCeil

	// opMathTrunc sets floats[A] = math.Trunc(floats[B]).
	opMathTrunc

	// opMathRound sets floats[A] = math.Round(floats[B]).
	opMathRound

	// opLenString sets ints[A] = len(strings[B]).
	opLenString

	// opStringIndex sets uints[A] = uint64(strings[B][ints[C]]).
	// Panics with errIndexOutOfRange if index is out of bounds.
	opStringIndex

	// opEqString sets ints[A] = (strings[B] == strings[C]) ? 1 : 0.
	opEqString

	// opNeString sets ints[A] = (strings[B] != strings[C]) ? 1 : 0.
	opNeString

	// opSliceString sets strings[A] = strings[B][low:high]. C encodes
	// flags (bit 0 = low present, bit 1 = high present) and an opExt
	// follows with A=lowReg, B=highReg.
	opSliceString

	// opStringIndexToInt sets ints[A] = int64(strings[B][ints[C]]).
	// Fuses opStringIndex + opUintToInt to avoid the intermediate
	// uint register and one tier-2 trampoline in string loops.
	opStringIndexToInt

	// opLenStringLtJumpFalse jumps if ints[A] >= len(strings[B]),
	// fusing opLenString + opLtInt + opJumpIfFalse for string loop
	// conditions. Uses one extension word for the jump offset.
	opLenStringLtJumpFalse

	// opMoveString copies strings[B] to strings[A].
	opMoveString

	// opMoveGeneral copies general[B] to general[A].
	opMoveGeneral

	// opMoveBool copies bools[B] to bools[A].
	opMoveBool

	// opMoveUint copies uints[B] to uints[A].
	opMoveUint

	// opMoveComplex copies complex[B] to complex[A].
	opMoveComplex

	// opLoadStringConst loads stringConstants[B|(C<<8)] into strings[A].
	opLoadStringConst

	// opLoadGeneralConst loads generalConstants[B|(C<<8)] into general[A].
	opLoadGeneralConst

	// opLoadBoolConst loads BoolConstants[B] into bools[A].
	opLoadBoolConst

	// opLoadUintConst loads UintConstants[B|(C<<8)] into uints[A].
	opLoadUintConst

	// opLoadComplexConst loads ComplexConstants[B|(C<<8)] into complex[A].
	opLoadComplexConst

	// opLoadNil sets general[A] to the zero reflect.Value (nil).
	opLoadNil

	// opLoadZero zeroes the register at index A in bank B.
	opLoadZero

	// opAddUint sets uints[A] = uints[B] + uints[C].
	opAddUint

	// opSubUint sets uints[A] = uints[B] - uints[C].
	opSubUint

	// opMulUint sets uints[A] = uints[B] * uints[C].
	opMulUint

	// opDivUint sets uints[A] = uints[B] / uints[C]. Panics on zero divisor.
	opDivUint

	// opRemUint sets uints[A] = uints[B] % uints[C]. Panics on zero divisor.
	opRemUint

	// opBitAndUint sets uints[A] = uints[B] & uints[C].
	opBitAndUint

	// opBitOrUint sets uints[A] = uints[B] | uints[C].
	opBitOrUint

	// opBitXorUint sets uints[A] = uints[B] ^ uints[C].
	opBitXorUint

	// opBitAndNotUint sets uints[A] = uints[B] &^ uints[C].
	opBitAndNotUint

	// opBitNotUint sets uints[A] = ^uints[B].
	opBitNotUint

	// opShiftLeftUint sets uints[A] = uints[B] << uints[C].
	opShiftLeftUint

	// opShiftRightUint sets uints[A] = uints[B] >> uints[C].
	opShiftRightUint

	// opEqUint sets ints[A] = (uints[B] == uints[C]) ? 1 : 0.
	opEqUint

	// opNeUint sets ints[A] = (uints[B] != uints[C]) ? 1 : 0.
	opNeUint

	// opLtUint sets ints[A] = (uints[B] < uints[C]) ? 1 : 0.
	opLtUint

	// opLeUint sets ints[A] = (uints[B] <= uints[C]) ? 1 : 0.
	opLeUint

	// opGtUint sets ints[A] = (uints[B] > uints[C]) ? 1 : 0.
	opGtUint

	// opGeUint sets ints[A] = (uints[B] >= uints[C]) ? 1 : 0.
	opGeUint

	// opIncUint increments uints[A] by one.
	opIncUint

	// opDecUint decrements uints[A] by one.
	opDecUint

	// opAddComplex sets complex[A] = complex[B] + complex[C].
	opAddComplex

	// opSubComplex sets complex[A] = complex[B] - complex[C].
	opSubComplex

	// opMulComplex sets complex[A] = complex[B] * complex[C].
	opMulComplex

	// opDivComplex sets complex[A] = complex[B] / complex[C].
	opDivComplex

	// opNegComplex sets complex[A] = -complex[B].
	opNegComplex

	// opEqComplex sets ints[A] = (complex[B] == complex[C]) ? 1 : 0.
	opEqComplex

	// opNeComplex sets ints[A] = (complex[B] != complex[C]) ? 1 : 0.
	opNeComplex

	// opRealComplex sets floats[A] = real(complex[B]).
	opRealComplex

	// opImagComplex sets floats[A] = imag(complex[B]).
	opImagComplex

	// opBuildComplex sets complex[A] = complex(floats[B], floats[C]).
	opBuildComplex

	// opConcatString sets strings[A] = strings[B] + strings[C].
	opConcatString

	// opRuneToString sets strings[A] = string(rune(ints[B])).
	opRuneToString

	// opConcatRuneString sets strings[A] = strings[B] + string(rune(ints[C])).
	// Fuses opRuneToString + opConcatString with in-place arena extension.
	opConcatRuneString

	// opLtString sets ints[A] = (strings[B] < strings[C]) ? 1 : 0.
	opLtString

	// opLeString sets ints[A] = (strings[B] <= strings[C]) ? 1 : 0.
	opLeString

	// opGtString sets ints[A] = (strings[B] > strings[C]) ? 1 : 0.
	opGtString

	// opGeString sets ints[A] = (strings[B] >= strings[C]) ? 1 : 0.
	opGeString

	// opEqGeneral sets ints[A] = (general[B] == general[C]) ? 1 : 0 via reflect.
	opEqGeneral

	// opNeGeneral sets ints[A] = (general[B] != general[C]) ? 1 : 0
	// via reflect, using the same equality logic as opEqGeneral.
	opNeGeneral

	// opLtGeneral sets ints[A] = (general[B] < general[C]) ? 1 : 0 via reflect.
	opLtGeneral

	// opLeGeneral sets ints[A] = (general[B] <= general[C]) ? 1 : 0 via reflect.
	opLeGeneral

	// opGtGeneral sets ints[A] = (general[B] > general[C]) ? 1 : 0 via reflect.
	opGtGeneral

	// opGeGeneral sets ints[A] = (general[B] >= general[C]) ? 1 : 0 via reflect.
	opGeGeneral

	// opAdd sets general[A] = general[B] + general[C] via reflect.
	opAdd

	// opSub sets general[A] = general[B] - general[C] via reflect.
	opSub

	// opMul sets general[A] = general[B] * general[C] via reflect.
	opMul

	// opDiv sets general[A] = general[B] / general[C] via reflect.
	opDiv

	// opRem sets general[A] = general[B] % general[C] via reflect.
	opRem

	// opBoolToInt sets ints[A] = int64(bools[B]) (0 or 1).
	opBoolToInt

	// opIntToBool sets bools[A] = ints[B] != 0.
	opIntToBool

	// opIntToUint sets uints[A] = uint64(ints[B]).
	opIntToUint

	// opUintToInt sets ints[A] = int64(uints[B]).
	opUintToInt

	// opUintToFloat sets floats[A] = float64(uints[B]).
	opUintToFloat

	// opFloatToUint sets uints[A] = uint64(floats[B]).
	opFloatToUint

	// opMoveIntToGeneral sets general[A] = reflect.ValueOf(ints[B]).
	opMoveIntToGeneral

	// opMoveGeneralToInt sets ints[A] = general[B].Int().
	opMoveGeneralToInt

	// opMoveFloatToGeneral sets general[A] = reflect.ValueOf(floats[B]).
	opMoveFloatToGeneral

	// opMoveGeneralToFloat sets floats[A] = general[B].Float().
	opMoveGeneralToFloat

	// opMoveStringToGeneral sets general[A] = reflect.ValueOf(strings[B]).
	opMoveStringToGeneral

	// opMoveGeneralToString sets strings[A] = general[B].String().
	opMoveGeneralToString

	// opPackInterface wraps a typed value into an interface.
	opPackInterface

	// opUnpackInterface extracts the concrete value from an interface.
	opUnpackInterface

	// opTestNilJumpTrue tests if general[A] is nil/invalid and jumps
	// by signed offset B|(C<<8) if true.
	opTestNilJumpTrue

	// opTestNilJumpFalse tests if general[A] is nil/invalid and jumps
	// by signed offset B|(C<<8) if false (i.e. not nil).
	opTestNilJumpFalse

	// opEqStringConstJumpFalse compares strings[A] == stringConstants[B]
	// and jumps by offset in the following opExt if false (not equal).
	// Fuses opLoadStringConst + opEqString + opJumpIfFalse.
	opEqStringConstJumpFalse

	// opCallNative calls a native reflect.Value function in general[A]
	// with B arguments.
	opCallNative

	// opCallBuiltin calls a built-in function identified by A.
	// B and C are builtin-specific operands.
	opCallBuiltin

	// opCallMethod dispatches a method call using the runtime method
	// table with call site index B|C<<8, where the receiver is the
	// first argument.
	//
	// The method name string constant index is encoded in an extension word.
	opCallMethod

	// opCallIIFE calls an immediately invoked function expression
	// with call site index B|(C<<8).
	//
	// It fuses opMakeClosure + opCall, skipping runtimeClosure allocation
	// and reflect.ValueOf boxing.
	opCallIIFE

	// opMakeClosure creates a closure in general[A] from function
	// index B|(C<<8). Upvalue descriptors are in the CompiledFunction.
	opMakeClosure

	// opGetUpvalue loads upvalue[B] into the register at A.
	// C encodes the register kind.
	opGetUpvalue

	// opSetUpvalue stores the register at A into upvalue[B].
	// C encodes the register kind.
	opSetUpvalue

	// opSyncClosureUpvalues reads upvalue cells from the closure in
	// general[A] back to parent registers. Used after an immediately
	// invoked function expression to propagate captured mutations.
	opSyncClosureUpvalues

	// opResetSharedCell invalidates the shared upvalue cell for the
	// register at index A in bank B, forcing the next opMakeClosure
	// to create a fresh cell. Used for Go 1.22+ per-iteration loop
	// variable scoping.
	opResetSharedCell

	// opWriteSharedCell copies the current register value (bank B,
	// index A) into the corresponding shared upvalue cell if one
	// exists, keeping the cell in sync after a parent-frame write
	// to a variable that has already been captured by a closure
	// (e.g. `var f func(); f = func() { f() }`).
	opWriteSharedCell

	// opDefer pushes a deferred call. general[A] is the function,
	// B is the number of arguments.
	opDefer

	// opPanic panics with the value in general[A].
	opPanic

	// opRecover sets general[A] to the recovered panic value, or nil.
	opRecover

	// opGo spawns a goroutine calling general[A] with B arguments.
	opGo

	// opMakeSlice creates a slice: general[A] = make([]T, ints[B], ints[C]).
	// The type T is looked up from the function's type table via an
	// extension instruction.
	opMakeSlice

	// opMakeMap creates a map: general[A] = make(map[K]V, ints[B]).
	opMakeMap

	// opMakeChan creates a channel: general[A] = make(chan T, ints[B]).
	opMakeChan

	// opIndex reads element: target = general[A][ints[B]].
	// C encodes the destination register kind and index via extension.
	opIndex

	// opIndexSet writes element: general[A][ints[B]] = source.
	opIndexSet

	// opSliceOp slices: general[A] = general[B][low:high:max].
	// Indices come from int registers via extension instructions.
	opSliceOp

	// opMapIndex reads map element: target = general[A][key].
	opMapIndex

	// opMapSet writes map element: general[A][key] = value.
	opMapSet

	// opMapIndexOk reads map element with ok flag:
	// general[A] = general[B][general[C]], ints[extensionWord.a] = ok (0 or 1).
	opMapIndexOk

	// opMapDelete deletes map key: delete(general[A], key).
	opMapDelete

	// opAppend appends: general[A] = append(general[B], arguments...).
	opAppend

	// opCopy copies: ints[A] = copy(general[B], general[C]).
	opCopy

	// opLen sets ints[A] = len(general[B]).
	opLen

	// opCap sets ints[A] = cap(general[B]).
	opCap

	// opSliceGetInt reads an integer element from a slice/array without
	// reflect boxing. ints[A] = general[B].Index(ints[C]).Int() (or .Uint()
	// for unsigned element types).
	opSliceGetInt

	// opSliceSetInt writes an integer value to a slice/array element
	// without reflect boxing. general[A].Index(ints[B]).SetInt(ints[C])
	// (or .SetUint() for unsigned element types).
	opSliceSetInt

	// opSliceGetFloat reads a float element from a slice/array without
	// reflect boxing. floats[A] = general[B].Index(ints[C]).Float().
	opSliceGetFloat

	// opSliceSetFloat writes a float value to a slice/array element
	// without reflect boxing. general[A].Index(ints[B]).SetFloat(floats[C]).
	opSliceSetFloat

	// opSliceGetString reads a string element from a slice/array.
	// strings[A] = general[B].Index(ints[C]).String().
	opSliceGetString

	// opSliceSetString writes a string value to a slice/array element.
	// general[A].Index(ints[B]).SetString(strings[C]).
	opSliceSetString

	// opSliceGetBool reads a bool element from a slice/array.
	// bools[A] = general[B].Index(ints[C]).Bool().
	opSliceGetBool

	// opSliceSetBool writes a bool value to a slice/array element.
	// general[A].Index(ints[B]).SetBool(bools[C]).
	opSliceSetBool

	// opSliceGetUint reads a uint element from a slice/array.
	// uints[A] = general[B].Index(ints[C]).Uint().
	opSliceGetUint

	// opSliceSetUint writes a uint value to a slice/array element.
	// general[A].Index(ints[B]).SetUint(uints[C]).
	opSliceSetUint

	// opAppendInt appends ints[C] to []int in general[B], result in general[A].
	opAppendInt

	// opAppendString appends strings[C] to []string in
	// general[B], result in general[A].
	opAppendString

	// opAppendFloat appends floats[C] to []float64 in
	// general[B], result in general[A].
	opAppendFloat

	// opAppendBool appends bools[C] to []bool in general[B], result in general[A].
	opAppendBool

	// opMapGetIntInt reads ints[A] = map[int]int in general[B] with key ints[C].
	opMapGetIntInt

	// opMapSetIntInt writes general[A][ints[B]] = ints[C] for map[int]int.
	opMapSetIntInt

	// opGetField reads general[B].Field(C) into the target register. A
	// encodes the destination and the register kind comes from an
	// extension word.
	opGetField

	// opSetField writes source register to general[A].Field(B).
	opSetField

	// opSetFieldInt writes ints[C] to general[A].Field(B).
	opSetFieldInt

	// opGetFieldInt reads ints[A] = general[B].Field(C).Int().
	opGetFieldInt

	// opGetMethod reads general[B].MethodByName(name) into general[A].
	// The method name is stored as a string constant referenced by the
	// following opExt extension word.
	opGetMethod

	// opBindMethod creates a bound method value for an
	// interpreter-defined method, storing the result in general[A]
	// with receiver general[B].
	//
	// C encodes the embedded field traversal count; the function index
	// and field indices come from extension words.
	opBindMethod

	// opMakeMethodExpr creates an unbound method expression for an
	// interpreter-defined method, storing in general[A] a function
	// whose first parameter is the receiver.
	//
	// C encodes the embedded field traversal count; the function index
	// and field indices come from extension words.
	opMakeMethodExpr

	// opChanSend sends a value on general[A].
	opChanSend

	// opChanRecv receives from general[A] into target register.
	// ints[B] receives the ok flag.
	opChanRecv

	// opChanClose closes channel general[A].
	opChanClose

	// opSelect executes a select statement. Extension instructions
	// encode the cases.
	opSelect

	// opRangeInit initialises a range iterator for general[A].
	// Stores iterator state in general[B].
	opRangeInit

	// opRangeNext advances the iterator in general[A], writes key and
	// value to designated registers, and sets ints[B] to 0 when
	// iteration is complete.
	opRangeNext

	// opAddr takes the address: general[A] = &register[B].
	// C encodes the source register kind.
	opAddr

	// opDeref dereferences *general[B] into the target register. A
	// encodes the destination register index and C encodes its kind.
	opDeref

	// opAllocIndirect heap-escapes a variable via reflect.New of the
	// type at the opExt type table index. general[A] = the pointer;
	// B = source register index; C = source registerKind.
	opAllocIndirect

	// opTypeAssert performs general[A] = general[B].(T). ints[C]
	// receives the ok flag for the comma-ok form; the type T index
	// comes from an extension instruction.
	opTypeAssert

	// opConvert performs type conversion: general[A] = T(general[B]).
	opConvert

	// opStringToBytes converts strings[B] to general[A] as []byte.
	opStringToBytes

	// opBytesToString converts general[B] ([]byte) to strings[A].
	opBytesToString

	// opGetGlobal loads a package-level variable into the register at
	// index A in bank C. B is the global variable index.
	opGetGlobal

	// opSetGlobal stores the register at index A (bank C) into the
	// package-level variable at index B.
	opSetGlobal

	// opUnsafeString sets strings[A] = unsafe.String(general[B], ints[C]).
	opUnsafeString

	// opUnsafeStringData sets general[A] = unsafe.StringData(strings[B]).
	opUnsafeStringData

	// opUnsafeSlice sets general[A] = unsafe.Slice(general[B], ints[C]).
	opUnsafeSlice

	// opUnsafeSliceData sets general[A] = unsafe.SliceData(general[B]).
	opUnsafeSliceData

	// opUnsafeAdd sets general[A] = unsafe.Add(general[B], ints[C]).
	opUnsafeAdd

	// opStrContainsRune sets bools[A] =
	// strings.ContainsRune(strings[B], rune(ints[C])).
	opStrContainsRune

	// opStrContains sets bools[A] = strings.Contains(strings[B], strings[C]).
	opStrContains

	// opStrHasPrefix sets bools[A] = strings.HasPrefix(strings[B], strings[C]).
	opStrHasPrefix

	// opStrHasSuffix sets bools[A] = strings.HasSuffix(strings[B], strings[C]).
	opStrHasSuffix

	// opStrEqualFold sets bools[A] = strings.EqualFold(strings[B], strings[C]).
	opStrEqualFold

	// opStrIndex sets ints[A] = strings.Index(strings[B], strings[C]).
	opStrIndex

	// opStrCount sets ints[A] = strings.Count(strings[B], strings[C]).
	opStrCount

	// opStrToUpper sets strings[A] = strings.ToUpper(strings[B]).
	opStrToUpper

	// opStrToLower sets strings[A] = strings.ToLower(strings[B]).
	opStrToLower

	// opStrTrimSpace sets strings[A] = strings.TrimSpace(strings[B]).
	opStrTrimSpace

	// opStrTrimPrefix sets strings[A] = strings.TrimPrefix(strings[B], strings[C]).
	opStrTrimPrefix

	// opStrTrimSuffix sets strings[A] = strings.TrimSuffix(strings[B], strings[C]).
	opStrTrimSuffix

	// opStrTrim sets strings[A] = strings.Trim(strings[B], strings[C]).
	opStrTrim

	// opStrIndexRune sets ints[A] = strings.IndexRune(strings[B], rune(ints[C])).
	opStrIndexRune

	// opStrRepeat sets strings[A] = strings.Repeat(strings[B], int(ints[C])).
	opStrRepeat

	// opStrLastIndex sets ints[A] = strings.LastIndex(strings[B], strings[C]).
	opStrLastIndex

	// opStrJoin sets strings[A] = strings.Join(general[B], strings[C]).
	opStrJoin

	// opStrSplit sets general[A] = strings.Split(strings[B], strings[C]).
	opStrSplit

	// opStrReplaceAll sets strings[A] = strings.ReplaceAll(strings[B],
	// strings[C], strings[extensionWord.a]). The replacement string register
	// index is in the following opExt.
	opStrReplaceAll

	// opMathPow sets floats[A] = math.Pow(floats[B], floats[C]).
	opMathPow

	// opMathExp sets floats[A] = math.Exp(floats[B]).
	opMathExp

	// opMathSin sets floats[A] = math.Sin(floats[B]).
	opMathSin

	// opMathCos sets floats[A] = math.Cos(floats[B]).
	opMathCos

	// opMathTan sets floats[A] = math.Tan(floats[B]).
	opMathTan

	// opMathMod sets floats[A] = math.Mod(floats[B], floats[C]).
	opMathMod

	// opStrconvItoa sets strings[A] = strconv.Itoa(int(ints[B])).
	opStrconvItoa

	// opStrconvFormatBool sets strings[A] = strconv.FormatBool(bools[B]).
	opStrconvFormatBool

	// opStrconvFormatInt sets strings[A] = strconv.FormatInt(ints[B], int(ints[C])).
	opStrconvFormatInt

	// opSetZero zeroes the struct/composite value in general[A].
	// Used by the assign-through optimisation to clear all fields
	// before setting individual fields in-place.
	opSetZero

	// opGetGlobalWide loads a package-level variable into the register
	// at index A in bank C. The global variable index is a 16-bit
	// value read from the following extension word (A|(B<<8)).
	opGetGlobalWide

	// opSetGlobalWide stores the register at index A (bank C) into the
	// package-level variable. The global variable index is a 16-bit
	// value read from the following extension word (A|(B<<8)).
	opSetGlobalWide

	// opSpill stores the value in register A (bank indicated by B)
	// into the spill area.
	//
	// The spill slot index is a 24-bit value read from the following
	// extension word (A|(B<<8)|(C<<16)).
	// Runtime: registers[bank][spillAreaOffset+spillSlotIndex] =
	// registers[bank][A].
	opSpill

	// opReload loads a value from the spill area into register A
	// (bank indicated by B).
	//
	// The spill slot index is a 24-bit value read from the following
	// extension word (A|(B<<8)|(C<<16)).
	// Runtime: registers[bank][A] =
	// registers[bank][spillAreaOffset+spillSlotIndex].
	opReload

	// opCount is a sentinel marking the number of opcodes. It is not
	// a valid opcode.
	opCount
)

// opcodeNames maps each opcode to its string representation for debugging.
var opcodeNames = [opCount]string{
	opNop:                    "NOP",
	opExt:                    "EXT",
	opMoveInt:                "MOVE_INT",
	opMoveFloat:              "MOVE_FLOAT",
	opLoadIntConst:           "LOAD_INT_CONST",
	opLoadFloatConst:         "LOAD_FLOAT_CONST",
	opLoadBool:               "LOAD_BOOL",
	opLoadIntConstSmall:      "LOAD_INT_CONST_SMALL",
	opAddInt:                 "ADD_INT",
	opSubInt:                 "SUB_INT",
	opMulInt:                 "MUL_INT",
	opDivInt:                 "DIV_INT",
	opRemInt:                 "REM_INT",
	opNegInt:                 "NEG_INT",
	opIncInt:                 "INC_INT",
	opDecInt:                 "DEC_INT",
	opBitAnd:                 "BIT_AND",
	opBitOr:                  "BIT_OR",
	opBitXor:                 "BIT_XOR",
	opBitAndNot:              "BIT_AND_NOT",
	opBitNot:                 "BIT_NOT",
	opShiftLeft:              "SHIFT_LEFT",
	opShiftRight:             "SHIFT_RIGHT",
	opAddFloat:               "ADD_FLOAT",
	opSubFloat:               "SUB_FLOAT",
	opMulFloat:               "MUL_FLOAT",
	opDivFloat:               "DIV_FLOAT",
	opNegFloat:               "NEG_FLOAT",
	opEqInt:                  "EQ_INT",
	opNeInt:                  "NE_INT",
	opLtInt:                  "LT_INT",
	opLeInt:                  "LE_INT",
	opGtInt:                  "GT_INT",
	opGeInt:                  "GE_INT",
	opEqFloat:                "EQ_FLOAT",
	opNeFloat:                "NE_FLOAT",
	opLtFloat:                "LT_FLOAT",
	opLeFloat:                "LE_FLOAT",
	opGtFloat:                "GT_FLOAT",
	opGeFloat:                "GE_FLOAT",
	opIntToFloat:             "INT_TO_FLOAT",
	opFloatToInt:             "FLOAT_TO_INT",
	opNot:                    "NOT",
	opJump:                   "JUMP",
	opJumpIfTrue:             "JUMP_IF_TRUE",
	opJumpIfFalse:            "JUMP_IF_FALSE",
	opCall:                   "CALL",
	opReturn:                 "RETURN",
	opReturnVoid:             "RETURN_VOID",
	opTailCall:               "TAIL_CALL",
	opSubIntConst:            "SUB_INT_CONST",
	opAddIntConst:            "ADD_INT_CONST",
	opLeIntConstJumpFalse:    "LE_INT_CONST_JUMP_FALSE",
	opLtIntConstJumpFalse:    "LT_INT_CONST_JUMP_FALSE",
	opEqIntConstJumpFalse:    "EQ_INT_CONST_JUMP_FALSE",
	opEqIntConstJumpTrue:     "EQ_INT_CONST_JUMP_TRUE",
	opGeIntConstJumpFalse:    "GE_INT_CONST_JUMP_FALSE",
	opGtIntConstJumpFalse:    "GT_INT_CONST_JUMP_FALSE",
	opMulIntConst:            "MUL_INT_CONST",
	opAddIntJump:             "ADD_INT_JUMP",
	opIncIntJumpLt:           "INC_INT_JUMP_LT",
	opMathSqrt:               "MATH_SQRT",
	opMathAbs:                "MATH_ABS",
	opMathFloor:              "MATH_FLOOR",
	opMathCeil:               "MATH_CEIL",
	opMathTrunc:              "MATH_TRUNC",
	opMathRound:              "MATH_ROUND",
	opMoveString:             "MOVE_STRING",
	opMoveGeneral:            "MOVE_GENERAL",
	opMoveBool:               "MOVE_BOOL",
	opMoveUint:               "MOVE_UINT",
	opMoveComplex:            "MOVE_COMPLEX",
	opLoadStringConst:        "LOAD_STRING_CONST",
	opLoadGeneralConst:       "LOAD_GENERAL_CONST",
	opLoadBoolConst:          "LOAD_BOOL_CONST",
	opLoadUintConst:          "LOAD_UINT_CONST",
	opLoadComplexConst:       "LOAD_COMPLEX_CONST",
	opLoadNil:                "LOAD_NIL",
	opLoadZero:               "LOAD_ZERO",
	opAddUint:                "ADD_UINT",
	opSubUint:                "SUB_UINT",
	opMulUint:                "MUL_UINT",
	opDivUint:                "DIV_UINT",
	opRemUint:                "REM_UINT",
	opBitAndUint:             "BIT_AND_UINT",
	opBitOrUint:              "BIT_OR_UINT",
	opBitXorUint:             "BIT_XOR_UINT",
	opBitAndNotUint:          "BIT_AND_NOT_UINT",
	opBitNotUint:             "BIT_NOT_UINT",
	opShiftLeftUint:          "SHIFT_LEFT_UINT",
	opShiftRightUint:         "SHIFT_RIGHT_UINT",
	opEqUint:                 "EQ_UINT",
	opNeUint:                 "NE_UINT",
	opLtUint:                 "LT_UINT",
	opLeUint:                 "LE_UINT",
	opGtUint:                 "GT_UINT",
	opGeUint:                 "GE_UINT",
	opIncUint:                "INC_UINT",
	opDecUint:                "DEC_UINT",
	opAddComplex:             "ADD_COMPLEX",
	opSubComplex:             "SUB_COMPLEX",
	opMulComplex:             "MUL_COMPLEX",
	opDivComplex:             "DIV_COMPLEX",
	opNegComplex:             "NEG_COMPLEX",
	opEqComplex:              "EQ_COMPLEX",
	opNeComplex:              "NE_COMPLEX",
	opRealComplex:            "REAL_COMPLEX",
	opImagComplex:            "IMAG_COMPLEX",
	opBuildComplex:           "BUILD_COMPLEX",
	opConcatString:           "CONCAT_STRING",
	opLenString:              "LEN_STRING",
	opStringIndex:            "STRING_INDEX",
	opRuneToString:           "RUNE_TO_STRING",
	opSliceString:            "SLICE_STRING",
	opConcatRuneString:       "CONCAT_RUNE_STRING",
	opEqString:               "EQ_STRING",
	opNeString:               "NE_STRING",
	opLtString:               "LT_STRING",
	opLeString:               "LE_STRING",
	opGtString:               "GT_STRING",
	opGeString:               "GE_STRING",
	opEqGeneral:              "EQ_GENERAL",
	opNeGeneral:              "NE_GENERAL",
	opLtGeneral:              "LT_GENERAL",
	opLeGeneral:              "LE_GENERAL",
	opGtGeneral:              "GT_GENERAL",
	opGeGeneral:              "GE_GENERAL",
	opAdd:                    "ADD",
	opSub:                    "SUB",
	opMul:                    "MUL",
	opDiv:                    "DIV",
	opRem:                    "REM",
	opBoolToInt:              "BOOL_TO_INT",
	opIntToBool:              "INT_TO_BOOL",
	opIntToUint:              "INT_TO_UINT",
	opUintToInt:              "UINT_TO_INT",
	opUintToFloat:            "UINT_TO_FLOAT",
	opFloatToUint:            "FLOAT_TO_UINT",
	opMoveIntToGeneral:       "MOVE_INT_TO_GENERAL",
	opMoveGeneralToInt:       "MOVE_GENERAL_TO_INT",
	opMoveFloatToGeneral:     "MOVE_FLOAT_TO_GENERAL",
	opMoveGeneralToFloat:     "MOVE_GENERAL_TO_FLOAT",
	opMoveStringToGeneral:    "MOVE_STRING_TO_GENERAL",
	opMoveGeneralToString:    "MOVE_GENERAL_TO_STRING",
	opPackInterface:          "PACK_INTERFACE",
	opUnpackInterface:        "UNPACK_INTERFACE",
	opTestNilJumpTrue:        "TEST_NIL_JUMP_TRUE",
	opTestNilJumpFalse:       "TEST_NIL_JUMP_FALSE",
	opEqStringConstJumpFalse: "EQ_STRING_CONST_JUMP_FALSE",
	opCallNative:             "CALL_NATIVE",
	opCallBuiltin:            "CALL_BUILTIN",
	opCallMethod:             "CALL_METHOD",
	opCallIIFE:               "CALL_IIFE",
	opMakeClosure:            "MAKE_CLOSURE",
	opGetUpvalue:             "GET_UPVALUE",
	opSetUpvalue:             "SET_UPVALUE",
	opSyncClosureUpvalues:    "SYNC_CLOSURE_UPVALUES",
	opResetSharedCell:        "RESET_SHARED_CELL",
	opWriteSharedCell:        "WRITE_SHARED_CELL",
	opDefer:                  "DEFER",
	opPanic:                  "PANIC",
	opRecover:                "RECOVER",
	opGo:                     "GO",
	opMakeSlice:              "MAKE_SLICE",
	opMakeMap:                "MAKE_MAP",
	opMakeChan:               "MAKE_CHAN",
	opIndex:                  "INDEX",
	opIndexSet:               "INDEX_SET",
	opSliceOp:                "SLICE_OP",
	opMapIndex:               "MAP_INDEX",
	opMapSet:                 "MAP_SET",
	opMapIndexOk:             "MAP_INDEX_OK",
	opMapDelete:              "MAP_DELETE",
	opAppend:                 "APPEND",
	opCopy:                   "COPY",
	opLen:                    "LEN",
	opCap:                    "CAP",
	opSliceGetInt:            "SLICE_GET_INT",
	opSliceSetInt:            "SLICE_SET_INT",
	opSliceGetFloat:          "SLICE_GET_FLOAT",
	opSliceSetFloat:          "SLICE_SET_FLOAT",
	opSliceGetString:         "SLICE_GET_STRING",
	opSliceSetString:         "SLICE_SET_STRING",
	opSliceGetBool:           "SLICE_GET_BOOL",
	opSliceSetBool:           "SLICE_SET_BOOL",
	opSliceGetUint:           "SLICE_GET_UINT",
	opSliceSetUint:           "SLICE_SET_UINT",
	opAppendInt:              "APPEND_INT",
	opAppendString:           "APPEND_STRING",
	opAppendFloat:            "APPEND_FLOAT",
	opAppendBool:             "APPEND_BOOL",
	opMapGetIntInt:           "MAP_GET_INT_INT",
	opMapSetIntInt:           "MAP_SET_INT_INT",
	opGetField:               "GET_FIELD",
	opSetField:               "SET_FIELD",
	opSetFieldInt:            "SET_FIELD_INT",
	opGetFieldInt:            "GET_FIELD_INT",
	opGetMethod:              "GET_METHOD",
	opBindMethod:             "BIND_METHOD",
	opMakeMethodExpr:         "MAKE_METHOD_EXPR",
	opChanSend:               "CHAN_SEND",
	opChanRecv:               "CHAN_RECV",
	opChanClose:              "CHAN_CLOSE",
	opSelect:                 "SELECT",
	opRangeInit:              "RANGE_INIT",
	opRangeNext:              "RANGE_NEXT",
	opAddr:                   "ADDR",
	opDeref:                  "DEREF",
	opAllocIndirect:          "ALLOC_INDIRECT",
	opTypeAssert:             "TYPE_ASSERT",
	opConvert:                "CONVERT",
	opStringToBytes:          "STRING_TO_BYTES",
	opBytesToString:          "BYTES_TO_STRING",
	opGetGlobal:              "GET_GLOBAL",
	opSetGlobal:              "SET_GLOBAL",
	opUnsafeString:           "UNSAFE_STRING",
	opUnsafeStringData:       "UNSAFE_STRING_DATA",
	opUnsafeSlice:            "UNSAFE_SLICE",
	opUnsafeSliceData:        "UNSAFE_SLICE_DATA",
	opUnsafeAdd:              "UNSAFE_ADD",
	opStrContainsRune:        "STR_CONTAINS_RUNE",
	opStrContains:            "STR_CONTAINS",
	opStrHasPrefix:           "STR_HAS_PREFIX",
	opStrHasSuffix:           "STR_HAS_SUFFIX",
	opStrEqualFold:           "STR_EQUAL_FOLD",
	opStrIndex:               "STR_INDEX",
	opStrCount:               "STR_COUNT",
	opStrToUpper:             "STR_TO_UPPER",
	opStrToLower:             "STR_TO_LOWER",
	opStrTrimSpace:           "STR_TRIM_SPACE",
	opStrTrimPrefix:          "STR_TRIM_PREFIX",
	opStrTrimSuffix:          "STR_TRIM_SUFFIX",
	opStrTrim:                "STR_TRIM",
	opStrIndexRune:           "STR_INDEX_RUNE",
	opStrRepeat:              "STR_REPEAT",
	opStrLastIndex:           "STR_LAST_INDEX",
	opStrJoin:                "STR_JOIN",
	opStrSplit:               "STR_SPLIT",
	opStrReplaceAll:          "STR_REPLACE_ALL",
	opMathPow:                "MATH_POW",
	opMathExp:                "MATH_EXP",
	opMathSin:                "MATH_SIN",
	opMathCos:                "MATH_COS",
	opMathTan:                "MATH_TAN",
	opMathMod:                "MATH_MOD",
	opSpill:                  "SPILL",
	opReload:                 "RELOAD",
	opStrconvItoa:            "STRCONV_ITOA",
	opStrconvFormatBool:      "STRCONV_FORMAT_BOOL",
	opStrconvFormatInt:       "STRCONV_FORMAT_INT",
	opSetZero:                "SET_ZERO",
	opGetGlobalWide:          "GET_GLOBAL_WIDE",
	opSetGlobalWide:          "SET_GLOBAL_WIDE",
	opStringIndexToInt:       "STRING_INDEX_TO_INT",
	opLenStringLtJumpFalse:   "LEN_STRING_LT_JUMP_FALSE",
}

// String returns the human-readable name of the opcode.
//
// Returns the mnemonic string, or "UNKNOWN" for unregistered opcodes.
func (op opcode) String() string {
	if int(op) < len(opcodeNames) && opcodeNames[op] != "" {
		return opcodeNames[op]
	}
	return "UNKNOWN"
}
