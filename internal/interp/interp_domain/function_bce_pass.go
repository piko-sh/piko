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
	"piko.sh/piko/wdk/safeconv"
)

const (
	// minBceBodyLength is the smallest function body the BCE pass can usefully analyse;
	// three instructions are required to form the minimal LEN, LT, JUMP_IF_FALSE pattern
	// that establishes a proof.
	minBceBodyLength = 3

	// bceGenBankFlag tags a lengthOf or safeIndex entry as belonging to the reflect
	// (general) register bank.
	//
	// The low byte still holds (sliceReg + 1); the high bit makes the typed-bank rewrite
	// path skip reflect-bank facts and vice versa so the two banks share one state table
	// without colliding on register numbers that exist in both.
	bceGenBankFlag uint16 = 0x8000
)

// elideRedundantBoundsChecks rewrites bounds-checked slice / string access opcodes to
// their Unchecked variants at PCs where a prior conditional-jump pattern proves the index
// is in range.
//
// Pattern recognised:
//
//	1:LEN_SLICE_INT_DIRECT  lenReg, sliceReg        ; lenReg = len(slice)
//	... (linear straight-line code preserving lenReg and sliceReg) ...
//	0:LT_INT                condReg, idxReg, lenReg ; condReg = idx < len
//	0:JUMP_IF_FALSE         condReg, offset         ; exit on out-of-range
//	; fall-through here: idxReg < len(slice[sliceReg])
//	0:SLICE_GET_INT_DIRECT  dest, sliceReg, idxReg  ; <- rewritten to Unchecked
//
// State tracked per-PC: lengthOf[lenReg] is the slice register whose length lives in
// ints[lenReg] (0 means "no fact"; otherwise sliceReg + 1), and safeIndex[idxReg] is the
// slice register whose length bounds ints[idxReg] (0 means "no fact"; otherwise sliceReg
// + 1).
//
// State invalidation: jump targets clear all state (predecessors may have left different
// register contents); any write to ints[X] kills lengthOf[X] and safeIndex[X]; any write
// to slicesInt[X] kills any lengthOf[Y] and safeIndex[Y] that name X (the slice's length
// may have changed); and calls, append, and slice-operations clear all state
// conservatively.
//
// Takes body ([]instruction) which is the function's instruction stream rewritten in
// place.
func (cf *CompiledFunction) elideRedundantBoundsChecks(body []instruction) {
	if !peepholeBceEnabled {
		return
	}
	if len(body) < minBceBodyLength {
		return
	}
	jumpTargets := buildAllJumpTargets(body)
	var lengthOf [generalRegisterBankSize]uint16
	var safeIndex [generalRegisterBankSize]uint16
	var proofPCOf [generalRegisterBankSize]uint16
	var nonNegative [generalRegisterBankSize]bool
	tracker := newDerefStabilityTracker()
	for pc := range body {
		if jumpTargets[pc] {
			lengthOf = [generalRegisterBankSize]uint16{}
			safeIndex = [generalRegisterBankSize]uint16{}
			proofPCOf = [generalRegisterBankSize]uint16{}
			nonNegative = [generalRegisterBankSize]bool{}
			tracker.reset()
		}
		inst := body[pc]
		cf.rewriteIfBoundsSafe(body, pc, &safeIndex, &proofPCOf, &nonNegative)
		invalidateBoundsFacts(cf, inst, &lengthOf, &safeIndex, &tracker)
		invalidateProofPCsForCleared(&safeIndex, &proofPCOf)
		invalidateNonNegativeFacts(cf, inst, &nonNegative)
		recordLenFact(inst, &lengthOf, &safeIndex)
		recordNonNegativeFact(cf, body, pc, &nonNegative)
		recordLtJumpFact(body, pc, &lengthOf, &safeIndex, &proofPCOf)
	}
}

// derefStabilityTracker tracks idempotent DEREF writes to general-bank registers.
//
// It remembers the last byte-pattern that wrote each general-bank register together with
// a monotonic version that increments on every non-idempotent general-bank write. The
// stable-DEREF rule lets a subsequent byte-identical DEREF preserve length facts about
// its destination register: when both DEREFs are identical and the source operand
// register has not been written since the first DEREF, the destination's value is
// unchanged.
//
// Without this, every (*p)[i] access compiles to DEREF then SLICE_GET_INT, and the
// DEREF's write to the slice register invalidates the BCE proof established by the prior
// len(*p) call, killing the rewrite on every reflect-bank pointer-to-slice pattern
// (dijkstra's heap, 17_closures pipelines, etc.).
type derefStabilityTracker struct {
	// lastProducer holds the most recent instruction that wrote each general-bank register.
	lastProducer [generalRegisterBankSize]instruction

	// writeVersion holds the monotonic version stamp of the most recent non-idempotent write
	// per register.
	writeVersion [generalRegisterBankSize]uint32

	// nextVersion is the version stamp assigned to the next non-idempotent write.
	nextVersion uint32
}

// reset clears tracker state at basic-block boundaries (jump targets), matching the rest
// of the BCE pass's reset semantics.
func (t *derefStabilityTracker) reset() {
	t.lastProducer = [generalRegisterBankSize]instruction{}
	t.writeVersion = [generalRegisterBankSize]uint32{}
	t.nextVersion = 1
}

// isIdempotentGeneralWrite reports whether writing inst to general[reg] reproduces the
// value already held there.
//
// Restricted to opDeref, the only single-general-source op the compiler emits
// idempotently on hot reflect-bank paths. Adding further ops here requires the same
// source-stability proof: the instruction's general-bank source operand must not have
// been written since the previous write of inst to general[reg].
//
// Takes inst (instruction) which is the candidate instruction about to write
// general[reg], and reg (uint8) which is the destination general-bank register index.
//
// Returns true when the write reproduces the existing value.
func (t *derefStabilityTracker) isIdempotentGeneralWrite(inst instruction, reg uint8) bool {
	if inst.op != opDeref {
		return false
	}
	if inst != t.lastProducer[reg] {
		return false
	}
	return t.writeVersion[inst.b] <= t.writeVersion[reg]
}

// recordGeneralWrite snapshots a non-idempotent write to general[reg] so subsequent
// idempotency checks have a producer to compare against.
//
// Bumps nextVersion so the source-stability test for any other tracked register sees that
// a real write has intervened.
//
// Takes inst (instruction) which is the instruction that performed the write, and reg
// (uint8) which is the destination general-bank register index.
func (t *derefStabilityTracker) recordGeneralWrite(inst instruction, reg uint8) {
	t.lastProducer[reg] = inst
	t.writeVersion[reg] = t.nextVersion
	t.nextVersion++
}

// rewriteIfBoundsSafe rewrites a checked slice access at pc to its Unchecked variant when
// the safeIndex table contains a matching (index register, slice register, bank) tuple.
//
// Rewritten ops: opSliceGetIntDirect and opSliceSetIntDirect (typed slicesInt bank);
// opSliceGetInt and opSliceSetInt (reflect general bank).
//
// Records a peepholeRewriteBce annotation whose origin is the PC of the proof source (the
// LtInt/JumpIfFalse pair that established the in-range fact for the index register). When
// the proof was seeded by the range-loop bridge rather than by the dataflow pass,
// proofPCOf is zero and origin falls back to pc.
//
// The Unchecked rewrite fires only when the index register is BOTH recorded in safeIndex
// (proven idx < len) AND flagged in nonNegative (proven idx >= 0). The signed opLtInt
// comparison behind safeIndex does not exclude negative indices, so the second proof is
// required for soundness; without it the rewrite is refused.
//
// Takes body which is the instruction stream rewritten in place, pc which is the index of
// the candidate access instruction, safeIndex which is the safe-index fact table
// consulted for the proof, proofPCOf which is the proof-source PC table used for
// annotation origins, and nonNegative which is the per-register non-negativity table that
// gates the rewrite.
func (cf *CompiledFunction) rewriteIfBoundsSafe(
	body []instruction,
	pc int,
	safeIndex *[generalRegisterBankSize]uint16,
	proofPCOf *[generalRegisterBankSize]uint16,
	nonNegative *[generalRegisterBankSize]bool,
) {
	inst := body[pc]
	idxReg := uint8(0)
	switch inst.op {
	case opSliceGetIntDirect:
		if safeIndex[inst.c] != uint16(inst.b)+1 || !nonNegative[inst.c] {
			return
		}
		idxReg = inst.c
		body[pc] = makeInstruction(opSliceGetIntDirectUnchecked, inst.a, inst.b, inst.c)
	case opSliceSetIntDirect:
		if safeIndex[inst.b] != uint16(inst.a)+1 || !nonNegative[inst.b] {
			return
		}
		idxReg = inst.b
		body[pc] = makeInstruction(opSliceSetIntDirectUnchecked, inst.a, inst.b, inst.c)
	case opSliceGetInt:
		if safeIndex[inst.c] != uint16(inst.b)+1|bceGenBankFlag || !nonNegative[inst.c] {
			return
		}
		idxReg = inst.c
		body[pc] = makeInstruction(opSliceGetIntUnchecked, inst.a, inst.b, inst.c)
	case opSliceSetInt:
		if safeIndex[inst.b] != uint16(inst.a)+1|bceGenBankFlag || !nonNegative[inst.b] {
			return
		}
		idxReg = inst.b
		body[pc] = makeInstruction(opSliceSetIntUnchecked, inst.a, inst.b, inst.c)
	default:
		return
	}
	origin := pc
	if proofPCOf[idxReg] != 0 {
		origin = int(proofPCOf[idxReg]) - 1
	}
	cf.recordPeepholeRewrite(pc, peepholeRewriteBce, origin)
}

// recordNonNegativeFact marks an int-bank register as provably non-negative when the
// instruction at pc establishes a lower bound of zero or greater.
//
// The bounds proof recorded by recordLtJumpFact comes from a signed opLtInt comparison,
// so a negative index satisfies idx < len yet is still out of range. The Unchecked
// rewrite must therefore also see the index proven non-negative, otherwise a negative
// index produces a differently routed panic (the checked variant goes through
// vm.evalError; the Unchecked variant lets the Go runtime or reflect raise the panic).
//
// Recognised non-negativity sources are all conservative: subOpLoadIntConstSmall loads an
// immediate in 0..255 which is always non-negative, and opLoadIntConst records the fact
// only when the pool index is in range and the constant is >= 0. Other patterns
// (range-induction variables, explicit idx >= 0 guards) are not modelled; their absence
// only costs optimisation.
//
// Takes cf which is the compiled function supplying the constant pool, body which is the
// instruction stream, pc which is the current program counter, and nonNegative which is
// the per-register non-negativity table updated in place.
func recordNonNegativeFact(cf *CompiledFunction, body []instruction, pc int, nonNegative *[generalRegisterBankSize]bool) {
	inst := body[pc]
	if inst.op == opDrillTier1 && subOpcode(inst.a) == subOpLoadIntConstSmall {
		nonNegative[inst.b] = true
		return
	}
	if inst.op != opLoadIntConst || inst.c != 0 {
		return
	}
	poolIndex := int(inst.b)
	if poolIndex < 0 || poolIndex >= len(cf.intConstants) {
		return
	}
	if cf.intConstants[poolIndex] >= 0 {
		nonNegative[inst.a] = true
	}
}

// invalidateNonNegativeFacts clears the non-negativity fact for every int-bank register
// that inst writes.
//
// Any write to an int register may install a value of unknown sign, so the fact is
// dropped unless recordNonNegativeFact re-establishes it for this same instruction. Calls
// and other broad-effect instructions are covered because instructionWritesRegisterInBank
// reports their typed-bank destinations; the conservative direction is to drop the fact,
// which only blocks optimisation.
//
// Takes cf which is the compiled function (unused but kept for signature symmetry with
// the other invalidation helpers), inst which is the instruction whose writes are
// applied, and nonNegative which is the per-register non-negativity table updated in
// place.
func invalidateNonNegativeFacts(cf *CompiledFunction, inst instruction, nonNegative *[generalRegisterBankSize]bool) {
	for reg := range generalRegisterBankSize {
		if !nonNegative[reg] {
			continue
		}
		if instructionWritesRegisterInBank(inst, roleRegInt, uint8(reg)) {
			nonNegative[reg] = false
		}
	}
	_ = cf
}

// newDerefStabilityTracker builds a tracker primed so the first real DEREF write
// registers as fresh.
//
// The zero-value instruction is opDrillTier1, which is never a DEREF, so the first real
// DEREF write never matches the all-zero default producer.
//
// Returns a tracker ready for use at the start of a basic block.
func newDerefStabilityTracker() derefStabilityTracker {
	return derefStabilityTracker{nextVersion: 1}
}

// invalidateProofPCsForCleared clears the proofPCOf entry for any safeIndex slot that the
// invalidation step cleared.
//
// Keeps the proof table tightly in sync with the fact table.
//
// Takes safeIndex which is the safe-index fact table, and proofPCOf which is the
// proof-source PC table for each safeIndex slot.
func invalidateProofPCsForCleared(safeIndex *[generalRegisterBankSize]uint16, proofPCOf *[generalRegisterBankSize]uint16) {
	for k := range safeIndex {
		if safeIndex[k] == 0 {
			proofPCOf[k] = 0
		}
	}
}

// recordLenFact captures lengthOf[lenReg] = sliceReg + 1 when inst is one of the
// recognised length opcodes.
//
// Recognised forms (all tier-1 sub-ops via opDrillTier1): subOpLenSliceIntDirect where
// ints[B] = len(slicesInt[C]) sets a typed-bank fact, and subOpLen where ints[B] =
// len(general[C]) sets a reflect (general) bank fact.
//
// Reflect-bank facts are tagged with bceGenBankFlag so the rewrite step keeps the two
// banks distinct (register numbers in slicesInt and general are independent address
// spaces). Other typed-direct len ops (Float, String, Bool, Uint) are not recognised
// because the BCE rewrites target only int-element slices.
//
// Takes inst (instruction) which is the candidate instruction to inspect, lengthOf which
// is the length-fact table to update, and safeIndex which is the safe-index fact table to
// clear when lenReg is overwritten.
func recordLenFact(inst instruction, lengthOf *[generalRegisterBankSize]uint16, safeIndex *[generalRegisterBankSize]uint16) {
	if inst.op != opDrillTier1 {
		return
	}
	switch subOpcode(inst.a) {
	case subOpLenSliceIntDirect:
		lenReg := inst.b
		sliceReg := inst.c
		clearFactsTargetingLenReg(lengthOf, safeIndex, lenReg)
		lengthOf[lenReg] = uint16(sliceReg) + 1
	case subOpLen:
		lenReg := inst.b
		sliceReg := inst.c
		clearFactsTargetingLenReg(lengthOf, safeIndex, lenReg)
		lengthOf[lenReg] = uint16(sliceReg) + 1 | bceGenBankFlag
	default:
	}
}

// recordLtJumpFact captures safeIndex[idxReg] = sliceReg + 1 when the two-instruction
// window starting at pc matches the bounds-proof pattern.
//
// The pattern is opLtInt followed by opJumpIfFalse with a matching condition register,
// and the right-hand side of the comparison must resolve to a known length via lengthOf.
//
// On the fall-through edge (subsequent PCs in linear order), the index register is
// provably less than the slice's length.
//
// Takes body which is the instruction stream being analysed, pc which is the start index
// of the candidate two-instruction window, lengthOf which is the length-fact table
// consulted to verify the comparison, safeIndex which is the safe-index fact table
// updated on success, and proofPCOf which is the proof-source PC table recorded for later
// attribution.
func recordLtJumpFact(body []instruction, pc int, lengthOf *[generalRegisterBankSize]uint16, safeIndex *[generalRegisterBankSize]uint16, proofPCOf *[generalRegisterBankSize]uint16) {
	if pc+1 >= len(body) {
		return
	}
	cmp := body[pc]
	jmp := body[pc+1]
	if cmp.op != opLtInt {
		return
	}
	if jmp.op != opJumpIfFalse {
		return
	}
	if cmp.a != jmp.a {
		return
	}
	lenReg := cmp.c
	idxReg := cmp.b
	if lengthOf[lenReg] == 0 {
		return
	}
	safeIndex[idxReg] = lengthOf[lenReg]
	proofPCOf[idxReg] = safeconv.IntToUint16(pc) + 1
}

// invalidateBoundsFacts kills facts whose underlying registers were just written, or
// clears all state when the instruction has effects the analysis cannot model.
//
// Takes cf which is the compiled function used by per-bank register-write helpers, inst
// which is the instruction whose effects are being applied, lengthOf which is the
// length-fact table to update, safeIndex which is the safe-index fact table to update,
// and tracker which is the deref-stability tracker used to preserve idempotent re-DEREF
// facts.
func invalidateBoundsFacts(cf *CompiledFunction, inst instruction, lengthOf *[generalRegisterBankSize]uint16, safeIndex *[generalRegisterBankSize]uint16, tracker *derefStabilityTracker) {
	if instructionClearsAllBoundsFacts(inst) {
		*lengthOf = [generalRegisterBankSize]uint16{}
		*safeIndex = [generalRegisterBankSize]uint16{}
		tracker.reset()
		return
	}
	invalidateIntRegisterFacts(cf, inst, lengthOf, safeIndex)
	invalidateSliceIntRegisterFacts(cf, inst, lengthOf, safeIndex)
	invalidateGeneralRegisterFacts(cf, inst, lengthOf, safeIndex, tracker)
}

// instructionClearsAllBoundsFacts reports whether inst has effects the BCE analysis
// cannot model precisely.
//
// The block list covers calls (callee may mutate any slice's length or any register),
// append (changes len), and slice-operation primitives that materially reshape slice
// headers. The pass clears all tracked facts when this returns true.
//
// Takes inst (instruction) which is the instruction to classify.
//
// Returns true when inst belongs to the conservative block list.
func instructionClearsAllBoundsFacts(inst instruction) bool {
	switch inst.op {
	case opCall, opTailCall, opCallMethod, opCallMethodInlineable, opCallNative, opCallIIFE,
		opMakeClosure, opAppend, opAppendSpread, opAppendByteFast, opAppendByteFastInPlace,
		opAppendInPlace, opAppendSpreadInPlace,
		opSliceString:
		return true
	default:
	}
	if inst.op == opDrillTier1 {
		switch subOpcode(inst.a) {
		case subOpAppendInt, subOpAppendUint, subOpAppendFloat,
			subOpAppendString, subOpAppendBool,
			subOpStarAppendByteFast, subOpStarAppendByteSpread,
			subOpDrillTier2:
			return true
		default:
		}
	}
	return false
}

// invalidateIntRegisterFacts kills lengthOf[X] and safeIndex[X] for every int-bank
// register X that inst writes.
//
// The CSE pass's register-write detection is reused so the analysis stays consistent with
// downstream peephole passes.
//
// Takes cf which is the compiled function used by the register-write helper, inst which
// is the instruction whose writes are being applied, lengthOf which is the length-fact
// table to update, and safeIndex which is the safe-index fact table to update.
func invalidateIntRegisterFacts(cf *CompiledFunction, inst instruction, lengthOf *[generalRegisterBankSize]uint16, safeIndex *[generalRegisterBankSize]uint16) {
	for reg := range generalRegisterBankSize {
		if lengthOf[reg] == 0 && safeIndex[reg] == 0 {
			continue
		}
		if instructionWritesRegisterInBank(inst, roleRegInt, uint8(reg)) {
			lengthOf[reg] = 0
			safeIndex[reg] = 0
		}
	}
	_ = cf
}

// invalidateSliceIntRegisterFacts kills any fact whose underlying sliceReg matches a
// register written by inst in the slicesInt bank.
//
// A write to slicesInt[X] (for example via opMakeSliceInt) can change the slice's length,
// so any lengthOf and safeIndex entry naming X is stale.
//
// Takes cf which is the compiled function used by the register-write helper, inst which
// is the instruction whose writes are being applied, lengthOf which is the length-fact
// table to update, and safeIndex which is the safe-index fact table to update.
func invalidateSliceIntRegisterFacts(cf *CompiledFunction, inst instruction, lengthOf *[generalRegisterBankSize]uint16, safeIndex *[generalRegisterBankSize]uint16) {
	for reg := range generalRegisterBankSize {
		if !instructionWritesRegisterInBank(inst, roleRegSliceInt, uint8(reg)) {
			continue
		}
		target := uint16(reg) + 1
		for k := range lengthOf {
			if lengthOf[k] == target {
				lengthOf[k] = 0
			}
		}
		for k := range safeIndex {
			if safeIndex[k] == target {
				safeIndex[k] = 0
			}
		}
	}
	_ = cf
}

// clearFactsTargetingLenReg removes any fact whose target register is lenReg, called
// before a fresh length fact is installed at lenReg.
//
// Walks safeIndex masked against bceGenBankFlag so entries for either bank tagged with
// the same slice register number are cleared together; the safeIndex slot is keyed by
// int-register (the index register) which is bank-free, so no further disambiguation is
// needed.
//
// Takes lengthOf which is the length-fact table to clear, safeIndex which is the
// safe-index fact table to clear, and lenReg which is the int-bank register about to
// receive a new length fact.
func clearFactsTargetingLenReg(lengthOf *[generalRegisterBankSize]uint16, safeIndex *[generalRegisterBankSize]uint16, lenReg uint8) {
	lengthOf[lenReg] = 0
	target := uint16(lenReg) + 1
	for k := range safeIndex {
		if safeIndex[k]&^bceGenBankFlag == target {
			safeIndex[k] = 0
		}
	}
}

// invalidateGeneralRegisterFacts kills reflect-bank facts whose sliceReg matches a
// register written by inst in the general bank.
//
// A write to general[X] (for example via opMakeSlice into an addressable destination) can
// change the slice's length, so any lengthOf and safeIndex entry naming X in the reflect
// bank is stale, except when the write is an idempotent re-DEREF; see
// derefStabilityTracker for the soundness argument. Idempotent writes preserve the value
// general[X] already held, so any length fact pointing at general[X] remains valid.
//
// Takes cf which is the compiled function used by the register-write helper, inst which
// is the instruction whose writes are being applied, lengthOf which is the length-fact
// table to update, safeIndex which is the safe-index fact table to update, and tracker
// which is the deref-stability tracker consulted for idempotent writes.
func invalidateGeneralRegisterFacts(cf *CompiledFunction, inst instruction, lengthOf *[generalRegisterBankSize]uint16, safeIndex *[generalRegisterBankSize]uint16, tracker *derefStabilityTracker) {
	for reg := range generalRegisterBankSize {
		if !instructionWritesRegisterInBank(inst, roleRegGeneral, uint8(reg)) {
			continue
		}
		if tracker.isIdempotentGeneralWrite(inst, uint8(reg)) {
			continue
		}
		target := uint16(reg) + 1 | bceGenBankFlag
		for k := range lengthOf {
			if lengthOf[k] == target {
				lengthOf[k] = 0
			}
		}
		for k := range safeIndex {
			if safeIndex[k] == target {
				safeIndex[k] = 0
			}
		}
		tracker.recordGeneralWrite(inst, uint8(reg))
	}
	_ = cf
}
