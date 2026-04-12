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
	"cmp"
	"context"
	"fmt"
	"slices"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// maxLicmHoistsPerFunction caps the number of struct-field reads that the loop-invariant
	// code motion pass will lift per CompiledFunction. Each hoist allocates one instruction
	// word in the body and may inflate register lifetimes; capping the count bounds the
	// cost.
	maxLicmHoistsPerFunction = 8

	// initialLoopCapacity is the initial size of the loops slice used by identifyLoops. Most
	// functions in practice contain a small number of loops, so pre-allocating four avoids
	// the early grow steps.
	initialLoopCapacity = 4

	// maxLoopDominatorIterations caps the iterative dataflow loop used by
	// computeBackEdgeDominators. The bitset lattice converges quickly for real loop bodies;
	// the cap protects against pathological CFGs.
	maxLoopDominatorIterations = 32
)

// hoistLoopInvariantStructFieldReads lifts loop-invariant field reads to the pre-header.
//
// It targets tier-0 general-bank struct-field reads (opGetStructFieldGeneral and
// opGetStructFieldRawPointerT0) located in the loop header basic block (the straight-line
// prefix ending at the first internal jump or jump target) when the read's result is
// provably constant across the loop body. The hoisted instruction is inserted at the loop
// header PC, the original becomes opNop, and jump offsets that cross the insertion point
// are adjusted so the back-edge skips the hoisted instruction on subsequent iterations.
//
// The invariance gate refuses the hoist whenever the receiver register is written
// anywhere in the loop body, the destination register is written by any other instruction
// in the loop body, the body contains any opCall, opTailCall, opCallMethod, opCallNative
// or opCallIIFE (callees may mutate the receiver's heap object), any opSet* or opSwap*
// targets the same (receiver, layout), or any opDrillTier1 sub-op on the heap-mutator
// block list fires (subOpSetStructField*, subOpIncStructField*, subOpMapDelete,
// subOpChannelReceive, subOpAppend*, subOpStarAppend*, subOpSpill, subOpReload,
// subOpDrillTier2; the same list the CSE pass uses).
//
// The pass runs to fixed-point: each successful hoist shifts every PC at or after the
// insertion site by +1, so the inner loop re-derives loop structure on the post-hoist
// body. At most maxLicmHoistsPerFunction lifts per call. Must be invoked before any pass
// that depends on stable instruction indices (precomputed alloc counts, source-map
// clean-up).
//
// Takes ctx (context.Context) which threads cancellation.
//
// Returns error when cancellation interrupts the pass.
func (cf *CompiledFunction) hoistLoopInvariantStructFieldReads(ctx context.Context) error {
	if !peepholeLicmEnabled {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("hoistLoopInvariantStructFieldReads cancelled: %w", err)
	}
	for range maxLicmHoistsPerFunction {
		if !cf.tryOneStructFieldHoist() {
			return nil
		}
	}
	return nil
}

// tryOneStructFieldHoist performs at most one hoist on cf.body.
//
// Returns true when a hoist was applied, false when no hoistable read is found anywhere
// in the body.
func (cf *CompiledFunction) tryOneStructFieldHoist() bool {
	loops := identifyLoops(cf.body)
	for _, loop := range loops {
		if !loopHeaderReachableByFallThrough(cf.body, loop) {
			continue
		}
		readPC, ok := cf.findHoistableReadInLoop(loop)
		if !ok {
			continue
		}
		cf.applyStructFieldHoist(loop.header, readPC)
		return true
	}
	return false
}

// loopRange describes a natural loop discovered from a back-edge.
type loopRange struct {
	// header is the PC at the top of the loop body (the back-edge's target). Linear
	// fall-through enters the loop here.
	header int

	// latch is the PC of the back-edge instruction (the source of the negative-offset jump).
	// The loop body covers [header, latch].
	latch int
}

// findHoistableReadInLoop locates the first hoistable read dominating the back-edge.
//
// A read at PC X dominates the back-edge when every path from the loop header to the
// back-edge passes through X. Equivalently, X is reached on every iteration that
// completes the loop. Hoisting such a read out of the loop preserves the read's effect on
// register state without running the read on iterations that wouldn't have executed it
// (the "speculation problem").
//
// The pass uses an iterative dominator-set computation restricted to PCs inside
// [loop.header, loop.latch], treating predecessors as the linear fall-through and any
// in-loop branch source. Reads outside dom[latch] are conservatively refused.
//
// Takes loop which is the natural loop within which to search for a hoistable read.
//
// Returns the first dominating read's PC and true on success; (0, false) otherwise.
func (cf *CompiledFunction) findHoistableReadInLoop(loop loopRange) (int, bool) {
	body := cf.body
	dominatingBackEdge := computeBackEdgeDominators(body, loop)
	if dominatingBackEdge == nil {
		return 0, false
	}
	for pc := loop.header; pc <= loop.latch; pc++ {
		if !dominatingBackEdge[pc] {
			continue
		}
		if !isHoistableReadAt(body, pc) {
			continue
		}
		if !cf.readIsLoopInvariant(loop, pc) {
			continue
		}
		return pc, true
	}
	return 0, false
}

// readIsLoopInvariant reports whether a struct-field read is loop-invariant.
//
// The read at readPC qualifies when its value does not change across iterations of the
// loop in [loop.header, loop.latch]. It checks that the receiver register is not written
// by any instruction in the loop body, that the destination register is not written by
// any other instruction in the loop body (the hoist relies on the register holding the
// cached value across iterations), and that no instruction in the loop body mutates the
// heap such that the receiver's field could change. The check skips every word the read
// itself occupies (one for tier-0, two for tier-1 with EXT).
//
// Calls consult cf's heapMutationClass via invalidatesCachedFieldReads, so callees
// classified as heapPureCallee do not invalidate the hoist, unlocking LICM on loops that
// contain pure helper calls. Struct-field writes are reconsidered through
// structFieldWriteWithKnownReceiverDistinct so a write whose receiver provably does not
// alias the cached read's receiver does not invalidate either.
//
// Takes loop which is the natural loop being analysed.
// Takes readPC which is the PC of the candidate read.
//
// Returns true when the read is loop-invariant and safe to hoist.
func (cf *CompiledFunction) readIsLoopInvariant(loop loopRange, readPC int) bool {
	body := cf.body
	receiverReg := hoistedReadReceiverRegister(body, readPC)
	destReg := hoistedReadDestRegister(body, readPC)
	destBank := hoistedReadDestBank(body, readPC)
	width := hoistedReadWordCount(body, readPC)
	for pc := loop.header; pc <= loop.latch; pc++ {
		if pc >= readPC && pc < readPC+width {
			continue
		}
		if !cf.loopBodyInstructionPreservesRead(pc, receiverReg, destReg, destBank) {
			return false
		}
	}
	return true
}

// loopBodyInstructionPreservesRead reports whether the instruction at pc keeps a hoisted
// struct-field read valid. The instruction must not mutate the read's heap state, must
// have a described operand shape, and must not clobber the receiver or destination
// register.
//
// Takes pc which is the PC of the loop-body instruction being analysed.
// Takes receiverReg which is the general-bank register holding the read's receiver.
// Takes destReg which is the destination register the hoisted read writes.
// Takes destBank which is the operand bank role of destReg.
//
// Returns true when the instruction at pc preserves the hoisted read.
func (cf *CompiledFunction) loopBodyInstructionPreservesRead(pc int, receiverReg, destReg uint8, destBank operandRole) bool {
	inst := cf.body[pc]
	if invalidatesCachedFieldReads(cf, inst) && !structFieldWriteWithKnownReceiverDistinct(cf, pc, inst, receiverReg) {
		return false
	}
	if !instructionShapeAllowsCseScan(inst) {
		return false
	}
	if instructionWritesRegisterInBank(inst, roleRegGeneral, receiverReg) {
		return false
	}
	if destBank != roleNone && instructionWritesRegisterInBank(inst, destBank, destReg) {
		return false
	}
	return true
}

// applyStructFieldHoist inserts a copy of body[readPC] at the loop header PC, replaces
// the (now-shifted) original read with opNop, adjusts all jump offsets that cross the
// insertion point, and updates the function's source map so post-insertion PCs continue
// to map to their original source positions.
//
// The hoisted instruction executes once on first-iteration entry (linear fall-through
// from PC headerPC - 1) and is skipped on subsequent iterations because every existing
// jump that previously targeted headerPC now targets headerPC + 1.
//
// Takes headerPC (int) which is the loop header PC (insertion site for the hoist).
// Takes readPC (int) which is the original PC of the loop-invariant read. readPC must
// satisfy readPC >= headerPC.
func (cf *CompiledFunction) applyStructFieldHoist(headerPC, readPC int) {
	width := hoistedReadWordCount(cf.body, readPC)
	hoisted := make([]instruction, width)
	for offset := range width {
		hoisted[offset] = cf.body[readPC+offset]
	}
	cf.insertInstructionsAt(headerPC, hoisted)
	for offset := range width {
		cf.recordPeepholeRewrite(headerPC+offset, peepholeRewriteLicmHoist, readPC+width+offset)
	}
	shiftedOriginal := readPC + width
	for offset := range width {
		cf.body[shiftedOriginal+offset] = makeInstruction(opNop, 0, 0, 0)
		cf.recordPeepholeRewrite(shiftedOriginal+offset, peepholeRewriteLicmOpNop, headerPC+offset)
	}
}

// insertInstructionsAt splices insts into cf.body starting at insertPC, shifts every
// later PC by len(insts), and rewrites every existing jump's offset so post-insert
// targets equal pre-insert targets.
//
// Source-map positions and peephole provenance entries for instructions at or after
// insertPC shift by len(insts); the inserted instructions inherit zero source positions
// (best-effort attribution for debugger frames).
//
// cf.aliasInfo is invalidated unconditionally: it is a per-absolute-PC environment
// computed against the pre-insertion body, so every PC at or after insertPC would now
// name a different instruction. A stale alias environment could let a downstream CSE/GVN
// query return a wrong "definitely-not-aliasing" verdict and elide a field read across an
// aliasing write. Dropping it forces mayAlias back to its conservative "true" fallback,
// which only costs optimisation opportunities, never correctness. Any pass that needs
// alias information after an insertion must re-run runPointerAliasAnalysis.
//
// Takes insertPC (int) which is the index where the new block is placed.
// Takes insts ([]instruction) which are the instructions to insert in order.
func (cf *CompiledFunction) insertInstructionsAt(insertPC int, insts []instruction) {
	width := len(insts)
	if width == 0 {
		return
	}
	cf.body = append(cf.body, make([]instruction, width)...)
	copy(cf.body[insertPC+width:], cf.body[insertPC:])
	copy(cf.body[insertPC:], insts)
	rewriteJumpOffsetsAfterInsertWidth(cf.body, insertPC, width)
	for range width {
		cf.shiftSourceMapAfterInsert(insertPC)
		cf.shiftPeepholeProvenanceAfterInsert(insertPC)
	}
	cf.aliasInfo = nil
}

// shiftSourceMapAfterInsert inserts a zero source position at insertPC and pushes every
// later entry down by one, so post-insert PCs continue to map to their original source
// positions.
//
// The hoisted instruction inherits a zero source position; the debugger will attribute it
// to the next-following source line via the SourcePosition lookup's fallback behaviour.
//
// Takes insertPC which is the index where the zero source position is spliced in.
func (cf *CompiledFunction) shiftSourceMapAfterInsert(insertPC int) {
	if cf.debugSourceMap == nil {
		return
	}
	positions := cf.debugSourceMap.positions
	if insertPC > len(positions) {
		return
	}
	positions = append(positions, sourcePosition{})
	copy(positions[insertPC+1:], positions[insertPC:])
	positions[insertPC] = sourcePosition{}
	cf.debugSourceMap.positions = positions
}

// loopHeaderReachableByFallThrough reports whether linear fall-through enters the loop
// header.
//
// When a loop is reachable only via in-loop jumps (e.g. synthetic back-edges emitted by
// the inline splicer or by a deferred-recover plumbing pattern), the pre-header is dead
// on the entry path and a hoist placed there would never execute on entry while still
// leaving its destination register clobbered for any other reader that observes it before
// the loop body re-enters via the back-edge.
//
// Function entry (loop.header == 0) is always reachable. Otherwise the preceding
// instruction must not be an unconditional terminator (unconditional jump, tail call,
// return-through-tier-2).
//
// Takes body which is the compiled function's instruction stream.
// Takes loop which is the natural loop being considered for hoisting.
//
// Returns true when the loop header is reached via fall-through, false otherwise.
func loopHeaderReachableByFallThrough(body []instruction, loop loopRange) bool {
	if loop.header <= 0 {
		return true
	}
	return isLinearFallThroughFrom(body[loop.header-1])
}

// buildAllJumpTargets returns every jump or branch target PC in body.
//
// Recognises the full set: tier-0 conditional and unconditional jumps, the fused
// compare-and-jump opcodes, the test-nil-and-jump pair, and tier-1 subOpJump. CSE and
// LICM rely on a complete target set to avoid rewriting an instruction reachable from
// multiple predecessors with distinct register states.
//
// Uses jumpOffsetOf (compiler_inliner.go) to decode signed offsets from any recognised
// jump shape, ensuring the helper stays in sync with new fused-jump opcodes added to the
// enum table.
//
// Takes body which is the instruction stream to scan for jump targets.
//
// Returns a map whose keys are every PC reached by any jump in body.
func buildAllJumpTargets(body []instruction) map[int]bool {
	targets := make(map[int]bool, len(body)/4)
	for pc, inst := range body {
		offset, isJump := jumpOffsetOf(inst)
		if !isJump {
			continue
		}
		targets[pc+1+offset] = true
	}
	return targets
}

// identifyLoops scans the body for natural loops.
//
// A natural loop is induced by a back-edge: a jump instruction whose target PC is less
// than or equal to its own PC. Each back-edge defines a loop covering the closed interval
// [target, source].
//
// Takes body which is the instruction stream to scan.
//
// Returns the loops in a deterministic order that places tight inner loops ahead of wider
// outer ones when the hoist cap is reached.
func identifyLoops(body []instruction) []loopRange {
	loops := make([]loopRange, 0, initialLoopCapacity)
	for pc, inst := range body {
		offset, isJump := jumpOffsetOf(inst)
		if !isJump || offset >= 0 {
			continue
		}
		targetPC := pc + 1 + offset
		if targetPC < 0 || targetPC > pc {
			continue
		}
		loops = append(loops, loopRange{header: targetPC, latch: pc})
	}
	sortLoopRangesAscendingSpan(loops)
	return loops
}

// sortLoopRangesAscendingSpan reorders loops with the smallest body span first. Innermost
// loops (small span) get hoist preference over surrounding loops (large span) when the
// per-function cap forces a choice.
//
// Takes loops which is the slice of loop ranges to sort in place.
func sortLoopRangesAscendingSpan(loops []loopRange) {
	slices.SortFunc(loops, func(a, b loopRange) int {
		return cmp.Compare(a.latch-a.header, b.latch-b.header)
	})
}

// computeBackEdgeDominators returns dominators of the loop back-edge: the set of PCs in
// [loop.header, loop.latch] that every iteration completing the loop passes through.
//
// The algorithm is the textbook iterative dominator computation (Aho/Sethi/Ullman) using
// dom[header] = {header} and dom[P] = {P} union the intersection of dom[pred] over P's
// predecessors, iterated until no set changes. Within the loop body a PC's predecessors
// are the linear fall-through and any in-loop branch targeting it; function returns leave
// their successors without an in-loop predecessor. The header dominates itself trivially.
// Complexity is O(N^2) in body length, bounded in practice by the per-loop hoist cap and
// typical loop sizes of 50 to 100 instructions.
//
// If the dataflow has not reached a fixpoint within maxLoopDominatorIterations the
// dominator table is non-exact, so the helper returns nil; findHoistableReadInLoop reads
// nil as "nothing dominates" and refuses every hoist for this loop. This conservative
// bailout only costs optimisation; it never executes a speculative read.
//
// Takes body ([]instruction) which is the compiled function's instruction stream.
// Takes loop (loopRange) which is the natural loop whose back-edge dominators are being
// computed.
//
// Returns []bool which is a body-sized slice with true for every PC that dominates the
// loop's back-edge (entries outside the loop range are false); nil when the loop is
// degenerate or the dataflow did not converge.
func computeBackEdgeDominators(body []instruction, loop loopRange) []bool {
	loopLen := loop.latch - loop.header + 1
	if loopLen <= 0 {
		return nil
	}
	predecessors := buildLoopPredecessors(body, loop)
	dom := initialiseLoopDominators(loopLen)
	converged := false
	for range maxLoopDominatorIterations {
		if !relaxLoopDominatorsOnce(dom, predecessors, loopLen) {
			converged = true
			break
		}
	}
	if !converged {
		return nil
	}
	return collectLatchDominators(dom, loop, len(body))
}

// initialiseLoopDominators returns the seed dominator table for a loop: every PC starts
// dominated by every other (universe), then the header is fixed to dominating only
// itself.
//
// Takes loopLen which is the number of instructions covered by the loop.
//
// Returns a loopLen by loopLen boolean matrix seeded for the iterative dominator
// dataflow.
func initialiseLoopDominators(loopLen int) [][]bool {
	dom := make([][]bool, loopLen)
	for index := range dom {
		dom[index] = make([]bool, loopLen)
		for j := range dom[index] {
			dom[index][j] = true
		}
	}
	clear(dom[0])
	dom[0][0] = true
	return dom
}

// relaxLoopDominatorsOnce performs one sweep of the in-loop dataflow relaxation.
//
// Takes dom which is the current dominator table, updated in place.
// Takes predecessors which is the per-offset predecessor lists within the loop.
// Takes loopLen which is the number of instructions covered by the loop.
//
// Returns true when any offset's bitset changed during the sweep.
func relaxLoopDominatorsOnce(dom [][]bool, predecessors [][]int, loopLen int) bool {
	changed := false
	for offset := 1; offset < loopLen; offset++ {
		preds := predecessors[offset]
		if len(preds) == 0 {
			continue
		}
		intersected := intersectLoopDominators(dom, preds, loopLen)
		intersected[offset] = true
		if applyLoopDominatorUpdate(dom[offset], intersected) {
			changed = true
		}
	}
	return changed
}

// intersectLoopDominators returns the bitwise AND of every predecessor's dominator bitset
// within the loop.
//
// Takes dom which is the dominator table indexed by loop-relative offset.
// Takes preds which is the predecessor offsets to intersect.
// Takes loopLen which is the number of instructions covered by the loop.
//
// Returns a fresh bitset whose entry at j is true only when every predecessor's dominator
// set has j set.
func intersectLoopDominators(dom [][]bool, preds []int, loopLen int) []bool {
	intersected := make([]bool, loopLen)
	for j := range intersected {
		intersected[j] = true
	}
	for _, pred := range preds {
		for j := range loopLen {
			if !dom[pred][j] {
				intersected[j] = false
			}
		}
	}
	return intersected
}

// applyLoopDominatorUpdate copies updated bits into destination when they differ.
//
// Takes destination which is the destination bitset, written in place.
// Takes updated which is the candidate bitset to merge into destination.
//
// Returns true when at least one position changed.
func applyLoopDominatorUpdate(destination, updated []bool) bool {
	changed := false
	for j := range destination {
		if updated[j] != destination[j] {
			destination[j] = updated[j]
			changed = true
		}
	}
	return changed
}

// collectLatchDominators projects the latch offset's dominator bitset back onto a
// body-sized result slice.
//
// Takes dom which is the dominator table indexed by loop-relative offset.
// Takes loop which is the natural loop whose latch dominators are being collected.
// Takes bodyLen which is the length of the function body for sizing the result.
//
// Returns a body-sized boolean slice with true for every PC in the loop that dominates
// the latch.
func collectLatchDominators(dom [][]bool, loop loopRange, bodyLen int) []bool {
	latchOffset := loop.latch - loop.header
	result := make([]bool, bodyLen)
	for offset, dominatesLatch := range dom[latchOffset] {
		if dominatesLatch {
			result[loop.header+offset] = true
		}
	}
	return result
}

// buildLoopPredecessors computes the predecessor list for every PC inside the loop body,
// indexed by offset relative to loop.header.
//
// Predecessors of a PC are the linear fall-through (the previous PC) when the previous
// instruction is not an unconditional terminator (opJump-style or return), plus any
// in-loop branch whose target lands on this PC.
//
// PCs that have no in-loop predecessor are treated as unreachable from the loop header
// (the dominator iteration leaves their set fixed at "all dominators", which the latch
// intersection ignores via the predecessor check).
//
// Takes body which is the compiled function's instruction stream.
// Takes loop which is the natural loop whose predecessors are being computed.
//
// Returns a slice indexed by loop-relative offset, where each entry lists the predecessor
// offsets.
func buildLoopPredecessors(body []instruction, loop loopRange) [][]int {
	loopLen := loop.latch - loop.header + 1
	predecessors := make([][]int, loopLen)
	for offset := 1; offset < loopLen; offset++ {
		pc := loop.header + offset
		if isLinearFallThroughFrom(body[pc-1]) {
			predecessors[offset] = append(predecessors[offset], offset-1)
		}
	}
	for offset := range loopLen {
		pc := loop.header + offset
		jumpOffset, isJump := jumpOffsetOf(body[pc])
		if !isJump {
			continue
		}
		targetPC := pc + 1 + jumpOffset
		if targetPC < loop.header || targetPC > loop.latch {
			continue
		}
		targetOffset := targetPC - loop.header
		predecessors[targetOffset] = append(predecessors[targetOffset], offset)
	}
	return predecessors
}

// isLinearFallThroughFrom reports whether inst falls through to the next PC.
//
// An instruction falls through when its execution does not unconditionally redirect to a
// different PC. Unconditional jumps and function-terminators (return, tail-call) do not
// fall through.
//
// Conditional jumps DO fall through on the not-taken side, so they count as
// linear-fall-through predecessors for the dominator analysis.
//
// Takes inst which is the instruction whose control-flow behaviour is being classified.
//
// Returns true when inst falls through to the next sequential PC.
func isLinearFallThroughFrom(inst instruction) bool {
	if inst.op == opTailCall {
		return false
	}
	if inst.op != opDrillTier1 {
		return true
	}
	return !tier1RedirectsControlFlow(inst)
}

// tier1RedirectsControlFlow reports whether a tier-1 instruction transfers control rather
// than falling through: direct jumps, returns, and the return-void tier-3 sub-op.
//
// Takes inst which is the tier-1 instruction whose control-flow behaviour is being
// classified.
//
// Returns true when inst redirects control rather than falling through.
func tier1RedirectsControlFlow(inst instruction) bool {
	sub := subOpcode(inst.a)
	if sub == subOpJump {
		return true
	}
	if sub != subOpDrillTier2 {
		return false
	}
	tier2 := subOpcodeTier2(inst.b)
	if tier2 == subOpTier2Return {
		return true
	}
	return tier2 == subOpTier2DrillTier3 && subOpcodeTier3(inst.c) == subOpTier3ReturnVoid
}

// isHoistableReadAt reports whether body[pc] is the start of a struct-field read eligible
// for LICM.
//
// Covers tier-0 single-word reads (general bank, scalar banks, generic opGetField,
// opGetFieldInt), which are recognised by isTier0StructFieldRead from the CSE pass and
// are hoisted as one instruction; and tier-1 reads (opDrillTier1 + opExt), recognised by
// isTier1StructFieldReadAt, which span two instruction words and the hoist must shift
// both.
//
// Takes body which is the compiled function's instruction stream.
// Takes pc which is the candidate PC to classify.
//
// Returns true when body[pc] begins a hoistable read.
func isHoistableReadAt(body []instruction, pc int) bool {
	if pc >= len(body) {
		return false
	}
	if isTier0StructFieldRead(body[pc].op) {
		return true
	}
	if isTier1StructFieldReadAt(body, pc) {
		return true
	}
	return false
}

// hoistedReadWordCount returns the number of instruction words a hoistable read at pc
// occupies in the body. Tier-0 reads are one word; tier-1 reads are two (the opDrillTier1
// umbrella plus the opExt layout extension).
//
// Behaviour is undefined when isHoistableReadAt(body, pc) is false.
//
// Takes body which is the compiled function's instruction stream.
// Takes pc which is the PC of the hoistable read whose width is being measured.
//
// Returns the instruction-word width of the read.
func hoistedReadWordCount(body []instruction, pc int) int {
	if isTier1StructFieldReadAt(body, pc) {
		return 2
	}
	return 1
}

// hoistedReadDestBank returns the destination bank role of the read starting at pc. Used
// by the invariance analysis to detect writes that would clobber the cached value once
// the read is lifted to the pre-header.
//
// Takes body which is the compiled function's instruction stream.
// Takes pc which is the PC of the hoistable read whose destination bank is being queried.
//
// Returns the operandRole identifying the destination bank.
func hoistedReadDestBank(body []instruction, pc int) operandRole {
	if isTier1StructFieldReadAt(body, pc) {
		return tier1FieldReadDestRole(subOpcode(body[pc].a))
	}
	return tier0FieldReadDestRole(body[pc].op)
}

// hoistedReadDestRegister returns the destination register index of the read starting at
// pc.
//
// Takes body which is the compiled function's instruction stream.
// Takes pc which is the PC of the hoistable read whose destination register is being
// queried.
//
// Returns the destination register index.
func hoistedReadDestRegister(body []instruction, pc int) uint8 {
	if isTier1StructFieldReadAt(body, pc) {
		return body[pc].b
	}
	return body[pc].a
}

// hoistedReadReceiverRegister returns the general-bank register holding the struct
// receiver for the read starting at pc.
//
// Takes body which is the compiled function's instruction stream.
// Takes pc which is the PC of the hoistable read whose receiver register is being
// queried.
//
// Returns the receiver register index.
func hoistedReadReceiverRegister(body []instruction, pc int) uint8 {
	if isTier1StructFieldReadAt(body, pc) {
		return body[pc].c
	}
	return body[pc].b
}

// rewriteJumpOffsetsAfterInsertWidth walks every jump instruction in body and rewrites
// its offset so the post-insert target equals the pre-insert target after a block of
// `width` instructions was spliced in at insertPC.
//
// Treats the body as already shifted: every jump originally at sourcePC < insertPC is
// still at sourcePC; every jump originally at sourcePC >= insertPC is now at sourcePC +
// width. Same rule for targetPC. The relative offset changes by +width only when the
// original source was strictly less than insertPC and the original target was at-or-after
// insertPC; by -width only when the original source was at-or-after insertPC and the
// original target was strictly less. All other configurations leave the offset unchanged.
//
// Takes body which is the already-shifted instruction stream to patch.
// Takes insertPC which is the index where the new block was placed.
// Takes width which is the number of instructions inserted at insertPC.
func rewriteJumpOffsetsAfterInsertWidth(body []instruction, insertPC, width int) {
	for newPC, inst := range body {
		offset, isJump := jumpOffsetOf(inst)
		if !isJump {
			continue
		}
		newTarget := newPC + 1 + offset
		originalSource := newPC
		if newPC >= insertPC+width {
			originalSource = newPC - width
		}
		originalTarget := newTarget
		if newTarget >= insertPC+width {
			originalTarget = newTarget - width
		}
		delta := 0
		switch {
		case originalSource < insertPC && originalTarget >= insertPC:
			delta = width
		case originalSource >= insertPC && originalTarget < insertPC:
			delta = -width
		}
		if delta == 0 {
			continue
		}
		body[newPC] = encodeJumpWithOffset(inst, offset+delta)
	}
}

// encodeJumpWithOffset returns inst with its B/C bytes overwritten to carry the signed
// 16-bit offset. The opcode and operand A are preserved.
//
// Takes inst which is the source jump instruction to rewrite.
// Takes offset which is the new signed 16-bit jump offset.
//
// Returns the rewritten instruction carrying the new offset.
func encodeJumpWithOffset(inst instruction, offset int) instruction {
	low, high := splitWide(safeconv.Int16ToUint16(safeconv.MustIntToInt16(offset)))
	return makeInstruction(inst.op, inst.a, low, high)
}
