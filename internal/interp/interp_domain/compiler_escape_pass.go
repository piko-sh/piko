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
	"context"
	"fmt"
)

const (
	// maxEscapeFixpointIter caps the fixpoint iteration count as a safety net for
	// pathological call graphs. Typical inputs converge in 2-3 iterations.
	maxEscapeFixpointIter = 6

	// generalRegisterBankSize is the number of slots in a single register bank, used to size
	// the per-bank taint bitmap. The runtime addresses each bank with a single uint8
	// operand, so 256 entries cover the entire bank.
	generalRegisterBankSize = 256
)

var (
	// tier1SafeSubOpTable is the precomputed allowlist of tier-1 sub-ops whose effect on a
	// tainted general register is provably non-escaping.
	//
	// Lookup is a single indexed bool fetch. Populated from the per-category slices below.
	tier1SafeSubOpTable = func() [generalRegisterBankSize]bool {
		var table [generalRegisterBankSize]bool
		for _, sub := range tier1SafeSubOpAllowList {
			table[sub] = true
		}
		return table
	}()

	// tier1SafeSubOpAllowList is the flat union of every category below. Each category
	// groups sub-ops by their escape rationale so the soundness reasoning stays close to the
	// membership decision.
	tier1SafeSubOpAllowList = func() []subOpcode {
		categories := [][]subOpcode{
			tier1SafeMathIntrinsics,
			tier1SafeStringConversion,
			tier1SafeComplexOps,
			tier1SafeTypedSliceMakes,
			tier1SafeTypedSliceLengths,
			tier1SafeTypedSliceGetSet,
			tier1SafeByteSliceOps,
			tier1SafeTypedBankMoves,
			tier1SafeTypedBankUnary,
			tier1SafeStringMisc,
			tier1SafeStringCaseTrim,
			tier1SafeConstantLoadsAndJump,
			tier1SafeFusedJumpArith,
			tier1SafeUintArithConst,
			tier1SafeStructFieldGet,
			tier1SafeStructFieldSet,
			tier1SafeStructFieldIncDec,
			tier1SafeAppendTypedValue,
			tier1SafeStarAppendByte,
			tier1SafeReadOnlyInspection,
			tier1SafeCrossBankMoves,
		}
		var total int
		for _, group := range categories {
			total += len(group)
		}
		out := make([]subOpcode, 0, total)
		for _, group := range categories {
			out = append(out, group...)
		}
		return out
	}()

	// tier1SafeMathIntrinsics carries float-bank-only math intrinsics.
	tier1SafeMathIntrinsics = []subOpcode{
		subOpMathSin, subOpMathCos, subOpMathExp, subOpMathTan, subOpMathMod,
		subOpMathSqrt, subOpMathAbs, subOpMathFloor, subOpMathCeil,
		subOpMathTrunc, subOpMathRound,
	}

	// tier1SafeStringConversion carries int/bool/string-bank string conversion sub-ops.
	tier1SafeStringConversion = []subOpcode{
		subOpStrconvFormatBool, subOpStrconvFormatInt, subOpStrconvItoa,
	}

	// tier1SafeComplexOps carries complex/float-bank complex-number sub-ops.
	tier1SafeComplexOps = []subOpcode{
		subOpRealComplex, subOpImagComplex, subOpNegComplex, subOpMoveComplex,
	}

	// tier1SafeTypedSliceMakes carries typed-slice constructors that write to the
	// typed-slice bank from typed int args.
	tier1SafeTypedSliceMakes = []subOpcode{
		subOpMakeSliceInt, subOpMakeSliceFloat, subOpMakeSliceString,
		subOpMakeSliceBool, subOpMakeSliceUint, subOpMakeSliceByte,
	}

	// tier1SafeTypedSliceLengths carries typed-slice length and capacity sub-ops.
	tier1SafeTypedSliceLengths = []subOpcode{
		subOpLenSliceIntDirect, subOpLenSliceFloatDirect, subOpLenSliceStringDirect,
		subOpLenSliceBoolDirect, subOpLenSliceUintDirect, subOpLenSliceByteDirect,
		subOpCapSliceIntDirect, subOpCapSliceFloatDirect, subOpCapSliceStringDirect,
		subOpCapSliceBoolDirect, subOpCapSliceUintDirect, subOpCapSliceByteDirect,
	}

	// tier1SafeTypedSliceGetSet carries typed-slice get/set sub-ops.
	tier1SafeTypedSliceGetSet = []subOpcode{
		subOpSliceGetFloatDirect, subOpSliceSetFloatDirect,
		subOpSliceGetStringDirect, subOpSliceSetStringDirect,
		subOpSliceGetBoolDirect, subOpSliceSetBoolDirect,
		subOpSliceGetUintDirect, subOpSliceSetUintDirect,
		subOpSliceGetByteDirect, subOpSliceSetByteDirect,
	}

	// tier1SafeByteSliceOps carries byte-slice helpers.
	tier1SafeByteSliceOps = []subOpcode{
		subOpSliceByteSlice, subOpRangeNextSliceByte, subOpSliceByteToString,
	}

	// tier1SafeTypedBankMoves carries typed-bank-only move sub-ops that never touch general
	// registers.
	tier1SafeTypedBankMoves = []subOpcode{
		subOpMoveInt, subOpMoveFloat, subOpMoveString, subOpMoveBool, subOpMoveUint,
	}

	// tier1SafeTypedBankUnary carries typed-bank unary / conversion sub-ops.
	tier1SafeTypedBankUnary = []subOpcode{
		subOpNegInt, subOpNegFloat, subOpBitNot, subOpBitNotUint,
		subOpIntToFloat, subOpFloatToInt, subOpIntToUint, subOpUintToInt,
		subOpUintToFloat, subOpFloatToUint,
		subOpNot, subOpBoolToInt, subOpIntToBool,
	}

	// tier1SafeStringMisc carries string length / construction sub-ops in typed banks.
	tier1SafeStringMisc = []subOpcode{
		subOpLenString, subOpRuneToString,
	}

	// tier1SafeStringCaseTrim carries string case / trim sub-ops in typed banks.
	tier1SafeStringCaseTrim = []subOpcode{
		subOpStrToUpper, subOpStrToLower, subOpStrTrimSpace,
	}

	// tier1SafeConstantLoadsAndJump carries the constant-load and unconditional jump sub-ops
	// (no general-bank effect).
	tier1SafeConstantLoadsAndJump = []subOpcode{
		subOpLoadIntConstSmall, subOpLoadBool, subOpLoadZero,
		subOpLoadUintConstSmall, subOpJump,
	}

	// tier1SafeFusedJumpArith carries the int/uint-bank fused jump-arith sub-ops.
	tier1SafeFusedJumpArith = []subOpcode{
		subOpIncIntJumpLt, subOpLenStringLtJumpFalse,
		subOpEqUintConstJumpFalse, subOpRangeCheckUintJumpFalse,
	}

	// tier1SafeUintArithConst carries uint-bank const-fused arithmetic sub-ops.
	tier1SafeUintArithConst = []subOpcode{
		subOpAddUintConst, subOpSubUintConst, subOpBitAndUintConst,
	}

	// tier1SafeStructFieldGet carries the typed-output struct-field get sub-ops. The
	// receiver pointer is dereferenced for the field load (not propagated); the loaded value
	// lands in a typed bank that cannot carry pointer-ness.
	tier1SafeStructFieldGet = []subOpcode{
		subOpGetStructFieldInt, subOpGetStructFieldUint,
		subOpGetStructFieldFloat, subOpGetStructFieldBool,
		subOpGetStructFieldString,
	}

	// tier1SafeStructFieldSet carries the typed-input struct-field set sub-ops. Same logic
	// as the get variants: the pointer receiver is dereferenced, and the typed value cannot
	// carry pointer-ness.
	tier1SafeStructFieldSet = []subOpcode{
		subOpSetStructFieldInt, subOpSetStructFieldUint,
		subOpSetStructFieldFloat, subOpSetStructFieldBool,
		subOpSetStructFieldString,
	}

	// tier1SafeStructFieldIncDec carries in-place increment/decrement sub-ops through a
	// struct receiver (no value flow at all - the field is both read and written in-place).
	tier1SafeStructFieldIncDec = []subOpcode{
		subOpIncStructFieldInt, subOpDecStructFieldInt,
		subOpIncStructFieldUint, subOpDecStructFieldUint,
	}

	// tier1SafeAppendTypedValue carries append-typed-value sub-ops. The general slice
	// receiver is dereferenced; the typed value cannot carry pointer-ness.
	tier1SafeAppendTypedValue = []subOpcode{
		subOpAppendInt, subOpAppendString, subOpAppendFloat,
		subOpAppendBool, subOpAppendUint,
	}

	// tier1SafeStarAppendByte carries *general[B] = append(*general[B], uints[C]) sub-ops.
	// The pointer is dereferenced (not escaped) and the appended value comes from the uint
	// bank.
	tier1SafeStarAppendByte = []subOpcode{
		subOpStarAppendByteFast, subOpStarAppendByteSpread,
	}

	// tier1SafeReadOnlyInspection carries len/cap/bytes-to-string sub-ops that read general
	// registers without propagating their pointer value.
	tier1SafeReadOnlyInspection = []subOpcode{
		subOpLen, subOpCap, subOpBytesToString,
	}

	// tier1SafeCrossBankMoves carries the unbox/box cross-bank moves. Both are read/value
	// operations that cannot let a tainted general pointer reach a heap-anchored slot.
	tier1SafeCrossBankMoves = []subOpcode{
		subOpMoveGeneralToInt, subOpMoveIntToGeneral,
		subOpMoveGeneralToFloat, subOpMoveFloatToGeneral,
		subOpMoveGeneralToString, subOpMoveStringToGeneral,
	}
)

// runEscapeAnalysisPass computes parameterEscapes for every reachable function and
// arenaSafeAllocPCs for every opAllocIndirect site whose output is provably non-escaping.
//
// Invoked from runPostCompilationChecks after recursion detection and the inliner. The
// inliner does not touch opAllocIndirect emit positions, so the PC-keyed
// arenaSafeAllocPCs remains valid.
//
// Takes ctx (context.Context) which can cancel the pass.
// Takes root (*CompiledFunction) which is the program's top-level compiled function whose
// nested functions are walked.
//
// Returns error when the context is cancelled before completion.
func runEscapeAnalysisPass(ctx context.Context, root *CompiledFunction) error {
	if root == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("runEscapeAnalysisPass cancelled: %w", err)
	}
	all := collectReachableFunctions(root)
	if len(all) == 0 {
		return nil
	}
	initialiseParameterEscapes(all)
	runEscapeFixpoint(all)
	for _, cf := range all {
		annotateArenaSafeAllocs(cf)
	}
	return nil
}

// initialiseParameterEscapes seeds each function's parameterEscapes slice with the
// optimistic "no parameter escapes" baseline so the fixpoint can monotonically demote
// individual parameters to escaping.
//
// Non-general parameter entries remain false; the runtime never consults them.
//
// Takes functions ([]*CompiledFunction) which are the reachable compiled functions to
// initialise.
func initialiseParameterEscapes(functions []*CompiledFunction) {
	for _, cf := range functions {
		if cf.parameterEscapes == nil {
			cf.parameterEscapes = make([]bool, len(cf.parameterKinds))
		}
	}
}

// runEscapeFixpoint runs the bounded monotonic fixpoint that demotes general-bank
// parameters to "escapes" when analyseParameterEscape reports any unmodelled use. Loops
// up to maxEscapeFixpointIter passes.
//
// When the fixpoint does NOT converge within the iteration cap the table is non-exact: a
// parameter could still be marked non-escaping while a deeper transitive call leaks it.
// annotateArenaSafeAllocs would then arena-allocate a value that actually escapes, a
// use-after-free. forceAllParametersEscaping is the conservative fallback, collapsing
// every parameter to "escapes" so no arena allocation can be justified by stale parameter
// information.
//
// Takes functions ([]*CompiledFunction) which are the reachable compiled functions whose
// parameterEscapes are computed.
func runEscapeFixpoint(functions []*CompiledFunction) {
	for range maxEscapeFixpointIter {
		if !classifyEscapingParameters(functions) {
			return
		}
	}
	forceAllParametersEscaping(functions)
}

// forceAllParametersEscaping marks every general-bank parameter of every function as
// escaping.
//
// Reached only when runEscapeFixpoint did not converge. The monotonic lattice flows only
// toward "escapes", so this is a sound over-approximation that forgoes arena allocation;
// correctness is preserved because nothing can then be proven non-escaping on the
// strength of an unconverged table.
//
// Takes functions ([]*CompiledFunction) whose general-bank parameters are forced to
// escaping.
func forceAllParametersEscaping(functions []*CompiledFunction) {
	for _, cf := range functions {
		for paramIdx := range cf.parameterKinds {
			if cf.parameterKinds[paramIdx] != registerGeneral {
				continue
			}
			cf.parameterEscapes[paramIdx] = true
		}
	}
}

// classifyEscapingParameters performs a single fixpoint pass over every general-bank
// parameter of every function.
//
// Takes functions ([]*CompiledFunction) whose parameters are re-classified.
//
// Returns true when at least one parameter flipped from non-escaping to escaping in this
// pass.
func classifyEscapingParameters(functions []*CompiledFunction) bool {
	changed := false
	for _, cf := range functions {
		for paramIdx := range cf.parameterKinds {
			if cf.parameterEscapes[paramIdx] {
				continue
			}
			if cf.parameterKinds[paramIdx] != registerGeneral {
				continue
			}
			if analyseParameterEscape(cf, paramIdx) {
				cf.parameterEscapes[paramIdx] = true
				changed = true
			}
		}
	}
	return changed
}

// paramRegisterSlot returns the register-bank slot index for the paramIdx-th parameter of
// cf.
//
// Parameters are declared in order and declareVar allocates per-bank slots sequentially,
// so the slot for paramIdx is the count of preceding parameters that share its bank.
//
// Takes cf (*CompiledFunction) whose parameter layout is queried.
// Takes paramIdx (int) which is the parameter position in cf.parameterKinds.
//
// Returns the register slot within cf.parameterKinds[paramIdx]'s bank.
func paramRegisterSlot(cf *CompiledFunction, paramIdx int) uint8 {
	kind := cf.parameterKinds[paramIdx]
	slot := uint8(0)
	for j := range paramIdx {
		if cf.parameterKinds[j] == kind {
			slot++
		}
	}
	return slot
}

// generalResultSlots returns the register slots in the general bank that subOpTier2Return
// reads on return.
//
// handleReturn iterates cf.resultKinds and, for each kind, reads the next sequential slot
// from that bank starting at 0; the general-bank slots holding returnable values are
// therefore 0..N-1, where N is the count of resultKinds entries equal to registerGeneral.
//
// Takes cf (*CompiledFunction) whose result kinds drive the slot count.
//
// Returns a slice (length 0 when no general-bank returns) of the register slot indices
// that constitute the return-value reads.
func generalResultSlots(cf *CompiledFunction) []uint8 {
	count := uint8(0)
	for _, kind := range cf.resultKinds {
		if kind == registerGeneral {
			count++
		}
	}
	if count == 0 {
		return nil
	}
	slots := make([]uint8, count)
	for i := range slots {
		slots[i] = uint8(i)
	}
	return slots
}

// analyseParameterEscape returns true when the paramIdx-th parameter (which must be in
// the registerGeneral bank) escapes cf's frame.
//
// Walks cf.body for uses of the parameter's register slot, tracking
// alias-via-opMoveGeneral so the walker does not lose the value when the compiler copies
// it. Uses cf.callSites and the callee's parameterEscapes to evaluate opCall arguments.
//
// Conservative: returns true on any opcode whose effect on the tainted register cannot be
// classified safely. Recognised patterns include deref read/write, recursive call, and
// parameter pass to a known non-escaping callee.
//
// Takes cf (*CompiledFunction) whose body is walked.
// Takes paramIdx (int) which is the index in cf.parameterKinds.
//
// Returns true when paramIdx's pointer escapes; false when every use is safe.
func analyseParameterEscape(cf *CompiledFunction, paramIdx int) bool {
	rootSlot := paramRegisterSlot(cf, paramIdx)
	return analyseGeneralRegisterEscape(cf, rootSlot, 0)
}

// analyseGeneralRegisterEscape returns true when the value initially in general[rootSlot]
// escapes cf's frame.
//
// The tainted set tracks the alias closure: rootSlot plus any general register dst
// observed as the destination of an opMoveGeneral from a tainted source. Reads of tainted
// registers in safe contexts (deref, pass to non-escaping callee param) do not escape;
// any other context conservatively does.
//
// Iteration is per-instruction with stride-independent extension-word skipping: after
// each main instruction the walker advances past any consecutive opExt words.
//
// Takes cf (*CompiledFunction) whose body is walked.
// Takes rootSlot (uint8) which is the initially-tainted general register.
// Takes startPC (int) which is the first PC to inspect.
//
// Returns true on any provable or unprovable escape; false only when every use of every
// tainted register is classifiably safe.
func analyseGeneralRegisterEscape(cf *CompiledFunction, rootSlot uint8, startPC int) bool {
	tainted := [generalRegisterBankSize]bool{}
	tainted[rootSlot] = true

	pc := startPC
	for pc < len(cf.body) {
		inst := cf.body[pc]
		op := inst.op
		if op == opMoveGeneral {
			if tainted[inst.b] {
				tainted[inst.a] = true
			}
			pc = advancePastExtensions(cf, pc)
			continue
		}
		if escapesAtInstruction(cf, inst, &tainted) {
			return true
		}
		pc = advancePastExtensions(cf, pc)
	}
	return false
}

// advancePastExtensions advances pc past the current instruction and any following opExt
// extension words.
//
// Stride-independent: works for any opcode because extension words always carry opExt as
// their primary opcode.
//
// Takes cf (*CompiledFunction) whose body holds the instruction sequence.
// Takes pc (int) which is the current instruction position.
//
// Returns pc+1 at minimum, advanced further past trailing opExt words.
func advancePastExtensions(cf *CompiledFunction, pc int) int {
	pc++
	for pc < len(cf.body) && cf.body[pc].op == opExt {
		pc++
	}
	return pc
}

// escapesAtInstruction returns true when inst uses any tainted register in an
// escape-relevant context.
//
// Per-opcode classification. Unhandled opcodes that read any tainted general-bank
// register default to "escapes". opAllocIndirect with a tainted source means the new
// pointer's pointee holds a copy of the tainted value (heap escape); opAddr produces the
// address of the source's storage so a tainted source becomes a new pointer to it
// (escape); opGetField and opDeref read through the receiver to produce a new value that
// does not alias it (safe); opSetField uses A as the receiver and C as the value being
// stored, so a tainted C is written into a heap-anchored slot (escape).
//
// Takes cf (*CompiledFunction) whose body is walked.
// Takes inst (instruction) which is the current bytecode instruction.
// Takes tainted (*[generalRegisterBankSize]bool) which is the alias-closure derived from
// the parameter.
//
// Returns true on any escape; false when the instruction's effect is classifiably safe
// for the tainted set.
func escapesAtInstruction(cf *CompiledFunction, inst instruction, tainted *[generalRegisterBankSize]bool) bool {
	switch inst.op {
	case opCall:
		return callArgsEscape(cf, inst, tainted)
	case opAllocIndirect, opAddr:
		return tainted[inst.b]
	case opGetField, opDeref, opMoveGeneral:
		return false
	case opSetField:
		return tainted[inst.c]
	case opMakeClosure:
		return makeClosureCaptures(cf, inst, tainted)
	case opDrillTier1:
		return drilledSubOpEscapes(cf, inst, tainted)
	default:
	}
	return escapesAtUnclassifiedOp(inst, tainted)
}

// escapesAtUnclassifiedOp handles every opcode not covered by the explicit switch in
// escapesAtInstruction. When the operand-shape table describes which bytes are
// general-bank reads those bytes are checked directly; otherwise the analysis falls back
// to a conservative "any operand byte may be a tainted general read" rule for
// call/store/return-shaped opcodes.
//
// Takes inst (instruction) which is the current instruction.
// Takes tainted (*[generalRegisterBankSize]bool) which holds the alias closure.
//
// Returns true on any conservatively-detected escape; false when no operand byte reads a
// tainted slot.
func escapesAtUnclassifiedOp(inst instruction, tainted *[generalRegisterBankSize]bool) bool {
	shape := operandShapes[inst.op]
	if shape.flags&shapeFlagDescribed == 0 {
		if (tainted[inst.a] || tainted[inst.b] || tainted[inst.c]) && opcodeMayReadGeneral(inst.op) {
			return true
		}
		return false
	}
	if shape.reads[0] && shape.a == roleRegGeneral && tainted[inst.a] {
		return true
	}
	if shape.reads[1] && shape.b == roleRegGeneral && tainted[inst.b] {
		return true
	}
	if shape.reads[2] && shape.c == roleRegGeneral && tainted[inst.c] {
		return true
	}
	return false
}

// makeClosureCaptures returns true when the opMakeClosure at inst captures any tainted
// general-bank register as a local upvalue.
//
// opMakeClosure encoding: A is the destination general register and B|C encode the wide
// funcIndex into cf.functions. The nested function's upvalueDescriptors list every
// captured variable; entries with isLocal=true reference a slot in cf's register banks.
// For general-bank locals, capture creates a heap-anchored upvalue cell that outlives the
// parent frame, so any tainted pointer captured this way escapes.
//
// Transitive captures (isLocal=false) reference parent-frame upvalues rather than local
// registers, so they cannot taint cf's registers.
//
// Takes cf (*CompiledFunction) whose nested functions list is consulted.
// Takes inst (instruction) which is the opMakeClosure instruction.
// Takes tainted (*[generalRegisterBankSize]bool) which holds the alias closure.
//
// Returns true when any local general-bank capture is tainted.
func makeClosureCaptures(cf *CompiledFunction, inst instruction, tainted *[generalRegisterBankSize]bool) bool {
	funcIdx := int(inst.wideIndex())
	if funcIdx >= len(cf.functions) {
		return true
	}
	nested := cf.functions[funcIdx]
	if nested == nil {
		return true
	}
	for _, uv := range nested.upvalueDescriptors {
		if !uv.isLocal {
			continue
		}
		if uv.kind != registerGeneral {
			continue
		}
		if tainted[uv.index] {
			return true
		}
	}
	return false
}

// drilledSubOpEscapes returns true when the tier-1/2/3 drilled sub-opcode at inst uses
// any tainted register in an escape sink.
//
// opDrillTier1 is the tier wrapper; the actual sub-op is decoded by walking the drill
// chain. When inst.a equals subOpDrillTier2 (=0) the walker descends to tier 2: if inst.b
// equals subOpTier2DrillTier3 (=0) it descends again to tier 3 where the sub-op lives in
// inst.c (e.g. subOpTier3ReturnVoid), otherwise the tier-2 sub-op is in inst.b (e.g.
// subOpTier2Return) with its operand in inst.c. Otherwise the tier-1 sub-op is in inst.a
// with operands in inst.b and inst.c. Conservative default for unrecognised sub-ops: any
// of inst.a/b/c matching a tainted slot is treated as an escape.
//
// Takes cf (*CompiledFunction) whose result kinds drive return-slot detection.
// Takes inst (instruction) which is the opDrillTier1 wrapper.
// Takes tainted (*[generalRegisterBankSize]bool) which holds the alias closure.
//
// Returns true on any sub-op escape; false when the resolved sub-op is classifiably safe
// for the tainted set.
func drilledSubOpEscapes(cf *CompiledFunction, inst instruction, tainted *[generalRegisterBankSize]bool) bool {
	if subOpcode(inst.a) != subOpDrillTier2 {
		return tier1SubOpEscapes(subOpcode(inst.a), inst, tainted)
	}
	tier2 := subOpcodeTier2(inst.b)
	if tier2 != subOpTier2DrillTier3 {
		return tier2SubOpEscapes(cf, tier2, inst, tainted)
	}
	tier3 := subOpcodeTier3(inst.c)
	return tier3SubOpEscapes(cf, tier3, inst, tainted)
}

// tier1SubOpEscapes classifies a tier-1 sub-op.
//
// Most tier-1 sub-ops operate on typed banks (int/float/bool/uint/string/typed-slice) and
// cannot escape a general-bank pointer; the few that touch general either use it as a
// pointer receiver (deref read, in-place modify) or as a value-store sink. Classification
// is driven by tier1SubOpSafe; sub-ops outside the allowlist fall back to the
// conservative "any tainted operand escapes" rule.
//
// Takes sub (subOpcode) which is the tier-1 sub-opcode from inst.a.
// Takes inst (instruction) which is the opDrillTier1 wrapper.
// Takes tainted (*[generalRegisterBankSize]bool) which holds the alias closure.
//
// Returns true on any escape; false for sub-ops in the safe allowlist and for unknown
// sub-ops where no operand byte matches a tainted slot.
func tier1SubOpEscapes(sub subOpcode, inst instruction, tainted *[generalRegisterBankSize]bool) bool {
	if tier1SubOpSafe(sub) {
		return false
	}
	if tainted[inst.b] || tainted[inst.c] {
		return true
	}
	return false
}

// tier1SubOpSafe reports whether the sub-op's effect on a tainted general register is
// provably non-escaping.
//
// The allowlist covers three categories: typed-bank-only ops (math, strconv, conversions,
// typed-slice get/set, typed-bank moves) that cannot read general; general-receiver-deref
// ops (struct field get/set through *T, star-append through *[]byte, len/cap, in-place
// inc/dec) that read general as a pointer receiver without propagating it; and
// safe-typed-value-store ops (append typed value to general slice, set struct field from
// typed bank) where the value side is a typed bank and the receiver side is a deref
// pattern. Sub-ops outside this list are conservatively treated as escape sinks.
//
// Takes sub (subOpcode) which is the tier-1 sub-opcode to classify.
//
// Returns true when the sub-op is in the verified-safe allowlist.
func tier1SubOpSafe(sub subOpcode) bool {
	return tier1SafeSubOpTable[sub]
}

// tier2SubOpEscapes classifies a tier-2 sub-op. subOpTier2Return returns from the
// function by reading per-bank result slots, so any tainted general-bank result slot
// escapes via the return boundary.
//
// Takes cf (*CompiledFunction) whose result kinds drive return-slot reads.
// Takes sub (subOpcodeTier2) which is the tier-2 sub-opcode from inst.b.
// Takes inst (instruction) which is the opDrillTier1 wrapper.
// Takes tainted (*[generalRegisterBankSize]bool) which holds the alias closure.
//
// Returns true on any escape; false for safe tier-2 sub-ops.
func tier2SubOpEscapes(cf *CompiledFunction, sub subOpcodeTier2, inst instruction, tainted *[generalRegisterBankSize]bool) bool {
	if sub == subOpTier2Return {
		for _, slot := range generalResultSlots(cf) {
			if tainted[slot] {
				return true
			}
		}
		return false
	}
	return tainted[inst.c]
}

// tier3SubOpEscapes classifies a tier-3 sub-op. Tier-3 holds zero-operand ops
// (subOpTier3Nop, subOpTier3ReturnVoid) that read no registers, so no tainted value can
// escape through them.
//
// Returns false unconditionally; tier-3 sub-ops have no general-bank effect.
func tier3SubOpEscapes(_ *CompiledFunction, _ subOpcodeTier3, _ instruction, _ *[generalRegisterBankSize]bool) bool {
	return false
}

// callArgsEscape reports whether the opCall at inst passes any tainted register to a
// callee whose corresponding parameter escapes.
//
// Resolves the call site via cf.callSites and uses the callee's parameterEscapes (set by
// a prior fixpoint pass) to decide each argument. When the callee is unknown (closure
// call, native call, dynamic dispatch) the analysis conservatively treats every tainted
// argument as escaping.
//
// Takes cf (*CompiledFunction) whose call sites are consulted.
// Takes inst (instruction) which is the opCall instruction; operands B/C encode the wide
// call-site index.
// Takes tainted (*[generalRegisterBankSize]bool) which is the alias-closure.
//
// Returns true on any escape; false when every tainted argument is bound to a
// non-escaping parameter.
func callArgsEscape(cf *CompiledFunction, inst instruction, tainted *[generalRegisterBankSize]bool) bool {
	siteIdx := int(inst.wideIndex())
	if siteIdx >= len(cf.callSites) {
		return anyGeneralTainted(inst, tainted)
	}
	site := &cf.callSites[siteIdx]
	if site.cachedCallee == nil {
		return anyTaintedGeneralArg(site.arguments, tainted)
	}
	return anyEscapingTaintedArg(site, tainted)
}

// anyTaintedGeneralArg returns true when at least one general-bank argument register is
// in the tainted set. Used when the callee is unknown (closure / native / dynamic
// dispatch) and any passed tainted register is assumed to leak.
//
// Takes arguments ([]varLocation) which are the call-site arguments.
// Takes tainted (*[generalRegisterBankSize]bool) which holds the alias closure.
//
// Returns true on the first tainted general-bank argument.
func anyTaintedGeneralArg(arguments []varLocation, tainted *[generalRegisterBankSize]bool) bool {
	for _, argument := range arguments {
		if argument.kind == registerGeneral && tainted[argument.register] {
			return true
		}
	}
	return false
}

// anyEscapingTaintedArg returns true when at least one tainted argument is bound to a
// callee parameter classified as escaping. Variadic and shape mismatches conservatively
// count as escaping.
//
// Takes site (*callSite) which carries the call-site arguments and the cached callee.
// Takes tainted (*[generalRegisterBankSize]bool) which holds the alias closure.
//
// Returns true on the first tainted argument bound to an escaping parameter, or to an
// unmatched parameter.
func anyEscapingTaintedArg(site *callSite, tainted *[generalRegisterBankSize]bool) bool {
	callee := site.cachedCallee
	for argumentIndex, argument := range site.arguments {
		if argument.kind != registerGeneral || !tainted[argument.register] {
			continue
		}
		if argumentIndex >= len(callee.parameterEscapes) {
			return true
		}
		if callee.parameterEscapes[argumentIndex] {
			return true
		}
	}
	return false
}

// anyGeneralTainted is a fallback used when no per-operand shape info is available and
// the bare a/b/c bytes are tested against the tainted set. Production bytecode always
// carries shape info via the operandShapes table.
//
// Takes inst (instruction) whose operand bytes are checked.
// Takes tainted (*[generalRegisterBankSize]bool) which holds the alias closure.
//
// Returns true when any of a/b/c is tainted.
func anyGeneralTainted(inst instruction, tainted *[generalRegisterBankSize]bool) bool {
	return tainted[inst.a] || tainted[inst.b] || tainted[inst.c]
}

// opcodeMayReadGeneral reports whether an opcode might read a general-bank register
// without providing shape info. Conservative allowlist: call-shaped, store-shaped, and
// return-shaped opcodes.
//
// Takes op (opcode) which is the instruction's primary opcode.
//
// Returns true when the opcode's effect on general-bank reads is ambiguous.
func opcodeMayReadGeneral(op opcode) bool {
	switch op {
	case opCall, opCallMethod, opCallMethodInlineable, opCallNative, opCallBuiltin, opCallIIFE,
		opDefer, opTailCall, opSetGlobal, opSetUpvalue,
		opWriteSharedCell, opMakeClosure, opMapSet, opIndexSet,
		opPackInterface, opPackTyped:
		return true
	default:
	}
	return false
}

// annotateArenaSafeAllocs scans cf.body for opAllocIndirect sites whose output register's
// value cannot escape cf's frame and records each in cf.arenaSafeAllocPCs.
//
// Reuses analyseGeneralRegisterEscape but seeds the taint set from the opAllocIndirect's
// output register (operand A). The walk starts after the alloc instruction so the source
// operand's slot cannot collide with the freshly-allocated output slot.
//
// Takes cf (*CompiledFunction) whose body is scanned.
func annotateArenaSafeAllocs(cf *CompiledFunction) {
	pc := 0
	for pc < len(cf.body) {
		inst := cf.body[pc]
		if inst.op != opAllocIndirect {
			pc = advancePastExtensions(cf, pc)
			continue
		}
		outputSlot := inst.a
		afterAlloc := advancePastExtensions(cf, pc)
		if !analyseGeneralRegisterEscape(cf, outputSlot, afterAlloc) {
			if cf.arenaSafeAllocPCs == nil {
				cf.arenaSafeAllocPCs = make(map[int]bool)
			}
			cf.arenaSafeAllocPCs[pc] = true
		}
		pc = afterAlloc
	}
}
