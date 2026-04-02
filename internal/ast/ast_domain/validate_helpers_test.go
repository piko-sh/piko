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
)

func TestHasDifferentSourcePath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		pForAnn  *GoGeneratorAnnotation
		otherAnn *GoGeneratorAnnotation
		name     string
		expected bool
	}{
		{
			name:     "both annotations nil returns false",
			pForAnn:  nil,
			otherAnn: nil,
			expected: false,
		},
		{
			name:     "pForAnn nil returns false",
			pForAnn:  nil,
			otherAnn: &GoGeneratorAnnotation{OriginalSourcePath: new("file.pk")},
			expected: false,
		},
		{
			name:     "otherAnn nil returns false",
			pForAnn:  &GoGeneratorAnnotation{OriginalSourcePath: new("file.pk")},
			otherAnn: nil,
			expected: false,
		},
		{
			name:     "pForAnn has nil OriginalSourcePath returns false",
			pForAnn:  &GoGeneratorAnnotation{OriginalSourcePath: nil},
			otherAnn: &GoGeneratorAnnotation{OriginalSourcePath: new("file.pk")},
			expected: false,
		},
		{
			name:     "otherAnn has nil OriginalSourcePath returns false",
			pForAnn:  &GoGeneratorAnnotation{OriginalSourcePath: new("file.pk")},
			otherAnn: &GoGeneratorAnnotation{OriginalSourcePath: nil},
			expected: false,
		},
		{
			name:     "both have nil OriginalSourcePath returns false",
			pForAnn:  &GoGeneratorAnnotation{OriginalSourcePath: nil},
			otherAnn: &GoGeneratorAnnotation{OriginalSourcePath: nil},
			expected: false,
		},
		{
			name:     "same source path returns false",
			pForAnn:  &GoGeneratorAnnotation{OriginalSourcePath: new("component.pk")},
			otherAnn: &GoGeneratorAnnotation{OriginalSourcePath: new("component.pk")},
			expected: false,
		},
		{
			name:     "different source paths returns true",
			pForAnn:  &GoGeneratorAnnotation{OriginalSourcePath: new("component_a.pk")},
			otherAnn: &GoGeneratorAnnotation{OriginalSourcePath: new("component_b.pk")},
			expected: true,
		},
		{
			name:     "empty vs non-empty source path returns true",
			pForAnn:  &GoGeneratorAnnotation{OriginalSourcePath: new("")},
			otherAnn: &GoGeneratorAnnotation{OriginalSourcePath: new("component.pk")},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := hasDifferentSourcePath(tc.pForAnn, tc.otherAnn)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindPreviousElementSibling(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when current index is zero", func(t *testing.T) {
		t.Parallel()

		nodes := []*TemplateNode{
			{NodeType: NodeElement, TagName: "div"},
		}
		result := findPreviousElementSibling(nodes, 0)
		assert.Nil(t, result)
	})

	t.Run("returns previous element node", func(t *testing.T) {
		t.Parallel()

		previousElement := &TemplateNode{NodeType: NodeElement, TagName: "span"}
		nodes := []*TemplateNode{
			previousElement,
			{NodeType: NodeElement, TagName: "div"},
		}
		result := findPreviousElementSibling(nodes, 1)
		assert.Same(t, previousElement, result)
	})

	t.Run("skips whitespace-only text nodes", func(t *testing.T) {
		t.Parallel()

		previousElement := &TemplateNode{NodeType: NodeElement, TagName: "span"}
		nodes := []*TemplateNode{
			previousElement,
			{NodeType: NodeText, TextContent: "   \n\t  "},
			{NodeType: NodeElement, TagName: "div"},
		}
		result := findPreviousElementSibling(nodes, 2)
		assert.Same(t, previousElement, result)
	})

	t.Run("returns nil when non-whitespace text separates elements", func(t *testing.T) {
		t.Parallel()

		nodes := []*TemplateNode{
			{NodeType: NodeElement, TagName: "span"},
			{NodeType: NodeText, TextContent: "meaningful text"},
			{NodeType: NodeElement, TagName: "div"},
		}
		result := findPreviousElementSibling(nodes, 2)
		assert.Nil(t, result)
	})

	t.Run("returns nil when no previous element exists", func(t *testing.T) {
		t.Parallel()

		nodes := []*TemplateNode{
			{NodeType: NodeText, TextContent: "   "},
			{NodeType: NodeElement, TagName: "div"},
		}
		result := findPreviousElementSibling(nodes, 1)
		assert.Nil(t, result)
	})

	t.Run("skips comment nodes to find previous element", func(t *testing.T) {
		t.Parallel()

		previousElement := &TemplateNode{NodeType: NodeElement, TagName: "p"}
		nodes := []*TemplateNode{
			previousElement,
			{NodeType: NodeComment, TextContent: "a comment"},
			{NodeType: NodeElement, TagName: "div"},
		}
		result := findPreviousElementSibling(nodes, 2)
		assert.Same(t, previousElement, result)
	})

	t.Run("skips multiple whitespace text nodes", func(t *testing.T) {
		t.Parallel()

		previousElement := &TemplateNode{NodeType: NodeElement, TagName: "h1"}
		nodes := []*TemplateNode{
			previousElement,
			{NodeType: NodeText, TextContent: "   "},
			{NodeType: NodeText, TextContent: "\n"},
			{NodeType: NodeText, TextContent: "\t"},
			{NodeType: NodeElement, TagName: "div"},
		}
		result := findPreviousElementSibling(nodes, 4)
		assert.Same(t, previousElement, result)
	})

	t.Run("returns nil for empty node list", func(t *testing.T) {
		t.Parallel()

		var nodes []*TemplateNode
		result := findPreviousElementSibling(nodes, 0)
		assert.Nil(t, result)
	})

	t.Run("returns nearest element when multiple elements precede", func(t *testing.T) {
		t.Parallel()

		firstElement := &TemplateNode{NodeType: NodeElement, TagName: "h1"}
		secondElement := &TemplateNode{NodeType: NodeElement, TagName: "h2"}
		nodes := []*TemplateNode{
			firstElement,
			secondElement,
			{NodeType: NodeElement, TagName: "div"},
		}
		result := findPreviousElementSibling(nodes, 2)
		assert.Same(t, secondElement, result)
	})
}

func TestGetElseDirective(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node          *TemplateNode
		name          string
		expectedType  DirectiveType
		expectedFound bool
	}{
		{
			name:          "node with DirElseIf returns it",
			node:          &TemplateNode{DirElseIf: &Directive{Type: DirectiveElseIf}},
			expectedFound: true,
			expectedType:  DirectiveElseIf,
		},
		{
			name:          "node with DirElse returns it",
			node:          &TemplateNode{DirElse: &Directive{Type: DirectiveElse}},
			expectedFound: true,
			expectedType:  DirectiveElse,
		},
		{
			name: "node with both DirElseIf and DirElse returns DirElseIf",
			node: &TemplateNode{
				DirElseIf: &Directive{Type: DirectiveElseIf},
				DirElse:   &Directive{Type: DirectiveElse},
			},
			expectedFound: true,
			expectedType:  DirectiveElseIf,
		},
		{
			name:          "node without else directives returns nil and false",
			node:          &TemplateNode{TagName: "div"},
			expectedFound: false,
		},
		{
			name:          "node with only DirIf returns nil and false",
			node:          &TemplateNode{DirIf: &Directive{Type: DirectiveIf}},
			expectedFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			directive, found := getElseDirective(tc.node)
			assert.Equal(t, tc.expectedFound, found)
			if tc.expectedFound {
				assert.NotNil(t, directive)
				assert.Equal(t, tc.expectedType, directive.Type)
			} else {
				assert.Nil(t, directive)
			}
		})
	}
}

func TestGetContentDirective(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node         *TemplateNode
		name         string
		expectedType DirectiveType
		expectedNil  bool
	}{
		{
			name:         "node with DirText returns it",
			node:         &TemplateNode{DirText: &Directive{Type: DirectiveText}},
			expectedNil:  false,
			expectedType: DirectiveText,
		},
		{
			name:         "node with DirHTML returns it",
			node:         &TemplateNode{DirHTML: &Directive{Type: DirectiveHTML}},
			expectedNil:  false,
			expectedType: DirectiveHTML,
		},
		{
			name: "node with both DirText and DirHTML returns DirText",
			node: &TemplateNode{
				DirText: &Directive{Type: DirectiveText},
				DirHTML: &Directive{Type: DirectiveHTML},
			},
			expectedNil:  false,
			expectedType: DirectiveText,
		},
		{
			name:        "node without content directives returns nil",
			node:        &TemplateNode{TagName: "div"},
			expectedNil: true,
		},
		{
			name:        "node with only DirIf returns nil",
			node:        &TemplateNode{DirIf: &Directive{Type: DirectiveIf}},
			expectedNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := getContentDirective(tc.node)
			if tc.expectedNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tc.expectedType, result.Type)
			}
		})
	}
}

func TestIsElementNode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "element node returns true",
			node:     &TemplateNode{NodeType: NodeElement, TagName: "div"},
			expected: true,
		},
		{
			name:     "text node returns false",
			node:     &TemplateNode{NodeType: NodeText, TextContent: "hello"},
			expected: false,
		},
		{
			name:     "comment node returns false",
			node:     &TemplateNode{NodeType: NodeComment, TextContent: "comment"},
			expected: false,
		},
		{
			name:     "fragment node returns false",
			node:     &TemplateNode{NodeType: NodeFragment},
			expected: false,
		},
		{
			name:     "zero-value node type returns false",
			node:     &TemplateNode{},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isElementNode(tc.node)
			assert.Equal(t, tc.expected, result)
		})
	}
}
