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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoerceForOperandPassesThroughOnMatch(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	bodyLenBefore := len(c.function.body)
	loc := varLocation{register: 5, kind: registerInt}

	out := c.coerceForOperand(context.Background(), roleRegInt, true, loc)
	require.Equal(t, loc, out, "matching kind should pass through unchanged")
	require.Equal(t, bodyLenBefore, len(c.function.body),
		"no instructions should be emitted on match")
}

func TestCoerceForOperandNonReadDoesNothing(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	bodyLenBefore := len(c.function.body)
	loc := varLocation{register: 3, kind: registerGeneral}

	out := c.coerceForOperand(context.Background(), roleRegInt, false, loc)
	require.Equal(t, loc, out, "non-read operands pass through")
	require.Equal(t, bodyLenBefore, len(c.function.body))
}

func TestCoerceForOperandInsertsUnpackForGeneralToInt(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	src := varLocation{register: 7, kind: registerGeneral}

	out := c.coerceForOperand(context.Background(), roleRegInt, true, src)
	require.Equal(t, registerInt, out.kind, "result should be in int bank")
	require.NotEmpty(t, c.function.body, "an instruction should have been emitted")
	last := c.function.body[len(c.function.body)-1]
	require.True(t, instrIsTier1SubOp(last, subOpMoveGeneralToInt),
		"a general->int move should have been inserted; got %v", last)
}

func TestEmitTypedFunnelInjectsCoercion(t *testing.T) {
	t.Parallel()

	c := newTestCompiler(t)
	dst := varLocation{register: 0, kind: registerInt}
	srcA := varLocation{register: 4, kind: registerGeneral}
	srcB := varLocation{register: 1, kind: registerInt}

	c.emitTyped(context.Background(), opAddInt, dst, srcA, srcB)

	require.GreaterOrEqual(t, len(c.function.body), 2,
		"funnel should emit at least one coerce + the opAddInt")
	final := c.function.body[len(c.function.body)-1]
	require.Equal(t, opAddInt, final.op, "final emitted op should be opAddInt")
	prev := c.function.body[len(c.function.body)-2]
	require.True(t, instrIsTier1SubOp(prev, subOpMoveGeneralToInt),
		"a general->int coercion should precede the opAddInt; got %v", prev)
}

func newTestCompiler(t *testing.T) *compiler {
	t.Helper()
	cf := &CompiledFunction{name: "<test>"}
	return &compiler{
		fileSet:         nil,
		info:            nil,
		function:        cf,
		scopes:          newScopeStack("<test>"),
		funcTable:       make(map[string]uint16),
		rootFunction:    cf,
		globalVariables: make(map[string]globalVariableInfo),
	}
}
