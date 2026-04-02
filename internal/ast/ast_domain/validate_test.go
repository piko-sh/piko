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
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseForValidation(t *testing.T, source string) *TemplateAST {
	t.Helper()
	tree, err := Parse(context.Background(), source, "test.pk", nil)
	require.NoError(t, err)

	applyTemplateTransformations(context.Background(), tree)

	return tree
}

func assertHasValidationError(t *testing.T, tree *TemplateAST, expectedSeverity Severity, expectedMessage string) {
	t.Helper()
	found := false
	for _, diagnostic := range tree.Diagnostics {
		if diagnostic.Severity == expectedSeverity && strings.Contains(diagnostic.Message, expectedMessage) {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find a diagnostic with severity '%s' containing message: '%s'\nFull Diagnostics:\n%s", expectedSeverity, expectedMessage, formatDiagsForTest(tree.Diagnostics))
}

func TestValidateAST_Adjacency(t *testing.T) {
	t.Parallel()

	t.Run("should pass for correct p-if -> p-else-if -> p-else chain", func(t *testing.T) {
		t.Parallel()

		source := `
			<div p-if="cond1">A</div>
			<div p-else-if="cond2">B</div>
			<div p-else>C</div>
		`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assert.Empty(t, tree.Diagnostics)
	})

	t.Run("should fail for p-else without preceding element", func(t *testing.T) {
		t.Parallel()

		source := `<div p-else>Standalone else</div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Error, "'p-else' directive must immediately follow an element with 'p-if' or 'p-else-if'")
	})

	t.Run("should fail for p-else-if separated by another element", func(t *testing.T) {
		t.Parallel()

		source := `
			<div p-if="cond1">A</div>
			<h1>Separator</h1>
			<div p-else-if="cond2">B</div>
		`
		tree := parseForValidation(t, source)
		ValidateAST(tree)

		assertHasValidationError(t, tree, Error, "'p-else-if' directive must immediately follow an element with 'p-if' or 'p-else-if'")
	})

	t.Run("should PASS for p-else separated by a comment", func(t *testing.T) {
		t.Parallel()

		source := `
			<div p-if="cond1">A</div>
			<!-- This comment should be ignored -->
			<div p-else>B</div>
		`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assert.Empty(t, tree.Diagnostics, "A comment should not break the p-if/p-else chain")
	})

	t.Run("should FAIL for p-else separated by meaningful text", func(t *testing.T) {
		t.Parallel()

		source := `
			<div p-if="cond1">A</div>
			Some meaningful text here.
			<div p-else>B</div>
		`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Error, "'p-else' directive must immediately follow an element with 'p-if' or 'p-else-if'")
	})

	t.Run("should PASS for p-else-if separated by whitespace-only text node", func(t *testing.T) {
		t.Parallel()

		source := `
			<div p-if="cond1">A</div>

			<div p-else-if="cond2">B</div>
		`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assert.Empty(t, tree.Diagnostics, "A whitespace-only text node should not break the p-if/p-else-if chain")
	})
}

func TestValidateAST_AttributeConflicts(t *testing.T) {
	t.Parallel()

	t.Run("should NOT warn when static class is used with p-class (they merge)", func(t *testing.T) {
		t.Parallel()

		source := `<div class="static" p-class="dynamic"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assert.Empty(t, tree.Diagnostics, "p-class merges with static class, no warning expected")
	})

	t.Run("should NOT warn when static style is used with p-style (they merge)", func(t *testing.T) {
		t.Parallel()

		source := `<div style="color: red;" p-style="dynamicStyles"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assert.Empty(t, tree.Diagnostics, "p-style merges with static style, no warning expected")
	})

	t.Run("should warn when static attribute conflicts with p-bind", func(t *testing.T) {
		t.Parallel()

		source := `<a href="/static" p-bind:href="dynamicUrl"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Warning, "attribute is defined statically but also targeted by a dynamic 'p-bind:href' binding")
	})

	t.Run("should warn when static attribute conflicts with shorthand binding", func(t *testing.T) {
		t.Parallel()

		source := `<input disabled :disabled="isReadonly">`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Warning, "attribute is defined statically but also targeted by a dynamic ':disabled' binding")
	})

	t.Run("should not warn when there are no conflicts", func(t *testing.T) {
		t.Parallel()

		source := `<div class="static" :title="dynamicTitle"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assert.Empty(t, tree.Diagnostics)
	})
}

func TestValidateAST_ContentDirectives(t *testing.T) {
	t.Parallel()

	t.Run("should error when p-text and p-html are on the same element", func(t *testing.T) {
		t.Parallel()

		source := `<div p-text="message" p-html="rawHtml"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Error, "Cannot use both 'p-text' and 'p-html' on the same element")
	})

	t.Run("should warn when p-text overwrites child elements", func(t *testing.T) {
		t.Parallel()

		source := `<div p-text="message"><p>This will be overwritten</p></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Warning, "contains child nodes that will be overwritten by the 'p-text' directive")
	})

	t.Run("should warn when p-html overwrites child text with interpolation", func(t *testing.T) {
		t.Parallel()

		source := `<div p-html="rawHtml">This text and {{ interpolation }} will be gone</div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Warning, "contains child nodes that will be overwritten by the 'p-html' directive")
	})

	t.Run("should not warn when p-text has only whitespace or comment children", func(t *testing.T) {
		t.Parallel()

		source := `<div p-text="message"> <!-- a comment --> </div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assert.Empty(t, tree.Diagnostics)
	})
}

func TestValidateAST_RedundantConditionals(t *testing.T) {
	t.Parallel()

	t.Run("should error when p-else and p-else-if are on the same element", func(t *testing.T) {
		t.Parallel()

		source := `
			<div p-if="cond1">A</div>
			<div p-else-if="cond2" p-else>B</div>
		`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Error, "cannot have both 'p-else' and 'p-else-if' directives")
	})
}

func TestValidateAST_DirectiveUsage(t *testing.T) {
	t.Parallel()

	t.Run("should error when p-model is used on a non-form element", func(t *testing.T) {
		t.Parallel()

		source := `<div p-model="form.name"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Error, "'p-model' can only be used on <input>, <textarea>, and <select> elements, not on <div>")
	})

	t.Run("should pass when p-model is used on a valid element", func(t *testing.T) {
		t.Parallel()

		sources := []string{
			`<input p-model="form.name">`,
			`<textarea p-model="form.bio"></textarea>`,
			`<select p-model="form.selection"></select>`,
		}
		for i, source := range sources {
			t.Run(fmt.Sprintf("valid element %d", i), func(t *testing.T) {
				t.Parallel()

				tree := parseForValidation(t, source)
				ValidateAST(tree)
				assert.Empty(t, tree.Diagnostics)
			})
		}
	})
}

func TestValidateAST_BestPractices(t *testing.T) {
	t.Parallel()

	t.Run("should warn when p-for is missing a p-key", func(t *testing.T) {
		t.Parallel()

		source := `<div p-for="item in items"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Warning, "should have a unique 'p-key' binding for efficient updates")
	})

	t.Run("should not warn when p-for has a p-key", func(t *testing.T) {
		t.Parallel()

		source := `<div p-for="item in items" p-key="item.id"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assert.Empty(t, tree.Diagnostics)
	})
}

func TestValidateAST_DirectivePrecedence(t *testing.T) {
	t.Parallel()

	t.Run("should warn when p-bind directive appears before p-for", func(t *testing.T) {
		t.Parallel()

		source := `<div p-bind:class="className" p-for="item in items" p-key="item.id"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Warning, "p-bind:class")
		assertHasValidationError(t, tree, Warning, "written before `p-for`")
	})

	t.Run("should warn when multiple p-bind directives appear before p-for", func(t *testing.T) {
		t.Parallel()

		source := `<div p-bind:class="className" p-bind:id="dynId" p-for="item in items" p-key="item.id"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)

		var precedenceWarnings int
		for _, diagnostic := range tree.Diagnostics {
			if strings.Contains(diagnostic.Message, "p-bind:") && strings.Contains(diagnostic.Message, "before `p-for`") {
				precedenceWarnings++
			}
		}
		assert.GreaterOrEqual(t, precedenceWarnings, 2)
	})

	t.Run("should not warn when p-bind directive appears after p-for", func(t *testing.T) {
		t.Parallel()

		source := `<div p-for="item in items" p-bind:class="className" p-key="item.id"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)

		for _, diagnostic := range tree.Diagnostics {
			assert.NotContains(t, diagnostic.Message, "written before `p-for`")
		}
	})

	t.Run("should warn when p-on event directive appears before p-for", func(t *testing.T) {
		t.Parallel()

		source := `<button p-on:click="handleClick()" p-for="item in items" p-key="item.id"></button>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Warning, "p-on:click")
		assertHasValidationError(t, tree, Warning, "written before `p-for`")
	})

	t.Run("should warn when p-event custom event directive appears before p-for", func(t *testing.T) {
		t.Parallel()

		source := `<my-component p-event:submit="handleSubmit()" p-for="item in items" p-key="item.id"></my-component>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Warning, "p-event:submit")
		assertHasValidationError(t, tree, Warning, "written before `p-for`")
	})

	t.Run("should not warn when p-on directive appears after p-for", func(t *testing.T) {
		t.Parallel()

		source := `<button p-for="item in items" p-on:click="handleClick()" p-key="item.id"></button>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)

		for _, diagnostic := range tree.Diagnostics {
			if strings.Contains(diagnostic.Message, "p-on:") {
				assert.NotContains(t, diagnostic.Message, "written before `p-for`")
			}
		}
	})

	t.Run("should warn when dynamic attribute appears before p-for", func(t *testing.T) {
		t.Parallel()

		source := `<div :class="className" p-for="item in items" p-key="item.id"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Warning, ":class")
		assertHasValidationError(t, tree, Warning, "written before `p-for`")
	})

	t.Run("should not warn when dynamic attribute appears after p-for", func(t *testing.T) {
		t.Parallel()

		source := `<div p-for="item in items" :class="className" p-key="item.id"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)

		for _, diagnostic := range tree.Diagnostics {
			if strings.Contains(diagnostic.Message, ":class") {
				assert.NotContains(t, diagnostic.Message, "written before `p-for`")
			}
		}
	})

	t.Run("should warn when standard directive appears before p-for", func(t *testing.T) {
		t.Parallel()

		source := `<div p-if="cond" p-for="item in items" p-key="item.id"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)
		assertHasValidationError(t, tree, Warning, "p-if")
		assertHasValidationError(t, tree, Warning, "written before `p-for`")
	})

	t.Run("should not warn when there is no p-for directive", func(t *testing.T) {
		t.Parallel()

		source := `<div p-bind:class="className" p-on:click="handleClick()"></div>`
		tree := parseForValidation(t, source)
		ValidateAST(tree)

		for _, diagnostic := range tree.Diagnostics {
			assert.NotContains(t, diagnostic.Message, "written before `p-for`")
		}
	})
}

func TestHoistDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("nil tree is handled gracefully", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			HoistDiagnostics(nil)
		})
	})

	t.Run("tree without source path is handled gracefully", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					TagName:     "div",
					Diagnostics: []*Diagnostic{{Message: "test"}},
				},
			},
		}
		HoistDiagnostics(tree)

		assert.Len(t, tree.RootNodes[0].Diagnostics, 1)
	})

	t.Run("hoists node diagnostics to tree level", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			SourcePath: new("test.html"),
			RootNodes: []*TemplateNode{
				{
					TagName: "div",
					Diagnostics: []*Diagnostic{
						{Message: "error 1", Severity: Error},
						{Message: "warning 1", Severity: Warning},
					},
				},
			},
		}
		HoistDiagnostics(tree)

		assert.Len(t, tree.Diagnostics, 2)
		assert.Empty(t, tree.RootNodes[0].Diagnostics)
		assert.Equal(t, "error 1", tree.Diagnostics[0].Message)
		assert.Equal(t, "warning 1", tree.Diagnostics[1].Message)
	})

	t.Run("sets source path on diagnostics without one", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			SourcePath: new("test.html"),
			RootNodes: []*TemplateNode{
				{
					TagName: "div",
					Diagnostics: []*Diagnostic{
						{Message: "error without path", SourcePath: ""},
						{Message: "error with path", SourcePath: "other.html"},
					},
				},
			},
		}
		HoistDiagnostics(tree)

		assert.Equal(t, "test.html", tree.Diagnostics[0].SourcePath)
		assert.Equal(t, "other.html", tree.Diagnostics[1].SourcePath)
	})

	t.Run("hoists diagnostics from nested nodes", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			SourcePath: new("test.html"),
			RootNodes: []*TemplateNode{
				{
					TagName: "div",
					Diagnostics: []*Diagnostic{
						{Message: "parent error"},
					},
					Children: []*TemplateNode{
						{
							TagName: "span",
							Diagnostics: []*Diagnostic{
								{Message: "child error"},
							},
							Children: []*TemplateNode{
								{
									TagName: "p",
									Diagnostics: []*Diagnostic{
										{Message: "grandchild error"},
									},
								},
							},
						},
					},
				},
			},
		}
		HoistDiagnostics(tree)

		assert.Len(t, tree.Diagnostics, 3)
		assert.Empty(t, tree.RootNodes[0].Diagnostics)
		assert.Empty(t, tree.RootNodes[0].Children[0].Diagnostics)
		assert.Empty(t, tree.RootNodes[0].Children[0].Children[0].Diagnostics)
	})

	t.Run("handles nodes without diagnostics", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			SourcePath: new("test.html"),
			RootNodes: []*TemplateNode{
				{TagName: "div"},
				{TagName: "span", Diagnostics: []*Diagnostic{{Message: "has error"}}},
				{TagName: "p"},
			},
		}
		HoistDiagnostics(tree)

		assert.Len(t, tree.Diagnostics, 1)
		assert.Equal(t, "has error", tree.Diagnostics[0].Message)
	})
}

func TestValidateHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("isWhitespaceOnlyText identifies whitespace-only text", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			node     *TemplateNode
			name     string
			expected bool
		}{
			{
				name:     "non-text node returns false",
				node:     &TemplateNode{NodeType: NodeElement},
				expected: false,
			},
			{
				name:     "text node with content returns false",
				node:     &TemplateNode{NodeType: NodeText, TextContent: "hello"},
				expected: false,
			},
			{
				name:     "text node with whitespace only returns true",
				node:     &TemplateNode{NodeType: NodeText, TextContent: "   "},
				expected: true,
			},
			{
				name:     "text node with newlines and tabs returns true",
				node:     &TemplateNode{NodeType: NodeText, TextContent: "\n\t\r"},
				expected: true,
			},
			{
				name: "text node with RichText non-literal part returns false",
				node: &TemplateNode{
					NodeType:    NodeText,
					TextContent: "",
					RichText: []TextPart{
						{IsLiteral: false, Expression: &Identifier{Name: "test"}},
					},
				},
				expected: false,
			},
			{
				name: "text node with RichText whitespace literal returns true",
				node: &TemplateNode{
					NodeType:    NodeText,
					TextContent: "",
					RichText: []TextPart{
						{IsLiteral: true, Literal: "   "},
					},
				},
				expected: true,
			},
			{
				name: "text node with RichText non-whitespace literal returns false",
				node: &TemplateNode{
					NodeType:    NodeText,
					TextContent: "",
					RichText: []TextPart{
						{IsLiteral: true, Literal: "text"},
					},
				},
				expected: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				result := isWhitespaceOnlyText(tc.node)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("hasMeaningfulContent identifies meaningful content", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			node     *TemplateNode
			name     string
			expected bool
		}{
			{
				name:     "node without children returns false",
				node:     &TemplateNode{TagName: "div"},
				expected: false,
			},
			{
				name: "node with element child returns true",
				node: &TemplateNode{
					TagName:  "div",
					Children: []*TemplateNode{{NodeType: NodeElement, TagName: "span"}},
				},
				expected: true,
			},
			{
				name: "node with non-whitespace text child returns true",
				node: &TemplateNode{
					TagName:  "div",
					Children: []*TemplateNode{{NodeType: NodeText, TextContent: "hello"}},
				},
				expected: true,
			},
			{
				name: "node with whitespace-only text child returns false",
				node: &TemplateNode{
					TagName:  "div",
					Children: []*TemplateNode{{NodeType: NodeText, TextContent: "   "}},
				},
				expected: false,
			},
			{
				name: "node with comment child returns false",
				node: &TemplateNode{
					TagName:  "div",
					Children: []*TemplateNode{{NodeType: NodeComment, TextContent: "comment"}},
				},
				expected: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				result := hasMeaningfulContent(tc.node)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("isModelableElement identifies modelable elements", func(t *testing.T) {
		t.Parallel()

		modelable := []string{"input", "textarea", "select"}
		notModelable := []string{"div", "span", "button", "form", "p"}

		for _, tag := range modelable {
			assert.True(t, isModelableElement(tag), "%s should be modelable", tag)
		}
		for _, tag := range notModelable {
			assert.False(t, isModelableElement(tag), "%s should not be modelable", tag)
		}
	})

	t.Run("isValidConditionalPredecessor checks p-if/p-else-if", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			node     *TemplateNode
			name     string
			expected bool
		}{
			{
				name:     "nil node returns false",
				node:     nil,
				expected: false,
			},
			{
				name:     "node with p-if returns true",
				node:     &TemplateNode{DirIf: &Directive{}},
				expected: true,
			},
			{
				name:     "node with p-else-if returns true",
				node:     &TemplateNode{DirElseIf: &Directive{}},
				expected: true,
			},
			{
				name:     "node without p-if or p-else-if returns false",
				node:     &TemplateNode{TagName: "div"},
				expected: false,
			},
			{
				name:     "node with p-else only returns false",
				node:     &TemplateNode{DirElse: &Directive{}},
				expected: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				result := isValidConditionalPredecessor(tc.node)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("getNodeSourcePath returns correct path", func(t *testing.T) {
		t.Parallel()

		t.Run("returns node's original source path when available", func(t *testing.T) {
			t.Parallel()

			node := &TemplateNode{
				GoAnnotations: &GoGeneratorAnnotation{
					OriginalSourcePath: new("component.html"),
				},
			}
			tree := &TemplateAST{SourcePath: new("main.html")}

			result := getNodeSourcePath(node, tree)
			assert.Equal(t, "component.html", result)
		})

		t.Run("returns tree's source path when node has no GoAnnotations", func(t *testing.T) {
			t.Parallel()

			node := &TemplateNode{TagName: "div"}
			tree := &TemplateAST{SourcePath: new("main.html")}

			result := getNodeSourcePath(node, tree)
			assert.Equal(t, "main.html", result)
		})

		t.Run("returns empty string when neither node nor tree have paths", func(t *testing.T) {
			t.Parallel()

			node := &TemplateNode{TagName: "div"}
			tree := &TemplateAST{}

			result := getNodeSourcePath(node, tree)
			assert.Empty(t, result)
		})

		t.Run("returns tree path when node GoAnnotations has nil OriginalSourcePath", func(t *testing.T) {
			t.Parallel()

			node := &TemplateNode{
				GoAnnotations: &GoGeneratorAnnotation{
					OriginalSourcePath: nil,
				},
			}
			tree := &TemplateAST{SourcePath: new("main.html")}

			result := getNodeSourcePath(node, tree)
			assert.Equal(t, "main.html", result)
		})
	})
}
