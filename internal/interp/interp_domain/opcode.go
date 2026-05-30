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
	// opDrillTier1 is the main-tier drill marker.
	//
	// An instruction is 4 bytes: {op, A, B, C}. The op byte has only 256 distinct slots,
	// which is not enough for the full opcode set, so piko packs opcodes into four tiers
	// determined by operand arity: tier 0 holds 3-operand ops (use A, B, C), tier 1 holds
	// 2-operand ops (use B, C; A is the sub-op id), tier 2 holds 1-operand ops (use C; A and
	// B are sub-op ids), and tier 3 holds 0-operand ops. The compiler prefers higher tiers
	// when an opcode can fit so the capped tier-0 slot budget is reserved for opcodes that
	// genuinely need three operands. When the op byte is 0 the dispatcher descends into the
	// tier-1 sub-opcode space using operand A as the tier-1 discriminator; reserving iota=0
	// enforces the "byte == 0 means descend a tier" convention shared by every tier. The ASM
	// dispatch uses a single flat 1024-entry jump table indexed by (tier << 8) | sub_op, so
	// the drill cascade is purely a decoding convention, not a runtime cascade of indirect
	// jumps.
	opDrillTier1 opcode = iota

	// opExt is an extension word carrying a 24-bit payload formed from A|(B<<8)|(C<<16).
	// Used for wide jump offsets, large constant indices, and multi-word encodings.
	opExt

	// opLoadIntConst loads intConstants[B|(C<<8)] into ints[A].
	opLoadIntConst

	// opLoadFloatConst loads floatConstants[B|(C<<8)] into floats[A].
	opLoadFloatConst

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

	// opBitAnd sets ints[A] = ints[B] & ints[C].
	opBitAnd

	// opBitOr sets ints[A] = ints[B] | ints[C].
	opBitOr

	// opBitXor sets ints[A] = ints[B] ^ ints[C].
	opBitXor

	// opBitAndNot sets ints[A] = ints[B] &^ ints[C].
	opBitAndNot

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

	// opJumpIfTrue jumps if ints[A] != 0. Offset in B|(C<<8).
	opJumpIfTrue

	// opJumpIfFalse jumps if ints[A] == 0. Offset in B|(C<<8).
	opJumpIfFalse

	// opCall calls a compiled function. general[A] holds the callee, B is the argument
	// count, and C encodes return destination info.
	opCall

	// opTailCall performs an optimised tail call that reuses the current frame instead of
	// pushing a new one.
	opTailCall

	// opSubIntConst sets ints[A] = ints[B] - intConstants[C]. Fuses opLoadIntConst +
	// opSubInt when the constant index fits in 8 bits.
	opSubIntConst

	// opAddIntConst sets ints[A] = ints[B] + intConstants[C]. Fuses opLoadIntConst +
	// opAddInt when the constant index fits in 8 bits.
	opAddIntConst

	// opLeIntConstJumpFalse compares ints[A] <= intConstants[B] and jumps by offset in the
	// following opExt if false (ints[A] > constant). Fuses opLoadIntConst + opLeInt +
	// opJumpIfFalse.
	opLeIntConstJumpFalse

	// opLtIntConstJumpFalse compares ints[A] < intConstants[B] and jumps by offset in the
	// following opExt if false (ints[A] >= constant). Fuses opLoadIntConst + opLtInt +
	// opJumpIfFalse.
	opLtIntConstJumpFalse

	// opEqIntConstJumpFalse compares ints[A] == intConstants[B] and jumps by offset in the
	// following opExt if false (not equal). Fuses opLoadIntConst + opEqInt + opJumpIfFalse.
	opEqIntConstJumpFalse

	// opEqIntConstJumpTrue compares ints[A] == intConstants[B] and jumps by offset in the
	// following opExt if true (equal). Fuses opLoadIntConst + opEqInt + opJumpIfTrue.
	opEqIntConstJumpTrue

	// opGeIntConstJumpFalse compares ints[A] >= intConstants[B] and jumps by offset in the
	// following opExt if false (less than). Fuses opLoadIntConst + opGeInt + opJumpIfFalse.
	opGeIntConstJumpFalse

	// opGtIntConstJumpFalse compares ints[A] > intConstants[B] and jumps by offset in the
	// following opExt if false (less or equal). Fuses opLoadIntConst + opGtInt +
	// opJumpIfFalse.
	opGtIntConstJumpFalse

	// opMulIntConst sets ints[A] = ints[B] * intConstants[C]. Fuses opLoadIntConst +
	// opMulInt when the constant index fits in 8 bits.
	opMulIntConst

	// opAddIntJump sets ints[A] = ints[B] + intConstants[C] and unconditionally jumps by
	// offset in the following opExt. Fuses opAddIntConst + (opDrillTier1, subOpJump).
	opAddIntJump

	// opStringIndex sets uints[A] = uint64(strings[B][ints[C]]). Panics with
	// errIndexOutOfRange if index is out of bounds.
	opStringIndex

	// opEqString sets ints[A] = (strings[B] == strings[C]) ? 1 : 0.
	opEqString

	// opNeString sets ints[A] = (strings[B] != strings[C]) ? 1 : 0.
	opNeString

	// opSliceString sets strings[A] = strings[B][low:high]. C encodes flags (bit 0 = low
	// present, bit 1 = high present) and an opExt follows with A=lowRegister,
	// B=highRegister.
	opSliceString

	// opStringIndexToInt sets ints[A] = int64(strings[B][ints[C]]). Fuses opStringIndex +
	// opUintToInt to avoid the intermediate uint register and one tier-2 trampoline in
	// string loops.
	opStringIndexToInt

	// opMoveGeneral copies general[B] to general[A].
	//
	// The instruction's C operand encodes a snapshot mode determined at compile time:
	//
	// 	moveGeneralModeDynamic (default, 0): runtime kind switch via
	// 	valueCopyForBoundary. Used when the source operand's static
	// 	type is interface, type parameter, or unknown.
	//
	// 	moveGeneralModeAlias (1): direct register copy without the
	// 	helper. The source's static type is alias-safe (pointer,
	// 	slice, map, chan, signature, basic) so the reflect.Value
	// 	header copy already matches Go's reference semantics.
	//
	// 	moveGeneralModeSnapshot (2): unconditional snapshot via
	// 	valueCopyForBoundary. The source's static type is struct or
	// 	array; the runtime kind switch is therefore elided.
	opMoveGeneral

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

	// opAddComplex sets complex[A] = complex[B] + complex[C].
	opAddComplex

	// opSubComplex sets complex[A] = complex[B] - complex[C].
	opSubComplex

	// opMulComplex sets complex[A] = complex[B] * complex[C].
	opMulComplex

	// opDivComplex sets complex[A] = complex[B] / complex[C].
	opDivComplex

	// opEqComplex sets ints[A] = (complex[B] == complex[C]) ? 1 : 0.
	opEqComplex

	// opNeComplex sets ints[A] = (complex[B] != complex[C]) ? 1 : 0.
	opNeComplex

	// opBuildComplex sets complex[A] = complex(floats[B], floats[C]).
	opBuildComplex

	// opConcatString sets strings[A] = strings[B] + strings[C].
	opConcatString

	// opConcatRuneString sets strings[A] = strings[B] + string(rune(ints[C])). Fuses
	// opRuneToString + opConcatString with in-place arena extension.
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

	// opNeGeneral sets ints[A] = (general[B] != general[C]) ? 1 : 0 via reflect, using the
	// same equality logic as opEqGeneral.
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

	// opTruncateNarrow truncates a narrow integer register in place.
	//
	// Operand B encodes the bit width (8, 16, or 32). Operand C is 0 for the uint bank (mask
	// only) and 1 for the int bank (mask then sign-extend). The compiler emits this op after
	// arithmetic or conversion that produces a narrow uint or signed int result so the
	// stored value matches Go's modular wrap semantics. Width 64 is a no-op so the compiler
	// does not emit it.
	opTruncateNarrow

	// opPackInterface wraps a typed value into an interface.
	opPackInterface

	// opUnpackInterface extracts the concrete value from an interface.
	opUnpackInterface

	// opTestNilJumpTrue tests if general[A] is nil/invalid and jumps by signed offset
	// B|(C<<8) if true.
	opTestNilJumpTrue

	// opTestNilJumpFalse tests if general[A] is nil/invalid and jumps by signed offset
	// B|(C<<8) if false (i.e. not nil).
	opTestNilJumpFalse

	// opEqStringConstJumpFalse compares strings[A] == stringConstants[B] and jumps by offset
	// in the following opExt if false (not equal). Fuses opLoadStringConst + opEqString +
	// opJumpIfFalse.
	opEqStringConstJumpFalse

	// opCallNative calls a native reflect.Value function in general[A] with B arguments.
	opCallNative

	// opCallBuiltin calls a built-in function identified by A. B and C are builtin-specific
	// operands.
	opCallBuiltin

	// opCallMethod dispatches a method call using the runtime method table with call site
	// index B|C<<8, where the receiver is the first argument.
	//
	// The method name string constant index is encoded in an extension word.
	opCallMethod

	// opCallIIFE calls an immediately invoked function expression with call site index
	// B|(C<<8).
	//
	// It fuses opMakeClosure + opCall, skipping runtimeClosure allocation and
	// reflect.ValueOf boxing.
	opCallIIFE

	// opMakeClosure creates a closure in general[A] from function index B|(C<<8). Upvalue
	// descriptors are in the CompiledFunction.
	opMakeClosure

	// opGetUpvalue loads upvalue[B] into the register at A. C encodes the register kind.
	opGetUpvalue

	// opSetUpvalue stores the register at A into upvalue[B]. C encodes the register kind.
	opSetUpvalue

	// opSyncClosureUpvalues reads upvalue cells from the closure in general[A] back to
	// parent registers. Used after an immediately invoked function expression to propagate
	// captured mutations.
	opSyncClosureUpvalues

	// opResetSharedCell invalidates the shared upvalue cell for the register at index A in
	// bank B, forcing the next opMakeClosure to create a fresh cell. Used for Go 1.22+
	// per-iteration loop variable scoping.
	opResetSharedCell

	// opWriteSharedCell copies the current register value (bank B, index A) into the
	// corresponding shared upvalue cell if one exists, keeping the cell in sync after a
	// parent-frame write to a variable that has already been captured by a closure (e.g.
	// `var f func(); f = func() { f() }`).
	opWriteSharedCell

	// opDefer pushes a deferred call. general[A] is the function, B is the number of
	// arguments.
	opDefer

	// opGo spawns a goroutine calling general[A] with B arguments.
	opGo

	// opMakeSlice creates a slice: general[A] = make([]T, ints[B], ints[C]). The type T is
	// looked up from the function's type table via an extension instruction.
	opMakeSlice

	// opIndex reads element: target = general[A][ints[B]]. C encodes the destination
	// register kind and index via extension.
	opIndex

	// opIndexSet writes element: general[A][ints[B]] = source.
	opIndexSet

	// opSliceOp slices: general[A] = general[B][low:high:max]. Indices come from int
	// registers via extension instructions.
	opSliceOp

	// opMapIndex reads map element: target = general[A][key].
	opMapIndex

	// opMapSet writes map element: general[A][key] = value.
	opMapSet

	// opMapIndexOk reads map element with ok flag: general[A] = general[B][general[C]],
	// ints[extensionWord.a] = ok (0 or 1).
	opMapIndexOk

	// opAppend appends: general[A] = append(general[B], arguments...).
	opAppend

	// opAppendSpread sets general[A] = append(general[B], general[C]...).
	//
	// Appends a slice via variadic spread; general[C] must be assignable to general[B]'s
	// element type as a slice, and the handler calls reflect.AppendSlice. Emitted for source
	// expressions of the form `append(slice, other...)`.
	opAppendSpread

	// opAppendByteFast is the byte-builder append fast path.
	//
	// Specialised tier-0 form of the `*p = append(*p, b)` pattern that dominates expr_eval.
	// Encoding:
	//
	// 	A = destination general register (resulting []byte)
	// 	B = source slice general register (input []byte)
	// 	C = byte-value uint register (single byte to append)
	//
	// Handler does direct arena byte-append with no reflect.TypeAssert cascade, no
	// checkAppendLimit, no IsValid probe - the compiler only emits this opcode when the
	// element type is statically byte and the source slice is statically []byte. Routed to a
	// per-op direct exit so the dispatch overhead skips the generic processExitTier2
	// indirection.
	opAppendByteFast

	// opCopy copies: ints[A] = copy(general[B], general[C]).
	opCopy

	// opSliceGetInt reads an integer element from a slice/array without reflect boxing.
	// ints[A] = general[B].Index(ints[C]).Int() (or .Uint() for unsigned element types).
	opSliceGetInt

	// opSliceSetInt writes an integer value to a slice/array element without reflect boxing.
	// general[A].Index(ints[B]).SetInt(ints[C]) (or .SetUint() for unsigned element types).
	opSliceSetInt

	// opSliceGetFloat reads a float element from a slice/array without reflect boxing.
	// floats[A] = general[B].Index(ints[C]).Float().
	opSliceGetFloat

	// opSliceSetFloat writes a float value to a slice/array element without reflect boxing.
	// general[A].Index(ints[B]).SetFloat(floats[C]).
	opSliceSetFloat

	// opSliceGetString reads a string element from a slice/array. strings[A] =
	// general[B].Index(ints[C]).String().
	opSliceGetString

	// opSliceSetString writes a string value to a slice/array element.
	// general[A].Index(ints[B]).SetString(strings[C]).
	opSliceSetString

	// opSliceGetBool reads a bool element from a slice/array. bools[A] =
	// general[B].Index(ints[C]).Bool().
	opSliceGetBool

	// opSliceSetBool writes a bool value to a slice/array element.
	// general[A].Index(ints[B]).SetBool(bools[C]).
	opSliceSetBool

	// opSliceGetUint reads a uint element from a slice/array. uints[A] =
	// general[B].Index(ints[C]).Uint().
	opSliceGetUint

	// opSliceSetUint writes a uint value to a slice/array element.
	// general[A].Index(ints[B]).SetUint(uints[C]).
	opSliceSetUint

	_

	_

	_

	_

	// opMapGetIntInt reads ints[A] = map[int]int in general[B] with key ints[C].
	opMapGetIntInt

	// opMapSetIntInt writes general[A][ints[B]] = ints[C] for map[int]int.
	opMapSetIntInt

	// opMapGetStringInt reads ints[A] = map[string]int in general[B] with key strings[C].
	// Bypasses reflect.MapIndex via the cached typed map handle on vm.typedHandleCache.
	opMapGetStringInt

	// opMapSetStringInt writes general[A][strings[B]] = ints[C] for map[string]int. Bypasses
	// reflect.SetMapIndex via the cached typed map handle.
	opMapSetStringInt

	// opMapGetStringString reads strings[A] = map[string]string in general[B] with key
	// strings[C].
	opMapGetStringString

	// opMapSetStringString writes general[A][strings[B]] = strings[C] for map[string]string.
	opMapSetStringString

	// opMapGetIntString reads strings[A] = map[int]string in general[B] with key ints[C].
	opMapGetIntString

	// opMapSetIntString writes general[A][ints[B]] = strings[C] for map[int]string.
	opMapSetIntString

	// opMapIndexOkIntInt reads ints[A] = map[int]int in general[B] with key ints[C], plus an
	// extension word whose A field is the int register holding the ok flag.
	opMapIndexOkIntInt

	// opMapIndexOkStringInt reads ints[A] = map[string]int in general[B] with key
	// strings[C], plus an extension word whose A field is the int register holding the ok
	// flag.
	opMapIndexOkStringInt

	// opMapIndexOkStringString reads strings[A] = map[string]string in general[B] with key
	// strings[C], plus an extension word whose A field is the int register holding the ok
	// flag.
	opMapIndexOkStringString

	// opMapIndexOkIntString reads strings[A] = map[int]string in general[B] with key
	// ints[C], plus an extension word whose A field is the int register holding the ok flag.
	opMapIndexOkIntString

	// opMapGetIntGeneral reads general[A] = map[int]V via general[B][ints[C]].
	//
	// V is a pointer-like or general-bank type (e.g. *UserStruct, interface{}, []T,
	// map[K]V). Avoids boxing the int key into the general bank: the index path uses a
	// pooled key reflect.Value (cached on vm.mapKeyScratch keyed by reflect.Type) so the
	// per-call cost is one reflect.MapIndex with no extra allocation for the key. The result
	// is materialised into the general bank as a reflect.Value (the per-call result
	// allocation is unavoidable without unsafe map access).
	opMapGetIntGeneral

	// opMapIndexOkIntGeneral is the comma-ok form of opMapGetIntGeneral.
	//
	// Reads general[A] = map[int]V (with V in the general bank) plus an extension word whose
	// A field is the int register holding the ok flag. Same semantics as opMapGetIntGeneral;
	// ok=0 when the key is absent and the destination is set to the map's zero value
	// (typically a nil reflect.Value or zero pointer).
	opMapIndexOkIntGeneral

	// opMapGetStringGeneral reads general[A] = map[string]V in general[B] with key
	// strings[C], where V is any value type whose register kind is registerGeneral (slices,
	// maps, interfaces, pointers). Mirrors opMapGetIntGeneral for string keys: routes
	// through runtime.mapaccess2_faststr via the mapAccessFastStrToGeneral helper,
	// eliminating the reflect.Value.MapIndex per-call boxing alloc.
	opMapGetStringGeneral

	// opMapIndexOkStringGeneral reads general[A] = map[string]V (with V in the general bank)
	// plus an extension word whose A field is the int register holding the ok flag. Same
	// semantics as opMapGetStringGeneral but with the comma-ok shape.
	opMapIndexOkStringGeneral

	// opMapSetStringGeneral writes general[A][strings[B]] = general[C] for map[string]V
	// where V is in the general bank. Routes through runtime.mapassign_faststr +
	// runtime.typedmemmove, mirroring the existing handleSetStructFieldGeneralT0
	// write-barrier pattern.
	opMapSetStringGeneral

	// opMapAddIntInt fuses get-add-set for map[int]int.
	//
	// Performs general[A][ints[B]] += ints[C] in a single dispatch with one map probe
	// instead of two. The destination is the map (general[A]); the key is the integer in
	// ints[B]; the delta to add is ints[C]. Absent keys are treated as 0 (Go map default),
	// matching the semantics of m[k] = m[k] + v.
	opMapAddIntInt

	// opMapAddStringInt fuses get-add-set for map[string]int.
	//
	// Performs general[A][strings[B]] += ints[C] in a single dispatch with one map probe.
	// Absent keys are treated as 0. Hashes the string key once via Go's runtime map fast
	// path.
	opMapAddStringInt

	// opGetField reads general[B].Field(C) into the target register. A encodes the
	// destination and the register kind comes from an extension word.
	opGetField

	// opSetField writes source register to general[A].Field(B).
	opSetField

	// opSetFieldInt writes ints[C] to general[A].Field(B).
	opSetFieldInt

	// opGetFieldInt reads ints[A] = general[B].Field(C).Int().
	opGetFieldInt

	// opBindMethod creates a bound method value for an interpreter-defined method, storing
	// the result in general[A] with receiver general[B].
	//
	// C encodes the embedded field traversal count; the function index and field indices
	// come from extension words.
	opBindMethod

	// opChannelSend sends a value on general[A].
	opChannelSend

	// opSelect executes a select statement. Extension instructions encode the cases.
	opSelect

	// opRangeInit initialises a range iterator for general[A]. Stores iterator state in
	// general[B].
	opRangeInit

	// opRangeNext advances the iterator in general[A], writes key and value to designated
	// registers, and sets ints[B] to 0 when iteration is complete.
	opRangeNext

	// opAddr takes the address: general[A] = &register[B]. C encodes the source register
	// kind.
	opAddr

	// opDeref dereferences *general[B] into the target register. A encodes the destination
	// register index and C encodes its kind.
	opDeref

	// opAllocIndirect heap-escapes a variable via reflect.New of the type at the opExt type
	// table index. general[A] = the pointer; B = source register index; C = source
	// registerKind.
	opAllocIndirect

	// opTypeAssert performs general[A] = general[B].(T). ints[C] receives the ok flag for
	// the comma-ok form; the type T index comes from an extension instruction.
	opTypeAssert

	// opConvert performs type conversion: general[A] = T(general[B]).
	opConvert

	// opGetGlobal loads a package-level variable into the register at index A in bank C. B
	// is the global variable index.
	opGetGlobal

	// opSetGlobal stores the register at index A (bank C) into the package-level variable at
	// index B.
	opSetGlobal

	// opUnsafeString sets strings[A] = unsafe.String(general[B], ints[C]).
	opUnsafeString

	// opUnsafeSlice sets general[A] = unsafe.Slice(general[B], ints[C]).
	opUnsafeSlice

	// opUnsafeAdd sets general[A] = unsafe.Add(general[B], ints[C]).
	opUnsafeAdd

	// opStrContainsRune sets bools[A] = strings.ContainsRune(strings[B], rune(ints[C])).
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

	// opStrReplaceAll sets strings[A] = strings.ReplaceAll(strings[B], strings[C],
	// strings[extensionWord.a]). The replacement string register index is in the following
	// opExt.
	opStrReplaceAll

	// opMathPow sets floats[A] = math.Pow(floats[B], floats[C]).
	opMathPow

	// opGetGlobalWide loads a package-level variable into the register at index A in bank C.
	// The global variable index is a 16-bit value read from the following extension word
	// (A|(B<<8)).
	opGetGlobalWide

	// opSetGlobalWide stores the register at index A (bank C) into the package-level
	// variable. The global variable index is a 16-bit value read from the following
	// extension word (A|(B<<8)).
	opSetGlobalWide

	// opSliceGetIntDirect sets ints[A] = slicesInt[B][ints[C]].
	//
	// Reads an int64 element from a slicesInt bank entry via direct Go indexing, no reflect,
	// no allocation. Bounds-checked via unsigned compare so negative indices route through
	// the panic path.
	opSliceGetIntDirect

	// opSliceSetIntDirect sets slicesInt[A][ints[B]] = ints[C].
	//
	// Writes an int64 value to a slicesInt bank entry via direct Go indexing.
	opSliceSetIntDirect

	// opRangeNextSliceInt advances a typed range over a slicesInt entry.
	//
	// Operand A is the index register (typed int), B is the source slicesInt register, C is
	// the destination value register (typed int).
	opRangeNextSliceInt

	// opGetStructFieldIntT0 is the tier-0 fast-path int struct-field read.
	//
	// Operand A=destination int register, B=source general register holding the struct,
	// C=layoutTable index (uint8). The handler resolves layoutIdx, computes base+offset,
	// performs a typed unsafe load matching the field's kind. This is the tier-0 variant of
	// subOpGetStructFieldInt; the compiler emits the tier-0 form when layoutIdx fits in
	// uint8 (the common case) and falls back to the tier-1 sub-op encoding when the
	// per-function layoutTable has > 256 entries.
	opGetStructFieldIntT0

	// opSetStructFieldIntT0 is the tier-0 fast-path int struct-field write. Operand
	// A=structReg general, B=value int register, C=layoutTable index (uint8).
	opSetStructFieldIntT0

	// opGetStructFieldUint is the tier-0 fast-path uint struct-field read. Same operand
	// layout as opGetStructFieldIntT0 but writes into the uints bank.
	opGetStructFieldUint

	// opSetStructFieldUint is the tier-0 fast-path uint struct-field write. Operand
	// A=structReg general, B=value uint register, C=layoutTable index (uint8).
	opSetStructFieldUint

	// opGetStructFieldFloat is the tier-0 fast-path float struct-field read. Same operand
	// layout as opGetStructFieldUint but writes into the floats bank.
	opGetStructFieldFloat

	// opSetStructFieldFloat is the tier-0 fast-path float struct-field write.
	opSetStructFieldFloat

	// opGetStructFieldBool is the tier-0 fast-path bool struct-field read.
	opGetStructFieldBool

	// opSetStructFieldBool is the tier-0 fast-path bool struct-field write.
	opSetStructFieldBool

	// opGetStructFieldGeneral is the tier-0 fast-path read for a pointer- or interface-typed
	// struct field. Operand A=destination general register, B=source general register (the
	// struct, pointer-to-struct, or interface holding a struct), C=layoutTable index
	// (uint8).
	//
	// Computes base+offset directly from the layout descriptor, reads the field via
	// reflect.NewAt (no allocation), and unwraps an interface leaf so the result is the
	// user's actual value (the concrete pointer or struct stored inside the cycle-broken
	// interface{} that piko substitutes for self-referential pointer fields).
	//
	// Used by self-referential and pointer-to-other-struct fields such as `node.next *node`,
	// `node.parent *Tree`, or `wrapper.Inner interface{...}`. Eliminates the reflect.Field
	// walk that handleGetField pays per access.
	opGetStructFieldGeneral

	// opSetStructFieldGeneral is the tier-0 fast-path write for a pointer- or
	// interface-typed struct field. Operand A=destination general register (the struct,
	// pointer-to-struct, or interface holding a struct), B=value general register,
	// C=layoutTable index (uint8).
	//
	// Computes base+offset directly, then uses reflect.NewAt to bind the field with the GC
	// write barrier intact (raw pointer stores would skip the barrier and risk silently
	// dropping references on the next GC cycle). The Set call coerces the value through
	// reflect's assignability rules, matching handleSetField semantics for interface-typed
	// leaves (the value is wrapped in interface{} automatically when the field type is
	// interface{}).
	opSetStructFieldGeneral

	// opCopyStructFieldGeneralT0 is the fused general-bank field copy.
	//
	// Encodes `dstRecv.dstField = srcRecv.srcField` for general-bank struct fields. Operand
	// A=srcRecv (general bank receiver register), B=dstRecv (general bank receiver
	// register), C=srcLayoutIdx (structLayoutTable index for the source field). The
	// following opExt word carries A=dstLayoutIdx (structLayoutTable index for the
	// destination field), B=0, C=0.
	//
	// Replaces the GET_STRUCT_FIELD_GENERAL_T0 + SET_STRUCT_FIELD_GENERAL_T0 pair the
	// peephole pass finds when the GET's destination is a compiler-emitted temp that is read
	// exactly once (by the SET) and written exactly once (by the GET). For the LRU's
	// linked-list pointer copies (e.g. `removed.previous.next = removed.next` after inlining
	// `lruCache.detach` into `lruCache.put`), this collapses two trampoline round trips into
	// one and skips the intermediate reflect.Value materialisation entirely.
	opCopyStructFieldGeneralT0

	// opSliceIndexStructFieldInt is the tier-0 fused `slice[i].field` read for int leaves.
	//
	// Targets `[]Struct` slices where the leaf field is an int-kind value. Operand
	// A=destination int register, B=source slice general register, C=index int register. The
	// following opExt word carries the structLayoutTable index. Replaces the two-op (opIndex
	// + opGetStructFieldIntT0) sequence with a single dispatch that:
	//  1. Reads the slice header from general[B] (UnsafePointer for data, Len for
	//     bounds-check).
	//  2. Bounds-checks ints[C] against Len (negative or >= Len exits to tier-2 for a proper
	//     out-of-range panic).
	//  3. Computes elementPtr = data + index * elementSize + fieldOffset, where elementSize
	//     comes from structLayoutTable[layoutIdx].TypeIndex (the struct type's reflect.Size)
	//     and fieldOffset is layout.Offset.
	//  4. Reads the leaf field as an int64 directly into ints[A].
	//
	// Eliminates the per-iteration reflect.Value allocation that opIndex pays for
	// slice-of-struct iteration (used by 07_dijkstra in `graph[u][i].target` / `.weight`, by
	// any `for i := range s` loop iterating a struct slice with field access).
	opSliceIndexStructFieldInt

	// opSliceIndexStructFieldUint mirrors opSliceIndexStructFieldInt for uint-kind leaf
	// fields. Destination is uint register A.
	opSliceIndexStructFieldUint

	// opSliceIndexStructFieldFloat mirrors opSliceIndexStructFieldInt for float-kind leaf
	// fields. Destination is float register A.
	opSliceIndexStructFieldFloat

	// opSliceIndexStructFieldBool mirrors opSliceIndexStructFieldInt for bool-kind leaf
	// fields. Destination is bool register A.
	opSliceIndexStructFieldBool

	// opSliceIndexStructFieldString mirrors opSliceIndexStructFieldInt for string-kind
	// leaves.
	//
	// Destination is string register A. The string header (16 bytes) is copied from the
	// field's storage into the strings bank without allocating a fresh Go string.
	opSliceIndexStructFieldString

	// opLtIntJumpFalse compares ints[A] < ints[B] and jumps by offset in the following opExt
	// if false (ints[A] >= ints[B]). Fuses opLtInt + opJumpIfFalse when neither operand is a
	// constant.
	opLtIntJumpFalse

	// opLeIntJumpFalse compares ints[A] <= ints[B] and jumps by offset in the following
	// opExt if false. Fuses opLeInt + opJumpIfFalse.
	opLeIntJumpFalse

	// opGtIntJumpFalse compares ints[A] > ints[B] and jumps by offset in the following opExt
	// if false. Fuses opGtInt + opJumpIfFalse.
	opGtIntJumpFalse

	// opGeIntJumpFalse compares ints[A] >= ints[B] and jumps by offset in the following
	// opExt if false. Fuses opGeInt + opJumpIfFalse.
	opGeIntJumpFalse

	// opEqIntJumpFalse compares ints[A] == ints[B] and jumps by offset in the following
	// opExt if false. Fuses opEqInt + opJumpIfFalse.
	opEqIntJumpFalse

	// opNeIntJumpFalse compares ints[A] != ints[B] and jumps by offset in the following
	// opExt if false. Fuses opNeInt + opJumpIfFalse.
	opNeIntJumpFalse

	// opEqInterfaceNil sets ints[A] = (!general[B].IsValid()) ? 1 : 0.
	//
	// Emitted by the compiler when comparing an interface-typed value against the nil
	// literal so a typed-nil pointer wrapped in an interface compares non-equal to nil,
	// matching Go's "interface holding typed nil != nil" semantics that reflect.DeepEqual
	// hides.
	opEqInterfaceNil

	// opNeInterfaceNil sets ints[A] = (general[B].IsValid()) ? 1 : 0.
	//
	// Mirror of opEqInterfaceNil for the != operator.
	opNeInterfaceNil

	// opMapIndexOkJumpIfFalseIntInt fuses opMapIndexOkIntInt + opJumpIfFalse.
	//
	// Reads ints[A] = map[int]int (or int64) in general[B] with key ints[C]; the extension
	// word carries the ok-register index in ext.a and the signed jump offset packed into
	// ext.b (lo) and ext.c (hi). On !ok, programCounter += jumpOffset (taken relative to the
	// slot AFTER the trailing opNop, matching the original opJumpIfFalse landing site).
	// Pre-fusion shape (3 slots): primary opMapIndexOkIntInt, ext (ok_reg in ext.a),
	// opJumpIfFalse{ok_reg, offLo, offHi}. Post-fusion (3 slots): primary
	// opMapIndexOkJumpIfFalseIntInt, ext (ok_reg, offLo, offHi), opNop.
	opMapIndexOkJumpIfFalseIntInt

	// opMapIndexOkJumpIfFalseStringInt mirrors opMapIndexOkJumpIfFalseIntInt for
	// map[string]int with string keys.
	opMapIndexOkJumpIfFalseStringInt

	// opMapIndexOkJumpIfFalseStringString mirrors opMapIndexOkJumpIfFalseIntInt for
	// map[string]string (string keys + string values).
	opMapIndexOkJumpIfFalseStringString

	// opMapIndexOkJumpIfFalseIntString mirrors opMapIndexOkJumpIfFalseIntInt for
	// map[int]string (int keys + string values).
	opMapIndexOkJumpIfFalseIntString

	// opMapIndexOkJumpIfFalseIntGeneral mirrors opMapIndexOkJumpIfFalseIntInt for map[int]V.
	//
	// V is in the general bank (e.g. *T pointers, slices, maps, interfaces). Primary hot
	// path for LRU-style `node, ok := m[k]; if !ok { ... }` lookups.
	opMapIndexOkJumpIfFalseIntGeneral

	// opMapIndexOkJumpIfFalseStringGeneral mirrors opMapIndexOkJumpIfFalseIntGeneral for
	// map[string]V with string keys.
	opMapIndexOkJumpIfFalseStringGeneral

	// opSwapStructFieldsGeneralT0 fuses the six-instruction `t.x, t.y = t.y, t.x` swap of
	// two general-bank fields of the same struct into a single dispatch. Operand A=general
	// structReg holding *Struct, B=layoutTable index for field X, C=layoutTable index for
	// field Y.
	//
	// Compiles from cross-paired tuple assignments where both LHS and both RHS selectors
	// share the same struct receiver and both fields have the same kind (both
	// Interface-wrapped pointer or both Pointer). Avoids the GET-MOVE-GET-MOVE-SET-SET
	// expansion the generic tuple-assign lowerer produces (786 K dispatches per iter on
	// invert_binary_tree shrink to 131 K).
	opSwapStructFieldsGeneralT0

	// opGetStructFieldRawPointerT0 specialises opGetStructFieldGeneral for the cycle-broken
	// interface{} case the compiler can prove always holds a pointer of a statically-known
	// type (the *Self to any substitution performed by convertFieldBreakingCycles).
	//
	// Operand A=destination general register, B=source general register (the struct holding
	// the cycle-broken interface{} field), C=layoutTable index (uint8). Same operand shape
	// as opGetStructFieldGeneral.
	//
	// The handler skips the runtime abi.Type Kind_ walk and the 5-way kind dispatch that the
	// generic handler performs to classify the eface's held value. It reads the held type
	// and pointer from the eface header directly and constructs the reflect.Value via
	// unsafePointerKindValue without any kind classification. The handler is safe because
	// the compiler only emits this opcode for fields whose held value is guaranteed to be a
	// pointer (the cycle-break substitution preserves the original *Self type's pointer kind
	// even though the static field type is interface{}).
	//
	// Falls back to opGetStructFieldGeneral when the compiler cannot prove the held type is
	// fixed.
	opGetStructFieldRawPointerT0

	// opSliceGetIntDirectUnchecked is the bounds-elided variant of opSliceGetIntDirect.
	//
	// Operand shape is identical: A=int dest, B=slicesInt source, C=int index. The handler
	// skips the `uint64(index) >= uint64(len(slice))` check and dereferences directly. The
	// compiler emits this opcode only when the access is provably in range (range-loop body,
	// post-conditional gate, or constant in-range index proved by the BCE pass).
	opSliceGetIntDirectUnchecked

	// opSliceSetIntDirectUnchecked is the bounds-elided variant of opSliceSetIntDirect.
	//
	// Operand shape: A=slicesInt dest, B=int index, C=int value. Same proof requirements as
	// the get variant.
	opSliceSetIntDirectUnchecked

	// opStringIndexUnchecked is the bounds-elided variant of opStringIndex.
	//
	// Operand shape: A=uint dest, B=string source, C=int index. Skips the runtime range
	// check; the compiler emits this only when the index is provably in [0, len(string)).
	opStringIndexUnchecked

	// opStringIndexToIntUnchecked is the bounds-elided variant of opStringIndexToInt.
	//
	// Same operand shape as opStringIndex but writes an int destination. Same proof
	// requirements.
	opStringIndexToIntUnchecked

	// opSliceGetIntUnchecked is the bounds-elided variant of opSliceGetInt.
	//
	// Operand shape is identical: A=int dest, B=general (reflect) slice source, C=int index.
	// The handler skips checkSliceBounds() and dereferences directly. The compiler emits
	// this opcode only when the access is provably in range. The BCE pass proves the fact
	// via the LEN + LT + JUMP_IF_FALSE pattern against a reflect-bank length op (subOpLen).
	opSliceGetIntUnchecked

	// opSliceSetIntUnchecked is the bounds-elided variant of opSliceSetInt.
	//
	// Operand shape: A=general (reflect) slice dest, B=int index, C=int value. Same proof
	// requirements as the get variant.
	opSliceSetIntUnchecked

	// opPackTyped boxes a typed-bank value into the general bank while preserving its exact
	// source-level reflect.Type, carried via a following opExt word holding a 16-bit
	// typeTable index.
	//
	// Operand shape: A=general dest, B=source register, C=source registerKind. The
	// instruction is two words wide; the handler (handlePackTyped) advances programCounter
	// past the extension.
	//
	// Unlike opPackInterface - which boxes ints unconditionally as int64 and floats as
	// float64 - opPackTyped reconstructs the boxed value with the precise type the compiler
	// recorded (int8, int32, float32, named primitives, ...). This is what lets type
	// assertions distinguish `int` from `int64` and `float64` from `float32` (bug 686).
	opPackTyped

	// opCallScalar is a lean sibling of opCall for scalar-only calls.
	//
	// Emitted at compile time when the callee's full signature fits in piko's typed register
	// banks (int / uint / float / bool / string / complex) with no general-bank parameters
	// or results, no variadic spreading, and the call site is neither a closure nor a
	// linked-generic instantiation. The handler omits the closure, snapshot, variadic, and
	// upvalue branches that are dead for that shape, leaving the dispatch path that
	// recursive scalar algorithms exercise (fib, AST visitors returning ints, expression
	// evaluators) measurably tighter. Eligibility is proven once at emit time by
	// calleeUsesScalarBanksOnly; runtime safety guards on the site/funcIndex bounds remain
	// identical to opCall's so a corrupt bytecode operand still produces a clean interpreted
	// panic. Sited last in the enum so adding it does not shift any existing opcode's
	// numeric value (the ASM jump table indexes opcodes by their iota slot).
	opCallScalar

	// opCallMethodInlineable mirrors opCallMethod with inline dispatch.
	//
	// Behaves identically to opCallMethod at the bytecode level - same operand shape, same
	// call-site lookup. The runtime handler consults a per-callSite per-receiver-type
	// inlineDescriptor cache and, on a known fused shape (binary-op Eval today, extensible
	// to other shapes), runs the body inline in the caller's frame. On a non-inlineable
	// shape it falls back to the standard dispatch via dispatchMethodCallSite. Emitted in
	// place of opCallMethod by rewriteInlineableMethodCalls. Sited after opCallScalar so
	// adding it does not shift any existing opcode's numeric value (ASM jump table
	// convention).
	opCallMethodInlineable

	// opAppendByteFastInPlace is the byte-builder in-place append.
	//
	// A single opcode covers BOTH of the byte-builder mutation shapes:
	//
	//  1. x = append(x, b) - operand B holds the []byte slice reflect.Value directly. The
	//     handler reads the arenaSliceHeader through srcShape.ptr and writes Data/Len/ Cap
	//     back into the same slot.
	//
	//  2. *p = append(*p, b) - operand B holds a *[]byte pointer reflect.Value. The handler
	//     dereferences once to find the slice header (handling flagIndir per the
	//     unsafeReflectValue layout) and writes the new header via runtimeTypedmemmove so
	//     heap-resident headers honour Go's GC write barrier.
	//
	// Covers both byte-builder mutation shapes; subOpStarAppendByteFast still serves the
	// pointer-only `*p = append(*p, b)` site. The compiler emits opAppendByteFastInPlace
	// from both the x-form fast path (tryCompileInPlaceAppend) and the *p-form fast path
	// (tryCompileStarAppendByteFast); the runtime kind switch picks the right extraction
	// path.
	//
	// The safety predicate in compiler_inplace_append_predicate.go guards x-form emission
	// (no aliasing for the slot). *p-form safety follows directly from the address-taken
	// form's explicit pointer indirection - the caller declared the mutation boundary.
	// Post-bytecode alias analysis in function_inplace_append_pass.go demotes
	// opAppendByteFastInPlace back to opAppendByteFast at any site it cannot prove the x
	// form is alias-free.
	//
	// Same operand shape as opAppendByteFast: A=dest general, B=src general (slice or
	// pointer-to-slice), C=byte uint. At runtime A == B by construction. Handler defensively
	// falls back to handleAppendByteFast on a non-arena slice header in the x case; the *p
	// case always uses runtimeTypedmemmove and so is type-safe regardless of where the
	// pointed-to header lives. Sited after opCallMethodInlineable so adding it does not
	// shift any existing opcode's numeric value.
	opAppendByteFastInPlace

	// opAppendInPlace is the generic in-place sibling of opAppend.
	//
	// Handles both x = append(x, e) and *p = append(*p, e) via the same kind-switch the byte
	// variant uses. Falls back to the allocate-fresh-slot path (handleAppend) when the
	// source's slice header is not arena-owned. The element register lives on the general
	// bank for this opcode (boxed values for arbitrary element types); typed-element shapes
	// route through the more specialised tier-1 sub-ops below.
	//
	// Operand shape: A=dest general, B=src general, C=element general. Followed by an opExt
	// extension instruction (mirrors opAppend's shape). At runtime A == B by construction
	// (the safety predicate guarantees no aliasing).
	opAppendInPlace

	// opAppendSpreadInPlace is the spread sibling of opAppendInPlace.
	//
	// Handles x = append(x, src...) for arbitrary element types. The runtime handler routes
	// through arenaAppendByteSpread for byte slices and through the appendSpreadFastPath
	// helper for other element types; either way it writes the result header back to the
	// source's existing slot rather than allocating a fresh one.
	//
	// Operand shape: A=dest general, B=src general, C=source slice general. Followed by
	// opExt. At runtime A == B.
	opAppendSpreadInPlace
)

const (
	// opNop is the alias for the tier-3 no-op encoding.
	//
	// The no-op is expressed as the all-drill instruction word {opDrillTier1,
	// subOpDrillTier2, subOpTier2DrillTier3, subOpTier3Nop} = {0, 0, 0, 0}, which the
	// dispatch macro fast-paths inline (TESTL+JZ on amd64, CBZ on arm64) without ever
	// reaching a handler. The alias preserves the mnemonic at compile sites that emit a
	// literal no-op as padding (function.go peephole-fusion sites, disassembler tests). Both
	// names compile to opcode value 0.
	opNop = opDrillTier1
)

// subOpcode identifies which tier-1 body an opDrillTier1 instruction dispatches to.
//
// Tier 1 holds 2-operand ops: the sub-op id sits in operand A of opDrillTier1, leaving B
// and C as the two register operands. Lives in its own iota block separate from the main
// opcode space so the two enumerations can grow independently.
type subOpcode uint8

const (
	// subOpDrillTier2 is the tier-1 drill marker.
	//
	// The dispatcher reads operand A as the tier-1 sub-opcode; when operand A is 0 it
	// descends into tier 2 (the 1-operand tier) using operand B as the tier-2 discriminator.
	// Reserving iota=0 at the start of the tier-1 enum mirrors the main-tier convention
	// (opDrillTier1 at main-iota 0) so every tier shares the same "byte == 0 means descend"
	// structural rule.
	subOpDrillTier2 subOpcode = iota

	// subOpMathSin sets floats[B] = math.Sin(floats[C]).
	subOpMathSin

	// subOpMathCos sets floats[B] = math.Cos(floats[C]).
	subOpMathCos

	// subOpMathExp sets floats[B] = math.Exp(floats[C]).
	subOpMathExp

	// subOpMathTan sets floats[B] = math.Tan(floats[C]).
	subOpMathTan

	// subOpMathMod sets floats[B] = math.Mod(floats[C], floats[ext.A]).
	subOpMathMod

	// subOpStrconvFormatBool sets strings[B] = strconv.FormatBool(bools[C]).
	subOpStrconvFormatBool

	// subOpStrconvFormatInt sets strings[B] = strconv.FormatInt(ints[C], int(ints[ext.A])).
	subOpStrconvFormatInt

	// subOpStrconvItoa sets strings[B] = strconv.Itoa(int(ints[C])).
	subOpStrconvItoa

	// subOpRealComplex sets floats[B] = real(complex[C]).
	subOpRealComplex

	// subOpImagComplex sets floats[B] = imag(complex[C]).
	subOpImagComplex

	// subOpBytesToString sets strings[B] = string(general[C]) for a []byte source.
	subOpBytesToString

	// subOpMakeMethodExpr builds a method expression value in general[B] from general[C].
	subOpMakeMethodExpr

	// subOpCap sets ints[B] = cap(general[C]) for any capacity-bearing collection bank.
	subOpCap

	// subOpNegComplex sets complex[B] = -complex[C].
	subOpNegComplex

	// subOpMoveComplex copies complex[C] to complex[B].
	subOpMoveComplex

	// subOpMakeSliceInt creates a typed []int64 slice in the slicesInt bank.
	//
	// Operand layout: A=subOpcode, B=dest slicesInt register, C=length int register. The
	// capacity is supplied via the following opExt extension word (operand A). Cold path
	// (per-call, not per-element) so it lives in tier 1 rather than burning a tier-0 slot.
	subOpMakeSliceInt

	// subOpLenSliceIntDirect sets ints[B] = int64(len(slicesInt[C])).
	//
	// Reads len() in pure Go without a reflect.Value.Len call. Lives in tier 1 because it
	// only needs two operands.
	subOpLenSliceIntDirect

	// subOpMakeSliceFloat creates a typed []float64 slice in the slicesFloat bank:
	// slicesFloat[B] = make([]float64, ints[C], ints[ext.A]).
	subOpMakeSliceFloat

	// subOpSliceGetFloatDirect reads a float element from a typed []float64 slice without
	// crossing the reflect boundary: floats[B] = slicesFloat[C][ints[ext.A]]. The index is
	// in the extension word so the three primary operands cover destination, source and the
	// umbrella sub-opcode.
	subOpSliceGetFloatDirect

	// subOpSliceSetFloatDirect writes a float value to a typed []float64 slice element:
	// slicesFloat[B][ints[C]] = floats[ext.A].
	subOpSliceSetFloatDirect

	// subOpLenSliceFloatDirect sets ints[B] = int64(len(slicesFloat[C])).
	subOpLenSliceFloatDirect

	// subOpMakeSliceString creates a typed []string slice in the slicesString bank.
	subOpMakeSliceString

	// subOpSliceGetStringDirect reads a string element from a typed []string slice:
	// strings[B] = slicesString[C][ints[ext.A]].
	subOpSliceGetStringDirect

	// subOpSliceSetStringDirect writes a string value to a typed []string slice element:
	// slicesString[B][ints[C]] = strings[ext.A].
	subOpSliceSetStringDirect

	// subOpLenSliceStringDirect sets ints[B] = int64(len(slicesString[C])).
	subOpLenSliceStringDirect

	// subOpMakeSliceBool creates a typed []bool slice in the slicesBool bank.
	subOpMakeSliceBool

	// subOpSliceGetBoolDirect reads a bool element from a typed []bool slice: bools[B] =
	// slicesBool[C][ints[ext.A]].
	subOpSliceGetBoolDirect

	// subOpSliceSetBoolDirect writes a bool value to a typed []bool slice element:
	// slicesBool[B][ints[C]] = bools[ext.A].
	subOpSliceSetBoolDirect

	// subOpLenSliceBoolDirect sets ints[B] = int64(len(slicesBool[C])).
	subOpLenSliceBoolDirect

	// subOpMakeSliceUint creates a typed []uint64 slice in the slicesUint bank.
	subOpMakeSliceUint

	// subOpSliceGetUintDirect reads a uint element from a typed []uint64 slice: uints[B] =
	// slicesUint[C][ints[ext.A]].
	subOpSliceGetUintDirect

	// subOpSliceSetUintDirect writes a uint value to a typed []uint64 slice element:
	// slicesUint[B][ints[C]] = uints[ext.A].
	subOpSliceSetUintDirect

	// subOpLenSliceUintDirect sets ints[B] = int64(len(slicesUint[C])).
	subOpLenSliceUintDirect

	// subOpBoxSliceInt converts a typed []int64 slice in the slicesInt bank to a
	// reflect.Value in the general bank: general[B] = reflect.ValueOf(slicesInt[C]). Used at
	// boundaries where a typed slice is consumed by a reflect-bank operation (native
	// function call, map value insert, interface conversion).
	subOpBoxSliceInt

	// subOpUnboxSliceInt converts a reflect.Value of kind Slice with int64 elements back
	// into the slicesInt bank: slicesInt[B] = reflect.TypeAssert[[]int64](general[C]). Used
	// after receiving a typed slice from a reflect-bank source (function return, channel
	// receive, slice expression on a general slice).
	subOpUnboxSliceInt

	// subOpMoveInt copies ints[C] to ints[B].
	//
	// Two-register-operand sub-op where the pair of operands is encoded as destination in
	// slot B and source in slot C; the Go-side handler is invoked from handleUmbrella with a
	// synthetic instruction {a: instruction.b, b: instruction.c} so the handler reads the
	// operands from its conventional positions.
	subOpMoveInt

	// subOpMoveFloat copies floats[C] to floats[B].
	subOpMoveFloat

	// subOpMoveString copies strings[C] to strings[B].
	subOpMoveString

	// subOpMoveBool copies bools[C] to bools[B].
	subOpMoveBool

	// subOpMoveUint copies uints[C] to uints[B].
	subOpMoveUint

	// subOpMoveIntToGeneral boxes ints[C] into general[B].
	subOpMoveIntToGeneral

	// subOpMoveGeneralToInt unboxes general[C] into ints[B].
	subOpMoveGeneralToInt

	// subOpMoveFloatToGeneral boxes floats[C] into general[B].
	subOpMoveFloatToGeneral

	// subOpMoveGeneralToFloat unboxes general[C] into floats[B].
	subOpMoveGeneralToFloat

	// subOpMoveStringToGeneral boxes strings[C] into general[B].
	subOpMoveStringToGeneral

	// subOpMoveGeneralToString unboxes general[C] into strings[B].
	subOpMoveGeneralToString

	// subOpNegInt sets ints[B] = -ints[C].
	subOpNegInt

	// subOpNegFloat sets floats[B] = -floats[C].
	subOpNegFloat

	// subOpBitNot sets ints[B] = ^ints[C].
	subOpBitNot

	// subOpBitNotUint sets uints[B] = ^uints[C].
	subOpBitNotUint

	// subOpIntToFloat converts ints[C] to floats[B].
	subOpIntToFloat

	// subOpFloatToInt converts floats[C] to ints[B].
	subOpFloatToInt

	// subOpNot inverts bools[C] into ints[B] (treated as bool by the caller; Go-side handler
	// reads/writes the int bank as bool).
	subOpNot

	// subOpBoolToInt converts bools[C] to ints[B].
	subOpBoolToInt

	// subOpIntToBool converts ints[C] to bools[B].
	subOpIntToBool

	// subOpIntToUint converts ints[C] to uints[B].
	subOpIntToUint

	// subOpUintToInt converts uints[C] to ints[B].
	subOpUintToInt

	// subOpUintToFloat converts uints[C] to floats[B].
	subOpUintToFloat

	// subOpFloatToUint converts floats[C] to uints[B].
	subOpFloatToUint

	// subOpMathSqrt sets floats[B] = math.Sqrt(floats[C]).
	subOpMathSqrt

	// subOpMathAbs sets floats[B] = math.Abs(floats[C]).
	subOpMathAbs

	// subOpMathFloor sets floats[B] = math.Floor(floats[C]).
	subOpMathFloor

	// subOpMathCeil sets floats[B] = math.Ceil(floats[C]).
	subOpMathCeil

	// subOpMathTrunc sets floats[B] = math.Trunc(floats[C]).
	subOpMathTrunc

	// subOpMathRound sets floats[B] = math.Round(floats[C]).
	subOpMathRound

	// subOpLenString sets ints[B] = len(strings[C]).
	subOpLenString

	// subOpRuneToString sets strings[B] = string(rune(ints[C])).
	subOpRuneToString

	// subOpStrToUpper sets strings[B] = strings.ToUpper(strings[C]).
	subOpStrToUpper

	// subOpStrToLower sets strings[B] = strings.ToLower(strings[C]).
	subOpStrToLower

	// subOpStrTrimSpace sets strings[B] = strings.TrimSpace(strings[C]).
	subOpStrTrimSpace

	// subOpLen sets ints[B] = len(general[C]) for any collection bank.
	subOpLen

	// subOpStringToBytes sets general[B] = []byte(strings[C]).
	subOpStringToBytes

	// subOpUnsafeStringData sets general[B] to the *byte data pointer of strings[C].
	subOpUnsafeStringData

	// subOpUnsafeSliceData sets general[B] to the *element data pointer of the slice in
	// general[C].
	subOpUnsafeSliceData

	// subOpJump unconditionally jumps by the signed offset B|(C<<8).
	subOpJump

	// subOpLoadIntConstSmall sets ints[B] = int64(C).
	//
	// Materialises a small integer literal inline from the instruction word without a
	// constant-pool indirection.
	subOpLoadIntConstSmall

	// subOpLoadBool sets ints[B] = C (0 or 1).
	//
	// Materialises a boolean literal inline from the instruction word.
	subOpLoadBool

	// subOpIncIntJumpLt fuses opIncInt + opLtInt + opJumpIfTrue.
	//
	// Peephole-emitted superinstruction for the canonical for-loop back-edge. Increments
	// ints[B] and conditionally jumps (offset in the following opExt) when ints[B] <
	// ints[C].
	subOpIncIntJumpLt

	// subOpLenStringLtJumpFalse fuses opLenString + opLtInt + opJumpIfFalse.
	//
	// Peephole-emitted superinstruction for string-loop conditions. Jumps when ints[B] >=
	// len(strings[C]).
	subOpLenStringLtJumpFalse

	// subOpLoadZero zeroes the register at index B in the bank named by kind C.
	subOpLoadZero

	// subOpMakeMap allocates a new map of the type indexed by general[B] with initial
	// capacity ints[C].
	subOpMakeMap

	// subOpMakeChannel allocates a new channel of the type indexed by general[B] with
	// capacity ints[C].
	subOpMakeChannel

	// subOpMapDelete deletes general[C] from the map at general[B].
	subOpMapDelete

	// subOpChannelReceive receives a value from the channel at general[B] into the
	// destination encoded by ints[C].
	subOpChannelReceive

	// subOpGetMethod resolves the method named by the following opExt word on general[B] and
	// stores the bound method in general[C].
	subOpGetMethod

	// subOpSpill writes the register at index B (in bank kind C) into a reserved spill slot
	// encoded in the following opExt word.
	subOpSpill

	// subOpReload reads from the spill slot encoded in the following opExt word into the
	// register at index B (in bank kind C).
	subOpReload

	// subOpGetStructFieldInt reads an integer-kind struct field via the structLayoutTable.
	//
	// Operand layout: B=destination int register, C=source general register holding the
	// struct. The following opExt word carries the 16-bit layoutTable index (A=low byte,
	// B=high byte). The handler computes base + layout.Offset, switches on layout.Kind to do
	// the matching typed memory read, and stores into ints[B]. Covers int8/int16/int32/int64
	// (and int on 64-bit targets).
	subOpGetStructFieldInt

	// subOpGetStructFieldUint reads an unsigned-integer struct field via layoutTable.
	//
	// Stores into uints[B]. Covers uint8/uint16/uint32/uint64 (and uint, uintptr).
	subOpGetStructFieldUint

	// subOpGetStructFieldFloat reads a float-kind struct field via the layoutTable
	// mechanism.
	//
	// Stores into floats[B]. Covers float32 (sign-extended on read) and float64.
	subOpGetStructFieldFloat

	// subOpGetStructFieldBool reads a bool-kind struct field via the layoutTable mechanism.
	// Stores into bools[B].
	subOpGetStructFieldBool

	// subOpGetStructFieldString reads a string-kind struct field via the layoutTable
	// mechanism.
	//
	// Copies the 16-byte string header (data pointer + length) into strings[B]. Reads are
	// barrier-free; writes use a separate sub-op with barrier-correct semantics.
	subOpGetStructFieldString

	// subOpSetStructFieldInt writes an integer-kind struct field via the layoutTable
	// mechanism.
	//
	// Operand layout: B=destination general register holding the struct, C=source int
	// register. The following opExt word carries the layoutTable index.
	subOpSetStructFieldInt

	// subOpSetStructFieldUint writes an unsigned-integer-kind struct field via the
	// layoutTable mechanism.
	subOpSetStructFieldUint

	// subOpSetStructFieldFloat writes a float-kind struct field via the layoutTable
	// mechanism.
	subOpSetStructFieldFloat

	// subOpSetStructFieldBool writes a bool-kind struct field via the layoutTable mechanism.
	subOpSetStructFieldBool

	// subOpSetStructFieldString writes a string-kind struct field via the layoutTable
	// mechanism. Uses reflect.NewAt+SetString under the hood so the GC write barrier fires
	// correctly when the destination struct lives on the heap.
	subOpSetStructFieldString

	// subOpAddUintConst adds a uintConstants entry into the uint bank.
	//
	// Encoded with A = destination, B = source, C = const pool index, so uints[A] = uints[B]
	// + uintConstants[C]. Mirrors opAddIntConst but for the uint bank; emitted by
	// fuseArithConst when it spots LOAD_UINT_CONST followed by ADD_UINT.
	subOpAddUintConst

	// subOpSubUintConst sets uints[A] = uints[B] - uintConstants[C]. Mirror of opSubIntConst
	// for the uint bank.
	subOpSubUintConst

	// subOpBitAndUintConst sets uints[A] = uints[B] & uintConstants[C]. Common in
	// masking/checksum/LCG code that uses uint64 wraparound; fuses LOAD_UINT_CONST +
	// BIT_AND_UINT.
	subOpBitAndUintConst

	// subOpLoadUintConstSmall sets uints[B] = uint64(C).
	//
	// Materialises a small unsigned literal inline from the instruction word without a
	// constant-pool indirection. Mirror of subOpLoadIntConstSmall for the uint bank.
	subOpLoadUintConstSmall

	// subOpAppendUint appends uints[ext.A] to a uint-element slice.
	//
	// Source slice is general[C], result lands in general[B]. Element types include []byte /
	// []uint8 (expr_eval's `append(*output, '(')` hot path) as well as []uint64 / []uint /
	// []uintptr. The element is taken from the uint register bank so the compiler skips the
	// box-to-general step the generic opAppend would require. Falls back to reflect.Append
	// for the long-tail uint slice types; the steady-state cost on the byte fast path is one
	// reflect.TypeAssert[[]byte] plus the existing arenaAppendByte bump.
	subOpAppendUint

	// subOpAppendInt is the tier-1 sibling of subOpAppendUint for int-bank elements.
	//
	// Sets general[B] = append(general[C], int*(ints[ext.A])) where the slice's static
	// element kind resolved to registerInt (Int / Int8 / Int16 / Int32 / Int64).
	subOpAppendInt

	// subOpAppendString is the tier-1 sibling for string elements: general[B] =
	// append(general[C], strings[ext.A]).
	subOpAppendString

	// subOpAppendFloat is the tier-1 sibling for float elements: general[B] =
	// append(general[C], float*(floats[ext.A])).
	subOpAppendFloat

	// subOpAppendBool is the tier-1 sibling for bool elements: general[B] =
	// append(general[C], bools[ext.A]).
	subOpAppendBool

	// subOpStarAppendByteFast fuses `*p = append(*p, b)` into a single op.
	//
	// Eliminates the intermediate reflect.Value (and its 24 B composite-literal slice
	// header) that opAppendByteFast + opSetField (deref) would otherwise produce. The
	// handler reads *pointer, appends the byte via Go's builtin append, and writes the new
	// slice header back to *pointer through runtimeTypedmemmove (preserves the GC write
	// barrier on the Data field).
	//
	// Encoding:
	//
	// 	op = opDrillTier1
	// 	a  = subOpStarAppendByteFast
	// 	b  = pointer general register (a `*[]byte`)
	// 	c  = byte-value uint register
	subOpStarAppendByteFast

	// subOpStarAppendByteSpread fuses `*p = append(*p, X...)` where `*p` is `[]byte` and `X`
	// is `[]byte`. Same motivation as subOpStarAppendByteFast: skip the intermediate
	// reflect.Value.
	//
	// Encoding:
	//
	// 	op = opDrillTier1
	// 	a  = subOpStarAppendByteSpread
	// 	b  = pointer general register (a `*[]byte`)
	// 	c  = source slice general register (the `[]byte` to spread)
	//
	// expr_eval's `*output = append(*output, intToDecimalBytes(v)...)` pattern fires this
	// per emitted decimal number.
	subOpStarAppendByteSpread

	// subOpIncStructFieldInt fuses `s.field++` for an int-kind struct field.
	//
	// Eliminates the opGetFieldInt + subOpTier2IncInt + opSetFieldInt three-op sequence.
	// Reads the field via the resolved layout (compile-time-known offset + kind), increments
	// in place, writes back through the same unsafe pointer: no int register temp, no second
	// receiver lookup.
	//
	// Encoding:
	//
	// 	op = opDrillTier1
	// 	a  = subOpIncStructFieldInt
	// 	b  = receiver general register (Pointer / addressable struct)
	// 	c  = layoutTable index (uint8)
	//
	// expr_eval's `parser.position++` (Pratt parser token cursor) dominates this; LRU's
	// `cache.size++` and similar counters also benefit.
	subOpIncStructFieldInt

	// subOpDecStructFieldInt is the decrement sibling of subOpIncStructFieldInt. Same
	// encoding, same constraints, `s.field--` semantics.
	subOpDecStructFieldInt

	// subOpIncStructFieldUint mirrors subOpIncStructFieldInt for uint-kind fields. Same
	// encoding.
	subOpIncStructFieldUint

	// subOpDecStructFieldUint is the decrement sibling of subOpIncStructFieldUint.
	subOpDecStructFieldUint

	// subOpMakeSliceByte builds slicesByte[B] = make([]byte, ints[C], ints[ext.A]).
	//
	// Byte-slice typed-bank sub-op mirroring the int/float/string/bool/uint siblings but
	// operating on the slicesByte register bank. Element kind is byte (uint8); the typed
	// register holds the raw []byte slice header so element access avoids the reflect.Value
	// boxing pattern.
	subOpMakeSliceByte

	// subOpSliceGetByteDirect sets uints[B] = uint64(slicesByte[C][ints[ext.A]]).
	subOpSliceGetByteDirect

	// subOpSliceSetByteDirect sets slicesByte[B][ints[ext.A]] = byte(uints[ext.B]).
	subOpSliceSetByteDirect

	// subOpLenSliceByteDirect sets ints[B] = int64(len(slicesByte[C])).
	subOpLenSliceByteDirect

	// subOpSliceByteSlice performs a three-way slice on slicesByte[C] via ext words.
	subOpSliceByteSlice

	// subOpRangeNextSliceByte advances a typed range step over a slicesByte entry.
	subOpRangeNextSliceByte

	// subOpBoxSliceByte sets general[B] = reflect.ValueOf(slicesByte[C]).
	subOpBoxSliceByte

	// subOpUnboxSliceByte sets slicesByte[B] = reflect.TypeAssert[[]byte](general[C]).
	subOpUnboxSliceByte

	// subOpSliceByteToString converts the typed-bank []byte at slicesByte[C] to strings[B].
	//
	// Sibling of subOpBytesToString but reads from the typed bank instead of general. Goes
	// through the same arena-byte-slab string conversion as the general-bank path.
	subOpSliceByteToString

	// subOpRangeCheckUintJumpFalse is the fused range-check super-instruction.
	//
	// Collapses the canonical byte classifier idiom `value >= lo && value <= hi` (8 ops in
	// tier-1 form with LOAD_UINT_CONST_SMALL + GE_UINT + MOVE_INT + JUMP_IF_FALSE +
	// LOAD_UINT_CONST_SMALL + LE_UINT + MOVE_INT + JUMP_IF_FALSE) into a single tier-1
	// sub-op plus two extension words.
	//
	// Primary word operands:
	//
	// 	{opDrillTier1, subOpRangeCheckUintJumpFalse, valueReg, 0}
	//
	// Extension words:
	//
	// 	ext1: {opExt, loConst, hiConst, 0}
	// 	ext2: {opExt, jumpOffsetLo, jumpOffsetHi, 0}
	//
	// The handler reads value = uints[valueReg]. If value < loConst or value > hiConst,
	// programCounter += jumpOffset. The compiler emits the offset already adjusted for the
	// five trailing opNops that occupy the original 5 word slots after fusion.
	subOpRangeCheckUintJumpFalse

	// subOpEqUintConstJumpFalse fuses the `switch byte` / `if v == const` dispatch pattern
	// into a single tier-1 sub-op + 1 extension word. Mirrors subOpRangeCheckUintJumpFalse
	// but for the simpler 3-op equality case that dominates byte-switch dispatch (brainfuck,
	// generic byte classifiers, BF-style interpreters).
	//
	// Pre-fusion shape (3 ops):
	//
	// 	{opDrillTier1, subOpLoadUintConstSmall, constReg, immVal}
	// 	{opEqUint, condReg, valueReg, constReg}
	// 	{opJumpIfFalse, condReg, jumpOffsetLo, jumpOffsetHi}
	//
	// Post-fusion shape (3 ops, with 2 trailing nops):
	//
	// 	{opDrillTier1, subOpEqUintConstJumpFalse, valueReg, immConst}
	// 	{opExt, jumpOffsetLo, jumpOffsetHi, 0}
	// 	opNop
	//
	// Primary word operands: B = value reg, C = 8-bit immediate const. If uints[valueReg] !=
	// immConst, programCounter += jumpOffset.
	subOpEqUintConstJumpFalse

	// subOpSimdDotProductFloat64 computes sum += a[i] * b[i] for two equal-length []float64
	// slices.
	//
	// SIMD kernel sub-opcodes are emitted only by the AST pattern-recognition subsystem when
	// a canonical numerical loop shape matches a registered recogniser. They subsume an
	// entire scalar loop into a single dispatch + one kernel call, paying the per-op
	// dispatch overhead once instead of N times.
	//
	// The handlers dispatch to the architecture-tuned float64 kernels in
	// internal/vectormaths (SSE2 / AVX2 on amd64, scalar fallback elsewhere).
	//
	// Sited last in the tier-1 enum so adding them does not shift existing iota values (the
	// ASM jump table indexes by slot).
	//
	// Primary word: B = destination float register, C = slice A (general bank). Extension
	// word: A = slice B (general bank).
	subOpSimdDotProductFloat64

	// subOpSimdSumSliceFloat64 computes sum += a[i] for a []float64. Primary word: B =
	// destination float register, C = slice A (general bank).
	subOpSimdSumSliceFloat64

	// subOpSimdNormSquaredFloat64 computes sum += a[i]*a[i] for a []float64 - special case
	// of dot product with itself. Primary word: B = destination float register, C = slice A
	// (general bank).
	subOpSimdNormSquaredFloat64

	// subOpSimdEuclideanDistanceSquaredFloat64 sums squared deltas.
	//
	// Computes sum += (a[i] - b[i]) * (a[i] - b[i]) for two equal-length []float64 slices.
	// Primary word: B = destination float register, C = slice A (general bank). Extension
	// word: A = slice B (general bank).
	subOpSimdEuclideanDistanceSquaredFloat64

	// subOpSimdMaxSliceFloat64 reduces a []float64 to its maximum.
	//
	// An empty slice produces math.Inf(-1). Primary word: B = destination float register, C
	// = slice (general bank).
	subOpSimdMaxSliceFloat64

	// subOpSimdMinSliceFloat64 reduces a []float64 to its minimum.
	//
	// An empty slice produces math.Inf(+1). Primary word: B = destination float register, C
	// = slice (general bank).
	subOpSimdMinSliceFloat64

	// subOpSimdAddSliceFloat64 sums two []float64 slices elementwise.
	//
	// Computes dst[i] = a[i] + b[i] element-wise for three equal-length []float64 slices.
	// Primary word: B = destination slice (general bank), C = slice A (general bank).
	// Extension word: A = slice B (general bank).
	subOpSimdAddSliceFloat64

	// subOpSimdSubSliceFloat64 computes dst[i] = a[i] - b[i]. Operand layout same as
	// subOpSimdAddSliceFloat64.
	subOpSimdSubSliceFloat64

	// subOpSimdMulSliceFloat64 computes dst[i] = a[i] * b[i] (Hadamard product). Operand
	// layout same as subOpSimdAddSliceFloat64.
	subOpSimdMulSliceFloat64

	// subOpSimdAxpyFloat64 computes the BLAS axpy update.
	//
	// Computes y[i] += alpha * x[i] for two equal-length []float64 slices and a scalar
	// alpha. Primary word: B = slice Y (general bank), C = slice X (general bank). Extension
	// word: A = scalar alpha (float register).
	subOpSimdAxpyFloat64

	// subOpSimdScaleSliceFloat64 computes s[i] *= k in place for a []float64 and a scalar k.
	// Primary word: B = slice (general bank), C = scalar k (float register).
	subOpSimdScaleSliceFloat64

	// subOpSimdClearSliceFloat64 zeroes every element of a []float64. Primary word: B =
	// slice (general bank).
	subOpSimdClearSliceFloat64

	// subOpSimdFillSliceFloat64 sets every element of a []float64 to a scalar value. Primary
	// word: B = slice (general bank), C = scalar value (float register).
	subOpSimdFillSliceFloat64

	// subOpAdoptGeneralToSlicesFloat extracts []float64 into typed bank.
	//
	// Extracts a []float64 from a reflect.Value held in the general bank and writes the
	// slice header to a typed slicesFloat-bank register. Used by the SIMD recogniser to
	// bridge general-bank operands (function parameters of type []float64, returns from
	// non-typed-bank callers) into the typed-bank fast path the SIMD opcodes require.
	// Primary word: B = destination slicesFloat register, C = source general register. A
	// type-assertion mismatch (the static type system promised []float64 but the runtime
	// value is something else) raises an interpreted panic; the recogniser's compile-time
	// isFloat64Slice gate makes this defensive.
	subOpAdoptGeneralToSlicesFloat

	// subOpAdoptGeneralToSlicesInt extracts []int64 into typed bank.
	//
	// Extracts a []int64 from a reflect.Value in the general bank and writes the slice
	// header to a typed slicesInt register. Mirror of subOpAdoptGeneralToSlicesFloat for the
	// int bank. Used by the bytecode inliner when splicing a cross-bank argument (caller's
	// general-bank value flowing into a callee's []int64 parameter slot).
	subOpAdoptGeneralToSlicesInt

	// subOpAdoptGeneralToSlicesString mirrors subOpAdoptGeneralToSlicesFloat for the
	// slicesString bank ([]string).
	subOpAdoptGeneralToSlicesString

	// subOpAdoptGeneralToSlicesBool mirrors subOpAdoptGeneralToSlicesFloat for the
	// slicesBool bank ([]bool).
	subOpAdoptGeneralToSlicesBool

	// subOpAdoptGeneralToSlicesUint mirrors subOpAdoptGeneralToSlicesFloat for the
	// slicesUint bank ([]uint64).
	subOpAdoptGeneralToSlicesUint

	// subOpAdoptGeneralToSlicesByte mirrors subOpAdoptGeneralToSlicesFloat for the
	// slicesByte bank ([]byte).
	subOpAdoptGeneralToSlicesByte

	// subOpBoxSliceFloat wraps slicesFloat[C] into general[B].
	//
	// Wraps a typed []float64 slice header from slicesFloat[C] into a reflect.Value in the
	// general bank: general[B] = reflect.ValueOf(slicesFloat[C]). Mirror of subOpBoxSliceInt
	// for the float bank. Used by the bytecode inliner when splicing a cross-bank argument
	// (caller's typed-bank slice flowing into a callee's general-bank parameter slot) and by
	// emit sites that need to surface a typed slice into a reflect-bank consumer.
	subOpBoxSliceFloat

	// subOpBoxSliceString mirrors subOpBoxSliceFloat for the slicesString bank.
	subOpBoxSliceString

	// subOpBoxSliceBool mirrors subOpBoxSliceFloat for the slicesBool bank.
	subOpBoxSliceBool

	// subOpBoxSliceUint mirrors subOpBoxSliceFloat for the slicesUint bank.
	subOpBoxSliceUint

	// subOpMoveSliceInt copies slicesInt[C] to slicesInt[B] - a same-bank typed-slice header
	// move. Used by the bytecode inliner when splicing a callee body whose parameter and the
	// caller-side argument share a typed-slice bank but differ in register index.
	subOpMoveSliceInt

	// subOpMoveSliceFloat copies slicesFloat[C] to slicesFloat[B].
	subOpMoveSliceFloat

	// subOpMoveSliceString copies slicesString[C] to slicesString[B].
	subOpMoveSliceString

	// subOpMoveSliceBool copies slicesBool[C] to slicesBool[B].
	subOpMoveSliceBool

	// subOpMoveSliceUint copies slicesUint[C] to slicesUint[B].
	subOpMoveSliceUint

	// subOpMoveSliceByte copies slicesByte[C] to slicesByte[B].
	subOpMoveSliceByte

	// subOpAppendSliceIntDirect appends int64 to a typed-bank slice.
	//
	// Appends an int64 to a typed-bank []int64 slice without crossing through the
	// general/reflect bank. Reads source slice from slicesInt[C], element value from
	// ints[ext.A], writes destination to slicesInt[B]. Used when the slice operand of an
	// append call is statically known to live on the typed slicesInt bank
	// (parameter-survivor or local from make([]int64, ...)).
	subOpAppendSliceIntDirect

	// subOpAppendSliceFloatDirect appends a float64 to a typed-bank []float64 slice. Same
	// encoding as subOpAppendSliceIntDirect but reads slicesFloat / floats and writes
	// slicesFloat.
	subOpAppendSliceFloatDirect

	// subOpAppendSliceStringDirect appends a string to a typed-bank []string slice.
	subOpAppendSliceStringDirect

	// subOpAppendSliceBoolDirect appends a bool to a typed-bank []bool slice.
	subOpAppendSliceBoolDirect

	// subOpAppendSliceUintDirect appends a uint64 to a typed-bank []uint64 slice.
	subOpAppendSliceUintDirect

	// subOpAppendSliceByteDirect appends a byte to a typed-bank []byte slice. Reads element
	// from uints (piko's storage for uint8 scalar values).
	subOpAppendSliceByteDirect

	// subOpSliceSliceIntDirect three-way slices a slicesInt header.
	//
	// Performs `source[low:high:cap]` on slicesInt[C] and writes the resulting header to
	// slicesInt[B]. Mirrors subOpSliceByteSlice but for the typed int bank. Encoding:
	// primary word holds the source slicesInt index in C and destination in B; an extension
	// word holds the low/high/cap flags and the int-bank registers carrying the bound
	// values, in the same layout subOpSliceByteSlice uses.
	subOpSliceSliceIntDirect

	// subOpSliceSliceFloatDirect mirrors subOpSliceSliceIntDirect for slicesFloat.
	subOpSliceSliceFloatDirect

	// subOpSliceSliceStringDirect mirrors subOpSliceSliceIntDirect for slicesString.
	subOpSliceSliceStringDirect

	// subOpSliceSliceBoolDirect mirrors subOpSliceSliceIntDirect for slicesBool.
	subOpSliceSliceBoolDirect

	// subOpSliceSliceUintDirect mirrors subOpSliceSliceIntDirect for slicesUint.
	subOpSliceSliceUintDirect

	// subOpCopySliceIntDirect implements `copy(dst, src)` between two typed slicesInt
	// headers without crossing through reflect.Copy. Encoding: operand A holds the
	// destination int register that receives the element count (mirroring opCopy's
	// int-result convention), B holds the destination slicesInt register, C holds the source
	// slicesInt register.
	subOpCopySliceIntDirect

	// subOpCopySliceFloatDirect mirrors subOpCopySliceIntDirect for slicesFloat.
	subOpCopySliceFloatDirect

	// subOpCopySliceStringDirect mirrors subOpCopySliceIntDirect for slicesString.
	subOpCopySliceStringDirect

	// subOpCopySliceBoolDirect mirrors subOpCopySliceIntDirect for slicesBool.
	subOpCopySliceBoolDirect

	// subOpCopySliceUintDirect mirrors subOpCopySliceIntDirect for slicesUint.
	subOpCopySliceUintDirect

	// subOpCopySliceByteDirect mirrors subOpCopySliceIntDirect for slicesByte.
	subOpCopySliceByteDirect

	// subOpAppendUintInPlace is the in-place sibling of subOpAppendUint.
	//
	// Covers the uint-width slice types ([]uint16, []uint32, []uint64, []uint, []uintptr)
	// that the existing subOpAppendUint handler covers. Mutates the source slice's
	// arenaSliceHeader slot directly. ([]byte appends route through opAppendByteFastInPlace
	// at tier 0 - they don't need this sub-op.)
	//
	// Encoding:
	//
	// 	op = opDrillTier1
	// 	a  = subOpAppendUintInPlace
	// 	b  = destination general register (== c by construction)
	// 	c  = source slice general register
	// 	ext.a = element uint register
	//
	// At runtime B == C; the result reflect.Value remains in B/C with its slot now holding
	// the mutated header. Falls back to handleSubOpAppendUint when the source slot is not
	// arena-owned.
	subOpAppendUintInPlace

	// subOpAppendByteSpreadInPlace handles byte spread append in place.
	//
	// Unifies the plain-identifier and pointer forms through one runtime kind switch that
	// picks the right header-extraction path. It handles both x = append(x, src...) when
	// registers.general[c] holds a []byte, AND *p = append(*p, src...) when
	// registers.general[c] holds a *[]byte.
	//
	// Encoding:
	//
	// 	op = opDrillTier1
	// 	a  = subOpAppendByteSpreadInPlace
	// 	b  = pointer/slice general register
	// 	c  = source slice general register ([]byte to spread)
	subOpAppendByteSpreadInPlace

	// subOpGetStructFieldSliceInt reads a slice-typed struct field whose element kind is
	// exactly int64 directly into slicesInt[B], aliasing the field's backing array. Bypasses
	// the acquireSliceSnapshot path in tryGetStructFieldUnsafe's Slice arm.
	//
	// Encoding:
	//
	// 	opDrillTier1, sub-op id, B = destination slicesInt register,
	// 	C = source general register (the struct receiver).
	// 	EXT (next bytecode word): A | (B<<8) = layout index (uint16).
	//
	// The compile-side picker pickGetStructFieldSliceSubOp only admits this sub-op when the
	// field's element type is exactly int64; narrower widths fall back to
	// opGetStructFieldGeneral (which then routes through PR 1's unboxToTypedIntSlice on
	// cross-bank boundaries).
	//
	// Appended to the END of the tier-1 sub-op enum so existing pre-computed dispatch tables
	// (asmJumpTable etc.) keep their stable iota positions; inserting in the middle would
	// silently shift all subsequent values and misalign any precomputed slots.
	subOpGetStructFieldSliceInt

	// subOpGetStructFieldSliceFloat - float64 element bank counterpart of
	// subOpGetStructFieldSliceInt. Reads into slicesFloat[B].
	subOpGetStructFieldSliceFloat

	// subOpGetStructFieldSliceUint - uint64 element bank counterpart. Reads into
	// slicesUint[B].
	subOpGetStructFieldSliceUint

	// subOpGetStructFieldSliceString - string element bank counterpart. Reads into
	// slicesString[B].
	subOpGetStructFieldSliceString

	// subOpGetStructFieldSliceBool - bool element bank counterpart. Reads into
	// slicesBool[B].
	subOpGetStructFieldSliceBool

	// subOpGetStructFieldSliceByte - byte (uint8) element bank counterpart. Reads into
	// slicesByte[B].
	subOpGetStructFieldSliceByte

	// subOpSetStructFieldSliceInt writes slicesInt[C] into the receiver's []int64-typed
	// field, going through runtime.typedmemmove against the field's reflect.Type so the GC
	// write barrier fires correctly when the destination struct lives on the heap.
	//
	// Encoding:
	//
	// 	opDrillTier1, sub-op id, B = destination general register
	// 	(the struct receiver), C = source slicesInt register.
	// 	EXT (next bytecode word): A | (B<<8) = layout index (uint16).
	//
	// The compile-side picker pickSetStructFieldSliceSubOp only admits this sub-op when the
	// field's element type matches the bank's canonical storage width AND the source value's
	// kind is registerSliceInt; mixed-width or general-bank source fall back to opSetField.
	subOpSetStructFieldSliceInt

	// subOpSetStructFieldSliceFloat - float64 counterpart.
	subOpSetStructFieldSliceFloat

	// subOpSetStructFieldSliceUint - uint64 counterpart.
	subOpSetStructFieldSliceUint

	// subOpSetStructFieldSliceString - string counterpart.
	subOpSetStructFieldSliceString

	// subOpSetStructFieldSliceBool - bool counterpart.
	subOpSetStructFieldSliceBool

	// subOpSetStructFieldSliceByte - byte counterpart.
	subOpSetStructFieldSliceByte

	// subOpCapSliceIntDirect sets ints[B] = int64(cap(slicesInt[C])).
	//
	// Reads cap() in pure Go without a reflect.Value.Cap call; mirrors
	// subOpLenSliceIntDirect for the capacity axis. Compiled by compileBuiltinCap when the
	// argument is a typed-slice register; without this the call falls back to a "cap not
	// supported for register kind sliceInt" compile error, which broke modules that inspect
	// buffer capacity such as gopkg.in/yaml.v3's yaml_parser_update_raw_buffer.
	subOpCapSliceIntDirect

	// subOpCapSliceFloatDirect - float64 counterpart of subOpCapSliceIntDirect.
	subOpCapSliceFloatDirect

	// subOpCapSliceStringDirect - string counterpart.
	subOpCapSliceStringDirect

	// subOpCapSliceBoolDirect - bool counterpart.
	subOpCapSliceBoolDirect

	// subOpCapSliceUintDirect - uint64 counterpart.
	subOpCapSliceUintDirect

	// subOpCapSliceByteDirect - byte counterpart.
	subOpCapSliceByteDirect
)

const (
	// opcodeCount is a sentinel marking the number of opcodes. Declared as int rather than
	// as part of the opcode iota block so the array-sizing uses (handlerTable, opcodeNames,
	// operandShapes, CostTable) compile even when the active opcode set fills the uint8
	// range.
	opcodeCount = int(opAppendSpreadInPlace) + 1
)

const (
	// subOpTier2DrillTier3 is the tier-2 sub-opcode reserved at index 0 to drill from tier 2
	// down into tier 3. Following the "index 0 means drill down" convention, tier 2 dispatch
	// sees operand B == 0 and falls through to tier 3 dispatch using operand C as the tier-3
	// sub-opcode discriminator.
	subOpTier2DrillTier3 subOpcodeTier2 = iota

	// subOpTier2IncInt increments the int register named by operand C. Lives in tier 2
	// because the op uses only one register operand.
	subOpTier2IncInt

	// subOpTier2DecInt decrements the int register named by operand C.
	subOpTier2DecInt

	// subOpTier2IncUint increments the uint register named by operand C.
	subOpTier2IncUint

	// subOpTier2DecUint decrements the uint register named by operand C.
	subOpTier2DecUint

	// subOpTier2Panic raises a panic with the value in the general register named by operand
	// C.
	subOpTier2Panic

	// subOpTier2Recover stores the active panic value in the general register named by
	// operand C, or an invalid value if not panicking.
	subOpTier2Recover

	// subOpTier2SetZero zeroes the general register named by operand C (used by the
	// assign-through optimisation to clear composite values in place).
	subOpTier2SetZero

	// subOpTier2ChannelClose closes the channel held by the general register named by
	// operand C.
	subOpTier2ChannelClose

	// subOpTier2LoadNil writes a typed nil into the general register named by operand C.
	subOpTier2LoadNil

	// subOpTier2Return returns from the current function with operand C values. The only
	// operand (the return-value count) sits in C of the tier-2 encoding.
	subOpTier2Return
)

const (
	// subOpTier3Nop is the zero-operand no-op.
	//
	// Encoded as the all-drill instruction {opDrillTier1, subOpDrillTier2,
	// subOpTier2DrillTier3, subOpTier3Nop} - bit pattern {0, 0, 0, 0}. Lives at iota=0 of
	// the tier-3 enum because tier 3 has no further drill marker (no tier 4 to descend into)
	// so the most-zero-encoded slot is free for the most-trivial op. The flat-dispatch macro
	// fast-paths this encoding inline (TESTL+JZ on amd64, CBZ on arm64) without ever
	// reaching the dispatch table.
	subOpTier3Nop subOpcodeTier3 = iota

	// subOpTier3ReturnVoid returns from the current function with no values. Lives in tier 3
	// because the op consumes no register operands.
	subOpTier3ReturnVoid
)

var (
	// opcodeNames maps each opcode to its string representation for debugging.
	//
	// Slot 0 is opDrillTier1 (the tier-1 drill marker); the opNop alias maps to the same
	// slot, so a literal opNop entry would collide. Disassembly of an instruction whose
	// opcode byte is 0 displays either "DRILL_TIER1" (raw decode) or, for the canonical
	// no-op pattern {0,0,0,0}, "3:TIER3_NOP" via instructionDisplayName.
	opcodeNames = [opcodeCount]string{
		opDrillTier1:            "DRILL_TIER1",
		opExt:                   "EXT",
		opLoadIntConst:          "LOAD_INT_CONST",
		opLoadFloatConst:        "LOAD_FLOAT_CONST",
		opAddInt:                "ADD_INT",
		opSubInt:                "SUB_INT",
		opMulInt:                "MUL_INT",
		opDivInt:                "DIV_INT",
		opRemInt:                "REM_INT",
		opBitAnd:                "BIT_AND",
		opBitOr:                 "BIT_OR",
		opBitXor:                "BIT_XOR",
		opBitAndNot:             "BIT_AND_NOT",
		opShiftLeft:             "SHIFT_LEFT",
		opShiftRight:            "SHIFT_RIGHT",
		opAddFloat:              "ADD_FLOAT",
		opSubFloat:              "SUB_FLOAT",
		opMulFloat:              "MUL_FLOAT",
		opDivFloat:              "DIV_FLOAT",
		opEqInt:                 "EQ_INT",
		opNeInt:                 "NE_INT",
		opLtInt:                 "LT_INT",
		opLeInt:                 "LE_INT",
		opGtInt:                 "GT_INT",
		opGeInt:                 "GE_INT",
		opEqFloat:               "EQ_FLOAT",
		opNeFloat:               "NE_FLOAT",
		opLtFloat:               "LT_FLOAT",
		opLeFloat:               "LE_FLOAT",
		opGtFloat:               "GT_FLOAT",
		opGeFloat:               "GE_FLOAT",
		opJumpIfTrue:            "JUMP_IF_TRUE",
		opJumpIfFalse:           "JUMP_IF_FALSE",
		opCall:                  "CALL",
		opCallScalar:            "CALL_SCALAR",
		opCallMethodInlineable:  "CALL_METHOD_INLINEABLE",
		opAppendByteFastInPlace: "APPEND_BYTE_FAST_INPLACE",
		opAppendInPlace:         "APPEND_INPLACE",
		opAppendSpreadInPlace:   "APPEND_SPREAD_INPLACE",
		opTailCall:              "TAIL_CALL",
		opSubIntConst:           "SUB_INT_CONST",
		opAddIntConst:           "ADD_INT_CONST",
		opLeIntConstJumpFalse:   "LE_INT_CONST_JUMP_FALSE",
		opLtIntConstJumpFalse:   "LT_INT_CONST_JUMP_FALSE",
		opEqIntConstJumpFalse:   "EQ_INT_CONST_JUMP_FALSE",
		opEqIntConstJumpTrue:    "EQ_INT_CONST_JUMP_TRUE",
		opGeIntConstJumpFalse:   "GE_INT_CONST_JUMP_FALSE",
		opGtIntConstJumpFalse:   "GT_INT_CONST_JUMP_FALSE",
		opLtIntJumpFalse:        "LT_INT_JUMP_FALSE",
		opLeIntJumpFalse:        "LE_INT_JUMP_FALSE",
		opGtIntJumpFalse:        "GT_INT_JUMP_FALSE",
		opGeIntJumpFalse:        "GE_INT_JUMP_FALSE",
		opEqIntJumpFalse:        "EQ_INT_JUMP_FALSE",
		opNeIntJumpFalse:        "NE_INT_JUMP_FALSE",
		opEqInterfaceNil:        "EQ_INTERFACE_NIL",
		opNeInterfaceNil:        "NE_INTERFACE_NIL",

		opMapIndexOkJumpIfFalseIntInt:        "MAP_INDEX_OK_JUMP_IF_FALSE_INT_INT",
		opMapIndexOkJumpIfFalseStringInt:     "MAP_INDEX_OK_JUMP_IF_FALSE_STRING_INT",
		opMapIndexOkJumpIfFalseStringString:  "MAP_INDEX_OK_JUMP_IF_FALSE_STRING_STRING",
		opMapIndexOkJumpIfFalseIntString:     "MAP_INDEX_OK_JUMP_IF_FALSE_INT_STRING",
		opMapIndexOkJumpIfFalseIntGeneral:    "MAP_INDEX_OK_JUMP_IF_FALSE_INT_GENERAL",
		opMapIndexOkJumpIfFalseStringGeneral: "MAP_INDEX_OK_JUMP_IF_FALSE_STRING_GENERAL",
		opSwapStructFieldsGeneralT0:          "SWAP_STRUCT_FIELDS_GENERAL_T0",
		opGetStructFieldRawPointerT0:         "GET_STRUCT_FIELD_RAW_POINTER_T0",
		opSliceGetIntDirectUnchecked:         "SLICE_GET_INT_DIRECT_UNCHECKED",
		opSliceSetIntDirectUnchecked:         "SLICE_SET_INT_DIRECT_UNCHECKED",
		opStringIndexUnchecked:               "STRING_INDEX_UNCHECKED",
		opStringIndexToIntUnchecked:          "STRING_INDEX_TO_INT_UNCHECKED",
		opSliceGetIntUnchecked:               "SLICE_GET_INT_UNCHECKED",
		opSliceSetIntUnchecked:               "SLICE_SET_INT_UNCHECKED",
		opMulIntConst:                        "MUL_INT_CONST",
		opAddIntJump:                         "ADD_INT_JUMP",
		opMoveGeneral:                        "MOVE_GENERAL",
		opLoadStringConst:                    "LOAD_STRING_CONST",
		opLoadGeneralConst:                   "LOAD_GENERAL_CONST",
		opLoadBoolConst:                      "LOAD_BOOL_CONST",
		opLoadUintConst:                      "LOAD_UINT_CONST",
		opLoadComplexConst:                   "LOAD_COMPLEX_CONST",
		opAddUint:                            "ADD_UINT",
		opSubUint:                            "SUB_UINT",
		opMulUint:                            "MUL_UINT",
		opDivUint:                            "DIV_UINT",
		opRemUint:                            "REM_UINT",
		opBitAndUint:                         "BIT_AND_UINT",
		opBitOrUint:                          "BIT_OR_UINT",
		opBitXorUint:                         "BIT_XOR_UINT",
		opBitAndNotUint:                      "BIT_AND_NOT_UINT",
		opShiftLeftUint:                      "SHIFT_LEFT_UINT",
		opShiftRightUint:                     "SHIFT_RIGHT_UINT",
		opEqUint:                             "EQ_UINT",
		opNeUint:                             "NE_UINT",
		opLtUint:                             "LT_UINT",
		opLeUint:                             "LE_UINT",
		opGtUint:                             "GT_UINT",
		opGeUint:                             "GE_UINT",
		opAddComplex:                         "ADD_COMPLEX",
		opSubComplex:                         "SUB_COMPLEX",
		opMulComplex:                         "MUL_COMPLEX",
		opDivComplex:                         "DIV_COMPLEX",
		opEqComplex:                          "EQ_COMPLEX",
		opNeComplex:                          "NE_COMPLEX",
		opBuildComplex:                       "BUILD_COMPLEX",
		opConcatString:                       "CONCAT_STRING",
		opStringIndex:                        "STRING_INDEX",
		opSliceString:                        "SLICE_STRING",
		opConcatRuneString:                   "CONCAT_RUNE_STRING",
		opEqString:                           "EQ_STRING",
		opNeString:                           "NE_STRING",
		opLtString:                           "LT_STRING",
		opLeString:                           "LE_STRING",
		opGtString:                           "GT_STRING",
		opGeString:                           "GE_STRING",
		opEqGeneral:                          "EQ_GENERAL",
		opNeGeneral:                          "NE_GENERAL",
		opLtGeneral:                          "LT_GENERAL",
		opLeGeneral:                          "LE_GENERAL",
		opGtGeneral:                          "GT_GENERAL",
		opGeGeneral:                          "GE_GENERAL",
		opAdd:                                "ADD",
		opSub:                                "SUB",
		opMul:                                "MUL",
		opDiv:                                "DIV",
		opRem:                                "REM",
		opTruncateNarrow:                     "TRUNCATE_NARROW",
		opPackInterface:                      "PACK_INTERFACE",
		opUnpackInterface:                    "UNPACK_INTERFACE",
		opTestNilJumpTrue:                    "TEST_NIL_JUMP_TRUE",
		opTestNilJumpFalse:                   "TEST_NIL_JUMP_FALSE",
		opEqStringConstJumpFalse:             "EQ_STRING_CONST_JUMP_FALSE",
		opCallNative:                         "CALL_NATIVE",
		opCallBuiltin:                        "CALL_BUILTIN",
		opCallMethod:                         "CALL_METHOD",
		opCallIIFE:                           "CALL_IIFE",
		opMakeClosure:                        "MAKE_CLOSURE",
		opGetUpvalue:                         "GET_UPVALUE",
		opSetUpvalue:                         "SET_UPVALUE",
		opSyncClosureUpvalues:                "SYNC_CLOSURE_UPVALUES",
		opResetSharedCell:                    "RESET_SHARED_CELL",
		opWriteSharedCell:                    "WRITE_SHARED_CELL",
		opDefer:                              "DEFER",
		opGo:                                 "GO",
		opMakeSlice:                          "MAKE_SLICE",
		opIndex:                              "INDEX",
		opIndexSet:                           "INDEX_SET",
		opSliceOp:                            "SLICE_OP",
		opMapIndex:                           "MAP_INDEX",
		opMapSet:                             "MAP_SET",
		opMapIndexOk:                         "MAP_INDEX_OK",
		opAppend:                             "APPEND",
		opCopy:                               "COPY",
		opSliceGetInt:                        "SLICE_GET_INT",
		opSliceSetInt:                        "SLICE_SET_INT",
		opSliceGetFloat:                      "SLICE_GET_FLOAT",
		opSliceSetFloat:                      "SLICE_SET_FLOAT",
		opSliceGetString:                     "SLICE_GET_STRING",
		opSliceSetString:                     "SLICE_SET_STRING",
		opSliceGetBool:                       "SLICE_GET_BOOL",
		opSliceSetBool:                       "SLICE_SET_BOOL",
		opSliceGetUint:                       "SLICE_GET_UINT",
		opSliceSetUint:                       "SLICE_SET_UINT",
		opMapGetIntInt:                       "MAP_GET_INT_INT",
		opMapSetIntInt:                       "MAP_SET_INT_INT",
		opMapGetStringInt:                    "MAP_GET_STRING_INT",
		opMapSetStringInt:                    "MAP_SET_STRING_INT",
		opMapGetStringString:                 "MAP_GET_STRING_STRING",
		opMapSetStringString:                 "MAP_SET_STRING_STRING",
		opMapGetIntString:                    "MAP_GET_INT_STRING",
		opMapSetIntString:                    "MAP_SET_INT_STRING",
		opMapIndexOkIntInt:                   "MAP_INDEX_OK_INT_INT",
		opMapIndexOkStringInt:                "MAP_INDEX_OK_STRING_INT",
		opMapIndexOkStringString:             "MAP_INDEX_OK_STRING_STRING",
		opMapIndexOkIntString:                "MAP_INDEX_OK_INT_STRING",
		opMapGetIntGeneral:                   "MAP_GET_INT_GENERAL",
		opMapIndexOkIntGeneral:               "MAP_INDEX_OK_INT_GENERAL",
		opMapGetStringGeneral:                "MAP_GET_STRING_GENERAL",
		opMapIndexOkStringGeneral:            "MAP_INDEX_OK_STRING_GENERAL",
		opMapSetStringGeneral:                "MAP_SET_STRING_GENERAL",
		opMapAddIntInt:                       "MAP_ADD_INT_INT",
		opMapAddStringInt:                    "MAP_ADD_STRING_INT",
		opGetField:                           "GET_FIELD",
		opSetField:                           "SET_FIELD",
		opSetFieldInt:                        "SET_FIELD_INT",
		opGetFieldInt:                        "GET_FIELD_INT",
		opBindMethod:                         "BIND_METHOD",
		opChannelSend:                        "CHAN_SEND",
		opSelect:                             "SELECT",
		opRangeInit:                          "RANGE_INIT",
		opRangeNext:                          "RANGE_NEXT",
		opAddr:                               "ADDR",
		opDeref:                              "DEREF",
		opAllocIndirect:                      "ALLOC_INDIRECT",
		opTypeAssert:                         "TYPE_ASSERT",
		opConvert:                            "CONVERT",
		opGetGlobal:                          "GET_GLOBAL",
		opSetGlobal:                          "SET_GLOBAL",
		opUnsafeString:                       "UNSAFE_STRING",
		opUnsafeSlice:                        "UNSAFE_SLICE",
		opUnsafeAdd:                          "UNSAFE_ADD",
		opStrContainsRune:                    "STR_CONTAINS_RUNE",
		opStrContains:                        "STR_CONTAINS",
		opStrHasPrefix:                       "STR_HAS_PREFIX",
		opStrHasSuffix:                       "STR_HAS_SUFFIX",
		opStrEqualFold:                       "STR_EQUAL_FOLD",
		opStrIndex:                           "STR_INDEX",
		opStrCount:                           "STR_COUNT",
		opStrTrimPrefix:                      "STR_TRIM_PREFIX",
		opStrTrimSuffix:                      "STR_TRIM_SUFFIX",
		opStrTrim:                            "STR_TRIM",
		opStrIndexRune:                       "STR_INDEX_RUNE",
		opStrRepeat:                          "STR_REPEAT",
		opStrLastIndex:                       "STR_LAST_INDEX",
		opStrJoin:                            "STR_JOIN",
		opStrSplit:                           "STR_SPLIT",
		opStrReplaceAll:                      "STR_REPLACE_ALL",
		opMathPow:                            "MATH_POW",
		opSliceGetIntDirect:                  "SLICE_GET_INT_DIRECT",
		opSliceSetIntDirect:                  "SLICE_SET_INT_DIRECT",
		opRangeNextSliceInt:                  "RANGE_NEXT_SLICE_INT",
		opGetStructFieldIntT0:                "GET_STRUCT_FIELD_INT_T0",
		opSetStructFieldIntT0:                "SET_STRUCT_FIELD_INT_T0",
		opGetStructFieldUint:                 "GET_STRUCT_FIELD_UINT_T0",
		opSetStructFieldUint:                 "SET_STRUCT_FIELD_UINT_T0",
		opGetStructFieldFloat:                "GET_STRUCT_FIELD_FLOAT_T0",
		opSetStructFieldFloat:                "SET_STRUCT_FIELD_FLOAT_T0",
		opGetStructFieldBool:                 "GET_STRUCT_FIELD_BOOL_T0",
		opSetStructFieldBool:                 "SET_STRUCT_FIELD_BOOL_T0",
		opGetStructFieldGeneral:              "GET_STRUCT_FIELD_GENERAL_T0",
		opSetStructFieldGeneral:              "SET_STRUCT_FIELD_GENERAL_T0",
		opCopyStructFieldGeneralT0:           "COPY_STRUCT_FIELD_GENERAL_T0",
		opSliceIndexStructFieldInt:           "SLICE_INDEX_STRUCT_FIELD_INT",
		opSliceIndexStructFieldUint:          "SLICE_INDEX_STRUCT_FIELD_UINT",
		opSliceIndexStructFieldFloat:         "SLICE_INDEX_STRUCT_FIELD_FLOAT",
		opSliceIndexStructFieldBool:          "SLICE_INDEX_STRUCT_FIELD_BOOL",
		opSliceIndexStructFieldString:        "SLICE_INDEX_STRUCT_FIELD_STRING",
		opGetGlobalWide:                      "GET_GLOBAL_WIDE",
		opSetGlobalWide:                      "SET_GLOBAL_WIDE",
		opStringIndexToInt:                   "STRING_INDEX_TO_INT",
		opPackTyped:                          "PACK_TYPED",
	}
)

// String returns the human-readable name of the opcode.
//
// Returns the mnemonic string, or "UNKNOWN" for unregistered opcodes.
func (op opcode) String() string {
	if int(op) < len(opcodeNames) && opcodeNames[op] != "" {
		return opcodeNames[op]
	}
	return "UNKNOWN"
}

var (
	// subOpcodeNames maps each tier 1 sub-opcode to its mnemonic.
	//
	// The zero-valued slot at the start of the enum is reserved for subOpDrillTier2 (the
	// descent into tier 2) so disassembly of a {opDrillTier1, 0, 0, 0} instruction shows
	// "1:DRILL_TIER2" rather than a misleading sub-op name.
	subOpcodeNames = [...]string{
		subOpDrillTier2:                "DRILL_TIER2",
		subOpMathSin:                   "MATH_SIN",
		subOpMathCos:                   "MATH_COS",
		subOpMathExp:                   "MATH_EXP",
		subOpMathTan:                   "MATH_TAN",
		subOpMathMod:                   "MATH_MOD",
		subOpStrconvFormatBool:         "STRCONV_FORMAT_BOOL",
		subOpStrconvFormatInt:          "STRCONV_FORMAT_INT",
		subOpStrconvItoa:               "STRCONV_ITOA",
		subOpRealComplex:               "REAL_COMPLEX",
		subOpImagComplex:               "IMAG_COMPLEX",
		subOpBytesToString:             "BYTES_TO_STRING",
		subOpMakeMethodExpr:            "MAKE_METHOD_EXPR",
		subOpCap:                       "CAP",
		subOpNegComplex:                "NEG_COMPLEX",
		subOpMoveComplex:               "MOVE_COMPLEX",
		subOpMakeSliceInt:              "MAKE_SLICE_INT",
		subOpLenSliceIntDirect:         "LEN_SLICE_INT_DIRECT",
		subOpMakeSliceFloat:            "MAKE_SLICE_FLOAT",
		subOpSliceGetFloatDirect:       "SLICE_GET_FLOAT_DIRECT",
		subOpSliceSetFloatDirect:       "SLICE_SET_FLOAT_DIRECT",
		subOpLenSliceFloatDirect:       "LEN_SLICE_FLOAT_DIRECT",
		subOpMakeSliceString:           "MAKE_SLICE_STRING",
		subOpSliceGetStringDirect:      "SLICE_GET_STRING_DIRECT",
		subOpSliceSetStringDirect:      "SLICE_SET_STRING_DIRECT",
		subOpLenSliceStringDirect:      "LEN_SLICE_STRING_DIRECT",
		subOpMakeSliceBool:             "MAKE_SLICE_BOOL",
		subOpSliceGetBoolDirect:        "SLICE_GET_BOOL_DIRECT",
		subOpSliceSetBoolDirect:        "SLICE_SET_BOOL_DIRECT",
		subOpLenSliceBoolDirect:        "LEN_SLICE_BOOL_DIRECT",
		subOpMakeSliceUint:             "MAKE_SLICE_UINT",
		subOpSliceGetUintDirect:        "SLICE_GET_UINT_DIRECT",
		subOpSliceSetUintDirect:        "SLICE_SET_UINT_DIRECT",
		subOpLenSliceUintDirect:        "LEN_SLICE_UINT_DIRECT",
		subOpBoxSliceInt:               "BOX_SLICE_INT",
		subOpUnboxSliceInt:             "UNBOX_SLICE_INT",
		subOpMoveInt:                   "MOVE_INT",
		subOpMoveFloat:                 "MOVE_FLOAT",
		subOpMoveString:                "MOVE_STRING",
		subOpMoveBool:                  "MOVE_BOOL",
		subOpMoveUint:                  "MOVE_UINT",
		subOpMoveIntToGeneral:          "MOVE_INT_TO_GENERAL",
		subOpMoveGeneralToInt:          "MOVE_GENERAL_TO_INT",
		subOpMoveFloatToGeneral:        "MOVE_FLOAT_TO_GENERAL",
		subOpMoveGeneralToFloat:        "MOVE_GENERAL_TO_FLOAT",
		subOpMoveStringToGeneral:       "MOVE_STRING_TO_GENERAL",
		subOpMoveGeneralToString:       "MOVE_GENERAL_TO_STRING",
		subOpNegInt:                    "NEG_INT",
		subOpNegFloat:                  "NEG_FLOAT",
		subOpBitNot:                    "BIT_NOT",
		subOpBitNotUint:                "BIT_NOT_UINT",
		subOpIntToFloat:                "INT_TO_FLOAT",
		subOpFloatToInt:                "FLOAT_TO_INT",
		subOpNot:                       "NOT",
		subOpBoolToInt:                 "BOOL_TO_INT",
		subOpIntToBool:                 "INT_TO_BOOL",
		subOpIntToUint:                 "INT_TO_UINT",
		subOpUintToInt:                 "UINT_TO_INT",
		subOpUintToFloat:               "UINT_TO_FLOAT",
		subOpFloatToUint:               "FLOAT_TO_UINT",
		subOpMathSqrt:                  "MATH_SQRT",
		subOpMathAbs:                   "MATH_ABS",
		subOpMathFloor:                 "MATH_FLOOR",
		subOpMathCeil:                  "MATH_CEIL",
		subOpMathTrunc:                 "MATH_TRUNC",
		subOpMathRound:                 "MATH_ROUND",
		subOpLenString:                 "LEN_STRING",
		subOpRuneToString:              "RUNE_TO_STRING",
		subOpStrToUpper:                "STR_TO_UPPER",
		subOpStrToLower:                "STR_TO_LOWER",
		subOpStrTrimSpace:              "STR_TRIM_SPACE",
		subOpLen:                       "LEN",
		subOpStringToBytes:             "STRING_TO_BYTES",
		subOpUnsafeStringData:          "UNSAFE_STRING_DATA",
		subOpUnsafeSliceData:           "UNSAFE_SLICE_DATA",
		subOpJump:                      "JUMP",
		subOpLoadIntConstSmall:         "LOAD_INT_CONST_SMALL",
		subOpLoadBool:                  "LOAD_BOOL",
		subOpIncIntJumpLt:              "INC_INT_JUMP_LT",
		subOpLenStringLtJumpFalse:      "LEN_STRING_LT_JUMP_FALSE",
		subOpLoadZero:                  "LOAD_ZERO",
		subOpMakeMap:                   "MAKE_MAP",
		subOpMakeChannel:               "MAKE_CHANNEL",
		subOpMapDelete:                 "MAP_DELETE",
		subOpChannelReceive:            "CHANNEL_RECEIVE",
		subOpGetMethod:                 "GET_METHOD",
		subOpSpill:                     "SPILL",
		subOpReload:                    "RELOAD",
		subOpGetStructFieldInt:         "GET_STRUCT_FIELD_INT",
		subOpGetStructFieldUint:        "GET_STRUCT_FIELD_UINT",
		subOpGetStructFieldFloat:       "GET_STRUCT_FIELD_FLOAT",
		subOpGetStructFieldBool:        "GET_STRUCT_FIELD_BOOL",
		subOpGetStructFieldString:      "GET_STRUCT_FIELD_STRING",
		subOpSetStructFieldInt:         "SET_STRUCT_FIELD_INT",
		subOpSetStructFieldUint:        "SET_STRUCT_FIELD_UINT",
		subOpSetStructFieldFloat:       "SET_STRUCT_FIELD_FLOAT",
		subOpSetStructFieldBool:        "SET_STRUCT_FIELD_BOOL",
		subOpSetStructFieldString:      "SET_STRUCT_FIELD_STRING",
		subOpGetStructFieldSliceInt:    "GET_STRUCT_FIELD_SLICE_INT",
		subOpGetStructFieldSliceFloat:  "GET_STRUCT_FIELD_SLICE_FLOAT",
		subOpGetStructFieldSliceUint:   "GET_STRUCT_FIELD_SLICE_UINT",
		subOpGetStructFieldSliceString: "GET_STRUCT_FIELD_SLICE_STRING",
		subOpGetStructFieldSliceBool:   "GET_STRUCT_FIELD_SLICE_BOOL",
		subOpGetStructFieldSliceByte:   "GET_STRUCT_FIELD_SLICE_BYTE",
		subOpSetStructFieldSliceInt:    "SET_STRUCT_FIELD_SLICE_INT",
		subOpSetStructFieldSliceFloat:  "SET_STRUCT_FIELD_SLICE_FLOAT",
		subOpSetStructFieldSliceUint:   "SET_STRUCT_FIELD_SLICE_UINT",
		subOpSetStructFieldSliceString: "SET_STRUCT_FIELD_SLICE_STRING",
		subOpSetStructFieldSliceBool:   "SET_STRUCT_FIELD_SLICE_BOOL",
		subOpSetStructFieldSliceByte:   "SET_STRUCT_FIELD_SLICE_BYTE",
		subOpCapSliceIntDirect:         "CAP_SLICE_INT_DIRECT",
		subOpCapSliceFloatDirect:       "CAP_SLICE_FLOAT_DIRECT",
		subOpCapSliceStringDirect:      "CAP_SLICE_STRING_DIRECT",
		subOpCapSliceBoolDirect:        "CAP_SLICE_BOOL_DIRECT",
		subOpCapSliceUintDirect:        "CAP_SLICE_UINT_DIRECT",
		subOpCapSliceByteDirect:        "CAP_SLICE_BYTE_DIRECT",
		subOpAddUintConst:              "ADD_UINT_CONST",
		subOpSubUintConst:              "SUB_UINT_CONST",
		subOpBitAndUintConst:           "BIT_AND_UINT_CONST",
		subOpLoadUintConstSmall:        "LOAD_UINT_CONST_SMALL",
		subOpAppendUint:                "APPEND_UINT",
		subOpAppendInt:                 "APPEND_INT",
		subOpAppendString:              "APPEND_STRING",
		subOpAppendFloat:               "APPEND_FLOAT",
		subOpAppendBool:                "APPEND_BOOL",
		subOpMakeSliceByte:             "MAKE_SLICE_BYTE",
		subOpSliceGetByteDirect:        "SLICE_GET_BYTE_DIRECT",
		subOpSliceSetByteDirect:        "SLICE_SET_BYTE_DIRECT",
		subOpLenSliceByteDirect:        "LEN_SLICE_BYTE_DIRECT",
		subOpSliceByteSlice:            "SLICE_BYTE_SLICE",
		subOpRangeNextSliceByte:        "RANGE_NEXT_SLICE_BYTE",
		subOpBoxSliceByte:              "BOX_SLICE_BYTE",
		subOpUnboxSliceByte:            "UNBOX_SLICE_BYTE",
		subOpSliceByteToString:         "SLICE_BYTE_TO_STRING",
		subOpRangeCheckUintJumpFalse:   "RANGE_CHECK_UINT_JUMP_FALSE",
		subOpEqUintConstJumpFalse:      "EQ_UINT_CONST_JUMP_FALSE",

		// SIMD kernel sub-opcodes - see opcode.go for the operand layout per sub-op.
		subOpSimdDotProductFloat64:               "SIMD_DOT_PRODUCT_FLOAT64",
		subOpSimdSumSliceFloat64:                 "SIMD_SUM_SLICE_FLOAT64",
		subOpSimdNormSquaredFloat64:              "SIMD_NORM_SQUARED_FLOAT64",
		subOpSimdEuclideanDistanceSquaredFloat64: "SIMD_EUCLIDEAN_DISTANCE_SQUARED_FLOAT64",
		subOpSimdMaxSliceFloat64:                 "SIMD_MAX_SLICE_FLOAT64",
		subOpSimdMinSliceFloat64:                 "SIMD_MIN_SLICE_FLOAT64",
		subOpSimdAddSliceFloat64:                 "SIMD_ADD_SLICE_FLOAT64",
		subOpSimdSubSliceFloat64:                 "SIMD_SUB_SLICE_FLOAT64",
		subOpSimdMulSliceFloat64:                 "SIMD_MUL_SLICE_FLOAT64",
		subOpSimdAxpyFloat64:                     "SIMD_AXPY_FLOAT64",
		subOpSimdScaleSliceFloat64:               "SIMD_SCALE_SLICE_FLOAT64",
		subOpSimdClearSliceFloat64:               "SIMD_CLEAR_SLICE_FLOAT64",
		subOpSimdFillSliceFloat64:                "SIMD_FILL_SLICE_FLOAT64",
		subOpAdoptGeneralToSlicesFloat:           "ADOPT_GENERAL_TO_SLICES_FLOAT",
		subOpAdoptGeneralToSlicesInt:             "ADOPT_GENERAL_TO_SLICES_INT",
		subOpAdoptGeneralToSlicesString:          "ADOPT_GENERAL_TO_SLICES_STRING",
		subOpAdoptGeneralToSlicesBool:            "ADOPT_GENERAL_TO_SLICES_BOOL",
		subOpAdoptGeneralToSlicesUint:            "ADOPT_GENERAL_TO_SLICES_UINT",
		subOpAdoptGeneralToSlicesByte:            "ADOPT_GENERAL_TO_SLICES_BYTE",
		subOpBoxSliceFloat:                       "BOX_SLICE_FLOAT",
		subOpBoxSliceString:                      "BOX_SLICE_STRING",
		subOpBoxSliceBool:                        "BOX_SLICE_BOOL",
		subOpBoxSliceUint:                        "BOX_SLICE_UINT",
		subOpMoveSliceInt:                        "MOVE_SLICE_INT",
		subOpMoveSliceFloat:                      "MOVE_SLICE_FLOAT",
		subOpMoveSliceString:                     "MOVE_SLICE_STRING",
		subOpMoveSliceBool:                       "MOVE_SLICE_BOOL",
		subOpMoveSliceUint:                       "MOVE_SLICE_UINT",
		subOpMoveSliceByte:                       "MOVE_SLICE_BYTE",
		subOpAppendSliceIntDirect:                "APPEND_SLICE_INT_DIRECT",
		subOpAppendSliceFloatDirect:              "APPEND_SLICE_FLOAT_DIRECT",
		subOpAppendSliceStringDirect:             "APPEND_SLICE_STRING_DIRECT",
		subOpAppendSliceBoolDirect:               "APPEND_SLICE_BOOL_DIRECT",
		subOpAppendSliceUintDirect:               "APPEND_SLICE_UINT_DIRECT",
		subOpAppendSliceByteDirect:               "APPEND_SLICE_BYTE_DIRECT",
		subOpSliceSliceIntDirect:                 "SLICE_SLICE_INT_DIRECT",
		subOpSliceSliceFloatDirect:               "SLICE_SLICE_FLOAT_DIRECT",
		subOpSliceSliceStringDirect:              "SLICE_SLICE_STRING_DIRECT",
		subOpSliceSliceBoolDirect:                "SLICE_SLICE_BOOL_DIRECT",
		subOpSliceSliceUintDirect:                "SLICE_SLICE_UINT_DIRECT",
		subOpCopySliceIntDirect:                  "COPY_SLICE_INT_DIRECT",
		subOpCopySliceFloatDirect:                "COPY_SLICE_FLOAT_DIRECT",
		subOpCopySliceStringDirect:               "COPY_SLICE_STRING_DIRECT",
		subOpCopySliceBoolDirect:                 "COPY_SLICE_BOOL_DIRECT",
		subOpCopySliceUintDirect:                 "COPY_SLICE_UINT_DIRECT",
		subOpCopySliceByteDirect:                 "COPY_SLICE_BYTE_DIRECT",
		subOpAppendUintInPlace:                   "APPEND_UINT_INPLACE",
		subOpAppendByteSpreadInPlace:             "APPEND_BYTE_SPREAD_INPLACE",
	}
)

// String returns the mnemonic for a tier 1 sub-opcode dispatched by opDrillTier1.
// Sub-opcodes without a name in subOpcodeNames render as "UNKNOWN_SUBOP" so disassembly
// never silently produces a blank label.
//
// Returns the mnemonic string, or "UNKNOWN_SUBOP" for unregistered sub-opcodes.
func (op subOpcode) String() string {
	if int(op) < len(subOpcodeNames) && subOpcodeNames[op] != "" {
		return subOpcodeNames[op]
	}
	return "UNKNOWN_SUBOP"
}

// subOpcodeTier2 identifies a tier-2 body for a doubly-drilled instruction.
//
// Tier 2 holds 1-operand ops: operand A holds the tier-1 drill marker (subOpDrillTier2),
// operand B holds the tier-2 sub-opcode discriminator, and operand C carries the single
// register operand. New 1-operand ops opt into this tier rather than consuming a
// 3-operand tier-0 slot or a 2-operand tier-1 slot, keeping the capped tier-0 space
// reserved for ops that genuinely need three operands. Lives in its own iota block so the
// tier-2 and tier-1 enumerations grow independently.
type subOpcodeTier2 uint8

// subOpcodeTier3 identifies which tier-3 body a fully-drilled instruction dispatches to.
//
// Tier 3 holds 0-operand ops: operands A and B carry drill markers (subOpDrillTier2 and
// subOpTier2DrillTier3), and the actual op is encoded in operand C. Tier 3 has no further
// drill marker because there is no tier 4; iota=0 of tier 3 is therefore the natural slot
// for opNop, since the all-zero word {0, 0, 0, 0} encodes "do nothing" by descending
// every tier. Lives in its own iota block.
type subOpcodeTier3 uint8

var (
	// subOpcodeTier2Names maps each tier-2 sub-opcode to its mnemonic.
	subOpcodeTier2Names = [...]string{
		subOpTier2DrillTier3:   "DRILL_TIER3",
		subOpTier2IncInt:       "TIER2_INC_INT",
		subOpTier2DecInt:       "TIER2_DEC_INT",
		subOpTier2IncUint:      "TIER2_INC_UINT",
		subOpTier2DecUint:      "TIER2_DEC_UINT",
		subOpTier2Panic:        "TIER2_PANIC",
		subOpTier2Recover:      "TIER2_RECOVER",
		subOpTier2SetZero:      "TIER2_SET_ZERO",
		subOpTier2ChannelClose: "TIER2_CHANNEL_CLOSE",
		subOpTier2LoadNil:      "TIER2_LOAD_NIL",
		subOpTier2Return:       "TIER2_RETURN",
	}
)

// String returns the mnemonic for a tier-2 sub-opcode.
//
// Returns the mnemonic string, or "UNKNOWN_TIER2" for unregistered sub-opcodes.
func (op subOpcodeTier2) String() string {
	if int(op) < len(subOpcodeTier2Names) && subOpcodeTier2Names[op] != "" {
		return subOpcodeTier2Names[op]
	}
	return "UNKNOWN_TIER2"
}

var (
	// subOpcodeTier3Names maps each tier-3 sub-opcode to its mnemonic.
	subOpcodeTier3Names = [...]string{
		subOpTier3Nop:        "TIER3_NOP",
		subOpTier3ReturnVoid: "TIER3_RETURN_VOID",
	}
)

// String returns the mnemonic for a tier-3 sub-opcode.
//
// Returns the mnemonic string, or "UNKNOWN_TIER3" for unregistered sub-opcodes.
func (op subOpcodeTier3) String() string {
	if int(op) < len(subOpcodeTier3Names) && subOpcodeTier3Names[op] != "" {
		return subOpcodeTier3Names[op]
	}
	return "UNKNOWN_TIER3"
}

// instructionDisplayName returns the disassembly label for instr.
//
// Every label is prefixed with the dispatch tier the handler lives at: "0:" for direct
// opcodes that the main dispatch table resolves, "1:" for tier-1 sub-ops dispatched via
// opDrillTier1, "2:" for tier-2 sub-ops reached when operand A is subOpDrillTier2, and
// "3:" for tier-3 sub-ops reached when operands A and B both drill.
//
// Takes instr (instruction) which is the encoded instruction word.
//
// Returns the display label for instr.
func instructionDisplayName(instr instruction) string {
	if instr.op != opDrillTier1 {
		return "0:" + instr.op.String()
	}
	tier1 := subOpcode(instr.a)
	if tier1 != subOpDrillTier2 {
		return "1:" + tier1.String()
	}
	tier2 := subOpcodeTier2(instr.b)
	if tier2 != subOpTier2DrillTier3 {
		return "2:" + tier2.String()
	}
	tier3 := subOpcodeTier3(instr.c)
	return "3:" + tier3.String()
}
