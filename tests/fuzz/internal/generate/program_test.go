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

package generate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/fuzzkit/tape"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestProgramUsesLexicalLoopScopesAndRequestBranches(t *testing.T) {
	output := func(id, scope int, value *integer) templateNode {
		return templateNode{kind: "output", id: id, scope: scope, value: value}
	}
	nested := templateNode{kind: "loop", scope: 0, children: []templateNode{{kind: "loop", scope: 1, children: []templateNode{
		output(1, 0, &integer{operation: "-", left: &integer{operation: "id", value: 0}, right: &integer{operation: "index", value: 1}}),
	}}}}
	branch := templateNode{kind: "branch", children: []templateNode{
		output(2, -1, &integer{operation: "number"}),
		output(3, -1, &integer{operation: "bias"}),
		output(4, -1, &integer{operation: "neg", left: &integer{operation: "number"}}),
	}}
	c := casefile.Case{Files: map[string]string{}}
	model := program{trees: []templateNode{nested, branch}, rows: []row{{id: 7, label: "a"}, {id: 10, label: "b"}}, bias: 6}
	require.NoError(t, emitProgram(&c, "page", model))
	source := c.Files["pages/program.pk"]
	for _, required := range []string{"(i0, r0)", "(i1, r1)", "r0.ID - i1", "r0.Label"} {
		require.Contains(t, source, required)
	}
	wanted := [][]string{
		{"n1:7:a", "n1:6:a", "n1:10:b", "n1:9:b", "n2:-2:page"},
		{"n3:6:page"},
		{"n1:7:a", "n1:6:a", "n1:10:b", "n1:9:b", "n4:-3:page"},
	}
	for position, want := range wanted {
		var got []string
		for _, observed := range c.Probes[position].Output {
			require.True(t, observed.HasValue, "missing expected attribute")
			got = append(got, observed.ID+":"+observed.Text+":"+observed.Value)
		}
		assert.Equal(t, want, got, c.Probes[position].Path)
	}
}

func TestProgramRejectsMalformedModels(t *testing.T) {
	testCases := []struct {
		name  string
		model program
	}{
		{name: "unknown operation", model: program{trees: []templateNode{{kind: "output", scope: -1, value: &integer{operation: "%", left: &integer{operation: "number"}, right: &integer{operation: "bias"}}}}}},
		{name: "missing operand", model: program{trees: []templateNode{{kind: "output", scope: -1, value: &integer{operation: "+", left: &integer{operation: "number"}}}}}},
		{name: "unknown production", model: program{trees: []templateNode{{kind: "output", scope: -1, value: &integer{operation: "mystery"}}}}},
		{name: "loop reference outside scope", model: program{trees: []templateNode{{kind: "output", scope: -1, value: &integer{operation: "index", value: 2}}}}},
		{name: "label outside scope", model: program{trees: []templateNode{{kind: "output", scope: 3, value: &integer{operation: "number"}}}}},
		{name: "branch without arms", model: program{trees: []templateNode{{kind: "branch"}}}},
		{name: "loop without body", model: program{trees: []templateNode{{kind: "loop"}}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := casefile.Case{Files: map[string]string{}}
			require.ErrorIs(t, emitProgram(&c, "page", testCase.model), errMalformedProgram)
		})
	}
}

func TestProgramShrinkRetainsAnObservableLeaf(t *testing.T) {
	c := New("pages", 7, 0)
	hasOutput := func(candidate *casefile.Case) bool {
		for _, probe := range candidate.Probes {
			if probe.Path == "/program?input=right" && len(probe.Output) > 0 {
				return true
			}
		}
		return false
	}
	require.True(t, hasOutput(&c), "test case needs an observable right branch")
	reduced := ShrinkProject(&c, hasOutput)
	assert.True(t, hasOutput(&reduced))
	assert.Less(t, len(reduced.Files["pages/program.pk"]), len(c.Files["pages/program.pk"]))
	assert.Empty(t, reduced.Files["pages/arithmetic.pk"])
}

func TestReduceTypedProgramIndependentlyOfArithmetic(t *testing.T) {
	c := New("pages", 7, 0)
	needs := func(candidate *casefile.Case) bool {
		for _, probe := range candidate.Probes {
			if probe.Path != "/typed?mode=nil" {
				continue
			}
			for _, observed := range probe.Output {
				if observed.ID == "lookup" && observed.Text == "fallback" {
					return true
				}
			}
		}
		return false
	}
	reduced := ShrinkProject(&c, needs)
	require.True(t, needs(&reduced))
	assert.Empty(t, reduced.Files["pages/arithmetic.pk"])
	assert.NotContains(t, reduced.Files["pages/typed.pk"], "data-fuzz-output=\"duration\"")
	for _, probe := range reduced.Probes {
		if strings.HasPrefix(probe.Path, "/typed") {
			assert.Equal(t, "/typed?mode=nil", probe.Path)
			assert.Len(t, probe.Output, 1)
		}
	}
}

func TestProgramSizeBoundsAndReplay(t *testing.T) {
	for _, size := range []string{"tiny", "medium", "extended"} {
		limit := limitsFor(size)
		var deepest, roots, rows int
		var visit func([]templateNode, int)
		visit = func(nodes []templateNode, depth int) {
			for _, node := range nodes {
				deepest = max(deepest, depth)
				visit(node.children, depth+1)
			}
		}
		for index := range uint64(96) {
			project := drawProject(tape.New(tape.Mix(9, index)), "pages", 9, index, size)
			visit(project.program.trees, 0)
			roots = max(roots, len(project.program.trees))
			rows = max(rows, len(project.program.rows))
			c := NewSized("pages", 9, index, size)
			replayed := ReplaySized(c.Target, c.Seed, c.Index, c.Data, c.Size)
			require.Equal(t, c.Hash(), replayed.Hash(), "%s %d replay changed", size, index)
		}
		assert.Equal(t, limit.Nesting, deepest, size)
		assert.Equal(t, limit.Roots, roots, size)
		assert.Equal(t, limit.Rows, rows, size)
	}
}

func FuzzExpressionSemantics(f *testing.F) {
	for seed := range uint64(8) {
		stream := tape.New(seed)
		drawInteger(stream, 2, 2)
		f.Add(stream.Recording().Data, int8(seed)-3, int8(2)-int8(seed), int8(seed%5)-1)
	}
	f.Fuzz(func(t *testing.T, choices []byte, x, y, at int8) {
		stream := tape.Replay(&tape.Recording{Data: choices[:min(len(choices), 128)]})
		expression := drawInteger(stream, 2, 2)
		in := inputs{number: int(x), bias: int(y)}
		stack := []scope{{index: 0, row: row{id: int(x) + 1}}, {index: 1, row: row{id: int(y) - 1}}}
		scopeValues := map[string]any{
			"state":  map[string]any{"Number": float64(x), "Bias": float64(y)},
			"values": []float64{float64(x), float64(y)},
			"at":     float64(at),
			"label":  map[string]any{"empty": "", "zero": 0.0},
		}
		for depth, frame := range stack {
			scopeValues[fmt.Sprintf("r%d", depth)] = map[string]any{"ID": float64(frame.row.id)}
			scopeValues[fmt.Sprintf("i%d", depth)] = float64(frame.index)
		}
		evaluated, err := expression.eval(in, stack)
		require.NoError(t, err)
		value := float64(evaluated)
		check := func(source string, want any) {
			t.Helper()
			parser := ast_domain.NewExpressionParser(t.Context(), source, "oracle.pk")
			parsed, diagnostics := parser.ParseExpression(t.Context())
			parser.Release()
			require.Empty(t, diagnostics, "generated %q", source)
			require.NotNil(t, parsed, "generated %q", source)
			require.Equal(t, want, ast_domain.EvaluateExpression(parsed, scopeValues), "%q with x=%d y=%d at=%d", source, x, y, at)
		}
		source := expression.source()
		check(source, value)
		check(source+" + state.Number * state.Bias", value+float64(x)*float64(y))
		check(source+" < state.Number || state.Bias == state.Number && at < 0", value < float64(x) || x == y && at < 0)
		selected := float64(y)
		if value < float64(x) {
			selected = float64(x)
		}
		check("("+source+" < state.Number) ? state.Number : state.Bias", selected)
		element := value
		switch at {
		case 0:
			element = float64(x)
		case 1:
			element = float64(y)
		}
		check("values[at] ?? "+source, element)
		check("label.zero ?? "+source, 0.0)
		check(`label["empty"] ?? "fallback"`, "")
		check("label.missing ?? "+source, value)
	})
}
