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

package generator_helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestSequentialPoolReuse(t *testing.T) {
	t.Parallel()

	pageClasses := []struct {
		staticClass string
		dynamic     string
		expected    string
	}{
		{staticClass: "nav-item", dynamic: "active", expected: "nav-item active"},
		{staticClass: "text-emphasis", dynamic: "bold", expected: "text-emphasis bold"},
		{staticClass: "nav-item", dynamic: "active", expected: "nav-item active"},
		{staticClass: "sidebar", dynamic: "collapsed", expected: "sidebar collapsed"},
		{staticClass: "nav-item", dynamic: "active", expected: "nav-item active"},
		{staticClass: "text-emphasis", dynamic: "bold", expected: "text-emphasis bold"},
		{staticClass: "sidebar", dynamic: "collapsed", expected: "sidebar collapsed"},
		{staticClass: "nav-item", dynamic: "active", expected: "nav-item active"},
	}

	for iteration, tc := range pageClasses {
		t.Run("", func(t *testing.T) {

			arena := ast_domain.GetArena()
			ast := ast_domain.GetTemplateAST()
			ast.SetArena(arena)
			ast.RootNodes = arena.GetRootNodesSlice(1)

			node := arena.GetNode()
			node.NodeType = ast_domain.NodeElement
			node.TagName = "div"
			node.IsPooled = true

			bufferPointer := MergeClassesBytes(tc.staticClass, tc.dynamic)
			require.NotNil(t, bufferPointer, "iteration %d: MergeClassesBytes should return non-nil for %q + %q",
				iteration, tc.staticClass, tc.dynamic)

			dw := arena.GetDirectWriter()
			dw.SetName("class")
			dw.AppendPooledBytes(bufferPointer)
			_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
			node.AttributeWriters = append(node.AttributeWriters, dw)

			ast.RootNodes = append(ast.RootNodes, node)

			require.Greater(t, len(ast.RootNodes), 0, "iteration %d: should have root nodes", iteration)
			require.Greater(t, len(ast.RootNodes[0].AttributeWriters), 0, "iteration %d: should have AttributeWriters", iteration)

			renderDW := ast.RootNodes[0].AttributeWriters[0]
			var output []byte
			output = renderDW.WriteTo(output)
			got := string(output)

			assert.Equal(t, tc.expected, got,
				"iteration %d: expected %q, got %q (corruption from previous iteration?)",
				iteration, tc.expected, got)

			ast_domain.PutTree(ast)
		})
	}
}

func TestSequentialReuseManyIterations(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	classes := []struct {
		staticClass string
		dynamic     string
		expected    string
	}{
		{staticClass: "nav-item", dynamic: "active", expected: "nav-item active"},
		{staticClass: "text-emphasis", dynamic: "bold", expected: "text-emphasis bold"},
		{staticClass: "sidebar", dynamic: "collapsed", expected: "sidebar collapsed"},
	}

	for iteration := range 500 {
		tc := classes[iteration%len(classes)]

		arena := ast_domain.GetArena()
		ast := ast_domain.GetTemplateAST()
		ast.SetArena(arena)
		ast.RootNodes = arena.GetRootNodesSlice(1)

		node := arena.GetNode()
		node.NodeType = ast_domain.NodeElement
		node.TagName = "div"
		node.IsPooled = true

		bufferPointer := MergeClassesBytes(tc.staticClass, tc.dynamic)
		if bufferPointer == nil {
			t.Fatalf("iteration %d: MergeClassesBytes returned nil", iteration)
		}

		dw := arena.GetDirectWriter()
		dw.SetName("class")
		dw.AppendPooledBytes(bufferPointer)
		_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
		node.AttributeWriters = append(node.AttributeWriters, dw)

		ast.RootNodes = append(ast.RootNodes, node)

		if len(ast.RootNodes) > 0 && len(ast.RootNodes[0].AttributeWriters) > 0 {
			renderDW := ast.RootNodes[0].AttributeWriters[0]
			var output []byte
			output = renderDW.WriteTo(output)
			got := string(output)

			if got != tc.expected {
				t.Fatalf("iteration %d: CORRUPTION - expected %q, got %q",
					iteration, tc.expected, got)
			}
		}

		ast_domain.PutTree(ast)
	}
}

func TestLongerClassValueReuse(t *testing.T) {
	t.Parallel()

	testSequence := []struct {
		expected string
		inputs   []string
	}{
		{expected: "very-long-class-name-that-is-quite-lengthy another-long-one", inputs: []string{"very-long-class-name-that-is-quite-lengthy", "another-long-one"}},
		{expected: "short", inputs: []string{"short"}},
		{expected: "a", inputs: []string{"a"}},
		{expected: "very-long-class-name-that-is-quite-lengthy another-long-one", inputs: []string{"very-long-class-name-that-is-quite-lengthy", "another-long-one"}},
		{expected: "short", inputs: []string{"short"}},
	}

	for iteration, tc := range testSequence {

		arena := ast_domain.GetArena()
		ast := ast_domain.GetTemplateAST()
		ast.SetArena(arena)
		ast.RootNodes = arena.GetRootNodesSlice(1)

		node := arena.GetNode()
		node.NodeType = ast_domain.NodeElement
		node.TagName = "div"
		node.IsPooled = true

		bufferPointer := MergeClassesBytes(interfaceSlice(tc.inputs)...)
		if bufferPointer == nil {
			t.Fatalf("iteration %d: MergeClassesBytes returned nil", iteration)
		}

		dw := arena.GetDirectWriter()
		dw.SetName("class")
		dw.AppendPooledBytes(bufferPointer)
		_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
		node.AttributeWriters = append(node.AttributeWriters, dw)

		ast.RootNodes = append(ast.RootNodes, node)

		if len(ast.RootNodes) > 0 && len(ast.RootNodes[0].AttributeWriters) > 0 {
			renderDW := ast.RootNodes[0].AttributeWriters[0]
			var output []byte
			output = renderDW.WriteTo(output)
			got := string(output)

			if got != tc.expected {
				t.Fatalf("iteration %d: CORRUPTION - expected %q (len=%d), got %q (len=%d)",
					iteration, tc.expected, len(tc.expected), got, len(got))
			}
		}

		ast_domain.PutTree(ast)
	}
}

func interfaceSlice(ss []string) []any {
	result := make([]any, len(ss))
	for i, s := range ss {
		result[i] = s
	}
	return result
}
