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

	"piko.sh/piko/wdk/safeconv"
)

const (
	// defaultInlineBudget caps the callee hairyness score at non-loop sites.
	//
	// See calleeHairyness for the per-opcode weights. Each simple opcode contributes 1;
	// complex opcodes (maps, type-asserts, allocations) contribute more, mirroring the Go
	// compiler's inline/inl.go budget model where the default cap of 80 weights AST nodes by
	// estimated cost. Piko bytecode is denser than Go AST (one bytecode instruction often
	// represents multiple AST nodes), so the absolute number is smaller; 80-equivalent in
	// piko works out to roughly 40.
	defaultInlineBudget = 40

	// loopInlineBudget caps the callee hairyness score at loop sites.
	//
	// Loop sites' amortised win scales with iteration count, so larger callees are accepted
	// there. Matches Go's behaviour of being more aggressive about inlining inside loops.
	loopInlineBudget = 80

	// selfUnrollBudget caps the hairyness score of a callee considered for 1-level
	// self-recursive unrolling.
	selfUnrollBudget = 20

	// maxCallerBodyAfterInline bounds a caller's body length post-inlining.
	//
	// Beyond ~2 KB the verifier worklist costs grow superlinearly and jump-offset int16
	// ranges become hairy. Inlining is refused at sites that would push the caller past this
	// size.
	maxCallerBodyAfterInline = 2048

	// maxInlinesPerCaller caps how many distinct call sites within one caller may be
	// inlined. Defence against a runaway caller pulling in hundreds of small callees and
	// bloating the function.
	maxInlinesPerCaller = 8

	// callGraphSeedCapacity is the initial capacity hint used when allocating visit-state
	// tables for the call-graph walk. Sized to fit the common case (a couple-of-dozen
	// reachable nested functions) without growth.
	callGraphSeedCapacity = 32

	// registerBankWatermark is the maximum number of register slots per bank the runtime can
	// address with a single uint8 operand. A splice that would push the caller past this is
	// refused as inlineRefusalCapWatermark.
	registerBankWatermark = 255

	// instructionByteShift is the bit shift applied when packing the upper byte of a uint16
	// operand into the trailing C byte of an instruction.
	instructionByteShift = 8
)

// inlineRefusal records why a callee was rejected.
//
// Cached on the callee so the body is not re-scanned per call site. The zero value is
// inlineRefusalUnknown so a freshly-allocated CompiledFunction starts in the unprobed
// state; the first call to calleeInlineRefusal scans the body and writes the real result.
type inlineRefusal uint8

const (
	// inlineRefusalUnknown is the zero value meaning the callee has not been probed.
	inlineRefusalUnknown inlineRefusal = iota

	// inlineEligible marks a callee that passed every refusal check.
	inlineEligible

	// inlineRefusalUpvalues refuses callees with closure-captured variables.
	inlineRefusalUpvalues

	// inlineRefusalDefer refuses callees whose body contains opDefer.
	inlineRefusalDefer

	// inlineRefusalGo refuses callees whose body contains opGo.
	inlineRefusalGo

	// inlineRefusalRecursion refuses callees that sit inside an SCC of the call graph.
	inlineRefusalRecursion

	// inlineRefusalMethodCall refuses callees whose body contains opCallMethod.
	inlineRefusalMethodCall

	// inlineRefusalNativeCall refuses callees whose body contains opCallNative.
	inlineRefusalNativeCall

	// inlineRefusalTailCall refuses callees whose body contains opTailCall.
	inlineRefusalTailCall

	// inlineRefusalGenericPlaceholder refuses uninstantiated generic placeholders.
	inlineRefusalGenericPlaceholder

	// inlineRefusalClosureOps refuses callees using makeClosure / get / set / sync upvalue
	// ops.
	inlineRefusalClosureOps

	// inlineRefusalChannelOps refuses callees using opSelect, opChannelSend, or recv ops.
	inlineRefusalChannelOps

	// inlineRefusalVariadic refuses callees that cannot be packed at the call site.
	inlineRefusalVariadic

	// inlineRefusalNoBody refuses callees with an empty body (forward declarations).
	inlineRefusalNoBody

	// inlineRefusalSiteIndirect refuses sites that are native, closure, or method.
	inlineRefusalSiteIndirect

	// inlineRefusalOversize refuses callees whose hairyness score exceeds the budget.
	inlineRefusalOversize

	// inlineRefusalCapWatermark refuses splices that would push numRegisters past 256.
	inlineRefusalCapWatermark

	// inlineRefusalCallerCap refuses sites once the caller has hit maxInlinesPerCaller.
	inlineRefusalCallerCap

	// inlineRefusalAlreadyUnrolled refuses self-recursive sites that have already been
	// unrolled once (preventing infinite expansion).
	inlineRefusalAlreadyUnrolled

	// inlineRefusalSelfHairy refuses self-recursive sites whose callee hairyness exceeds
	// selfUnrollBudget.
	inlineRefusalSelfHairy

	// inlineRefusalSelfInLoop refuses self-recursive unrolling at call sites inside a loop
	// body, avoiding inflated hot-loop bytecode.
	inlineRefusalSelfInLoop

	// inlineRefusalUnrollDisabled refuses self-recursive unrolling when the build-tag flag
	// turns it off.
	inlineRefusalUnrollDisabled
)

// inlineContext bundles per-splice mutable state.
//
// One context per (caller, callee, callSiteIndex) splice attempt.
type inlineContext struct {
	// caller is the function being mutated; the splice appends to its body.
	caller *CompiledFunction

	// callee is the function whose body is being spliced into caller.
	callee *CompiledFunction

	// site is the call site descriptor in caller.callSites being inlined.
	site *callSite

	// paramPreCopies records the (kind, dst, src) tuples for pre-copy MOVEs emitted before
	// the inlined body.
	//
	// Each callee parameter slot is remapped to a fresh caller-side slot rather than aliased
	// to the caller's argument source register, because the callee's body may write to the
	// parameter slot (e.g. the trailing "MOVE return-slot, result" idiom every non-void
	// function ends with). Pre-copy preserves Go's by-value parameter semantics across the
	// splice; without it the body's writes silently mutate the caller's argument variable.
	// Cleared by buildRegisterRemap on each splice attempt.
	paramPreCopies []paramPreCopy

	// opCallPC is the PC of the opCall instruction in caller.body being inlined.
	opCallPC int

	// remap is the per-bank callee-slot to caller-slot mapping.
	//
	// remap[bank]calleeSlot = callerSlot, encoded as int16 with -1 meaning "unset".
	// Fixed-size array: no heap allocation, no hash-map dispatch per lookup. Reset via
	// resetRegisterRemap at the start of buildRegisterRemap.
	remap [NumRegisterKinds][256]int16

	// siteIndex is the index of site within caller.callSites.
	siteIndex uint16
}

// paramPreCopy describes one MOVE emitted before the inlined callee body.
//
// Seeds a fresh caller-side parameter slot from the caller's argument source register.
// Same-bank pre-copies set sourceKind == kind and emit one tier-1 move sub-op. Cross-bank
// pre-copies (sourceKind != kind) emit an adopt or box sub-op via
// crossBankAdoptOrBoxSubOp; supported only for general<->typed-slice pairs.
type paramPreCopy struct {
	// sourceKind names the bank holding the caller-side argument register. For same-bank
	// pre-copies this equals kind.
	sourceKind registerKind

	// kind names the register bank the move is emitted INTO (the fresh parameter slot's
	// bank). For same-bank pre-copies this equals sourceKind.
	kind registerKind

	// destination is the fresh caller-side slot allocated for the callee's parameter.
	destination uint8

	// source is the caller's argument source register feeding the pre-copy.
	source uint8
}

// resetRegisterRemap reinitialises ctx.remap to all -1 (unset).
func (ctx *inlineContext) resetRegisterRemap() {
	for k := range ctx.remap {
		for i := range ctx.remap[k] {
			ctx.remap[k][i] = -1
		}
	}
}

// lookupRegister returns the caller-side slot for a callee register.
//
// Takes bank (registerKind) which names the bank holding the mapping.
// Takes calleeSlot (uint8) which is the callee-side register index.
//
// Returns the caller-side slot.
// Returns false when no mapping exists for the (bank, calleeSlot) pair.
func (ctx *inlineContext) lookupRegister(bank registerKind, calleeSlot uint8) (uint8, bool) {
	v := ctx.remap[bank][calleeSlot]
	if v < 0 {
		return 0, false
	}
	return safeconv.Int16ToByte(v), true
}

// setRegister records the calleeSlot -> callerSlot mapping in bank.
//
// Takes bank (registerKind) which names the bank holding the mapping.
// Takes calleeSlot (uint8) which is the callee-side register index.
// Takes callerSlot (uint8) which is the freshly allocated caller-side slot.
func (ctx *inlineContext) setRegister(bank registerKind, calleeSlot, callerSlot uint8) {
	ctx.remap[bank][calleeSlot] = int16(callerSlot)
}

// reoptimiseAfterInline re-runs peephole fusion after a splice.
//
// Distinct from the full CompiledFunction.optimise() because it must not recurse into
// nested functions: they were already optimised before the inliner pass started, and
// re-optimising would only matter if they got inlined into, which is handled by the
// caller's bottom-up loop iterating each function separately. Invalidates and recomputes
// precomputedAllocCounts because the splice grew numRegisters[k] by adding fresh
// caller-side slots for writeable parameters; the hot path otherwise allocates too few
// register slots from the arena and panics on the first write to a fresh slot.
//
// Takes ctx (context.Context) which carries cancellation.
//
// Returns error when context cancellation fires.
func (cf *CompiledFunction) reoptimiseAfterInline(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("reoptimiseAfterInline cancelled: %w", err)
	}
	body := cf.body
	n := len(body)
	jumpTargets := cf.buildJumpTargets(body)
	for i := range n {
		if i&optimisationLoopCheckMask == 0 {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("reoptimiseAfterInline cancelled: %w", err)
			}
		}
		if cf.fuseCopyStructFieldGeneralT0(body, i, n, jumpTargets) {
			continue
		}
		if cf.fuseThreeInstrPatterns(body, i, n, jumpTargets) ||
			cf.fuseArithConst(body, i, n, jumpTargets) ||
			cf.fuseAddIntJump(body, i, n, jumpTargets) ||
			cf.fuseConcatRune(body, i, n, jumpTargets) ||
			cf.fuseAppendMove(body, i, n, jumpTargets) ||
			cf.fuseStringIndexToInt(body, i, n, jumpTargets) {
			continue
		}
		cf.optimiseLoadIntConst(body, i)
		cf.optimiseLoadUintConst(body, i)
	}
	cf.precomputedAllocCountsValid = false
	cf.ensurePrecomputedAllocCounts()
	return nil
}

// runBytecodeInliner is the top-level inliner pass entry.
//
// Invoked from runPostCompilationChecks AFTER recursion detection and BEFORE the bytecode
// verifier so the verifier checks the spliced shape. Walks the call graph bottom-up: for
// each function, considers every call site, and if both callee and site pass canInline,
// splices the callee's body into the caller. The bottom-up order means an already-inlined
// callee will not be considered as a splice target for a higher-up caller (it has been
// folded into its own callers); this is desirable - multi-level inlining happens
// naturally without a separate iteration.
//
// Takes root (*CompiledFunction) which is the program's top-level compiled function whose
// nested closures are walked.
//
// Returns nil even when inlining is disabled or no callee is eligible; errors only
// surface on internal-consistency failures (e.g., bytecode that fails our own structural
// invariants).
func runBytecodeInliner(ctx context.Context, root *CompiledFunction) error {
	if root == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("runBytecodeInliner cancelled: %w", err)
	}
	allFuncs := collectReachableFunctions(root)
	if !anyInlineableCallSite(allFuncs) {
		return nil
	}
	if root.functions != nil {
		adjacency := buildCallAdjacency(root.functions)
		inSCC := findCallGraphSCCs(adjacency)
		for i, cf := range root.functions {
			if i < len(inSCC) {
				cf.inRecursionCycle = inSCC[i]
			}
		}
	}
	order := bottomUpOrder(allFuncs)
	for _, caller := range order {
		if inlineCallsIn(caller) > 0 {
			if err := caller.reoptimiseAfterInline(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// anyInlineableCallSite reports whether any function holds an inlineable site.
//
// Used as the runBytecodeInliner precheck to skip the full pass when no inlining can
// possibly happen. Only the cheap callee-property eligibility check is consulted;
// per-site checks (size headroom, register watermark) run later inside canInline.
//
// Takes allFuncs ([]*CompiledFunction) which holds every reachable function in the
// program.
//
// Returns true when at least one site's cachedCallee passes calleeInlineRefusal, false
// otherwise.
func anyInlineableCallSite(allFuncs []*CompiledFunction) bool {
	for _, cf := range allFuncs {
		if cf == nil {
			continue
		}
		for i := range cf.callSites {
			site := &cf.callSites[i]
			if site.isNative || site.isClosure || site.isMethod {
				continue
			}
			callee := site.cachedCallee
			if callee == nil {
				continue
			}
			if calleeInlineRefusal(callee) == inlineEligible {
				return true
			}
		}
	}
	return false
}

// inlineCallsIn drives the splice for every eligible site in caller.
//
// Iterates call sites by index up to maxInlinesPerCaller; when a splice succeeds, the
// caller's body changes but call-site indices remain stable because the splice appends to
// the body and never reorders callSites. Refused sites are skipped silently.
// caller.cachedInlineRefusal is intentionally left untouched after a successful splice:
// once-inlined bodies grow significantly, and re-considering them as callees for outer
// functions tends to over-inline and bloat. The per-function eligibility scan therefore
// answers whether a callee was originally inlineable rather than whether it remains
// inlineable after its own body has been mutated.
//
// Takes caller (*CompiledFunction) whose body is mutated by each successful splice.
//
// Returns the count of call sites successfully inlined.
func inlineCallsIn(caller *CompiledFunction) int {
	if caller == nil || len(caller.callSites) == 0 {
		return 0
	}
	inLoopMask := computeInLoopMask(caller)
	sitePCs := buildSitePCTable(caller)
	spliced := 0
	for siteIndex := range caller.callSites {
		if spliced >= maxInlinesPerCaller {
			break
		}
		if !trySpliceSite(caller, siteIndex, sitePCs, inLoopMask) {
			continue
		}
		spliced++
		extendInLoopMaskForAppendedBody(caller, &inLoopMask)
		sitePCs = buildSitePCTable(caller)
	}
	return spliced
}

// trySpliceSite evaluates the inliner gates for one call site and, on success, performs
// the splice.
//
// Takes caller (*CompiledFunction) which is the function being optimised.
// Takes siteIndex (int) which is the index of the call site within caller.callSites.
// Takes sitePCs ([]int) which lists the program counters of each call site's opCall
// within the caller body.
// Takes inLoopMask ([]bool) which is the parallel mask marking sites that reside inside a
// loop.
//
// Returns true when the splice landed and false otherwise.
func trySpliceSite(caller *CompiledFunction, siteIndex int, sitePCs []int, inLoopMask []bool) bool {
	site := &caller.callSites[siteIndex]
	opCallPC := -1
	if siteIndex < len(sitePCs) {
		opCallPC = sitePCs[siteIndex]
	}
	if opCallPC < 0 {
		return false
	}
	inLoop := opCallPC < len(inLoopMask) && inLoopMask[opCallPC]
	if reason := canInline(caller, site, opCallPC, inLoop); reason != inlineEligible {
		return false
	}
	result := trySpliceCallAt(caller, safeconv.IntToUint16(siteIndex), opCallPC)
	if !result.spliced {
		return false
	}
	if site.cachedCallee == caller {
		site.recursionUnrolled = true
	}
	return true
}

// buildSitePCTable maps each call-site index to its opCall PC.
//
// One body walk produces the full table; only opCall instructions are matched.
// opCallMethod, opCallNative, and opCallIIFE sites are returned as -1 because the inliner
// refuses them in canInline. This mirrors findOpCallPC's behaviour.
//
// Takes caller (*CompiledFunction) whose body is scanned.
//
// Returns a slice indexed by call-site index with the PC of that site's opCall, or -1
// when the site has no matching opCall (already inlined, or non-opCall variant).
func buildSitePCTable(caller *CompiledFunction) []int {
	n := len(caller.callSites)
	if n == 0 {
		return nil
	}
	pcs := make([]int, n)
	for i := range pcs {
		pcs[i] = -1
	}
	for pc := range caller.body {
		instr := caller.body[pc]
		if instr.op != opCall {
			continue
		}
		index := int(uint16(instr.b) | uint16(instr.c)<<8)
		if index < n {
			pcs[index] = pc
		}
	}
	return pcs
}

// extendInLoopMaskForAppendedBody grows *mask to match caller body length.
//
// New PCs default to false (no back-edge target in or into the appended region) unless
// the appended bytes contain a backward jump, in which case the full mask is recomputed
// for the affected window.
//
// Takes caller (*CompiledFunction) whose body length sets the target size.
// Takes mask (*[]bool) which is grown (or reallocated) in place.
func extendInLoopMaskForAppendedBody(caller *CompiledFunction, mask *[]bool) {
	newLen := len(caller.body)
	oldLen := len(*mask)
	if newLen <= oldLen {
		return
	}
	hasBackwardJump := false
	for pc := oldLen; pc < newLen; pc++ {
		instr := caller.body[pc]
		if off, ok := jumpOffsetOf(instr); ok && off < 0 {
			hasBackwardJump = true
			break
		}
	}
	if hasBackwardJump {
		*mask = computeInLoopMask(caller)
		return
	}
	if cap(*mask) >= newLen {
		*mask = (*mask)[:newLen]
		for i := oldLen; i < newLen; i++ {
			(*mask)[i] = false
		}
		return
	}
	grown := make([]bool, newLen)
	copy(grown, *mask)
	*mask = grown
}

// collectReachableFunctions returns every function reachable from root.
//
// Walks the closure-nesting tree. The output order matches walk order; bottomUpOrder
// reorders into topological form.
//
// Takes root (*CompiledFunction) whose nested functions are traversed.
//
// Returns the reachable set in walk order.
func collectReachableFunctions(root *CompiledFunction) []*CompiledFunction {
	visited := make(map[*CompiledFunction]struct{}, callGraphSeedCapacity)
	out := make([]*CompiledFunction, 0, callGraphSeedCapacity)
	var walk func(*CompiledFunction)
	walk = func(cf *CompiledFunction) {
		if cf == nil {
			return
		}
		if _, ok := visited[cf]; ok {
			return
		}
		visited[cf] = struct{}{}
		out = append(out, cf)
		for _, child := range cf.functions {
			walk(child)
		}
	}
	walk(root)
	return out
}

// bottomUpOrder reorders allFuncs so call-graph leaves come first.
//
// Functions with no outgoing calls (or whose calls only target already-visited callees)
// are added to the result before their callers. Cycles, when present, are broken
// arbitrarily; the inliner refuses sites in cycles via canInline's recursion check.
//
// Takes allFuncs ([]*CompiledFunction) which is the unsorted reachable set.
//
// Returns a fresh slice with allFuncs reordered bottom-up.
func bottomUpOrder(allFuncs []*CompiledFunction) []*CompiledFunction {
	indexOf := make(map[*CompiledFunction]int, len(allFuncs))
	for i, cf := range allFuncs {
		indexOf[cf] = i
	}
	visited := make([]uint8, len(allFuncs))
	out := make([]*CompiledFunction, 0, len(allFuncs))
	var visit func(i int)
	visit = func(i int) {
		if visited[i] != 0 {
			return
		}
		visited[i] = 1
		cf := allFuncs[i]
		for siteIndex := range cf.callSites {
			callee := cf.callSites[siteIndex].cachedCallee
			if callee == nil {
				continue
			}
			j, ok := indexOf[callee]
			if !ok {
				continue
			}
			if visited[j] == 1 {
				continue
			}
			visit(j)
		}
		visited[i] = 2
		out = append(out, cf)
	}
	for i := range allFuncs {
		visit(i)
	}
	return out
}

// canInline decides whether a (caller, callee, site) tuple is eligible.
//
// Most refusals are callee-properties (closure, defer, recursion) and are cached on the
// callee via cf.cachedInlineRefusal so the body is not re-scanned for every call site.
// Per-site refusals (size headroom, register overflow) are computed fresh.
//
// Takes caller (*CompiledFunction) whose body would absorb the splice.
// Takes site (*callSite) which describes the candidate call.
// Takes _ (int) which is unused; kept on the signature for symmetry with callers that
// already supply the value.
// Takes inLoop (bool) which selects loopInlineBudget over defaultInlineBudget.
//
// Returns inlineEligible on success or one of the inlineRefusal* constants describing why
// the splice was refused.
func canInline(caller *CompiledFunction, site *callSite, _ int, inLoop bool) inlineRefusal {
	if reason := canInlineSiteShape(site); reason != inlineEligible {
		return reason
	}
	callee := site.cachedCallee
	if callee == caller {
		return canInlineSelfRecursive(site, callee, inLoop)
	}
	if callee.inRecursionCycle {
		return inlineRefusalRecursion
	}
	if reason := calleeInlineRefusal(callee); reason != inlineEligible {
		return reason
	}
	if reason := canInlineBodyBudget(caller, callee, inLoop); reason != inlineEligible {
		return reason
	}
	return canInlineRegisterBudget(caller, callee)
}

// canInlineSiteShape rejects call sites the inliner cannot handle at all.
//
// Takes site (*callSite) which is the call site to classify.
//
// Returns inlineEligible when the shape is acceptable, or a specific refusal reason
// otherwise.
func canInlineSiteShape(site *callSite) inlineRefusal {
	if site == nil {
		return inlineRefusalUnknown
	}
	if site.isNative || site.isClosure || site.isMethod {
		return inlineRefusalSiteIndirect
	}
	if site.isEllipsisSpread {
		return inlineRefusalVariadic
	}
	if site.cachedCallee == nil {
		return inlineRefusalNoBody
	}
	return inlineEligible
}

// canInlineBodyBudget enforces the hairyness-weighted size budget and the absolute caller
// body cap.
//
// Takes caller (*CompiledFunction) which is the function receiving the splice.
// Takes callee (*CompiledFunction) which is the function being spliced in.
// Takes inLoop (bool) which is true when the call site sits inside a loop body.
//
// Returns inlineEligible when both budgets pass, or a specific refusal reason otherwise.
func canInlineBodyBudget(caller, callee *CompiledFunction, inLoop bool) inlineRefusal {
	budget := defaultInlineBudget
	if inLoop {
		budget = loopInlineBudget
	}
	if calleeHairyness(callee) > budget {
		return inlineRefusalOversize
	}
	if len(caller.body)+len(callee.body) > maxCallerBodyAfterInline {
		return inlineRefusalCallerCap
	}
	return inlineEligible
}

// canInlineRegisterBudget rejects splices that would push the caller's per-bank register
// footprint past the watermark.
//
// Takes caller (*CompiledFunction) which is the function receiving the splice.
// Takes callee (*CompiledFunction) which is the function being spliced in.
//
// Returns inlineEligible when all banks stay under the watermark, or
// inlineRefusalCapWatermark otherwise.
func canInlineRegisterBudget(caller, callee *CompiledFunction) inlineRefusal {
	for kind := range registerKind(NumRegisterKinds) {
		paramCountK := countParamsInBank(callee, kind)
		localCountK := max(int(callee.numRegisters[kind])-paramCountK, 0)
		if int(caller.numRegisters[kind])+localCountK > registerBankWatermark {
			return inlineRefusalCapWatermark
		}
	}
	return inlineEligible
}

// canInlineSelfRecursive gates 1-level recursive unrolling for the special case where the
// call site's callee is the caller itself. The general inliner refuses recursive callees
// outright; this helper permits a single splice per site, with the inner recursive call
// inside the spliced body left as a regular opCall.
//
// Refusal modes are inlineRefusalUnrollDisabled when the build-tag flag is off;
// inlineRefusalAlreadyUnrolled when this site already absorbed one splice in a prior
// iteration of the inliner; inlineRefusalSelfInLoop when the call site is inside a loop
// body (declined to avoid bloating hot loops); inlineRefusalSelfHairy when the callee
// body is too large for the tighter selfUnrollBudget; and inlineRefusalTailCall when the
// callee body contains opTailCall (the blocker list also catches this via
// calleeInlineRefusal; checked explicitly here so the refusal code is precise).
//
// Takes site (*callSite) which is the call site under consideration.
// Takes callee (*CompiledFunction) which is the function being recursively spliced.
// Takes inLoop (bool) which is true when the call site sits inside a loop body.
//
// Returns inlineEligible when all gates pass.
func canInlineSelfRecursive(site *callSite, callee *CompiledFunction, inLoop bool) inlineRefusal {
	if !unrollSelfRecursiveEnabled {
		return inlineRefusalUnrollDisabled
	}
	if site.recursionUnrolled {
		return inlineRefusalAlreadyUnrolled
	}
	if inLoop {
		return inlineRefusalSelfInLoop
	}
	if reason := calleeInlineRefusal(callee); reason != inlineEligible {
		return reason
	}
	if calleeHairyness(callee) > selfUnrollBudget {
		return inlineRefusalSelfHairy
	}
	return inlineEligible
}

// calleeInlineRefusal returns the per-callee refusal reason.
//
// Uses the cached value when available. On first call, scans the body for
// refusal-triggering opcodes and caches the result. The cache is only valid while the
// callee body is fixed after initial compilation; the inliner runs in
// runPostCompilationChecks after optimise() and before any body modifications by the
// inliner itself, so this holds at scan time. If later phases mutate a callee body (e.g.,
// via recursive inlining), they MUST clear callee.cachedInlineRefusal.
//
// Takes callee (*CompiledFunction) whose body is probed.
//
// Returns the cached or freshly-computed inlineRefusal.
func calleeInlineRefusal(callee *CompiledFunction) inlineRefusal {
	if callee.cachedInlineRefusal != inlineRefusalUnknown {
		return callee.cachedInlineRefusal
	}
	reason := scanCalleeForRefusal(callee)
	callee.cachedInlineRefusal = reason
	return reason
}

// scanCalleeForRefusal answers eligibility from the emit-time flag.
//
// Consults cf.emittedInlineBlocker first. When the flag is unset (the case for synthetic
// CompiledFunctions built directly by tests without going through emit), falls back to a
// body walk so the existing unit-test contract still holds.
//
// Takes callee (*CompiledFunction) whose body is probed.
//
// Returns the inlineRefusal describing why the callee was refused, or inlineEligible.
func scanCalleeForRefusal(callee *CompiledFunction) inlineRefusal {
	if len(callee.body) == 0 {
		return inlineRefusalNoBody
	}
	if len(callee.upvalueDescriptors) > 0 {
		return inlineRefusalUpvalues
	}
	if callee.isGenericFunc && callee.specialisationOrigin == nil {
		return inlineRefusalGenericPlaceholder
	}
	if callee.emittedInlineBlocker != inlineRefusalUnknown {
		if callee.emittedInlineBlocker == inlineRefusalTailCall && calleeTailCallsAreAllCrossTarget(callee) {
			return inlineEligible
		}
		return callee.emittedInlineBlocker
	}
	for i := range callee.body {
		r := blockerForOpcode(callee.body[i].op)
		if r == inlineRefusalUnknown {
			continue
		}
		if r == inlineRefusalTailCall && calleeTailCallsAreAllCrossTarget(callee) {
			continue
		}
		return r
	}
	return inlineEligible
}

// calleeTailCallsAreAllCrossTarget reports whether every opTailCall in callee targets a
// different function.
//
// When all tail calls are cross-target, they can be safely re-lowered to opCall at splice
// time because the inlined body's continuation is the caller's frame, not recursive stack
// growth into the callee. Self-tail calls cannot be re-lowered without creating unbounded
// frame growth. Unresolved tail-call sites (cachedCallee == nil) are treated as
// ineligible because the recursive-versus-cross-target classification is unknown at
// splice time.
//
// Takes callee (*CompiledFunction) which is the function whose tail calls are being
// classified.
//
// Returns true when every opTailCall is unambiguously cross-target, false otherwise.
func calleeTailCallsAreAllCrossTarget(callee *CompiledFunction) bool {
	for i := range callee.body {
		instr := callee.body[i]
		if instr.op != opTailCall {
			continue
		}
		siteIndex := instr.wideIndex()
		if int(siteIndex) >= len(callee.callSites) {
			return false
		}
		site := &callee.callSites[siteIndex]
		if site.cachedCallee == nil {
			return false
		}
		if site.cachedCallee == callee {
			return false
		}
	}
	return true
}

// blockerForOpcode returns the refusal implied by an opcode appearing in a callee.
//
// Used by emit() to populate cf.emittedInlineBlocker, and by scanCalleeForRefusal's
// fallback body walk.
//
// Takes op (opcode) which is the opcode under inspection.
//
// Returns the inlineRefusal kind when op blocks inlining, or inlineRefusalUnknown when op
// does not block inlining.
func blockerForOpcode(op opcode) inlineRefusal {
	switch op {
	case opDefer:
		return inlineRefusalDefer
	case opGo:
		return inlineRefusalGo
	case opCallMethod, opCallMethodInlineable:
		return inlineRefusalMethodCall
	case opCallNative:
		return inlineRefusalNativeCall
	case opTailCall:
		return inlineRefusalTailCall
	case opCallIIFE, opMakeClosure, opGetUpvalue, opSetUpvalue,
		opSyncClosureUpvalues:
		return inlineRefusalClosureOps
	case opSelect, opChannelSend:
		return inlineRefusalChannelOps
	default:
	}
	return inlineRefusalUnknown
}

// calleeHairyness computes a weighted cost score for inlining a callee.
//
// Roughly tracks the Go compiler's inline/inl.go model. Each opcode contributes per
// hairyOpcodeCost: trivial moves/loads/arithmetic count as 1, allocation-driven ops
// (opMakeSlice, opMakeClosure, opAppend, opAppendSpread, opMapSet) count as 2, and opNop
// costs 0. The point is to bias the budget away from callees that already pay heavy
// per-op costs (where inlining gives less relative benefit than for a body of cheap
// typed-bank ops).
//
// Takes callee (*CompiledFunction) whose body is summed.
//
// Returns the total hairyness score, or 0 when callee is nil.
func calleeHairyness(callee *CompiledFunction) int {
	if callee == nil {
		return 0
	}
	score := 0
	for i := range callee.body {
		score += hairyOpcodeCost(callee.body[i].op)
	}
	return score
}

// hairyOpcodeCost returns the per-opcode contribution to the score.
//
// Most opcodes weight 1 (representing one instruction's worth of caller-body growth).
// Heavier weights only for genuinely allocation-driven ops where the per-instruction cost
// dominates and inlining offers proportionally less benefit.
//
// Takes op (opcode) which is the opcode being weighed.
//
// Returns 0 for opNop, 2 for allocation-driven ops, or 1 for all others.
func hairyOpcodeCost(op opcode) int {
	switch op {
	case opMakeSlice, opMakeClosure,
		opAppend, opAppendSpread,
		opMapSet:
		return 2
	case opNop:
		return 0
	default:
	}
	return 1
}

// countParamsInBank counts callee parameters that live in the given bank.
//
// Takes callee (*CompiledFunction) whose parameterKinds are scanned.
// Takes bank (registerKind) which is the bank being counted.
//
// Returns the number of parameters whose registerKind matches bank.
func countParamsInBank(callee *CompiledFunction, bank registerKind) int {
	n := 0
	for _, k := range callee.parameterKinds {
		if k == bank {
			n++
		}
	}
	return n
}

// findOpCallPC locates the opCall instruction for a given call-site index.
//
// For the common opCall case the site index lives in instruction.b | (instruction.c <<
// 8). Matching is restricted to opCall because other call variants encode the site index
// differently.
//
// Takes caller (*CompiledFunction) whose body is searched.
// Takes siteIndex (uint16) which is the call-site index to find.
//
// Returns the PC of the matching opCall, or -1 when none is found.
func findOpCallPC(caller *CompiledFunction, siteIndex uint16) int {
	for i := range caller.body {
		instr := caller.body[i]
		if instr.op != opCall {
			continue
		}
		index := uint16(instr.b) | uint16(instr.c)<<instructionByteShift
		if index == siteIndex {
			return i
		}
	}
	return -1
}

// computeInLoopMask flags every PC that sits inside a loop body.
//
// A PC is "in loop" when some backward jump in the body has a source PC after it and a
// target PC at-or-before it. Computed in O(N) by walking the body, recording every
// backward jump as a (target_pc, source_pc) interval, and marking every PC in any
// interval. Conservative: nested loops mark the same PC multiple times (harmless).
// Forward-jump constructs (if/else, switch) do not mark anything as in-loop, which is
// correct.
//
// Takes caller (*CompiledFunction) whose body is scanned.
//
// Returns a slice indexed by PC, true where the PC is inside a loop.
func computeInLoopMask(caller *CompiledFunction) []bool {
	n := len(caller.body)
	if n == 0 {
		return nil
	}
	mask := make([]bool, n)
	for i := range n {
		instr := caller.body[i]
		offset, isJump := jumpOffsetOf(instr)
		if !isJump {
			continue
		}
		if offset >= 0 {
			continue
		}
		targetPC := i + 1 + offset
		if targetPC < 0 || targetPC > i {
			continue
		}
		for pc := targetPC; pc <= i && pc < n; pc++ {
			mask[pc] = true
		}
	}
	return mask
}

// jumpOffsetOf decodes a jump instruction's relative offset.
//
// Recognises the standalone jump opcodes and the fused compare+jump variants, all of
// which carry the offset in operands B and C.
//
// Takes instr (instruction) which is the candidate jump instruction.
//
// Returns the signed offset.
// Returns true when the instruction is a recognised jump, false otherwise.
func jumpOffsetOf(instr instruction) (int, bool) {
	if instr.op == opDrillTier1 && subOpcode(instr.a) == subOpJump {
		return decodeJumpOffset(instr), true
	}
	switch instr.op {
	case opJumpIfTrue, opJumpIfFalse,
		opLeIntConstJumpFalse, opLtIntConstJumpFalse,
		opLtIntJumpFalse, opLeIntJumpFalse,
		opGtIntJumpFalse, opGeIntJumpFalse,
		opEqIntJumpFalse, opNeIntJumpFalse,
		opEqIntConstJumpFalse, opEqIntConstJumpTrue,
		opGeIntConstJumpFalse, opGtIntConstJumpFalse,
		opAddIntJump,
		opEqStringConstJumpFalse,
		opTestNilJumpTrue, opTestNilJumpFalse:
		return decodeJumpOffset(instr), true
	default:
	}
	return 0, false
}

// decodeJumpOffset extracts the signed 16-bit offset from a jump instruction.
//
// Mirrors joinOffset(b, c) but kept local to the inliner.
//
// Takes instr (instruction) whose B and C bytes carry the packed offset.
//
// Returns the sign-extended int offset.
func decodeJumpOffset(instr instruction) int {
	lo := uint16(instr.b)
	hi := uint16(instr.c)
	raw := lo | hi<<instructionByteShift
	return int(safeconv.Uint16ToInt16(raw))
}
