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

// Defines feature flags for controlling which Go language constructs are
// allowed during compilation. Provides bitmask constants for loops,
// recursion, goroutines, channels, and other features with predefined
// sets for restricted and minimal execution environments.

import "fmt"

// InterpFeature is a bitmask that controls which Go language features
// are allowed during compilation. When a feature is not present in the
// set, code using that construct will fail at compile time with a clear
// error message.
type InterpFeature uint32

const (
	// InterpFeatureForLoops allows for statements.
	InterpFeatureForLoops InterpFeature = 1 << iota

	// InterpFeatureRangeLoops allows range iterations.
	InterpFeatureRangeLoops

	// InterpFeatureRecursion allows direct and mutual recursion.
	// Checked post-compilation via call graph analysis.
	InterpFeatureRecursion

	// InterpFeatureGoroutines allows go statements.
	InterpFeatureGoroutines

	// InterpFeatureChannels allows channel operations (make, send,
	// receive, close, select).
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
)

const (
	// InterpFeaturesNone is the default value that disables all
	// language features.
	InterpFeaturesNone InterpFeature = 0

	// InterpFeaturesAll enables all language features. This is the
	// default for Piko dev mode.
	InterpFeaturesAll = InterpFeatureForLoops | InterpFeatureRangeLoops |
		InterpFeatureRecursion | InterpFeatureGoroutines |
		InterpFeatureChannels | InterpFeatureDefer |
		InterpFeatureGoto | InterpFeatureClosures |
		InterpFeatureUnsafeOps | InterpFeaturePanicRecover

	// InterpFeaturesRestricted allows most features but disables
	// goroutines, unsafe operations, goto, and panic/recover. Suitable
	// for CMS environments where concurrency and low-level access are
	// not needed.
	InterpFeaturesRestricted = InterpFeatureForLoops | InterpFeatureRangeLoops |
		InterpFeatureRecursion | InterpFeatureChannels |
		InterpFeatureDefer | InterpFeatureClosures

	// InterpFeaturesMinimal allows only basic sequential code with no
	// loops, recursion, goroutines, channels, defer, goto, closures,
	// unsafe, or panic/recover. Suitable for simple expression
	// evaluation.
	InterpFeaturesMinimal InterpFeature = 0
)

// Has checks if the feature set includes the given feature.
//
// Takes feature (InterpFeature) which is the feature to check for.
//
// Returns bool which is true if the feature is present in the set.
func (f InterpFeature) Has(feature InterpFeature) bool {
	return f&feature == feature
}

// String returns a readable name for the feature for use in error
// messages.
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

// detectRecursion walks the call graph of a compiled function set and
// returns an error if any cycle (direct or mutual recursion) is found.
//
// Takes root (*CompiledFunction) which is the root function containing
// all compiled functions.
//
// Returns error which wraps errFeatureNotAllowed if a cycle is detected,
// or nil if the call graph is acyclic.
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

// buildCallAdjacency constructs an adjacency list for the call graph,
// excluding closure and method calls.
//
// Takes functions ([]*CompiledFunction) which are the compiled
// functions to analyse.
//
// Returns [][]uint16 where each entry lists the function indices
// called by that function.
func buildCallAdjacency(functions []*CompiledFunction) [][]uint16 {
	adjacency := make([][]uint16, len(functions))
	for i := range functions {
		seen := make(map[uint16]bool)
		for j := range functions[i].callSites {
			cs := &functions[i].callSites[j]
			if cs.isClosure || cs.isMethod {
				continue
			}
			idx := cs.funcIndex
			if int(idx) < len(functions) && !seen[idx] {
				adjacency[i] = append(adjacency[i], idx)
				seen[idx] = true
			}
		}
	}
	return adjacency
}

// callGraphHasCycle performs a DFS-based cycle detection on a directed
// adjacency list.
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
