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

package pikotest_domain_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pikotest/pikotest_domain"
	"piko.sh/piko/internal/pikotest/pikotest_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

func newTestASTQueryResult(t *testing.T, nodes []*ast_domain.TemplateNode, selector string) *pikotest_domain.ASTQueryResult {
	t.Helper()

	ast := &ast_domain.TemplateAST{
		RootNodes: nodes,
	}

	buildAST := func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*pikotest_dto.RuntimeDiagnostic) {
		return ast, templater_dto.InternalMetadata{}, nil
	}

	tester := pikotest_domain.NewComponentTester(t, buildAST)
	request := pikotest_domain.NewRequest("GET", "/test").Build(context.Background())
	view := tester.Render(request, nil)

	return view.QueryAST(selector)
}

func makeElementNode(tag string, attrs ...ast_domain.HTMLAttribute) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType:   ast_domain.NodeElement,
		TagName:    tag,
		Attributes: attrs,
	}
}

func makeTextNode(text string) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: text,
	}
}

func makeElementWithChildren(tag string, children ...*ast_domain.TemplateNode) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  tag,
		Children: children,
	}
}

func TestASTQueryResult_Nodes(t *testing.T) {
	h1 := makeElementWithChildren("h1", makeTextNode("Hello"))
	result := newTestASTQueryResult(t, []*ast_domain.TemplateNode{h1}, "h1")

	nodes := result.Nodes()
	require.Len(t, nodes, 1)
	assert.Equal(t, "h1", nodes[0].TagName)
}

func TestASTQueryResult_Len(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		makeElementNode("p"),
		makeElementNode("p"),
		makeElementNode("p"),
	}
	result := newTestASTQueryResult(t, nodes, "p")

	assert.Equal(t, 3, result.Len())
}

func TestASTQueryResult_First(t *testing.T) {
	tests := []struct {
		name      string
		selector  string
		expectTag string
		nodes     []*ast_domain.TemplateNode
		expectNil bool
	}{
		{
			name:      "returns first matching node",
			nodes:     []*ast_domain.TemplateNode{makeElementNode("h1"), makeElementNode("h2")},
			selector:  "*",
			expectNil: false,
			expectTag: "h1",
		},
		{
			name:      "returns nil when no nodes match",
			nodes:     []*ast_domain.TemplateNode{makeElementNode("h1")},
			selector:  "h2",
			expectNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := newTestASTQueryResult(t, tc.nodes, tc.selector)
			first := result.First()

			if tc.expectNil {
				assert.Nil(t, first)
			} else {
				require.NotNil(t, first)
				assert.Equal(t, tc.expectTag, first.TagName)
			}
		})
	}
}

func TestASTQueryResult_Last(t *testing.T) {
	tests := []struct {
		name      string
		selector  string
		expectTag string
		nodes     []*ast_domain.TemplateNode
		expectNil bool
	}{
		{
			name:      "returns last matching node",
			nodes:     []*ast_domain.TemplateNode{makeElementNode("h1"), makeElementNode("h2")},
			selector:  "*",
			expectNil: false,
			expectTag: "h2",
		},
		{
			name:      "returns nil when no nodes match",
			nodes:     []*ast_domain.TemplateNode{makeElementNode("h1")},
			selector:  "h2",
			expectNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := newTestASTQueryResult(t, tc.nodes, tc.selector)
			last := result.Last()

			if tc.expectNil {
				assert.Nil(t, last)
			} else {
				require.NotNil(t, last)
				assert.Equal(t, tc.expectTag, last.TagName)
			}
		})
	}
}

func TestASTQueryResult_At(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		makeElementNode("h1"),
		makeElementNode("h2"),
		makeElementNode("h3"),
	}

	tests := []struct {
		name      string
		expectTag string
		index     int
		expectNil bool
	}{
		{name: "valid index 0", index: 0, expectTag: "h1"},
		{name: "valid index 1", index: 1, expectTag: "h2"},
		{name: "valid index 2", index: 2, expectTag: "h3"},
		{name: "negative index", index: -1, expectNil: true},
		{name: "out of bounds", index: 5, expectNil: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := newTestASTQueryResult(t, nodes, "*")
			node := result.At(tc.index)

			if tc.expectNil {
				assert.Nil(t, node)
			} else {
				require.NotNil(t, node)
				assert.Equal(t, tc.expectTag, node.TagName)
			}
		})
	}
}

func TestASTQueryResult_Index(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		makeElementNode("h1"),
		makeElementNode("h2"),
	}

	result := newTestASTQueryResult(t, nodes, "*")
	indexed := result.Index(1)

	require.Equal(t, 1, indexed.Len())
	require.NotNil(t, indexed.First())
	assert.Equal(t, "h2", indexed.First().TagName)
}

func TestASTQueryResult_FirstResult(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		makeElementNode("h1"),
		makeElementNode("h2"),
	}

	result := newTestASTQueryResult(t, nodes, "*")
	first := result.FirstResult()

	require.Equal(t, 1, first.Len())
	require.NotNil(t, first.First())
	assert.Equal(t, "h1", first.First().TagName)
}

func TestASTQueryResult_Exists(t *testing.T) {
	h1 := makeElementNode("h1")
	result := newTestASTQueryResult(t, []*ast_domain.TemplateNode{h1}, "h1")

	chained := result.Exists()
	assert.NotNil(t, chained)
}

func TestASTQueryResult_NotExists(t *testing.T) {
	h1 := makeElementNode("h1")
	result := newTestASTQueryResult(t, []*ast_domain.TemplateNode{h1}, "h2")

	chained := result.NotExists()
	assert.NotNil(t, chained)
}

func TestASTQueryResult_Count(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		makeElementNode("p"),
		makeElementNode("p"),
	}

	result := newTestASTQueryResult(t, nodes, "p")
	chained := result.Count(2)
	assert.NotNil(t, chained)
}

func TestASTQueryResult_MinCount(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		makeElementNode("p"),
		makeElementNode("p"),
		makeElementNode("p"),
	}

	result := newTestASTQueryResult(t, nodes, "p")
	chained := result.MinCount(2)
	assert.NotNil(t, chained)
}

func TestASTQueryResult_MaxCount(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		makeElementNode("p"),
	}

	result := newTestASTQueryResult(t, nodes, "p")
	chained := result.MaxCount(5)
	assert.NotNil(t, chained)
}

func TestASTQueryResult_HasText(t *testing.T) {
	h1 := makeElementWithChildren("h1", makeTextNode("Hello World"))
	result := newTestASTQueryResult(t, []*ast_domain.TemplateNode{h1}, "h1")

	chained := result.HasText("Hello World")
	assert.NotNil(t, chained)
}

func TestASTQueryResult_ContainsText(t *testing.T) {
	h1 := makeElementWithChildren("h1", makeTextNode("Hello World"))
	result := newTestASTQueryResult(t, []*ast_domain.TemplateNode{h1}, "h1")

	chained := result.ContainsText("World")
	assert.NotNil(t, chained)
}

func TestASTQueryResult_HasAttribute(t *testing.T) {
	node := makeElementNode("a", ast_domain.HTMLAttribute{Name: "href", Value: "/home"})
	result := newTestASTQueryResult(t, []*ast_domain.TemplateNode{node}, "a")

	chained := result.HasAttribute("href", "/home")
	assert.NotNil(t, chained)
}

func TestASTQueryResult_HasAttributeContaining(t *testing.T) {
	node := makeElementNode("a", ast_domain.HTMLAttribute{Name: "href", Value: "/users/123/profile"})
	result := newTestASTQueryResult(t, []*ast_domain.TemplateNode{node}, "a")

	chained := result.HasAttributeContaining("href", "123")
	assert.NotNil(t, chained)
}

func TestASTQueryResult_HasAttributePresent(t *testing.T) {
	node := makeElementNode("input", ast_domain.HTMLAttribute{Name: "required", Value: ""})
	result := newTestASTQueryResult(t, []*ast_domain.TemplateNode{node}, "input")

	chained := result.HasAttributePresent("required")
	assert.NotNil(t, chained)
}

func TestASTQueryResult_HasClass(t *testing.T) {
	node := makeElementNode("div", ast_domain.HTMLAttribute{Name: "class", Value: "container active"})
	result := newTestASTQueryResult(t, []*ast_domain.TemplateNode{node}, "div")

	chained := result.HasClass("active")
	assert.NotNil(t, chained)
}

func TestASTQueryResult_HasTag(t *testing.T) {
	node := makeElementNode("section")
	result := newTestASTQueryResult(t, []*ast_domain.TemplateNode{node}, "section")

	chained := result.HasTag("section")
	assert.NotNil(t, chained)
}

func TestASTQueryResult_Each(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		makeElementNode("p"),
		makeElementNode("p"),
	}

	result := newTestASTQueryResult(t, nodes, "p")

	var visited int
	result.Each(func(index int, node *ast_domain.TemplateNode) {
		visited++
		assert.Equal(t, "p", node.TagName)
	})
	assert.Equal(t, 2, visited)
}

func TestASTQueryResult_Filter(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		makeElementNode("div", ast_domain.HTMLAttribute{Name: "class", Value: "active"}),
		makeElementNode("div", ast_domain.HTMLAttribute{Name: "class", Value: "inactive"}),
		makeElementNode("div", ast_domain.HTMLAttribute{Name: "class", Value: "active"}),
	}

	result := newTestASTQueryResult(t, nodes, "div")
	filtered := result.Filter(func(node *ast_domain.TemplateNode) bool {
		return node.HasClass("active")
	})

	assert.Equal(t, 2, filtered.Len())
}

func TestASTQueryResult_Map(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		makeElementWithChildren("h1", makeTextNode("One")),
		makeElementWithChildren("h2", makeTextNode("Two")),
	}

	result := newTestASTQueryResult(t, nodes, "*")
	tags := result.Map(func(node *ast_domain.TemplateNode) any {
		return node.TagName
	})

	require.Len(t, tags, 2)
	assert.Equal(t, "h1", tags[0])
	assert.Equal(t, "h2", tags[1])
}

func TestASTQueryResult_Dump(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		makeElementNode("div", ast_domain.HTMLAttribute{Name: "id", Value: "main"}),
		makeTextNode("Hello"),
	}

	result := newTestASTQueryResult(t, nodes, "*")

	chained := result.Dump()
	assert.NotNil(t, chained)
}

func TestASTQueryResult_Dump_NoNodes(t *testing.T) {
	result := newTestASTQueryResult(t, []*ast_domain.TemplateNode{makeElementNode("h1")}, "h2")

	chained := result.Dump()
	assert.NotNil(t, chained)
}
