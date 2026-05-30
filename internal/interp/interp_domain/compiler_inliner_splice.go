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
	"math"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// maxParamSlotsScanned bounds the per-bank bitmap.
	//
	// Parameter slots always live at the low end of each bank, so a small fixed cap
	// suffices; functions with more than this many params per bank fall through to "assume
	// all written" via markAllParamSlotsWritten.
	maxParamSlotsScanned = 32

	// instructionOperandCount is the number of explicit operand bytes (A, B, C) carried by
	// each instruction.
	instructionOperandCount = 3
)

var (
	// phase2OpcodeAllowList enumerates every opcode the splice driver may emit into the
	// caller.
	//
	// Grouped by category so each allow/refuse decision is documented next to its opcodes.
	// Lookup is via a flat lookup table indexed by opcode, populated once during package
	// init from the per-category slices below.
	phase2OpcodeAllowList = func() [256]bool {
		var table [256]bool
		for _, op := range phase2AllowedTier0Categories {
			table[op] = true
		}
		return table
	}()

	// phase2AllowedTier0Categories is the flat union of every opcode category the splice
	// driver allows.
	//
	// Each category is documented below. opTailCall is included because appendCalleeBody
	// converts it to opCall during the splice; the conversion is safe only for
	// cross-tail-calls (target != callee), which calleeTailCallsAreAllCrossTarget enforces
	// at scan time, and self-recursive tail calls remain refused via inlineRefusalTailCall.
	phase2AllowedTier0Categories = func() []opcode {
		categories := [][]opcode{
			{opDrillTier1, opCall, opTailCall, opExt, opCopyStructFieldGeneralT0},
			phase2AllowedConstantLoads,
			phase2AllowedConstFusedArith,
			phase2AllowedTier0StructField,
			phase2AllowedPureArithmetic,
			phase2AllowedComparisons,
			phase2AllowedJumpsAndFusions,
			phase2AllowedGeneralEquality,
			phase2AllowedFieldAccessors,
			phase2AllowedMoveAndString,
			phase2AllowedMapOps,
			phase2AllowedExtensionOps,
			phase2AllowedMapIndexOkVariants,
			phase2AllowedTypedMapOps,
		}
		var total int
		for _, group := range categories {
			total += len(group)
		}
		out := make([]opcode, 0, total)
		for _, group := range categories {
			out = append(out, group...)
		}
		return out
	}()

	// phase2AllowedConstantLoads carries the constant-load opcodes the splice driver allows.
	phase2AllowedConstantLoads = []opcode{
		opLoadIntConst, opLoadFloatConst, opLoadStringConst,
		opLoadBoolConst, opLoadUintConst, opLoadComplexConst,
		opLoadGeneralConst,
	}

	// phase2AllowedConstFusedArith carries the const-fused arithmetic opcodes the splice
	// driver allows.
	phase2AllowedConstFusedArith = []opcode{
		opAddIntConst, opSubIntConst, opMulIntConst,
	}

	// phase2AllowedTier0StructField carries the tier-0 struct-field opcodes whose operand C
	// is a layoutTable index merged via inlinePoolShapes.
	phase2AllowedTier0StructField = []opcode{
		opGetStructFieldIntT0, opSetStructFieldIntT0,
		opGetStructFieldUint, opSetStructFieldUint,
		opGetStructFieldFloat, opSetStructFieldFloat,
		opGetStructFieldBool, opSetStructFieldBool,
		opGetStructFieldGeneral, opSetStructFieldGeneral,
	}

	// phase2AllowedPureArithmetic carries the register-only arithmetic opcodes the splice
	// driver allows.
	phase2AllowedPureArithmetic = []opcode{
		opAddInt, opSubInt, opMulInt, opDivInt, opRemInt,
		opBitAnd, opBitOr, opBitXor, opBitAndNot,
		opShiftLeft, opShiftRight,
		opAddFloat, opSubFloat, opMulFloat, opDivFloat,
		opAddUint, opSubUint, opMulUint, opDivUint, opRemUint,
		opBitAndUint, opBitOrUint, opBitXorUint, opBitAndNotUint,
		opShiftLeftUint, opShiftRightUint,
	}

	// phase2AllowedComparisons carries the typed comparison opcodes the splice driver
	// allows.
	phase2AllowedComparisons = []opcode{
		opEqInt, opNeInt, opLtInt, opLeInt, opGtInt, opGeInt,
		opEqFloat, opNeFloat, opLtFloat, opLeFloat, opGtFloat, opGeFloat,
		opEqUint, opNeUint, opLtUint, opLeUint, opGtUint, opGeUint,
		opEqString, opNeString, opLtString, opLeString, opGtString, opGeString,
	}

	// phase2AllowedJumpsAndFusions carries the jump and compare+jump fusion opcodes the
	// splice driver allows. opTestNilJump* are included because LRU's linked-list traversal
	// depends on them.
	phase2AllowedJumpsAndFusions = []opcode{
		opJumpIfTrue, opJumpIfFalse,
		opTestNilJumpTrue, opTestNilJumpFalse,
		opLtIntJumpFalse, opLeIntJumpFalse,
		opGtIntJumpFalse, opGeIntJumpFalse,
		opEqIntJumpFalse, opNeIntJumpFalse,
	}

	// phase2AllowedGeneralEquality carries general-bank equality opcodes. Needed for LRU's
	// `node == cache.head` style comparisons.
	phase2AllowedGeneralEquality = []opcode{
		opEqGeneral, opNeGeneral,
	}

	// phase2AllowedFieldAccessors carries the slow opGet/Set field opcodes where operand B/C
	// is a uint8 field index and no pool merge is needed.
	phase2AllowedFieldAccessors = []opcode{
		opGetField, opSetField, opGetFieldInt, opSetFieldInt,
	}

	// phase2AllowedMoveAndString carries the move and string-concat opcodes the splice
	// driver allows.
	phase2AllowedMoveAndString = []opcode{
		opMoveGeneral, opConcatString,
	}

	// phase2AllowedMapOps carries the pure-register map opcodes (no extension) the splice
	// driver allows.
	phase2AllowedMapOps = []opcode{
		opMapIndex, opMapSet,
	}

	// phase2AllowedExtensionOps carries extension-word ops whose extension
	// A|B<<instructionByteShift is a typeTable pool index.
	phase2AllowedExtensionOps = []opcode{
		opMakeSlice, opConvert, opTypeAssert, opAllocIndirect, opPackTyped,
	}

	// phase2AllowedMapIndexOkVariants carries opMapIndexOk variants whose extension carries
	// an okRegister to remap.
	phase2AllowedMapIndexOkVariants = []opcode{
		opMapIndexOk,
		opMapIndexOkIntInt, opMapIndexOkIntString, opMapIndexOkIntGeneral,
		opMapIndexOkStringInt, opMapIndexOkStringString,
	}

	// phase2AllowedTypedMapOps carries the typed map operations with primitive key+value
	// banks. The comma-ok variants are listed in phase2AllowedMapIndexOkVariants; these
	// direct variants are pure-register without extension.
	phase2AllowedTypedMapOps = []opcode{
		opMapGetIntInt, opMapSetIntInt,
		opMapGetStringInt, opMapSetStringInt,
		opMapGetIntString, opMapSetIntString,
		opMapGetStringString, opMapSetStringString,
		opMapGetIntGeneral,
		opMapAddIntInt, opMapAddStringInt,
	}
)

// inlineSpliceResult reports the outcome of a single splice attempt.
type inlineSpliceResult struct {
	// spliced is true when the splice succeeded and the caller's body was mutated.
	spliced bool

	// reason records the refusal kind when spliced is false.
	reason inlineRefusal
}

// trySpliceCall attempts to inline the call at site siteIndex in caller.
//
// The caller is responsible for having already passed canInline. The splice may discover
// additional restrictions (e.g. constant-pool uses) and refuse.
//
// Takes caller (*CompiledFunction) whose body is mutated on success.
// Takes siteIndex (uint16) which selects the call site within caller.callSites.
//
// Returns inlineSpliceResult which is the splice outcome; spliced=true means the body was
// mutated, spliced=false carries the refusal reason.
func trySpliceCall(caller *CompiledFunction, siteIndex uint16) inlineSpliceResult {
	return trySpliceCallAt(caller, siteIndex, -1)
}

// trySpliceCallAt is the inner splice driver.
//
// When opCallPCHint >= 0 it is trusted as the call's PC; otherwise the body is scanned
// via findOpCallPC. Callers in inlineCallsIn pass a precomputed hint so the per-site body
// scan is avoided.
//
// Takes caller (*CompiledFunction) whose body is mutated on success.
// Takes siteIndex (uint16) which selects the call site within caller.callSites.
// Takes opCallPCHint (int) which is the opCall PC, or -1 to force a body scan.
//
// Returns inlineSpliceResult which is the splice outcome; spliced=true means the body was
// mutated, spliced=false carries the refusal reason.
func trySpliceCallAt(caller *CompiledFunction, siteIndex uint16, opCallPCHint int) inlineSpliceResult {
	if caller == nil {
		return inlineSpliceResult{reason: inlineRefusalUnknown}
	}
	if int(siteIndex) >= len(caller.callSites) {
		return inlineSpliceResult{reason: inlineRefusalUnknown}
	}
	site := &caller.callSites[siteIndex]
	callee := site.cachedCallee
	if callee == nil {
		return inlineSpliceResult{reason: inlineRefusalNoBody}
	}
	opCallPC := opCallPCHint
	if opCallPC < 0 {
		opCallPC = findOpCallPC(caller, siteIndex)
	}
	if opCallPC < 0 {
		return inlineSpliceResult{reason: inlineRefusalUnknown}
	}
	if reason := phase2OpcodeScan(callee); reason != inlineEligible {
		return inlineSpliceResult{reason: reason}
	}
	ctx := inlineContext{
		caller:    caller,
		callee:    callee,
		site:      site,
		siteIndex: siteIndex,
		opCallPC:  opCallPC,
	}
	if reason := buildRegisterRemap(&ctx); reason != inlineEligible {
		return inlineSpliceResult{reason: reason}
	}
	if reason := performSplice(&ctx); reason != inlineEligible {
		return inlineSpliceResult{reason: reason}
	}
	return inlineSpliceResult{spliced: true}
}

// phase2OpcodeScan rejects callees containing any unhandled opcode.
//
// The check is allowlist-based: every opcode the splice driver must support has to be on
// this list. Adding a new opcode involves verifying both (a) its operand shape is
// described in operandShapes, and (b) any pool indices it references are wired into
// inlinePoolShapes.
//
// Takes callee (*CompiledFunction) whose body is scanned.
//
// Returns inlineRefusal which is inlineEligible when every opcode is on the allowlist, or
// inlineRefusalOversize when an unsupported opcode is encountered.
func phase2OpcodeScan(callee *CompiledFunction) inlineRefusal {
	for i := range callee.body {
		instr := callee.body[i]
		if !phase2OpcodeAllowed(instr.op) {
			return inlineRefusalOversize
		}
	}
	return inlineEligible
}

// phase2OpcodeAllowed reports whether the splice driver supports op.
//
// Curated rather than auto-derived from operandShapes because some opcodes have shape
// descriptions but use extension words the inliner does not walk; some have implicit
// dependencies on the callee's frame state (opAllocIndirect, opResetSharedCell) that do
// not survive splicing; the explicit list documents the inliner's coverage and makes
// coverage expansion a conscious decision.
//
// Takes op (opcode) which is the opcode under inspection.
//
// Returns bool which is true when op is on the splice driver's allowlist.
func phase2OpcodeAllowed(op opcode) bool {
	return phase2OpcodeAllowList[op]
}

// buildRegisterRemap fills ctx.remap for every callee-side register.
//
// Parameter slots get fresh caller-side slots and a pre-copy MOVE recorded on
// ctx.paramPreCopies seeding the fresh slot from the caller's argument source. The
// fresh-slot indirection is required because the callee's bytecode writes to parameter
// slots (notably the trailing "MOVE return-slot N, result" idiom every non-void function
// ends with). Without it, those writes would silently mutate the caller's argument
// variable, breaking Go's by-value parameter semantics. Read-only params (detected
// per-slot via scanWrittenParamSlots) keep the direct alias to arg.register and emit no
// pre-copy, preserving the zero-MOVE-overhead property for pure-read parameters.
//
// Local slots get fresh caller-side registers by bumping caller.numRegisters[bank],
// starting after any param-fresh slots. Return-position slots overlap with param slots in
// callee register space; because param slots now resolve to fresh caller-side slots, the
// return-prep rewrite reads from those fresh slots where the callee body legitimately
// deposits the return value, without disturbing the caller's argument. Mutates
// ctx.caller.numRegisters and ctx.paramPreCopies as a side effect.
//
// Takes ctx (*inlineContext) which holds caller, callee, site, and the remap arrays being
// populated.
//
// Returns inlineEligible on success, inlineRefusalCapWatermark on register-watermark
// overflow, or inlineRefusalVariadic when the site's argument count does not match the
// callee's parameter count.
func buildRegisterRemap(ctx *inlineContext) inlineRefusal {
	ctx.resetRegisterRemap()
	ctx.paramPreCopies = ctx.paramPreCopies[:0]
	if len(ctx.site.arguments) != len(ctx.callee.parameterKinds) {
		return inlineRefusalVariadic
	}
	calleeSlots, isParamSlot := resolveCalleeParameterSlots(ctx.callee)
	if refusal := remapInlineParameters(ctx, calleeSlots); refusal != inlineEligible {
		return refusal
	}
	return remapInlineLocals(ctx, &isParamSlot)
}

// resolveCalleeParameterSlots returns per-parameter slot indices.
//
// Computes the per-parameter callee slot index and the per-bank bitmap of slots occupied
// by parameters. parameterRegisters, when populated, gives the slot promoteToIndirect
// ended up at after any heap-promote prologue allocations pushed later same-bank
// parameters off the naive per-bank counter; older synthetic functions and tests fall
// back to the per-bank counter.
//
// Takes callee (*CompiledFunction) whose parameter layout is resolved.
//
// Returns []uint8 which holds the callee slot index per parameter.
// Returns [NumRegisterKinds][maxParamSlotsScanned]bool which marks slots occupied by
// parameters.
func resolveCalleeParameterSlots(callee *CompiledFunction) ([]uint8, [NumRegisterKinds][maxParamSlotsScanned]bool) {
	var bankParamCounter [NumRegisterKinds]uint8
	calleeSlots := make([]uint8, len(callee.parameterKinds))
	hasRecordedRegisters := len(callee.parameterRegisters) == len(callee.parameterKinds)
	var isParamSlot [NumRegisterKinds][maxParamSlotsScanned]bool
	for i, paramKind := range callee.parameterKinds {
		var slot uint8
		if hasRecordedRegisters {
			slot = callee.parameterRegisters[i]
		} else {
			slot = bankParamCounter[paramKind]
		}
		bankParamCounter[paramKind]++
		calleeSlots[i] = slot
		if int(slot) < maxParamSlotsScanned {
			isParamSlot[paramKind][slot] = true
		}
	}
	return calleeSlots, isParamSlot
}

// remapInlineParameters wires each callee parameter slot to its source.
//
// Arguments of a different kind to the parameter are routed through a fresh same-kind
// temporary plus a param-pre-copy; arguments to a slot the callee writes to also get a
// fresh temporary so the caller's slot is not clobbered by the inlined body.
//
// Takes ctx (*inlineContext) which holds caller, callee, and the remap table being
// populated.
// Takes calleeSlots ([]uint8) which is the per-parameter slot index.
//
// Returns inlineRefusal which is inlineEligible on success or the refusal kind on
// failure.
func remapInlineParameters(ctx *inlineContext, calleeSlots []uint8) inlineRefusal {
	writtenParamSlots := scanWrittenParamSlots(ctx.callee)
	for i, paramKind := range ctx.callee.parameterKinds {
		argument := ctx.site.arguments[i]
		calleeSlot := calleeSlots[i]
		if argument.kind != paramKind {
			if _, supported := crossBankAdoptOrBoxSubOp(argument.kind, paramKind); !supported {
				return inlineRefusalCapWatermark
			}
			if refusal := allocateInlineParamCopy(ctx, paramKind, calleeSlot, argument.kind, argument.register); refusal != inlineEligible {
				return refusal
			}
			continue
		}
		if !writtenParamSlots[paramKind][calleeSlot] {
			ctx.setRegister(paramKind, calleeSlot, argument.register)
			continue
		}
		if refusal := allocateInlineParamCopy(ctx, paramKind, calleeSlot, paramKind, argument.register); refusal != inlineEligible {
			return refusal
		}
	}
	return inlineEligible
}

// allocateInlineParamCopy reserves a fresh caller slot for a defensive copy.
//
// Reserves the slot for a parameter that needs a defensive copy (cross- kind argument or
// a slot the callee writes to) and records the matching pre-copy entry.
//
// Takes ctx (*inlineContext) which holds caller and paramPreCopies.
// Takes paramKind (registerKind) which is the callee parameter bank.
// Takes calleeSlot (uint8) which is the callee-side slot being remapped.
// Takes sourceKind (registerKind) which is the caller argument bank.
// Takes source (uint8) which is the caller argument slot.
//
// Returns inlineRefusal which is inlineEligible on success or inlineRefusalCapWatermark
// when the bank overflows.
func allocateInlineParamCopy(ctx *inlineContext, paramKind registerKind, calleeSlot uint8, sourceKind registerKind, source uint8) inlineRefusal {
	fresh := ctx.caller.numRegisters[paramKind]
	if fresh > registerBankWatermark {
		return inlineRefusalCapWatermark
	}
	ctx.setRegister(paramKind, calleeSlot, uint8(fresh))
	ctx.caller.numRegisters[paramKind] = fresh + 1
	ctx.paramPreCopies = append(ctx.paramPreCopies, paramPreCopy{
		sourceKind:  sourceKind,
		kind:        paramKind,
		destination: uint8(fresh),
		source:      source,
	})
	return inlineEligible
}

// remapInlineLocals maps every non-parameter callee slot to a fresh local.
//
// Covers heap-promote pointer slots, temporaries, and named results. With recorded
// parameterRegisters the parameter slots may be scattered (e.g. general[0] is a
// heap-promote local but general[1] holds the parameter), so isParamSlot is honoured
// explicitly to keep scattered locals from reusing a parameter remap.
//
// Takes ctx (*inlineContext) which holds caller, callee, and the remap table.
// Takes isParamSlot (*[NumRegisterKinds][maxParamSlotsScanned]bool) which marks slots
// already mapped as parameters.
//
// Returns inlineRefusal which is inlineEligible on success or inlineRefusalCapWatermark
// when any bank overflows.
func remapInlineLocals(ctx *inlineContext, isParamSlot *[NumRegisterKinds][maxParamSlotsScanned]bool) inlineRefusal {
	for k := range registerKind(NumRegisterKinds) {
		calleeTotal := ctx.callee.numRegisters[k]
		if calleeTotal == 0 {
			continue
		}
		callerBase := ctx.caller.numRegisters[k]
		var allocated uint32
		for slot := range calleeTotal {
			if int(slot) < maxParamSlotsScanned && isParamSlot[k][slot] {
				continue
			}
			if callerBase+allocated > registerBankWatermark {
				return inlineRefusalCapWatermark
			}
			ctx.setRegister(k, safeconv.MustUintToUint8(uint(slot)), safeconv.MustUintToUint8(uint(callerBase+allocated)))
			allocated++
		}
		ctx.caller.numRegisters[k] = callerBase + allocated
	}
	return inlineEligible
}

// scanWrittenParamSlots returns a per-bank bitmap of written param slots.
//
// Only slots up to the param count (bank-K slot 0..paramCount[K]-1) are interesting;
// writes to higher slots are local slots and never alias to caller storage. The scan is
// O(len(callee.body)) and runs once per splice attempt. Drives the
// skip-pre-copy-when-read-only optimisation in buildRegisterRemap so functions like a
// range-loop helper that only reads its parameters incur zero pre-copy overhead. Operands
// are interpreted via the operandShapes table: an operand whose role names a register
// bank and whose writes-bit is set counts as a write to that slot. Pseudo-roles like
// roleRegDynamic are treated conservatively (assume write) so the fast path never
// silently aliases a slot that turns out to be mutated. opDrillTier1 instructions are
// decoded by recordTier1SubOpParamWrite because the meaningful writes happen inside the
// sub-op dispatch; in particular the trailing return-slot MOVE every non-void function
// emits goes through this opcode.
//
// Takes callee (*CompiledFunction) whose body is scanned.
//
// Returns the per-bank write bitmap; written[bank][slot]=true means the callee writes to
// that slot at least once.
func scanWrittenParamSlots(callee *CompiledFunction) [NumRegisterKinds][maxParamSlotsScanned]bool {
	var written [NumRegisterKinds][maxParamSlotsScanned]bool
	for i := range callee.body {
		instr := callee.body[i]
		op := instr.op
		shape := operandShapes[op]
		if shape.flags&shapeFlagDescribed == 0 {
			markAllParamSlotsWritten(&written, callee)
			return written
		}
		if op == opDrillTier1 {
			recordTier1SubOpParamWrite(&written, callee, instr)
			continue
		}
		recordDescribedParamWrites(&written, callee, instr, shape)
	}
	return written
}

// markAllParamSlotsWritten marks every callee parameter slot as written.
//
// Used as the conservative fallback when the inliner cannot classify an opcode's writes
// (unknown shape, roleRegDynamic operand). Forces every param through the pre-copy path
// so no aliasing can occur.
//
// Takes written (*[NumRegisterKinds][maxParamSlotsScanned]bool) which is the per-bank
// bitmap being populated.
// Takes callee (*CompiledFunction) whose parameterKinds drive which slots get marked.
func markAllParamSlotsWritten(written *[NumRegisterKinds][maxParamSlotsScanned]bool, callee *CompiledFunction) {
	hasRecordedRegisters := len(callee.parameterRegisters) == len(callee.parameterKinds)
	var bankCounter [NumRegisterKinds]uint8
	for i, paramKind := range callee.parameterKinds {
		var slot uint8
		if hasRecordedRegisters {
			slot = callee.parameterRegisters[i]
		} else {
			slot = bankCounter[paramKind]
		}
		bankCounter[paramKind]++
		if int(slot) < maxParamSlotsScanned {
			written[paramKind][slot] = true
		}
	}
}

// recordDescribedParamWrites marks param slots that an instruction writes.
//
// Slots beyond maxParamSlotsScanned are ignored: they cannot be parameter slots because
// the param scan caps at that cap. Also records indirect writes through a register
// handle: opcodes like opSetField, opSetStructFieldInt, opIndexSet, opMapSet,
// opMapDelete, etc. do not replace operand A's register value, but they mutate the
// storage A points at (the struct's field, the slice's element, the map's bucket). For
// the inliner's by-value-parameter contract, that mutation is observable to the caller if
// A is aliased to the caller's argument source register. Such operands are treated as
// writes so the splice routes them through a fresh pre-copied slot.
//
// Takes written (*[NumRegisterKinds][maxParamSlotsScanned]bool) which accumulates the
// per-bank write bitmap.
// Takes callee (*CompiledFunction) used only for the conservative fallback to
// markAllParamSlotsWritten on dynamic-role operands.
// Takes instr (instruction) whose operand bytes are read.
// Takes shape (operandShape) which carries the per-operand role and writes-bit
// annotations.
func recordDescribedParamWrites(written *[NumRegisterKinds][maxParamSlotsScanned]bool, callee *CompiledFunction, instr instruction, shape operandShape) {
	bytes := [instructionOperandCount]uint8{instr.a, instr.b, instr.c}
	roles := [instructionOperandCount]operandRole{shape.a, shape.b, shape.c}
	indirectPos, indirectBank, hasIndirect := indirectWriteThroughOperand(instr.op)
	for pos, role := range roles {
		isWrite := shape.writes[pos]
		if hasIndirect && pos == indirectPos {
			slot := bytes[pos] //nolint:gosec // pos bounded to len(roles)==3 by range
			if int(slot) < maxParamSlotsScanned {
				written[indirectBank][slot] = true
			}
			continue
		}
		if !isWrite {
			continue
		}
		bank, isReg := kindForRole(role)
		if !isReg {
			markAllParamSlotsWritten(written, callee)
			return
		}
		slot := bytes[pos] //nolint:gosec // pos bounded to len(roles)==3 by range
		if int(slot) < maxParamSlotsScanned {
			written[bank][slot] = true
		}
	}
}

// indirectWriteThroughOperand reports an opcode's mutation-target operand.
//
// opSetField writes through A (the struct handle), opMapSet writes through A (the map
// handle), opIndexSet writes through A (the slice or array handle), and so on. The bank
// is always general for these mutation-target operands; it is returned explicitly so
// callers do not have to consult operandShapes again.
//
// Takes op (opcode) which is the opcode under inspection.
//
// Returns int which is the operand position (0=A, 1=B, 2=C) of the mutation target.
// Returns registerKind which is the bank of the register through which storage is
// mutated.
// Returns bool which is true when op mutates through a register.
func indirectWriteThroughOperand(op opcode) (int, registerKind, bool) {
	switch op {
	case opSetField, opSetFieldInt,
		opSetStructFieldIntT0, opSetStructFieldUint,
		opSetStructFieldFloat, opSetStructFieldBool,
		opSetStructFieldGeneral,
		opIndexSet,
		opMapSet,
		opMapSetIntInt, opMapSetIntString,
		opMapSetStringInt, opMapSetStringString, opMapSetStringGeneral,
		opMapAddIntInt, opMapAddStringInt:
		return 0, registerGeneral, true
	default:
	}
	return 0, 0, false
}

// recordTier1SubOpParamWrite records writes for opDrillTier1 dispatch.
//
// The trailing return-slot MOVE every non-void function emits goes through this opcode
// (see compileReturn / finalisation), so missing it would defeat the optimisation: the
// return-prep MOVE would not be recognised as a write and the inliner would alias the
// param slot directly. The subOpMapDelete sub-op is handled too because it mutates the
// map handle in operand B (same shape as the opMapSet* indirect-write family). Sub-op
// decoding mirrors the runtime tier-1 dispatch: instruction.a names the sub-opcode, and
// the remaining operands carry bank-typed register payloads.
//
// Takes written (*[NumRegisterKinds][maxParamSlotsScanned]bool) which accumulates the
// per-bank write bitmap.
// Takes callee (*CompiledFunction) which is currently unused but kept so the signature
// matches its sibling helpers.
// Takes instr (instruction) whose A byte selects the sub-op and whose B byte names the
// destination slot for move-style sub-ops.
func recordTier1SubOpParamWrite(written *[NumRegisterKinds][maxParamSlotsScanned]bool, callee *CompiledFunction, instr instruction) {
	sub := subOpcode(instr.a)
	if sub == subOpMapDelete {
		if int(instr.b) < maxParamSlotsScanned {
			written[registerGeneral][instr.b] = true
		}
		return
	}
	bank, writesBSlot := tier1SubOpWriteBank(sub)
	if !writesBSlot {
		return
	}
	if int(instr.b) < maxParamSlotsScanned {
		written[bank][instr.b] = true
	}
	_ = callee
}

// tier1SubOpWriteBank returns the bank of a tier-1 sub-op's B operand.
//
// Applies to the canonical "MOVE dst, src" pattern where the sub-op writes to operand B's
// slot. Sub-ops that either do not write a register or write to a different operand
// position are handled conservatively (no recorded write) because the splice driver would
// refuse them at phase2OpcodeScan or remapTier1SubOp if they touched a parameter slot.
//
// Takes sub (subOpcode) which is the tier-1 sub-opcode under inspection.
//
// Returns the destination bank.
// Returns false when the sub-op does not write to operand B's slot.
func tier1SubOpWriteBank(sub subOpcode) (registerKind, bool) {
	switch sub {
	case subOpMoveInt:
		return registerInt, true
	case subOpMoveFloat:
		return registerFloat, true
	case subOpMoveString:
		return registerString, true
	case subOpMoveBool:
		return registerBool, true
	case subOpMoveUint:
		return registerUint, true
	default:
	}
	return 0, false
}

// buildParamPreCopyInstruction emits the bytecode for one pre-copy MOVE.
//
// Copies the caller's argument source register into the fresh caller-side parameter slot
// the inliner allocated. Bank-typed pre-copies go through the corresponding tier-1 MOVE
// sub-op (same as the splice uses for return-prep moves); general-bank pre-copies go
// through the top-level opMoveGeneral with the dynamic mode marker that matches runtime
// copyCallArgs' valueCopyForBoundary semantics, so arena-resident values are correctly
// snapshotted at the call boundary.
//
// Takes pc (paramPreCopy) which carries the bank and the dst/src slots.
//
// Returns the encoded instruction.
// Returns false when the bank has no corresponding tier-1 move sub-op.
func buildParamPreCopyInstruction(pc paramPreCopy) (instruction, bool) {
	if pc.sourceKind != pc.kind {
		subOp, ok := crossBankAdoptOrBoxSubOp(pc.sourceKind, pc.kind)
		if !ok {
			return instruction{}, false
		}
		return makeInstruction(opDrillTier1, byte(subOp), pc.destination, pc.source), true
	}
	if pc.kind == registerGeneral {
		return makeInstruction(opMoveGeneral, pc.destination, pc.source, moveGeneralModeDynamic), true
	}
	subOp, ok := typedMoveSubOp(pc.kind)
	if !ok {
		return instruction{}, false
	}
	return makeInstruction(opDrillTier1, byte(subOp), pc.destination, pc.source), true
}

// crossBankAdoptOrBoxSubOp returns the tier-1 sub-op bridging two banks.
//
// Bridges a cross-bank MOVE between general and one of the six typed- slice banks. Adopt
// (general -> slicesX) reads reflect.Value.Interface() and type-asserts; box (slicesX ->
// general) calls reflect.ValueOf. All other cross-bank pairs (int<->float,
// slicesInt<->slicesFloat, etc.) are rejected; the splicer falls back to
// inlineRefusalCapWatermark for those.
//
// Takes sourceKind (registerKind) which holds the caller's argument.
// Takes destinationKind (registerKind) which is the callee's parameter slot bank.
//
// Returns subOpcode which is the matching adopt/box sub-op when supported, zero
// otherwise.
// Returns bool which is true on a supported pair.
func crossBankAdoptOrBoxSubOp(sourceKind, destinationKind registerKind) (subOpcode, bool) {
	if sourceKind == registerGeneral {
		switch destinationKind {
		case registerSliceInt:
			return subOpAdoptGeneralToSlicesInt, true
		case registerSliceFloat:
			return subOpAdoptGeneralToSlicesFloat, true
		case registerSliceString:
			return subOpAdoptGeneralToSlicesString, true
		case registerSliceBool:
			return subOpAdoptGeneralToSlicesBool, true
		case registerSliceUint:
			return subOpAdoptGeneralToSlicesUint, true
		case registerSliceByte:
			return subOpAdoptGeneralToSlicesByte, true
		default:
		}
		return 0, false
	}
	if destinationKind == registerGeneral {
		switch sourceKind {
		case registerSliceInt:
			return subOpBoxSliceInt, true
		case registerSliceFloat:
			return subOpBoxSliceFloat, true
		case registerSliceString:
			return subOpBoxSliceString, true
		case registerSliceBool:
			return subOpBoxSliceBool, true
		case registerSliceUint:
			return subOpBoxSliceUint, true
		case registerSliceByte:
			return subOpBoxSliceByte, true
		default:
		}
		return 0, false
	}
	return 0, false
}

// performSplice executes the actual bytecode mutation for one splice.
//
// Uses the "common return-prep block" pattern: each return point in the appended body
// becomes a single forward jump to a shared block at the end of the appended region. That
// block emits the typed moves from callee return-position slots to the caller's
// site.returns destinations, then a single back-jump to afterCallPC. Each callee
// instruction maps to exactly one appended instruction (no PC remap needed); the only
// growth beyond callee.body length is the single return-prep block. On any failure
// (unknown opcode shape, pool overflow, jump-offset out-of-range), caller state is rolled
// back via the captured snapshots of body length, numRegisters, callSites, and every
// constant pool length.
//
// Takes ctx (*inlineContext) holding caller, callee, site, opCallPC, remap, and
// paramPreCopies.
//
// Returns inlineEligible on success, or the refusal kind reported by the failing step on
// rollback.
func performSplice(ctx *inlineContext) inlineRefusal {
	snapshot := captureSpliceSnapshot(ctx.caller)
	rollback := func() { restoreSpliceSnapshot(ctx.caller, snapshot) }

	skipJumpPC := emitSpliceSkipJump(ctx.caller)
	preCopyStart, ok := emitParamPreCopies(ctx)
	if !ok {
		rollback()
		return inlineRefusalCapWatermark
	}
	calleeBodyStart, refusal := appendCalleeBody(ctx)
	if refusal != inlineEligible {
		rollback()
		return refusal
	}
	retPrepStart := len(ctx.caller.body)
	if reason := emitReturnPrep(ctx, retPrepStart); reason != inlineEligible {
		rollback()
		return reason
	}
	if !rewriteReturnsToRetPrep(ctx, calleeBodyStart, retPrepStart) {
		rollback()
		return inlineRefusalUnknown
	}
	if !patchOpCallForward(ctx, preCopyStart) {
		rollback()
		return inlineRefusalUnknown
	}
	if !patchSkipJump(ctx, skipJumpPC) {
		rollback()
		return inlineRefusalUnknown
	}
	recordSelfRecursionAnnotation(ctx, calleeBodyStart)
	return inlineEligible
}

// recordSelfRecursionAnnotation marks the first instruction of an inlined self-recursive
// callee body with peepholeRewriteUnroll.
//
// The origin is the original opCall PC that the unroll replaced; the annotation only
// applies to the self-recursive case (callee == caller); ordinary inlining is not
// annotated by this helper.
//
// Takes ctx (*inlineContext) which is the inline context carrying caller, callee, and
// opCallPC.
// Takes calleeBodyStart (int) which is the caller PC at which the appended body starts.
func recordSelfRecursionAnnotation(ctx *inlineContext, calleeBodyStart int) {
	if ctx == nil || ctx.caller == nil {
		return
	}
	if ctx.callee != ctx.caller {
		return
	}
	if calleeBodyStart < 0 || calleeBodyStart >= len(ctx.caller.body) {
		return
	}
	ctx.caller.recordPeepholeRewrite(calleeBodyStart, peepholeRewriteUnroll, ctx.opCallPC)
}

// spliceSnapshot captures every caller slice length / numRegisters value that the splice
// may mutate. restoreSpliceSnapshot uses it on failure to roll the caller back to its
// pre-splice state.
type spliceSnapshot struct {
	// bodyLen records the caller body length before the splice.
	bodyLen int

	// numRegisters records the per-bank register counts before the splice.
	numRegisters [NumRegisterKinds]uint32

	// callSites records the caller's callSites slice length.
	callSites int

	// intConsts records the caller's intConstants slice length.
	intConsts int

	// floatConsts records the caller's floatConstants slice length.
	floatConsts int

	// stringConsts records the caller's stringConstants slice length.
	stringConsts int

	// boolConsts records the caller's boolConstants slice length.
	boolConsts int

	// uintConsts records the caller's uintConstants slice length.
	uintConsts int

	// complexConsts records the caller's complexConstants slice length.
	complexConsts int

	// generalConsts records the caller's generalConstants slice length.
	generalConsts int

	// typeTable records the caller's typeTable slice length.
	typeTable int

	// structLayoutTable records the caller's structLayoutTable slice length.
	structLayoutTable int
}

// captureSpliceSnapshot records the caller's mutable state prior to the splice so any
// failure can roll back cleanly.
//
// Takes caller (*CompiledFunction) whose state is captured.
//
// Returns the snapshot describing the pre-splice state.
func captureSpliceSnapshot(caller *CompiledFunction) spliceSnapshot {
	return spliceSnapshot{
		bodyLen:           len(caller.body),
		numRegisters:      caller.numRegisters,
		callSites:         len(caller.callSites),
		intConsts:         len(caller.intConstants),
		floatConsts:       len(caller.floatConstants),
		stringConsts:      len(caller.stringConstants),
		boolConsts:        len(caller.boolConstants),
		uintConsts:        len(caller.uintConstants),
		complexConsts:     len(caller.complexConstants),
		generalConsts:     len(caller.generalConstants),
		typeTable:         len(caller.typeTable),
		structLayoutTable: len(caller.structLayoutTable),
	}
}

// restoreSpliceSnapshot rewinds every caller slice and numRegisters counter to the
// lengths recorded in the snapshot.
//
// Takes caller (*CompiledFunction) whose state is restored.
// Takes snapshot (spliceSnapshot) carrying the pre-splice lengths.
func restoreSpliceSnapshot(caller *CompiledFunction, snapshot spliceSnapshot) {
	caller.body = caller.body[:snapshot.bodyLen]
	caller.numRegisters = snapshot.numRegisters
	caller.callSites = caller.callSites[:snapshot.callSites]
	caller.intConstants = caller.intConstants[:snapshot.intConsts]
	caller.floatConstants = caller.floatConstants[:snapshot.floatConsts]
	caller.stringConstants = caller.stringConstants[:snapshot.stringConsts]
	caller.boolConstants = caller.boolConstants[:snapshot.boolConsts]
	caller.uintConstants = caller.uintConstants[:snapshot.uintConsts]
	caller.complexConstants = caller.complexConstants[:snapshot.complexConsts]
	caller.generalConstants = caller.generalConstants[:snapshot.generalConsts]
	caller.typeTable = caller.typeTable[:snapshot.typeTable]
	caller.structLayoutTable = caller.structLayoutTable[:snapshot.structLayoutTable]
}

// emitSpliceSkipJump appends a forward jump (with a zero offset that is later
// back-patched) which vaults over the entire appended trampoline region.
//
// Without this, fall-through from the original caller body (e.g. end-of-body implicit
// return) lands inside the appended bytecode and walks the retPrep back-jump, looping
// forever. The exact offset is back-patched in patchSkipJump once the final body length
// is known.
//
// Takes caller (*CompiledFunction) whose body receives the stub jump.
//
// Returns the PC of the stub jump.
func emitSpliceSkipJump(caller *CompiledFunction) int {
	skipJumpPC := len(caller.body)
	caller.body = append(caller.body, makeInstruction(
		opDrillTier1,
		byte(subOpJump),
		0, 0,
	))
	return skipJumpPC
}

// emitParamPreCopies appends one MOVE per parameter that the callee writes to. Each MOVE
// seeds a fresh caller-side slot from the caller's argument source so the inlined body
// can write to its "parameter slot" without clobbering the caller's argument variable.
//
// Empty when the callee's parameters are all read-only - the inliner's zero-MOVE-overhead
// property is preserved for the common case.
//
// Takes ctx (*inlineContext) carrying caller and paramPreCopies.
//
// Returns int which is the PC at which the pre-copy block starts.
// Returns bool which is false when any pre-copy lacks a tier-1 move sub-op (caller should
// rollback and refuse with inlineRefusalCapWatermark) and true otherwise.
func emitParamPreCopies(ctx *inlineContext) (int, bool) {
	preCopyStart := len(ctx.caller.body)
	for _, pc := range ctx.paramPreCopies {
		copyInstr, ok := buildParamPreCopyInstruction(pc)
		if !ok {
			return 0, false
		}
		ctx.caller.body = append(ctx.caller.body, copyInstr)
	}
	return preCopyStart, true
}

// appendCalleeBody appends the callee body to the caller.
//
// Return instructions are temporarily emitted unchanged; rewriteReturnsToRetPrep patches
// them in a later step. Extension words (opExt) following ops marked hasExtensionWord get
// remapped and appended alongside their main instruction, with the loop counter advanced
// past them so they are not processed twice. opTailCall is rewritten to opCall after the
// remap: tail-call semantics in the original callee would replace the caller's frame with
// the tail-callee's frame, but once the callee has been inlined the surrounding
// continuation is the outer caller, so opCall preserves correct flow. Self-recursive tail
// calls are refused upstream at scanCalleeForRefusal to avoid unbounded frame growth.
//
// Takes ctx (*inlineContext) carrying caller, callee, and remap.
//
// Returns int which is the PC at which the callee body starts in the caller.
// Returns inlineRefusal which is inlineEligible on success or the refusal kind describing
// the failure.
func appendCalleeBody(ctx *inlineContext) (int, inlineRefusal) {
	calleeBodyStart := len(ctx.caller.body)
	for i := 0; i < len(ctx.callee.body); i++ {
		instr := ctx.callee.body[i]
		if isReturnInstruction(instr) {
			ctx.caller.body = append(ctx.caller.body, instr)
			continue
		}
		remapped, ok := remapOperands(instr, ctx)
		if !ok {
			return 0, inlineRefusalUnknown
		}
		if remapped.op == opTailCall {
			remapped.op = opCall
		}
		ctx.caller.body = append(ctx.caller.body, remapped)
		shape := inlinePoolShapeFor(instr.op)
		if !shape.hasExtensionWord {
			continue
		}
		if i+1 >= len(ctx.callee.body) {
			return 0, inlineRefusalUnknown
		}
		ext := ctx.callee.body[i+1]
		if ext.op != opExt {
			return 0, inlineRefusalUnknown
		}
		remappedExt, ok := remapExtensionOperands(ext, shape, ctx)
		if !ok {
			return 0, inlineRefusalUnknown
		}
		ctx.caller.body = append(ctx.caller.body, remappedExt)
		i++
	}
	return calleeBodyStart, inlineEligible
}

// rewriteReturnsToRetPrep rewrites each return instruction inside the appended callee
// body into a forward jump that targets retPrepStart.
//
// Takes ctx (*inlineContext) carrying caller and callee.
// Takes calleeBodyStart (int) which is the caller PC at which the appended body starts.
// Takes retPrepStart (int) which is the caller PC of the return-prep block.
//
// Returns true on success; false when a return's offset cannot fit a signed 16-bit jump.
func rewriteReturnsToRetPrep(ctx *inlineContext, calleeBodyStart, retPrepStart int) bool {
	for i := range ctx.callee.body {
		appendedPC := calleeBodyStart + i
		instr := ctx.caller.body[appendedPC]
		if !isReturnInstruction(instr) {
			continue
		}
		offset := retPrepStart - appendedPC - 1
		if !fitsJumpOffset(offset) {
			return false
		}
		lo, hi := splitJumpOffsetBytes(offset)
		ctx.caller.body[appendedPC] = makeInstruction(
			opDrillTier1,
			byte(subOpJump),
			lo,
			hi,
		)
	}
	return true
}

// patchOpCallForward replaces the opCall at ctx.opCallPC with a forward jump to the start
// of the pre-copy block (or, when no pre-copies were needed, directly to the callee body;
// in that case preCopyStart equals calleeBodyStart).
//
// Pre-copies must run before the body so the body sees its fresh parameter slots seeded.
//
// Takes ctx (*inlineContext) carrying caller and opCallPC.
// Takes preCopyStart (int) which is the caller PC of the pre-copy block.
//
// Returns true on success; false when the jump offset cannot fit a signed 16-bit
// immediate.
func patchOpCallForward(ctx *inlineContext, preCopyStart int) bool {
	offset := preCopyStart - ctx.opCallPC - 1
	if !fitsJumpOffset(offset) {
		return false
	}
	lo, hi := splitJumpOffsetBytes(offset)
	ctx.caller.body[ctx.opCallPC] = makeInstruction(
		opDrillTier1,
		byte(subOpJump),
		lo,
		hi,
	)
	return true
}

// patchSkipJump back-patches the stub skip-jump emitted in emitSpliceSkipJump so it lands
// past the end of the appended trampoline (i.e., at the current end-of-body PC, where
// implicit return fires).
//
// Takes ctx (*inlineContext) carrying caller.
// Takes skipJumpPC (int) which is the PC of the stub jump.
//
// Returns true on success; false when the offset cannot fit a signed 16-bit immediate.
func patchSkipJump(ctx *inlineContext, skipJumpPC int) bool {
	pastTrampolinePC := len(ctx.caller.body)
	offset := pastTrampolinePC - skipJumpPC - 1
	if !fitsJumpOffset(offset) {
		return false
	}
	lo, hi := splitJumpOffsetBytes(offset)
	ctx.caller.body[skipJumpPC] = makeInstruction(
		opDrillTier1,
		byte(subOpJump),
		lo,
		hi,
	)
	return true
}

// emitReturnPrep appends the shared return-prep block to caller.body.
//
// Emits one typed-move sub-op per return value, copying from the callee's return-position
// slot (bank-K, slot i_in_bank, REMAPPED to caller-side via ctx.remap) to the caller's
// site.returns[i] destination register. i_in_bank is a per-bank counter that matches how
// vm_handler_calls.handleReturn reads the values at runtime. A final unconditional
// subOpJump back to afterCallPC closes the block. Supports same-bank returns
// (resultKinds[i] == site.returns[i].kind). Cross-bank returns (e.g., int callee ->
// general interface caller) require boxing/unboxing and are refused with
// inlineRefusalCapWatermark.
//
// Correctness invariant: by the time the callee's return instruction fires at runtime,
// bank-K-slot-(i_in_bank) holds return value i. The compiler enforces this via
// moveLocsToReturnPositions when the result expression does not naturally land in that
// slot. The per-bank counter mirrors handleReturn's bankCounters.
//
// Takes ctx (*inlineContext) which carries callee, site, and ctx.remap.
// Takes retPrepStart (int) which is unused at runtime; it names the PC where the first
// emitted instruction lands.
//
// Returns inlineEligible on success, inlineRefusalUnknown when result counts disagree or
// the back-jump offset will not fit, and inlineRefusalCapWatermark for unsupported
// cross-bank returns or missing tier-1 move sub-ops.
func emitReturnPrep(ctx *inlineContext, retPrepStart int) inlineRefusal {
	_ = retPrepStart
	resultCount := len(ctx.callee.resultKinds)
	if resultCount != len(ctx.site.returns) {
		return inlineRefusalUnknown
	}
	var bankResultCounter [NumRegisterKinds]uint8
	for i, kind := range ctx.callee.resultKinds {
		calleeSlot := bankResultCounter[kind]
		bankResultCounter[kind]++
		sourceRegister, ok := ctx.lookupRegister(kind, calleeSlot)
		if !ok {
			return inlineRefusalCapWatermark
		}
		destination := ctx.site.returns[i]
		var subOp subOpcode
		if destination.kind != kind {
			adapter, supported := crossBankAdoptOrBoxSubOp(kind, destination.kind)
			if !supported {
				return inlineRefusalCapWatermark
			}
			subOp = adapter
		} else {
			move, ok := typedMoveSubOp(kind)
			if !ok {
				return inlineRefusalCapWatermark
			}
			subOp = move
		}
		ctx.caller.body = append(ctx.caller.body, makeInstruction(
			opDrillTier1,
			byte(subOp),
			destination.register,
			sourceRegister,
		))
	}
	afterCallPC := ctx.opCallPC + 1
	jumpPC := len(ctx.caller.body)
	offset := afterCallPC - jumpPC - 1
	if !fitsJumpOffset(offset) {
		return inlineRefusalUnknown
	}
	lo, hi := splitJumpOffsetBytes(offset)
	ctx.caller.body = append(ctx.caller.body, makeInstruction(
		opDrillTier1,
		byte(subOpJump),
		lo,
		hi,
	))
	return inlineEligible
}

// typedMoveSubOp returns the tier-1 register-to-register move sub-op.
//
// Slice banks and the complex bank have no tier-1 move; callees that use them are not
// inlineable.
//
// Takes bank (registerKind) which selects the move's bank.
//
// Returns the tier-1 sub-opcode for that bank.
// Returns false when the bank has no tier-1 move.
func typedMoveSubOp(bank registerKind) (subOpcode, bool) {
	switch bank {
	case registerInt:
		return subOpMoveInt, true
	case registerFloat:
		return subOpMoveFloat, true
	case registerString:
		return subOpMoveString, true
	case registerBool:
		return subOpMoveBool, true
	case registerUint:
		return subOpMoveUint, true
	case registerSliceInt:
		return subOpMoveSliceInt, true
	case registerSliceFloat:
		return subOpMoveSliceFloat, true
	case registerSliceString:
		return subOpMoveSliceString, true
	case registerSliceBool:
		return subOpMoveSliceBool, true
	case registerSliceUint:
		return subOpMoveSliceUint, true
	case registerSliceByte:
		return subOpMoveSliceByte, true
	default:
	}
	return 0, false
}

// remapOperands rewrites register and pool operands for a callee instruction.
//
// Register-typed operands are rewritten via ctx.remap, and any pool indices (constants,
// type table, struct layouts) are merged into the caller's tables. opDrillTier1
// instructions are handled specially: operand A holds the tier-1 sub-opcode marker (not a
// register), and operands B/C carry the sub-op's register/index payload whose meaning
// depends on the specific sub-op. The generic operandShapes table marks B/C as
// roleRegDynamic in this case, so dispatch is routed through remapTier1SubOp to get the
// correct bank per sub-op. opCall and opTailCall return after the pool remap because
// their operand bytes encode only the call-site index (no register operands); the
// register-remap step would otherwise misinterpret those bytes. Mutates ctx.caller's pool
// slices as a side effect via mergePoolIndex / mergeCallSiteForCtx.
//
// Takes instr (instruction) which is the callee-side instruction.
// Takes ctx (*inlineContext) holding caller pools and the remap table.
//
// Returns the rewritten instruction.
// Returns false on any failure (unknown shape, unmapped register, pool overflow).
func remapOperands(instr instruction, ctx *inlineContext) (instruction, bool) {
	if instr.op == opDrillTier1 {
		return remapTier1SubOp(instr, ctx)
	}
	out := instr
	poolShape := inlinePoolShapeFor(instr.op)
	if !remapPoolOperands(&out, poolShape, ctx) {
		return instruction{}, false
	}
	if instr.op == opCall || instr.op == opTailCall {
		return out, true
	}
	if !remapRegisterOperands(&out, instr.op, ctx) {
		return instruction{}, false
	}
	return out, true
}

// remapPoolOperands merges every pool-index operand on instruction into the caller's
// tables.
//
// Handles three encodings: a B-byte pool index (bKindByte != poolNone) that must fit in 8
// bits, a C-byte pool index (cKindByte != poolNone) that must fit in 8 bits, and a
// B|C<<instructionByteShift wide pool index (bcWide16 != poolNone) using the full 16-bit
// range. The poolCallSites variant routes through mergeCallSiteForCtx so register
// operands inside the merged site are remapped via ctx.
//
// Takes out (*instruction) which is mutated in place.
// Takes poolShape (inlinePoolShape) describing the encoding.
// Takes ctx (*inlineContext) holding caller pools.
//
// Returns true on success; false on overflow or unresolved merge.
func remapPoolOperands(out *instruction, poolShape inlinePoolShape, ctx *inlineContext) bool {
	if poolShape.bKindByte != poolNone {
		newIdx, ok := mergePoolIndex(ctx.caller, ctx.callee, poolShape.bKindByte, uint16(out.b), true)
		if !ok || newIdx >= byteEncodingLimit {
			return false
		}
		out.b = byte(newIdx)
	}
	if poolShape.cKindByte != poolNone {
		newIdx, ok := mergePoolIndex(ctx.caller, ctx.callee, poolShape.cKindByte, uint16(out.c), true)
		if !ok || newIdx >= byteEncodingLimit {
			return false
		}
		out.c = byte(newIdx)
	}
	if poolShape.bcWide16 != poolNone {
		return remapWidePoolOperand(out, poolShape.bcWide16, ctx)
	}
	return true
}

// remapWidePoolOperand merges a wide (B|C<<instructionByteShift) pool-index operand into
// the caller's tables.
//
// Takes out (*instruction) which is mutated in place.
// Takes pool (inlinePool) which selects the destination pool.
// Takes ctx (*inlineContext) holding caller pools and the remap table.
//
// Returns true on success; false on overflow or unresolved merge.
func remapWidePoolOperand(out *instruction, pool inlinePool, ctx *inlineContext) bool {
	oldIdx := uint16(out.b) | uint16(out.c)<<instructionByteShift
	var newIdx uint16
	var ok bool
	if pool == poolCallSites {
		newIdx, ok = mergeCallSiteForCtx(ctx, oldIdx, false)
	} else {
		newIdx, ok = mergePoolIndex(ctx.caller, ctx.callee, pool, oldIdx, false)
	}
	if !ok {
		return false
	}
	out.b = byte(newIdx & 0xFF)
	out.c = byte((newIdx >> instructionByteShift) & 0xFF)
	return true
}

// remapRegisterOperands rewrites every register byte described by the opcode's
// operandShapes entry, looking each callee slot up in the caller-side remap table.
//
// Takes out (*instruction) which is mutated in place.
// Takes op (opcode) used to fetch the operandShapes entry.
// Takes ctx (*inlineContext) holding the remap table.
//
// Returns true on success; false when the opcode lacks a `described` shape or a
// referenced register has no remap entry.
func remapRegisterOperands(out *instruction, op opcode, ctx *inlineContext) bool {
	shape := operandShapes[op]
	if shape.flags&shapeFlagDescribed == 0 {
		return false
	}
	bytePtrs := [instructionOperandCount]*byte{&out.a, &out.b, &out.c}
	roles := [instructionOperandCount]operandRole{shape.a, shape.b, shape.c}
	for pos, role := range roles {
		bank, isReg := kindForRole(role)
		if !isReg {
			continue
		}
		oldSlot := *bytePtrs[pos] //nolint:gosec // pos bounded to len(roles)==instructionOperandCount by range
		newSlot, ok := ctx.lookupRegister(bank, oldSlot)
		if !ok {
			return false
		}
		*bytePtrs[pos] = newSlot //nolint:gosec // pos bounded to len(roles)==instructionOperandCount by range
	}
	return true
}

// remapExtensionOperands rewrites a follow-on opExt extension word.
//
// The extension's content is per-opcode: most carry a uint16 pool index in operands A|B,
// some carry a register reference, and some are pure immediates. The shape parameter
// selects which interpretation applies.
//
// Takes ext (instruction) which is the extension word emitted after its parent opcode.
// Takes shape (inlinePoolShape) describing the extension's operand layout.
// Takes ctx (*inlineContext) holding caller pools and the remap table.
//
// Returns the rewritten extension instruction.
// Returns false on any failure (pool overflow, unmapped register).
func remapExtensionOperands(ext instruction, shape inlinePoolShape, ctx *inlineContext) (instruction, bool) {
	out := ext
	if shape.extAWide16 != poolNone {
		oldIdx := uint16(out.a) | uint16(out.b)<<instructionByteShift
		newIdx, ok := mergePoolIndex(ctx.caller, ctx.callee, shape.extAWide16, oldIdx, false)
		if !ok {
			return instruction{}, false
		}
		out.a = byte(newIdx & 0xFF)
		out.b = byte((newIdx >> instructionByteShift) & 0xFF)
	}
	if shape.extARegSet {
		newSlot, ok := ctx.lookupRegister(shape.extARegBank, out.a)
		if !ok {
			return instruction{}, false
		}
		out.a = newSlot
	}
	return out, true
}

// remapTier1SubOp rewrites the register payloads of an opDrillTier1 instruction.
//
// Operand A names the sub-opcode (not a register); operands B and C carry the sub-op's
// per-bank register payload, with the bank determined by the sub-op. The five typed move
// sub-ops (Int, Float, String, Bool, Uint) treat B as the destination register and C as
// the source register in their named bank. subOpJump encodes a signed 16-bit offset in
// B|C<<8 with no register. subOpLoadIntConstSmall and subOpLoadBool use B as an int-bank
// destination with C as a literal. Sub-ops not listed here are refused.
//
// Takes instr (instruction) which is the opDrillTier1 instruction.
// Takes ctx (*inlineContext) holding the remap table.
//
// Returns the rewritten instruction.
// Returns false when a referenced register has no remap entry or the sub-op is
// unsupported.
func remapTier1SubOp(instr instruction, ctx *inlineContext) (instruction, bool) {
	out := instr
	sub := subOpcode(instr.a)
	switch sub {
	case subOpJump:
		return out, true
	case subOpMoveInt:
		return remapRegBC(&out, ctx, registerInt, registerInt)
	case subOpMoveFloat:
		return remapRegBC(&out, ctx, registerFloat, registerFloat)
	case subOpMoveString:
		return remapRegBC(&out, ctx, registerString, registerString)
	case subOpMoveBool:
		return remapRegBC(&out, ctx, registerBool, registerBool)
	case subOpMoveUint:
		return remapRegBC(&out, ctx, registerUint, registerUint)
	case subOpLoadIntConstSmall, subOpLoadBool:
		return remapTier1BWithIntDest(&out, ctx)
	case subOpDrillTier2:
		return remapTier2SubOp(&out, ctx)
	default:
	}
	return instruction{}, false
}

// remapTier1BWithIntDest remaps operand B as an int-bank destination register for sub-ops
// whose operand C is a literal (no remap).
//
// Takes out (*instruction) which is mutated in place when the lookup succeeds.
// Takes ctx (*inlineContext) holding the remap table.
//
// Returns the updated instruction by value (also reflected in *out).
// Returns false when the destination register has no remap entry.
func remapTier1BWithIntDest(out *instruction, ctx *inlineContext) (instruction, bool) {
	newB, ok := remapRegByte(out.b, ctx, registerInt)
	if !ok {
		return instruction{}, false
	}
	out.b = newB
	return *out, true
}

// remapTier2SubOp rewrites the register payloads of a subOpDrillTier2 instruction.
// Operand B carries the tier-2 sub-op; operand C is its register operand (banked per
// sub-op).
//
// Takes out (*instruction) which is mutated in place.
// Takes ctx (*inlineContext) holding the remap table.
//
// Returns the rewritten instruction.
// Returns false on unsupported sub-ops or unresolved register lookups.
func remapTier2SubOp(out *instruction, ctx *inlineContext) (instruction, bool) {
	tier2 := subOpcodeTier2(out.b)
	switch tier2 {
	case subOpTier2DrillTier3:
		return *out, true
	case subOpTier2IncInt, subOpTier2DecInt:
		return remapTier2OperandC(out, ctx, registerInt)
	case subOpTier2IncUint, subOpTier2DecUint:
		return remapTier2OperandC(out, ctx, registerUint)
	case subOpTier2LoadNil, subOpTier2SetZero:
		return remapTier2OperandC(out, ctx, registerGeneral)
	default:
	}
	return instruction{}, false
}

// remapTier2OperandC remaps operand C of a tier-2 sub-op as a register in the named bank.
//
// Takes out (*instruction) which is mutated in place when the lookup succeeds.
// Takes ctx (*inlineContext) holding the remap table.
// Takes bank (registerKind) which is the bank carrying operand C.
//
// Returns the updated instruction by value (also reflected in *out).
// Returns false when the register has no remap entry.
func remapTier2OperandC(out *instruction, ctx *inlineContext, bank registerKind) (instruction, bool) {
	newC, ok := remapRegByte(out.c, ctx, bank)
	if !ok {
		return instruction{}, false
	}
	out.c = newC
	return *out, true
}

// remapRegBC remaps operand B and operand C as registers in the given banks.
//
// Takes out (*instruction) which is mutated in place when both lookups succeed.
// Takes ctx (*inlineContext) holding the remap table.
// Takes bankB (registerKind) which is the bank for operand B.
// Takes bankC (registerKind) which is the bank for operand C.
//
// Returns the updated instruction by value (also reflected in *out).
// Returns false when either register has no remap entry.
func remapRegBC(out *instruction, ctx *inlineContext, bankB, bankC registerKind) (instruction, bool) {
	newB, okB := remapRegByte(out.b, ctx, bankB)
	if !okB {
		return instruction{}, false
	}
	newC, okC := remapRegByte(out.c, ctx, bankC)
	if !okC {
		return instruction{}, false
	}
	out.b = newB
	out.c = newC
	return *out, true
}

// remapRegByte looks up a callee register byte in the remap table.
//
// Takes calleeSlot (byte) which is the callee-side register index.
// Takes ctx (*inlineContext) holding the remap table.
// Takes bank (registerKind) which selects the bank.
//
// Returns the caller-side register byte.
// Returns false when no mapping exists for the (bank, calleeSlot) pair.
func remapRegByte(calleeSlot byte, ctx *inlineContext, bank registerKind) (byte, bool) {
	return ctx.lookupRegister(bank, calleeSlot)
}

// isReturnInstruction reports whether instr is a return encoding.
//
// Matches the two return sub-op encodings: opDrillTier1 with a=subOpDrillTier2 and
// b=subOpTier2Return (c=N), or opDrillTier1 with a=subOpDrillTier2,
// b=subOpTier2DrillTier3, c=subOpTier3ReturnVoid.
//
// Takes instr (instruction) which is the candidate instruction.
//
// Returns true when instr is a return encoding.
func isReturnInstruction(instr instruction) bool {
	if instr.op != opDrillTier1 {
		return false
	}
	if subOpcode(instr.a) != subOpDrillTier2 {
		return false
	}
	switch subOpcodeTier2(instr.b) {
	case subOpTier2Return:
		return true
	case subOpTier2DrillTier3:
		return subOpcodeTier3(instr.c) == subOpTier3ReturnVoid
	default:
	}
	return false
}

// fitsJumpOffset reports whether offset fits the signed 16-bit jump encoding.
//
// Takes offset (int) which is the prospective PC-relative jump offset.
//
// Returns true when the offset fits within [math.MinInt16, math.MaxInt16].
func fitsJumpOffset(offset int) bool {
	return offset >= math.MinInt16 && offset <= math.MaxInt16
}

// splitJumpOffsetBytes splits a signed offset into the B and C bytes.
//
// Mirrors the internal splitOffset helper but kept local to the inliner.
//
// Takes offset (int) which is the signed PC-relative offset.
//
// Returns lowByte (the low operand-B byte of the encoded offset).
// Returns highByte (the high operand-C byte of the encoded offset).
func splitJumpOffsetBytes(offset int) (lowByte, highByte byte) {
	raw := safeconv.Int16ToUint16(safeconv.MustIntToInt16(offset))
	return byte(raw & 0xFF), byte((raw >> instructionByteShift) & 0xFF)
}
