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
	// heapPureCallee marks a function whose body provably mutates no heap-resident state (no
	// opSet*, opMapSet, opIndexSet, opAddr, no SET sub-ops) and whose every
	// transitively-reachable callee is also heapPureCallee.
	heapPureCallee heapMutationClassification = iota + 1

	// heapMutatingCallee marks a function whose body contains any heap mutator or which
	// transitively calls another mutating function.
	heapMutatingCallee

	// maxHeapPurityFixpointIter caps the SCC fixpoint loop as a safety net for unforeseen
	// pathological call graphs.
	//
	// The lattice is two-valued (pure / mutating) and only flows toward "mutating", so the
	// fixpoint converges in O(SCC depth) iterations.
	maxHeapPurityFixpointIter = 16
)

// heapMutationClassification labels a function by its observable effect on the heap.
//
// The CSE and LICM passes use the classification at opCall sites whose callee is
// statically resolvable to decide whether the call invalidates cached struct-field reads.
// The zero value represents "unknown"; callers treat unknown as "may mutate" so the
// absence of classification is safe.
type heapMutationClassification uint8

// runHeapPurityAnalysis classifies every reachable function by its heap-mutation
// behaviour and sets CompiledFunction.heapMutationClass to either heapPureCallee or
// heapMutatingCallee.
//
// Invoked from runPostCompilationChecks after the inliner has spliced callee bodies. The
// inliner can fold pure callees into their callers; remaining callees appear as opCall
// instructions referencing cachedCallee.
//
// Takes ctx (context.Context) which carries cancellation.
// Takes root (*CompiledFunction) which is the program's top-level compiled function whose
// nested functions are walked.
//
// Returns error when the context is cancelled before completion.
func runHeapPurityAnalysis(ctx context.Context, root *CompiledFunction) error {
	if root == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("runHeapPurityAnalysis cancelled: %w", err)
	}
	all := collectReachableFunctions(root)
	if len(all) == 0 {
		return nil
	}
	for _, cf := range all {
		cf.heapMutationClass = heapPureCallee
	}
	for range maxHeapPurityFixpointIter {
		if !demoteMutatingFunctions(all) {
			return nil
		}
	}
	demoteAllToMutating(all)
	return nil
}

// demoteAllToMutating forces every function to heapMutatingCallee.
//
// Reached only when the purity fixpoint did NOT converge within maxHeapPurityFixpointIter
// passes. A non-converged table could still carry a function wrongly marked
// heapPureCallee while a deeper transitive callee is mutating; LICM and CSE would then
// elide a cached struct-field read across a call that actually mutates the heap,
// returning a stale value. Collapsing every function to "mutating" is the conservative
// fallback: calls then always invalidate cached reads, which only costs optimisation and
// never correctness. Non-convergence requires a call chain deeper than the iteration cap,
// so in practice this path is never taken.
//
// Takes functions ([]*CompiledFunction) whose classification is forced to
// heapMutatingCallee.
func demoteAllToMutating(functions []*CompiledFunction) {
	for _, cf := range functions {
		cf.heapMutationClass = heapMutatingCallee
	}
}

// demoteMutatingFunctions performs a single fixpoint pass. Each function whose body
// contains any heap-mutator instruction, or any call to an already-classified mutating
// callee, is demoted from heapPureCallee to heapMutatingCallee.
//
// Takes functions ([]*CompiledFunction) being classified.
//
// Returns true when at least one function flipped in this pass.
func demoteMutatingFunctions(functions []*CompiledFunction) bool {
	changed := false
	for _, cf := range functions {
		if cf.heapMutationClass == heapMutatingCallee {
			continue
		}
		if functionHasHeapMutator(cf) {
			cf.heapMutationClass = heapMutatingCallee
			changed = true
		}
	}
	return changed
}

// functionHasHeapMutator reports whether cf's body either directly mutates the heap
// (struct-field write, map set, etc.) or transitively does so via a statically-resolvable
// call to a heapMutatingCallee or an unknown callee.
//
// Unknown callees include opCallNative/opCallMethod with non-static receiver types,
// closures, and method dispatch through interface types. All are conservatively treated
// as mutating.
//
// Takes cf (*CompiledFunction) being classified.
//
// Returns true when cf is impure.
func functionHasHeapMutator(cf *CompiledFunction) bool {
	for _, inst := range cf.body {
		if instructionDirectlyMutatesHeap(inst) {
			return true
		}
		if !isCallOpcode(inst.op) {
			continue
		}
		if callInvalidatesPurity(cf, inst) {
			return true
		}
	}
	return false
}

// instructionDirectlyMutatesHeap reports whether inst, in isolation (ignoring call
// targets), can mutate heap-resident state.
//
// Mirrors the conservative block list used by the CSE and LICM passes, minus the call
// opcodes; callInvalidatesPurity handles those separately so the analyser can defer to
// per-callee classification.
//
// Takes inst (instruction) which is the instruction under inspection.
//
// Returns true when the instruction itself performs a heap mutation.
func instructionDirectlyMutatesHeap(inst instruction) bool {
	switch inst.op {
	case opSetField, opIndexSet, opAddr,
		opSetStructFieldIntT0, opSetStructFieldUint, opSetStructFieldFloat,
		opSetStructFieldBool, opSetStructFieldGeneral,
		opSwapStructFieldsGeneralT0,
		opMapSet,
		opMakeClosure,
		opSyncClosureUpvalues, opSetUpvalue,
		opSetGlobal, opSetGlobalWide:
		return true
	default:
	}
	if inst.op == opDrillTier1 && subOpDrillTier1MutatesHeap(inst) {
		return true
	}
	return false
}

// isCallOpcode reports whether op transfers control to another function, regardless of
// static resolvability.
//
// Takes op (opcode) which is the opcode under inspection.
//
// Returns true when the opcode is a call variant.
func isCallOpcode(op opcode) bool {
	switch op {
	case opCall, opTailCall, opCallMethod, opCallMethodInlineable, opCallNative, opCallIIFE:
		return true
	default:
	}
	return false
}

// callInvalidatesPurity reports whether the call instruction at inst invalidates cf's
// purity classification.
//
// A call invalidates when the opcode is opCallMethod or opCallNative (targets resolved at
// runtime), opCallIIFE (synthesised inner function), opCall or opTailCall with no
// cachedCallee (forward reference or closure call), or when the cachedCallee's
// heapMutationClass is heapMutatingCallee.
//
// Takes cf (*CompiledFunction) which is the function being classified.
// Takes inst (instruction) which is the candidate call instruction.
//
// Returns false only when the callee is statically resolvable and classified as
// heapPureCallee; true otherwise.
func callInvalidatesPurity(cf *CompiledFunction, inst instruction) bool {
	switch inst.op {
	case opCallMethod, opCallMethodInlineable, opCallNative, opCallIIFE:
		return true
	case opCall, opTailCall:
		callee := resolveCachedCallee(cf, inst)
		if callee == nil {
			return true
		}
		return callee.heapMutationClass != heapPureCallee
	default:
	}
	return true
}

// resolveCachedCallee returns the statically-resolved callee for the call at inst, or nil
// when the call site has no cachedCallee.
//
// inst's operands B and C encode the call-site index as a uint16 (wideIndex).
// cf.callSites maps the index to the resolved callee; closure and forward-reference sites
// have a nil cachedCallee.
//
// Takes cf (*CompiledFunction) whose call sites are consulted.
// Takes inst (instruction) which carries the wide call-site index.
//
// Returns the resolved callee, or nil when no cachedCallee is recorded.
func resolveCachedCallee(cf *CompiledFunction, inst instruction) *CompiledFunction {
	siteIndex := int(inst.wideIndex())
	if siteIndex < 0 || siteIndex >= len(cf.callSites) {
		return nil
	}
	return cf.callSites[siteIndex].cachedCallee
}
