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

package annotator_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
)

func TestIsExternalURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "http URL is external",
			url:      "http://example.com/image.png",
			expected: true,
		},
		{
			name:     "https URL is external",
			url:      "https://example.com/image.png",
			expected: true,
		},
		{
			name:     "protocol-relative URL is external",
			url:      "//cdn.example.com/image.png",
			expected: true,
		},
		{
			name:     "data URL is external",
			url:      "data:image/png;base64,abc123",
			expected: true,
		},
		{
			name:     "relative path is not external",
			url:      "./images/logo.png",
			expected: false,
		},
		{
			name:     "absolute path is not external",
			url:      "/assets/images/logo.png",
			expected: false,
		},
		{
			name:     "module path is not external",
			url:      "@/assets/images/logo.png",
			expected: false,
		},
		{
			name:     "empty string is not external",
			url:      "",
			expected: false,
		},
		{
			name:     "http in middle of path is not external",
			url:      "/assets/http-icons/logo.png",
			expected: false,
		},
		{
			name:     "ftp is not external",
			url:      "ftp://example.com/file.png",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isExternalURL(tc.url)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsStaticKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		key      ast_domain.Expression
		name     string
		expected bool
	}{
		{
			name:     "string literal is static",
			key:      &ast_domain.StringLiteral{Value: "my-key"},
			expected: true,
		},
		{
			name:     "identifier is not static",
			key:      &ast_domain.Identifier{Name: "dynamicKey"},
			expected: false,
		},
		{
			name:     "member expression is not static",
			key:      &ast_domain.MemberExpression{Base: &ast_domain.Identifier{Name: "item"}, Property: &ast_domain.Identifier{Name: "id"}},
			expected: false,
		},
		{
			name:     "integer literal is not static",
			key:      &ast_domain.IntegerLiteral{Value: 42},
			expected: false,
		},
		{
			name:     "template literal is not static",
			key:      &ast_domain.TemplateLiteral{Parts: []ast_domain.TemplateLiteralPart{}},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isStaticKey(tc.key)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasDynamicKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "node with no key has no dynamic key",
			node:     &ast_domain.TemplateNode{Key: nil},
			expected: false,
		},
		{
			name: "node with static string key has no dynamic key",
			node: &ast_domain.TemplateNode{
				Key: &ast_domain.StringLiteral{Value: "static-key"},
			},
			expected: false,
		},
		{
			name: "node with identifier key has dynamic key",
			node: &ast_domain.TemplateNode{
				Key: &ast_domain.Identifier{Name: "dynamicKey"},
			},
			expected: true,
		},
		{
			name: "node with member expression key has dynamic key",
			node: &ast_domain.TemplateNode{
				Key: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "item"},
					Property: &ast_domain.Identifier{Name: "id"},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := hasDynamicKey(tc.node)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasStructuralOrPresenceDirectives(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "node with no directives",
			node:     &ast_domain.TemplateNode{},
			expected: false,
		},
		{
			name: "node with DirFor",
			node: &ast_domain.TemplateNode{
				DirFor: &ast_domain.Directive{RawExpression: "item in items"},
			},
			expected: true,
		},
		{
			name: "node with DirIf",
			node: &ast_domain.TemplateNode{
				DirIf: &ast_domain.Directive{RawExpression: "showItem"},
			},
			expected: true,
		},
		{
			name: "node with DirElseIf",
			node: &ast_domain.TemplateNode{
				DirElseIf: &ast_domain.Directive{RawExpression: "altCondition"},
			},
			expected: true,
		},
		{
			name: "node with DirElse",
			node: &ast_domain.TemplateNode{
				DirElse: &ast_domain.Directive{RawExpression: ""},
			},
			expected: true,
		},
		{
			name: "node with DirText has no structural directives",
			node: &ast_domain.TemplateNode{
				DirText: &ast_domain.Directive{RawExpression: "message"},
			},
			expected: false,
		},
		{
			name: "node with DirShow has no structural directives",
			node: &ast_domain.TemplateNode{
				DirShow: &ast_domain.Directive{RawExpression: "isVisible"},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := hasStructuralOrPresenceDirectives(tc.node)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasDynamicTextContent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name: "text node with rich text has dynamic content",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
				RichText: []ast_domain.TextPart{
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "value"}},
				},
			},
			expected: true,
		},
		{
			name: "text node without rich text has no dynamic content",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
				RichText: []ast_domain.TextPart{},
			},
			expected: false,
		},
		{
			name: "element node with rich text has no dynamic text content",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				RichText: []ast_domain.TextPart{
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "value"}},
				},
			},
			expected: false,
		},
		{
			name: "text node with nil rich text has no dynamic content",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
				RichText: nil,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := hasDynamicTextContent(tc.node)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasRenderingDirectives(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "node with no directives",
			node:     &ast_domain.TemplateNode{},
			expected: false,
		},
		{
			name: "node with DirText",
			node: &ast_domain.TemplateNode{
				DirText: &ast_domain.Directive{RawExpression: "message"},
			},
			expected: true,
		},
		{
			name: "node with DirHTML",
			node: &ast_domain.TemplateNode{
				DirHTML: &ast_domain.Directive{RawExpression: "htmlContent"},
			},
			expected: true,
		},
		{
			name: "node with DirClass",
			node: &ast_domain.TemplateNode{
				DirClass: &ast_domain.Directive{RawExpression: "{'active': isActive}"},
			},
			expected: true,
		},
		{
			name: "node with DirStyle",
			node: &ast_domain.TemplateNode{
				DirStyle: &ast_domain.Directive{RawExpression: "styleObj"},
			},
			expected: true,
		},
		{
			name: "node with DirShow",
			node: &ast_domain.TemplateNode{
				DirShow: &ast_domain.Directive{RawExpression: "isVisible"},
			},
			expected: true,
		},
		{
			name: "node with DirFor has no rendering directives",
			node: &ast_domain.TemplateNode{
				DirFor: &ast_domain.Directive{RawExpression: "item in items"},
			},
			expected: false,
		},
		{
			name: "node with DirIf has no rendering directives",
			node: &ast_domain.TemplateNode{
				DirIf: &ast_domain.Directive{RawExpression: "condition"},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := hasRenderingDirectives(tc.node)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasNonStaticEvents(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		events   map[string][]ast_domain.Directive
		name     string
		expected bool
	}{
		{
			name:     "nil events map",
			events:   nil,
			expected: false,
		},
		{
			name:     "empty events map",
			events:   map[string][]ast_domain.Directive{},
			expected: false,
		},
		{
			name: "all static events",
			events: map[string][]ast_domain.Directive{
				"click": {
					{RawExpression: "handleClick", IsStaticEvent: true},
				},
				"submit": {
					{RawExpression: "handleSubmit", IsStaticEvent: true},
				},
			},
			expected: false,
		},
		{
			name: "one non-static event",
			events: map[string][]ast_domain.Directive{
				"click": {
					{RawExpression: "handleClick(item)", IsStaticEvent: false},
				},
			},
			expected: true,
		},
		{
			name: "mixed static and non-static events",
			events: map[string][]ast_domain.Directive{
				"click": {
					{RawExpression: "handleClick", IsStaticEvent: true},
				},
				"mouseover": {
					{RawExpression: "highlight(index)", IsStaticEvent: false},
				},
			},
			expected: true,
		},
		{
			name: "multiple directives on same event with one non-static",
			events: map[string][]ast_domain.Directive{
				"click": {
					{RawExpression: "handleClick", IsStaticEvent: true},
					{RawExpression: "trackClick(id)", IsStaticEvent: false},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := hasNonStaticEvents(tc.events)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasDynamicBindings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "node with no bindings",
			node:     &ast_domain.TemplateNode{},
			expected: false,
		},
		{
			name: "node with dynamic attributes",
			node: &ast_domain.TemplateNode{
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "class", Expression: &ast_domain.Identifier{Name: "className"}},
				},
			},
			expected: true,
		},
		{
			name: "node with non-static OnEvents",
			node: &ast_domain.TemplateNode{
				OnEvents: map[string][]ast_domain.Directive{
					"click": {
						{RawExpression: "handleClick(item)", IsStaticEvent: false},
					},
				},
			},
			expected: true,
		},
		{
			name: "node with static OnEvents",
			node: &ast_domain.TemplateNode{
				OnEvents: map[string][]ast_domain.Directive{
					"click": {
						{RawExpression: "handleClick", IsStaticEvent: true},
					},
				},
			},
			expected: false,
		},
		{
			name: "node with non-static CustomEvents",
			node: &ast_domain.TemplateNode{
				CustomEvents: map[string][]ast_domain.Directive{
					"itemSelected": {
						{RawExpression: "onSelect(item)", IsStaticEvent: false},
					},
				},
			},
			expected: true,
		},
		{
			name: "node with static CustomEvents",
			node: &ast_domain.TemplateNode{
				CustomEvents: map[string][]ast_domain.Directive{
					"itemSelected": {
						{RawExpression: "onSelect", IsStaticEvent: true},
					},
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := hasDynamicBindings(tc.node)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNodeHasPartialInvocation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "node with nil GoAnnotations",
			node:     &ast_domain.TemplateNode{GoAnnotations: nil},
			expected: false,
		},
		{
			name: "node with GoAnnotations but nil PartialInfo",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{PartialInfo: nil},
			},
			expected: false,
		},
		{
			name: "node with PartialInfo",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey: "abc123",
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := nodeHasPartialInvocation(tc.node)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsRuntimeProcessedElement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name: "text node is not runtime processed",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
				TagName:  "",
			},
			expected: false,
		},
		{
			name: "comment node is not runtime processed",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeComment,
				TagName:  "",
			},
			expected: false,
		},
		{
			name: "piko:svg element is runtime processed",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:svg",
			},
			expected: true,
		},
		{
			name: "piko:a element is runtime processed",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:a",
			},
			expected: true,
		},
		{
			name: "regular div element is not runtime processed",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
			},
			expected: false,
		},
		{
			name: "element with partial invocation is runtime processed",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey: "abc123",
					},
				},
			},
			expected: true,
		},
		{
			name: "fragment with partial invocation is runtime processed",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeFragment,
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey: "abc123",
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isRuntimeProcessedElement(tc.node)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasDynamicFeatures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "plain element with no features",
			node:     &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "div"},
			expected: false,
		},
		{
			name: "element with dynamic key",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Key:      &ast_domain.Identifier{Name: "itemId"},
			},
			expected: true,
		},
		{
			name: "runtime processed element",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:svg",
			},
			expected: true,
		},
		{
			name: "text node with dynamic content",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
				RichText: []ast_domain.TextPart{{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "value"}}},
			},
			expected: true,
		},
		{
			name: "element with rendering directive",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "span",
				DirText:  &ast_domain.Directive{RawExpression: "message"},
			},
			expected: true,
		},
		{
			name: "element with dynamic bindings",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "button",
				DynamicAttributes: []ast_domain.DynamicAttribute{{Name: "disabled"}},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := hasDynamicFeatures(tc.node)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAnalyseNodeForStaticity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node                       *ast_domain.TemplateNode
		name                       string
		expectedStructurallyStatic bool
		expectedSemanticallyStatic bool
	}{
		{
			name:                       "plain div is fully static",
			node:                       &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "div"},
			expectedStructurallyStatic: true,
			expectedSemanticallyStatic: true,
		},
		{
			name: "element with dynamic attribute is not structurally static",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{{Name: "class"}},
			},
			expectedStructurallyStatic: false,
			expectedSemanticallyStatic: false,
		},
		{
			name: "element with p-for is structurally static but not semantically static",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DirFor:   &ast_domain.Directive{RawExpression: "item in items"},
			},
			expectedStructurallyStatic: true,
			expectedSemanticallyStatic: false,
		},
		{
			name: "element with p-if is structurally static but not semantically static",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DirIf:    &ast_domain.Directive{RawExpression: "condition"},
			},
			expectedStructurallyStatic: true,
			expectedSemanticallyStatic: false,
		},
		{
			name: "static element with non-static child is not fully static",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:      ast_domain.NodeElement,
						TagName:       "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: false},
					},
				},
			},
			expectedStructurallyStatic: true,
			expectedSemanticallyStatic: false,
		},
		{
			name: "static element with static children is fully static",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:      ast_domain.NodeElement,
						TagName:       "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: true},
					},
				},
			},
			expectedStructurallyStatic: true,
			expectedSemanticallyStatic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			analyseNodeForStaticity(tc.node)

			assert.NotNil(t, tc.node.GoAnnotations)
			assert.Equal(t, tc.expectedStructurallyStatic, tc.node.GoAnnotations.IsStructurallyStatic,
				"IsStructurallyStatic mismatch")
			assert.Equal(t, tc.expectedSemanticallyStatic, tc.node.GoAnnotations.IsStatic,
				"IsStatic mismatch")
		})
	}
}

func TestCollectCustomTags(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil templateAST", func(t *testing.T) {
		t.Parallel()

		result := collectCustomTags(nil, nil)

		assert.Nil(t, result)
	})

	t.Run("collects hyphenated tags without registry", func(t *testing.T) {
		t.Parallel()

		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "my-button"},
					{NodeType: ast_domain.NodeElement, TagName: "custom-card"},
					{NodeType: ast_domain.NodeElement, TagName: "span"},
				},
			}},
		}

		result := collectCustomTags(templateAST, nil)

		assert.Len(t, result, 2)
		assert.Contains(t, result, "my-button")
		assert.Contains(t, result, "custom-card")
	})

	t.Run("excludes piko: prefixed tags", func(t *testing.T) {
		t.Parallel()

		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "piko:img"},
					{NodeType: ast_domain.NodeElement, TagName: "piko:svg"},
					{NodeType: ast_domain.NodeElement, TagName: "my-component"},
				},
			}},
		}

		result := collectCustomTags(templateAST, nil)

		assert.Len(t, result, 1)
		assert.Contains(t, result, "my-component")
	})

	t.Run("excludes pml- prefixed tags", func(t *testing.T) {
		t.Parallel()

		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "pml-img"},
					{NodeType: ast_domain.NodeElement, TagName: "pml-video"},
					{NodeType: ast_domain.NodeElement, TagName: "custom-element"},
				},
			}},
		}

		result := collectCustomTags(templateAST, nil)

		assert.Len(t, result, 1)
		assert.Contains(t, result, "custom-element")
	})

	t.Run("deduplicates repeated tags", func(t *testing.T) {
		t.Parallel()

		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "my-button"},
					{NodeType: ast_domain.NodeElement, TagName: "my-button"},
					{NodeType: ast_domain.NodeElement, TagName: "my-button"},
				},
			}},
		}

		result := collectCustomTags(templateAST, nil)

		assert.Len(t, result, 1)
		assert.Contains(t, result, "my-button")
	})

	t.Run("returns sorted tags", func(t *testing.T) {
		t.Parallel()

		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "z-component"},
					{NodeType: ast_domain.NodeElement, TagName: "a-component"},
					{NodeType: ast_domain.NodeElement, TagName: "m-component"},
				},
			}},
		}

		result := collectCustomTags(templateAST, nil)

		assert.Len(t, result, 3)
		assert.Equal(t, "a-component", result[0])
		assert.Equal(t, "m-component", result[1])
		assert.Equal(t, "z-component", result[2])
	})

	t.Run("ignores non-element nodes", func(t *testing.T) {
		t.Parallel()

		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "some text"},
					{NodeType: ast_domain.NodeComment, TextContent: "a comment"},
					{NodeType: ast_domain.NodeElement, TagName: "my-element"},
				},
			}},
		}

		result := collectCustomTags(templateAST, nil)

		assert.Len(t, result, 1)
		assert.Contains(t, result, "my-element")
	})
}

type mockComponentRegistry struct {
	registered map[string]bool
}

func (m *mockComponentRegistry) IsRegistered(tagName string) bool {
	return m.registered[tagName]
}

func (m *mockComponentRegistry) GetEntryPoints() []string {
	return nil
}

func TestCollectCustomTags_WithRegistry(t *testing.T) {
	t.Parallel()

	t.Run("uses registry for tag detection", func(t *testing.T) {
		t.Parallel()

		registry := &mockComponentRegistry{
			registered: map[string]bool{
				"registered-component": true,
			},
		}

		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "registered-component"},
					{NodeType: ast_domain.NodeElement, TagName: "unregistered-component"},
				},
			}},
		}

		result := collectCustomTags(templateAST, registry)

		assert.Len(t, result, 1)
		assert.Contains(t, result, "registered-component")
	})
}

type transformTestFSReader struct {
	files map[string][]byte
}

func (m *transformTestFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	data, ok := m.files[filePath]
	if !ok {
		return nil, errors.New("file not found: " + filePath)
	}
	return data, nil
}

type transformTestResolver struct {
	assetPaths map[string]string
	assetErrs  map[string]error
	baseDir    string
}

func (m *transformTestResolver) DetectLocalModule(_ context.Context) error { return nil }
func (m *transformTestResolver) GetModuleName() string                     { return "test/module" }
func (m *transformTestResolver) GetBaseDir() string                        { return m.baseDir }
func (m *transformTestResolver) ResolvePKPath(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}
func (m *transformTestResolver) ResolveCSSPath(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}
func (m *transformTestResolver) ResolveAssetPath(_ context.Context, importPath string, _ string) (string, error) {
	if m.assetErrs != nil {
		if err, ok := m.assetErrs[importPath]; ok {
			return "", err
		}
	}
	if m.assetPaths != nil {
		if resolved, ok := m.assetPaths[importPath]; ok {
			return resolved, nil
		}
	}
	return "", errors.New("asset not found: " + importPath)
}
func (m *transformTestResolver) ConvertEntryPointPathToManifestKey(entryPointPath string) string {
	return entryPointPath
}
func (m *transformTestResolver) GetModuleDir(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (m *transformTestResolver) FindModuleBoundary(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}

func TestAnalyseNodeForPrerenderability(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node                      *ast_domain.TemplateNode
		name                      string
		expectedPrerenderable     bool
		expectAnnotationUntouched bool
	}{
		{
			name: "nil GoAnnotations leaves node unchanged",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				TagName:       "div",
				GoAnnotations: nil,
			},
			expectedPrerenderable:     false,
			expectAnnotationUntouched: true,
		},
		{
			name: "non-static node is not prerenderable",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				TagName:       "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: false},
			},
			expectedPrerenderable: false,
		},
		{
			name: "static div with no children is prerenderable",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				TagName:       "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: true},
			},
			expectedPrerenderable: true,
		},
		{
			name: "static piko:svg is not prerenderable (runtime tag)",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				TagName:       "piko:svg",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: true},
			},
			expectedPrerenderable: false,
		},
		{
			name: "static piko:img is not prerenderable (runtime tag)",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				TagName:       "piko:img",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: true},
			},
			expectedPrerenderable: false,
		},
		{
			name: "static piko:video is not prerenderable (runtime tag)",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				TagName:       "piko:video",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: true},
			},
			expectedPrerenderable: false,
		},
		{
			name: "static node with partial invocation is not prerenderable",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					IsStatic:    true,
					PartialInfo: &ast_domain.PartialInvocationInfo{InvocationKey: "abc"},
				},
			},
			expectedPrerenderable: false,
		},
		{
			name: "static node with non-prerenderable child is not prerenderable",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				TagName:       "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: true},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:      ast_domain.NodeElement,
						TagName:       "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: true, IsFullyPrerenderable: false},
					},
				},
			},
			expectedPrerenderable: false,
		},
		{
			name: "static node with all prerenderable children is prerenderable",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				TagName:       "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: true},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:      ast_domain.NodeElement,
						TagName:       "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: true, IsFullyPrerenderable: true},
					},
				},
			},
			expectedPrerenderable: true,
		},
		{
			name: "static node with child having nil GoAnnotations is not prerenderable",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				TagName:       "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{IsStatic: true},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:      ast_domain.NodeElement,
						TagName:       "span",
						GoAnnotations: nil,
					},
				},
			},
			expectedPrerenderable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			analyseNodeForPrerenderability(tc.node)

			if tc.expectAnnotationUntouched {
				assert.Nil(t, tc.node.GoAnnotations)
			} else {
				require.NotNil(t, tc.node.GoAnnotations)
				assert.Equal(t, tc.expectedPrerenderable, tc.node.GoAnnotations.IsFullyPrerenderable)
			}
		})
	}
}

func TestPerformStaticAnalysis(t *testing.T) {
	t.Parallel()

	t.Run("marks plain static tree as prerenderable", func(t *testing.T) {
		t.Parallel()

		child := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "span",
		}
		root := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: []*ast_domain.TemplateNode{child},
		}
		templateAST := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{root}}

		performStaticAnalysis(context.Background(), templateAST)

		require.NotNil(t, child.GoAnnotations)
		assert.True(t, child.GoAnnotations.IsStatic)
		assert.True(t, child.GoAnnotations.IsFullyPrerenderable)

		require.NotNil(t, root.GoAnnotations)
		assert.True(t, root.GoAnnotations.IsStatic)
		assert.True(t, root.GoAnnotations.IsFullyPrerenderable)
	})

	t.Run("dynamic child prevents parent prerenderability", func(t *testing.T) {
		t.Parallel()

		dynamicChild := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeText,
			RichText: []ast_domain.TextPart{
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "x"}},
			},
		}
		root := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: []*ast_domain.TemplateNode{dynamicChild},
		}
		templateAST := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{root}}

		performStaticAnalysis(context.Background(), templateAST)

		require.NotNil(t, dynamicChild.GoAnnotations)
		assert.False(t, dynamicChild.GoAnnotations.IsStatic)
		assert.False(t, dynamicChild.GoAnnotations.IsFullyPrerenderable)

		require.NotNil(t, root.GoAnnotations)
		assert.True(t, root.GoAnnotations.IsStructurallyStatic)
		assert.False(t, root.GoAnnotations.IsStatic)
		assert.False(t, root.GoAnnotations.IsFullyPrerenderable)
	})

	t.Run("runtime tag prevents prerenderability but stays static", func(t *testing.T) {
		t.Parallel()

		svgNode := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:svg",
		}
		templateAST := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{svgNode}}

		performStaticAnalysis(context.Background(), templateAST)

		require.NotNil(t, svgNode.GoAnnotations)
		assert.False(t, svgNode.GoAnnotations.IsStatic,
			"piko:svg should not be marked static because it is a runtime-processed element")
		assert.False(t, svgNode.GoAnnotations.IsFullyPrerenderable)
	})
}

func TestHasDynamicSrc(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "no dynamic attributes",
			node:     &ast_domain.TemplateNode{},
			expected: false,
		},
		{
			name: "dynamic attribute with name src",
			node: &ast_domain.TemplateNode{
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "src", Expression: &ast_domain.Identifier{Name: "imgSrc"}},
				},
			},
			expected: true,
		},
		{
			name: "dynamic attribute with different name",
			node: &ast_domain.TemplateNode{
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "alt", Expression: &ast_domain.Identifier{Name: "altText"}},
				},
			},
			expected: false,
		},
		{
			name: "multiple dynamic attributes including src",
			node: &ast_domain.TemplateNode{
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "class", Expression: &ast_domain.Identifier{Name: "cls"}},
					{Name: "src", Expression: &ast_domain.Identifier{Name: "imgSrc"}},
				},
			},
			expected: true,
		},
	}

	ac := &assetCollectionContext{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := ac.hasDynamicSrc(tc.node)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildDependency(t *testing.T) {
	t.Parallel()

	t.Run("builds dependency from piko:img node", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			originComponentPath: "pages/index.pk",
		}
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:img",
			Location: ast_domain.Location{Line: 10, Column: 5, Offset: 100},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "/images/hero.jpg"},
				{Name: "width", Value: "300"},
				{Name: "alt", Value: "Hero image"},
			},
		}

		dependency := ac.buildDependency(node, "/images/hero.jpg")

		assert.Equal(t, "/images/hero.jpg", dependency.SourcePath)
		assert.Equal(t, "img", dependency.AssetType)
		assert.Equal(t, "pages/index.pk", dependency.OriginComponentPath)
		assert.Equal(t, 10, dependency.Location.Line)
		_, hasSrc := dependency.TransformationParams["src"]
		assert.False(t, hasSrc, "src should not be in TransformationParams")
		assert.Equal(t, "300", dependency.TransformationParams["width"])
		assert.Equal(t, "Hero image", dependency.TransformationParams["alt"])
	})

	t.Run("builds dependency from piko:svg node", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			originComponentPath: "partials/icon.pk",
		}
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:svg",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "/icons/arrow.svg"},
			},
		}

		dependency := ac.buildDependency(node, "/icons/arrow.svg")

		assert.Equal(t, "svg", dependency.AssetType)
		assert.Empty(t, dependency.TransformationParams)
	})
}

func TestBuildPosterDependency(t *testing.T) {
	t.Parallel()

	t.Run("creates poster dependency with correct asset type", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			originComponentPath: "pages/video.pk",
		}
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:video",
			Location: ast_domain.Location{Line: 5, Column: 1},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "/videos/intro.mp4"},
				{Name: "poster", Value: "/images/poster.jpg"},
				{Name: "poster-widths", Value: "320 640"},
				{Name: "poster-formats", Value: "webp"},
			},
		}

		dependency := ac.buildPosterDependency(node, "/images/poster.jpg")

		assert.Equal(t, "/images/poster.jpg", dependency.SourcePath)
		assert.Equal(t, "img", dependency.AssetType)
		assert.Equal(t, "pages/video.pk", dependency.OriginComponentPath)
		assert.Equal(t, 5, dependency.Location.Line)
		assert.Equal(t, "320 640", dependency.TransformationParams["widths"])
		assert.Equal(t, "webp", dependency.TransformationParams["formats"])
	})

	t.Run("maps poster-density to densities", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "poster-density", Value: "1x 2x"},
				{Name: "poster-sizes", Value: "50vw"},
			},
		}

		dependency := ac.buildPosterDependency(node, "/img/poster.jpg")

		assert.Equal(t, "1x 2x", dependency.TransformationParams["densities"])
		assert.Equal(t, "50vw", dependency.TransformationParams["sizes"])
	})

	t.Run("ignores non-poster attributes", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "1920"},
				{Name: "height", Value: "1080"},
			},
		}

		dependency := ac.buildPosterDependency(node, "/img/poster.jpg")

		assert.Empty(t, dependency.TransformationParams)
	})
}

func TestValidateDensitiesFormat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		densities       string
		expectedDiagLen int
	}{
		{
			name:            "valid densities 1x 2x",
			densities:       "1x 2x",
			expectedDiagLen: 0,
		},
		{
			name:            "valid densities x1 x2",
			densities:       "x1 x2",
			expectedDiagLen: 0,
		},
		{
			name:            "valid single density 3x",
			densities:       "3x",
			expectedDiagLen: 0,
		},
		{
			name:            "unrecognised density abc silently defaults",
			densities:       "abc",
			expectedDiagLen: 0,
		},
		{
			name:            "mixed valid and unrecognised silently defaults",
			densities:       "1x badvalue 2x",
			expectedDiagLen: 0,
		},
		{
			name:            "empty string produces no diagnostics",
			densities:       "",
			expectedDiagLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ac := &assetCollectionContext{
				originComponentPath: "test.pk",
			}
			node := &ast_domain.TemplateNode{
				TagName: "piko:img",
			}

			ac.validateDensitiesFormat(node, tc.densities)

			assert.Len(t, ac.diagnostics, tc.expectedDiagLen)
		})
	}
}

func TestUpdateNodeSrcAttribute(t *testing.T) {
	t.Parallel()

	t.Run("updates existing src attribute", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "hero"},
				{Name: "src", Value: "@/images/old.jpg"},
				{Name: "alt", Value: "image"},
			},
		}

		ac.updateNodeSrcAttribute(node, "module/images/new.jpg")

		assert.Equal(t, "module/images/new.jpg", node.Attributes[1].Value)
	})

	t.Run("case-insensitive match", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "SRC", Value: "@/images/old.jpg"},
			},
		}

		ac.updateNodeSrcAttribute(node, "module/images/new.jpg")

		assert.Equal(t, "module/images/new.jpg", node.Attributes[0].Value)
	})

	t.Run("does nothing when no src attribute", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "alt", Value: "text"},
			},
		}

		ac.updateNodeSrcAttribute(node, "new-value")

		assert.Equal(t, "text", node.Attributes[0].Value)
	})
}

func TestUpdatePosterAttribute(t *testing.T) {
	t.Parallel()

	t.Run("updates existing poster attribute", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "/vid.mp4"},
				{Name: "poster", Value: "@/images/old-poster.jpg"},
			},
		}

		ac.updatePosterAttribute(node, "module/images/new-poster.jpg")

		assert.Equal(t, "module/images/new-poster.jpg", node.Attributes[1].Value)
	})

	t.Run("case-insensitive match for poster", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "POSTER", Value: "old.jpg"},
			},
		}

		ac.updatePosterAttribute(node, "new.jpg")

		assert.Equal(t, "new.jpg", node.Attributes[0].Value)
	})

	t.Run("does nothing when no poster attribute", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "/vid.mp4"},
			},
		}

		ac.updatePosterAttribute(node, "new.jpg")

		assert.Equal(t, "/vid.mp4", node.Attributes[0].Value)
	})
}

func TestAddDiagnostic(t *testing.T) {
	t.Parallel()

	t.Run("appends diagnostic with origin component path", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			originComponentPath: "pages/home.pk",
		}
		node := &ast_domain.TemplateNode{
			TagName:  "piko:img",
			Location: ast_domain.Location{Line: 12, Column: 3},
		}

		ac.addDiagnostic(node, "test warning message", "")

		require.Len(t, ac.diagnostics, 1)
		diagnostic := ac.diagnostics[0]
		assert.Equal(t, ast_domain.Warning, diagnostic.Severity)
		assert.Equal(t, "test warning message", diagnostic.Message)
		assert.Equal(t, "piko:img", diagnostic.Expression)
		assert.Equal(t, 12, diagnostic.Location.Line)
		assert.Equal(t, "pages/home.pk", diagnostic.SourcePath)
	})

	t.Run("uses OriginalSourcePath when available", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			originComponentPath: "pages/home.pk",
		}
		node := &ast_domain.TemplateNode{
			TagName: "piko:img",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("partials/card.pk"),
			},
		}

		ac.addDiagnostic(node, "message", "")

		require.Len(t, ac.diagnostics, 1)
		assert.Equal(t, "partials/card.pk", ac.diagnostics[0].SourcePath)
	})

	t.Run("accumulates multiple diagnostics", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			originComponentPath: "test.pk",
		}
		node := &ast_domain.TemplateNode{TagName: "piko:img"}

		ac.addDiagnostic(node, "first", "")
		ac.addDiagnostic(node, "second", "")
		ac.addDiagnostic(node, "third", "")

		assert.Len(t, ac.diagnostics, 3)
	})
}

func TestMergeProfileAttributes(t *testing.T) {
	t.Parallel()

	t.Run("merges profile params into node attributes", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			TagName: "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "/images/hero.jpg"},
			},
		}
		profileDef := []config.AssetTransformationStep{
			{
				Params: map[string]string{
					"width":   "800",
					"quality": "80",
				},
			},
		}

		ac.mergeProfileAttributes(context.Background(), node, profileDef, "hero")

		attributeMap := make(map[string]string)
		for _, attr := range node.Attributes {
			attributeMap[attr.Name] = attr.Value
		}
		assert.Equal(t, "800", attributeMap["width"])
		assert.Equal(t, "80", attributeMap["quality"])
		assert.Equal(t, "/images/hero.jpg", attributeMap["src"])
	})

	t.Run("node attributes override profile params", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			TagName: "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "width", Value: "1200"},
			},
		}
		profileDef := []config.AssetTransformationStep{
			{
				Params: map[string]string{
					"width":   "800",
					"quality": "80",
				},
			},
		}

		ac.mergeProfileAttributes(context.Background(), node, profileDef, "test")

		attributeMap := make(map[string]string)
		for _, attr := range node.Attributes {
			attributeMap[attr.Name] = attr.Value
		}
		assert.Equal(t, "1200", attributeMap["width"], "node attribute should override profile")
		assert.Equal(t, "80", attributeMap["quality"])
	})

	t.Run("removes profile attribute from merged result", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			TagName: "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "profile", Value: "hero"},
				{Name: "src", Value: "/img.jpg"},
			},
		}
		profileDef := []config.AssetTransformationStep{
			{
				Params: map[string]string{
					"width": "800",
				},
			},
		}

		ac.mergeProfileAttributes(context.Background(), node, profileDef, "hero")

		for _, attr := range node.Attributes {
			assert.NotEqual(t, "profile", attr.Name, "profile attribute should be removed")
		}
	})

	t.Run("attributes are sorted alphabetically", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			TagName: "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "/img.jpg"},
			},
		}
		profileDef := []config.AssetTransformationStep{
			{
				Params: map[string]string{
					"zindex":  "10",
					"alt":     "text",
					"quality": "80",
				},
			},
		}

		ac.mergeProfileAttributes(context.Background(), node, profileDef, "test")

		names := make([]string, len(node.Attributes))
		for i, attr := range node.Attributes {
			names[i] = attr.Name
		}
		for i := 1; i < len(names); i++ {
			assert.True(t, names[i-1] <= names[i],
				"attributes should be sorted, got %s before %s", names[i-1], names[i])
		}
	})
}

func TestCollectStaticAssetDependencies(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil AST", func(t *testing.T) {
		t.Parallel()

		deps, diagnostics := collectStaticAssetDependencies(
			context.Background(),
			nil,
			&transformTestResolver{baseDir: "/project"},
			AnnotatorPathsConfig{},
			&config.AssetsConfig{},
			&transformTestFSReader{},
		)

		assert.Nil(t, deps)
		assert.Nil(t, diagnostics)
	})

	t.Run("collects dependency from piko:img with static src", func(t *testing.T) {
		t.Parallel()

		resolver := &transformTestResolver{
			baseDir: "/project",
			assetPaths: map[string]string{
				"/images/hero.jpg": "/project/lib/images/hero.jpg",
			},
		}
		fsReader := &transformTestFSReader{
			files: map[string][]byte{
				"/project/lib/images/hero.jpg": []byte("fake-image-data"),
			},
		}
		pathsConfig := AnnotatorPathsConfig{
			AssetsSourceDir: "lib",
		}
		templateAST := &ast_domain.TemplateAST{
			SourcePath: new("pages/index.pk"),
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "piko:img",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "src", Value: "/images/hero.jpg"},
						{Name: "alt", Value: "Hero"},
					},
				},
			},
		}

		deps, diagnostics := collectStaticAssetDependencies(
			context.Background(),
			templateAST,
			resolver,
			pathsConfig,
			&config.AssetsConfig{},
			fsReader,
		)

		assert.Empty(t, diagnostics)
		require.Len(t, deps, 1)
		assert.Equal(t, "/images/hero.jpg", deps[0].SourcePath)
		assert.Equal(t, "img", deps[0].AssetType)
		assert.Equal(t, "Hero", deps[0].TransformationParams["alt"])
	})

	t.Run("skips non-asset tags", func(t *testing.T) {
		t.Parallel()

		resolver := &transformTestResolver{baseDir: "/project"}
		fsReader := &transformTestFSReader{}
		pathsConfig := AnnotatorPathsConfig{}
		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "src", Value: "/images/hero.jpg"},
					},
				},
			},
		}

		deps, diagnostics := collectStaticAssetDependencies(
			context.Background(),
			templateAST,
			resolver,
			pathsConfig,
			&config.AssetsConfig{},
			fsReader,
		)

		assert.Empty(t, deps)
		assert.Empty(t, diagnostics)
	})

	t.Run("skips nodes with dynamic src", func(t *testing.T) {
		t.Parallel()

		resolver := &transformTestResolver{baseDir: "/project"}
		fsReader := &transformTestFSReader{}
		pathsConfig := AnnotatorPathsConfig{}
		templateAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "piko:img",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{Name: "src", Expression: &ast_domain.Identifier{Name: "imgPath"}},
					},
				},
			},
		}

		deps, diagnostics := collectStaticAssetDependencies(
			context.Background(),
			templateAST,
			resolver,
			pathsConfig,
			&config.AssetsConfig{},
			fsReader,
		)

		assert.Empty(t, deps)
		assert.Empty(t, diagnostics)
	})
}

func TestProcessAssetNode(t *testing.T) {
	t.Parallel()

	t.Run("skips non-element nodes", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeText,
		}

		result := ac.processAssetNode(context.Background(), node)

		assert.True(t, result)
		assert.Empty(t, ac.dependencies)
	})

	t.Run("skips non-asset-tag elements", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{}
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}

		result := ac.processAssetNode(context.Background(), node)

		assert.True(t, result)
		assert.Empty(t, ac.dependencies)
	})

	t.Run("skips element with dynamic src", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{},
		}
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:img",
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "src", Expression: &ast_domain.Identifier{Name: "dynamicPath"}},
			},
		}

		result := ac.processAssetNode(context.Background(), node)

		assert.True(t, result)
		assert.Empty(t, ac.dependencies)
	})

	t.Run("skips element with no src attribute", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{},
		}
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "alt", Value: "no src here"},
			},
		}

		result := ac.processAssetNode(context.Background(), node)

		assert.True(t, result)
		assert.Empty(t, ac.dependencies)
	})

	t.Run("skips element with empty src attribute", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{},
		}
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: ""},
			},
		}

		result := ac.processAssetNode(context.Background(), node)

		assert.True(t, result)
		assert.Empty(t, ac.dependencies)
	})
}

func TestExpandProfileIfPresent(t *testing.T) {
	t.Parallel()

	t.Run("does nothing when no profile attribute", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{},
		}
		node := &ast_domain.TemplateNode{
			TagName: "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "/img.jpg"},
			},
		}
		originalAttrs := len(node.Attributes)

		ac.expandProfileIfPresent(context.Background(), node)

		assert.Len(t, node.Attributes, originalAttrs)
	})

	t.Run("adds diagnostic when image profile not found", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{
				Image: config.ImageAssetsConfig{
					Profiles: map[string][]config.AssetTransformationStep{},
				},
			},
			originComponentPath: "test.pk",
		}
		node := &ast_domain.TemplateNode{
			TagName: "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "profile", Value: "nonexistent"},
				{Name: "src", Value: "/img.jpg"},
			},
		}

		ac.expandProfileIfPresent(context.Background(), node)

		require.Len(t, ac.diagnostics, 1)
		assert.Contains(t, ac.diagnostics[0].Message, "not found")
	})

	t.Run("adds diagnostic when video profile not found", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{
				Video: config.VideoAssetsConfig{
					Profiles: map[string][]config.AssetTransformationStep{},
				},
			},
			originComponentPath: "test.pk",
		}
		node := &ast_domain.TemplateNode{
			TagName: "piko:video",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "profile", Value: "nonexistent"},
				{Name: "src", Value: "/vid.mp4"},
			},
		}

		ac.expandProfileIfPresent(context.Background(), node)

		require.Len(t, ac.diagnostics, 1)
		assert.Contains(t, ac.diagnostics[0].Message, "Video profile")
		assert.Contains(t, ac.diagnostics[0].Message, "not found")
	})

	t.Run("adds diagnostic when profile has no capabilities", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{
				Image: config.ImageAssetsConfig{
					Profiles: map[string][]config.AssetTransformationStep{
						"empty-profile": {},
					},
				},
			},
			originComponentPath: "test.pk",
		}
		node := &ast_domain.TemplateNode{
			TagName: "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "profile", Value: "empty-profile"},
				{Name: "src", Value: "/img.jpg"},
			},
		}

		ac.expandProfileIfPresent(context.Background(), node)

		require.Len(t, ac.diagnostics, 1)
		assert.Contains(t, ac.diagnostics[0].Message, "no capabilities")
	})

	t.Run("expands valid image profile and merges attributes", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{
				Image: config.ImageAssetsConfig{
					Profiles: map[string][]config.AssetTransformationStep{
						"hero": {
							{
								Params: map[string]string{
									"width":   "800",
									"quality": "80",
									"format":  "webp",
								},
							},
						},
					},
				},
			},
			originComponentPath: "test.pk",
		}
		node := &ast_domain.TemplateNode{
			TagName: "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "profile", Value: "hero"},
				{Name: "src", Value: "/img.jpg"},
			},
		}

		ac.expandProfileIfPresent(context.Background(), node)

		assert.Empty(t, ac.diagnostics)
		attributeMap := make(map[string]string)
		for _, attr := range node.Attributes {
			attributeMap[attr.Name] = attr.Value
		}
		assert.Equal(t, "800", attributeMap["width"])
		assert.Equal(t, "80", attributeMap["quality"])
		assert.Equal(t, "webp", attributeMap["format"])
		assert.Equal(t, "/img.jpg", attributeMap["src"])
		_, hasProfile := attributeMap["profile"]
		assert.False(t, hasProfile, "profile attribute should be removed after expansion")
	})
}

func TestValidateAndEnrichResponsiveImage(t *testing.T) {
	t.Parallel()

	t.Run("does nothing when no densities and no sizes", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{},
		}
		node := &ast_domain.TemplateNode{TagName: "piko:img"}
		dependency := &annotator_dto.StaticAssetDependency{
			TransformationParams: map[string]string{
				"width": "300",
			},
		}

		ac.validateAndEnrichResponsiveImage(context.Background(), node, dependency, "/img.jpg")

		_, hasResponsive := dependency.TransformationParams["_responsive"]
		assert.False(t, hasResponsive)
	})

	t.Run("marks as responsive when densities present", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig:        &config.AssetsConfig{},
			originComponentPath: "test.pk",
		}
		node := &ast_domain.TemplateNode{TagName: "piko:img"}
		dependency := &annotator_dto.StaticAssetDependency{
			TransformationParams: map[string]string{
				"densities": "1x 2x",
				"width":     "300",
			},
		}

		ac.validateAndEnrichResponsiveImage(context.Background(), node, dependency, "/img.jpg")

		assert.Equal(t, "true", dependency.TransformationParams["_responsive"])
	})

	t.Run("marks as responsive when sizes present", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{
				Image: config.ImageAssetsConfig{
					DefaultDensities: []string{"1x", "2x"},
				},
			},
		}
		node := &ast_domain.TemplateNode{TagName: "piko:img"}
		dependency := &annotator_dto.StaticAssetDependency{
			TransformationParams: map[string]string{
				"sizes": "100vw",
			},
		}

		ac.validateAndEnrichResponsiveImage(context.Background(), node, dependency, "/img.jpg")

		assert.Equal(t, "true", dependency.TransformationParams["_responsive"])
	})

	t.Run("adds diagnostic when densities present without width or sizes", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig:        &config.AssetsConfig{},
			originComponentPath: "test.pk",
		}
		node := &ast_domain.TemplateNode{TagName: "piko:img"}
		dependency := &annotator_dto.StaticAssetDependency{
			TransformationParams: map[string]string{
				"densities": "1x 2x",
			},
		}

		ac.validateAndEnrichResponsiveImage(context.Background(), node, dependency, "/img.jpg")

		require.Len(t, ac.diagnostics, 1)
		assert.Contains(t, ac.diagnostics[0].Message, "densities")
		assert.Contains(t, ac.diagnostics[0].Message, "width")
	})

	t.Run("no diagnostic when densities present with width", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{},
		}
		node := &ast_domain.TemplateNode{TagName: "piko:img"}
		dependency := &annotator_dto.StaticAssetDependency{
			TransformationParams: map[string]string{
				"densities": "1x 2x",
				"width":     "400",
			},
		}

		ac.validateAndEnrichResponsiveImage(context.Background(), node, dependency, "/img.jpg")

		assert.Empty(t, ac.diagnostics)
	})

	t.Run("no diagnostic when densities present with sizes", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig: &config.AssetsConfig{},
		}
		node := &ast_domain.TemplateNode{TagName: "piko:img"}
		dependency := &annotator_dto.StaticAssetDependency{
			TransformationParams: map[string]string{
				"densities": "1x 2x",
				"sizes":     "50vw",
			},
		}

		ac.validateAndEnrichResponsiveImage(context.Background(), node, dependency, "/img.jpg")

		assert.Empty(t, ac.diagnostics)
	})

	t.Run("unrecognised density format is silently treated as default", func(t *testing.T) {
		t.Parallel()

		ac := &assetCollectionContext{
			assetsConfig:        &config.AssetsConfig{},
			originComponentPath: "test.pk",
		}
		node := &ast_domain.TemplateNode{TagName: "piko:img"}
		dependency := &annotator_dto.StaticAssetDependency{
			TransformationParams: map[string]string{
				"densities": "bad",
				"width":     "300",
			},
		}

		ac.validateAndEnrichResponsiveImage(context.Background(), node, dependency, "/img.jpg")

		assert.Empty(t, ac.diagnostics)
		assert.Equal(t, "true", dependency.TransformationParams["_responsive"])
	})
}
