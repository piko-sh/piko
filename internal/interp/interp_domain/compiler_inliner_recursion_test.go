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

//go:build !nounroll

package interp_domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanInlineSelfRecursiveRefusesAlreadyUnrolled(t *testing.T) {
	t.Parallel()
	callee := &CompiledFunction{
		body: []instruction{mk(opAddInt, 0, 0, 0)},
	}
	site := &callSite{recursionUnrolled: true}
	require.Equal(t, inlineRefusalAlreadyUnrolled,
		canInlineSelfRecursive(site, callee, false))
}

func TestCanInlineSelfRecursiveRefusesInLoop(t *testing.T) {
	t.Parallel()
	callee := &CompiledFunction{
		body: []instruction{mk(opAddInt, 0, 0, 0)},
	}
	site := &callSite{}
	require.Equal(t, inlineRefusalSelfInLoop,
		canInlineSelfRecursive(site, callee, true))
}

func TestCanInlineSelfRecursiveRefusesHairyCallee(t *testing.T) {
	t.Parallel()
	body := make([]instruction, selfUnrollBudget+5)
	for i := range body {
		body[i] = mk(opAddInt, 0, 0, 0)
	}
	callee := &CompiledFunction{body: body}
	site := &callSite{}
	require.Equal(t, inlineRefusalSelfHairy,
		canInlineSelfRecursive(site, callee, false))
}

func TestCanInlineSelfRecursiveRefusesTailCallInCallee(t *testing.T) {
	t.Parallel()
	callee := &CompiledFunction{
		body: []instruction{
			mk(opAddInt, 0, 0, 0),
			mk(opTailCall, 0, 0, 0),
		},
	}
	site := &callSite{}
	got := canInlineSelfRecursive(site, callee, false)
	require.Equal(t, inlineRefusalTailCall, got,
		"callee with opTailCall is refused by the shared scanner")
}

func TestCanInlineSelfRecursiveAcceptsEligibleCallee(t *testing.T) {
	t.Parallel()
	callee := &CompiledFunction{
		body: []instruction{
			mk(opAddInt, 0, 0, 0),
			mk(opMulInt, 1, 0, 0),
		},
	}
	site := &callSite{}
	require.Equal(t, inlineEligible,
		canInlineSelfRecursive(site, callee, false))
}

func TestRecursionUnrolledFlagPersistsOnCallSite(t *testing.T) {
	t.Parallel()
	site := callSite{}
	require.False(t, site.recursionUnrolled, "default flag is false")
	site.recursionUnrolled = true
	require.True(t, site.recursionUnrolled, "flag stores through copies")
}
