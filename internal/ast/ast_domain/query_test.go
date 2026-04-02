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

package ast_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryAll(t *testing.T) {
	t.Run("nil AST returns nil", func(t *testing.T) {
		results, diagnostics := QueryAll(nil, "div", "test.pkc")
		assert.Nil(t, results)
		assert.Nil(t, diagnostics)
	})

	t.Run("empty selector returns nil", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{NodeType: NodeElement, TagName: "div"},
			},
		}
		results, diagnostics := QueryAll(tree, "", "test.pkc")
		assert.Nil(t, results)
		assert.Nil(t, diagnostics)
	})

	t.Run("simple tag selector", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{NodeType: NodeElement, TagName: "div"},
				{NodeType: NodeElement, TagName: "span"},
				{NodeType: NodeElement, TagName: "div"},
			},
		}
		results, diagnostics := QueryAll(tree, "div", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 2)
		for _, r := range results {
			assert.Equal(t, "div", r.TagName)
		}
	})

	t.Run("class selector", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "class", Value: "container active"},
					},
				},
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "class", Value: "inactive"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, ".active", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("id selector", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "id", Value: "main"},
					},
				},
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "id", Value: "sidebar"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "#main", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
		assert.Equal(t, "main", results[0].Attributes[0].Value)
	})

	t.Run("descendant combinator", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "span"},
						{
							NodeType: NodeElement,
							TagName:  "p",
							Children: []*TemplateNode{
								{NodeType: NodeElement, TagName: "span"},
							},
						},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "div span", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 2)
	})

	t.Run("child combinator", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "span"},
						{
							NodeType: NodeElement,
							TagName:  "p",
							Children: []*TemplateNode{
								{NodeType: NodeElement, TagName: "span"},
							},
						},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "div > span", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("adjacent sibling combinator", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "h1"},
						{NodeType: NodeElement, TagName: "p"},
						{NodeType: NodeElement, TagName: "p"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "h1 + p", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("general sibling combinator", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "h1"},
						{NodeType: NodeElement, TagName: "p"},
						{NodeType: NodeElement, TagName: "p"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "h1 ~ p", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 2)
	})

	t.Run("multiple selectors", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{NodeType: NodeElement, TagName: "div"},
				{NodeType: NodeElement, TagName: "span"},
				{NodeType: NodeElement, TagName: "p"},
			},
		}
		results, diagnostics := QueryAll(tree, "div, span", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 2)
	})

	t.Run("invalid selector returns diagnostics", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{NodeType: NodeElement, TagName: "div"},
			},
		}
		results, diagnostics := QueryAll(tree, "!!invalid!!", "test.pkc")

		assert.Nil(t, results)
		assert.NotEmpty(t, diagnostics)
	})
}

func TestQuery(t *testing.T) {
	t.Run("returns first match", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{NodeType: NodeElement, TagName: "div"},
				{NodeType: NodeElement, TagName: "div"},
			},
		}
		result, diagnostics := Query(tree, "div")

		assert.Nil(t, diagnostics)
		require.NotNil(t, result)
		assert.Equal(t, "div", result.TagName)
	})

	t.Run("returns nil when no match", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{NodeType: NodeElement, TagName: "div"},
			},
		}
		result, diagnostics := Query(tree, "span")

		assert.Nil(t, diagnostics)
		assert.Nil(t, result)
	})
}

func TestMustQuery(t *testing.T) {
	t.Run("returns first match", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{NodeType: NodeElement, TagName: "div"},
			},
		}
		result := MustQuery(tree, "div")

		require.NotNil(t, result)
		assert.Equal(t, "div", result.TagName)
	})

	t.Run("returns nil when no match", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{NodeType: NodeElement, TagName: "div"},
			},
		}
		result := MustQuery(tree, "span")

		assert.Nil(t, result)
	})
}

func TestTemplateNodeQueryAll(t *testing.T) {
	t.Run("nil node returns nil", func(t *testing.T) {
		var node *TemplateNode
		results, diagnostics := node.QueryAll("div", "test.pkc")

		assert.Nil(t, results)
		assert.Nil(t, diagnostics)
	})

	t.Run("searches from node as root", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			Children: []*TemplateNode{
				{NodeType: NodeElement, TagName: "span"},
				{NodeType: NodeElement, TagName: "span"},
			},
		}
		results, diagnostics := node.QueryAll("span", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 2)
	})
}

func TestTemplateNodeQuery(t *testing.T) {
	t.Run("returns first match", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			Children: []*TemplateNode{
				{NodeType: NodeElement, TagName: "span"},
			},
		}
		result, diagnostics := node.Query("span")

		assert.Nil(t, diagnostics)
		require.NotNil(t, result)
		assert.Equal(t, "span", result.TagName)
	})

	t.Run("returns nil when no match", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
		}
		result, diagnostics := node.Query("span")

		assert.Nil(t, diagnostics)
		assert.Nil(t, result)
	})
}

func TestTemplateNodeMustQuery(t *testing.T) {
	t.Run("returns first match", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			Children: []*TemplateNode{
				{NodeType: NodeElement, TagName: "p"},
			},
		}
		result := node.MustQuery("p")

		require.NotNil(t, result)
		assert.Equal(t, "p", result.TagName)
	})

	t.Run("returns nil when no match", func(t *testing.T) {
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
		}
		result := node.MustQuery("span")

		assert.Nil(t, result)
	})
}

func TestPseudoClassSelectors(t *testing.T) {
	t.Run("first-child", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "p"},
						{NodeType: NodeElement, TagName: "p"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "p:first-child", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("last-child", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "p"},
						{NodeType: NodeElement, TagName: "span"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "span:last-child", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("only-child", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "span"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "span:only-child", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("nth-child with number", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "ul",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "li"},
						{NodeType: NodeElement, TagName: "li"},
						{NodeType: NodeElement, TagName: "li"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "li:nth-child(2)", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("nth-child odd", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "ul",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "li"},
						{NodeType: NodeElement, TagName: "li"},
						{NodeType: NodeElement, TagName: "li"},
						{NodeType: NodeElement, TagName: "li"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "li:nth-child(odd)", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 2)
	})

	t.Run("nth-child even", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "ul",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "li"},
						{NodeType: NodeElement, TagName: "li"},
						{NodeType: NodeElement, TagName: "li"},
						{NodeType: NodeElement, TagName: "li"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "li:nth-child(even)", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 2)
	})

	t.Run("first-of-type", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "span"},
						{NodeType: NodeElement, TagName: "p"},
						{NodeType: NodeElement, TagName: "span"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "span:first-of-type", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("last-of-type", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "span"},
						{NodeType: NodeElement, TagName: "p"},
						{NodeType: NodeElement, TagName: "span"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "span:last-of-type", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("only-of-type", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "span"},
						{NodeType: NodeElement, TagName: "p"},
						{NodeType: NodeElement, TagName: "span"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "p:only-of-type", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("not pseudo-class", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "class", Value: "active"},
					},
				},
				{
					NodeType:   NodeElement,
					TagName:    "div",
					Attributes: []HTMLAttribute{},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "div:not(.active)", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})
}

func TestAttributeSelectors(t *testing.T) {
	t.Run("presence selector", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "input",
					Attributes: []HTMLAttribute{
						{Name: "disabled", Value: ""},
					},
				},
				{
					NodeType:   NodeElement,
					TagName:    "input",
					Attributes: []HTMLAttribute{},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "input[disabled]", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("exact value selector", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "input",
					Attributes: []HTMLAttribute{
						{Name: "type", Value: "text"},
					},
				},
				{
					NodeType: NodeElement,
					TagName:  "input",
					Attributes: []HTMLAttribute{
						{Name: "type", Value: "password"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, `input[type="text"]`, "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("prefix selector", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "a",
					Attributes: []HTMLAttribute{
						{Name: "href", Value: "https://example.com"},
					},
				},
				{
					NodeType: NodeElement,
					TagName:  "a",
					Attributes: []HTMLAttribute{
						{Name: "href", Value: "/local/path"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, `a[href^="https"]`, "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("suffix selector", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "img",
					Attributes: []HTMLAttribute{
						{Name: "src", Value: "image.png"},
					},
				},
				{
					NodeType: NodeElement,
					TagName:  "img",
					Attributes: []HTMLAttribute{
						{Name: "src", Value: "image.jpg"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, `img[src$=".png"]`, "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("contains selector", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "a",
					Attributes: []HTMLAttribute{
						{Name: "href", Value: "https://example.com/path"},
					},
				},
				{
					NodeType: NodeElement,
					TagName:  "a",
					Attributes: []HTMLAttribute{
						{Name: "href", Value: "https://other.com"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, `a[href*="example"]`, "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})
}

func TestFragmentTransparency(t *testing.T) {
	t.Run("fragments are transparent in queries", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{
							NodeType: NodeFragment,
							Children: []*TemplateNode{
								{NodeType: NodeElement, TagName: "span"},
							},
						},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "div > span", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})
}

func TestMatchesNth(t *testing.T) {
	tests := []struct {
		name     string
		formula  string
		index    int
		expected bool
	}{
		{name: "odd index 1", index: 1, formula: "odd", expected: true},
		{name: "odd index 2", index: 2, formula: "odd", expected: false},
		{name: "odd index 3", index: 3, formula: "odd", expected: true},
		{name: "even index 1", index: 1, formula: "even", expected: false},
		{name: "even index 2", index: 2, formula: "even", expected: true},
		{name: "even index 4", index: 4, formula: "even", expected: true},
		{name: "specific number match", index: 3, formula: "3", expected: true},
		{name: "specific number no match", index: 2, formula: "3", expected: false},
		{name: "zero index odd", index: 0, formula: "odd", expected: false},
		{name: "zero index even", index: 0, formula: "even", expected: false},
		{name: "invalid formula", index: 1, formula: "abc", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matchesNth(tc.index, tc.formula)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDeduplicateNodes(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := deduplicateNodes([]*TemplateNode{})
		assert.Empty(t, result)
	})

	t.Run("no duplicates", func(t *testing.T) {
		node1 := &TemplateNode{TagName: "div"}
		node2 := &TemplateNode{TagName: "span"}
		result := deduplicateNodes([]*TemplateNode{node1, node2})

		assert.Len(t, result, 2)
	})

	t.Run("removes duplicates", func(t *testing.T) {
		node1 := &TemplateNode{TagName: "div"}
		node2 := &TemplateNode{TagName: "span"}
		result := deduplicateNodes([]*TemplateNode{node1, node2, node1, node2})

		assert.Len(t, result, 2)
	})

	t.Run("preserves order", func(t *testing.T) {
		node1 := &TemplateNode{TagName: "div"}
		node2 := &TemplateNode{TagName: "span"}
		result := deduplicateNodes([]*TemplateNode{node2, node1, node2})

		assert.Len(t, result, 2)
		assert.Equal(t, "span", result[0].TagName)
		assert.Equal(t, "div", result[1].TagName)
	})
}

func TestMatchesWordInList(t *testing.T) {
	tests := []struct {
		name       string
		nodeVal    string
		targetWord string
		expected   bool
	}{
		{name: "exact match", nodeVal: "active", targetWord: "active", expected: true},
		{name: "word in space-separated list", nodeVal: "btn active primary", targetWord: "active", expected: true},
		{name: "first word", nodeVal: "active primary", targetWord: "active", expected: true},
		{name: "last word", nodeVal: "btn active", targetWord: "active", expected: true},
		{name: "partial match not allowed", nodeVal: "inactive", targetWord: "active", expected: false},
		{name: "empty value", nodeVal: "", targetWord: "active", expected: false},
		{name: "empty target", nodeVal: "active", targetWord: "", expected: false},
		{name: "no match", nodeVal: "btn primary", targetWord: "active", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matchesWordInList(tc.nodeVal, tc.targetWord)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchesDashPrefix(t *testing.T) {
	tests := []struct {
		name     string
		nodeVal  string
		prefix   string
		expected bool
	}{
		{name: "exact match", nodeVal: "en", prefix: "en", expected: true},
		{name: "dash prefix match", nodeVal: "en-GB", prefix: "en", expected: true},
		{name: "dash prefix US", nodeVal: "en-US", prefix: "en", expected: true},
		{name: "no match different language", nodeVal: "fr", prefix: "en", expected: false},
		{name: "partial prefix not allowed", nodeVal: "english", prefix: "en", expected: false},
		{name: "empty value", nodeVal: "", prefix: "en", expected: false},
		{name: "longer prefix without dash", nodeVal: "eng", prefix: "en", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matchesDashPrefix(tc.nodeVal, tc.prefix)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchNthFormula(t *testing.T) {
	tests := []struct {
		name     string
		formula  string
		index    int
		expected bool
	}{
		{name: "2n formula index 2", index: 2, formula: "2n", expected: true},
		{name: "2n formula index 4", index: 4, formula: "2n", expected: true},
		{name: "2n formula index 3", index: 3, formula: "2n", expected: false},
		{name: "2n+1 formula index 1", index: 1, formula: "2n+1", expected: true},
		{name: "2n+1 formula index 3", index: 3, formula: "2n+1", expected: true},
		{name: "2n+1 formula index 2", index: 2, formula: "2n+1", expected: false},
		{name: "n formula index 1", index: 1, formula: "n", expected: true},
		{name: "n formula index 5", index: 5, formula: "n", expected: true},
		{name: "-n+3 formula index 1", index: 1, formula: "-n+3", expected: true},
		{name: "-n+3 formula index 3", index: 3, formula: "-n+3", expected: true},
		{name: "-n+3 formula index 4", index: 4, formula: "-n+3", expected: false},
		{name: "3n+2 formula index 2", index: 2, formula: "3n+2", expected: true},
		{name: "3n+2 formula index 5", index: 5, formula: "3n+2", expected: true},
		{name: "3n+2 formula index 3", index: 3, formula: "3n+2", expected: false},
		{name: "zero index", index: 0, formula: "2n", expected: false},
		{name: "invalid formula", index: 1, formula: "abc", expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matchNthFormula(tc.index, tc.formula)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseNthFormulaCoefficients(t *testing.T) {
	tests := []struct {
		name      string
		formula   string
		expectedA int
		expectedB int
		expectOk  bool
	}{
		{name: "simple n", formula: "n", expectedA: 1, expectedB: 0, expectOk: true},
		{name: "2n", formula: "2n", expectedA: 2, expectedB: 0, expectOk: true},
		{name: "-n", formula: "-n", expectedA: -1, expectedB: 0, expectOk: true},
		{name: "3n+2", formula: "3n+2", expectedA: 3, expectedB: 2, expectOk: true},
		{name: "2n-1", formula: "2n-1", expectedA: 2, expectedB: -1, expectOk: true},
		{name: "-2n+5", formula: "-2n+5", expectedA: -2, expectedB: 5, expectOk: true},
		{name: "+n", formula: "+n", expectedA: 1, expectedB: 0, expectOk: true},
		{name: "invalid a", formula: "abcn", expectedA: 0, expectedB: 0, expectOk: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, b, ok := parseNthFormulaCoefficients(tc.formula)
			assert.Equal(t, tc.expectOk, ok)
			if ok {
				assert.Equal(t, tc.expectedA, a)
				assert.Equal(t, tc.expectedB, b)
			}
		})
	}
}

func TestNthFormulaMatches(t *testing.T) {
	tests := []struct {
		name     string
		index    int
		a        int
		b        int
		expected bool
	}{
		{name: "a=0 exact match", index: 3, a: 0, b: 3, expected: true},
		{name: "a=0 no match", index: 2, a: 0, b: 3, expected: false},
		{name: "a=2 b=0 index 2", index: 2, a: 2, b: 0, expected: true},
		{name: "a=2 b=0 index 4", index: 4, a: 2, b: 0, expected: true},
		{name: "a=2 b=0 index 3", index: 3, a: 2, b: 0, expected: false},
		{name: "a=2 b=1 index 1", index: 1, a: 2, b: 1, expected: true},
		{name: "a=2 b=1 index 3", index: 3, a: 2, b: 1, expected: true},
		{name: "a=-1 b=3 index 1", index: 1, a: -1, b: 3, expected: true},
		{name: "a=-1 b=3 index 3", index: 3, a: -1, b: 3, expected: true},
		{name: "a=-1 b=3 index 4", index: 4, a: -1, b: 3, expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := nthFormulaMatches(tc.index, tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNthOfTypePseudoClasses(t *testing.T) {
	t.Run("nth-of-type", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "p"},
						{NodeType: NodeElement, TagName: "span"},
						{NodeType: NodeElement, TagName: "p"},
						{NodeType: NodeElement, TagName: "span"},
						{NodeType: NodeElement, TagName: "p"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "p:nth-of-type(2)", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("nth-last-child", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "ul",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "li"},
						{NodeType: NodeElement, TagName: "li"},
						{NodeType: NodeElement, TagName: "li"},
						{NodeType: NodeElement, TagName: "li"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "li:nth-last-child(1)", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("nth-last-of-type", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "p"},
						{NodeType: NodeElement, TagName: "span"},
						{NodeType: NodeElement, TagName: "p"},
						{NodeType: NodeElement, TagName: "span"},
						{NodeType: NodeElement, TagName: "p"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, "p:nth-last-of-type(1)", "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})
}

func TestAttributeWordSelector(t *testing.T) {
	t.Run("word selector ~=", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "class", Value: "btn active primary"},
					},
				},
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "class", Value: "inactive"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, `div[class~="active"]`, "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("dash prefix selector |=", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "lang", Value: "en-GB"},
					},
				},
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "lang", Value: "fr"},
					},
				},
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "lang", Value: "en"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, `div[lang|="en"]`, "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 2)
	})
}

func TestPartialAttributeScoping(t *testing.T) {

	t.Run("matches single scope with ~=", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "partial", Value: "partials_card_abc123"},
						{Name: "class", Value: "card"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, `div[partial~="partials_card_abc123"]`, "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1)
	})

	t.Run("matches combined scope (child first, parent second)", func(t *testing.T) {

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "partial", Value: "partials_child_xyz789 partials_parent_abc123"},
						{Name: "class", Value: "card"},
					},
				},
			},
		}

		t.Run("matches child scope", func(t *testing.T) {
			results, diagnostics := QueryAll(tree, `div[partial~="partials_child_xyz789"]`, "test.pkc")
			assert.Nil(t, diagnostics)
			assert.Len(t, results, 1, "Should match the child's scope")
		})

		t.Run("matches parent scope", func(t *testing.T) {
			results, diagnostics := QueryAll(tree, `div[partial~="partials_parent_abc123"]`, "test.pkc")
			assert.Nil(t, diagnostics)
			assert.Len(t, results, 1, "Should match the parent's scope (for :deep CSS)")
		})

		t.Run("does not match partial scope string", func(t *testing.T) {
			results, diagnostics := QueryAll(tree, `div[partial~="partials_child"]`, "test.pkc")
			assert.Nil(t, diagnostics)
			assert.Len(t, results, 0, "Should not match partial scope (word boundary)")
		})
	})

	t.Run("scoped CSS selector with class", func(t *testing.T) {

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "partial", Value: "partials_card_abc123"},
						{Name: "class", Value: "card"},
					},
				},
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "partial", Value: "partials_other_def456"},
						{Name: "class", Value: "card"},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, `.card[partial~="partials_card_abc123"]`, "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1, "Should only match .card with the correct partial scope")
	})

	t.Run("deep selector pattern", func(t *testing.T) {

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "partial", Value: "partials_parent_abc123"},
					},
					Children: []*TemplateNode{
						{
							NodeType: NodeElement,
							TagName:  "div",
							Attributes: []HTMLAttribute{
								{Name: "partial", Value: "partials_child_xyz789"},
								{Name: "class", Value: "card"},
							},
						},
					},
				},
			},
		}
		results, diagnostics := QueryAll(tree, `[partial~="partials_parent_abc123"] .card`, "test.pkc")

		assert.Nil(t, diagnostics)
		assert.Len(t, results, 1, "Deep selector should match child .card through parent scope")
	})
}

func TestInvalidateQueryContext(t *testing.T) {
	t.Parallel()

	tree := &TemplateAST{
		RootNodes: []*TemplateNode{
			{
				NodeType: NodeElement,
				TagName:  "div",
				Attributes: []HTMLAttribute{
					{Name: "class", Value: "original"},
				},
			},
			{
				NodeType: NodeElement,
				TagName:  "span",
			},
		},
	}

	results1, diags1 := QueryAll(tree, "div", "test.pkc")
	assert.Nil(t, diags1)
	assert.Len(t, results1, 1)
	assert.Equal(t, "div", results1[0].TagName)

	tree.RootNodes = append(tree.RootNodes, &TemplateNode{
		NodeType: NodeElement,
		TagName:  "div",
		Attributes: []HTMLAttribute{
			{Name: "class", Value: "new"},
		},
	})

	tree.InvalidateQueryContext()

	results2, diags2 := QueryAll(tree, "div", "test.pkc")
	assert.Nil(t, diags2)
	assert.Len(t, results2, 2,
		"After invalidation and adding a new div, query should find 2 div elements")
}
