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

// Defines feature flags for controlling which Go language constructs are allowed during
// compilation. Provides bitmask constants for loops, recursion, goroutines, channels, and
// other features with predefined sets for restricted and minimal execution environments.

import (
	"fmt"
	"slices"
)

const (
	// InterpFeatureForLoops allows for statements.
	InterpFeatureForLoops InterpFeature = 1 << iota

	// InterpFeatureRangeLoops allows range iterations.
	InterpFeatureRangeLoops

	// InterpFeatureRecursion allows direct and mutual recursion. Checked post-compilation
	// via call graph analysis.
	InterpFeatureRecursion

	// InterpFeatureGoroutines allows go statements.
	InterpFeatureGoroutines

	// InterpFeatureChannels allows channel operations (make, send, receive, close, select).
	InterpFeatureChannels

	// InterpFeatureDefer allows defer statements.
	InterpFeatureDefer

	// InterpFeatureGoto allows goto statements.
	InterpFeatureGoto

	// InterpFeatureClosures allows function literals.
	InterpFeatureClosures

	// InterpFeatureUnsafeOps allows unsafe package operations.
	InterpFeatureUnsafeOps

	// InterpFeaturePanicRecover allows panic and recover calls.
	InterpFeaturePanicRecover

	// InterpFeaturesNone is the default value that disables all language features.
	InterpFeaturesNone InterpFeature = 0

	// InterpFeaturesAll enables all language features. This is the default for Piko dev
	// mode.
	InterpFeaturesAll = InterpFeatureForLoops | InterpFeatureRangeLoops |
		InterpFeatureRecursion | InterpFeatureGoroutines |
		InterpFeatureChannels | InterpFeatureDefer |
		InterpFeatureGoto | InterpFeatureClosures |
		InterpFeatureUnsafeOps | InterpFeaturePanicRecover

	// InterpFeaturesRestricted allows most features but disables goroutines, unsafe
	// operations, goto, and panic/recover. Suitable for CMS environments where concurrency
	// and low-level access are not needed.
	InterpFeaturesRestricted = InterpFeatureForLoops | InterpFeatureRangeLoops |
		InterpFeatureRecursion | InterpFeatureChannels |
		InterpFeatureDefer | InterpFeatureClosures

	// InterpFeaturesMinimal allows only basic sequential code with no loops, recursion,
	// goroutines, channels, defer, goto, closures, unsafe, or panic/recover. Suitable for
	// simple expression evaluation.
	InterpFeaturesMinimal InterpFeature = 0
)

// InterpFeature is a bitmask selecting which interpreter language features (loops,
// closures, goroutines, etc.) are permitted in a compiled program. Use the
// InterpFeatures* constants for set members.
type InterpFeature uint32

// Has checks if the feature set includes the given feature.
//
// Takes feature (InterpFeature) which is the feature to check for.
//
// Returns bool which is true if the feature is present in the set.
func (f InterpFeature) Has(feature InterpFeature) bool {
	return f&feature == feature
}

// String returns a readable name for the feature for use in error messages.
//
// Returns string which is the display name shown in diagnostics.
func (f InterpFeature) String() string {
	switch f {
	case InterpFeatureForLoops:
		return "for loops"
	case InterpFeatureRangeLoops:
		return "range loops"
	case InterpFeatureRecursion:
		return "recursion"
	case InterpFeatureGoroutines:
		return "goroutines"
	case InterpFeatureChannels:
		return "channels"
	case InterpFeatureDefer:
		return "defer"
	case InterpFeatureGoto:
		return "goto"
	case InterpFeatureClosures:
		return "closures"
	case InterpFeatureUnsafeOps:
		return "unsafe operations"
	case InterpFeaturePanicRecover:
		return "panic/recover"
	default:
		return fmt.Sprintf("InterpFeature(%d)", uint32(f))
	}
}

// tarjanState holds the mutable state for a single Tarjan SCC pass. Keeping it in a
// struct avoids the nested-closure shape that inflates cognitive complexity in
// classifyLocalEscape's caller.
type tarjanState struct {
	// adjacency is the call-graph adjacency list under analysis.
	adjacency [][]uint16

	// inSCC marks each node true when it belongs to a non-trivial SCC.
	inSCC []bool

	// indices stores each node's DFS discovery index, -1 when unvisited.
	indices []int

	// lowLinks stores each node's Tarjan low-link value.
	lowLinks []int

	// onStack marks each node true while it sits on the DFS stack.
	onStack []bool

	// stack holds the node indices currently in the active SCC frame.
	stack []int

	// index is the next DFS discovery counter to assign.
	index int
}

// strongconnect performs a single Tarjan DFS step starting from node, updating low-link
// values and emitting any SCC rooted at it.
//
// Takes node which is the DFS entry-point index into adjacency.
func (state *tarjanState) strongconnect(node int) {
	state.indices[node] = state.index
	state.lowLinks[node] = state.index
	state.index++
	state.stack = append(state.stack, node)
	state.onStack[node] = true
	state.exploreNeighbours(node)
	if state.lowLinks[node] == state.indices[node] {
		state.emitSCCRootedAt(node)
	}
}

// exploreNeighbours walks every neighbour of node, recursing into unvisited ones and
// lowering node's low-link via back-edges to nodes currently on the stack.
//
// Takes node which is the DFS frame whose neighbours are walked.
func (state *tarjanState) exploreNeighbours(node int) {
	nodeCount := len(state.adjacency)
	for _, adjacent := range state.adjacency[node] {
		neighbour := int(adjacent)
		if neighbour < 0 || neighbour >= nodeCount {
			continue
		}
		if state.indices[neighbour] == -1 {
			state.strongconnect(neighbour)
			if state.lowLinks[neighbour] < state.lowLinks[node] {
				state.lowLinks[node] = state.lowLinks[neighbour]
			}
		} else if state.onStack[neighbour] {
			if state.indices[neighbour] < state.lowLinks[node] {
				state.lowLinks[node] = state.indices[neighbour]
			}
		}
	}
}

// emitSCCRootedAt pops the stack down to node, marking each member as in-SCC iff the
// component is non-trivial (size > 1 or self-edge).
//
// Takes node which is the SCC root index sitting on the DFS stack.
func (state *tarjanState) emitSCCRootedAt(node int) {
	startIndex := state.findStackIndex(node)
	memberCount := len(state.stack) - startIndex
	hasSelfLoop := memberCount == 1 && state.hasSelfEdge(node)
	if memberCount > 1 || hasSelfLoop {
		for stackIndex := startIndex; stackIndex < len(state.stack); stackIndex++ {
			state.inSCC[state.stack[stackIndex]] = true
		}
	}
	for stackIndex := startIndex; stackIndex < len(state.stack); stackIndex++ {
		state.onStack[state.stack[stackIndex]] = false
	}
	state.stack = state.stack[:startIndex]
}

// findStackIndex returns the stack index of node. The node is guaranteed to be on the
// stack because strongconnect only emits an SCC when its root sits at the bottom of an
// unpopped frame.
//
// Takes node which is the SCC root being located.
//
// Returns the stack position of node, or len(state.stack) when absent.
func (state *tarjanState) findStackIndex(node int) int {
	for stackIndex, stackedNode := range slices.Backward(state.stack) {
		if stackedNode == node {
			return stackIndex
		}
	}
	return len(state.stack)
}

// hasSelfEdge reports whether node has any edge back to itself in the adjacency list.
// Used to classify size-1 SCCs as either trivial (no self-edge) or non-trivial (direct
// recursion).
//
// Takes node which is the candidate for a self-loop edge.
//
// Returns true when adjacency[node] contains node itself.
func (state *tarjanState) hasSelfEdge(node int) bool {
	for _, adjacent := range state.adjacency[node] {
		if int(adjacent) == node {
			return true
		}
	}
	return false
}

// detectRecursion walks the call graph of a compiled function set and returns an error if
// any cycle (direct or mutual recursion) is found.
//
// Takes root (*CompiledFunction) which is the root function containing all compiled
// functions.
//
// Returns error which wraps errFeatureNotAllowed if a cycle is detected, or nil if the
// call graph is acyclic.
func detectRecursion(root *CompiledFunction) error {
	functions := root.functions
	if len(functions) == 0 {
		return nil
	}

	adjacency := buildCallAdjacency(functions)

	if callGraphHasCycle(adjacency) {
		return fmt.Errorf("%w: %s", errFeatureNotAllowed, InterpFeatureRecursion)
	}

	return nil
}

// buildCallAdjacency constructs an adjacency list for the call graph, excluding closure
// and method calls.
//
// Takes functions ([]*CompiledFunction) which are the compiled functions to analyse.
//
// Returns [][]uint16 where each entry lists the function indices called by that function.
func buildCallAdjacency(functions []*CompiledFunction) [][]uint16 {
	adjacency := make([][]uint16, len(functions))
	for i := range functions {
		seen := make(map[uint16]bool)
		for j := range functions[i].callSites {
			cs := &functions[i].callSites[j]
			if cs.isClosure || cs.isMethod {
				continue
			}
			index := cs.funcIndex
			if int(index) < len(functions) && !seen[index] {
				adjacency[i] = append(adjacency[i], index)
				seen[index] = true
			}
		}
	}
	return adjacency
}

// findCallGraphSCCs runs Tarjan's algorithm on the call-graph adjacency list and returns
// a per-node "isInNonTrivialSCC" flag. A non-trivial SCC has either (a) two or more nodes
// (mutual recursion) or (b) a single node with a self-edge (direct recursion).
//
// O(V + E). Used by the inliner to refuse inlining of recursive callees without aborting
// the entire pass; non-recursive functions in the same program remain eligible.
//
// Takes adjacency ([][]uint16) which is the call graph adjacency list.
//
// Returns []bool indexed identically; entry i is true iff function i is in a non-trivial
// SCC.
func findCallGraphSCCs(adjacency [][]uint16) []bool {
	nodeCount := len(adjacency)
	if nodeCount == 0 {
		return nil
	}
	state := newTarjanState(adjacency)
	for i := range adjacency {
		if state.indices[i] == -1 {
			state.strongconnect(i)
		}
	}
	return state.inSCC
}

// newTarjanState initialises a fresh Tarjan SCC state for the given adjacency list. All
// node indices start as -1 (unvisited).
//
// Takes adjacency which is the call-graph edges to analyse.
//
// Returns the initialised tarjanState ready for strongconnect.
func newTarjanState(adjacency [][]uint16) *tarjanState {
	nodeCount := len(adjacency)
	state := &tarjanState{
		adjacency: adjacency,
		inSCC:     make([]bool, nodeCount),
		indices:   make([]int, nodeCount),
		lowLinks:  make([]int, nodeCount),
		onStack:   make([]bool, nodeCount),
		stack:     make([]int, 0, nodeCount),
	}
	for i := range state.indices {
		state.indices[i] = -1
	}
	return state
}

// callGraphHasCycle performs a DFS-based cycle detection on a directed adjacency list.
//
// Takes adjacency ([][]uint16) which is the call graph adjacency list.
//
// Returns true if any cycle exists, false otherwise.
func callGraphHasCycle(adjacency [][]uint16) bool {
	const (
		white = 0
		grey  = 1
		black = 2
	)
	colour := make([]int, len(adjacency))

	var dfs func(u int) bool
	dfs = func(u int) bool {
		colour[u] = grey
		for _, v := range adjacency[u] {
			switch colour[v] {
			case grey:
				return true
			case white:
				if dfs(int(v)) {
					return true
				}
			}
		}
		colour[u] = black
		return false
	}

	for i := range adjacency {
		if colour[i] == white && dfs(i) {
			return true
		}
	}

	return false
}
