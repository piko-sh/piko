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
	"reflect"
	"sync"

	"piko.sh/piko/wdk/safeconv"
)

// CompiledFunction holds the bytecode and metadata for a single function.
type CompiledFunction struct {
	// methodTable maps "TypeName.MethodName" to a function index in
	// functions. Used for runtime dispatch of interface method calls
	// where the concrete type is unknown at compile time.
	methodTable map[string]uint16

	// typeNames maps reflect.Type to the source-level type name for
	// types created via reflect.StructOf (which have an empty Name()).
	// Used by handleCallMethod to resolve method table keys at runtime.
	typeNames map[reflect.Type]string

	// asmCallInfoTables caches the pre-computed asmCallInfo tables for this
	// function when used as the root of an Execute() call. Built once
	// on first execution, then reused - the tables are deterministic
	// for a given compiled function and its function table.
	asmCallInfoTables map[*CompiledFunction][]asmCallInfo

	// intConstIndex accelerates constant deduplication during compilation.
	// Nil at runtime after optimise() releases it.
	intConstIndex map[int64]uint16

	// floatConstIndex accelerates constant deduplication during compilation.
	// Nil at runtime after optimise() releases it.
	floatConstIndex map[float64]uint16

	// stringConstIndex accelerates constant deduplication during compilation.
	// Nil at runtime after optimise() releases it.
	stringConstIndex map[string]uint16

	// uintConstIndex accelerates constant deduplication during compilation.
	// Nil at runtime after optimise() releases it.
	uintConstIndex map[uint64]uint16

	// typeRefIndex accelerates type table deduplication during compilation.
	// Nil at runtime after optimise() releases it.
	typeRefIndex map[reflect.Type]uint16

	// variableInitFunction holds bytecode for package-level variable initialisers.
	// When non-nil, Execute runs it before the main body to reset
	// globals to their declared values on each invocation.
	variableInitFunction *CompiledFunction

	// name is the function's qualified name (e.g., "main.BuildAST").
	name string

	// sourceFile is the source file path for error reporting.
	sourceFile string

	// paramKinds maps each parameter position to its register kind.
	// Parameters occupy the first N registers in their respective banks.
	paramKinds []registerKind

	// upvalueDescriptors describes captured variables for closures. Each
	// entry tells the VM how to initialise an upvalue when creating
	// a closure from the function.
	upvalueDescriptors []UpvalueDescriptor

	// boolConstants holds bool constants referenced by opLoadBoolConst.
	boolConstants []bool

	// uintConstants holds uint64 constants referenced by opLoadUintConst.
	uintConstants []uint64

	// complexConstants holds complex128 constants referenced by opLoadComplexConst.
	complexConstants []complex128

	// generalConstants holds reflect.Value constants referenced by
	// opLoadGeneralConst. These include type values, function values,
	// and complex constants.
	generalConstants []reflect.Value

	// floatConstants holds float64 constants referenced by opLoadFloatConst.
	floatConstants []float64

	// resultKinds maps each return value position to its register kind.
	resultKinds []registerKind

	// namedResultLocs holds the register locations of named return
	// values, if any. Used by bare return statements to copy named
	// result variables to return positions.
	namedResultLocs []varLocation

	// stringConstants holds string constants referenced by opLoadStringConst.
	stringConstants []string

	// functions holds nested function literals (closures) defined
	// within the enclosing function. Referenced by opMakeClosure via index.
	functions []*CompiledFunction

	// callSites describes each function call in the bytecode. opCall
	// references call sites by index.
	callSites []callSite

	// typeTable holds reflect.Type values referenced by type operation
	// instructions (opTypeAssert, opConvert, opMakeSlice, etc.).
	typeTable []reflect.Type

	// generalConstantDescriptors records how each generalConstants
	// entry was created so it can be reconstructed from a serialised
	// representation.
	generalConstantDescriptors []generalConstantDescriptor

	// typeTableDescriptors records a serialisable description of each
	// typeTable entry for bytecode serialisation.
	typeTableDescriptors []typeDescriptor

	// intConstants holds int64 constants referenced by opLoadIntConst.
	intConstants []int64

	// debugSourceMap maps program counter offsets to source file
	// positions. Nil when debug info is disabled.
	debugSourceMap *sourceMap

	// debugVarTable holds variable debug information (names,
	// liveness ranges). Nil when debug info is disabled.
	debugVarTable *debugVarTable

	// debugEmitHook is called after each instruction is emitted
	// during compilation, used by the compiler to record source
	// positions.
	debugEmitHook func(pc int)

	// body is the bytecode instruction sequence.
	body []instruction

	// asmCallInfoTablesOnce guards one-time computation of asmCallInfoTables.
	asmCallInfoTablesOnce sync.Once

	// numRegisters tracks peak register usage per bank [int, float,
	// string, general, bool, uint, complex]. Used to allocate the
	// register file for each call frame.
	numRegisters [NumRegisterKinds]uint32

	// isVariadic is true when the last parameter is variadic (...T).
	isVariadic bool
}

// FuncName returns the function's qualified name.
//
// Returns string which is the fully qualified function name.
func (cf *CompiledFunction) FuncName() string { return cf.name }

// BodyLen returns the number of bytecode instructions.
//
// Returns int which is the instruction count in the body.
func (cf *CompiledFunction) BodyLen() int { return len(cf.body) }

// SubFunctions returns the nested function literals defined within the function.
//
// Returns []*CompiledFunction which holds the closure definitions nested inside the
// function.
func (cf *CompiledFunction) SubFunctions() []*CompiledFunction { return cf.functions }

// RegisterCounts returns the peak register usage per bank.
//
// Returns [NumRegisterKinds]uint32 which holds the maximum
// register index used in each register bank.
func (cf *CompiledFunction) RegisterCounts() [NumRegisterKinds]uint32 { return cf.numRegisters }

// reflectFuncType returns the reflect.Type for the method's signature excluding the
// receiver parameter. Used for creating bound method values via reflect.MakeFunc.
//
// Returns the function type and true if the function has parameters,
// or nil and false otherwise.
func (cf *CompiledFunction) reflectFuncType() (reflect.Type, bool) {
	if len(cf.paramKinds) == 0 {
		return nil, false
	}
	paramKinds := cf.paramKinds[1:]
	inTypes := make([]reflect.Type, len(paramKinds))
	for i, k := range paramKinds {
		inTypes[i] = kindDefaultReflectType(k)
	}
	outTypes := make([]reflect.Type, len(cf.resultKinds))
	for i, k := range cf.resultKinds {
		outTypes[i] = kindDefaultReflectType(k)
	}
	return reflect.FuncOf(inTypes, outTypes, cf.isVariadic), true
}

// reflectMethodExprType returns the reflect.Type for the method's signature
// including the receiver as the first parameter. General-register params use
// interface{} so concrete struct types can pass through reflect.Call without type
// mismatches.
//
// Returns the function type and true if the function has parameters,
// or nil and false otherwise.
func (cf *CompiledFunction) reflectMethodExprType() (reflect.Type, bool) {
	if len(cf.paramKinds) == 0 {
		return nil, false
	}
	inTypes := make([]reflect.Type, len(cf.paramKinds))
	for i, k := range cf.paramKinds {
		if k == registerGeneral {
			inTypes[i] = reflect.TypeFor[any]()
		} else {
			inTypes[i] = kindDefaultReflectType(k)
		}
	}
	outTypes := make([]reflect.Type, len(cf.resultKinds))
	for i, k := range cf.resultKinds {
		outTypes[i] = kindDefaultReflectType(k)
	}
	return reflect.FuncOf(inTypes, outTypes, cf.isVariadic), true
}

// UpvalueDescriptor describes a single captured variable in a closure.
type UpvalueDescriptor struct {
	// index is the register index in the enclosing scope, or the
	// upvalue index if isLocal is false (transitive capture).
	index uint8

	// kind is the register bank of the captured variable.
	kind registerKind

	// isLocal is true when the upvalue captures a register directly
	// from the immediately enclosing function. When false, the
	// upvalue is captured transitively from the enclosing function's
	// own upvalue table.
	isLocal bool
}

// callSite describes a function call in the compiled bytecode. It
// stores all the information the VM needs to set up arguments and
// retrieve return values without additional opcodes.
type callSite struct {
	// nativeFastPath caches the extracted function value for native
	// call sites dispatched via handleCallNative. nil = unprobed,
	// nativeFastPathNone = no fast-path match, any other value =
	// cached fn.Interface() result for direct type-asserted calls
	// bypassing reflect.Value.Call().
	nativeFastPath any

	// arguments records where each argument lives in the CALLER's frame.
	arguments []varLocation

	// returns records where to put each return value in the CALLER's
	// frame after the call completes.
	returns []varLocation

	// paramTypes caches the expected parameter types for native calls.
	// Populated lazily on first call to avoid repeated fn.Type().In(i).
	paramTypes []reflect.Type

	// argumentsBuffer is a pre-allocated buffer for building the argument
	// slice for native calls. Avoids make([]reflect.Value, n) per call.
	argumentsBuffer []reflect.Value

	// variadicArgumentsBuffer is a pre-allocated buffer for variadic fast-path
	// calls. Avoids make([]any, n) per call for functions like
	// fmt.Sprintf that take ...interface{} parameters.
	variadicArgumentsBuffer []any

	// tailArgsBuf is a pre-allocated buffer for tail call argument
	// snapshots. Avoids make([]tailCallArg, n) per tail call.
	tailArgsBuf []tailCallArg

	// linkedTypeArgs holds the instantiated type arguments for a
	// //piko:link-routed generic call, resolved at compile time from
	// types.Info.Instances. When non-empty the native call handler
	// prepends one reflect.Type value per element before the regular
	// arguments and skips the nativeFastPath / paramTypes caches.
	linkedTypeArgs []reflect.Type

	// cachedRecvAddr stores the address of the receiver value when
	// nativeFastPath was last populated, only used when isMethod is
	// true. A mismatch means the receiver changed and the cached
	// bound method must be refreshed.
	cachedRecvAddr uintptr

	// funcIndex is the index into the enclosing function's functions
	// slice for the callee. Ignored when isClosure is true.
	funcIndex uint16

	// closureRegister is the general register holding the closure value.
	// Only used when isClosure is true.
	closureRegister uint8

	// nativeRegister is the general register holding the native function
	// value. Only used when isNative is true.
	nativeRegister uint8

	// isClosure is true when the callee is a closure stored in a
	// general register rather than a static function reference.
	isClosure bool

	// isNative is true when the callee is a native Go function stored
	// in a general register (not a compiled function).
	isNative bool

	// isMethod is true when the callee is a bound method obtained
	// via opGetMethod, where handleCallNative validates the cached
	// fast path against the current receiver address before reuse.
	isMethod bool

	// methodRecvReg is the general register holding the receiver
	// for method calls (only valid when isMethod is true). Used
	// to validate the cached fast path by comparing the receiver
	// address across invocations.
	methodRecvReg uint8

	// nativeFastPathTag identifies which fast-path case matched, allowing
	// subsequent calls to dispatch via a uint8 jump table instead
	// of the full interface type switch.
	nativeFastPathTag nativeFastPathTag
}

// addCallSite adds a call site and returns its index.
//
// Takes site (CallSite) which specifies the call site descriptor
// to append.
//
// Returns the index of the newly added call site.
func (cf *CompiledFunction) addCallSite(site callSite) uint16 {
	index := len(cf.callSites)
	cf.callSites = append(cf.callSites, site)
	return safeconv.MustIntToUint16(index)
}

// addIntConstant adds an int64 constant and returns its index.
//
// Takes v (int64) which specifies the constant value to add or
// look up.
//
// Returns the index of the constant in the IntConstants pool.
func (cf *CompiledFunction) addIntConstant(v int64) uint16 {
	if index, ok := cf.intConstIndex[v]; ok {
		return index
	}
	index := safeconv.MustIntToUint16(len(cf.intConstants))
	cf.intConstants = append(cf.intConstants, v)
	if cf.intConstIndex == nil {
		cf.intConstIndex = make(map[int64]uint16)
	}
	cf.intConstIndex[v] = index
	return index
}

// addFloatConstant adds a float64 constant and returns its index.
//
// Takes v (float64) which specifies the constant value to add or
// look up.
//
// Returns the index of the constant in the FloatConstants pool.
func (cf *CompiledFunction) addFloatConstant(v float64) uint16 {
	if index, ok := cf.floatConstIndex[v]; ok {
		return index
	}
	index := safeconv.MustIntToUint16(len(cf.floatConstants))
	cf.floatConstants = append(cf.floatConstants, v)
	if cf.floatConstIndex == nil {
		cf.floatConstIndex = make(map[float64]uint16)
	}
	cf.floatConstIndex[v] = index
	return index
}

// addStringConstant adds a string constant and returns its index.
//
// Takes v (string) which specifies the constant value to add or
// look up.
//
// Returns the index of the constant in the StringConstants pool.
func (cf *CompiledFunction) addStringConstant(v string) uint16 {
	if index, ok := cf.stringConstIndex[v]; ok {
		return index
	}
	index := safeconv.MustIntToUint16(len(cf.stringConstants))
	cf.stringConstants = append(cf.stringConstants, v)
	if cf.stringConstIndex == nil {
		cf.stringConstIndex = make(map[string]uint16)
	}
	cf.stringConstIndex[v] = index
	return index
}

// addGeneralConstant adds a reflect.Value constant with its
// reconstruction descriptor and returns its index.
//
// Takes v (reflect.Value) which specifies the constant value to
// append.
// Takes descriptor (generalConstantDescriptor) which records how to
// reconstruct the value from a serialised form.
//
// Returns the index of the constant in the GeneralConstants pool.
func (cf *CompiledFunction) addGeneralConstant(v reflect.Value, descriptor generalConstantDescriptor) uint16 {
	index := len(cf.generalConstants)
	cf.generalConstants = append(cf.generalConstants, v)
	cf.generalConstantDescriptors = append(cf.generalConstantDescriptors, descriptor)
	return safeconv.MustIntToUint16(index)
}

// addBoolConstant adds a bool constant and returns its index.
//
// Takes v (bool) which specifies the constant value to add or
// look up.
//
// Returns the index of the constant in the BoolConstants pool.
func (cf *CompiledFunction) addBoolConstant(v bool) uint16 {
	for i, c := range cf.boolConstants {
		if c == v {
			return safeconv.MustIntToUint16(i)
		}
	}
	index := len(cf.boolConstants)
	cf.boolConstants = append(cf.boolConstants, v)
	return safeconv.MustIntToUint16(index)
}

// addUintConstant adds a uint64 constant and returns its index.
//
// Takes v (uint64) which specifies the constant value to add or
// look up.
//
// Returns the index of the constant in the UintConstants pool.
func (cf *CompiledFunction) addUintConstant(v uint64) uint16 {
	if index, ok := cf.uintConstIndex[v]; ok {
		return index
	}
	index := safeconv.MustIntToUint16(len(cf.uintConstants))
	cf.uintConstants = append(cf.uintConstants, v)
	if cf.uintConstIndex == nil {
		cf.uintConstIndex = make(map[uint64]uint16)
	}
	cf.uintConstIndex[v] = index
	return index
}

// addComplexConstant adds a complex128 constant and returns its index.
//
// Takes v (complex128) which specifies the constant value to add
// or look up.
//
// Returns the index of the constant in the ComplexConstants pool.
func (cf *CompiledFunction) addComplexConstant(v complex128) uint16 {
	for i, c := range cf.complexConstants {
		if c == v {
			return safeconv.MustIntToUint16(i)
		}
	}
	index := len(cf.complexConstants)
	cf.complexConstants = append(cf.complexConstants, v)
	return safeconv.MustIntToUint16(index)
}

// addTypeRef adds a reflect.Type to the type table and returns its index.
//
// Takes t (reflect.Type) which specifies the type to add or look
// up.
//
// Returns the index of the type in the TypeTable.
func (cf *CompiledFunction) addTypeRef(t reflect.Type) uint16 {
	if index, ok := cf.typeRefIndex[t]; ok {
		return index
	}
	index := safeconv.MustIntToUint16(len(cf.typeTable))
	cf.typeTable = append(cf.typeTable, t)
	cf.typeTableDescriptors = append(cf.typeTableDescriptors, reflectTypeToDescriptor(t))
	if cf.typeRefIndex == nil {
		cf.typeRefIndex = make(map[reflect.Type]uint16)
	}
	cf.typeRefIndex[t] = index
	return index
}

// emit appends an instruction to the function body and returns its
// offset for later patching.
//
// Takes op (opcode) which specifies the opcode.
// Takes a (uint8) which specifies the first instruction operand.
// Takes b (uint8) which specifies the second instruction operand.
// Takes c (uint8) which specifies the third instruction operand.
//
// Returns the instruction offset in the body.
func (cf *CompiledFunction) emit(op opcode, a, b, c uint8) int {
	pc := len(cf.body)
	cf.body = append(cf.body, makeInstruction(op, a, b, c))
	if cf.debugEmitHook != nil {
		cf.debugEmitHook(pc)
	}
	return pc
}

// emitWide emits an instruction with a 16-bit wide index split
// across B and C operands.
//
// Takes op (opcode) which specifies the opcode.
// Takes a (uint8) which specifies the first operand.
// Takes wide (uint16) which specifies the 16-bit index to encode
// in B (low byte) and C (high byte).
//
// Returns the instruction offset in the body.
func (cf *CompiledFunction) emitWide(op opcode, a uint8, wide uint16) int {
	lo, hi := splitWide(wide)
	return cf.emit(op, a, lo, hi)
}

// emitExtension emits an opExt instruction with a 16-bit payload
// in A (low byte) and B (high byte), plus an extra byte in C.
//
// Takes wide (uint16) which specifies the 16-bit payload.
// Takes c (uint8) which specifies the extra operand in C.
//
// Returns the instruction offset in the body.
func (cf *CompiledFunction) emitExtension(wide uint16, c uint8) int {
	lo, hi := splitWide(wide)
	return cf.emit(opExt, lo, hi, c)
}

// emitJump emits a jump instruction with a placeholder offset.
//
// Takes op (opcode) which specifies the jump opcode.
// Takes condReg (uint8) which specifies the condition register
// index.
//
// Returns the instruction offset for later patching with patchJump.
func (cf *CompiledFunction) emitJump(op opcode, condReg uint8) int {
	return cf.emit(op, condReg, 0, 0)
}

// patchJump patches a previously emitted jump instruction at patchPC
// to jump to the current instruction offset.
//
// Takes patchPC (int) which specifies the instruction offset to
// patch.
func (cf *CompiledFunction) patchJump(patchPC int) {
	target := len(cf.body)
	offset := safeconv.MustIntToInt16(target - patchPC - 1)
	lo, hi := splitOffset(offset)
	cf.body[patchPC].b = lo
	cf.body[patchPC].c = hi
}

// currentPC returns the offset where the next instruction will be emitted.
//
// Returns the current program counter offset.
func (cf *CompiledFunction) currentPC() int {
	return len(cf.body)
}

// optimise applies peephole optimisations to the instruction body,
// fusing common instruction sequences into superinstructions. Dead
// slots are replaced with opNop to preserve instruction indices
// (jump offsets reference indices, not byte offsets).
func (cf *CompiledFunction) optimise() {
	body := cf.body
	n := len(body)
	jumpTargets := cf.buildJumpTargets(body)

	for i := range n {
		if cf.fuseThreeInstrPatterns(body, i, n, jumpTargets) ||
			cf.fuseArithConst(body, i, n, jumpTargets) ||
			cf.fuseAddIntJump(body, i, n, jumpTargets) ||
			cf.fuseConcatRune(body, i, n, jumpTargets) ||
			cf.fuseAppendMove(body, i, n, jumpTargets) ||
			cf.fuseStringIndexToInt(body, i, n, jumpTargets) {
			continue
		}
		cf.optimiseLoadIntConst(body, i)
	}

	cf.intConstIndex = nil
	cf.floatConstIndex = nil
	cf.stringConstIndex = nil
	cf.uintConstIndex = nil
	cf.typeRefIndex = nil
	cf.debugEmitHook = nil

	cf.syncSourceMapAfterOptimise(body)

	for _, child := range cf.functions {
		child.optimise()
	}
}

// buildJumpTargets pre-computes the set of instruction indices that are
// jump destinations, enabling O(1) lookup during pattern matching.
//
// Takes body ([]instruction) which specifies the instruction
// sequence to scan.
//
// Returns a map of instruction indices that are jump targets.
func (*CompiledFunction) buildJumpTargets(body []instruction) map[int]bool {
	jumpTargets := make(map[int]bool, len(body)/4)
	for i, instr := range body {
		switch instr.op {
		case opJump, opJumpIfTrue, opJumpIfFalse:
			offset := instr.signedOffset()
			jumpTargets[i+1+int(offset)] = true
		}
	}
	return jumpTargets
}

// fuseThreeInstrPatterns dispatches all 3-instruction fusion patterns.
// Must run before 2-instruction patterns to avoid partial consumption.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of
// protected jump destinations.
//
// Returns true if a pattern was matched and applied.
func (cf *CompiledFunction) fuseThreeInstrPatterns(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	return cf.fuseCompareConstJump(body, i, n, jumpTargets, opLeInt, opJumpIfFalse, opLeIntConstJumpFalse) ||
		cf.fuseCompareConstJump(body, i, n, jumpTargets, opLtInt, opJumpIfFalse, opLtIntConstJumpFalse) ||
		cf.fuseCompareConstJump(body, i, n, jumpTargets, opEqInt, opJumpIfFalse, opEqIntConstJumpFalse) ||
		cf.fuseCompareConstJump(body, i, n, jumpTargets, opEqInt, opJumpIfTrue, opEqIntConstJumpTrue) ||
		cf.fuseCompareConstJump(body, i, n, jumpTargets, opGeInt, opJumpIfFalse, opGeIntConstJumpFalse) ||
		cf.fuseCompareConstJump(body, i, n, jumpTargets, opGtInt, opJumpIfFalse, opGtIntConstJumpFalse) ||
		cf.fuseStringConstJump(body, i, n, jumpTargets) ||
		cf.fuseNilTestJump(body, i, n, jumpTargets) ||
		cf.fuseIncIntJumpLt(body, i, n, jumpTargets) ||
		cf.fuseLenStringLtJump(body, i, n, jumpTargets)
}

// fuseArithConst fuses LoadIntConst + arithmetic op (AddInt, SubInt,
// MulInt) into the corresponding constant-operand superinstruction.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of
// protected jump destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseArithConst(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n ||
		body[i].op != opLoadIntConst || body[i].c != 0 ||
		jumpTargets[i+1] {
		return false
	}
	next := body[i+1]
	if body[i].a != next.c {
		return false
	}
	var fusedOp opcode
	switch next.op {
	case opSubInt:
		fusedOp = opSubIntConst
	case opAddInt:
		fusedOp = opAddIntConst
	case opMulInt:
		fusedOp = opMulIntConst
	default:
		return false
	}
	body[i] = makeInstruction(fusedOp, next.a, next.b, body[i].b)
	body[i+1] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseAddIntJump fuses AddIntConst + Jump into opAddIntJump + opExt.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of
// protected jump destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseAddIntJump(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n ||
		body[i].op != opAddIntConst || body[i+1].op != opJump ||
		jumpTargets[i+1] {
		return false
	}
	raw := body[i+1].signedOffset()
	adj := raw + 1
	lo, hi := splitOffset(adj)
	body[i] = makeInstruction(opAddIntJump, body[i].a, body[i].b, body[i].c)
	body[i+1] = makeInstruction(opExt, lo, hi, 0)
	return true
}

// fuseConcatRune fuses RuneToString + ConcatString into opConcatRuneString.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of
// protected jump destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseConcatRune(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n ||
		body[i].op != opRuneToString || body[i+1].op != opConcatString ||
		body[i].a != body[i+1].c ||
		jumpTargets[i+1] {
		return false
	}
	body[i] = makeInstruction(opConcatRuneString, body[i+1].a, body[i+1].b, body[i].b)
	body[i+1] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseAppendMove fuses APPEND_xxx + MOVE_GENERAL into an in-place
// APPEND_xxx where A == B. This enables the handler to use
// Grow/SetLen/Index.Set on addressable slice values instead of
// allocating via reflect.ValueOf on every append.
//
// Pattern:
//
//	APPEND_INT  R_new, R_src, R_elem   ->  APPEND_INT  R_src, R_src, R_elem
//	MOVE_GEN    R_src, R_new           ->  NOP
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of
// protected jump destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseAppendMove(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n || jumpTargets[i+1] {
		return false
	}
	appendInstr := body[i]
	moveInstr := body[i+1]

	switch appendInstr.op {
	case opAppendInt, opAppendString, opAppendFloat, opAppendBool, opAppend:
	default:
		return false
	}

	if moveInstr.op != opMoveGeneral {
		return false
	}

	if moveInstr.a != appendInstr.b || moveInstr.b != appendInstr.a {
		return false
	}

	body[i] = makeInstruction(appendInstr.op, appendInstr.b, appendInstr.b, appendInstr.c)
	body[i+1] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// optimiseLoadIntConst rewrites LoadIntConst to LoadIntConstSmall when
// the constant value fits in [0, 255].
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current instruction index.
func (cf *CompiledFunction) optimiseLoadIntConst(body []instruction, i int) {
	if body[i].op != opLoadIntConst || body[i].c != 0 {
		return
	}
	value := cf.intConstants[body[i].b]
	if value >= 0 && value <= maxSmallConstant {
		body[i] = makeInstruction(opLoadIntConstSmall, body[i].a, safeconv.MustIntToUint8(int(value)), 0)
	}
}

// fuseCompareConstJump attempts to fuse a LoadIntConst + cmpOp +
// jumpOp sequence into a single superinstruction.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of
// protected jump destinations.
// Takes cmpOp (opcode) which specifies the comparison opcode.
// Takes jumpOp (opcode) which specifies the jump opcode.
// Takes fusedOp (opcode) which specifies the fused opcode.
//
// Returns true if the pattern matched and was applied.
func (*CompiledFunction) fuseCompareConstJump(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
	cmpOp, jumpOp, fusedOp opcode,
) bool {
	if i+2 >= n ||
		body[i].op != opLoadIntConst || body[i+1].op != cmpOp ||
		body[i+2].op != jumpOp ||
		body[i].a != body[i+1].c ||
		body[i+1].a != body[i+2].a ||
		body[i].c != 0 ||
		jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	raw := body[i+2].signedOffset()
	adj := raw + 1
	lo, hi := splitOffset(adj)
	body[i] = makeInstruction(fusedOp, body[i+1].b, body[i].b, 0)
	body[i+1] = makeInstruction(opExt, lo, hi, 0)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseIncIntJumpLt fuses IncInt(A) + LtInt(cmp, A, B) + JumpIfTrue(cmp, off)
// into opIncIntJumpLt(A, B, 0) + opExt(lo, hi, 0) + opNop.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of
// protected jump destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseIncIntJumpLt(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+2 >= n ||
		body[i].op != opIncInt || body[i+1].op != opLtInt ||
		body[i+2].op != opJumpIfTrue ||
		body[i+1].b != body[i].a ||
		body[i+2].a != body[i+1].a ||
		jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	raw := body[i+2].signedOffset()
	adj := raw + 1
	lo, hi := splitOffset(adj)
	body[i] = makeInstruction(opIncIntJumpLt, body[i].a, body[i+1].c, 0)
	body[i+1] = makeInstruction(opExt, lo, hi, 0)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseStringIndexToInt fuses opStringIndex + opUintToInt into
// opStringIndexToInt, avoiding the intermediate uint register and
// one tier-2 trampoline per string byte access converted to int.
//
// Pattern:
//
//	STRING_INDEX  R_uint, R_str, R_idx
//	UINT_TO_INT   R_int,  R_uint, 0
//	-> STRING_INDEX_TO_INT  R_int, R_str, R_idx
//	  NOP
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of
// protected jump destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseStringIndexToInt(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+1 >= n ||
		body[i].op != opStringIndex || body[i+1].op != opUintToInt ||
		body[i].a != body[i+1].b ||
		jumpTargets[i+1] {
		return false
	}
	body[i] = makeInstruction(opStringIndexToInt, body[i+1].a, body[i].b, body[i].c)
	body[i+1] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseLenStringLtJump fuses opLenString + opLtInt + opJumpIfFalse into
// opLenStringLtJumpFalse + opExt, collapsing the entire for-loop
// condition `i < len(s)` into a single fused instruction.
//
// Pattern:
//
//	LEN_STRING    R_len, R_str, 0
//	LT_INT        R_bool, R_i, R_len
//	JUMP_IF_FALSE R_bool, lo, hi
//	-> LEN_STRING_LT_JUMP_FALSE  R_i, R_str, 0
//	  EXT                        lo,  hi,    0
//	  NOP
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of
// protected jump destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseLenStringLtJump(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+2 >= n ||
		body[i].op != opLenString || body[i+1].op != opLtInt ||
		body[i+2].op != opJumpIfFalse ||
		body[i].a != body[i+1].c ||
		body[i+1].a != body[i+2].a ||
		jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	raw := body[i+2].signedOffset()
	adj := raw + 1
	lo, hi := splitOffset(adj)
	body[i] = makeInstruction(opLenStringLtJumpFalse, body[i+1].b, body[i].b, 0)
	body[i+1] = makeInstruction(opExt, lo, hi, 0)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseNilTestJump fuses LoadNil(R) + EqGeneral/NeGeneral(C, X, R) + Jump
// into opTestNilJumpTrue/False(X, lo, hi) + opNop + opNop.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of
// protected jump destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseNilTestJump(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+2 >= n ||
		body[i].op != opLoadNil ||
		(body[i+1].op != opEqGeneral && body[i+1].op != opNeGeneral) ||
		(body[i+2].op != opJumpIfTrue && body[i+2].op != opJumpIfFalse) ||
		body[i+1].a != body[i+2].a ||
		jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	nilReg := body[i].a
	var testReg uint8
	if body[i+1].b == nilReg {
		testReg = body[i+1].c
	} else if body[i+1].c == nilReg {
		testReg = body[i+1].b
	} else {
		return false
	}
	wantNilJump := (body[i+1].op == opEqGeneral && body[i+2].op == opJumpIfTrue) ||
		(body[i+1].op == opNeGeneral && body[i+2].op == opJumpIfFalse)
	raw := body[i+2].signedOffset()
	adj := raw + 2
	lo, hi := splitOffset(adj)
	var fusedOp opcode
	if wantNilJump {
		fusedOp = opTestNilJumpTrue
	} else {
		fusedOp = opTestNilJumpFalse
	}
	body[i] = makeInstruction(fusedOp, testReg, lo, hi)
	body[i+1] = makeInstruction(opNop, 0, 0, 0)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// fuseStringConstJump fuses LoadStringConst(R, index) + EqString(C, X, R) +
// JumpIfFalse(C, off) into opEqStringConstJumpFalse(X, index, 0) +
// opExt(lo, hi, 0) + opNop.
//
// Takes body ([]instruction) which specifies the instruction sequence.
// Takes i (int) which specifies the current index.
// Takes n (int) which specifies the length.
// Takes jumpTargets (map[int]bool) which specifies the set of
// protected jump destinations.
//
// Returns true if a pattern was matched and applied.
func (*CompiledFunction) fuseStringConstJump(
	body []instruction, i, n int,
	jumpTargets map[int]bool,
) bool {
	if i+2 >= n ||
		body[i].op != opLoadStringConst || body[i+1].op != opEqString ||
		body[i+2].op != opJumpIfFalse ||
		body[i].c != 0 ||
		body[i].a != body[i+1].c ||
		body[i+1].a != body[i+2].a ||
		jumpTargets[i+1] || jumpTargets[i+2] {
		return false
	}
	raw := body[i+2].signedOffset()
	adj := raw + 1
	lo, hi := splitOffset(adj)
	body[i] = makeInstruction(opEqStringConstJumpFalse, body[i+1].b, body[i].b, 0)
	body[i+1] = makeInstruction(opExt, lo, hi, 0)
	body[i+2] = makeInstruction(opNop, 0, 0, 0)
	return true
}

// syncSourceMapAfterOptimise zeroes source positions for
// instructions that were replaced with NOPs during optimisation,
// keeping the source map consistent with the instruction body.
//
// Takes body ([]instruction) which specifies the instruction
// sequence to synchronise.
func (cf *CompiledFunction) syncSourceMapAfterOptimise(body []instruction) {
	if cf.debugSourceMap == nil {
		return
	}
	for i, instr := range body {
		if instr.op == opNop && i < len(cf.debugSourceMap.positions) {
			cf.debugSourceMap.positions[i] = sourcePosition{}
		}
	}
}
