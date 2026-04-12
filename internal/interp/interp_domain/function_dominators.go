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

const (
	// maxDominatorIterations caps the iterative dataflow loop used by
	// computeFunctionDominators. The bitset lattice converges quickly for real CFGs; the cap
	// protects against pathological non-convergence.
	maxDominatorIterations = 64
)

// Full-function dominator computation runs classic iterative dataflow, generalising the
// loop-scoped `computeBackEdgeDominators` in function_licm_field_read.go to the entire
// function body.
//
// Used by GVN (function_gvn_pass.go) to verify that a candidate earlier definition of a
// value provably executes on every path to the later use, a precondition for replacing
// the use with a register move from the earlier definition.

// functionDominators holds per-PC dominator bitsets for every reachable PC in a function
// body. dominators[pc][k] == true iff PC k dominates PC pc, meaning every path from the
// function entry to pc passes through k.
//
// PCs not reached by buildFunctionDominators have a nil bitset; the dominates() query
// returns false for those, matching their status as dead code that GVN must not consider
// as a candidate definition.
type functionDominators struct {
	// bits indexes the per-PC dominator bitset; bits[pc][k] is true when PC k dominates PC
	// pc, and bits[pc] is nil for unreachable PCs.
	bits [][]bool
}

// dominates reports whether definePC dominates usePC, meaning every path from the
// function entry to usePC passes through definePC.
//
// Takes definePC (int) which is the program counter of the candidate dominator
// definition.
// Takes usePC (int) which is the program counter of the use under test.
//
// Returns false for unreachable PCs (nil bitset) and for PCs outside the function's body
// length.
func (dom *functionDominators) dominates(definePC, usePC int) bool {
	if dom == nil {
		return false
	}
	if usePC < 0 || usePC >= len(dom.bits) {
		return false
	}
	row := dom.bits[usePC]
	if row == nil {
		return false
	}
	if definePC < 0 || definePC >= len(row) {
		return false
	}
	return row[definePC]
}

// computeFunctionDominators returns the dominator table for body. Uses the textbook
// iterative dataflow where dom[entry] is {entry}, dom[pc] is {pc} unioned with the
// intersection of dom[pred] for every predecessor of pc, iterated until no set changes.
//
// PCs unreachable from the entry never have their bitset populated; they stay nil, which
// the dominates() query interprets as "not a valid definition site". The dataflow loop
// bounds itself at maxDominatorIterations to prevent pathological non-convergence.
//
// When the dataflow does NOT reach a fixpoint within the iteration cap, the table is
// non-exact: dominates() could then return a wrong "definePC dominates usePC" answer and
// let GVN reuse a value that is not actually available on every path. Rather than hand
// back an unsound table, the helper returns nil in that case, which GVN treats as "no
// dominator information" and falls back to performing no rewrites. A missed optimisation
// is always acceptable; an unsound dominance answer is not.
//
// Takes body ([]instruction) which is the linear instruction stream of the function under
// analysis.
//
// Returns the populated dominator table, or nil when body is empty or the iterative
// dataflow did not converge within maxDominatorIterations.
func computeFunctionDominators(body []instruction) *functionDominators {
	if len(body) == 0 {
		return nil
	}
	n := len(body)
	dom := initialiseDominatorTable(n)
	predecessors := buildAliasPredecessors(body)
	reachable := computeReachable(body, n)
	seedReachableDominators(dom, n, reachable)
	converged := false
	for range maxDominatorIterations {
		if !relaxDominatorsOnce(dom, n, predecessors, reachable) {
			converged = true
			break
		}
	}
	if !converged {
		return nil
	}
	return dom
}

// initialiseDominatorTable allocates the dominator structure and seeds the entry PC's
// bitset to {0}.
//
// Takes n (int) which is the number of PCs in the function body.
//
// Returns a dominator table with the entry row populated and all other rows nil.
func initialiseDominatorTable(n int) *functionDominators {
	dom := &functionDominators{bits: make([][]bool, n)}
	dom.bits[0] = make([]bool, n)
	dom.bits[0][0] = true
	return dom
}

// seedReachableDominators fills every reachable non-entry PC's bitset with the universal
// "every PC dominates" starting state.
//
// Takes dom (*functionDominators) which is the dominator table whose rows are being
// seeded.
// Takes n (int) which is the number of PCs in the function body.
// Takes reachable ([]bool) which is the per-PC reachability mask from the entry.
func seedReachableDominators(dom *functionDominators, n int, reachable []bool) {
	universe := make([]bool, n)
	for index := range universe {
		universe[index] = true
	}
	for pc := 1; pc < n; pc++ {
		if !reachable[pc] {
			continue
		}
		dom.bits[pc] = make([]bool, n)
		copy(dom.bits[pc], universe)
	}
}

// relaxDominatorsOnce performs one sweep of the dataflow relaxation.
//
// Takes dom (*functionDominators) which is the dominator table mutated in place.
// Takes n (int) which is the number of PCs in the function body.
// Takes predecessors ([][]int) which is the per-PC predecessor lists from the alias CFG.
// Takes reachable ([]bool) which is the per-PC reachability mask from the entry.
//
// Returns true when any PC's bitset changed, prompting another sweep.
func relaxDominatorsOnce(dom *functionDominators, n int, predecessors [][]int, reachable []bool) bool {
	changed := false
	for pc := 1; pc < n; pc++ {
		if !reachable[pc] {
			continue
		}
		preds := predecessors[pc]
		if len(preds) == 0 {
			continue
		}
		intersected := computeDominatorIntersection(dom.bits, preds, n)
		intersected[pc] = true
		if !boolSliceEqual(dom.bits[pc], intersected) {
			copy(dom.bits[pc], intersected)
			changed = true
		}
	}
	return changed
}

// computeReachable returns a slice indexed by PC marking every PC reachable from the
// function entry via the CFG built by buildAliasSuccessors. Unreachable PCs are skipped
// during dominator iteration so they remain nil in the table, which GVN treats as dead
// code that cannot serve as a candidate definition.
//
// Takes body ([]instruction) which is the linear instruction stream of the function under
// analysis.
// Takes n (int) which is the number of PCs in the function body.
//
// Returns the per-PC reachability mask from the entry.
func computeReachable(body []instruction, n int) []bool {
	reachable := make([]bool, n)
	if n == 0 {
		return reachable
	}
	reachable[0] = true
	successors := buildAliasSuccessors(body)
	worklist := []int{0}
	for len(worklist) > 0 {
		pc := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		for _, succ := range successors[pc] {
			if succ < 0 || succ >= n {
				continue
			}
			if reachable[succ] {
				continue
			}
			reachable[succ] = true
			worklist = append(worklist, succ)
		}
	}
	return reachable
}

// computeDominatorIntersection returns the bitwise AND of the dominator bitsets of every
// reachable predecessor.
//
// Takes bits ([][]bool) which is the per-PC dominator bitsets from the dominator table.
// Takes preds ([]int) which is the predecessor PCs of the PC under relaxation.
// Takes n (int) which is the number of PCs in the function body.
//
// Returns the intersected bitset, zeroed when no predecessor was reachable.
func computeDominatorIntersection(bits [][]bool, preds []int, n int) []bool {
	result := make([]bool, n)
	seeded := false
	for _, pred := range preds {
		predBits := lookupReachableDomBits(bits, pred)
		if predBits == nil {
			continue
		}
		if !seeded {
			copy(result, predBits)
			seeded = true
			continue
		}
		intersectInto(result, predBits)
	}
	if !seeded {
		clear(result)
	}
	return result
}

// lookupReachableDomBits returns bits[pred] when the predecessor is in range and has been
// visited (its bitset allocated).
//
// Takes bits ([][]bool) which is the per-PC dominator bitsets from the dominator table.
// Takes pred (int) which is the predecessor PC to look up.
//
// Returns the predecessor's bitset, or nil when out of range or unreachable.
func lookupReachableDomBits(bits [][]bool, pred int) []bool {
	if pred < 0 || pred >= len(bits) {
		return nil
	}
	return bits[pred]
}

// intersectInto applies the bitwise AND of source into destination in place, clearing any
// destination position where source is false.
//
// Takes destination ([]bool) which is the bitset mutated in place.
// Takes source ([]bool) which is the bitset whose false positions clear destination.
func intersectInto(destination, source []bool) {
	for index := range destination {
		if !source[index] {
			destination[index] = false
		}
	}
}

// boolSliceEqual reports whether two slices of bool hold identical values in every
// position.
//
// Takes a ([]bool) which is the first slice to compare.
// Takes b ([]bool) which is the second slice to compare.
//
// Returns true when lengths match and every position is equal, otherwise false.
func boolSliceEqual(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}
